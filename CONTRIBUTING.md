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
claude-dashboard/
├── cmd/claude-dashboard/
│   └── main.go              # CLI entry point, flags, modes
├── internal/
│   ├── display/
│   │   └── display.go       # Static table + JSON rendering (--table, --json)
│   ├── history/
│   │   └── history.go       # Reads ~/.claude/history.jsonl
│   ├── process/
│   │   └── process.go       # Lists Claude processes via ps, kill support
│   ├── session/
│   │   └── session.go       # Reads ~/.claude/sessions/*.json
│   └── tui/
│       ├── model.go          # Bubbletea model, key handling, state
│       ├── views.go          # List view + detail view rendering
│       ├── styles.go         # Lipgloss styles (adaptive light/dark)
│       └── data.go           # Data fetching (aggregates session/history/process)
├── mise.toml                 # Build tasks and Go version
├── go.mod
└── .gitignore
```

### Architecture

The project follows a clear separation:

- **`internal/session`**, **`internal/history`**, **`internal/process`** — data layer, each reads from one source
- **`internal/display`** — static output rendering (table, JSON)
- **`internal/tui`** — interactive TUI using bubbletea's Model-View-Update pattern
- **`cmd/claude-dashboard`** — CLI entry point, flag parsing, mode selection

Data flows: `session + history + process` -> `display.Row` -> `tui` or `display`

## Development commands

All commands use mise:

```bash
mise run build      # Compile to dist/claude-dashboard
mise run install    # Build + copy to ~/.claude/bin/
mise run test       # Run go test ./...
mise run clean      # Remove dist/ and build artifacts
mise run uninstall  # Remove binary from ~/.claude/bin/
```

## Building

The build injects the version from `git describe` into the binary via ldflags:

```bash
go build -ldflags "-s -w -X main.version=$(git describe --tags --always --dirty)" \
  -o dist/claude-dashboard ./cmd/claude-dashboard/
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

The dashboard reads Claude Code's local state files. These are undocumented and may change between Claude Code versions. Key locations:

- `~/.claude/sessions/*.json` — one file per active session
- `~/.claude/history.jsonl` — append-only log of user messages
- Process detection via `ps aux` filtering for `claude` CLI processes

### Adding a new column

1. Add the field to `display.Row` in `internal/display/display.go`
2. Populate it in `internal/tui/data.go` (`fetchRows`)
3. Render it in `internal/tui/views.go` (`renderRow`) and `internal/display/display.go` (`RenderTable`)
4. Update the help text in `cmd/claude-dashboard/main.go`

### Adding a new TUI action

1. Define the key binding in `internal/tui/model.go` (`handleKey`, under `viewList` or `viewDetail`)
2. If the action is destructive, use the `confirmAction` pattern (see kill implementation)
3. Update the help bar in `internal/tui/views.go`
4. Update the help text in `cmd/claude-dashboard/main.go`

## Releasing

Tag a version and rebuild:

```bash
git tag v0.1.0
mise run install
claude-dashboard --version
# claude-dashboard v0.1.0
```
