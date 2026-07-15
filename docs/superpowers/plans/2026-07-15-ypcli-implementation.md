# ypcli Implementation Plan (Waterfall)

> **For agentic workers:** Steps use checkbox (`- [ ]`) syntax. Execute phases in order;
> each phase is independently buildable and testable. Commit at every task boundary. TDD
> where a unit has logic; scaffolding tasks commit once green.

**Goal:** Build a cross-platform, CI/agents/team-first yopass CLI (`ypcli`) that is
byte-for-byte interoperable with the yopass web frontend and server API.

**Architecture:** Layered Go module (`cli` → `api` + `crypto` + `config` + `output` +
`clipboard`), cobra/viper front end, vendored OpenPGP crypto over `ProtonMail/go-crypto`,
context-aware HTTP transport with Bearer auth and JSON output.

**Tech Stack:** Go 1.26, cobra, viper, ProtonMail/go-crypto, go-qrcode (or skip2/go-qrcode),
golang.design/x/clipboard, goreleaser, golangci-lint v2.

**Module:** `github.com/dantte-lp/ypcli` · **Binary:** `ypcli`

---

## Phase 0 — Project scaffold & toolchain

### Task 0.1: Module + layout
**Files:** Create `go.mod`, `cmd/ypcli/main.go`, `.gitignore`

- [ ] Init module `github.com/dantte-lp/ypcli` with `go 1.26`.
- [ ] `cmd/ypcli/main.go`: `package main` with `version`, `commit`, `date` vars (ldflags
  targets) that calls `cli.Execute(context.Background(), buildInfo)`.
- [ ] `.gitignore`: `/dist/`, `ypcli`, `*.out`, `coverage.*`.
- [ ] `go build ./...` succeeds (stub `cli.Execute`).
- [ ] Commit: `chore: scaffold go module and entrypoint`.

### Task 0.2: Lint + task runner
**Files:** Create `.golangci.yml` (v2 schema), `Taskfile.yml`, `.editorconfig`

- [ ] `.golangci.yml` v2: enable `errcheck, govet, staticcheck, revive, gosec, gocritic,
  misspell, unconvert, unparam, bodyclose, contextcheck, errorlint, copyloopvar`.
- [ ] `Taskfile.yml`: `build`, `test` (`go test -race -cover ./...`), `lint`
  (`golangci-lint run`), `vuln` (`govulncheck ./...`), `tidy`, `release-snapshot`.
- [ ] Verify `golangci-lint run` clean on scaffold.
- [ ] Commit: `chore: add golangci-lint v2 config and Taskfile`.

---

## Phase 1 — Crypto core (interop-critical)

### Task 1.1: OpenPGP config + key/ID generation
**Files:** Create `internal/crypto/crypto.go`, Test `internal/crypto/crypto_test.go`

- [ ] **Test first:** `GenerateKey()` returns 22-char base64url; `GenerateID()` returns
  22-char base62; both distinct across calls.
- [ ] Implement `pgpConfig`/`pgpConfigArgon2` (`AES256`, `SHA256`, `CompressionNone`,
  `AEADModeGCM`; Argon2 variant sets `S2KConfig{S2KMode: s2k.Argon2S2K}`) mirroring yopass
  exactly; `GenerateKey`, `GenerateID`.
- [ ] Tests pass. Commit: `feat(crypto): key/id generation and pgp config`.

### Task 1.2: Encrypt / Decrypt (text + binary)
**Files:** Modify `internal/crypto/crypto.go`, `internal/crypto/crypto_test.go`

- [ ] **Test first:** round-trip `Encrypt`→`Decrypt` for text; `EncryptBinary`→`Decrypt`
  recovers filename; wrong key → `ErrInvalidKey`; empty key → `ErrEmptyKey`; `Decrypt`
  auto-detects armored vs binary.
- [ ] Implement `Encrypt(r, key)`, `EncryptBinary(r, key, filename)`, `*WithArgon2`,
  `Decrypt(r, key) (content, filename, err)` and sentinel errors, adapted from yopass.
- [ ] Tests pass. Commit: `feat(crypto): symmetric encrypt/decrypt with argon2 option`.

### Task 1.3: URL + expiration helpers
**Files:** Modify `internal/crypto/crypto.go`, `internal/crypto/crypto_test.go`

- [ ] **Test first:** `SecretURL`/`ParseURL` round-trip for all prefixes `s/f/c/d`;
  `ExpirationSeconds("1h"/"1d"/"1w")` + invalid.
