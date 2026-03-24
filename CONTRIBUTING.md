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
│   │   └── ghostty.go       # Jump-to-terminal (Ghostty only, via AppleScript)
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
- **`internal/terminal`** — jump-to-terminal support (Ghostty only, via AppleScript)
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

## Testing

### Running tests

```bash
mise run test                                  # Run all tests
go test ./internal/tui/ -v                     # Run TUI tests with verbose output
go test ./internal/tui/ -run TestGolden        # Run only golden file tests
go test ./internal/tui/ -run TestTeatest       # Run only teatest pipeline tests
go test ./internal/session/ -run TestName      # Run a single test by name
```

### Test architecture

The TUI test suite is organized in three layers, each catching different kinds of regressions:

```
┌──────────────────────────────────────────────────────────────────┐
│  Layer 3: Teatest pipeline tests                                │
│  Full bubbletea lifecycle with simulated Claude sessions         │
│  Files: views_test.go (TestTeatest_*)                           │
├──────────────────────────────────────────────────────────────────┤
│  Layer 2: Golden file tests (visual regression)                  │
│  Exact terminal output snapshots with ANSI codes preserved       │
│  Files: views_test.go (TestGolden_*), testdata/*.golden          │
├──────────────────────────────────────────────────────────────────┤
│  Layer 1: Model state machine tests                              │
│  Direct Update() calls, assert on model fields                   │
│  Files: model_test.go                                            │
├──────────────────────────────────────────────────────────────────┤
│  Foundation: Fixture helpers                                     │
│  Deterministic display.Row data + fake Claude home directory     │
│  Files: helpers_test.go                                          │
└──────────────────────────────────────────────────────────────────┘
```

### Layer 1: Model state machine tests

These tests drive the bubbletea model **synchronously** by calling `Update()` directly and asserting on the resulting model state. They don't require a terminal, file system, or running processes.

**How it works:**

1. Create a model pre-loaded with fixture data using `newTestModel(t)`
2. Send messages by calling `m.Update(msg)` — this returns the new model
3. Assert on model fields (`cursor`, `view`, `filter`, `confirm`, etc.)

```go
func TestNavigation_JDown(t *testing.T) {
    m := newTestModel(t)                           // 3 fixture rows, 120x40 terminal

    updated, _ := m.Update(tea.KeyMsg{             // simulate pressing 'j'
        Type: tea.KeyRunes, Runes: []rune{'j'},
    })
    m = updated.(Model)

    if m.cursor != 1 {                             // assert cursor moved down
        t.Errorf("cursor = %d, want 1", m.cursor)
    }
}
```

**What `newTestModel` does under the hood:**

```go
func newTestModel(t *testing.T) Model {
    m := NewModel("test")
    // 1. Set terminal dimensions (required before View() works)
    sized, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
    m = sized.(Model)
    // 2. Inject deterministic rows (bypasses file I/O entirely)
    updated, _ := m.Update(dataMsg{rows: fixtureRows(), err: nil})
    m = updated.(Model)
    return m
}
```

This gives you a model with 3 sessions (ACTIVE, IDLE, DEAD) using fixed values for PID, CPU, cost, etc. Since the data is injected via `dataMsg` rather than read from disk, the output is fully deterministic.

**What these tests cover:**

| Area | Tests |
|------|-------|
| Navigation | `j`/`k` movement, `g`/`G` jump, bounds checking |
| Filtering | `/` enters filter mode, typing filters, backspace, esc clears, cursor adjustment |
| Sorting | `s` cycles column, `S` toggles direction, row reordering |
| View transitions | enter → detail, esc/q → back, enter on empty list |
| Log view | `f` toggles follow, `t` toggles thinking, esc returns to correct view |
| Kill confirmation | `d` triggers confirm, `y` confirms, `n`/esc cancels, dead sessions skip |
| Data refresh | `tickMsg` triggers fetch, `dataMsg` updates rows |

### Layer 2: Golden file tests (visual regression)

Golden tests capture the **exact terminal output** of each view — including ANSI escape codes (colors, bold, positioning from lipgloss). Any visual change (column width, icon, color, layout) breaks the test.

**How it works:**

1. Set up the model in the desired state
2. Call `m.View()` to render the terminal output
3. Pass it to `golden.RequireEqual(t, output)` from Charmbracelet's [golden](https://github.com/charmbracelet/x/tree/main/exp/golden) package
4. The package compares the output against a reference file in `testdata/{TestName}.golden`

```go
func TestGolden_ListView(t *testing.T) {
    m := newTestModel(t)
    golden.RequireEqual(t, []byte(m.View()))   // compare against testdata/TestGolden_ListView.golden
}

func TestGolden_FilterActive(t *testing.T) {
    m := newTestModel(t)
    // Simulate typing "/alpha" to filter
    updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
    m = updated.(Model)
    for _, ch := range "alpha" {
        updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
        m = updated.(Model)
    }
    golden.RequireEqual(t, []byte(m.View()))   // compare against testdata/TestGolden_FilterActive.golden
}
```

