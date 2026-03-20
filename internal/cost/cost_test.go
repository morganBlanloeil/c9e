package cost

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormat(t *testing.T) {
	tests := []struct {
		name string
		cost float64
		want string
	}{
		{name: "zero", cost: 0.0, want: "$0.000"},
		{name: "small cost", cost: 0.005, want: "$0.005"},
		{name: "sub-cent", cost: 0.001, want: "$0.001"},
		{name: "normal cost", cost: 0.42, want: "$0.42"},
		{name: "dollar cost", cost: 1.23, want: "$1.23"},
		{name: "large cost", cost: 15.50, want: "$15.50"},
		{name: "at threshold", cost: 0.01, want: "$0.01"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Format(tt.cost)
			if got != tt.want {
				t.Errorf("Format(%f) = %q, want %q", tt.cost, got, tt.want)
			}
		})
	}
}

func TestCalculateCost(t *testing.T) {
	// Sonnet 4: $3/MTok input, $15/MTok output
	pricing := ModelPricing{InputPerMTok: 3.0, OutputPerMTok: 15.0}

	tests := []struct {
		name         string
		inputTokens  int64
		outputTokens int64
		wantCost     float64
	}{
		{name: "zero tokens", inputTokens: 0, outputTokens: 0, wantCost: 0.0},
		{name: "1M input only", inputTokens: 1_000_000, outputTokens: 0, wantCost: 3.0},
		{name: "1M output only", inputTokens: 0, outputTokens: 1_000_000, wantCost: 15.0},
		{name: "mixed tokens", inputTokens: 500_000, outputTokens: 100_000, wantCost: 3.0},
		// 500K * 3/1M = 1.50, 100K * 15/1M = 1.50 => 3.00
		{name: "small usage", inputTokens: 1000, outputTokens: 500, wantCost: 0.0105},
		// 1000 * 3/1M = 0.003, 500 * 15/1M = 0.0075 => 0.0105
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateCost(tt.inputTokens, tt.outputTokens, pricing)
			if math.Abs(got-tt.wantCost) > 0.0001 {
				t.Errorf("calculateCost(%d, %d) = %f, want %f", tt.inputTokens, tt.outputTokens, got, tt.wantCost)
			}
		})
	}
}

func TestPricingForModel(t *testing.T) {
	tests := []struct {
		name      string
		model     string
		wantInput float64
	}{
		{name: "exact sonnet", model: "claude-sonnet-4-20250514", wantInput: 3.0},
		{name: "exact opus", model: "claude-opus-4-20250514", wantInput: 15.0},
		{name: "exact haiku", model: "claude-haiku-3-5-20241022", wantInput: 0.80},
		{name: "keyword opus", model: "claude-opus-4-20250601", wantInput: 15.0},
		{name: "keyword sonnet", model: "claude-sonnet-4-20250601", wantInput: 3.0},
		{name: "keyword haiku", model: "claude-haiku-4-20250601", wantInput: 0.80},
		{name: "unknown model", model: "claude-unknown-99", wantInput: 3.0},
		{name: "empty model", model: "", wantInput: 3.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := pricingForModel(tt.model)
			if p.InputPerMTok != tt.wantInput {
				t.Errorf("pricingForModel(%q).InputPerMTok = %f, want %f", tt.model, p.InputPerMTok, tt.wantInput)
			}
		})
	}
}

