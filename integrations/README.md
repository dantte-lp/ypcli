# Agent integrations

Ready-to-copy configuration connecting AI agents to the `ypcli mcp` server. The
MCP server is universal — the same `ypcli mcp` binary works for every client.

| Client | File | How |
|---|---|---|
| Claude Code | [`claude/.mcp.json`](claude/.mcp.json) | copy to your project root, or `claude mcp add ypcli -- ypcli mcp` |
| Claude (skill) | [`../skills/ypcli/`](../skills/ypcli) | copy to `~/.claude/skills/ypcli/` |
| Codex | [`codex/config.toml`](codex/config.toml) | merge into `~/.codex/config.toml`; prompt in [`codex/prompts/`](codex/prompts) |
| Gemini CLI | [`gemini/settings.json`](gemini/settings.json) | merge into `~/.gemini/settings.json`, or install the extension in [`gemini/`](gemini) |

Each config has a **stdio** variant (the agent launches `ypcli mcp` locally) and,
where the client supports it, an **HTTP** variant pointing at a shared
`ypcli mcp --http` server with a bearer token. See
[`docs/en/09-mcp.md`](../docs/en/09-mcp.md) and [`deploy/`](../deploy).

Before connecting, install ypcli and configure a profile:

```bash
go install github.com/dantte-lp/ypcli/cmd/ypcli@latest
ypcli config add work --api https://api.yopass.corp --url https://yopass.corp
```
