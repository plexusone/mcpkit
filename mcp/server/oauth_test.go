// Copyright 2025 John Wang. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestOAuthServer_TokenEndpoint(t *testing.T) {
	srv, err := newOAuthServer(&OAuthOptions{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		TokenExpiry:  time.Hour,
	})
	if err != nil {
		t.Fatalf("failed to create oauth server: %v", err)
	}

	handler := srv.TokenHandler()
	ts := httptest.NewServer(handler)
	defer ts.Close()

	t.Run("valid_credentials_form", func(t *testing.T) {
		resp, err := http.PostForm(ts.URL, url.Values{
			"grant_type":    {"client_credentials"},
			"client_id":     {"test-client-id"},
			"client_secret": {"test-client-secret"},
		})
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("warning: failed to close response body: %v", err)
			}
		}()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
		}

		var tokenResp tokenResponse
		if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if tokenResp.AccessToken == "" {
			t.Error("expected non-empty access token")
		}
		if tokenResp.TokenType != "Bearer" {
			t.Errorf("expected Bearer token type, got %s", tokenResp.TokenType)
		}
		if tokenResp.ExpiresIn != 3600 {
			t.Errorf("expected 3600 expires_in, got %d", tokenResp.ExpiresIn)
		}
	})

	t.Run("valid_credentials_basic_auth", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, ts.URL, strings.NewReader("grant_type=client_credentials"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth("test-client-id", "test-client-secret")

		resp, err := http.DefaultClient.Do(req) //nolint:gosec // G704: Test uses httptest server URL
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("warning: failed to close response body: %v", err)
			}
		}()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
		}
	})

	t.Run("invalid_credentials", func(t *testing.T) {
		resp, err := http.PostForm(ts.URL, url.Values{
			"grant_type":    {"client_credentials"},
			"client_id":     {"wrong-id"},
			"client_secret": {"wrong-secret"},
		})
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("warning: failed to close response body: %v", err)
			}
		}()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", resp.StatusCode)
		}
	})

	t.Run("unsupported_grant_type", func(t *testing.T) {
		resp, err := http.PostForm(ts.URL, url.Values{
			"grant_type":    {"authorization_code"},
			"client_id":     {"test-client-id"},
			"client_secret": {"test-client-secret"},
		})
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("warning: failed to close response body: %v", err)
			}
		}()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", resp.StatusCode)
		}
	})

	t.Run("method_not_allowed", func(t *testing.T) {
		resp, err := http.Get(ts.URL)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("warning: failed to close response body: %v", err)
			}
		}()

		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("expected 405, got %d", resp.StatusCode)
		}
	})
}

func TestOAuthServer_BearerAuthMiddleware(t *testing.T) {
	srv, err := newOAuthServer(&OAuthOptions{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		TokenExpiry:  time.Hour,
	})
	if err != nil {
		t.Fatalf("failed to create oauth server: %v", err)
	}

	// Get a token via the token endpoint
	tokenHandler := srv.TokenHandler()
	tokenServer := httptest.NewServer(tokenHandler)
	defer tokenServer.Close()

	resp, err := http.PostForm(tokenServer.URL, url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {"test-client-id"},
		"client_secret": {"test-client-secret"},
	})
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("warning: failed to close response body: %v", err)
		}
	}()

	var tokenResp tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Create a protected handler
	protectedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("protected content"))
	})

	// Wrap with auth middleware
	middleware := srv.BearerAuthMiddleware("https://example.com/.well-known/oauth-protected-resource")
	protectedServer := httptest.NewServer(middleware(protectedHandler))
	defer protectedServer.Close()

	t.Run("valid_token", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, protectedServer.URL, nil)
		req.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)

		resp, err := http.DefaultClient.Do(req) //nolint:gosec // G704: Test uses httptest server URL
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("warning: failed to close response body: %v", err)
			}
		}()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
	})

	t.Run("invalid_token", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, protectedServer.URL, nil)
		req.Header.Set("Authorization", "Bearer invalid-token")

		resp, err := http.DefaultClient.Do(req) //nolint:gosec // G704: Test uses httptest server URL
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("warning: failed to close response body: %v", err)
			}
		}()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", resp.StatusCode)
		}
	})

	t.Run("missing_token", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, protectedServer.URL, nil)

		resp, err := http.DefaultClient.Do(req) //nolint:gosec // G704: Test uses httptest server URL
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("warning: failed to close response body: %v", err)
			}
		}()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", resp.StatusCode)
		}

		// Should include WWW-Authenticate header
		wwwAuth := resp.Header.Get("WWW-Authenticate")
		if wwwAuth == "" {
			t.Error("expected WWW-Authenticate header")
		}
	})
}

