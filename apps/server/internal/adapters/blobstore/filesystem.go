// Package blobstore stores capture bytes under the hash of their content
// (ADR 0004, backend ADR 0004).
package blobstore

import (
	"context"
	"io"
)

// Store is the outbound port for capture bytes. The v1 adapter writes to a
// local directory; an S3 adapter is a second implementation of this interface.
type Store interface {
	Put(ctx context.Context, hash string, r io.Reader) error
	Get(ctx context.Context, hash string) (io.ReadCloser, error)
	Exists(ctx context.Context, hash string) (bool, error)
}
