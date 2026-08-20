// Command server runs the ozalid API.
//
// This file is the single point of assembly: dependencies are constructed here
// and passed explicitly, so a missing one is a compile error (backend ADR 0001).
// It is also the only place that reads the environment.
package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/haribo/ozalid/apps/server/internal/adapters/blobstore"
	ozhttp "github.com/haribo/ozalid/apps/server/internal/ports/http"
)

// version is stamped at build time with -ldflags.
var version = "dev"

type config struct {
	addr     string
	blobRoot string
}

func load() config {
	return config{
		addr:     env("OZALID_ADDR", ":8080"),
		blobRoot: env("OZALID_BLOB_ROOT", "var/blobs"),
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	cfg := load()

	blobs, err := blobstore.NewFileStore(cfg.blobRoot)
	if err != nil {
		slog.Error("cannot open the capture store", "root", cfg.blobRoot, "error", err)
		os.Exit(1)
	}

	srv := &http.Server{
		Addr:    cfg.addr,
		Handler: ozhttp.New(version, blobs).Handler(),
	}

	slog.Info("listening", "addr", cfg.addr, "blobs", cfg.blobRoot)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
