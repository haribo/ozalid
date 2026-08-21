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
	"github.com/haribo/ozalid/apps/server/internal/app/evidence"
	"github.com/haribo/ozalid/apps/server/internal/app/intake"
	ozhttp "github.com/haribo/ozalid/apps/server/internal/ports/http"
)

// version is stamped at build time with -ldflags.
var version = "dev"

type config struct {
	addr     string
	blobRoot string
	dsn      string
}

func load() config {
	return config{
		addr:     env("OZALID_ADDR", ":8080"),
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
		Intake:       intake.New(store),
		Evidence:     evidence.New(store),
	})

	srv := &http.Server{Addr: cfg.addr, Handler: api.Handler()}

	slog.Info("listening", "addr", cfg.addr, "blobs", cfg.blobRoot)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}
