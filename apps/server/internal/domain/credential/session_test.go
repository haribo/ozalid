package credential_test

import (
	"testing"
	"time"

	"github.com/haribo/ozalid/apps/server/internal/domain/credential"
)

func TestASecretMatchesTheHashKeptForIt(t *testing.T) {
	value, hash, err := credential.Secret()
	if err != nil {
		t.Fatalf("drawing: %v", err)
	}
	if !credential.Matches(value, hash) {
		t.Error("a secret did not match its own hash")
	}
	if credential.Matches(value+"x", hash) {
		t.Error("a tampered secret still matched")
	}
}

func TestASecretIsNotShapedLikeAServiceToken(t *testing.T) {
	// The prefix exists so a person finding a token in a configuration file
	// knows what it is. Nobody reads a session cookie out of a file, and giving
	// it the same shape would make the two mistakable for each other.
	value, _, err := credential.Secret()
	if err != nil {
		t.Fatalf("drawing: %v", err)
	}
	if _, ok := credential.FromHeader("Bearer " + value); ok {
		t.Error("a session secret was accepted as a service token")
	}
}

func TestALinkOutlivesAnEmailButNotADay(t *testing.T) {
	// Long enough to read your mail on another device, short enough that an old
	// message stops being a way in.
	if credential.LinkLifetime < 5*time.Minute || credential.LinkLifetime > time.Hour {
		t.Errorf("LinkLifetime = %v, want minutes rather than seconds or days", credential.LinkLifetime)
	}
}
