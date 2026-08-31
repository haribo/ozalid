package contract

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// HashAlgorithm is the only algorithm ozalid addresses content with.
//
// Changing it orphans every blob already stored, so it changes through a new
// ADR and never otherwise (ADR 0004).
const HashAlgorithm = "sha256"

// hashPattern is what a well-formed address looks like. It is deliberately
// strict: a hash reaches the filesystem as a path, so anything that is not
// exactly this must be refused before it gets there.
var hashPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

// HashReader reads r to completion and returns the address of its bytes.
//
// A client computes the same address before uploading, from the algorithm the
// OpenAPI document publishes. Getting it wrong means re-uploading the whole
// catalogue on every run, which is why the algorithm changes only through a new
// ADR (ADR 0004).
func HashReader(r io.Reader) (string, int64, error) {
	h := sha256.New()
	n, err := io.Copy(h, r)
	if err != nil {
		return "", 0, fmt.Errorf("reading the content: %w", err)
	}
	return HashAlgorithm + ":" + hex.EncodeToString(h.Sum(nil)), n, nil
}

// ValidHash reports whether s is a well-formed address.
func ValidHash(s string) bool { return hashPattern.MatchString(s) }

// HashPath turns an address into the relative path its bytes live at.
//
// The first two bytes become directory levels: a single flat directory holding
// a hundred thousand files is slow to list and unpleasant to back up.
func HashPath(hash string) (string, error) {
	if !ValidHash(hash) {
		return "", fmt.Errorf("%q is not a valid content address", hash)
	}
	hex := strings.TrimPrefix(hash, HashAlgorithm+":")
	return hex[0:2] + "/" + hex[2:4] + "/" + hex, nil
}
