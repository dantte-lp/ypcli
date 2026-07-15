---
name: ypcli
description: Share passwords, secrets, API keys, tokens, or files securely as end-to-end-encrypted one-time links via yopass, and fetch/decrypt yopass share URLs. Use whenever the user wants to send or share a secret/password/credential/token/file safely, deliver something sensitive without pasting it in plaintext, or open a yopass link. Backed by the ypcli MCP tools (send_secret, send_file, receive_secret) with a ypcli CLI fallback.
---

# Sharing secrets with ypcli (yopass)

ypcli publishes secrets to a [yopass](https://github.com/jhaals/yopass) server
with **client-side** OpenPGP encryption; each secret becomes a one-time URL that
expires. Prefer the MCP tools when the `ypcli` MCP server is connected; otherwise
use the `ypcli` CLI.

## Share a text secret

- MCP: call `send_secret` with `text` (optional: `expiration` = `1h`/`1d`/`1w`,
  `one_time`, `require_auth`, `profile`). It returns a one-time `url` — give that
  URL to the recipient. The decryption key is embedded in the URL fragment.
- CLI: `printf '%s' "$SECRET" | ypcli send --json` → take `.url`.

## Share a file

- MCP: `send_file` with the absolute `path`.
- CLI: `ypcli send --file <path> --json`.

## Receive / decrypt

- MCP: `receive_secret` with `url` (or `id` + `key`). Returns the decrypted
  `content` (binary payloads come back as `content_base64`).
- CLI: `ypcli receive '<url>'`.

> One-time secrets are **consumed (deleted) on the first successful fetch** —
> only receive when you intend to reveal and destroy the secret.

## Guidance

- Never paste the plaintext secret into the conversation; share only the URL.
- Keep `one_time` on (default) and pick the shortest workable `expiration`.
- For a private/self-hosted yopass, pass a `profile` (see `list_profiles`).

## Install

Copy this folder to `~/.claude/skills/ypcli/`, and connect the MCP server:
`claude mcp add ypcli -- ypcli mcp`. See the repo's `docs/en/09-mcp.md`.
