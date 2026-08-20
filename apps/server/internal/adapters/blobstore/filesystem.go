// Package blobstore stores capture bytes under the hash of their content
// (ADR 0004, backend ADR 0004).
package blobstore

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/haribo/ozalid/internal/contract"
)

// ErrNotFound is returned when the store holds nothing at that address.
var ErrNotFound = errors.New("blobstore: no content at that address")

// ErrHashMismatch is returned when the bytes written do not produce the
// address they were announced under. Accepting them would poison the store:
// every future reader would get content that is not what its address says.
var ErrHashMismatch = errors.New("blobstore: the content does not match its address")

// Store is the outbound port for capture bytes. The v1 adapter writes to a
// local directory; an S3 adapter is a second implementation of this interface
// (backend ADR 0004).
type Store interface {
	Put(ctx context.Context, hash string, r io.Reader) error
	Get(ctx context.Context, hash string) (io.ReadCloser, error)
	Exists(ctx context.Context, hash string) (bool, error)
}

// FileStore keeps blobs in a directory tree rooted at Root.
//
// Nothing is ever modified in place: an address always denotes the same bytes,
// so a write is either a no-op or a create.
type FileStore struct {
	root string
}

// NewFileStore returns a store rooted at dir, creating it if needed.
func NewFileStore(dir string) (*FileStore, error) {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("creating the blob root: %w", err)
	}
	return &FileStore{root: dir}, nil
}

// path resolves an address to its absolute location, refusing anything that is
// not a well-formed hash before it can reach the filesystem.
func (s *FileStore) path(hash string) (string, error) {
	rel, err := contract.HashPath(hash)
	if err != nil {
		return "", err
	}
	return filepath.Join(s.root, filepath.FromSlash(rel)), nil
}

// Exists reports whether the store already holds that content. Intake uses it
// to skip bytes it does not need to receive again.
func (s *FileStore) Exists(_ context.Context, hash string) (bool, error) {
	p, err := s.path(hash)
	if err != nil {
		return false, err
	}
	switch _, err := os.Stat(p); {
	case err == nil:
		return true, nil
	case errors.Is(err, os.ErrNotExist):
		return false, nil
	default:
		return false, fmt.Errorf("checking the store: %w", err)
	}
}

// Put stores r under hash.
//
// The content is written to a temporary file and hashed as it goes; the file is
// only renamed into place once the bytes prove they produce the announced
// address. A crash before that leaves a temporary file, never a blob claiming
// an address it does not have.
func (s *FileStore) Put(ctx context.Context, hash string, r io.Reader) error {
	final, err := s.path(hash)
	if err != nil {
		return err
	}

	held, err := s.Exists(ctx, hash)
	if err != nil {
		return err
	}
	if held {
		// Same address, same bytes. Rewriting would be work for nothing.
		return nil
	}

	dir := filepath.Dir(final)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("creating the shard: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".incoming-*")
	if err != nil {
		return fmt.Errorf("opening a temporary file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() {
		// best-effort: removing a temporary file that was already renamed is
		// expected to fail, and a leftover is inert.
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}()

	got, _, err := contract.HashReader(io.TeeReader(r, tmp))
	if err != nil {
		return fmt.Errorf("writing the content: %w", err)
	}
	if got != hash {
		return fmt.Errorf("%w: announced %s, got %s", ErrHashMismatch, hash, got)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("flushing the content: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing the temporary file: %w", err)
	}
	if err := os.Rename(tmpName, final); err != nil {
		return fmt.Errorf("moving the content into place: %w", err)
	}
	return nil
}

// Get opens the content at that address.
func (s *FileStore) Get(_ context.Context, hash string) (io.ReadCloser, error) {
	p, err := s.path(hash)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(p)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("opening the content: %w", err)
	}
	return f, nil
}

// Compile-time proof that the filesystem adapter satisfies the port.
var _ Store = (*FileStore)(nil)