func TestEstimateFromLog_WithUsageData(t *testing.T) {
	lines := []string{
		`{"type":"user","timestamp":"2026-03-20T10:00:00Z","message":{"role":"user","content":"Hello"}}`,
		`{"type":"assistant","timestamp":"2026-03-20T10:00:01Z","model":"claude-sonnet-4-20250514","usage":{"input_tokens":100,"output_tokens":50},"message":{"role":"assistant","content":[{"type":"text","text":"Hi there"}]}}`,
		`{"type":"user","timestamp":"2026-03-20T10:00:02Z","message":{"role":"user","content":"Do something"}}`,
		`{"type":"assistant","timestamp":"2026-03-20T10:00:03Z","model":"claude-sonnet-4-20250514","usage":{"input_tokens":200,"output_tokens":150},"message":{"role":"assistant","content":[{"type":"text","text":"Done"}]}}`,
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	c, err := EstimateFromLog(path)
	if err != nil {
		t.Fatal(err)
	}

	if !c.HasUsageData {
		t.Error("HasUsageData should be true")
	}
	if c.InputTokens != 300 {
		t.Errorf("InputTokens = %d, want 300", c.InputTokens)
	}
	if c.OutputTokens != 200 {
		t.Errorf("OutputTokens = %d, want 200", c.OutputTokens)
	}
	if c.Model != "claude-sonnet-4-20250514" {
		t.Errorf("Model = %q, want claude-sonnet-4-20250514", c.Model)
	}
	// 300 * 3/1M + 200 * 15/1M = 0.0009 + 0.003 = 0.0039
	if math.Abs(c.EstimatedCost-0.0039) > 0.0001 {
		t.Errorf("EstimatedCost = %f, want ~0.0039", c.EstimatedCost)
	}
}

func TestEstimateFromLog_WithoutUsageData(t *testing.T) {
	lines := []string{
		`{"type":"user","timestamp":"2026-03-20T10:00:00Z","message":{"role":"user","content":"Hello world this is a test message"}}`,
		`{"type":"assistant","timestamp":"2026-03-20T10:00:01Z","message":{"role":"assistant","content":[{"type":"text","text":"I will help you with that request right away"}]}}`,
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	c, err := EstimateFromLog(path)
	if err != nil {
		t.Fatal(err)
	}

	if c.HasUsageData {
		t.Error("HasUsageData should be false")
	}
	// "Hello world this is a test message" = 34 chars / 4 = 8 tokens input
	if c.InputTokens == 0 {
		t.Error("InputTokens should be > 0 from estimation")
	}
	if c.OutputTokens == 0 {
		t.Error("OutputTokens should be > 0 from estimation")
	}
	if c.EstimatedCost <= 0 {
		t.Error("EstimatedCost should be > 0")
	}
}

func TestEstimateFromLog_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.jsonl")
	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	c, err := EstimateFromLog(path)
	if err != nil {
		t.Fatal(err)
	}

	if c.InputTokens != 0 || c.OutputTokens != 0 {
		t.Error("expected zero tokens for empty file")
	}
	if c.EstimatedCost != 0 {
		t.Errorf("EstimatedCost = %f, want 0", c.EstimatedCost)
	}
}

func TestEstimateFromLog_FileNotFound(t *testing.T) {
	_, err := EstimateFromLog("/nonexistent/path.jsonl")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestEstimateFromLog_CacheTokens(t *testing.T) {
	lines := []string{
		`{"type":"assistant","timestamp":"2026-03-20T10:00:01Z","model":"claude-sonnet-4-20250514","usage":{"input_tokens":1000,"output_tokens":500,"cache_read_input_tokens":800,"cache_creation_input_tokens":200},"message":{"role":"assistant","content":[{"type":"text","text":"Hi"}]}}`,
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	c, err := EstimateFromLog(path)
	if err != nil {
		t.Fatal(err)
	}

	if c.CacheRead != 800 {
		t.Errorf("CacheRead = %d, want 800", c.CacheRead)
	}
	if c.CacheCreate != 200 {
		t.Errorf("CacheCreate = %d, want 200", c.CacheCreate)
	}
}

func TestEstimateTokensFromMessage(t *testing.T) {
	// String content: "Hello world" = 11 chars / 4 = 2
	raw := []byte(`"Hello world"`)
	got := estimateTokensFromMessage(raw)
	if got != 2 {
		t.Errorf("estimateTokensFromMessage string = %d, want 2", got)
	}

	// Message envelope with text content
	raw = []byte(`{"role":"assistant","content":[{"type":"text","text":"Hello world, how are you?"}]}`)
	got = estimateTokensFromMessage(raw)
	if got != 6 { // "Hello world, how are you?" = 25 chars / 4 = 6
		t.Errorf("estimateTokensFromMessage blocks = %d, want 6", got)
	}
}