func TestOAuthServer_AutoGenerateCredentials(t *testing.T) {
	srv, err := newOAuthServer(&OAuthOptions{})
	if err != nil {
		t.Fatalf("failed to create oauth server: %v", err)
	}

	clientID, clientSecret := srv.Credentials()

	if clientID == "" {
		t.Error("expected auto-generated client ID")
	}
	if clientSecret == "" {
		t.Error("expected auto-generated client secret")
	}
	if len(clientID) < 20 {
		t.Errorf("client ID too short: %d", len(clientID))
	}
	if len(clientSecret) < 40 {
		t.Errorf("client secret too short: %d", len(clientSecret))
	}
}

func TestServeHTTP_WithOAuth(t *testing.T) {
	rt := New(&mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	port := 19284
	addr := fmt.Sprintf("localhost:%d", port)

	resultChan := make(chan *HTTPServerResult, 1)
	errChan := make(chan error, 1)

	go func() {
		result, err := rt.ServeHTTP(ctx, &HTTPServerOptions{
			Addr: addr,
			OAuth: &OAuthOptions{
				ClientID:     "test-id",
				ClientSecret: "test-secret",
			},
			OnReady: func(result *HTTPServerResult) {
				resultChan <- result
			},
		})
		if err != nil {
			errChan <- err
		}
		_ = result
	}()

	// Wait for server to be ready
	var result *HTTPServerResult
	select {
	case result = <-resultChan:
	case err := <-errChan:
		t.Fatalf("server error: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for server to start")
	}

	// Verify OAuth credentials are returned
	if result.OAuth == nil {
		t.Fatal("expected OAuth credentials in result")
	}
	if result.OAuth.ClientID != "test-id" {
		t.Errorf("expected client ID 'test-id', got %s", result.OAuth.ClientID)
	}
	if result.OAuth.ClientSecret != "test-secret" {
		t.Errorf("expected client secret 'test-secret', got %s", result.OAuth.ClientSecret)
	}
	if !strings.Contains(result.OAuth.TokenEndpoint, "/oauth/token") {
		t.Errorf("expected token endpoint to contain /oauth/token, got %s", result.OAuth.TokenEndpoint)
	}

	// Test getting a token
	tokenResp, err := http.PostForm(result.OAuth.TokenEndpoint, url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {"test-id"},
		"client_secret": {"test-secret"},
	})
	if err != nil {
		t.Fatalf("token request failed: %v", err)
	}
	defer func() {
		if err := tokenResp.Body.Close(); err != nil {
			t.Logf("warning: failed to close response body: %v", err)
		}
	}()

	if tokenResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(tokenResp.Body)
		t.Fatalf("expected 200 for token, got %d: %s", tokenResp.StatusCode, body)
	}

	var token tokenResponse
	if err := json.NewDecoder(tokenResp.Body).Decode(&token); err != nil {
		t.Fatalf("failed to decode token: %v", err)
	}

	// Test MCP endpoint without token (should fail)
	mcpURL := result.LocalURL
	resp, err := http.Get(mcpURL) //nolint:gosec // G107: URL is from test result, not user input
	if err != nil {
		t.Fatalf("MCP request failed: %v", err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Logf("warning: failed to close response body: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 without token, got %d", resp.StatusCode)
	}

	// Test MCP endpoint with token (should succeed with 400 since we're not using proper MCP protocol)
	req, _ := http.NewRequest(http.MethodGet, mcpURL, nil)
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Accept", "text/event-stream")

	resp, err = http.DefaultClient.Do(req) //nolint:gosec // G704: Test uses httptest server URL
	if err != nil {
		t.Fatalf("MCP request with token failed: %v", err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Logf("warning: failed to close response body: %v", err)
	}

	// Should not be 401 anymore (token is valid)
	if resp.StatusCode == http.StatusUnauthorized {
		t.Error("expected authenticated request to not return 401")
	}

	cancel()
}
