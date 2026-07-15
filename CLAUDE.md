# CLAUDE.md

Guidance for AI assistants (and humans) working in this repository.

## Project

`ypcli` is a cross-platform CLI for [yopass](https://github.com/jhaals/yopass)
that publishes text and files as end-to-end-encrypted, self-expiring one-time
secrets. It is a CI/agents/team-first superset of the official yopass CLI.

- Module: `github.com/dantte-lp/ypcli` · Binary: `ypcli`
- Language: Go (see `go.mod`) · License: MIT · Default branch: `master`

## Architecture

Layered `internal/` packages, one responsibility each, no package-global state:

| Package | Responsibility |
|---|---|
| `internal/cli` | cobra command tree, flag/env/profile resolution, exit-code mapping |
| `internal/api` | context-aware HTTP transport, bearer auth, typed errors |
| `internal/crypto` | vendored OpenPGP over ProtonMail/go-crypto (interop-critical) |
| `internal/config` | YAML profiles, precedence merge, token sourcing |
| `internal/output` | text/json printers, terminal QR, download progress |
| `internal/clipboard` | cross-platform clipboard (no CGO) |

## Non-negotiable rules

1. **Interoperability.** `internal/crypto` must stay byte-for-byte compatible
   with the yopass server and openpgp.js. `internal/crypto/interop_test.go`
   proves this using a **test-only** import of `github.com/jhaals/yopass`, which
   must never link into the shipped binary (`go tool nm` shows 0 yopass symbols).
2. **Precedence.** Settings resolve as flag > env (`YPCLI_*`) > active profile >
   default. Keep this exact order.
3. **Secrets.** Tokens are never written to the config file. Keys come from
   `crypto/rand` only.
4. **Errors.** Wrap with `%w`, classify with `errors.Is`/`errors.As`, map to the
   documented exit codes in `internal/cli/exit.go`.
5. **Logging.** `log/slog` only; `--verbose` raises the level to debug.

## Workflow

```bash
make verify   # build + test (-race) + lint + docs + vuln — run before every PR
```

Commits follow Conventional Commits; documentation is bilingual with English
(`docs/en/`) canonical and Russian (`docs/ru/`) mirroring it.

## Reference

- Design: `docs/superpowers/specs/2026-07-15-ypcli-design.md`
- Plan: `docs/superpowers/plans/2026-07-15-ypcli-implementation.md`
- CLI reference: `docs/en/04-cli.md`