- [ ] Implement `SecretURL`, `ParseURL`, `ExpirationSeconds`, `ValidExpirationSeconds`.
- [ ] Tests pass. Commit: `feat(crypto): url building/parsing and expiration mapping`.

### Task 1.4: Interop proof (test-only dep)
**Files:** Create `internal/crypto/interop_test.go`, `internal/crypto/testdata/openpgpjs_*.asc`

- [ ] Add test-only import `github.com/jhaals/yopass/pkg/yopass`; assert ypcli-encrypted
  text/file decrypts via `yopass.Decrypt` and yopass-encrypted decrypts via ypcli.
- [ ] Add committed openpgp.js golden vector(s); assert ypcli decrypts them.
- [ ] `go test ./internal/crypto/...` green; `go build ./cmd/...` does NOT link yopass.
- [ ] Commit: `test(crypto): bidirectional interop with yopass and openpgp.js`.

---

## Phase 2 — Config & profiles

### Task 2.1: Config model + load/save
**Files:** Create `internal/config/config.go`, Test `internal/config/config_test.go`

- [ ] **Test first:** load missing file → defaults; round-trip save/load of a `Profile`
  (`api`, `url`, `expiration`, `one_time`, `argon2`, `token_command`); XDG path resolution.
- [ ] Implement `Config{Active string; Profiles map[string]Profile}`, `Load(path)`,
  `Save`, `DefaultPath()` (`$XDG_CONFIG_HOME/ypcli/config.yaml` fallback `~/.config`).
- [ ] Tests pass. Commit: `feat(config): profile model with yaml persistence`.

### Task 2.2: Resolution + token sourcing
**Files:** Modify `internal/config/config.go`, `internal/config/config_test.go`

- [ ] **Test first:** precedence flag>env>profile>default; `token_command` exec returns
  trimmed stdout; `--token`/`YPCLI_TOKEN` override.
- [ ] Implement `Resolve(flags, env) Settings` and `Token(ctx) (string, error)` (runs
  `token_command` via `exec.CommandContext`). Never persist token plaintext.
- [ ] Tests pass. Commit: `feat(config): settings resolution and token sourcing`.

---

## Phase 3 — API transport

### Task 3.1: Client + typed errors
**Files:** Create `internal/api/client.go`, `internal/api/errors.go`, Test `internal/api/client_test.go`

- [ ] **Test first (httptest):** Bearer header set when token present; `--timeout`
  honored; status→typed error mapping (401/403→`ErrUnauthorized`, 404/410→`ErrNotFound`,
  5xx→`ErrServer`, transport→`ErrNetwork`).
- [ ] Implement `Client{BaseAPI, Token, HTTP, Timeout}`, `New(...)`, request helper with
  `context`, JSON decode of `{message}` / `{error}` envelope, sentinel errors.
- [ ] Tests pass. Commit: `feat(api): context-aware client with typed errors`.

### Task 3.2: Secret + file endpoints
**Files:** Create `internal/api/secret.go`, `internal/api/file.go`, Tests alongside

- [ ] **Test first (httptest):** `CreateSecret` posts correct JSON + returns id;
  `FetchSecret` returns armored message; `CreateFile` sets octet-stream + `X-Yopass-*`
  headers; `FetchFile` streams body; error bodies surface server message.
- [ ] Implement the four methods on `Client` (context-aware, streaming for files).
- [ ] Tests pass. Commit: `feat(api): create/fetch secret and file endpoints`.

### Task 3.3: Meta endpoints
**Files:** Create `internal/api/meta.go`, Test `internal/api/meta_test.go`

- [ ] **Test first:** `Config()` decodes `{ARGON2:true}`; `Version()` decodes payload;
  missing `/version` on old servers → graceful `ErrUnsupported`.
- [ ] Implement `Config(ctx)` and `Version(ctx)`.
- [ ] Tests pass. Commit: `feat(api): config and version endpoints`.

---

## Phase 4 — Output, QR, clipboard

### Task 4.1: Printer abstraction
**Files:** Create `internal/output/output.go`, Test `internal/output/output_test.go`

- [ ] **Test first (golden):** text printer prints URL line; json printer emits
  `{id,url,key,expiration,one_time,file}`; error rendering in both modes.
- [ ] Implement `Printer` interface + `Text`/`JSON` impls; `SendResult`, `ReceiveResult`,
  `ErrorResult` types.
- [ ] Tests pass. Commit: `feat(output): text and json printers`.

