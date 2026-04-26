// Copyright 2025 John Wang. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package oauth2

import "context"

// contextKey is a type for context keys to avoid collisions.
type contextKey string

const tokenInfoKey contextKey = "oauth2server:tokenInfo" //nolint:gosec // G101: this is a context key, not credentials

// SetTokenInfoContext stores TokenInfo in the context.
func SetTokenInfoContext(ctx context.Context, info *TokenInfo) context.Context {
	return context.WithValue(ctx, tokenInfoKey, info)
}

// GetTokenInfoContext retrieves TokenInfo from the context.
// Returns nil if not found.
func GetTokenInfoContext(ctx context.Context) *TokenInfo {
	info, _ := ctx.Value(tokenInfoKey).(*TokenInfo)
	return info
}

// GetSubjectFromContext returns the authenticated subject (username) from the context.
// Returns empty string if not found.
func GetSubjectFromContext(ctx context.Context) string {
	info := GetTokenInfoContext(ctx)
	if info == nil {
		return ""
	}
	return info.Subject
}

// GetClientIDFromContext returns the client ID from the context.
// Returns empty string if not found.
func GetClientIDFromContext(ctx context.Context) string {
	info := GetTokenInfoContext(ctx)
	if info == nil {
		return ""
	}
	return info.ClientID
}

// GetScopeFromContext returns the scope from the context.
// Returns empty string if not found.
func GetScopeFromContext(ctx context.Context) string {
	info := GetTokenInfoContext(ctx)
	if info == nil {
		return ""
	}
	return info.Scope
}
