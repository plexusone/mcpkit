// Copyright 2025 John Wang. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

// Package oauth2 provides a standalone OAuth 2.1 Authorization Server
// with PKCE support, designed for MCP server authentication.
//
// Features:
//   - OAuth 2.1 Authorization Code Flow with PKCE (RFC 7636)
//   - Dynamic Client Registration (RFC 7591)
//   - Authorization Server Metadata (RFC 8414)
//   - Protected Resource Metadata (RFC 9728)
//   - Simple username/password authentication
//
// Usage:
//
//	srv, err := oauth2.New(&oauth2.Config{
//	    Issuer:   "https://example.com",
//	    Users:    map[string]string{"admin": "password"},
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Mount handlers on your HTTP mux
//	mux.Handle("/oauth/authorize", srv.AuthorizationHandler())
//	mux.Handle("/oauth/token", srv.TokenHandler())
//	mux.Handle("/oauth/register", srv.RegistrationHandler())
//	mux.Handle("/.well-known/oauth-authorization-server", srv.MetadataHandler())
package oauth2

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/grokify/mogo/log/slogutil"
)

// Config configures the OAuth 2.1 server.
type Config struct {
	// Issuer is the OAuth issuer URL (e.g., "https://example.com").
	// This is used in metadata and token responses.
	Issuer string

	// Users is a map of username to password for authentication.
	// For production, consider implementing a custom Authenticator.
	Users map[string]string

	// Authenticator is a custom authentication function.
	// If nil, Users map is used for authentication.
	Authenticator Authenticator

	// Storage is the storage backend for clients, codes, and tokens.
	// If nil, an in-memory storage is used.
	Storage Storage

	// AuthorizationCodeExpiry is how long authorization codes are valid.
	// Defaults to 10 minutes.
	AuthorizationCodeExpiry time.Duration

	// AccessTokenExpiry is how long access tokens are valid.
	// Defaults to 1 hour.
	AccessTokenExpiry time.Duration

	// RefreshTokenExpiry is how long refresh tokens are valid.
	// Defaults to 24 hours. Set to 0 to disable refresh tokens.
	RefreshTokenExpiry time.Duration

	// AllowedScopes is the list of scopes this server supports.
	// If empty, no scope validation is performed.
	AllowedScopes []string

	// LoginPageTemplate is custom HTML for the login page.
	// If empty, a default login page is used.
	LoginPageTemplate string

	// Paths configures the endpoint paths.
	Paths *PathConfig

	// Logger is used for debug logging. If nil, uses slog.Default().
	Logger *slog.Logger

	// Debug enables verbose debug logging.
	Debug bool
}

// PathConfig configures the OAuth endpoint paths.
type PathConfig struct {
	// Authorization is the authorization endpoint path. Defaults to "/oauth/authorize".
	Authorization string

	// Token is the token endpoint path. Defaults to "/oauth/token".
	Token string

	// Registration is the dynamic client registration path. Defaults to "/oauth/register".
	Registration string

	// Metadata is the authorization server metadata path.
	// Defaults to "/.well-known/oauth-authorization-server".
	Metadata string
}

// DefaultPaths returns the default OAuth endpoint paths.
func DefaultPaths() *PathConfig {
	//nolint:gosec // G101: These are URL paths, not credentials
	return &PathConfig{
		Authorization: "/oauth/authorize",
		Token:         "/oauth/token",
		Registration:  "/oauth/register",
		Metadata:      "/.well-known/oauth-authorization-server",
	}
}

// Authenticator is a function that validates user credentials.
// Returns true if the credentials are valid.
type Authenticator func(username, password string) bool

// Server is an OAuth 2.1 Authorization Server with PKCE support.
type Server struct {
	config  *Config
	storage Storage
	paths   *PathConfig

	// authenticator validates user credentials
	authenticator Authenticator

	// logger for debug output
	logger *slog.Logger
	debug  bool
}

// New creates a new OAuth 2.1 server with the given configuration.
func New(cfg *Config) (*Server, error) {
	if cfg == nil {
		cfg = &Config{}
	}

	if cfg.Issuer == "" {
		return nil, fmt.Errorf("issuer is required")
	}

	// Set defaults
	if cfg.AuthorizationCodeExpiry == 0 {
		cfg.AuthorizationCodeExpiry = 10 * time.Minute
	}
	if cfg.AccessTokenExpiry == 0 {
		cfg.AccessTokenExpiry = time.Hour
	}
	if cfg.RefreshTokenExpiry == 0 {
		cfg.RefreshTokenExpiry = 24 * time.Hour
	}

	paths := cfg.Paths
	if paths == nil {
		paths = DefaultPaths()
	}

	// Set up storage
	storage := cfg.Storage
	if storage == nil {
		storage = NewMemoryStorage()
	}

	// Set up authenticator
	var authenticator Authenticator
	if cfg.Authenticator != nil {
		authenticator = cfg.Authenticator
	} else if len(cfg.Users) > 0 {
		authenticator = func(username, password string) bool {
			if pw, ok := cfg.Users[username]; ok {
				return pw == password
			}
			return false
		}
	} else {
		return nil, fmt.Errorf("either Users or Authenticator must be provided")
	}

	// Set up logger
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &Server{
		config:        cfg,
		storage:       storage,
		paths:         paths,
		authenticator: authenticator,
		logger:        logger.With("component", "oauth2"),
		debug:         cfg.Debug,
	}, nil
}

