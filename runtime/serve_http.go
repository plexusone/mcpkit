// Copyright 2025 John Wang. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package runtime

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/plexusone/mcpkit/oauth2"
	"golang.ngrok.com/ngrok"
	"golang.ngrok.com/ngrok/config"
)

// HTTPServerOptions configures HTTP-based serving.
type HTTPServerOptions struct {
	// Addr is the local address to listen on (e.g., ":8080").
	// Required when Ngrok is nil. When Ngrok is configured, this is optional
	// and defaults to a random available port.
	Addr string

	// Path is the HTTP path for the MCP endpoint. Defaults to "/mcp".
	Path string

	// ReadHeaderTimeout is the timeout for reading request headers.
	// Defaults to 10 seconds.
	ReadHeaderTimeout time.Duration

	// Ngrok configures optional ngrok tunneling. When set, the server
	// is exposed via ngrok and the PublicURL in the result will be populated.
	Ngrok *NgrokOptions

	// StreamableHTTPOptions are passed to the MCP StreamableHTTP handler.
	StreamableHTTPOptions *mcp.StreamableHTTPOptions

	// OAuth configures simple OAuth 2.0 client credentials authentication.
	// When set, the /mcp endpoint requires a Bearer token and a token
	// endpoint is exposed at /oauth/token (or OAuth.TokenPath if set).
	// Deprecated: Use OAuth2 for full OAuth 2.1 with PKCE support (required by ChatGPT.com).
	OAuth *OAuthOptions

	// OAuth2 configures full OAuth 2.1 Authorization Code + PKCE authentication.
	// This is required for ChatGPT.com and other clients that need DCR and PKCE.
	// When set, the following endpoints are exposed:
	//   - /oauth/authorize - Authorization endpoint with login page
	//   - /oauth/token - Token endpoint
	//   - /oauth/register - Dynamic Client Registration
	//   - /.well-known/oauth-authorization-server - Metadata
	//   - /.well-known/oauth-protected-resource - Resource metadata
	OAuth2 *OAuth2Options

	// OnReady is called when the server is ready to accept connections,
	// before ServeHTTP blocks. This is useful for logging the server URL.
	OnReady func(result *HTTPServerResult)
}

// NgrokOptions configures ngrok tunneling.
type NgrokOptions struct {
	// Authtoken is the ngrok authentication token.
	// If empty, uses the NGROK_AUTHTOKEN environment variable.
	Authtoken string

	// Domain is an optional custom ngrok domain (e.g., "myapp.ngrok.io").
	// Requires a paid ngrok plan.
	Domain string
}

// OAuth2Options configures OAuth 2.1 Authorization Code + PKCE authentication.
type OAuth2Options struct {
	// Users is a map of username to password for authentication.
	// At least one user must be configured.
	Users map[string]string

	// ClientID is the pre-registered OAuth client ID.
	// If empty, one will be auto-generated.
	ClientID string

	// ClientSecret is the pre-registered OAuth client secret.
	// If empty, one will be auto-generated.
	ClientSecret string

	// RedirectURIs is the list of allowed redirect URIs for the pre-registered client.
	// Defaults to allowing any URI (for flexibility with ChatGPT.com).
	RedirectURIs []string

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

	// Debug enables verbose logging for OAuth operations.
	Debug bool
}

// OAuth2Credentials contains the OAuth 2.1 server information.
type OAuth2Credentials struct {
	// ClientID is the pre-registered client ID.
	ClientID string

	// ClientSecret is the pre-registered client secret.
	ClientSecret string

	// AuthorizationEndpoint is the authorization URL.
	AuthorizationEndpoint string

	// TokenEndpoint is the token URL.
	TokenEndpoint string

	// RegistrationEndpoint is the DCR URL.
	RegistrationEndpoint string

	// Users is the map of configured users (for display/logging).
	Users []string
}

// HTTPServerResult contains information about the running HTTP server.
type HTTPServerResult struct {
	// LocalAddr is the local address the server is listening on (e.g., "localhost:8080").
	LocalAddr string

	// LocalURL is the full local URL including path (e.g., "http://localhost:8080/mcp").
	LocalURL string

	// PublicURL is the ngrok public URL including path, if ngrok is enabled.
	// Empty string if ngrok is not configured.
	PublicURL string

	// OAuth contains the OAuth credentials if OAuth (client_credentials) is enabled.
	// Nil if OAuth is not configured.
	// Deprecated: Use OAuth2 for full OAuth 2.1 support.
	OAuth *OAuthCredentials

	// OAuth2 contains the OAuth 2.1 server information if OAuth2 is enabled.
	// Nil if OAuth2 is not configured.
	OAuth2 *OAuth2Credentials
}

