# OAuth 2.1 Authentication

The `mcp/oauth2` package provides an OAuth 2.1 Authorization Server for authenticated MCP servers. This is required for public MCP servers that integrate with services like ChatGPT.com.

## Overview

The OAuth2 implementation supports:

- Authorization Code Flow with PKCE (RFC 7636)
- Dynamic Client Registration (RFC 7591)
- Authorization Server Metadata (RFC 8414)
- Bearer token authentication

## Quick Start

Enable OAuth2 when serving over HTTP:

```go
result, err := rt.ServeHTTP(ctx, &server.HTTPServerOptions{
    Addr: ":8080",
    OAuth2: &server.OAuth2Options{
        Users: map[string]string{
            "admin": "password",
        },
    },
    OnReady: func(r *server.HTTPServerResult) {
        fmt.Printf("Client ID: %s\n", r.OAuth2.ClientID)
        fmt.Printf("Client Secret: %s\n", r.OAuth2.ClientSecret)
    },
})
```

## Endpoints

When OAuth2 is enabled, these endpoints are automatically configured:

| Endpoint | Purpose |
|----------|---------|
| `/.well-known/oauth-authorization-server` | Server metadata (RFC 8414) |
| `/authorize` | Authorization endpoint |
| `/token` | Token endpoint |
| `/register` | Dynamic client registration |

## Configuration

### Basic Configuration

```go
OAuth2: &server.OAuth2Options{
    // Username/password pairs for authentication
    Users: map[string]string{
        "user1": "password1",
        "user2": "password2",
    },
}
```

### With Pre-configured Clients

```go
OAuth2: &server.OAuth2Options{
    Users: map[string]string{"admin": "password"},

    // Pre-configure a client
    ClientID:     "my-client-id",
    ClientSecret: "my-client-secret",
}
```

### Custom Token Lifetime

```go
OAuth2: &server.OAuth2Options{
    Users:           map[string]string{"admin": "password"},
    TokenExpiration: 24 * time.Hour,  // Default is 1 hour
}
```

## OAuth2 Flow

### 1. Client Registration

Clients register dynamically:

```http
POST /register
Content-Type: application/json

{
  "client_name": "My MCP Client",
  "redirect_uris": ["https://myapp.com/callback"]
}
```

Response:

```json
{
  "client_id": "generated-client-id",
  "client_secret": "generated-client-secret"
}
```

### 2. Authorization

Redirect user to authorize:

```
GET /authorize?
  response_type=code&
  client_id=CLIENT_ID&
  redirect_uri=https://myapp.com/callback&
  code_challenge=CHALLENGE&
  code_challenge_method=S256&
  state=STATE
```

User authenticates, then redirected to:

```
https://myapp.com/callback?code=AUTH_CODE&state=STATE
```

### 3. Token Exchange

Exchange code for token:

```http
POST /token
Content-Type: application/x-www-form-urlencoded

grant_type=authorization_code&
code=AUTH_CODE&
redirect_uri=https://myapp.com/callback&
client_id=CLIENT_ID&
client_secret=CLIENT_SECRET&
code_verifier=VERIFIER
```

Response:

```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "token_type": "Bearer",
  "expires_in": 3600
}
```

### 4. Authenticated Requests

Include token in MCP requests:

```http
GET /mcp
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
```

## Server Metadata

Clients can discover OAuth2 configuration:

```http
GET /.well-known/oauth-authorization-server
```

Response:

```json
{
  "issuer": "https://myserver.com",
  "authorization_endpoint": "https://myserver.com/authorize",
  "token_endpoint": "https://myserver.com/token",
  "registration_endpoint": "https://myserver.com/register",
  "response_types_supported": ["code"],
  "grant_types_supported": ["authorization_code"],
  "code_challenge_methods_supported": ["S256"]
}
```

## PKCE (Proof Key for Code Exchange)

PKCE is required for all authorization requests:

1. Generate code verifier (random string)
2. Create code challenge: `BASE64URL(SHA256(code_verifier))`
3. Send challenge in authorization request
4. Send verifier in token request

Example (Go):

```go
import (
    "crypto/rand"
    "crypto/sha256"
    "encoding/base64"
)

// Generate verifier
verifier := make([]byte, 32)
rand.Read(verifier)
codeVerifier := base64.RawURLEncoding.EncodeToString(verifier)

// Create challenge
hash := sha256.Sum256([]byte(codeVerifier))
codeChallenge := base64.RawURLEncoding.EncodeToString(hash[:])
```

## Custom Authentication

For custom authentication logic, use the OAuth2 package directly:

```go
import "github.com/plexusone/omniskill/mcp/oauth2"

authServer := oauth2.NewServer(&oauth2.Config{
    Issuer: "https://myserver.com",

    // Custom user validator
    ValidateUser: func(username, password string) bool {
        // Check against database, LDAP, etc.
        return validateCredentials(username, password)
    },
})

// Use with HTTP server
http.Handle("/authorize", authServer.AuthorizeHandler())
http.Handle("/token", authServer.TokenHandler())
```

## ChatGPT.com Integration

For ChatGPT.com, OAuth2 is required. Configure your server:

```go
rt.ServeHTTP(ctx, &server.HTTPServerOptions{
    Addr:          ":443",
    NgrokAuthtoken: os.Getenv("NGROK_AUTHTOKEN"),
    OAuth2: &server.OAuth2Options{
        Users: map[string]string{
            os.Getenv("OAUTH_USER"): os.Getenv("OAUTH_PASSWORD"),
        },
    },
    OnReady: func(r *server.HTTPServerResult) {
        fmt.Println("Configure in ChatGPT:")
        fmt.Printf("  MCP URL: %s\n", r.PublicURL)
        fmt.Printf("  Client ID: %s\n", r.OAuth2.ClientID)
        fmt.Printf("  Client Secret: %s\n", r.OAuth2.ClientSecret)
    },
})
```

## Security Notes

1. **HTTPS Required** - Always use HTTPS in production
2. **Strong Secrets** - Use cryptographically random client secrets
3. **Token Expiration** - Set appropriate token lifetime
4. **PKCE Required** - Never disable PKCE
5. **Validate Redirect URIs** - Strictly validate registered redirect URIs
