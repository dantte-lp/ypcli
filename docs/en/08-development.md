# Development

## Prerequisites

- Go (see `go.mod` for the required version)
- `golangci-lint` v2
- Optional: `goreleaser`, `markdownlint-cli2`, `yamllint`, `cspell`

## Make targets

```bash
make build       # build the ypcli binary with version ldflags
make test        # go test -race -cover ./...
make lint        # golangci-lint v2
make lint-docs   # markdownlint + yamllint + cspell
make vuln        # govulncheck ./...
make verify      # build + test + lint + vuln
make snapshot    # local goreleaser snapshot
```

## Layout

```text
cmd/ypcli/        entrypoint (version ldflags)
internal/
  cli/            cobra commands, resolution, exit codes
  api/            HTTP transport
  crypto/         vendored OpenPGP (interop-critical)
  config/         profiles, token sourcing
  output/         printers, qr, progress
  clipboard/      clipboard wrapper
docs/en, docs/ru  bilingual documentation
```

## Interoperability gate

`internal/crypto/interop_test.go` imports `github.com/jhaals/yopass/pkg/yopass`
as a **test-only** dependency and asserts bidirectional round-trips (ypcli ↔
upstream, text + file + Argon2). It must pass for any crypto change.

Confirm the dependency never links into the binary:

```bash
go build -o /tmp/ypcli ./cmd/ypcli
go tool nm /tmp/ypcli | grep -c jhaals/yopass   # expect 0
```

## Coding standards

- Wrap errors with `%w`; classify with `errors.Is`/`errors.As`.
- `context.Context` is the first parameter, never stored in a struct.
- Logging via `log/slog` only.
- Table-driven tests, `t.Parallel()` where safe, always `-race`.
- Conventional Commits for messages; see [CONTRIBUTING.md](../../CONTRIBUTING.md).

## Releasing

Releases are cut by pushing a SemVer tag; goreleaser builds the platform matrix
and publishes archives plus Homebrew/Scoop/winget manifests.

```bash
make verify
git tag -a v0.1.0 -m "ypcli v0.1.0"
git push origin v0.1.0
```

The release workflow requires `TAP_GITHUB_TOKEN` (write access to the tap,
bucket, and winget repositories) in addition to the default `GITHUB_TOKEN`.
