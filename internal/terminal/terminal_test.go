package terminal

import (
	"os"
	"testing"
)

func TestAncestors(t *testing.T) {
	pid := os.Getpid()
	ancestors, err := Ancestors(pid)
	if err != nil {
		t.Fatalf("Ancestors(%d) returned error: %v", pid, err)
	}
	if len(ancestors) == 0 {
		t.Fatalf("Ancestors(%d) returned empty chain, expected at least one ancestor", pid)
	}
	// The last ancestor should be PID 1 (init/launchd).
	last := ancestors[len(ancestors)-1]
	if last != 1 && last != 0 {
		t.Errorf("expected final ancestor to be 0 or 1, got %d", last)
	}
}

func TestAvailable(t *testing.T) {
	// Just ensure it doesn't panic and returns a bool.
	_ = Available()
}
