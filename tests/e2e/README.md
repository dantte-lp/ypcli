# ypcli end-to-end tests

Automated black-box tests that drive the compiled `ypcli` binary against a
**live yopass server** (started in a container) plus a controllable in-process
fake server for auth and error-code cases. Managed with [uv], linted with
[ruff], type-checked with [ty].

## Run

```bash
# from the repository root
make e2e

# or directly
cd tests/e2e
uv run ruff check .
uv run --with ty ty check .
uv run pytest -v
```

## Requirements

- Go toolchain (to build the `ypcli` binary; or set `YPCLI_BIN`)
- A container engine — `podman` or `docker` — to run yopass + memcached
- `uv`

## Environment variables

| Variable | Effect |
|---|---|
| `YPCLI_BIN` | Use an existing `ypcli` binary instead of building one |
| `YPCLI_E2E_API` | Use an already-running yopass server (skip container startup) |
| `YPCLI_E2E_ARGON2` | Set to `1` when the external server has Argon2 enabled |

## Coverage

| File | Area |
|---|---|
| `test_roundtrip.py` | send↔receive text/file/stdin, one-time, multi-view, manual key, Argon2, expirations |
| `test_send.py` | input sources, flags, JSON shape, QR, clipboard, expiration validation |
| `test_receive.py` | URL / `--id` targets, output routing, JSON, usage errors |
| `test_config.py` | profile CRUD and flag/env/profile precedence |
| `test_version.py` | client + server version, legacy server fallback |
| `test_completion.py` | shell completion generation, help listing |
| `test_auth.py` | bearer token via flag/env/`token_command`, require-auth |
| `test_exit_codes.py` | every documented exit code (0,2,3,4,5,6,7) |

The fake server (`_harness.py`) covers cases the free yopass image cannot
produce deterministically: `401` (auth), missing `/version`, and header capture.
Cryptographic interoperability with the real yopass/openpgp.js is proven
separately by the Go interop test.

[uv]: https://docs.astral.sh/uv/
[ruff]: https://docs.astral.sh/ruff/
[ty]: https://github.com/astral-sh/ty
