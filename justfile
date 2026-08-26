# ozalid — task runner. See docs/git-workflow.md for the branch and PR rules.

set shell := ["bash", "-uc"]

web := "apps/web"

# The web client vendors npm packages that happen to ship Go files, and `./...`
# would walk into them. Every Go recipe works from this list instead.
gopkgs := "$(go list ./... | grep -v /node_modules/)"

# List the available recipes.
default:
    @just --list --unsorted

# ---------------------------------------------------------------- environment

# Start the local stack (Postgres).
up:
    docker compose up -d

# Stop the local stack, keeping the data.
down:
    docker compose down

# Stop the local stack and drop its data.
reset:
    docker compose down -v

# Install the hooks git will not run on its own (see docs/git-workflow.md).
hooks:
    git config core.hooksPath .githooks
    @echo "hooks enabled from .githooks"

# ------------------------------------------------------------------- database

# Off the default port on purpose — a dev machine often already runs a Postgres.
pg_port := env_var_or_default("OZALID_PG_PORT", "5442")
dsn := "postgres://ozalid:ozalid@localhost:" + pg_port + "/ozalid?sslmode=disable"
goose := "GOOSE_DRIVER=postgres GOOSE_DBSTRING='" + dsn + "' GOOSE_MIGRATION_DIR=apps/server/db/migrations go tool goose"

# Apply every pending migration.
db-up:
    {{goose}} up

# Roll the last migration back.
db-down:
    {{goose}} down

# Show which migrations have been applied.
db-status:
    {{goose}} status

# Create a migration. One feature, one migration; once merged, never edited.
db-new name:
    {{goose}} create {{name}} sql

# Generate the typed Go from the hand-written SQL.
gen-db:
    cd apps/server/db && go tool sqlc generate

# Run the tests that need a live database.
db-test: db-up
    OZALID_TEST_DSN="{{dsn}}" go test -count=1 ./apps/server/internal/adapters/...

# ------------------------------------------------------------------- contract

# Regenerate everything derived from the OpenAPI document.
# The document is the source of truth (backend ADR 0002); everything below it
# is an artifact and is never edited by hand.
gen: gen-bundle gen-server gen-web gen-db

# Bundle apps/server/api/src/ into the single committed artifact.
gen-bundle:
    {{web}}/node_modules/.bin/redocly bundle apps/server/api/src/openapi.yaml -o apps/server/api/openapi.yaml

# Generate the Go handler interface, in strict server mode.
# Generated from the bundle, not from src/: the bundle is the artifact every
# consumer reads, so both sides generate from exactly the same bytes.
gen-server:
    cd apps/server/api && go tool oapi-codegen -config oapi-codegen.yaml openapi.yaml

# Generate the web client's types.
gen-web:
    {{web}}/node_modules/.bin/openapi-typescript apps/server/api/openapi.yaml -o {{web}}/src/shared/api/schema.gen.ts

# Fail if the generated files are not what the document produces.
#
# Compares the tree before and after regenerating rather than asking git, so it
# gives the same answer whether or not the work is committed.
gen-check:
    #!/usr/bin/env bash
    set -euo pipefail
    targets=(
      apps/server/api/openapi.yaml
      apps/server/internal/ports/http/openapi
      apps/server/internal/adapters/postgres/sqlcgen
      {{web}}/src/shared/api/schema.gen.ts
    )
    before=$(find "${targets[@]}" -type f -exec sha256sum {} + | sort)
    just gen >/dev/null
    after=$(find "${targets[@]}" -type f -exec sha256sum {} + | sort)
    if [[ "$before" != "$after" ]]; then
      echo "generated files are stale — 'just gen' changed them:"
      diff <(echo "$before") <(echo "$after") | grep '^[<>]' | awk '{print "  " $3}' | sort -u
      exit 1
    fi

# ------------------------------------------------------------------- server

# Build the server.
be-build:
    go build {{gopkgs}}

# Lint and typecheck the Go side. Layer violations fail here (backend ADR 0001).
be-check:
    go vet {{gopkgs}}
    go tool golangci-lint run ./apps/... ./internal/...

# Run the Go tests.
be-test:
    go test {{gopkgs}}

