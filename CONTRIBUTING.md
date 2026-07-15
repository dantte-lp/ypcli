# Contributing to ypcli

## Contribution Policy

| Area | Requirement |
|---|---|
| Conduct | Follow [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md). |
| Support | Use [SUPPORT.md](SUPPORT.md) for request routing. |
| Security | Use [SECURITY.md](SECURITY.md) for private vulnerability disclosure. |
| Governance | Follow [GOVERNANCE.md](GOVERNANCE.md) for release and maintainer rules. |
| Maintainers | See [MAINTAINERS.md](MAINTAINERS.md) for review ownership. |

## Getting Started

1. Fork the repository and clone it locally.
2. Install Go (see `go.mod` for the required version) and the tools below.
3. Make your changes on a feature branch.
4. Submit a pull request.

## Development Workflow

```bash
make build       # build the ypcli binary with version ldflags
make test        # go test -race -cover ./...
make lint        # golangci-lint v2
make lint-docs   # markdownlint + yamllint + cspell
make vuln        # govulncheck ./...
make verify      # build + test + lint + docs + vuln
make e2e         # end-to-end suite (uv + ruff + ty + live yopass container)
```

A local Go toolchain is required; the CLI builds with `CGO_ENABLED=0`. The
end-to-end suite additionally needs [`uv`](https://docs.astral.sh/uv/) and a
container engine (`podman` or `docker`); see
[docs/en/08-development.md](docs/en/08-development.md#end-to-end-tests).

## Documentation and Release Standards

The repository follows:

- [Keep a Changelog 1.1.0](https://keepachangelog.com/en/1.1.0/)
- [Semantic Versioning 2.0.0](https://semver.org/spec/v2.0.0.html)
- [Conventional Commits 1.0.0](https://www.conventionalcommits.org/en/v1.0.0/)

Documentation is bilingual. **English (`docs/en/`) is canonical**; Russian
(`docs/ru/`) mirrors it. Update both when changing user-facing behavior.
Changelog entries are curated for users — do not paste raw git logs into
`CHANGELOG.md`. Keep the `Unreleased` section current while work is in progress.

## Commit Messages

Commits and PR titles use Conventional Commits:

```text
feat(cli): add editor mode for send
fix(crypto): preserve filename on binary decrypt
docs(security): document token sourcing
```

Allowed scopes: `cli`, `api`, `crypto`, `config`, `output`, `clipboard`,
`docs`, `ci`, `build`, `deps`, `release`, `test`, `lint`, `security`.

## Code Standards

### Go Conventions

- **Errors**: wrap with context using `%w`
  (`fmt.Errorf("fetch secret %s: %w", id, err)`).
- **Error handling**: use `errors.Is`/`errors.As`, never string matching.
- **Context**: first parameter, never stored in a struct.
- **Logging**: only `log/slog`. Never `fmt.Println` for diagnostics.
- **Naming**: avoid stutter (`package crypto; func Encrypt`, not `CryptoEncrypt`).
- **Imports**: stdlib, blank line, external, blank line, internal.
- **Tests**: table-driven, `t.Parallel()` where safe, always `-race`.

### Interoperability

- The `internal/crypto` packet configuration must stay identical to the yopass
  server and openpgp.js. Any change requires the interop test
  (`internal/crypto/interop_test.go`) to pass.
- The upstream `jhaals/yopass` module is a **test-only** dependency and must
  never link into the shipped binary.

### Security

- Never use the `unsafe` package.
- Never use `math/rand` for keys — use `crypto/rand`.
- Run `make vuln` before adding new dependencies.

## Pull Request Process

1. Open an issue first to discuss significant changes.
2. Create a feature branch from `master`.
3. Make focused, reviewable commits with descriptive messages.
4. Ensure `make verify` passes.
5. Update documentation (both languages) if behavior changes.
6. Add or update tests for new functionality.

### PR Checklist

- [ ] Tests added or updated
- [ ] `make verify` passes
- [ ] `make e2e` passes for CLI behavior changes
- [ ] `make lint-docs` passes for documentation changes
- [ ] Interop test passes for crypto changes
- [ ] Documentation updated in `docs/en/` and `docs/ru/` (if applicable)
- [ ] `CHANGELOG.md` and `CHANGELOG.ru.md` updated (if user-facing change)
- [ ] Commit messages follow Conventional Commits

## License

By contributing to ypcli, you agree that your contributions will be licensed
under the [MIT License](LICENSE).
