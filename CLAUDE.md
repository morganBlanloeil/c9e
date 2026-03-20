# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project overview

Terminal dashboard for monitoring running Claude Code instances — like k9s but for Claude Code. Built as a Go TUI app using the Charmbracelet stack (bubbletea, lipgloss, bubbles).

## Build & test commands

All tasks use [mise](https://mise.jdx.dev/):

```bash
mise run build      # Compile to dist/claude-dashboard (injects version via ldflags)
mise run test       # Run go test ./...
mise run install    # Build + copy to ~/.claude/bin/
mise run clean      # Remove dist/ and build artifacts
```

Run a single test: `go test ./internal/session/ -run TestName`

## Architecture

Data flows: `session + history + process` → `display.Row` → `tui` or `display`

- **Data layer** (`internal/session`, `internal/history`, `internal/process`) — each package reads from one source: session JSON files, history JSONL, or `ps aux`
- **Static output** (`internal/display`) — renders `display.Row` as table or JSON for `--table`/`--json` modes
- **Interactive TUI** (`internal/tui`) — bubbletea Model-View-Update pattern with list and detail views
- **Entry point** (`cmd/claude-dashboard/main.go`) — flag parsing, mode selection, static mode row assembly

## Key conventions

- No CLI framework (no cobra/viper) — stdlib flag parsing only
- All packages are `internal/` — nothing exported outside the module
- All TUI colors must use `lipgloss.AdaptiveColor{Light: "...", Dark: "..."}` — never hardcoded ANSI codes
- Errors are returned, not panicked

## Extending the dashboard

**New column:** add field to `display.Row` → populate in `tui/data.go` (`fetchRows`) → render in `tui/views.go` and `display/display.go` (`RenderTable`)

**New TUI action:** define key in `tui/model.go` (`handleKey`) → use `confirmAction` pattern for destructive actions → update help bar in `tui/views.go`

## Data sources

Reads Claude Code's undocumented local state (may change between versions):
- `~/.claude/sessions/*.json` — session metadata
- `~/.claude/history.jsonl` — user message log
- `ps aux` — live process stats
