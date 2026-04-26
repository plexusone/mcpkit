// Copyright 2025 John Wang. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package oauth2

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
)

// PKCE errors.
var (
	ErrPKCERequired           = errors.New("PKCE code_challenge is required")
	ErrPKCEMethodNotSupported = errors.New("only S256 code_challenge_method is supported")
	ErrPKCEVerificationFailed = errors.New("PKCE code_verifier verification failed")
	ErrPKCEVerifierTooShort   = errors.New("code_verifier must be at least 43 characters")
	ErrPKCEVerifierTooLong    = errors.New("code_verifier must be at most 128 characters")
	ErrPKCEVerifierInvalid    = errors.New("code_verifier contains invalid characters")
)

const (
	// PKCEMethodS256 is the only supported PKCE method (SHA-256).
	PKCEMethodS256 = "S256"

	// PKCEMethodPlain is not supported for security reasons.
	PKCEMethodPlain = "plain"

	// MinVerifierLength is the minimum length for a code verifier (RFC 7636).
	MinVerifierLength = 43

	// MaxVerifierLength is the maximum length for a code verifier (RFC 7636).
	MaxVerifierLength = 128
)

// GenerateCodeVerifier generates a cryptographically secure code verifier
// suitable for PKCE. The verifier is 43-128 characters from the unreserved
// character set [A-Z] / [a-z] / [0-9] / "-" / "." / "_" / "~".
func GenerateCodeVerifier() (string, error) {
	// Generate 32 bytes of random data, which will become 43 base64url characters
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generating random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// GenerateCodeChallenge generates a code challenge from a code verifier
// using the S256 method (SHA-256 hash, base64url encoded).
func GenerateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// ValidateCodeVerifier validates that a code verifier meets RFC 7636 requirements.
func ValidateCodeVerifier(verifier string) error {
	if len(verifier) < MinVerifierLength {
		return ErrPKCEVerifierTooShort
	}
	if len(verifier) > MaxVerifierLength {
		return ErrPKCEVerifierTooLong
	}

	// Check that all characters are in the unreserved set
	for _, c := range verifier {
		if !isUnreservedChar(c) {
			return ErrPKCEVerifierInvalid
		}
	}
	return nil
}

// VerifyCodeChallenge verifies that a code verifier matches a code challenge.
// Only supports the S256 method.
func VerifyCodeChallenge(verifier, challenge, method string) error {
	if method == "" {
		method = PKCEMethodS256
	}

	if method != PKCEMethodS256 {
		return ErrPKCEMethodNotSupported
	}

	if err := ValidateCodeVerifier(verifier); err != nil {
		return err
	}

	// Compute the challenge from the verifier
	computed := GenerateCodeChallenge(verifier)

	// Use constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(computed), []byte(challenge)) != 1 {
		return ErrPKCEVerificationFailed
	}

	return nil
}

// isUnreservedChar checks if a rune is in the PKCE unreserved character set.
// Allowed characters: [A-Z] / [a-z] / [0-9] / "-" / "." / "_" / "~"
func isUnreservedChar(c rune) bool {
	return (c >= 'A' && c <= 'Z') ||
		(c >= 'a' && c <= 'z') ||
		(c >= '0' && c <= '9') ||
		c == '-' || c == '.' || c == '_' || c == '~'
}

// GenerateSecureToken generates a cryptographically secure random token
// encoded as base64url without padding.
func GenerateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generating random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// GenerateClientID generates a unique client ID.
func GenerateClientID() (string, error) {
	return GenerateSecureToken(16)
}

// GenerateClientSecret generates a secure client secret.
func GenerateClientSecret() (string, error) {
	return GenerateSecureToken(32)
}

// GenerateAuthorizationCode generates a secure authorization code.
func GenerateAuthorizationCode() (string, error) {
	return GenerateSecureToken(32)
}

// GenerateAccessToken generates a secure access token.
func GenerateAccessToken() (string, error) {
	return GenerateSecureToken(32)
}

// GenerateRefreshToken generates a secure refresh token.
func GenerateRefreshToken() (string, error) {
	return GenerateSecureToken(32)
}
