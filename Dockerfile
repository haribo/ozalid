# One binary, one container.
#
# The Go server embeds the built client and serves it beside the API, so there
# is no second process, no proxy to configure, and no origin for the client to
# know — it calls /api on whatever host served it.

# ---------------------------------------------------------------- the client
FROM node:26-alpine AS client
WORKDIR /src/apps/web

COPY apps/web/package.json apps/web/package-lock.json ./
RUN npm ci

COPY apps/web/ ./
# The API document the typed client is generated from lives outside apps/web.
COPY apps/server/api/openapi.yaml /src/apps/server/api/openapi.yaml
RUN npm run build

# ---------------------------------------------------------------- the server
# Pinned rather than "1.26 or newer": the committed generated files depend on
# the toolchain that produced them (see `gotoolchain` in the justfile), and a
# build that regenerates nothing still has to agree with them.
FROM golang:1.26.0-alpine AS server
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY apps/server/ ./apps/server/
COPY internal/ ./internal/

# The real client replaces the placeholder committed so `go build` works in a
# checkout where npm never ran.
COPY --from=client /src/apps/web/dist/ ./apps/server/internal/ports/http/webui/dist/

ARG VERSION=dev
RUN CGO_ENABLED=0 go build \
      -ldflags="-s -w -X main.version=${VERSION}" \
      -o /ozalid ./apps/server/cmd/server

# ----------------------------------------------------------------- the image
FROM alpine:3.22

# The server reaches nothing but Postgres and its own disk, so the image needs
# certificates and nothing else.
RUN apk add --no-cache ca-certificates \
 && adduser -D -u 10001 ozalid

COPY --from=server /ozalid /usr/local/bin/ozalid

# Where capture bytes live. Mount a volume here, or lose every image on the
# first redeploy.
ENV OZALID_BLOB_ROOT=/var/lib/ozalid/blobs
RUN mkdir -p /var/lib/ozalid/blobs && chown -R ozalid:ozalid /var/lib/ozalid

USER ozalid
EXPOSE 8090

ENTRYPOINT ["ozalid"]
