# ypcli — sharing secrets via yopass

Use the `ypcli` MCP tools to share and fetch end-to-end-encrypted one-time
secrets:

- `send_secret` — encrypt and publish text; returns a one-time share URL.
- `send_file` — publish a file by path.
- `receive_secret` — fetch and decrypt a share URL (or `id` + `key`). One-time
  secrets are consumed on first fetch.

Never print the plaintext secret back to the user — return only the resulting
URL. Keep `one_time` on and choose the shortest workable `expiration`
(`1h`/`1d`/`1w`).
