# Deploying

ozalid ships as **two artefacts**, and a deployment decides how they meet.

| | what it is | who serves it |
| --- | --- | --- |
| `ozalid-linux-<arch>` | the server. Answers `/api` and nothing else | itself |
| `ozalid-web-<version>.zip` | the web client, Vite's `dist/` | whatever sits in front |

The server carries no copy of the client. Pointing a browser at it answers a
404 that says so, rather than leaving an operator to wonder whether the process
is broken.

## What has to be in front

**Something that terminates TLS.** The session cookie is `Secure`, so it is
never set over plain HTTP — without TLS nobody can sign in at all. This is loud,
not silent.

**Something that serves the client**, under the same host as the API. The
client calls `/api` relative to wherever it was served from, which is what makes
CORS a question nobody has to answer.

**A fallback to `index.html`.** The client owns its own routes: a browser
loading `/cases/abc123` directly must be handed the application, not a 404. In
nginx that is `try_files $uri /index.html`.

**Caching.** The names under `assets/` carry a content hash, so those files can
be kept forever and `index.html` must not be:

```nginx
location /assets/ {
    add_header Cache-Control "public, max-age=31536000, immutable";
}
location / {
    add_header Cache-Control "no-cache";
    try_files $uri /index.html;
}
```

## What the server needs

| | |
| --- | --- |
| `OZALID_DSN` | PostgreSQL. Migrations are applied at boot |
| `OZALID_BLOB_ROOT` | where capture bytes live. Losing it loses every image ever reviewed ([backups](backups.md)) |
| `OZALID_BASE_URL` | the address people reach, exactly. It is what a sign-in link contains, and the server cannot work it out from behind a proxy |
| `OZALID_SMTP_HOST`, `OZALID_SMTP_PORT`, `OZALID_SMTP_FROM` | where sign-in links are sent. The server refuses to start without them |
| `OZALID_TRUSTED_PROXY` | set it when a proxy is in front. Unset, `X-Forwarded-For` is ignored and every request looks like one source, which turns the per-source rate limit into a per-instance one |
| `OZALID_ADDR` | what to listen on, `:8090` by default |

## The first account

`ozalid bootstrap -name … -email … -project … -service-account …` makes the
first administrator, a project, and a token. It prints the token once.

It makes a **new administrator every time it runs**, so it is for the first one
only. Everybody after that is created from the `/accounts` screen — and a
second administrator is not optional: the server refuses to remove the last one,
so an instance with a single administrator who loses their access can no longer
add anybody.

## Pairing the two

Nothing checks that the binary and the archive are the same version. They are
published together and carry the same version in their names; a deployment that
mixes them fails in a reviewer's browser, on a call to an endpoint that moved,
rather than at startup.