# Format the Go side.
be-format:
    go tool golangci-lint fmt ./apps/... ./internal/...

# ------------------------------------------------------------------ web client

# Install the web dependencies.
fe-install:
    cd {{web}} && npm install

# Run the dev server.
fe-dev:
    cd {{web}} && npm run dev

# Build the web client.
fe-build:
    cd {{web}} && npm run build

# Typecheck and lint the web client. FSD violations fail here (frontend ADR 0002).
fe-check:
    cd {{web}} && npm run typecheck && npm run lint

# Run the web unit tests.
fe-test:
    cd {{web}} && npm test

# Format the web client.
fe-format:
    cd {{web}} && npm run format

# The ports the end-to-end suite owns. Off the development ones on purpose: a
# suite that borrows the API you are debugging is a suite you turn off.
e2e_port := env_var_or_default("OZALID_E2E_PORT", "8091")
e2e_web_port := env_var_or_default("OZALID_E2E_WEB_PORT", "4174")
e2e_dsn := "postgres://ozalid:ozalid@localhost:" + pg_port + "/ozalid_e2e?sslmode=disable"

# Run the end-to-end suite: a real browser, a real server, a real database.
#
# It owns everything it touches — its own database, its own blob directory, its
# own ports — and tears them down after. Running against the development
# database would destroy a developer's data on every run.
fe-test-e2e:
    #!/usr/bin/env bash
    set -euo pipefail

    docker compose up -d --wait postgres
    # A database of its own, inside the container that is already running.
    psql "postgres://ozalid:ozalid@localhost:{{pg_port}}/postgres"       -v ON_ERROR_STOP=1 -c 'DROP DATABASE IF EXISTS ozalid_e2e' -c 'CREATE DATABASE ozalid_e2e' >/dev/null
    GOOSE_DRIVER=postgres GOOSE_DBSTRING='{{e2e_dsn}}'       GOOSE_MIGRATION_DIR=apps/server/db/migrations go tool goose up >/dev/null

    blobs=$(mktemp -d)
    go build -o "$blobs/server" ./apps/server/cmd/server
    OZALID_ADDR=":{{e2e_port}}" OZALID_DSN='{{e2e_dsn}}' OZALID_BLOB_ROOT="$blobs/blobs"       "$blobs/server" >"$blobs/server.log" 2>&1 &
    server=$!
    # Everything below is torn down whether the suite passes or fails.
    trap 'kill $server 2>/dev/null || true; rm -rf "$blobs"' EXIT

    for _ in $(seq 1 50); do
      curl -sf "http://localhost:{{e2e_port}}/api/health" >/dev/null && break
      sleep 0.2
    done
    curl -sf "http://localhost:{{e2e_port}}/api/health" >/dev/null || {
      echo "the server never came up:"; cat "$blobs/server.log"; exit 1; }

    cd {{web}}
    # The suite runs against the built client, not the dev server: what CI
    # ships is what it watches.
    OZALID_API="http://localhost:{{e2e_port}}" npm run build >/dev/null
    OZALID_API="http://localhost:{{e2e_port}}" OZALID_E2E_WEB_PORT="{{e2e_web_port}}" \
      npx vite preview >"$blobs/web.log" 2>&1 &
    web=$!
    trap 'kill $server $web 2>/dev/null || true; rm -rf "$blobs"' EXIT

    for _ in $(seq 1 50); do
      curl -sf "http://localhost:{{e2e_web_port}}/" >/dev/null && break
      sleep 0.2
    done
    curl -sf "http://localhost:{{e2e_web_port}}/" >/dev/null || {
      echo "the client never came up:"; cat "$blobs/web.log"; exit 1; }

    OZALID_API="http://localhost:{{e2e_port}}" \
      OZALID_E2E_WEB="http://localhost:{{e2e_web_port}}" \
      npx playwright test

# ----------------------------------------------------------------------- both

# Everything a pull request must pass locally.
check: gen-check be-check fe-check

# Every test suite that needs nothing but the repository.
#
# `fe-test-e2e` is deliberately not here: it needs Docker, and a recipe that
# fails on a machine without it would teach people to skip the recipe.
test: be-test fe-test
