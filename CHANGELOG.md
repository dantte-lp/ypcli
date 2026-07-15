# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog 1.1.0](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning 2.0.0](https://semver.org/spec/v2.0.0.html).

Russian translation: [CHANGELOG.ru.md](CHANGELOG.ru.md).

## [Unreleased]

## [0.1.0] - 2026-07-15

### Added

- **`send` command** — encrypt text (stdin/`--text`) or files (`--file`) with
  client-side OpenPGP and publish a one-time share URL. Supports `--expiration`
  (`1h`/`1d`/`1w`), `--one-time`, `--require-auth`, manual `--key`, `--qr`
  (terminal QR code), and `--copy` (system clipboard).
- **`receive` command** — fetch and decrypt a secret from a share URL or
  `--id`/`--key`. Text is written to stdout; files are written to their embedded
  filename or `--output` (with a streaming download progress indicator).
- **`config` command** — manage named server profiles (`add`/`list`/`use`/`remove`)
  in `$XDG_CONFIG_HOME/ypcli/config.yaml` (mode 0600).
- **`version` command** — report the client build and the server `/version`
  endpoint, degrading gracefully on pre-13.x servers.
- **Bearer-token authentication** — `--token`, `YPCLI_TOKEN`, or a per-profile
  `token_command`; tokens are never persisted to disk.
- **Machine-readable output** — `--json` on every command, plus stable exit codes
  (2 usage, 3 config, 4 network, 5 auth, 6 not-found/consumed, 7 crypto).
- **Argon2id auto-detection** — key derivation is selected per request from the
  server `/config` endpoint.
- **Cross-platform release** — goreleaser matrix for macOS/Linux/Windows on
  amd64 + arm64, with Homebrew cask, Scoop, and winget publishing.

### Security

- Byte-for-byte OpenPGP interoperability with the yopass web frontend
  (openpgp.js v6), proven by a test-only round-trip against upstream yopass that
  never links into the shipped binary.

[Unreleased]: https://github.com/dantte-lp/ypcli/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/dantte-lp/ypcli/releases/tag/v0.1.0