// logDebug logs a debug message if debug mode is enabled.
func (s *Server) logDebug(msg string, args ...any) {
	if s.debug {
		s.logger.Debug(msg, args...)
	}
}

// logDebugCtx logs a debug message using a logger from context if available.
func (s *Server) logDebugCtx(ctx context.Context, msg string, args ...any) {
	if !s.debug {
		return
	}
	logger := slogutil.LoggerFromContext(ctx, s.logger)
	logger.Debug(msg, args...)
}

// Logger returns the server's logger.
func (s *Server) Logger() *slog.Logger {
	return s.logger
}

// Config returns a copy of the server configuration.
func (s *Server) Config() *Config {
	return s.config
}

// Paths returns the configured endpoint paths.
func (s *Server) Paths() *PathConfig {
	return s.paths
}

// AuthorizationHandler returns the HTTP handler for the authorization endpoint.
// This endpoint handles the OAuth 2.1 authorization code flow with PKCE.
func (s *Server) AuthorizationHandler() http.Handler {
	return s.authorizationHandler()
}

// TokenHandler returns the HTTP handler for the token endpoint.
// This endpoint exchanges authorization codes for access tokens.
func (s *Server) TokenHandler() http.Handler {
	return s.tokenHandler()
}

// RegistrationHandler returns the HTTP handler for dynamic client registration.
// This endpoint allows clients to register themselves (RFC 7591).
func (s *Server) RegistrationHandler() http.Handler {
	return s.registrationHandler()
}

// MetadataHandler returns the HTTP handler for authorization server metadata.
// This should be mounted at /.well-known/oauth-authorization-server (RFC 8414).
func (s *Server) MetadataHandler() http.Handler {
	return s.metadataHandler()
}

// ProtectedResourceMetadataHandler returns the HTTP handler for protected
// resource metadata (RFC 9728). The resourcePath is the path to the protected
// resource (e.g., "/mcp").
func (s *Server) ProtectedResourceMetadataHandler(resourcePath string) http.Handler {
	return s.protectedResourceMetadataHandler(resourcePath)
}

// TokenVerifier returns a function that verifies access tokens.
// This can be used with middleware to protect resources.
func (s *Server) TokenVerifier() func(token string) (*TokenInfo, error) {
	return func(token string) (*TokenInfo, error) {
		return s.storage.GetToken(token)
	}
}

// RegisterHandlers registers all OAuth handlers on the given mux using
// the configured paths. This is a convenience method for simple setups.
func (s *Server) RegisterHandlers(mux *http.ServeMux) {
	mux.Handle(s.paths.Authorization, s.AuthorizationHandler())
	mux.Handle(s.paths.Token, s.TokenHandler())
	mux.Handle(s.paths.Registration, s.RegistrationHandler())
	mux.Handle(s.paths.Metadata, s.MetadataHandler())
}

// RegisterClient pre-registers a client with the given credentials.
// This is useful for clients like ChatGPT.com that require you to enter
// client credentials during configuration rather than using DCR.
// If clientID or clientSecret is empty, they will be auto-generated.
// If redirectURIs is empty, all URIs will be allowed.
func (s *Server) RegisterClient(clientID, clientSecret string, redirectURIs []string) (string, string, error) {
	var err error

	if clientID == "" {
		clientID, err = GenerateClientID()
		if err != nil {
			return "", "", fmt.Errorf("generating client ID: %w", err)
		}
	}

	if clientSecret == "" {
		clientSecret, err = GenerateClientSecret()
		if err != nil {
			return "", "", fmt.Errorf("generating client secret: %w", err)
		}
	}

	// If no redirect URIs specified, use a wildcard marker
	if len(redirectURIs) == 0 {
		redirectURIs = []string{"*"} // Special marker for "allow any"
	}

	client := &Client{
		ClientID:                clientID,
		ClientSecret:            clientSecret,
		ClientName:              "Pre-registered Client",
		RedirectURIs:            redirectURIs,
		GrantTypes:              []string{"authorization_code", "refresh_token"},
		ResponseTypes:           []string{"code"},
		TokenEndpointAuthMethod: "client_secret_post",
	}

	if err := s.storage.CreateClient(client); err != nil {
		return "", "", fmt.Errorf("storing client: %w", err)
	}

	return clientID, clientSecret, nil
}
