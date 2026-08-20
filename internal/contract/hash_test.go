package contract_test

import (
	"strings"
	"testing"

	"github.com/haribo/ozalid/internal/contract"
)

func TestHashReaderIsStableAcrossCalls(t *testing.T) {
	first, n, err := contract.HashReader(strings.NewReader("ozalid"))
	if err != nil {
		t.Fatalf("hashing: %v", err)
	}
	if n != 6 {
		t.Errorf("size = %d, want 6", n)
	}

	second, _, err := contract.HashReader(strings.NewReader("ozalid"))
	if err != nil {
		t.Fatalf("hashing again: %v", err)
	}
	// The whole dedup story rests on this: identical bytes, identical address.
	if first != second {
		t.Errorf("the same bytes produced %q then %q", first, second)
	}
	if !strings.HasPrefix(first, "sha256:") {
		t.Errorf("address = %q, want a sha256 prefix", first)
	}
}

func TestDifferentBytesGetDifferentAddresses(t *testing.T) {
	a, _, _ := contract.HashReader(strings.NewReader("ozalid"))
	b, _, _ := contract.HashReader(strings.NewReader("ozalid "))
	if a == b {
		t.Error("two different inputs share an address")
	}
}

func TestValidHashRefusesAnythingThatCouldEscapeTheStore(t *testing.T) {
	valid, _, _ := contract.HashReader(strings.NewReader("ozalid"))

	cases := map[string]string{
		"well formed":      valid,
		"empty":            "",
		"no algorithm":     strings.TrimPrefix(valid, "sha256:"),
		"wrong algorithm":  strings.Replace(valid, "sha256", "md5", 1),
		"too short":        "sha256:abc",
		"uppercase":        strings.ToUpper(valid),
		"path traversal":   "sha256:../../etc/passwd",
		"separator inside": "sha256:aa/bb",
		"null byte":        "sha256:" + strings.Repeat("a", 63) + "\x00",
		"trailing newline": valid + "\n",
	}

	for name, in := range cases {
		want := name == "well formed"
		if got := contract.ValidHash(in); got != want {
			t.Errorf("%s: ValidHash(%q) = %v, want %v", name, in, got, want)
		}
	}
}

func TestHashPathShardsOnTheFirstTwoBytes(t *testing.T) {
	hash, _, _ := contract.HashReader(strings.NewReader("ozalid"))
	path, err := contract.HashPath(hash)
	if err != nil {
		t.Fatalf("building the path: %v", err)
	}

	hex := strings.TrimPrefix(hash, "sha256:")
	want := hex[0:2] + "/" + hex[2:4] + "/" + hex
	if path != want {
		t.Errorf("path = %q, want %q", path, want)
	}

	if _, err := contract.HashPath("sha256:../escape"); err == nil {
		t.Error("HashPath accepted an address that is not a hash")
	}
}
