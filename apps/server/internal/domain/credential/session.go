package credential

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"
)

// LinkLifetime is how long a sign-in link is good for.
//
// Short enough that a mail left open in a shared inbox stops being a way in,
// long enough that someone reading their mail on another device still makes it.
const LinkLifetime = 15 * time.Minute

// SessionLifetime is how long a browser stays signed in without acting.
//
// A reviewer's sitting is measured in minutes and their week in days; this is
// long enough to come back to a case tomorrow, short enough that a forgotten
// laptop stops being one.
const SessionLifetime = 14 * 24 * time.Hour

// Secret returns a random value and the hash to store for it.
//
// The same shape as a service token, without the prefix: nobody reads a session
// cookie or a link out of a configuration file, so there is nothing to make
// recognisable.
func Secret() (value, hash string, err error) {
	raw := make([]byte, bytes)
	if _, err := rand.Read(raw); err != nil {
		return "", "", fmt.Errorf("drawing a secret: %w", err)
	}
	value = base64.RawURLEncoding.EncodeToString(raw)
	return value, Hash(value), nil
}
