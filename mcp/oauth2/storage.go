// Copyright 2025 John Wang. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package oauth2

import (
	"errors"
	"sync"
	"time"
)

// Common errors returned by storage operations.
var (
	ErrClientNotFound = errors.New("client not found")
	ErrCodeNotFound   = errors.New("authorization code not found")
	ErrCodeExpired    = errors.New("authorization code expired")
	ErrCodeUsed       = errors.New("authorization code already used")
	ErrTokenNotFound  = errors.New("token not found")
	ErrTokenExpired   = errors.New("token expired")
)

// Storage defines the interface for OAuth data persistence.
type Storage interface {
	// Client operations
	CreateClient(client *Client) error
	GetClient(clientID string) (*Client, error)
	DeleteClient(clientID string) error

	// Authorization code operations
	CreateAuthorizationCode(code *AuthorizationCode) error
	GetAuthorizationCode(code string) (*AuthorizationCode, error)
	DeleteAuthorizationCode(code string) error
	MarkAuthorizationCodeUsed(code string) error

	// Token operations
	CreateToken(token *TokenInfo) error
	GetToken(accessToken string) (*TokenInfo, error)
	GetTokenByRefresh(refreshToken string) (*TokenInfo, error)
	DeleteToken(accessToken string) error
	DeleteTokensByClient(clientID string) error
}

// Client represents a registered OAuth client.
//
//nolint:gosec // G117: OAuth struct fields, not hardcoded secrets
type Client struct {
	// ClientID is the unique client identifier.
	ClientID string `json:"client_id"`

	// ClientSecret is the client secret (may be empty for public clients).
	ClientSecret string `json:"client_secret,omitempty"`

	// ClientName is the human-readable name of the client.
	ClientName string `json:"client_name,omitempty"`

	// RedirectURIs is the list of allowed redirect URIs.
	RedirectURIs []string `json:"redirect_uris"`

	// GrantTypes is the list of allowed grant types.
	// Defaults to ["authorization_code"] if empty.
	GrantTypes []string `json:"grant_types,omitempty"`

	// ResponseTypes is the list of allowed response types.
	// Defaults to ["code"] if empty.
	ResponseTypes []string `json:"response_types,omitempty"`

	// TokenEndpointAuthMethod is the authentication method for the token endpoint.
	// Values: "none", "client_secret_basic", "client_secret_post"
	TokenEndpointAuthMethod string `json:"token_endpoint_auth_method,omitempty"`

	// CreatedAt is when the client was registered.
	CreatedAt time.Time `json:"created_at"`
}

// AuthorizationCode represents a pending authorization code.
type AuthorizationCode struct {
	// Code is the authorization code value.
	Code string `json:"code"`

	// ClientID is the client that requested this code.
	ClientID string `json:"client_id"`

	// RedirectURI is the redirect URI used in the authorization request.
	RedirectURI string `json:"redirect_uri"`

	// Scope is the requested scope.
	Scope string `json:"scope,omitempty"`

	// CodeChallenge is the PKCE code challenge.
	CodeChallenge string `json:"code_challenge"`

	// CodeChallengeMethod is the PKCE challenge method (always "S256").
	CodeChallengeMethod string `json:"code_challenge_method"`

	// Subject is the authenticated user.
	Subject string `json:"subject"`

	// ExpiresAt is when this code expires.
	ExpiresAt time.Time `json:"expires_at"`

	// Used indicates if this code has been exchanged.
	Used bool `json:"used"`

	// CreatedAt is when this code was created.
	CreatedAt time.Time `json:"created_at"`
}

// TokenInfo represents an issued access token.
//
//nolint:gosec // G117: OAuth struct fields, not hardcoded secrets
type TokenInfo struct {
	// AccessToken is the access token value.
	AccessToken string `json:"access_token"`

	// RefreshToken is the refresh token value (if issued).
	RefreshToken string `json:"refresh_token,omitempty"`

	// TokenType is always "Bearer".
	TokenType string `json:"token_type"`

	// ClientID is the client this token was issued to.
	ClientID string `json:"client_id"`

	// Subject is the authenticated user.
	Subject string `json:"subject"`

	// Scope is the granted scope.
	Scope string `json:"scope,omitempty"`

	// ExpiresAt is when the access token expires.
	ExpiresAt time.Time `json:"expires_at"`

	// RefreshExpiresAt is when the refresh token expires.
	RefreshExpiresAt time.Time `json:"refresh_expires_at,omitempty"`

	// CreatedAt is when this token was created.
	CreatedAt time.Time `json:"created_at"`
}

