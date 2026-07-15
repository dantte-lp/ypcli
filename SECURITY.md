# Security Policy

## Public Policy Files

| File | Purpose |
|---|---|
| [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) | Community behavior and enforcement policy |
| [CONTRIBUTING.md](CONTRIBUTING.md) | Contribution workflow and validation gates |
| [SUPPORT.md](SUPPORT.md) | Non-security support routing |
| [GOVERNANCE.md](GOVERNANCE.md) | Maintainer and release governance |

## Reporting a Vulnerability

If you discover a security vulnerability in ypcli, please report it
responsibly. **Do not open a public GitHub issue for security vulnerabilities.**

Use [GitHub Security Advisories](https://github.com/dantte-lp/ypcli/security/advisories/new)
to report the vulnerability privately. Include:

- Description of the vulnerability
- Steps to reproduce (if applicable)
- Affected versions
- Potential impact

## Security Model

ypcli performs **client-side** end-to-end encryption; the server never sees
plaintext or the decryption key.

- **Encryption** (`internal/crypto`): OpenPGP symmetric encryption via
  `github.com/ProtonMail/go-crypto` — AES-256, SHA-256, AEAD GCM, with optional
  memory-hard Argon2id S2K when the server advertises it. The configuration is
  byte-for-byte identical to the yopass server and the openpgp.js frontend.
- **Keys** (`internal/crypto`): generated with `crypto/rand`. The random key
  lives only in the URL fragment (`#/…`), which browsers never transmit to the
  server. Manual keys are omitted from the URL entirely.
- **Authentication** (`internal/api`, `internal/config`): bearer tokens are read
  from `--token`, `YPCLI_TOKEN`, or a per-profile `token_command`, and are
  **never persisted** to the config file. The config file is written mode 0600.
- **Transport** (`internal/api`): all requests are context-bounded with a
  timeout; TLS verification uses the Go standard library defaults.

## Security Measures

- The `unsafe` package is never used.
- Only `crypto/rand` is used for key and identifier generation.
- `gosec` runs in CI; `govulncheck` runs on every push and pull request.
- The shipped binary excludes the upstream `jhaals/yopass` module (test-only
  dependency), keeping the release supply chain to `ProtonMail/go-crypto` and
  the CLI framework.

## Supported Versions

The latest published release is supported. Pre-1.0 releases may include breaking
changes per Semantic Versioning rule 4.
