package credential_test

import (
	"strings"
	"testing"

	"github.com/haribo/ozalid/apps/server/internal/domain/credential"
)

func TestAMintedTokenMatchesTheHashKeptForIt(t *testing.T) {
	// The one thing that must never drift: the server keeps a hash, is shown a
	// token, and the two have to agree.
	token, hash, err := credential.Mint()
	if err != nil {
		t.Fatalf("minting: %v", err)
	}
	if !credential.Matches(token, hash) {
		t.Error("a token did not match the hash minted with it")
	}
	if credential.Matches(token+"x", hash) {
		t.Error("a token that was tampered with still matched")
	}
}

func TestTwoTokensAreNeverTheSame(t *testing.T) {
	seen := make(map[string]struct{}, 500)
	for i := 0; i < 500; i++ {
		token, _, err := credential.Mint()
		if err != nil {
			t.Fatalf("minting: %v", err)
		}
		if _, again := seen[token]; again {
			t.Fatal("two mints produced the same token")
		}
		seen[token] = struct{}{}
	}
}

func TestTheStoredHashRevealsNothingOfTheToken(t *testing.T) {
	// Whoever reads a stolen dump of the table must get nothing usable.
	token, hash, err := credential.Mint()
	if err != nil {
		t.Fatalf("minting: %v", err)
	}
	if strings.Contains(hash, strings.TrimPrefix(token, credential.Prefix)) {
		t.Error("the stored hash contains the token")
	}
	if strings.HasPrefix(hash, credential.Prefix) {
		t.Error("the stored hash is shaped like a token; it must not be mistakable for one")
	}
}

func TestATokenIsRecognisableAtAGlance(t *testing.T) {
	// The prefix is for people and for the scanners that comb repositories, not
	// for the server.
	token, _, err := credential.Mint()
	if err != nil {
		t.Fatalf("minting: %v", err)
	}
	if !strings.HasPrefix(token, credential.Prefix) {
		t.Errorf("token = %q, want the %q prefix", token, credential.Prefix)
	}
}

func TestOnlyAWellFormedBearerHeaderCarriesAToken(t *testing.T) {
	// Anything malformed is refused as unauthenticated rather than looked up as
	// if it might be somebody.
	token, _, err := credential.Mint()
	if err != nil {
		t.Fatalf("minting: %v", err)
	}

	cases := []struct {
		name   string
		header string
		want   bool
	}{
		{"a bearer token", "Bearer " + token, true},
		{"nothing at all", "", false},
		{"a token with no scheme", token, false},
		{"another scheme", "Basic " + token, false},
		{"the scheme alone", "Bearer ", false},
		{"the prefix alone", "Bearer " + credential.Prefix, false},
		{"something else entirely", "Bearer ghp_0123456789", false},
		// Case matters: `bearer` is not the scheme this server declares, and
		// accepting variants invites the next one.
		{"a lowercase scheme", "bearer " + token, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := credential.FromHeader(c.header)
			if ok != c.want {
				t.Errorf("FromHeader(%q) ok = %v, want %v", c.header, ok, c.want)
			}
			if ok && got != token {
				t.Errorf("FromHeader(%q) = %q, want the token", c.header, got)
			}
		})
	}
}
