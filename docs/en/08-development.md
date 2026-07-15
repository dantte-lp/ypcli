# Development

## Prerequisites

- Go (see `go.mod` for the required version)
- `golangci-lint` v2
- For end-to-end tests: [`uv`] and a container engine (`podman` or `docker`)
- Optional: `goreleaser`, `markdownlint-cli2`, `yamllint`, `cspell`

## Make targets

```bash
make build       # build the ypcli binary with version ldflags
make test        # go test -race -cover ./...
make lint        # golangci-lint v2
make lint-docs   # markdownlint + yamllint + cspell
make vuln        # govulncheck ./...
make verify      # build + test + lint + vuln
make e2e         # end-to-end suite (uv + ruff + ty + live yopass container)
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
tests/e2e/        Python end-to-end suite (uv + ruff + ty)
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

## End-to-end tests

The Go unit tests cover functions in isolation; the `tests/e2e/` suite is a
black-box layer that drives the **compiled `ypcli` binary** against a **live
yopass server** (started in a container) to verify every command, flag, and exit
code end to end. It is a Python project managed with [`uv`], linted with
[`ruff`], and type-checked with [`ty`].

```bash
make e2e
# or, from tests/e2e:
uv run ruff check .
uv run ty check .
uv run pytest -v
```

The session fixture builds the binary once and starts `memcached` +
`jhaals/yopass` via `podman`/`docker`; a small in-process fake server covers
cases the free image cannot produce deterministically (auth `401`, missing
`/version`, request-header capture). Cryptographic interoperability with the
real yopass/openpgp.js is proven separately by the [interoperability
gate](#interoperability-gate).

Useful environment variables:

| Variable | Effect |
|---|---|
| `YPCLI_BIN` | Use an existing `ypcli` binary instead of building one |
| `YPCLI_E2E_API` | Target an already-running yopass server (skip container startup) |
| `YPCLI_E2E_ARGON2` | Set to `1` when that external server has Argon2 enabled |

Coverage is documented in [`tests/e2e/README.md`](../../tests/e2e/README.md).
The `e2e` GitHub Actions workflow runs the suite on every push and pull request.

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

[`uv`]: https://docs.astral.sh/uv/
[`ruff`]: https://docs.astral.sh/ruff/
[`ty`]: https://github.com/astral-sh/ty
