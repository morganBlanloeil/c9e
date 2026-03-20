# c9e

A terminal dashboard for monitoring running Claude Code instances — like [k9s](https://k9scli.io/) but for Claude Code.

![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)
![License](https://img.shields.io/badge/license-MIT-blue)

## Features

- **Interactive TUI** — navigate sessions with keyboard shortcuts (j/k, enter, /)
- **Live refresh** — auto-updates every 3 seconds
- **Session detail** — drill into any session to see full metadata
- **Kill sessions** — terminate idle or stuck sessions with confirmation
- **Filter** — search by directory, status, or last action
- **Adaptive colors** — works in both light and dark terminal themes
- **Multiple output modes** — TUI (default), static table, or JSON

## Installation

### Download prebuilt binary (recommended)

Download the latest release for your platform from [GitHub Releases](https://github.com/morganBlanloeil/c9e/releases/latest):

```bash
# macOS (Apple Silicon)
curl -Lo c9e https://github.com/morganBlanloeil/c9e/releases/latest/download/c9e-darwin-arm64
# macOS (Intel)
curl -Lo c9e https://github.com/morganBlanloeil/c9e/releases/latest/download/c9e-darwin-amd64
# Linux (x86_64)
curl -Lo c9e https://github.com/morganBlanloeil/c9e/releases/latest/download/c9e-linux-amd64
# Linux (ARM64)
curl -Lo c9e https://github.com/morganBlanloeil/c9e/releases/latest/download/c9e-linux-arm64

chmod +x c9e
mv c9e ~/.claude/bin/
```

Make sure `~/.claude/bin` is in your PATH:

```bash
# Add to your .zshrc or .bashrc
export PATH="$HOME/.claude/bin:$PATH"
```

### From source (mise)

```bash
git clone https://github.com/morganBlanloeil/c9e.git
cd c9e
mise run install
```

The binary is installed to `~/.claude/bin/c9e`.

### From source (go install)

```bash
go install github.com/morganBlanloeil/c9e/cmd/c9e@latest
```

## Usage

```bash
# Interactive TUI (default)
c9e

# Static table output (one-shot)
c9e --table

# JSON output (for scripting)
c9e --json

# Pipe to jq — find sessions idle for 10+ minutes
c9e --json | jq '.[] | select(.alive and .idle_s > 600)'
```

If stdout is not a TTY (e.g., piped), the dashboard automatically falls back to table mode.

## TUI keyboard shortcuts

| Key | Action |
|-----|--------|
| `j` / `k` or `↑` / `↓` | Navigate sessions |
| `enter` | Drill into session detail |
| `esc` / `q` | Back / quit |
| `d` | Kill selected session (with confirmation) |
| `/` | Filter by directory, status, or action |
| `g` / `G` | Jump to first / last |
| `ctrl+c` | Force quit |

## Columns

| Column | Description |
|--------|-------------|
| PID | Process ID of the Claude Code instance |
| STATUS | `ACTIVE` (< 5min idle), `IDLE` (> 5min), `DEAD` (process gone) |
| CPU% | Current CPU usage |
| MEM% | Current memory usage |
| UPTIME | Time since the instance was started |
| IDLE | Time since the last user message |
| DIRECTORY | Working directory of the instance |
| LAST ACTION | Last user message sent in the session |

## Data sources

The dashboard reads from Claude Code's local state files:

| Source | Data |
|--------|------|
| `~/.claude/sessions/*.json` | Session metadata (PID, working directory, start time) |
| `~/.claude/history.jsonl` | User action log (last message per session) |
| `ps aux` | Live process stats (CPU, memory, alive check) |

## Development

See [CONTRIBUTING.md](CONTRIBUTING.md) for details on the tech stack and how to contribute.

```bash
mise run build     # Build to dist/
mise run test      # Run tests
mise run install   # Build + install to ~/.claude/bin
mise run clean     # Remove build artifacts
```

## License

MIT
