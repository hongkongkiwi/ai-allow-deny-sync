# Allow/Deny Sync Service

A small Go service that keeps allow/deny command lists in sync across AI coding tools by reading each tool's files, merging them, and writing the merged result back.

## How it works

- Each client has an allow list and a deny list stored on disk.
- The service loads all clients, merges lists, normalizes them, and writes them back.
- Two modes:
  - `union`: merge all allow/deny entries from every client.
  - `authoritative`: pick a single source client and sync its lists to all others.

## Supported formats (built-in)

- `newline`: one entry per line, `#` comments allowed.
- `json`: a JSON array of strings.
- `json-object`: read/write lists inside a JSON document using `allow_key`/`deny_key` dot-paths.
- `codex-rules`: read/write Codex `prefix_rule(...)` lines from `~/.codex/rules/*.rules`.
- `json-bool-map`: read/write a map of `command -> true|false` at `allow_key` (true = allow, false = deny).

## Quick start

1. Copy the example config and update paths:

```bash
cp syncd.yaml.example syncd.yaml
```

2. Run once:

```bash
go run ./cmd/syncd -once
```

3. Run as a service:

```bash
go run ./cmd/syncd -interval 30s
```

## Example config

See `syncd.yaml.example`.

Paths beginning with `~` are expanded to your home directory.

## Adding a new tool format

If a tool stores allow/deny lists in a different format, add a format implementation in `internal/format/format.go` and reference it in `syncd.yaml`.

## Tools to include

The service is format-based, so you can add any tool that stores allow/deny lists on disk. Common targets:

- Claude Code
- Codex
- Kilo Code
- Roo
- Cline
- Cursor
- Aider
- Continue
- Zed extensions

## Known locations and keys

These are known defaults from docs; adjust for your setup and OS:

- Claude Code: `~/.claude/settings.json`, keys `permissions.allow` / `permissions.deny`
- Cursor CLI: `~/.cursor/cli-config.json`, keys `permissions.allow` / `permissions.deny`
- Roo/Cline (Cursor): `~/Library/Application Support/Cursor/User/settings.json`, key `roo-cline.allowedCommands`
- Kilo Code CLI: `~/.kilocode/config.json`, keys `autoApproval.execute.allowed` / `autoApproval.execute.denied`
- Gemini CLI: `~/.gemini/settings.json`, keys `coreTools` / `excludeTools`
- Qwen Code: `~/.qwen/settings.json`, keys `mcp.allowed` / `mcp.excluded` (MCP server allow/deny, not command permissions)
- Codex CLI: `~/.codex/rules/default.rules`, `prefix_rule(... decision=\"allow\"|\"forbidden\")`
- VS Code Copilot: `~/Library/Application Support/Code/User/settings.json`, key `chat.tools.terminal.autoApprove` (true/false map)

Tools like Codex, Roo, Cline, DeepSeek CLI, and Qwen CLI may not expose command allow/deny lists in a compatible way. If you can share where they store their permission rules (and their exact JSON/TOML shape), I can add adapters.

## Next steps

If you can share the exact file paths and formats for each tool, I can wire those adapters directly so you don't have to maintain custom formats.
