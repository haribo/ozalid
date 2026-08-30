# ozalid — task runner. See docs/git-workflow.md for the branch and PR rules.

set shell := ["bash", "-uc"]

web := "apps/web"

# The web client vendors npm packages that happen to ship Go files, and `./...`
# would walk into them. Every Go recipe works from this list instead.
gopkgs := "$(go list ./... | grep -v /node_modules/)"

# The toolchain the generators run under, read from go.mod so there is one
# source of truth.
#
# Named explicitly, because Go only ever switches *up*: a `toolchain` directive
# cannot pull a machine running a later release back down. And it has to be
# pulled down — oapi-codegen embeds the API document gzipped, and those bytes
# differ between releases, so a developer on a newer Go commits a file CI
# regenerates differently and `gen-check` fails for everyone but whoever
# happened to match.
gotoolchain := "go" + `grep '^go ' go.mod | cut -d' ' -f2`

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
    cd apps/server/db && GOTOOLCHAIN={{gotoolchain}} go tool sqlc generate

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
    cd apps/server/api && GOTOOLCHAIN={{gotoolchain}} go tool oapi-codegen -config oapi-codegen.yaml openapi.yaml

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
# Where mailpit listens. The suite reads the sign-in message from it rather than
# from the database, so the sending is watched too.
smtp_port := env_var_or_default("OZALID_SMTP_PORT", "1045")
mailpit_port := env_var_or_default("OZALID_MAILPIT_PORT", "8045")
e2e_dsn := "postgres://ozalid:ozalid@localhost:" + pg_port + "/ozalid_e2e?sslmode=disable"

# Run the end-to-end suite: a real browser, a real server, a real database.
#
# It owns everything it touches — its own database, its own blob directory, its
# own ports — and tears them down after. Running against the development
# database would destroy a developer's data on every run.
fe-test-e2e:
    #!/usr/bin/env bash
    set -euo pipefail

    docker compose up -d --wait postgres mailpit
    # A database of its own, inside the container that is already running.
    psql "postgres://ozalid:ozalid@localhost:{{pg_port}}/postgres"       -v ON_ERROR_STOP=1 -c 'DROP DATABASE IF EXISTS ozalid_e2e' -c 'CREATE DATABASE ozalid_e2e' >/dev/null
    GOOSE_DRIVER=postgres GOOSE_DBSTRING='{{e2e_dsn}}'       GOOSE_MIGRATION_DIR=apps/server/db/migrations go tool goose up >/dev/null

    blobs=$(mktemp -d)
    go build -o "$blobs/server" ./apps/server/cmd/server
    OZALID_ADDR=":{{e2e_port}}" OZALID_DSN='{{e2e_dsn}}' OZALID_BLOB_ROOT="$blobs/blobs" \
      OZALID_BASE_URL="http://localhost:{{e2e_web_port}}" \
      OZALID_SMTP_HOST=localhost OZALID_SMTP_PORT="{{smtp_port}}" \
      OZALID_SMTP_FROM="ozalid@localhost" \
      "$blobs/server" >"$blobs/server.log" 2>&1 &
    server=$!
    # Everything below is torn down whether the suite passes or fails.
    trap 'kill $server 2>/dev/null || true; rm -rf "$blobs"' EXIT

    for _ in $(seq 1 50); do
      curl -sf "http://localhost:{{e2e_port}}/api/health" >/dev/null && break
      sleep 0.2
    done
    curl -sf "http://localhost:{{e2e_port}}/api/health" >/dev/null || {
      echo "the server never came up:"; cat "$blobs/server.log"; exit 1; }

    # The suite pushes evidence, so it needs a token like any other client —
    # and a token belongs to one project, which is why the suite works inside a
    # single one and gives each test its own case rather than its own project.
    OZALID_DSN='{{e2e_dsn}}' "$blobs/server" bootstrap \
      -name "e2e" -email "e2e@ozalid.test" \
      -project "e2e" -service-account "e2e-runner" > "$blobs/bootstrap.txt"
    token=$(grep -o 'ozp_[A-Za-z0-9_-]*' "$blobs/bootstrap.txt")
    [ -n "$token" ] || { echo "bootstrap minted no token:"; cat "$blobs/bootstrap.txt"; exit 1; }

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
      OZALID_E2E_TOKEN="$token" OZALID_E2E_PROJECT="e2e" \
      OZALID_E2E_MAILPIT="http://localhost:{{mailpit_port}}" \
      npx playwright test

# --------------------------------------------------------------- restore drill

# The ports and database the drill owns. Off everything else, so it can be run
# while the development instance and the e2e suite are up.
drill_port := env_var_or_default("OZALID_DRILL_PORT", "8092")
drill_dsn := "postgres://ozalid:ozalid@localhost:" + pg_port + "/ozalid_drill?sslmode=disable"
drill_restored_dsn := "postgres://ozalid:ozalid@localhost:" + pg_port + "/ozalid_drill_restored?sslmode=disable"

