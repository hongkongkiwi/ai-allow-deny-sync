# Allow/Deny Sync Service

Keep command and MCP allow/deny lists in sync across AI coding tools. This service reads each tool’s config, merges policies, and (optionally) writes them back. Use dry‑runs for safety, and split command vs MCP policies to avoid mixing incompatible formats.

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
- `json-bool-map`: read/write a map of `command -> true|false` at `allow_key` (true = allow, false = deny).
- `codex-rules`: read/write Codex `prefix_rule(...)` lines from `~/.codex/rules/*.rules` (managed rules only).

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

Dry run (no writes):

```bash
go run ./cmd/syncd -once -dry-run
```

Validate config (no reads/writes to missing_ok paths):

```bash
go run ./cmd/syncd -validate
```

## Two-list mode (recommended)

Run separate configs for command allow/deny vs MCP allow/deny:

```bash
go run ./cmd/syncd -config syncd.commands.yaml -once -dry-run
go run ./cmd/syncd -config syncd.mcp.yaml -once -dry-run
```

### Taskfile (optional)

If you use Task, `Taskfile.yml` includes shortcuts:

```bash
task test
task build
task validate:commands
task validate:mcp
task dryrun:commands
task dryrun:mcp
task release:dry-run
```

## Example config

See `syncd.yaml.example`.

Paths beginning with `~` are expanded to your home directory.

## Adding a new tool format

If a tool stores allow/deny lists in a different format, add a format implementation in `internal/format/format.go` and reference it in your `syncd.yaml`.

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
- Codex CLI: `~/.codex/rules/default.rules`, `prefix_rule(... decision="allow"|"forbidden")`
- VS Code Copilot: `~/Library/Application Support/Code/User/settings.json`, key `chat.tools.terminal.autoApprove` (true/false map)

Tools like Codex, Roo, Cline, DeepSeek CLI, and Qwen CLI may not expose command allow/deny lists in a compatible way. If you can share where they store their permission rules (and their exact JSON/TOML shape), I can add adapters.

## Build

```bash
go build -o bin/syncd ./cmd/syncd
```

## Security notes

- These lists are **policy hints**, not hard security boundaries.
- Some tools apply allow/deny as a user‑experience layer and can be bypassed in certain modes.
- Prefer OS‑level sandboxing for strong isolation.

## Safety checklist

- Start with `-dry-run` to verify merged counts.
- Keep command and MCP policies in separate configs.
- Use `authoritative` mode if one tool should be the source of truth.
- Prefer staging lists (e.g., `/tmp`) when first configuring a new tool.

## Troubleshooting

- **Config errors**: Run `-validate` to check paths and basic schema requirements.
- **Nothing changes**: Ensure you’re using the right config file and not in `-dry-run`.
- **Unexpected list contents**: Confirm you’re not mixing MCP policies into command lists.

## FAQ

**Which tools map to command allow/deny?**  
Claude, Cursor CLI, Codex rules, Roo/Cline `allowedCommands`, and VS Code Copilot auto‑approve map to command allow/deny.

**Which tools map to MCP allow/deny?**  
Qwen (`mcp.allowed`/`mcp.excluded`) and Gemini (`allowMCPServers`/`excludeMCPServers`) are MCP‑level policies.

**Why split configs?**  
Some tools treat MCP allow/deny separately from command allow/deny, and merging them can produce unexpected behavior.

## Release

Tag `v*` to trigger a GitHub Release with attached binaries:

```bash
git tag v0.1.0
git push origin v0.1.0
```

## Next steps

If you can share the exact file paths and formats for each tool, I can wire those adapters directly so you don't have to maintain custom formats.