### Task 4.2: QR + clipboard
**Files:** Create `internal/output/qr.go`, `internal/clipboard/clipboard.go`, tests

- [ ] **Test first:** `QR(url)` returns non-empty string containing block chars; clipboard
  wrapper has a no-op/error path when headless (build-tag or runtime guard).
- [ ] Implement `QR` (skip2/go-qrcode → `ToSmallString`) and `Copy(string)` wrapper.
- [ ] Tests pass. Commit: `feat(output): terminal qr and clipboard helpers`.

---

## Phase 5 — CLI wiring (cobra)

### Task 5.1: Root command
**Files:** Create `internal/cli/root.go`, `internal/cli/exit.go`, Test `internal/cli/root_test.go`

- [ ] **Test first:** unknown command → exit 2; `--json` global flag parsed; version info
  threaded; central error→exit-code mapper.
- [ ] Implement `Execute(ctx, buildInfo)`, persistent flags (`--profile --json --verbose
  --timeout --api --url --token`), viper binding (`YPCLI_` env), `exitError` mapping.
- [ ] Tests pass. Commit: `feat(cli): root command, global flags, exit mapping`.

### Task 5.2: send
**Files:** Create `internal/cli/send.go`, Test `internal/cli/send_test.go`

- [ ] **Test first (httptest server):** stdin text → URL; `--file` → file URL with prefix
  `f`; `--json` output; `--one-time`/`--expiration`/`--require-auth` propagated; `--key`
  → manual-key URL (`c`/`d`); argon2 auto-selected from `/config`.
- [ ] Implement input resolution (`--text` | `--file` | stdin), argon2 detection, encrypt,
  create, URL assembly, `--qr`/`--copy` side effects.
- [ ] Tests pass. Commit: `feat(cli): send command`.

### Task 5.3: receive
**Files:** Create `internal/cli/receive.go`, Test `internal/cli/receive_test.go`

- [ ] **Test first (httptest):** URL positional → plaintext to stdout; file URL + `-o dir`
  → writes original filename; manual-key URL without `--key` → usage error (exit 2);
  consumed one-time → exit 6; `--json`.
- [ ] Implement URL/flag parsing, fetch (secret|file), decrypt, output routing (stdout|`-o`).
- [ ] Tests pass. Commit: `feat(cli): receive command`.

### Task 5.4: config + version + completion
**Files:** Create `internal/cli/config.go`, `internal/cli/version.go`; completion via cobra builtin

- [ ] **Test first:** `config add/list/use/remove` mutate the file; `version` prints client
  build + server `/version` (and degrades if unsupported).
- [ ] Implement subcommands; register `completion` (cobra `GenBashCompletion` etc.).
- [ ] Tests pass. Commit: `feat(cli): config, version, completion commands`.

---

## Phase 6 — Release & CI

### Task 6.1: goreleaser
**Files:** Create `.goreleaser.yaml`

- [ ] Builds matrix `darwin|linux|windows` × `amd64|arm64`, CGO off, ldflags version
  injection, archives, checksums; Homebrew tap `dantte-lp/homebrew-tap`; Scoop bucket;
  winget manifest.
- [ ] `goreleaser release --snapshot --clean` succeeds locally.
- [ ] Commit: `chore: goreleaser multi-platform release config`.

### Task 6.2: GitHub Actions
**Files:** Create `.github/workflows/ci.yml`, `.github/workflows/release.yml`

- [ ] CI: lint + `go test -race -cover` + `govulncheck` on push/PR (matrix os).
- [ ] Release: on tag `v*`, run goreleaser with `GITHUB_TOKEN` + tap token.
- [ ] Commit: `ci: lint, test, vuln and release workflows`.

### Task 6.3: Docs
**Files:** Create `README.md`, `docs/cli.md`, `LICENSE`

- [ ] README: install (brew/scoop/go install), usage matrix, CI/agent examples (`--json`,
  `--token`, `token_command`), profiles, exit codes table.
- [ ] MIT `LICENSE`.
- [ ] Commit: `docs: readme, cli reference and license`.

---

## Definition of done
- `task lint test vuln` all green; `-race` clean.
- Interop test proves web/openpgp.js ↔ ypcli round-trip.
- `goreleaser --snapshot` produces all 6 platform archives.
- `ypcli send`/`receive` work end-to-end against a local yopass (`docker run jhaals/yopass`
  + memcached), verified manually before tagging v0.1.0.
