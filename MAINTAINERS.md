# Maintainers

## Current Maintainers

| GitHub user | Role | Scope |
|---|---|---|
| `@dantte-lp` | Owner | Repository administration, releases, security advisories, final merge authority |

## Review Ownership

| Path | Primary review focus |
|---|---|
| `internal/crypto/` | OpenPGP interoperability with yopass/openpgp.js, key handling |
| `internal/api/` | HTTP transport, authentication, error classification |
| `internal/cli/` | Command behavior, flags, exit codes, output modes |
| `internal/config/` | Profile model, precedence, token sourcing |
| `internal/output/`, `internal/clipboard/` | Rendering, QR, clipboard portability |
| `.github/`, `.goreleaser.yaml` | CI, release automation, packaging |
| `docs/`, `README.md`, `CHANGELOG*.md` | Declarative documentation, release notes |

## Maintainer Rules

- Repository settings changes require an issue, pull request, or recorded
  maintainer note.
- Security reports remain private until disclosure is coordinated.
- Release tags are immutable.
- Interoperability with the yopass web frontend must never regress; the
  crypto interop test gate is mandatory.

## Becoming a Maintainer

Maintainer status requires sustained contributions in correctness, security,
or documentation and explicit approval from an existing maintainer.
