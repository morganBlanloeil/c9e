# c9e

A terminal dashboard for monitoring running Claude Code instances — like [k9s](https://k9scli.io/) but for Claude Code.

![Version](https://img.shields.io/github/v/release/morganBlanloeil/c9e?label=version)
![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)
![License](https://img.shields.io/badge/license-MIT-blue)

## Features

- **Interactive TUI** — navigate sessions with keyboard shortcuts (j/k, enter, /)
- **Live refresh** — auto-updates every 3 seconds
- **Session detail** — drill into any session to see full metadata, token counts, and cost
- **Log tail** — stream a session's conversation log in real time (follow mode, thinking toggle)
- **Kill sessions** — terminate idle or stuck sessions with confirmation
- **Jump to terminal** — switch focus to a session's Ghostty terminal tab (`o` key, Ghostty only)
- **Done-row highlight** — when a session finishes, the row gets a golden highlight for 30 seconds
- **Desktop notifications** — get notified when a session completes (toggle with `n`)
- **Column sorting** — cycle sort column (`s`) and toggle direction (`S`)
- **Cost tracking** — per-session cost estimates with color coding (green/yellow/red)
- **Copy CWD** — copy a session's working directory to clipboard (`c`)
- **Filter** — search by directory, status, or last action
- **Column sorting** — sort by any column with ascending/descending toggle
- **Log tail view** — stream conversation logs with follow mode and thinking toggle
- **Cost estimation** — per-session USD cost based on token usage (color-coded)
- **Turn counter** — track conversation turns per session
- **Desktop notifications** — get notified when a session completes (macOS)
- **Copy to clipboard** — quickly copy a session's working directory
- **Aggregate stats** — total CPU, memory, and session counts in the footer
- **Adaptive colors** — works in both light and dark terminal themes
- **Multiple output modes** — TUI (default), static table, or JSON

## Installation

Download the latest binary for your platform from [GitHub Releases](https://github.com/morganBlanloeil/c9e/releases/latest) and install it:

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

> **Current version:** <!-- x-release-please-version -->1.3.1<!-- x-release-please-version-end --> — see [changelog](CHANGELOG.md) for details.

<details>
<summary>Alternative: install with mise</summary>

Install globally with [mise](https://mise.jdx.dev/) (downloads the binary from GitHub Releases):

```bash
mise use -g ubi:morganBlanloeil/c9e
```

The binary is automatically added to your PATH by mise.

</details>

<details>
<summary>Alternative: install with go</summary>

```bash
go install github.com/morganBlanloeil/c9e/cmd/c9e@latest
```

</details>

<details>
<summary>Alternative: build from source</summary>

Requires [mise](https://mise.jdx.dev/) and Go 1.26+:

```bash
git clone https://github.com/morganBlanloeil/c9e.git
cd c9e
mise run install
```

This builds the binary and installs it to `~/.claude/bin/c9e`.

</details>

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

### List view

| Key | Action |
|-----|--------|
| `j` / `k` or `↑` / `↓` | Navigate sessions |
| `enter` | Drill into session detail |
| `esc` / `q` | Clear filter / quit |
| `d` | Kill selected session (with confirmation) |
| `l` | Open log tail for selected session |
| `o` | Jump to session's Ghostty terminal tab |
| `/` | Filter by directory, status, or action |
| `s` / `S` | Cycle sort column / toggle sort direction |
| `c` | Copy selected session's working directory to clipboard |
| `n` | Toggle desktop notifications |
| `g` / `G` | Jump to first / last |
| `ctrl+c` | Force quit |

### Log tail view

| Key | Action |
|-----|--------|
| `j` / `k` or `↑` / `↓` | Scroll up / down |
| `g` / `G` | Jump to top / bottom |
| `f` | Toggle follow mode (auto-scroll like `tail -f`) |
| `t` | Toggle thinking blocks (show/hide Claude's internal reasoning) |
| `esc` / `q` | Back to previous view |

## Columns

| Column | Description |
|--------|-------------|
| PID | Process ID of the Claude Code instance |
| STATUS | Session status (see below) |
| TURNS | Number of conversation turns (user messages) |
| CPU% | Current CPU usage |
| MEM% | Current memory usage |
| COST | Estimated session cost in USD (color-coded: green < $0.10, yellow < $1, red >= $1) |
| UPTIME | Time since the instance was started |
| IDLE | Time since the last user message |
| DIRECTORY | Working directory of the instance |
| LAST ACTION | Last user message sent in the session |

## Status types

| Status | Icon | Meaning |
|--------|------|---------|
| ACTIVE | `●` | Interaction within the last 5 minutes |
| WAITING | `◇` | Claude has responded and is awaiting user input, or session has active agent subprocesses |
| IDLE | `◐` | No interaction for more than 5 minutes |
| DEAD | `○` | Session file exists but process is gone |
| DONE | `★` | Task recently completed (30-second highlight) |

## Data sources

The dashboard reads from Claude Code's local state files:

| Source | Data |
|--------|------|
| `~/.claude/sessions/*.json` | Session metadata (PID, working directory, start time) |
| `~/.claude/history.jsonl` | User action log (last message per session) |
| `~/.claude/projects/{slug}/{sessionID}.jsonl` | Full conversation logs (turns, cost, token usage, log tail) |
| `ps aux` | Live process stats (CPU, memory, alive check) |

> **Note:** These are undocumented Claude Code internal files and may change between versions.

## Development

See [CONTRIBUTING.md](CONTRIBUTING.md) for details on the tech stack and how to contribute.

```bash
mise run build     # Build to dist/
mise run test      # Run tests
mise run install   # Build + install to ~/.claude/bin
mise run dev       # Run locally with go run
mise run clean     # Remove build artifacts
```

## License

MIT
