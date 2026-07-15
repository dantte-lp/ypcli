# CLI Reference

## Global flags

Available on every command. Resolution precedence is
**flag > env (`YPCLI_*`) > active profile > built-in default**.

| Flag | Env | Description |
|---|---|---|
| `--profile, -p` | `YPCLI_PROFILE` | configuration profile to use |
| `--api` | `YPCLI_API` | yopass API base URL |
| `--url` | `YPCLI_URL` | yopass public URL (for share links) |
| `--token` | `YPCLI_TOKEN` | bearer token for authenticated instances |
| `--timeout` | `YPCLI_TIMEOUT` | request timeout (default `30s`) |
| `--json` | `YPCLI_JSON` | machine-readable JSON output |
| `--verbose, -v` | `YPCLI_VERBOSE` | debug logging to stderr |
| `--config` | `YPCLI_CONFIG` | config file path |

## `ypcli send`

Encrypt and publish a secret. Input comes from `--file`, `--text`, or piped stdin.

| Flag | Description |
|---|---|
| `--file, -f` | read the secret from a file (published as a file secret) |
| `--text, -t` | secret text (instead of stdin/file) |
| `--expiration, -e` | lifetime: `1h`, `1d`, or `1w` (default `1h`) |
| `--one-time` | delete after first view (default `true`) |
| `--require-auth` | require authentication to view (server support required) |
| `--key, -k` | manual encryption key; omitted from the URL |
| `--qr` | also render the URL as a terminal QR code (text mode) |
| `--copy` | copy the URL to the system clipboard |

JSON output:

```json
{"id":"…","url":"https://…/#/s/…/…","key":"…","manual_key":false,"file":false,"one_time":true,"expiration":"1h"}
```

## `ypcli receive`

Fetch and decrypt a secret. Accepts a share URL positional argument, or
`--id`/`--key`.

| Flag | Description |
|---|---|
| `--id` | secret ID (when no URL is given) |
| `--key, -k` | decryption key (required for manual-key links and `--id`) |
| `--file` | treat the secret as a file (with `--id`) |
| `--output, -o` | output file or directory for file secrets |

- Text secrets are written to **stdout**.
- File secrets are written to their original name, or under `-o`.

## `ypcli config`

Manage named server profiles in `$XDG_CONFIG_HOME/ypcli/config.yaml` (mode 0600).

```bash
ypcli config add work --api https://api.corp --url https://yp.corp \
  --expiration 1d --token-command 'vault read -field=token secret/yopass'
ypcli config list      # * marks the active profile
ypcli config use work
ypcli config remove work
```

| Subcommand | Flags |
|---|---|
| `add <name>` | `--api`, `--url`, `--expiration`, `--token-command` |
| `list` | — |
| `use <name>` | — |
| `remove <name>` | — |

## `ypcli version`

Prints the client build (version/commit/date) and queries the server `/version`
endpoint. Servers older than yopass 13.x report `unsupported`.

```bash
ypcli version --api https://api.yopass.se --json
```

## `ypcli completion`

Generate a shell completion script for `bash`, `zsh`, `fish`, or `powershell`.

```bash
ypcli completion zsh > "${fpath[1]}/_ypcli"
```

## Exit codes

| Code | Meaning |
|---|---|
| 0 | success |
| 1 | generic error |
| 2 | usage / bad flags |
| 3 | configuration error |
| 4 | network / timeout |
| 5 | auth failure (401/403) |
| 6 | not found / one-time consumed (404/410) |
| 7 | decryption / crypto failure |
