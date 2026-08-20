package blobstore_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/haribo/ozalid/apps/server/internal/adapters/blobstore"
	"github.com/haribo/ozalid/internal/contract"
)

func newStore(t *testing.T) (*blobstore.FileStore, string) {
	t.Helper()
	root := t.TempDir()
	s, err := blobstore.NewFileStore(root)
	if err != nil {
		t.Fatalf("creating the store: %v", err)
	}
	return s, root
}

func countFiles(t *testing.T, root string) int {
	t.Helper()
	n := 0
	err := filepath.WalkDir(root, func(_ string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			n++
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking the store: %v", err)
	}
	return n
}

func TestTheSameBytesAreStoredOnce(t *testing.T) {
	ctx := context.Background()
	store, root := newStore(t)

	content := []byte("a screenshot's worth of bytes")
	hash, _, _ := contract.HashReader(bytes.NewReader(content))

	if err := store.Put(ctx, hash, bytes.NewReader(content)); err != nil {
		t.Fatalf("first Put: %v", err)
	}
	if err := store.Put(ctx, hash, bytes.NewReader(content)); err != nil {
		t.Fatalf("second Put: %v", err)
	}

	// This is what makes full visual history affordable (ADR 0004).
	if got := countFiles(t, root); got != 1 {
		t.Errorf("the store holds %d files, want 1 for identical content", got)
	}

	held, err := store.Exists(ctx, hash)
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if !held {
		t.Error("Exists reported the content as absent after storing it")
	}
}

func TestContentThatDoesNotMatchItsAddressIsRefusedAndNothingIsWritten(t *testing.T) {
	ctx := context.Background()
	store, root := newStore(t)

	announced, _, _ := contract.HashReader(strings.NewReader("what was promised"))

	err := store.Put(ctx, announced, strings.NewReader("what was actually sent"))
	if !errors.Is(err, blobstore.ErrHashMismatch) {
		t.Fatalf("Put returned %v, want ErrHashMismatch", err)
	}

	// A blob sitting at an address that does not describe it would give every
	// future reader the wrong bytes with no way to notice.
	if got := countFiles(t, root); got != 0 {
		t.Errorf("the store holds %d files after a refused write, want 0", got)
	}
	held, _ := store.Exists(ctx, announced)
	if held {
		t.Error("Exists reports content at an address that was refused")
	}
}

func TestAnAddressThatIsNotAHashNeverReachesTheFilesystem(t *testing.T) {
	ctx := context.Background()
	store, root := newStore(t)

	for _, bad := range []string{
		"sha256:../../etc/passwd",
		"../../../etc/passwd",
		"sha256:aa/bb",
		"",
		"sha256:" + strings.Repeat("z", 64),
	} {
		if err := store.Put(ctx, bad, strings.NewReader("payload")); err == nil {
			t.Errorf("Put(%q) was accepted, want a refusal", bad)
		}
		if _, err := store.Get(ctx, bad); err == nil {
			t.Errorf("Get(%q) was accepted, want a refusal", bad)
		}
		if _, err := store.Exists(ctx, bad); err == nil {
			t.Errorf("Exists(%q) was accepted, want a refusal", bad)
		}
	}

	if got := countFiles(t, root); got != 0 {
		t.Errorf("the store holds %d files, want 0", got)
	}
}

func TestGetReturnsTheBytesThatWereStored(t *testing.T) {
	ctx := context.Background()
	store, _ := newStore(t)

	content := []byte("the evidence itself")
	hash, _, _ := contract.HashReader(bytes.NewReader(content))
	if err := store.Put(ctx, hash, bytes.NewReader(content)); err != nil {
		t.Fatalf("Put: %v", err)
	}

	rc, err := store.Get(ctx, hash)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	t.Cleanup(func() {
		if err := rc.Close(); err != nil {
			t.Errorf("closing the reader: %v", err)
		}
	})

	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("reading back: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("read back %q, want %q", got, content)
	}
}

func TestGetReportsAMissingAddressDistinctly(t *testing.T) {
	ctx := context.Background()
	store, _ := newStore(t)

	absent, _, _ := contract.HashReader(strings.NewReader("never stored"))
	_, err := store.Get(ctx, absent)
	if !errors.Is(err, blobstore.ErrNotFound) {
		t.Errorf("Get returned %v, want ErrNotFound", err)
	}
}

func TestAFailedWriteLeavesNoBlobClaimingAValidAddress(t *testing.T) {
	ctx := context.Background()
	store, root := newStore(t)

	content := []byte("half of this will arrive")
	hash, _, _ := contract.HashReader(bytes.NewReader(content))

	// A reader that dies partway is what a dropped connection looks like.
	truncated := io.MultiReader(
		bytes.NewReader(content[:8]),
		iotest{err: errors.New("connection reset")},
	)
	if err := store.Put(ctx, hash, truncated); err == nil {
		t.Fatal("Put accepted a truncated upload")
	}

	held, _ := store.Exists(ctx, hash)
	if held {
		t.Error("a truncated upload left content at the announced address")
	}
	// Only the temporary file may remain, and it is inert.
	for _, name := range listNames(t, root) {
		if !strings.HasPrefix(name, ".incoming-") {
			t.Errorf("unexpected file %q left behind", name)
		}
	}
}

// iotest is a reader that always fails.
type iotest struct{ err error }

func (r iotest) Read([]byte) (int, error) { return 0, r.err }

func listNames(t *testing.T, root string) []string {
	t.Helper()
	var names []string
	err := filepath.WalkDir(root, func(_ string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			names = append(names, d.Name())
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking the store: %v", err)
	}
	return names
}