// ServeHTTP starts an HTTP server for the MCP runtime.
//
// When opts.Ngrok is configured, the server is exposed via ngrok tunnel
// and the returned result includes the public URL.
//
// ServeHTTP blocks until the context is cancelled, at which point it
// performs a graceful shutdown.
//
// Example without ngrok:
//
//	result, err := rt.ServeHTTP(ctx, &mcpkit.HTTPServerOptions{
//	    Addr: ":8080",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	log.Printf("MCP server running at %s", result.LocalURL)
//
// Example with ngrok:
//
//	result, err := rt.ServeHTTP(ctx, &mcpkit.HTTPServerOptions{
//	    Ngrok: &mcpkit.NgrokOptions{
//	        Authtoken: os.Getenv("NGROK_AUTHTOKEN"),
//	    },
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	log.Printf("MCP server running at %s", result.PublicURL)
func (r *Runtime) ServeHTTP(ctx context.Context, opts *HTTPServerOptions) (*HTTPServerResult, error) {
	if opts == nil {
		opts = &HTTPServerOptions{}
	}

	path := opts.Path
	if path == "" {
		path = "/mcp"
	}

	readHeaderTimeout := opts.ReadHeaderTimeout
	if readHeaderTimeout == 0 {
		readHeaderTimeout = 10 * time.Second
	}

	// Determine token path for OAuth
	tokenPath := "/oauth/token" //nolint:gosec // G101: this is a URL path, not credentials
	if opts.OAuth != nil && opts.OAuth.TokenPath != "" {
		tokenPath = opts.OAuth.TokenPath
	}

	// Create listener FIRST to determine the base URL
	var listener net.Listener
	var err error
	var baseURL string

	result := &HTTPServerResult{}

	if opts.Ngrok != nil {
		listener, err = r.createNgrokListener(ctx, opts)
		if err != nil {
			return nil, fmt.Errorf("creating ngrok listener: %w", err)
		}
		// ngrok listener address is just hostname, need to add https scheme
		baseURL = "https://" + listener.Addr().String()
		result.PublicURL = baseURL + path

		// For ngrok, also set local info if Addr is specified
		if opts.Addr != "" {
			result.LocalAddr = opts.Addr
			result.LocalURL = "http://" + opts.Addr + path
		}
	} else {
		if opts.Addr == "" {
			return nil, fmt.Errorf("addr is required when ngrok is not configured")
		}
		listener, err = net.Listen("tcp", opts.Addr)
		if err != nil {
			return nil, fmt.Errorf("listening on %s: %w", opts.Addr, err)
		}
		result.LocalAddr = listener.Addr().String()
		result.LocalURL = "http://" + result.LocalAddr + path
		baseURL = "http://" + result.LocalAddr
	}

	// Now set up handlers with the known base URL
	mcpHandler := r.StreamableHTTPHandler(opts.StreamableHTTPOptions)
	mux := http.NewServeMux()

	// Set up OAuth 2.1 with PKCE if configured (preferred over legacy OAuth)
	if opts.OAuth2 != nil {
		oauth2Srv, err := oauth2.New(&oauth2.Config{
			Issuer:             baseURL,
			Users:              opts.OAuth2.Users,
			AccessTokenExpiry:  opts.OAuth2.AccessTokenExpiry,
			RefreshTokenExpiry: opts.OAuth2.RefreshTokenExpiry,
			AllowedScopes:      opts.OAuth2.AllowedScopes,
			LoginPageTemplate:  opts.OAuth2.LoginPageTemplate,
			Debug:              opts.OAuth2.Debug,
		})
		if err != nil {
			_ = listener.Close()
			return nil, fmt.Errorf("creating oauth2 server: %w", err)
		}

		// Pre-register a client for ChatGPT.com (requires client credentials during setup)
		clientID, clientSecret, err := oauth2Srv.RegisterClient(
			opts.OAuth2.ClientID,
			opts.OAuth2.ClientSecret,
			opts.OAuth2.RedirectURIs,
		)
		if err != nil {
			_ = listener.Close()
			return nil, fmt.Errorf("registering oauth2 client: %w", err)
		}

		// Register OAuth 2.1 handlers
		paths := oauth2Srv.Paths()
		mux.Handle(paths.Authorization, oauth2Srv.AuthorizationHandler())
		mux.Handle(paths.Token, oauth2Srv.TokenHandler())
		mux.Handle(paths.Registration, oauth2Srv.RegistrationHandler())
		mux.Handle(paths.Metadata, oauth2Srv.MetadataHandler())
		mux.Handle("/.well-known/oauth-protected-resource", oauth2Srv.ProtectedResourceMetadataHandler(path))

		// Wrap MCP handler with auth middleware
		resourceMetadataURL := baseURL + "/.well-known/oauth-protected-resource"
		mcpHandler = oauth2Srv.BearerAuthMiddleware(resourceMetadataURL)(mcpHandler)

		// Store OAuth2 info in result
		var users []string
		for u := range opts.OAuth2.Users {
			users = append(users, u)
		}
		result.OAuth2 = &OAuth2Credentials{
			ClientID:              clientID,
			ClientSecret:          clientSecret,
			AuthorizationEndpoint: baseURL + paths.Authorization,
			TokenEndpoint:         baseURL + paths.Token,
			RegistrationEndpoint:  baseURL + paths.Registration,
			Users:                 users,
		}
	} else if opts.OAuth != nil {
		// Legacy OAuth client_credentials flow
		oauthSrv, err := newOAuthServer(opts.OAuth)
		if err != nil {
			_ = listener.Close()
			return nil, fmt.Errorf("creating oauth server: %w", err)
		}

		// Token endpoint
		mux.Handle(tokenPath, oauthSrv.TokenHandler())

		// Authorization endpoint (placeholder for OAuth discovery)
		mux.Handle("/oauth/authorize", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			http.Error(w, "Authorization code flow not supported. Use client_credentials grant.", http.StatusNotImplemented)
		}))

		// Registration endpoint (placeholder for OAuth discovery)
		mux.Handle("/oauth/register", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			http.Error(w, "Dynamic client registration not supported.", http.StatusNotImplemented)
		}))

		// OAuth metadata discovery endpoints (RFC 8414 and RFC 9728)
		mux.Handle("/.well-known/oauth-authorization-server", AuthorizationServerMetadataHandler(tokenPath))
		mux.Handle("/.well-known/oauth-protected-resource", ProtectedResourceMetadataHandler(path))

		// Wrap MCP handler with auth middleware using ABSOLUTE URL
		resourceMetadataURL := baseURL + "/.well-known/oauth-protected-resource"
		mcpHandler = oauthSrv.BearerAuthMiddleware(resourceMetadataURL)(mcpHandler)

		// Store credentials in result
		clientID, clientSecret := oauthSrv.Credentials()
		result.OAuth = &OAuthCredentials{
			ClientID:      clientID,
			ClientSecret:  clientSecret,
			TokenEndpoint: baseURL + tokenPath,
		}
	}

	mux.Handle(path, mcpHandler)

	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	// Handle graceful shutdown
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	// Call OnReady callback before blocking on Serve
	if opts.OnReady != nil {
		opts.OnReady(result)
	}

	err = server.Serve(listener)
	if err == http.ErrServerClosed {
		return result, nil
	}
	return result, err
}

// createNgrokListener creates an ngrok tunnel listener.
func (r *Runtime) createNgrokListener(ctx context.Context, opts *HTTPServerOptions) (net.Listener, error) {
	authtoken := opts.Ngrok.Authtoken
	if authtoken == "" {
		authtoken = os.Getenv("NGROK_AUTHTOKEN")
	}
	if authtoken == "" {
		return nil, fmt.Errorf("ngrok authtoken is required: set Authtoken or NGROK_AUTHTOKEN environment variable")
	}

	ngrokOpts := []ngrok.ConnectOption{
		ngrok.WithAuthtoken(authtoken),
	}

	// Configure the HTTP endpoint
	httpConfig := config.HTTPEndpoint()
	if opts.Ngrok.Domain != "" {
		httpConfig = config.HTTPEndpoint(config.WithDomain(opts.Ngrok.Domain))
	}

	listener, err := ngrok.Listen(ctx, httpConfig, ngrokOpts...)
	if err != nil {
		return nil, err
	}

	return listener, nil
}
