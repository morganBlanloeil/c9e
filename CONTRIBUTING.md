# Contributing

## Tech stack

| Component | Technology | Purpose |
|-----------|-----------|---------|
| Language | **Go 1.26** | Compiled binary, no runtime dependencies |
| TUI framework | [bubbletea](https://github.com/charmbracelet/bubbletea) v1 | Model-View-Update architecture for terminal UI |
| Styling | [lipgloss](https://github.com/charmbracelet/lipgloss) v1 | Terminal styling with adaptive light/dark colors |
| Components | [bubbles](https://github.com/charmbracelet/bubbles) v1 | Reusable TUI components |
| TTY detection | [golang.org/x/term](https://pkg.go.dev/golang.org/x/term) | Auto-fallback to table mode when piped |
| Tooling | [mise](https://mise.jdx.dev/) | Task runner, Go version pinning |

## Project structure

```
c9e/
├── cmd/c9e/
│   └── main.go              # CLI entry point, flags, modes
├── internal/
│   ├── cost/
│   │   └── cost.go          # Per-session cost estimation from conversation logs
│   ├── display/
│   │   └── display.go       # Static table + JSON rendering (--table, --json)
│   ├── history/
│   │   └── history.go       # Reads ~/.claude/history.jsonl
│   ├── logs/
│   │   └── logs.go          # Reads session JSONL conversation logs (log tail, turns, roles)
│   ├── notify/
│   │   └── notify.go        # Desktop notifications (macOS/Linux)
│   ├── process/
│   │   └── process.go       # Lists Claude processes via ps, kill support
│   ├── session/
│   │   └── session.go       # Reads ~/.claude/sessions/*.json
│   ├── terminal/
│   │   └── terminal.go      # Jump-to-terminal (tmux, iTerm2, Ghostty, Terminal.app)
│   └── tui/
│       ├── model.go          # Bubbletea model, key handling, state
│       ├── views.go          # List view, detail view, log tail rendering
│       ├── styles.go         # Lipgloss styles (adaptive light/dark)
│       └── data.go           # Data fetching (aggregates session/history/process/logs/cost)
├── mise.toml                 # Build tasks and Go version
├── go.mod
└── .gitignore
```

### Architecture

The project follows a clear separation:

- **`internal/session`**, **`internal/history`**, **`internal/process`**, **`internal/logs`**, **`internal/cost`** — data layer, each reads from one source
- **`internal/terminal`** — jump-to-terminal support (tmux, iTerm2, Ghostty, Terminal.app)
- **`internal/notify`** — desktop notifications when sessions complete
- **`internal/display`** — static output rendering (table, JSON)
- **`internal/tui`** — interactive TUI using bubbletea's Model-View-Update pattern (list, detail, and log tail views)
- **`cmd/c9e`** — CLI entry point, flag parsing, mode selection

Data flows: `session + history + process + logs + cost` -> `display.Row` -> `tui` or `display`

## Development commands

All commands use mise:

```bash
mise run build      # Compile to dist/c9e
mise run install    # Build + copy to ~/.claude/bin/
mise run test       # Run go test ./...
mise run dev        # Run the dashboard locally with go run
mise run clean      # Remove dist/ and build artifacts
mise run uninstall  # Remove binary from ~/.claude/bin/
```

## Building

The build injects the version from `git describe` into the binary via ldflags:

```bash
go build -ldflags "-s -w -X main.version=$(git describe --tags --always --dirty)" \
  -o dist/c9e ./cmd/c9e/
```

## Conventions

### Go

- **No CLI framework** (no cobra/viper) — stdlib flag parsing keeps it simple
- **`internal/`** packages only — nothing is exported outside the module
- **`cmd/<binary>/main.go`** pattern for the entry point
- Errors are returned, not panicked

### Styles

All colors use `lipgloss.AdaptiveColor{Light: "...", Dark: "..."}` to support both light and dark terminal themes. Never use hardcoded ANSI color codes in the TUI package.

### Data sources

The dashboard reads Claude Code's local state files. These are undocumented and may change between Claude Code versions.

#### Sessions — `~/.claude/sessions/*.json`

Each running Claude Code instance creates a JSON file named `<pid>.json` in `~/.claude/sessions/`. The file is created at startup and removed when the session ends cleanly.

```json
{
  "pid": 43074,
  "sessionId": "a792857c-fcc5-4698-80ff-23497d4971b6",
  "cwd": "/Users/me/repos/my-project",
  "startedAt": 1773995830484
}
```

| Field | Type | Description |
|-------|------|-------------|
| `pid` | int | OS process ID, used to check if the process is still alive |
| `sessionId` | string (UUID) | Unique session identifier, used as join key with history |
| `cwd` | string | Working directory at session start |
| `startedAt` | int64 | Unix timestamp in milliseconds |

If a Claude Code process crashes or is killed, the session file remains — this is how we detect `DEAD` sessions (file exists but process is gone).

**Code:** `internal/session/session.go`

#### History — `~/.claude/history.jsonl`

Every user message sent to Claude Code is appended as a JSON line to `~/.claude/history.jsonl`. The dashboard reads the tail of this file (~512KB) to find the last action per session.

```json
{
  "display": "fix the login bug",
  "pastedContents": {},
  "timestamp": 1773947857584,
  "project": "/Users/me/repos/my-project",
  "sessionId": "a792857c-fcc5-4698-80ff-23497d4971b6"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `display` | string | The user's message text |
| `timestamp` | int64 | Unix timestamp in milliseconds |
| `project` | string | Project directory |
| `sessionId` | string (UUID) | Links to the session file |

The `idle` duration is calculated as `now - last_action_timestamp`. A session with no action for more than 5 minutes is marked `IDLE`.

**Code:** `internal/history/history.go`

#### Process stats — `ps aux`

Live CPU and memory usage is obtained by running `ps aux` and filtering lines that contain `claude` (excluding `Claude.app` desktop processes). The PID from `ps` is matched against session files to determine if a session is alive.

A process is identified as a Claude Code CLI instance if:

- The command line contains "claude" (case-insensitive)
- The command line does NOT contain "Claude.app" or "Claude Helper"

**Code:** `internal/process/process.go`

#### Conversation logs — `~/.claude/projects/{slug}/{sessionID}.jsonl`

Each session's full conversation is stored as a JSONL file under `~/.claude/projects/`. The directory slug is derived from the session's working directory (with `/` and `.` replaced by `-`). These logs contain user messages, assistant responses, tool use, and thinking blocks.

The dashboard uses conversation logs for:

- **Turn count** — number of user messages in the session
- **Cost estimation** — token usage data is extracted to estimate per-session cost
- **WAITING status** — if the last log entry's role is "assistant", the session is marked as waiting for user input
- **Log tail view** — streams the conversation log with follow mode and thinking toggle

**Code:** `internal/logs/logs.go`, `internal/cost/cost.go`

#### How it all connects

```
sessions/*.json ──(pid)──→ ps aux        → alive? cpu? mem?
                 ──(sessionId)──→ history.jsonl → last action? idle time?
                 ──(sessionId+cwd)──→ projects/{slug}/{sessionID}.jsonl → turns, cost, tokens, log tail
```

The `sessionId` is the join key between session files, history, and conversation logs. The `pid` is used to match against running processes.

### Adding a new column

1. Add the field to `display.Row` in `internal/display/display.go`
2. Populate it in `internal/tui/data.go` (`fetchRows`)
3. Render it in `internal/tui/views.go` (`renderRow`) and `internal/display/display.go` (`RenderTable`)
4. Update the help text in `cmd/c9e/main.go`

### Adding a new TUI action

1. Define the key binding in `internal/tui/model.go` (`handleKey`, under `viewList` or `viewDetail`)
2. If the action is destructive, use the `confirmAction` pattern (see kill implementation)
3. Update the help bar in `internal/tui/views.go`
4. Update the help text in `cmd/c9e/main.go`

## Releasing

Releases are automated via [release-please](https://github.com/googleapis/release-please). On merge to `main`:

1. Release-please creates/updates a release PR with changelog based on conventional commits
2. Merging the release PR creates a GitHub Release with tag
3. The CI automatically builds binaries for all platforms and attaches them to the release

Binaries are available at: <https://github.com/morganBlanloeil/c9e/releases>
