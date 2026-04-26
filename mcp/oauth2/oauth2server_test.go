// Copyright 2025 John Wang. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package oauth2

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	t.Run("valid_config", func(t *testing.T) {
		srv, err := New(&Config{
			Issuer: "https://example.com",
			Users:  map[string]string{"admin": "password"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if srv == nil {
			t.Fatal("expected server to be non-nil")
		}
	})

	t.Run("missing_issuer", func(t *testing.T) {
		_, err := New(&Config{
			Users: map[string]string{"admin": "password"},
		})
		if err == nil {
			t.Fatal("expected error for missing issuer")
		}
	})

	t.Run("missing_users", func(t *testing.T) {
		_, err := New(&Config{
			Issuer: "https://example.com",
		})
		if err == nil {
			t.Fatal("expected error for missing users")
		}
	})
}

func TestDynamicClientRegistration(t *testing.T) {
	srv, err := New(&Config{
		Issuer: "https://example.com",
		Users:  map[string]string{"admin": "password"},
	})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	handler := srv.RegistrationHandler()
	ts := httptest.NewServer(handler)
	defer ts.Close()

	t.Run("register_public_client", func(t *testing.T) {
		reqBody := `{"redirect_uris":["https://app.example.com/callback"],"client_name":"Test App"}`
		resp, err := http.Post(ts.URL, "application/json", strings.NewReader(reqBody))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("warning: failed to close response body: %v", err)
			}
		}()

		if resp.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 201, got %d: %s", resp.StatusCode, body)
		}

		var result RegistrationResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if result.ClientID == "" {
			t.Error("expected non-empty client_id")
		}
		if result.ClientName != "Test App" {
			t.Errorf("expected client_name 'Test App', got %s", result.ClientName)
		}
		if len(result.RedirectURIs) != 1 || result.RedirectURIs[0] != "https://app.example.com/callback" {
			t.Errorf("unexpected redirect_uris: %v", result.RedirectURIs)
		}
	})

	t.Run("missing_redirect_uris", func(t *testing.T) {
		reqBody := `{"client_name":"Test App"}`
		resp, err := http.Post(ts.URL, "application/json", strings.NewReader(reqBody))
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
}

func TestPKCE(t *testing.T) {
	t.Run("generate_verifier", func(t *testing.T) {
		verifier, err := GenerateCodeVerifier()
		if err != nil {
			t.Fatalf("failed to generate verifier: %v", err)
		}
		if len(verifier) < MinVerifierLength {
			t.Errorf("verifier too short: %d", len(verifier))
		}
	})

	t.Run("generate_challenge", func(t *testing.T) {
		verifier, _ := GenerateCodeVerifier()
		challenge := GenerateCodeChallenge(verifier)
		if challenge == "" {
			t.Error("expected non-empty challenge")
		}
		if challenge == verifier {
			t.Error("challenge should not equal verifier")
		}
	})

	t.Run("verify_challenge", func(t *testing.T) {
		verifier, _ := GenerateCodeVerifier()
		challenge := GenerateCodeChallenge(verifier)

		err := VerifyCodeChallenge(verifier, challenge, PKCEMethodS256)
		if err != nil {
			t.Errorf("verification should succeed: %v", err)
		}
	})

	t.Run("verify_wrong_verifier", func(t *testing.T) {
		verifier1, _ := GenerateCodeVerifier()
		verifier2, _ := GenerateCodeVerifier()
		challenge := GenerateCodeChallenge(verifier1)

		err := VerifyCodeChallenge(verifier2, challenge, PKCEMethodS256)
		if err == nil {
			t.Error("verification should fail with wrong verifier")
		}
	})

	t.Run("validate_verifier_too_short", func(t *testing.T) {
		err := ValidateCodeVerifier("short")
		if err != ErrPKCEVerifierTooShort {
			t.Errorf("expected ErrPKCEVerifierTooShort, got %v", err)
		}
	})
}

func TestAuthorizationFlow(t *testing.T) {
	srv, err := New(&Config{
		Issuer: "https://example.com",
		Users:  map[string]string{"admin": "password"},
	})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// First register a client
	regHandler := srv.RegistrationHandler()
	regServer := httptest.NewServer(regHandler)
	defer regServer.Close()

	reqBody := `{"redirect_uris":["https://app.example.com/callback"]}`
	resp, err := http.Post(regServer.URL, "application/json", strings.NewReader(reqBody))
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("warning: failed to close response body: %v", err)
		}
	}()

	var client RegistrationResponse
	if err := json.NewDecoder(resp.Body).Decode(&client); err != nil {
		t.Fatalf("failed to decode client: %v", err)
	}

	// Create the authorization server
	authHandler := srv.AuthorizationHandler()
	authServer := httptest.NewServer(authHandler)
	defer authServer.Close()

	// Generate PKCE parameters
	verifier, _ := GenerateCodeVerifier()
	challenge := GenerateCodeChallenge(verifier)

	t.Run("authorization_request_shows_login", func(t *testing.T) {
		authURL := authServer.URL + "?" + url.Values{
			"client_id":             {client.ClientID},
			"redirect_uri":          {"https://app.example.com/callback"},
			"response_type":         {"code"},
			"code_challenge":        {challenge},
			"code_challenge_method": {"S256"},
			"state":                 {"test-state"},
		}.Encode()

		resp, err := http.Get(authURL) //nolint:gosec // G107: URL is from test server
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

		body, _ := io.ReadAll(resp.Body)
		if !strings.Contains(string(body), "Sign In") {
			t.Error("expected login page to contain 'Sign In'")
		}
	})

	t.Run("authorization_missing_pkce", func(t *testing.T) {
		// Follow redirects to check the error
		httpClient := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}

		authURL := authServer.URL + "?" + url.Values{
			"client_id":     {client.ClientID},
			"redirect_uri":  {"https://app.example.com/callback"},
			"response_type": {"code"},
			"state":         {"test-state"},
		}.Encode()

		resp, err := httpClient.Get(authURL)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("warning: failed to close response body: %v", err)
			}
		}()

		// Should redirect with error
		if resp.StatusCode != http.StatusFound {
			t.Errorf("expected 302 redirect, got %d", resp.StatusCode)
		}

		location := resp.Header.Get("Location")
		if !strings.Contains(location, "error=invalid_request") {
			t.Errorf("expected PKCE error in redirect, got: %s", location)
		}
	})
}

