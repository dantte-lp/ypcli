# ypcli — Design Specification

**Date:** 2026-07-15
**Author:** Pavel Lavrukhin (`dantte-lp`)
**Status:** Approved

## 1. Purpose

`ypcli` is a cross-platform command-line client for [yopass](https://github.com/jhaals/yopass)
that publishes text and files as end-to-end-encrypted, self-expiring one-time secrets.

It is a **CI / agents / team-first** superset of the official `cmd/yopass` client. The
official client is a single-mode, flag-driven tool with no non-interactive authentication
and no machine-readable output. `ypcli` prioritises unattended usage while remaining fully
interoperable with the yopass web frontend (openpgp.js v6) and server API.

## 2. Differentiators (grounded in upstream issues)

| Capability | Upstream gap / issue | ypcli |
|---|---|---|
| Machine-to-machine auth | `#3654` — only interactive OIDC cookie, no `Authorization: Bearer` | `--token` / `YPCLI_TOKEN` / per-profile `token_command` |
| Machine-readable output | `#19 #776 #749 #2506 #1276 #2219 #3297` — users reverse-engineer curl+gpg | `--json` + strict exit codes |
| Multiple servers | single `--api/--url` | named profiles in `config.yaml` |
| Server compatibility check | `#3374` — `/version` endpoint added | `ypcli version` surfaces server version |
| Sub-command UX | flag-mode monolith | `send` / `receive` / `config` / `version` / `completion` |
| Argon2 S2K | `#3369` server-advertised via `/config` | auto-detect via `/config`, `ARGON2` |

## 3. Interoperability contract (must not break)

Encryption uses OpenPGP symmetric encryption via `github.com/ProtonMail/go-crypto`, with the
**exact** packet configuration used by yopass server and openpgp.js:

- Cipher `AES-256`, hash `SHA-256`, compression `none`, AEAD mode `GCM`.
- S2K: default iterated SHA-256, or `Argon2id` when the server advertises `ARGON2:true`.
- Text secrets: **ASCII-armored** PGP (`-----BEGIN PGP MESSAGE-----`), header
  `Comment: https://yopass.se`.
- Files: **binary** PGP with `FileHints{IsBinary:true, FileName:<name>}` (filename travels
  inside the encrypted payload, never in cleartext).
- Key: 22-char `base64.URLEncoding` (matches current JS/CLI `GenerateKey`).
- ID: 22-char base62 from 16 random bytes.
- Share URL: `{url}/#/{prefix}/{id}[/{key}]`, prefix `s`=secret, `f`=file, `c`=secret+manual
  key, `d`=file+manual key.

**Decision:** production code vendors this ~150-line crypto surface (dependency = only
`go-crypto`). A **test-only** import of `github.com/jhaals/yopass/pkg/yopass` proves
bidirectional round-trip interop and is excluded from the shipped binary.

## 4. API contract (yopass server, v13+)

| Operation | Method / path | Body / headers | Response |
|---|---|---|---|
| Create secret | `POST /create/secret` | JSON `{message, expiration, one_time, require_auth}` | `{message: <id>}` |
| Fetch secret | `GET /secret/{id}` | — | `{message: <armored PGP>}` |
| Create file | `POST /create/file` | `application/octet-stream` body (binary PGP) + `X-Yopass-Expiration`, `X-Yopass-OneTime` | `{message: <id>}` |
| Fetch file | `GET /file/{id}` | `Accept: application/octet-stream` | binary PGP body |
| Server config | `GET /config` | — | `{ARGON2: bool, ...}` |
| Server version | `GET /version` | — | version payload |

Auth: optional `Authorization: Bearer <token>` on create endpoints (and fetch on
auth-gated instances). Expirations currently accepted: `1h`=3600, `1d`=86400, `1w`=604800.
`ypcli` validates client-side against this set and surfaces the server's error verbatim if
it rejects a value.

## 5. Architecture

Layered, each package one responsibility, no global mutable state:

```
cmd/ypcli/main.go        thin entry: inject version ldflags, call cli.Execute(ctx)
internal/
  cli/                   cobra commands: root, send, receive, config, version, completion
  api/                   context-aware HTTP transport: client, secret, file, meta, errors
  crypto/                vendored OpenPGP wrapper (interop-critical) + key/URL helpers
  config/                viper config + named profiles + precedence resolution
  output/                Printer (text|json) abstraction + qr rendering
  clipboard/             cross-platform clipboard wrapper
```

Precedence: `flag > env (YPCLI_*) > active profile in ~/.config/ypcli/config.yaml > default`.

## 6. CLI surface

```
ypcli send    [--file F | --text T | -] [--expiration 1h|1d|1w] [--one-time]
              [--key K] [--require-auth] [--token T] [--qr] [--copy] [--json] [--profile P]
ypcli receive <url | --id ID --key K> [-o FILE|DIR] [--json] [--profile P]
ypcli config  add|list|use|remove <name> [--api ...] [--url ...]
ypcli version
ypcli completion bash|zsh|fish|powershell
```

## 7. Errors & exit codes

Typed errors in `api`, mapped centrally in `cli`:

| Code | Meaning |
|---|---|
| 0 | success |
| 1 | generic |
| 2 | usage / bad flags |
| 3 | config error |
| 4 | network / timeout |
| 5 | auth (401/403) |
| 6 | not found / one-time consumed (404/410) |
| 7 | decrypt / crypto failure |

`--json` renders `{"error":{"code":<int>,"message":<str>}}`. All network calls use
`context.Context` with `--timeout` (default 30s). `--verbose` enables `log/slog` debug.

## 8. Testing

- Unit tests per package; `httptest.Server` for `api`; golden files for `output`.
- Interop round-trip test (test-only dep on `pkg/yopass`): encrypt with ypcli → decrypt
  with yopass and vice versa; plus committed openpgp.js golden vectors.
- CI: `golangci-lint` (v2), `go test -race ./...`, `govulncheck`.
- Go 1.26 idioms: `context`, `log/slog`, `errors.Join`/`errors.Is`, no package globals.

## 9. Distribution

`goreleaser` → GitHub Releases (`darwin|linux|windows` × `amd64|arm64`), Homebrew tap
(`dantte-lp/homebrew-tap`), Scoop bucket + winget. Version/commit/date injected via
`-ldflags`. Container image and SBOM/cosign deferred to a later milestone.
