// Package credential mints and reads the tokens a service account presents.
//
// Nothing here touches a database or a request: minting is randomness plus
// encoding, and reading is string handling. Both are worth keeping pure so the
// one thing that must never drift — that a stored hash matches the token that
// produced it — can be tested without a server.
package credential

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
)

// Prefix marks a token as ozalid's.
//
// The server does not need it. A person finding the string in a configuration
// file does, and so do the scanners that comb repositories for leaked secrets —
// which is the point: a secret that leaks should be recognisable at a glance.
const Prefix = "ozp_"

// bytes of randomness behind a token. Thirty-two is well past what a guess
// could reach and short enough to paste on one line.
const bytes = 32

// Mint returns a new token and the hash to store for it.
//
// The token is returned once and never again: only its hash is kept, so the
// server can check a token it is shown and cannot show one back.
func Mint() (token, hash string, err error) {
	raw := make([]byte, bytes)
	if _, err := rand.Read(raw); err != nil {
		return "", "", fmt.Errorf("drawing a token: %w", err)
	}
	token = Prefix + base64.RawURLEncoding.EncodeToString(raw)
	return token, Hash(token), nil
}

// Hash returns what is stored for a token.
func Hash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// Matches reports whether a presented token produces a stored hash.
//
// The comparison is constant-time. The difference is unmeasurable on one call
// and observable across many, and a caller who can measure it can learn a hash
// character by character.
func Matches(presented, storedHash string) bool {
	return subtle.ConstantTimeCompare([]byte(Hash(presented)), []byte(storedHash)) == 1
}

// FromHeader pulls a token out of an Authorization header.
//
// Anything that is not `Bearer` followed by an ozalid token is no token at all.
// Being strict here means a malformed header is refused as unauthenticated
// rather than looked up as if it might be somebody.
func FromHeader(header string) (string, bool) {
	const scheme = "Bearer "
	if !strings.HasPrefix(header, scheme) {
		return "", false
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, scheme))
	if !strings.HasPrefix(token, Prefix) || len(token) <= len(Prefix) {
		return "", false
	}
	return token, true
}