**Golden files contain the full terminal rendering:**

```
   Claude Code Dashboard  test
────────────────────────────────────────────────────────────────
  ● 1 active  ◐ 1 idle  ○ 1 dead  Total: $1.92
────────────────────────────────────────────────────────────────
  PID     STATUS    TURNS   CPU%   MEM%      COST  UPTIME ...
────────────────────────────────────────────────────────────────
  ● 1001    ACTIVE        5   12.3    4.5     $0.42  1h 0m ...
  ◐ 2002    IDLE         12    0.0    1.2     $1.50  2h 0m ...
  ○ 3003    DEAD          3    0.0    0.0         —  15m   ...
```

The actual files also embed ANSI codes for colors and styles. This means a change like swapping `●` for `◆` or changing the green color of ACTIVE will show up as a diff.

**When a golden test fails**, you get a unified diff showing exactly what changed:

```
--- testdata/TestGolden_ListView.golden
+++ actual output
@@ -7,3 +7,3 @@
-  ● 1001    ACTIVE        5   12.3    4.5     $0.42  1h 0m
+  ◆ 1001    ACTIVE        5   12.3    4.5     $0.42  1h 0m
```

**Regenerating golden files** after an intentional visual change:

```bash
go test ./internal/tui/ -update
```

This overwrites all `testdata/*.golden` files with the current output. Review the diff before committing.

**Current golden file coverage:**

| Golden file | What it captures |
|---|---|
| `TestGolden_ListView` | Main list with 3 sessions, header, footer, stats |
| `TestGolden_DetailView` | Detail panel for a single session |
| `TestGolden_EmptyList` | Empty state (no sessions) |
| `TestGolden_FilterActive` | List with `/alpha` filter applied |
| `TestGolden_ConfirmKill` | Kill confirmation prompt in footer |
| `TestGolden_LogView` | Log tail view with follow/thinking status |
| `TestGolden_SortDescending` | List sorted by PID descending |

**Important:** `.gitattributes` marks `*.golden` as binary (`-text`) to prevent git from altering line endings or ANSI codes.

### Layer 3: Teatest pipeline tests

These tests run the **real bubbletea program** (Init → Update → View loop) in a virtual terminal using Charmbracelet's [teatest](https://github.com/charmbracelet/x/tree/main/exp/teatest). They validate that the full async lifecycle works: data fetching, tick-based refresh, user input, and rendering.

**The key difference from Layer 1:** teatest runs the program in a goroutine with a real message loop. Commands returned by `Init()` and `Update()` are actually executed asynchronously, so `fetchDataCmd` runs and reads from the file system.

**How it works:**

1. Create a fake `~/.claude/` directory with fixture session files using `setupFakeClaudeHome(t)`
2. Create the model with `WithHomeDir(fakeHome)` so it reads from the fixtures
3. Wrap it in `teatest.NewTestModel` which starts the bubbletea program
4. Use `teatest.WaitFor` to poll the terminal output until expected content appears
5. Send key events with `tm.Send()` or `tm.Type()`
6. Quit and wait for the program to finish

```go
func TestTeatest_ListViewPipeline(t *testing.T) {
    // 1. Create fake Claude home with 2 session files + history
    home := setupFakeClaudeHome(t)

    // 2. Create model pointing to fake home
    m := NewModel("test").WithHomeDir(home)

    // 3. Start the real bubbletea program in a virtual 120x40 terminal
    tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))

    // 4. Wait for data to load and render (Init → fetchDataCmd → dataMsg → View)
    teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
        return strings.Contains(string(bts), "99991") &&
            strings.Contains(string(bts), "99992")
    }, teatest.WithDuration(5*time.Second))

    // 5. Quit the program
    tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
    tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}
```

**What `setupFakeClaudeHome` creates:**

```
$TMPDIR/
└── .claude/
    ├── sessions/
    │   ├── aaaabbbb-1111-2222-3333-444444444444.json   # PID 99991, /tmp/project-one
    │   └── ccccdddd-5555-6666-7777-888888888888.json   # PID 99992, /tmp/project-two
    └── history.jsonl                                     # Last action per session
```

Since PIDs 99991/99992 don't match any real process, both sessions appear as DEAD — this is intentional and deterministic.

**Current teatest coverage:**

| Test | Scenario |
|---|---|
| `TestTeatest_ListViewPipeline` | Init loads fixture data, both sessions render |
| `TestTeatest_NavigateToDetail` | Load data → enter detail → verify fields → go back |
| `TestTeatest_FilterSessions` | Load data → filter for "project-one" → verify only one session visible |

