// Copyright 2025 John Wang. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package runtime

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// OAuthOptions configures OAuth 2.0 client credentials grant authentication.
//
//nolint:gosec // G117: OAuth struct fields, not hardcoded secrets
type OAuthOptions struct {
	// ClientID is the OAuth client ID. If empty, one will be auto-generated.
	ClientID string

	// ClientSecret is the OAuth client secret. If empty, one will be auto-generated.
	ClientSecret string

	// TokenExpiry is how long access tokens are valid. Defaults to 1 hour.
	TokenExpiry time.Duration

	// TokenPath is the path for the token endpoint. Defaults to "/oauth/token".
	TokenPath string
}

// OAuthCredentials contains the OAuth credentials for the server.
// This is returned in HTTPServerResult when OAuth is enabled.
//
//nolint:gosec // G117: OAuth struct fields, not hardcoded secrets
type OAuthCredentials struct {
	// ClientID is the OAuth client ID (provided or auto-generated).
	ClientID string

	// ClientSecret is the OAuth client secret (provided or auto-generated).
	ClientSecret string

	// TokenEndpoint is the full URL of the token endpoint.
	TokenEndpoint string
}

// oauthServer handles OAuth 2.0 client credentials grant.
type oauthServer struct {
	clientID     string
	clientSecret string
	tokenExpiry  time.Duration

	// tokens maps access tokens to their expiry time
	mu     sync.RWMutex
	tokens map[string]time.Time
}

// tokenResponse is the OAuth 2.0 token response.
//
//nolint:gosec // G117: OAuth struct fields, not hardcoded secrets
type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// tokenError is the OAuth 2.0 error response.
type tokenError struct {
	Error       string `json:"error"`
	Description string `json:"error_description,omitempty"`
}

// authorizationServerMetadata is the OAuth 2.0 Authorization Server Metadata (RFC 8414).
type authorizationServerMetadata struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	RegistrationEndpoint              string   `json:"registration_endpoint,omitempty"`
	GrantTypesSupported               []string `json:"grant_types_supported,omitempty"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported,omitempty"`
	ResponseTypesSupported            []string `json:"response_types_supported"`
	CodeChallengeMethodsSupported     []string `json:"code_challenge_methods_supported,omitempty"`
}

// protectedResourceMetadata is the OAuth 2.0 Protected Resource Metadata (RFC 9728).
type protectedResourceMetadata struct {
	Resource               string   `json:"resource"`
	AuthorizationServers   []string `json:"authorization_servers,omitempty"`
	BearerMethodsSupported []string `json:"bearer_methods_supported,omitempty"`
}

// newOAuthServer creates a new OAuth server with the given options.
func newOAuthServer(opts *OAuthOptions) (*oauthServer, error) {
	clientID := opts.ClientID
	if clientID == "" {
		var err error
		clientID, err = generateSecureToken(16)
		if err != nil {
			return nil, err
		}
	}

	clientSecret := opts.ClientSecret
	if clientSecret == "" {
		var err error
		clientSecret, err = generateSecureToken(32)
		if err != nil {
			return nil, err
		}
	}

	tokenExpiry := opts.TokenExpiry
	if tokenExpiry == 0 {
		tokenExpiry = time.Hour
	}

	s := &oauthServer{
		clientID:     clientID,
		clientSecret: clientSecret,
		tokenExpiry:  tokenExpiry,
		tokens:       make(map[string]time.Time),
	}

	// Start background cleanup of expired tokens
	go s.runCleanup()

	return s, nil
}

// TokenHandler returns an http.Handler for the OAuth token endpoint.
// This implements the client_credentials grant type per RFC 6749.
func (s *oauthServer) TokenHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			s.writeError(w, http.StatusMethodNotAllowed, "invalid_request", "Method not allowed")
			return
		}

		// Limit request body to 1MB and parse form data (G120)
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		if err := r.ParseForm(); err != nil {
			s.writeError(w, http.StatusBadRequest, "invalid_request", "Failed to parse request")
			return
		}

		// Check grant type
		grantType := r.Form.Get("grant_type")
		if grantType != "client_credentials" {
			s.writeError(w, http.StatusBadRequest, "unsupported_grant_type", "Only client_credentials grant is supported")
			return
		}

		// Get client credentials from Basic auth or form body
		clientID, clientSecret, ok := r.BasicAuth()
		if !ok {
			clientID = r.Form.Get("client_id")
			clientSecret = r.Form.Get("client_secret")
		}

		// Validate credentials using constant-time comparison
		if !s.validateCredentials(clientID, clientSecret) {
			s.writeError(w, http.StatusUnauthorized, "invalid_client", "Invalid client credentials")
			return
		}

		// Generate access token
		accessToken, err := generateSecureToken(32)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "server_error", "Failed to generate token")
			return
		}

		// Store token with expiry
		expiry := time.Now().Add(s.tokenExpiry)
		s.mu.Lock()
		s.tokens[accessToken] = expiry
		s.mu.Unlock()

		// Return token response
		resp := tokenResponse{
			AccessToken: accessToken,
			TokenType:   "Bearer",
			ExpiresIn:   int(s.tokenExpiry.Seconds()),
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Pragma", "no-cache")
		_ = json.NewEncoder(w).Encode(resp) //nolint:gosec // G117: OAuth token response contains access_token by spec
	})
}

