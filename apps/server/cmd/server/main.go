// Command server runs the ozalid API.
//
// This file is the single point of assembly: dependencies are constructed here
// and passed explicitly, so a missing one is a compile error (backend ADR 0001).
// It is also the only place that reads the environment.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/haribo/ozalid/apps/server/internal/adapters/blobstore"
	"github.com/haribo/ozalid/apps/server/internal/adapters/mail"
	"github.com/haribo/ozalid/apps/server/internal/adapters/postgres"
	"github.com/haribo/ozalid/apps/server/internal/app/account"
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
	// baseURL is the address this instance is reached at. The sign-in link in
	// a message has to point back here, and the server cannot work it out from
	// a request — it sits behind whatever terminates TLS.
	baseURL string
	// trustProxy says whether X-Forwarded-For may be believed. Off unless the
	// deployment says a proxy is in front: trusting it without one lets anybody
	// claim any source and walk past the sign-in rate limit.
	trustProxy bool
	mail       mail.Config
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
		baseURL:  env("OZALID_BASE_URL", "http://localhost:8090"),
		// Off by default. On, the caller's address is read from
		// X-Forwarded-For, which is only true when something in front sets it.
		trustProxy: env("OZALID_TRUSTED_PROXY", "") != "",
		mail: mail.Config{
			Host:     env("OZALID_SMTP_HOST", ""),
			Port:     env("OZALID_SMTP_PORT", "587"),
			Username: env("OZALID_SMTP_USERNAME", ""),
			Password: env("OZALID_SMTP_PASSWORD", ""),
			From:     env("OZALID_SMTP_FROM", ""),
		},
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

	// Signing in is the only way a person gets in, and it sends a message. An
	// instance that cannot send one is an instance nobody can enter, so it
	// fails here rather than at the first person who tries.
	if !cfg.mail.Complete() {
		return fmt.Errorf(
			"sign-in sends a link, so OZALID_SMTP_HOST and OZALID_SMTP_FROM are required")
	}

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
		Account:      account.New(store, store),
		Catalogue:    catalogue.New(store),
		Tokens:       store,
		SignIn:       store,
		Mail:         mail.NewSMTP(cfg.mail, cfg.baseURL),
		Standings:    store,
		TrustProxy:   cfg.trustProxy,
		Now:          time.Now,
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

	slog.Info("listening", "addr", cfg.addr, "blobs", cfg.blobRoot, "version", version,
		"trusting a proxy for the caller's address", cfg.trustProxy)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	// Sign-in links are sent off the request's path, so stopping without
	// waiting would drop one somebody is watching their inbox for.
	api.WaitForSending()
	return nil
}
