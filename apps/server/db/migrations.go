// Package db carries the schema's migrations, so a binary can apply them
// without a checkout beside it.
package db

import "embed"

// Migrations are the files goose reads.
//
// Embedded rather than read from disk: a container has no repository in it, and
// a server that cannot migrate itself needs a second thing shipped alongside
// and kept in step.
//
//go:embed migrations/*.sql
var Migrations embed.FS
