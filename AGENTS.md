# AGENTS.md

Instructions for AI agents and human contributors working in this repository.
This is the canonical agent guide; `CLAUDE.md` imports it. It follows the
[agents.md](https://agents.md) convention, so Codex, Cursor, Zed, Gemini CLI,
Aider, and similar tools read it too.

## Project

`ypcli` is a cross-platform CLI for [yopass](https://github.com/jhaals/yopass)
that publishes text and files as end-to-end-encrypted, self-expiring one-time
secrets. It is a CI/agents/team-first superset of the official yopass CLI.

- Module: `github.com/dantte-lp/ypcli` · Binary: `ypcli`
- Language: Go (version in `go.mod`) · License: MIT · Default branch: `master`

## Start here

```bash
make build     # build the ypcli binary
make test      # go test -race -cover ./...
make lint      # golangci-lint v2
make lint-docs # markdownlint + yamllint + cspell
make vuln      # govulncheck ./...
make verify    # build + test + lint + vuln  (run before every change set)
make e2e       # Python end-to-end suite (uv + ruff + ty + live yopass container)
```

CLI builds are `CGO_ENABLED=0`; the race detector needs `CGO_ENABLED=1`.

## Architecture

Layered `internal/` packages, one responsibility each, no package-global state:

| Package | Responsibility |
|---|---|
| `internal/cli` | cobra command tree, flag/env/profile resolution, exit-code mapping |
| `internal/api` | context-aware HTTP transport, bearer auth, typed errors |
| `internal/crypto` | vendored OpenPGP over ProtonMail/go-crypto (interop-critical) |
| `internal/config` | YAML profiles, global defaults, precedence, token/command sourcing |
| `internal/vault` | Vault/OpenBao KV v2 read (secret payload source) |
| `internal/output` | text/json printers, terminal QR, download progress |
| `internal/clipboard` | cross-platform clipboard (no CGO) |
| `tests/e2e` | Python black-box suite driving the binary against a live yopass |

## Non-negotiable rules

1. **Interoperability.** `internal/crypto` stays byte-for-byte compatible with the
   yopass server and openpgp.js. `internal/crypto/interop_test.go` proves it via a
   **test-only** import of `github.com/jhaals/yopass`, which must never link into
   the shipped binary (`go tool nm` shows 0 yopass symbols).
2. **Precedence.** Settings resolve as flag > env (`YPCLI_*`) > active profile >
   global defaults > built-in default. `send` input priority is
   `--vault-path` > `--input-command` > `--file` > `--text` > piped stdin > editor.
3. **Secrets.** Tokens are never written to the config file; keys come from
   `crypto/rand` only; no `math/rand`, no `unsafe`.
4. **Errors.** Wrap with `%w`, classify with `errors.Is`/`errors.As`, map to the
   documented exit codes in `internal/cli/exit.go`.
5. **Logging.** `log/slog` only; `--verbose` raises the level to debug.

## Contribution workflow

- `master` is protected: changes land via pull request and the full CI gate
  (lint, docs, tests on 3 OSes, e2e, govulncheck, gosec, CodeQL, commitlint).
- Commits and PR titles use [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/);
  allowed scopes are in `.commitlintrc.yaml`.
- Documentation is bilingual: **English (`docs/en/`) is canonical**, Russian
  (`docs/ru/`) mirrors it file-for-file and section-for-section. Update both,
  plus `CHANGELOG.md` and `CHANGELOG.ru.md`, for any user-facing change.
- Keep `docs/en/04-cli.md` (and its RU mirror) in sync with the actual cobra
  flags, and the exit-code tables in sync with `internal/cli/exit.go`.

## Reference

- Documentation index: [`docs/README.md`](docs/README.md)
- Architecture: [`docs/en/01-architecture.md`](docs/en/01-architecture.md)
- CLI reference: [`docs/en/04-cli.md`](docs/en/04-cli.md)
- Security model: [`docs/en/07-security.md`](docs/en/07-security.md)
- Contributing / security / support: `CONTRIBUTING.md`, `SECURITY.md`, `SUPPORT.md`
