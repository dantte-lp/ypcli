# Governance

## Project Model

ypcli is an independent open source project maintained in this repository.

| Area | Policy |
|---|---|
| Default branch | `master` |
| License | MIT |
| Versioning | Semantic Versioning 2.0.0; current release line is `0.x` |
| Changelog | Keep a Changelog 1.1.0 |
| Commits | Conventional Commits 1.0.0 |
| Validation | `make verify` (build + test + lint + docs + vuln) |

## Maintainer Responsibilities

- Review pull requests for correctness, security, tests, and documentation impact.
- Keep release notes curated in `CHANGELOG.md` and `CHANGELOG.ru.md`.
- Keep public documentation declarative and source-backed.
- Maintain repository settings, branch protection, dependency automation, and
  security scanning.
- Preserve the byte-for-byte interoperability guarantee with the yopass web
  frontend and server API.

## Decision Records

| Decision type | Location |
|---|---|
| Design specification | `docs/superpowers/specs/2026-07-15-ypcli-design.md` |
| Implementation plan | `docs/superpowers/plans/2026-07-15-ypcli-implementation.md` |
| Security posture | `SECURITY.md`, `docs/en/07-security.md` |
| Release process | `docs/en/08-development.md` |

## Release Authority

Release tags require:

1. `make verify`.
2. The crypto interop gate passing.
3. Updated changelog entries (`CHANGELOG.md` + `CHANGELOG.ru.md`).
4. A Conventional Commit release commit.
5. An immutable SemVer tag `vX.Y.Z`.

## References

- Semantic Versioning 2.0.0: <https://semver.org/spec/v2.0.0.html>
- Keep a Changelog 1.1.0: <https://keepachangelog.com/en/1.1.0/>
- Conventional Commits 1.0.0: <https://www.conventionalcommits.org/en/v1.0.0/>