func TestTokenEndpoint(t *testing.T) {
	srv, err := New(&Config{
		Issuer: "https://example.com",
		Users:  map[string]string{"admin": "password"},
	})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	tokenHandler := srv.TokenHandler()
	ts := httptest.NewServer(tokenHandler)
	defer ts.Close()

	t.Run("unsupported_grant_type", func(t *testing.T) {
		resp, err := http.PostForm(ts.URL, url.Values{
			"grant_type": {"client_credentials"},
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

	t.Run("missing_code", func(t *testing.T) {
		resp, err := http.PostForm(ts.URL, url.Values{
			"grant_type": {"authorization_code"},
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
}

func TestMetadataEndpoint(t *testing.T) {
	srv, err := New(&Config{
		Issuer: "https://example.com",
		Users:  map[string]string{"admin": "password"},
	})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	handler := srv.MetadataHandler()
	ts := httptest.NewServer(handler)
	defer ts.Close()

	resp, err := http.Get(ts.URL)
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

	var metadata map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		t.Fatalf("failed to decode metadata: %v", err)
	}

	if metadata["issuer"] != "https://example.com" {
		t.Errorf("unexpected issuer: %v", metadata["issuer"])
	}

	if metadata["authorization_endpoint"] == nil {
		t.Error("expected authorization_endpoint in metadata")
	}

	if metadata["token_endpoint"] == nil {
		t.Error("expected token_endpoint in metadata")
	}

	grantTypes, ok := metadata["grant_types_supported"].([]interface{})
	if !ok {
		t.Error("expected grant_types_supported to be an array")
	} else {
		found := false
		for _, gt := range grantTypes {
			if gt == "authorization_code" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected authorization_code in grant_types_supported")
		}
	}

	codeMethods, ok := metadata["code_challenge_methods_supported"].([]interface{})
	if !ok {
		t.Error("expected code_challenge_methods_supported to be an array")
	} else {
		found := false
		for _, cm := range codeMethods {
			if cm == "S256" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected S256 in code_challenge_methods_supported")
		}
	}
}

func TestProtectedResourceMetadata(t *testing.T) {
	srv, err := New(&Config{
		Issuer: "https://example.com",
		Users:  map[string]string{"admin": "password"},
	})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	handler := srv.ProtectedResourceMetadataHandler("/mcp")
	ts := httptest.NewServer(handler)
	defer ts.Close()

	resp, err := http.Get(ts.URL)
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

	var metadata map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		t.Fatalf("failed to decode metadata: %v", err)
	}

	if metadata["resource"] != "https://example.com/mcp" {
		t.Errorf("unexpected resource: %v", metadata["resource"])
	}

	authServers, ok := metadata["authorization_servers"].([]interface{})
	if !ok || len(authServers) == 0 {
		t.Error("expected authorization_servers to be a non-empty array")
	}
}

func TestMemoryStorage(t *testing.T) {
	storage := NewMemoryStorage()

	t.Run("client_operations", func(t *testing.T) {
		client := &Client{
			ClientID:     "test-client",
			RedirectURIs: []string{"https://example.com/callback"},
		}

		if err := storage.CreateClient(client); err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		retrieved, err := storage.GetClient("test-client")
		if err != nil {
			t.Fatalf("failed to get client: %v", err)
		}
		if retrieved.ClientID != "test-client" {
			t.Errorf("unexpected client ID: %s", retrieved.ClientID)
		}

		if err := storage.DeleteClient("test-client"); err != nil {
			t.Fatalf("failed to delete client: %v", err)
		}

		_, err = storage.GetClient("test-client")
		if err != ErrClientNotFound {
			t.Errorf("expected ErrClientNotFound, got %v", err)
		}
	})
}

func TestBearerAuthMiddleware(t *testing.T) {
	srv, err := New(&Config{
		Issuer: "https://example.com",
		Users:  map[string]string{"admin": "password"},
	})
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	protectedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if token info is in context
		info := GetTokenInfoContext(r.Context())
		if info == nil {
			t.Error("expected token info in context")
		}
		w.WriteHeader(http.StatusOK)
	})

	middleware := srv.BearerAuthMiddleware("https://example.com/.well-known/oauth-protected-resource")
	ts := httptest.NewServer(middleware(protectedHandler))
	defer ts.Close()

	t.Run("missing_token", func(t *testing.T) {
		resp, err := http.Get(ts.URL)
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

		wwwAuth := resp.Header.Get("WWW-Authenticate")
		if !strings.Contains(wwwAuth, "Bearer") {
			t.Error("expected WWW-Authenticate header with Bearer")
		}
		if !strings.Contains(wwwAuth, "resource_metadata") {
			t.Error("expected resource_metadata in WWW-Authenticate")
		}
	})

	t.Run("invalid_token", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, ts.URL, nil)
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
}
