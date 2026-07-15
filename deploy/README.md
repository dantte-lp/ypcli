# Deploying the ypcli MCP server

Two ways to expose ypcli to AI agents (Claude, Codex, Gemini):

- **stdio** (local) — the agent launches `ypcli mcp` as a subprocess. Nothing to
  deploy beyond installing the binary and a profile. See
  [docs/en/09-mcp.md](../docs/en/09-mcp.md).
- **HTTP** (shared server) — run `ypcli mcp --http` as a service that agents
  connect to over the network with a bearer token. That is what this directory
  covers.

## Install the binary

```bash
go install github.com/dantte-lp/ypcli/cmd/ypcli@latest
sudo install "$(go env GOPATH)/bin/ypcli" /usr/local/bin/ypcli   # or a release binary
```

## Configure

```bash
sudo mkdir -p /etc/ypcli

# 1) Profile config — no plaintext secrets; use token_command for yopass auth.
sudo tee /etc/ypcli/config.yaml >/dev/null <<'YAML'
defaults:
  api: https://api.yopass.corp
  url: https://yopass.corp
  # token_command: vault read -field=token secret/yopass   # if the server needs auth
YAML
sudo chmod 0644 /etc/ypcli/config.yaml

# 2) Bearer token for the HTTP endpoint (root-only).
printf 'YPCLI_MCP_TOKEN=%s\n' "$(openssl rand -hex 32)" | sudo tee /etc/ypcli/mcp.env >/dev/null
sudo chmod 0600 /etc/ypcli/mcp.env
```

## Run as a service

```bash
sudo cp deploy/systemd/ypcli-mcp.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now ypcli-mcp
systemctl status ypcli-mcp
```

The unit runs under `DynamicUser` with a strict sandbox (`ProtectSystem=strict`,
`NoNewPrivileges`, no capabilities, filtered syscalls) and binds to
`127.0.0.1:8765` by default.

## TLS / exposure

The server speaks plain HTTP and binds to loopback. Put it behind a
TLS-terminating reverse proxy (nginx, Caddy, Traefik) if agents connect from
other hosts, and keep the bearer token secret. Example Caddy:

```caddy
mcp.yopass.corp {
    reverse_proxy 127.0.0.1:8765
}
```

## Connect an agent

Point the client at the URL with the bearer token — see
[docs/en/09-mcp.md](../docs/en/09-mcp.md#http-shared-server) and the ready-made
snippets in [`integrations/`](../integrations).

```bash
# Claude Code
claude mcp add --transport http ypcli https://mcp.yopass.corp \
  --header "Authorization: Bearer $YPCLI_MCP_TOKEN"
```