// AuthorizationServerMetadataHandler returns an http.Handler for the OAuth 2.0
// Authorization Server Metadata endpoint (RFC 8414).
// This should be mounted at /.well-known/oauth-authorization-server
// The tokenPath is the path to the token endpoint (e.g., "/oauth/token").
func AuthorizationServerMetadataHandler(tokenPath string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Derive base URL from request
		baseURL := getBaseURL(r)

		metadata := &authorizationServerMetadata{
			Issuer:                            baseURL,
			AuthorizationEndpoint:             baseURL + "/oauth/authorize",
			TokenEndpoint:                     baseURL + tokenPath,
			RegistrationEndpoint:              baseURL + "/oauth/register",
			GrantTypesSupported:               []string{"client_credentials"},
			TokenEndpointAuthMethodsSupported: []string{"client_secret_basic", "client_secret_post"},
			ResponseTypesSupported:            []string{"code"},
			CodeChallengeMethodsSupported:     []string{"S256"},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(metadata)
	})
}

// ProtectedResourceMetadataHandler returns an http.Handler for the OAuth 2.0
// Protected Resource Metadata endpoint (RFC 9728).
// This should be mounted at /.well-known/oauth-protected-resource
// The mcpPath is the path to the MCP endpoint (e.g., "/mcp").
func ProtectedResourceMetadataHandler(mcpPath string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Derive base URL from request
		baseURL := getBaseURL(r)

		metadata := &protectedResourceMetadata{
			Resource:               baseURL + mcpPath,
			AuthorizationServers:   []string{baseURL},
			BearerMethodsSupported: []string{"header"},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(metadata)
	})
}

// getBaseURL derives the base URL from the request (scheme + host).
func getBaseURL(r *http.Request) string {
	scheme := "https"
	if r.TLS == nil {
		// Check for X-Forwarded-Proto header (common with proxies/ngrok)
		if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
			scheme = proto
		} else {
			scheme = "http"
		}
	}
	return scheme + "://" + r.Host
}

// BearerAuthMiddleware returns middleware that validates Bearer tokens.
func (s *oauthServer) BearerAuthMiddleware(resourceMetadataURL string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
				w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer resource_metadata="%s"`, resourceMetadataURL))
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			token := strings.TrimPrefix(auth, "Bearer ")

			s.mu.RLock()
			expiry, ok := s.tokens[token]
			s.mu.RUnlock()

			if !ok {
				w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer resource_metadata="%s", error="invalid_token"`, resourceMetadataURL))
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			if time.Now().After(expiry) {
				w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer resource_metadata="%s", error="invalid_token", error_description="token expired"`, resourceMetadataURL))
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// validateCredentials validates client credentials using constant-time comparison.
func (s *oauthServer) validateCredentials(clientID, clientSecret string) bool {
	idMatch := subtle.ConstantTimeCompare([]byte(clientID), []byte(s.clientID)) == 1
	secretMatch := subtle.ConstantTimeCompare([]byte(clientSecret), []byte(s.clientSecret)) == 1
	return idMatch && secretMatch
}

// Credentials returns the OAuth credentials.
func (s *oauthServer) Credentials() (clientID, clientSecret string) {
	return s.clientID, s.clientSecret
}

// writeError writes an OAuth error response.
func (s *oauthServer) writeError(w http.ResponseWriter, status int, errCode, description string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(tokenError{
		Error:       errCode,
		Description: description,
	})
}

// generateSecureToken generates a cryptographically secure random token.
func generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(bytes), nil
}

// runCleanup periodically removes expired tokens.
func (s *oauthServer) runCleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.cleanupExpiredTokens()
	}
}

// cleanupExpiredTokens removes expired tokens from the store.
func (s *oauthServer) cleanupExpiredTokens() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for token, expiry := range s.tokens {
		if now.After(expiry) {
			delete(s.tokens, token)
		}
	}
}