# Prove a backup can be restored, and that the wrong order cannot.
#
# Runs `docs/backups.md` rather than describing it: the procedure is the recipe,
# and a procedure nobody has executed is a procedure nobody has checked. It
# ends by reading a capture's bytes back out of a restored instance, because
# "the restore finished without an error" is not the same as "the evidence is
# there".
restore-drill: db-up
    #!/usr/bin/env bash
    set -euo pipefail

    say() { printf '\n\033[1m%s\033[0m\n' "$1"; }
    die() { printf '\n\033[31mFAILED: %s\033[0m\n' "$1" >&2; exit 1; }

    work=$(mktemp -d)
    trap 'kill $(jobs -p) 2>/dev/null || true; rm -rf "$work"' EXIT

    # Two one-pixel PNGs. Intake refuses anything else, and two colours give two
    # addresses (product.md §2).
    red=$(mktemp); blue=$(mktemp)
    base64 -d > "$red" <<'PNG'
    iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAIAAACQd1PeAAAADElEQVR4nGP4z8AAAAMBAQDJ/pLvAAAAAElFTkSuQmCC
    PNG
    base64 -d > "$blue" <<'PNG'
    iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAIAAACQd1PeAAAADElEQVR4nGNgYPgPAAEDAQAIicLsAAAAAElFTkSuQmCC
    PNG
    hash_of() { printf 'sha256:%s' "$(sha256sum "$1" | cut -d' ' -f1)"; }

    go build -o "$work/server" ./apps/server/cmd/server

    # The server refuses to start without somewhere to send a sign-in link. The
    # drill never sends one — it works with a token, the way a client does — so
    # this only has to be present, not reachable.
    export OZALID_SMTP_HOST=localhost OZALID_SMTP_PORT={{smtp_port}} OZALID_SMTP_FROM=drill@ozalid.test

    # ---------------------------------------------------------------- an instance
    say "an instance with evidence in it"
    psql "postgres://ozalid:ozalid@localhost:{{pg_port}}/postgres" -v ON_ERROR_STOP=1 \
      -c 'DROP DATABASE IF EXISTS ozalid_drill' -c 'CREATE DATABASE ozalid_drill' >/dev/null
    blobs="$work/blobs"

    # The server applies the migrations at boot, so it comes up before anything
    # tries to write: bootstrap against an empty database has no tables to
    # write into.
    OZALID_ADDR=":{{drill_port}}" OZALID_DSN='{{drill_dsn}}' OZALID_BLOB_ROOT="$blobs" \
      "$work/server" >"$work/server.log" 2>&1 &
    for _ in $(seq 1 50); do curl -sf "http://localhost:{{drill_port}}/api/health" >/dev/null && break; sleep 0.2; done
    curl -sf "http://localhost:{{drill_port}}/api/health" >/dev/null || { cat "$work/server.log"; die "the server never came up"; }

    OZALID_DSN='{{drill_dsn}}' "$work/server" bootstrap \
      -name drill -email drill@ozalid.test -project drill -service-account runner > "$work/boot.txt"
    token=$(grep -o 'ozp_[A-Za-z0-9_-]*' "$work/boot.txt")
    [ -n "$token" ] || { cat "$work/boot.txt"; die "bootstrap minted no token"; }

    api() { curl -sS -H "authorization: Bearer $token" "$@"; }
    push() {
      api -X PUT --data-binary "@$2" "http://localhost:{{drill_port}}/api/projects/drill/blobs/$(hash_of "$2")" >/dev/null
      api -X POST -H 'content-type: application/json' \
        "http://localhost:{{drill_port}}/api/projects/drill/editions" -d @- >/dev/null <<JSON
    {"cases":[{"id":"$1","steps":[{"name":"the screen","captures":[
      {"variant":{"theme":"light"},"hash":"$(hash_of "$2")","provenance":{"environmentId":"drill"}}]}]}]}
    JSON
    }
    kase=$(api -X POST -H 'content-type: application/json' \
      "http://localhost:{{drill_port}}/api/projects/drill/cases" -d '{"title":"the one that must survive"}' \
      | python3 -c 'import json,sys; print(json.load(sys.stdin)["id"])')
    push "$kase" "$red"

    # ------------------------------------------------------------- the backups
    # Two pairs are taken, so both orderings can be tried. The rule is: the
    # database first, the blobs second. The store is append-only, so a later
    # blob snapshot is a superset of what an earlier dump can reference.
    # Reversed, the dump names bytes the snapshot does not hold
    # (docs/backups.md).
    say "backing up, twice, so both orderings can be tried"
    tar -C "$blobs" -czf "$work/blobs-early.tar.gz" .
    pg_dump '{{drill_dsn}}' --format=custom --file="$work/early.dump"

    # A run lands between the two, exactly as one would in production.
    later=$(api -X POST -H 'content-type: application/json' \
      "http://localhost:{{drill_port}}/api/projects/drill/cases" -d '{"title":"pushed after the early dump"}' \
      | python3 -c 'import json,sys; print(json.load(sys.stdin)["id"])')
    push "$later" "$blue"

    tar -C "$blobs" -czf "$work/blobs-late.tar.gz" .
    pg_dump '{{drill_dsn}}' --format=custom --file="$work/late.dump"
    kill %1 2>/dev/null || true; wait %1 2>/dev/null || true

    # `early.dump` + `blobs-late.tar.gz` is the database first, the blobs
    # second. `late.dump` + `blobs-early.tar.gz` is the reverse, and is what
    # must fail.

    restore() {
      psql "postgres://ozalid:ozalid@localhost:{{pg_port}}/postgres" -v ON_ERROR_STOP=1 \
        -c 'DROP DATABASE IF EXISTS ozalid_drill_restored' -c 'CREATE DATABASE ozalid_drill_restored' >/dev/null
      pg_restore --dbname='{{drill_restored_dsn}}' --no-owner "$1" >/dev/null
      rm -rf "$work/restored"; mkdir -p "$work/restored"
      tar -C "$work/restored" -xzf "$2"
      OZALID_ADDR=":{{drill_port}}" OZALID_DSN='{{drill_restored_dsn}}' OZALID_BLOB_ROOT="$work/restored" \
        "$work/server" >"$3" 2>&1 &
      for _ in $(seq 1 50); do curl -sf "http://localhost:{{drill_port}}/api/health" >/dev/null && break; sleep 0.2; done
      curl -sf "http://localhost:{{drill_port}}/api/health" >/dev/null || { cat "$3"; die "the restored server never came up"; }
    }
    captureOf() {
      api "http://localhost:{{drill_port}}/api/projects/drill/cases/$1/captures" \
        | python3 -c 'import json,sys; g=json.load(sys.stdin); print(g["steps"][0]["cells"][0]["id"])'
    }
    status() { api -o /dev/null -w '%{http_code}' "$1"; }

    # ------------------------------------------------------ the correct order
    say "restoring the database first, the blobs second"
    restore "$work/early.dump" "$work/blobs-late.tar.gz" "$work/right.log"

    api -o "$work/back.png" "http://localhost:{{drill_port}}/api/projects/drill/captures/$(captureOf "$kase")"
    cmp -s "$red" "$work/back.png" || die "the capture that came back is not the capture that went in"
    echo "  the capture reads back byte for byte"

    # The case pushed after the dump is simply not there. Its bytes are, and
    # nothing references them: an orphan blob, which is what this ordering
    # trades for a missing one.
    [ "$(status "http://localhost:{{drill_port}}/api/projects/drill/cases/$later")" = 404 ] \
      || die "a case created after the dump came back — the dump is not the restore point it claims to be"
    echo "  the case pushed after the dump is absent, and its bytes are a harmless orphan"
    kill %1 2>/dev/null || true; wait %1 2>/dev/null || true

    # ------------------------------------------------------- the wrong order
    # Not a formality: this is the failure the ordering rule exists to prevent,
    # and an operator who reverses it will not find out for months.
    say "restoring the blobs first, the database second — this must break"
    restore "$work/late.dump" "$work/blobs-early.tar.gz" "$work/wrong.log"

    # The row is there. The bytes are not.
    [ "$(status "http://localhost:{{drill_port}}/api/projects/drill/cases/$later")" = 200 ] \
      || die "the later case is missing from the later dump, so this proves nothing"
    orphaned=$(captureOf "$later")
    code=$(status "http://localhost:{{drill_port}}/api/projects/drill/captures/$orphaned")
    [ "$code" = 404 ] || die "a capture whose bytes were not restored answered $code, want 404"
    echo "  a capture whose bytes are missing answers 404"

    grep -q 'a stored address has no bytes behind it' "$work/wrong.log" \
      || { cat "$work/wrong.log"; die "the store lost evidence and said nothing in its log"; }
    echo "  and the log says so, which is the only way an operator ever finds out"

    # The capture from before the snapshot still reads, so the failure above is
    # about the missing bytes and not about the instance being broken.
    [ "$(status "http://localhost:{{drill_port}}/api/projects/drill/captures/$(captureOf "$kase")")" = 200 ] \
      || die "the older capture stopped working too, so the check above proves nothing"
    echo "  the older capture still reads, so the failure is the missing bytes and nothing else"
    kill %1 2>/dev/null || true; wait %1 2>/dev/null || true

    say "the drill passed"

# ----------------------------------------------------------------------- both

# Everything a pull request must pass locally.
check: gen-check be-check fe-check

# Every test suite that needs nothing but the repository.
#
# `fe-test-e2e` is deliberately not here: it needs Docker, and a recipe that
# fails on a machine without it would teach people to skip the recipe.
test: be-test fe-test
