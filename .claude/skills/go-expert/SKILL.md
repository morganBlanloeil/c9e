---
name: go-expert
description: >-
  Go language expert for idiomatic code, error handling, testing patterns, and performance.
  Use when writing, reviewing, or debugging Go code. Triggers on Go files (.go),
  go.mod, go.sum, or when user mentions Go idioms, error handling, or Go best practices.
allowed-tools: Read, Edit, Write, Grep, Glob, Bash
---

# Go Expert

Expert in Go programming language best practices, idiomatic patterns, and tooling.

## Core Principles

- **Simplicity over cleverness** — Go favors clear, readable code
- **Errors are values** — handle them explicitly, never ignore
- **Accept interfaces, return structs** — design for flexibility
- **Make the zero value useful** — avoid constructors when possible

## Error Handling

```go
// DO: Wrap errors with context
if err != nil {
    return fmt.Errorf("reading session %s: %w", path, err)
}

// DON'T: Bare returns or generic messages
if err != nil {
    return err
}
```

- Always wrap errors with `%w` for unwrapping support
- Add context about what operation failed
- Use `errors.Is` / `errors.As` for error inspection
- Define sentinel errors with `var ErrNotFound = errors.New("not found")`

## Testing Patterns

### Table-driven tests

```go
func TestParseSession(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    Session
        wantErr bool
    }{
        {name: "valid session", input: `{"pid":123}`, want: Session{PID: 123}},
        {name: "empty input", input: "", wantErr: true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ParseSession(tt.input)
            if (err != nil) != tt.wantErr {
                t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
            }
            if !tt.wantErr && got != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Test helpers

```go
func testdataPath(t *testing.T, name string) string {
    t.Helper()
    return filepath.Join("testdata", name)
}
```

- Use `testdata/` directories for fixtures
- Use `t.Helper()` in helper functions
- Use `t.Cleanup()` instead of `defer` in tests
- Run single test: `go test ./internal/session/ -run TestParseSesssion`

## Naming Conventions

| Type | Convention | Example |
|------|-----------|---------|
| Package | short, lowercase, no underscores | `session`, `display` |
| Exported func | PascalCase, verb first | `ParseSession`, `RenderTable` |
| Unexported func | camelCase | `fetchRows`, `buildRow` |
| Interface | -er suffix when possible | `Reader`, `Formatter` |
| Acronyms | ALL CAPS in names | `PID`, `JSON`, `URL` |

## Struct Design

```go
// DO: Group related fields, use meaningful zero values
type Session struct {
    PID       int
    SessionID string
    CWD       string
    StartedAt time.Time
}

// DON'T: Use pointers for optional fields when zero value works
```

## Concurrency

- Use `sync.WaitGroup` for fan-out/fan-in
- Prefer channels for communication, mutexes for state
- Always use `context.Context` as first parameter for cancellable operations
- Use `errgroup.Group` for concurrent tasks that can fail

## Performance

- Profile before optimizing: `go test -bench=. -benchmem`
- Use `strings.Builder` for string concatenation
- Pre-allocate slices when size is known: `make([]T, 0, n)`
- Avoid allocations in hot paths: reuse buffers

## Go Modules

- Run `go mod tidy` after dependency changes
- Use `go vet ./...` before committing
- Use `gofmt` / `goimports` for formatting
