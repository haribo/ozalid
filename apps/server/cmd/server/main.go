// Command server runs the ozalid API.
//
// This file is the single point of assembly: dependencies are constructed here
// and passed explicitly, so a missing one is a compile error (backend ADR 0001).
// It is also the only place that reads the environment.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/haribo/ozalid/apps/server/internal/adapters/blobstore"
	"github.com/haribo/ozalid/apps/server/internal/adapters/postgres"
	"github.com/haribo/ozalid/apps/server/internal/app/catalogue"
	"github.com/haribo/ozalid/apps/server/internal/app/comment"
	"github.com/haribo/ozalid/apps/server/internal/app/evidence"
	"github.com/haribo/ozalid/apps/server/internal/app/intake"
	"github.com/haribo/ozalid/apps/server/internal/app/session"
	ozhttp "github.com/haribo/ozalid/apps/server/internal/ports/http"
	"github.com/haribo/ozalid/apps/server/internal/ports/http/webui"
)

// version is stamped at build time with -ldflags.
var version = "dev"

type config struct {
	addr     string
	blobRoot string
	dsn      string
}

// defaultAddr is off :8080 on purpose. That port is the first thing anything
// else on a developer machine grabs, and a collision here does not fail
// loudly — the server exits and every manual check quietly hits whatever else
// is listening.
const defaultAddr = ":8090"

func load() config {
	return config{
		addr:     env("OZALID_ADDR", defaultAddr),
		blobRoot: env("OZALID_BLOB_ROOT", "var/blobs"),
		dsn:      env("OZALID_DSN", "postgres://ozalid:ozalid@localhost:5442/ozalid?sslmode=disable"),
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	// One subcommand, and it is not a client: it seeds the instance so clients
	// can begin (see bootstrap.go).
	if len(os.Args) > 1 && os.Args[1] == "bootstrap" {
		if err := bootstrap(context.Background(), os.Args[2:]); err != nil {
			slog.Error("bootstrap failed", "error", err)
			os.Exit(1)
		}
		return
	}

	if err := run(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()
	cfg := load()

	blobs, err := blobstore.NewFileStore(cfg.blobRoot)
	if err != nil {
		return err
	}

	// Before anything else opens: a schema that is behind must fail the deploy,
	// not the first request that touches a missing column.
	if err := migrate(ctx, cfg.dsn); err != nil {
		return err
	}

	store, err := postgres.Open(ctx, cfg.dsn)
	if err != nil {
		return err
	}
	defer store.Close()

	// The assembly itself: the adapter satisfies the port the use cases
	// declared, and nothing above ever sees a generated row or a driver error.
	api := ozhttp.New(ozhttp.Deps{
		Version:      version,
		Blobs:        blobs,
		BlobRecorder: store,
		Catalogue:    catalogue.New(store),
		Tokens:       store,
		Standings:    store,
		Intake:       intake.New(store, blobs),
		Evidence:     evidence.New(store),
		Session:      session.New(store),
		Comment:      comment.New(store),
	})

	srv := &http.Server{Addr: cfg.addr, Handler: api.Handler()}

	if !webui.Built() {
		// Said at startup rather than discovered as a blank page.
		slog.Warn("the web client was not built into this binary; only the API is served")
	}

	slog.Info("listening", "addr", cfg.addr, "blobs", cfg.blobRoot, "version", version)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}
