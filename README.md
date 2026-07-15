# ypcli

A cross-platform command-line client for [yopass](https://github.com/jhaals/yopass)
that publishes text and files as **end-to-end-encrypted, self-expiring one-time
secrets**. Encryption happens client-side (OpenPGP); the decryption key never
reaches the server.

`ypcli` is a **CI / agents / team-first** superset of the official `yopass` CLI:

- 🔐 **Bearer-token auth** — works with `REQUIRE_AUTH`-gated / OIDC-protected instances
  ([`--token`](#authentication), `YPCLI_TOKEN`, or a per-profile `token_command`)
- 🤖 **Machine-readable** — `--json` output and strict, stable [exit codes](#exit-codes)
- 🗂 **Profiles** — target multiple yopass servers without repeating `--api/--url`
- 🧩 **Sub-commands** — `send` / `receive` / `config` / `version` + shell completion
- 🔁 **Byte-for-byte interoperable** with the yopass web frontend (openpgp.js v6),
  proven by a round-trip test against upstream
- 📦 **CGO-free static binaries** for macOS/Linux/Windows on amd64 + arm64

## Install

```bash
# Homebrew (macOS)
brew install dantte-lp/tap/ypcli

# Scoop (Windows)
scoop bucket add dantte-lp https://github.com/dantte-lp/scoop-bucket
scoop install ypcli

# Go
go install github.com/dantte-lp/ypcli/cmd/ypcli@latest
```

Or grab a prebuilt archive from [Releases](https://github.com/dantte-lp/ypcli/releases).

## Quick start

```bash
# Encrypt text from stdin and print a one-time share URL
printf 'my secret' | ypcli send

# Encrypt a file, valid for one day
ypcli send --file ./db.env --expiration 1d

# Receive and decrypt (text to stdout)
ypcli receive 'https://yopass.se/#/s/ID/KEY'

# Receive a file into a directory (original name preserved)
ypcli receive 'https://yopass.se/#/f/ID/KEY' -o ./out/
```

## CI / automation

`--json` and exit codes make `ypcli` safe to script:

```bash
url=$(printf "$PASSWORD" | ypcli send --json --one-time | jq -r .url)
echo "share: $url"
```

Against an authenticated instance, source the token from a secrets manager
instead of storing it:

```bash
ypcli config add prod \
  --api https://api.yopass.corp \
  --url https://yopass.corp \
  --token-command 'vault read -field=token secret/yopass'

ypcli send --profile prod --file ./service.key --json
```

Precedence for every setting is **flag > env (`YPCLI_*`) > active profile > default**.

### Authentication

| Source | Example |
|---|---|
| Flag | `ypcli send --token "$TOK" ...` |
| Environment | `YPCLI_TOKEN=… ypcli send ...` |
| Profile command | `token_command: vault read -field=token secret/yopass` |

The token is sent as `Authorization: Bearer <token>` and is never written to the
config file.

## Exit codes

| Code | Meaning |
|---|---|
| 0 | success |
| 1 | generic error |
| 2 | usage / bad flags |
| 3 | configuration error |
| 4 | network / timeout |
| 5 | auth failure (401/403) |
| 6 | not found / one-time already consumed (404/410) |
| 7 | decryption / crypto failure |

## How it works

Text is ASCII-armored OpenPGP; files are binary OpenPGP with the filename
embedded in the encrypted payload. Cipher `AES-256`, hash `SHA-256`, AEAD `GCM`;
key derivation is iterated SHA-256, or memory-hard **Argon2id** when the server
advertises it (`GET /config` → `ARGON2`), auto-detected per request. The random
key lives only in the share URL fragment (`#/…`), which browsers never send to
the server.

See [`docs/cli.md`](docs/cli.md) for the full command reference.

## Development

```bash
task test    # go test -race -cover ./...
task lint    # golangci-lint run
task vuln    # govulncheck ./...
task build   # build ./cmd/ypcli
```

Interoperability with upstream yopass is enforced by a **test-only** dependency
on `github.com/jhaals/yopass/pkg/yopass`; it never links into the shipped binary.

## License

MIT — see [LICENSE](LICENSE).
