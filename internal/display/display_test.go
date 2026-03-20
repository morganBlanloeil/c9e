package display

import "testing"

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name    string
		seconds int64
		want    string
	}{
		{name: "negative", seconds: -1, want: "—"},
		{name: "zero", seconds: 0, want: "0m"},
		{name: "seconds only", seconds: 30, want: "0m"},
		{name: "one minute", seconds: 60, want: "1m"},
		{name: "minutes", seconds: 300, want: "5m"},
		{name: "one hour", seconds: 3600, want: "1h 0m"},
		{name: "hours and minutes", seconds: 5400, want: "1h 30m"},
		{name: "one day", seconds: 86400, want: "1d 0h"},
		{name: "days and hours", seconds: 90000, want: "1d 1h"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDuration(tt.seconds)
			if got != tt.want {
				t.Errorf("formatDuration(%d) = %q, want %q", tt.seconds, got, tt.want)
			}
		})
	}
}

func TestFormatIdle(t *testing.T) {
	tests := []struct {
		name    string
		seconds int64
		want    string
	}{
		{name: "negative", seconds: -1, want: "—"},
		{name: "zero", seconds: 0, want: "0s"},
		{name: "seconds", seconds: 45, want: "45s"},
		{name: "one minute boundary", seconds: 60, want: "1m"},
		{name: "minutes", seconds: 300, want: "5m"},
		{name: "one hour boundary", seconds: 3600, want: "1h 0m"},
		{name: "hours and minutes", seconds: 5400, want: "1h 30m"},
		{name: "one day boundary", seconds: 86400, want: "1d"},
		{name: "multiple days", seconds: 172800, want: "2d"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatIdle(tt.seconds)
			if got != tt.want {
				t.Errorf("formatIdle(%d) = %q, want %q", tt.seconds, got, tt.want)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name string
		s    string
		max  int
		want string
	}{
		{name: "short string", s: "hello", max: 10, want: "hello"},
		{name: "exact length", s: "hello", max: 5, want: "hello"},
		{name: "needs truncation", s: "hello world", max: 8, want: "hello w…"},
		{name: "empty string", s: "", max: 5, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.s, tt.max)
			if got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.s, tt.max, got, tt.want)
			}
		})
	}
}

func TestCleanAction(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "no change", input: "hello world", want: "hello world"},
		{name: "newlines", input: "line1\nline2\nline3", want: "line1 line2 line3"},
		{name: "carriage returns", input: "line1\r\nline2", want: "line1 line2"},
		{name: "mixed", input: "a\nb\r\nc\rd", want: "a b cd"},
		{name: "empty", input: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CleanAction(tt.input)
			if got != tt.want {
				t.Errorf("CleanAction(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIconFor(t *testing.T) {
	tests := []struct {
		status Status
		want   string
	}{
		{StatusActive, "●"},
		{StatusIdle, "◐"},
		{StatusDead, "○"},
		{Status("UNKNOWN"), "?"},
	}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			got := iconFor(tt.status)
			if got != tt.want {
				t.Errorf("iconFor(%q) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestColorFor(t *testing.T) {
	tests := []struct {
		status Status
		want   string
	}{
		{StatusActive, ansiGreen},
		{StatusIdle, ansiYellow},
		{StatusDead, ansiRed},
		{Status("UNKNOWN"), ansiReset},
	}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			got := colorFor(tt.status)
			if got != tt.want {
				t.Errorf("colorFor(%q) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}
