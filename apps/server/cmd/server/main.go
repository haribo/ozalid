// Command server runs the ozalid API.
//
// This file is the single point of assembly: dependencies are constructed here
// and passed explicitly, so a missing one is a compile error (backend ADR 0001).
package main

import (
	"log/slog"
	"net/http"
	"os"

	ozhttp "github.com/haribo/ozalid/apps/server/internal/ports/http"
)

func main() {
	addr := os.Getenv("OZALID_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	srv := &http.Server{Addr: addr, Handler: ozhttp.NewMux()}

	slog.Info("listening", "addr", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