// IsExpired returns true if the access token has expired.
func (t *TokenInfo) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// MemoryStorage is an in-memory implementation of Storage.
// Suitable for development and testing.
type MemoryStorage struct {
	mu      sync.RWMutex
	clients map[string]*Client
	codes   map[string]*AuthorizationCode
	tokens  map[string]*TokenInfo
	refresh map[string]string // refresh token -> access token
}

// NewMemoryStorage creates a new in-memory storage.
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		clients: make(map[string]*Client),
		codes:   make(map[string]*AuthorizationCode),
		tokens:  make(map[string]*TokenInfo),
		refresh: make(map[string]string),
	}
}

// CreateClient stores a new client.
func (m *MemoryStorage) CreateClient(client *Client) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clients[client.ClientID] = client
	return nil
}

// GetClient retrieves a client by ID.
func (m *MemoryStorage) GetClient(clientID string) (*Client, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	client, ok := m.clients[clientID]
	if !ok {
		return nil, ErrClientNotFound
	}
	return client, nil
}

// DeleteClient removes a client.
func (m *MemoryStorage) DeleteClient(clientID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.clients, clientID)
	return nil
}

// CreateAuthorizationCode stores a new authorization code.
func (m *MemoryStorage) CreateAuthorizationCode(code *AuthorizationCode) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.codes[code.Code] = code
	return nil
}

// GetAuthorizationCode retrieves an authorization code.
func (m *MemoryStorage) GetAuthorizationCode(code string) (*AuthorizationCode, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	authCode, ok := m.codes[code]
	if !ok {
		return nil, ErrCodeNotFound
	}
	if authCode.Used {
		return nil, ErrCodeUsed
	}
	if time.Now().After(authCode.ExpiresAt) {
		return nil, ErrCodeExpired
	}
	return authCode, nil
}

// DeleteAuthorizationCode removes an authorization code.
func (m *MemoryStorage) DeleteAuthorizationCode(code string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.codes, code)
	return nil
}

// MarkAuthorizationCodeUsed marks a code as used.
func (m *MemoryStorage) MarkAuthorizationCodeUsed(code string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	authCode, ok := m.codes[code]
	if !ok {
		return ErrCodeNotFound
	}
	authCode.Used = true
	return nil
}

// CreateToken stores a new token.
func (m *MemoryStorage) CreateToken(token *TokenInfo) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tokens[token.AccessToken] = token
	if token.RefreshToken != "" {
		m.refresh[token.RefreshToken] = token.AccessToken
	}
	return nil
}

// GetToken retrieves a token by access token.
func (m *MemoryStorage) GetToken(accessToken string) (*TokenInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	token, ok := m.tokens[accessToken]
	if !ok {
		return nil, ErrTokenNotFound
	}
	if token.IsExpired() {
		return nil, ErrTokenExpired
	}
	return token, nil
}

// GetTokenByRefresh retrieves a token by refresh token.
func (m *MemoryStorage) GetTokenByRefresh(refreshToken string) (*TokenInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	accessToken, ok := m.refresh[refreshToken]
	if !ok {
		return nil, ErrTokenNotFound
	}
	token, ok := m.tokens[accessToken]
	if !ok {
		return nil, ErrTokenNotFound
	}
	// Check refresh token expiry
	if !token.RefreshExpiresAt.IsZero() && time.Now().After(token.RefreshExpiresAt) {
		return nil, ErrTokenExpired
	}
	return token, nil
}

// DeleteToken removes a token.
func (m *MemoryStorage) DeleteToken(accessToken string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	token, ok := m.tokens[accessToken]
	if ok && token.RefreshToken != "" {
		delete(m.refresh, token.RefreshToken)
	}
	delete(m.tokens, accessToken)
	return nil
}

// DeleteTokensByClient removes all tokens for a client.
func (m *MemoryStorage) DeleteTokensByClient(clientID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for at, token := range m.tokens {
		if token.ClientID == clientID {
			if token.RefreshToken != "" {
				delete(m.refresh, token.RefreshToken)
			}
			delete(m.tokens, at)
		}
	}
	return nil
}

// Cleanup removes expired codes and tokens. Call periodically.
func (m *MemoryStorage) Cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()

	// Clean expired codes
	for code, authCode := range m.codes {
		if now.After(authCode.ExpiresAt) {
			delete(m.codes, code)
		}
	}

	// Clean expired tokens
	for at, token := range m.tokens {
		if now.After(token.ExpiresAt) {
			if token.RefreshToken != "" {
				delete(m.refresh, token.RefreshToken)
			}
			delete(m.tokens, at)
		}
	}
}

// StartCleanup starts a background goroutine that periodically cleans up
// expired codes and tokens. Returns a stop function to halt cleanup.
func (m *MemoryStorage) StartCleanup(interval time.Duration) func() {
	stop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				m.Cleanup()
			case <-stop:
				return
			}
		}
	}()
	return func() { close(stop) }
}