### Simulating Claude sessions (CI without Claude)

The dashboard normally reads from `~/.claude/` which doesn't exist in CI. Two mechanisms enable testing without real Claude sessions:

#### Approach 1: Fixture rows (Layer 1 & 2)

`fixtureRows()` returns pre-built `display.Row` structs with deterministic data. These are injected directly into the model via `dataMsg`, completely bypassing the file system:

```go
// No file I/O — data is injected directly
m.Update(dataMsg{rows: fixtureRows(), err: nil})
```

This is used for model state machine tests and golden file tests where deterministic output is critical.

#### Approach 2: Fake home directory (Layer 3)

`setupFakeClaudeHome(t)` creates real files in a `t.TempDir()` that mirror the Claude Code directory structure. The model is configured to read from this directory via `WithHomeDir()`:

```go
home := setupFakeClaudeHome(t)           // creates sessions/*.json + history.jsonl
m := NewModel("test").WithHomeDir(home)  // fetchRows reads from $home/.claude/
```

This exercises the full data pipeline (file reading, JSON parsing, status detection) but with controlled data.

#### Dependency injection chain

The `homeDir` parameter flows through the entire data layer:

```
Model.homeDir
  └─→ fetchRows(homeDir)
        ├─→ session.LoadAllFrom(homeDir)       # reads $home/.claude/sessions/*.json
        ├─→ history.LastActionsFrom(homeDir)    # reads $home/.claude/history.jsonl
        └─→ logs.ResolvePathWithHome(id, cwd, homeDir)  # resolves log path under $home
```

When `homeDir` is empty (production), it falls back to `os.UserHomeDir()`.

### Adding a new golden test

1. Write a test function named `TestGolden_YourScenario`
2. Set up the model in the desired state
3. Call `golden.RequireEqual(t, []byte(m.View()))`
4. Run `go test ./internal/tui/ -update` to generate `testdata/TestGolden_YourScenario.golden`
5. Review the generated file, then commit it

```go
func TestGolden_YourScenario(t *testing.T) {
    m := newTestModel(t)
    // ... set up the desired state ...
    golden.RequireEqual(t, []byte(m.View()))
}
```

### Adding a new teatest pipeline test

1. Call `setupFakeClaudeHome(t)` or create custom fixture files
2. Create the model with `WithHomeDir`
3. Use `teatest.NewTestModel` to start the program
4. Use `teatest.WaitFor` to assert on rendered output
5. Always quit and call `WaitFinished` to clean up

```go
func TestTeatest_YourScenario(t *testing.T) {
    home := setupFakeClaudeHome(t)
    m := NewModel("test").WithHomeDir(home)
    tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))

    // Wait for expected content
    teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
        return strings.Contains(string(bts), "expected content")
    }, teatest.WithDuration(5*time.Second))

    // Interact and assert...

    tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
    tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}
```

### Test file organization

```
internal/tui/
├── helpers_test.go      # Fixture builders: fixtureRows(), setupFakeClaudeHome(), newTestModel()
├── model_test.go        # Layer 1: state machine tests (navigation, filter, sort, kill, etc.)
├── views_test.go        # Layer 2 & 3: golden file tests + teatest pipeline tests + unit tests
└── testdata/
    ├── TestGolden_ListView.golden
    ├── TestGolden_DetailView.golden
    ├── TestGolden_EmptyList.golden
    ├── TestGolden_FilterActive.golden
    ├── TestGolden_ConfirmKill.golden
    ├── TestGolden_LogView.golden
    └── TestGolden_SortDescending.golden
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

#### Process stats — `ps -eo pid,ppid,%cpu,%mem,args`

Live CPU and memory usage is obtained by running `ps -eo pid,ppid,%cpu,%mem,args` and filtering lines that contain `claude` (excluding `Claude.app` desktop processes). The PID from `ps` is matched against session files to determine if a session is alive. The PPID (parent PID) is used to build a process tree for detecting agent subprocesses.

A process is identified as a Claude Code CLI instance if:

- The command line contains "claude" (case-insensitive)
- The command line does NOT contain "Claude.app" or "Claude Helper"

**Code:** `internal/process/process.go`

#### Conversation logs — `~/.claude/projects/{slug}/{sessionID}.jsonl`

Each session's full conversation is stored as a JSONL file under `~/.claude/projects/`. The directory slug is derived from the session's working directory (with `/` and `.` replaced by `-`). These logs contain user messages, assistant responses, tool use, and thinking blocks.

The dashboard uses conversation logs for:

- **Turn count** — number of user messages in the session
- **Cost estimation** — token usage data is extracted to estimate per-session cost
- **WAITING status** — set when the session has active agent subprocesses (child Claude processes detected via process tree walk)
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
