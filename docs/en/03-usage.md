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
