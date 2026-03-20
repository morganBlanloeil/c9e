package history

import (
	"bufio"
	"strings"
	"testing"
)

func TestScanEntries(t *testing.T) {
	input := strings.Join([]string{
		`{"display":"fix bug","timestamp":1000,"project":"/home/user/proj","sessionId":"sess-1"}`,
		`{"display":"add feature","timestamp":2000,"project":"/home/user/proj","sessionId":"sess-2"}`,
		`{"display":"update tests","timestamp":3000,"project":"/home/user/proj","sessionId":"sess-1"}`,
		`invalid json line`,
		`{"display":"","timestamp":4000,"project":"/home/user/proj","sessionId":"sess-3"}`,
		`{"display":"no session","timestamp":5000,"project":"/home/user/proj","sessionId":""}`,
	}, "\n")

	result, err := scanEntries(bufio.NewReader(strings.NewReader(input)))
	if err != nil {
		t.Fatalf("scanEntries returned error: %v", err)
	}

	// Should have 2 sessions (sess-1 and sess-2)
	// sess-3 skipped (empty display), no-session skipped (empty sessionId)
	if len(result) != 2 {
		t.Fatalf("got %d entries, want 2", len(result))
	}

	// sess-1 should have the latest entry (timestamp 3000)
	if e, ok := result["sess-1"]; !ok {
		t.Error("missing sess-1")
	} else {
		if e.Display != "update tests" {
			t.Errorf("sess-1 display = %q, want %q", e.Display, "update tests")
		}
		if e.Timestamp != 3000 {
			t.Errorf("sess-1 timestamp = %d, want 3000", e.Timestamp)
		}
	}

	// sess-2 should be present
	if e, ok := result["sess-2"]; !ok {
		t.Error("missing sess-2")
	} else if e.Display != "add feature" {
		t.Errorf("sess-2 display = %q, want %q", e.Display, "add feature")
	}
}

func TestScanEntriesEmpty(t *testing.T) {
	result, err := scanEntries(bufio.NewReader(strings.NewReader("")))
	if err != nil {
		t.Fatalf("scanEntries returned error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("got %d entries, want 0", len(result))
	}
}

func TestScanEntriesKeepsLatest(t *testing.T) {
	input := strings.Join([]string{
		`{"display":"old","timestamp":100,"project":"/p","sessionId":"s1"}`,
		`{"display":"new","timestamp":200,"project":"/p","sessionId":"s1"}`,
		`{"display":"oldest","timestamp":50,"project":"/p","sessionId":"s1"}`,
	}, "\n")

	result, err := scanEntries(bufio.NewReader(strings.NewReader(input)))
	if err != nil {
		t.Fatalf("scanEntries returned error: %v", err)
	}

	if e := result["s1"]; e.Display != "new" || e.Timestamp != 200 {
		t.Errorf("expected latest entry, got display=%q timestamp=%d", e.Display, e.Timestamp)
	}
}
