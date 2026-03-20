---
name: go-testing
description: >-
  Go testing specialist for writing and improving test coverage.
  Use when creating tests, improving coverage, writing benchmarks, or debugging test failures.
  Triggers on _test.go files, testdata/ directories, or when user mentions
  test coverage, table-driven tests, or go test commands.
allowed-tools: Read, Edit, Write, Grep, Glob, Bash
---

# Go Testing Specialist

Expert in Go testing patterns, coverage improvement, and test tooling.

## Running Tests

```bash
# All tests
go test ./...

# Single package
go test ./internal/session/

# Single test
go test ./internal/session/ -run TestParseSession

# With verbose output
go test -v ./internal/session/ -run TestParseSession

# With coverage
go test -cover ./...

# Coverage report
go test -coverprofile=cover.out ./... && go tool cover -html=cover.out

# Benchmarks
go test -bench=. -benchmem ./internal/history/
```

## Test Structure

### Table-driven tests (preferred pattern)

```go
func TestStatusFromIdle(t *testing.T) {
    tests := []struct {
        name     string
        idleSecs float64
        alive    bool
        want     string
    }{
        {name: "active and alive", idleSecs: 60, alive: true, want: "ACTIVE"},
        {name: "idle and alive", idleSecs: 600, alive: true, want: "IDLE"},
        {name: "dead process", idleSecs: 0, alive: false, want: "DEAD"},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := StatusFromIdle(tt.idleSecs, tt.alive)
            if got != tt.want {
                t.Errorf("StatusFromIdle(%v, %v) = %q, want %q",
                    tt.idleSecs, tt.alive, got, tt.want)
            }
        })
    }
}
```

### Test file organization

```
internal/session/
    session.go
    session_test.go
    testdata/
        valid_session.json
        corrupt_session.json
```

- Test file lives next to source: `foo.go` → `foo_test.go`
- Use `testdata/` for fixtures (ignored by `go build`)
- Use same package name for white-box testing
- Use `_test` package suffix for black-box testing

## Fixtures and Helpers

### testdata directory

```go
func TestParseSessionFile(t *testing.T) {
    data, err := os.ReadFile(filepath.Join("testdata", "valid_session.json"))
    if err != nil {
        t.Fatal(err)
    }
    // ...
}
```

### t.TempDir for temporary files

```go
func TestWriteOutput(t *testing.T) {
    dir := t.TempDir() // auto-cleaned up
    path := filepath.Join(dir, "output.json")
    // ...
}
```

### Test helpers

```go
func mustParseJSON(t *testing.T, data string) Session {
    t.Helper() // error points to caller, not helper
    var s Session
    if err := json.Unmarshal([]byte(data), &s); err != nil {
        t.Fatal(err)
    }
    return s
}
```

## Testing This Project

### Priority areas for test coverage

| Package | What to test | Strategy |
|---------|-------------|----------|
| `internal/session` | JSON parsing, edge cases (corrupt files, missing fields) | Table-driven with testdata/ fixtures |
| `internal/history` | JSONL parsing, tail reading, session matching | Table-driven with testdata/ fixtures |
| `internal/process` | ps output parsing, process matching | Table-driven with mock ps output |
| `internal/display` | Table rendering, JSON output | Golden file tests |
| `internal/logs` | JSONL streaming, entry filtering | Already has tests — extend |

### Golden file testing (for display output)

```go
func TestRenderTable(t *testing.T) {
    rows := []display.Row{
        {PID: 123, Status: "ACTIVE", Directory: "/home/user/project"},
    }
    got := display.RenderTable(rows)

    golden := filepath.Join("testdata", t.Name()+".golden")
    if *update {
        os.WriteFile(golden, []byte(got), 0o644)
    }
    want, _ := os.ReadFile(golden)
    if got != string(want) {
        t.Errorf("output mismatch:\ngot:\n%s\nwant:\n%s", got, want)
    }
}

var update = flag.Bool("update", false, "update golden files")
```

### Testing ps output parsing

```go
func TestParseProcesses(t *testing.T) {
    // Mock ps output as string instead of calling real ps
    psOutput := `USER  PID %CPU %MEM    VSZ   RSS   TT STAT STARTED  TIME COMMAND
user 1234  5.0  1.2 123456 12345 s000 S+   10:00AM 0:30.00 node claude --session abc
user 5678  0.1  0.5  98765  6789 s001 S+   11:00AM 0:05.00 Claude.app Helper`

    procs := parseProcessLines(psOutput)
    if len(procs) != 1 { // Claude.app should be filtered
        t.Errorf("got %d processes, want 1", len(procs))
    }
}
```

## Workflow

When asked to improve test coverage:

1. Run `go test -cover ./...` to identify gaps
2. Read the source file to understand the logic
3. Create `_test.go` with table-driven tests
4. Add `testdata/` fixtures if needed
5. Run tests to verify they pass
6. Check coverage improved: `go test -cover ./...`
