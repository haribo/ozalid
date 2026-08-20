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

# ------------------------------------------------------------- server and cli

# Build both binaries.
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

# ----------------------------------------------------------------------- both

# Everything a pull request must pass locally.
check: be-check fe-check

# Every test suite.
test: be-test fe-test
