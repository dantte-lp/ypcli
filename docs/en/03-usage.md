# Usage

## Sending secrets

Text from stdin:

```bash
printf 'my secret' | ypcli send
```

Text from a flag:

```bash
ypcli send --text 'my secret'
```

A file (published as a file secret; the filename is embedded in the encrypted
payload, never in cleartext):

```bash
ypcli send --file ./db.env --expiration 1d
```

### Options

```bash
# Multi-view secret, valid for a week
cat notes.md | ypcli send --expiration 1w --one-time=false

# Show a scannable QR code and copy the URL to the clipboard
printf 'wifi-password' | ypcli send --qr --copy

# Manual key: the key is omitted from the URL and printed separately,
# so you can deliver it out of band
ypcli send --file secret.pem --key "$(openssl rand -hex 16)"
```

### Compose in an editor

Run `send` interactively (no `--file`/`--text`/stdin) and ypcli opens your
editor; the secret is sent when you save and quit. The editor is
`$YPCLI_EDITOR`, `$VISUAL`, or `$EDITOR` (falling back to `vi`, or `notepad` on
Windows). `--editor` forces it even with piped input.

```bash
ypcli send                 # opens the editor
ypcli send --editor --expiration 1d
```

### From a secrets manager (Vault / OpenBao)

Read the payload straight from a Vault or OpenBao KV v2 engine — nothing touches
your shell history or the filesystem. Standard `VAULT_*` / `BAO_*` environment
variables are honored.

```bash
export VAULT_ADDR=https://vault.corp VAULT_TOKEN=…
ypcli send --vault-path db --vault-field password
```

For the bearer token that authenticates to a private *yopass* server, use
`token_command` instead — see [Configuration](05-configuration.md#tokens).

## Receiving secrets

From a share URL:

```bash
ypcli receive 'https://yopass.se/#/s/ID/KEY'
```

A manual-key link requires `--key`:

```bash
ypcli receive 'https://yopass.se/#/c/ID' --key MANUALKEY
```

Without a URL, use `--id`/`--key` (add `--file` for file secrets):

```bash
ypcli receive --id ID --key KEY --file -o ./downloads/
```

### File output

- With no `--output`, a file secret is written to its embedded filename in the
  current directory.
- `--output DIR/` (a directory, existing or with a trailing separator) writes
  under the embedded filename, creating the directory if needed.
- `--output PATH` writes to that exact path.

## Global flags

All commands accept the global flags described in
[CLI Reference](04-cli.md) and [Configuration](05-configuration.md), including
`--profile`, `--api`, `--url`, `--token`, `--timeout`, `--json`, and
`--verbose`.
