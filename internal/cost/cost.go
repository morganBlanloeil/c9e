package cost

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	sonnetInputPrice  = 3.0
	sonnetOutputPrice = 15.0
	opusInputPrice    = 15.0
	opusOutputPrice   = 75.0
	haikuInputPrice   = 0.80
	haikuOutputPrice  = 4.0
	minDisplayCost    = 0.01
	scannerBufSize    = 512 * 1024
	charsPerToken     = 4
	tokensPerMillion  = 1_000_000
)

// Pricing per million tokens (USD).
type ModelPricing struct {
	InputPerMTok  float64
	OutputPerMTok float64
}

// Known model pricing.
var modelPricing = map[string]ModelPricing{
	"claude-sonnet-4-20250514":  {InputPerMTok: sonnetInputPrice, OutputPerMTok: sonnetOutputPrice},
	"claude-opus-4-20250514":    {InputPerMTok: opusInputPrice, OutputPerMTok: opusOutputPrice},
	"claude-haiku-3-5-20241022": {InputPerMTok: haikuInputPrice, OutputPerMTok: haikuOutputPrice},
}

// defaultPricing is used when the model is unknown (assumes Sonnet 4 pricing).
var defaultPricing = ModelPricing{InputPerMTok: sonnetInputPrice, OutputPerMTok: sonnetOutputPrice}

// Cost holds token usage and estimated cost for a session.
type Cost struct {
	InputTokens   int64   `json:"input_tokens"`
	OutputTokens  int64   `json:"output_tokens"`
	CacheRead     int64   `json:"cache_read_tokens,omitempty"`
	CacheCreate   int64   `json:"cache_create_tokens,omitempty"`
	EstimatedCost float64 `json:"estimated_cost"`
	Model         string  `json:"model"`
	HasUsageData  bool    `json:"has_usage_data"`
}

// Format renders a cost as a dollar string, e.g. "$0.42" or "$1.23".
func Format(cost float64) string {
	if cost < minDisplayCost {
		return fmt.Sprintf("$%.3f", cost)
	}
	return fmt.Sprintf("$%.2f", cost)
}

// EstimateFromLog reads a session JSONL log file and estimates cost based on
// token usage data found in assistant responses.
func EstimateFromLog(logPath string) (result Cost, err error) {
	f, err := os.Open(logPath)
	if err != nil {
		return Cost{}, fmt.Errorf("opening log file: %w", err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("closing log file: %w", cerr)
		}
	}()

	return estimateFromReader(f)
}

func estimateFromReader(r io.Reader) (Cost, error) {
	var result Cost
	var estimatedInput, estimatedOutput int64

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, scannerBufSize), scannerBufSize)

	for scanner.Scan() {
		data := scanner.Bytes()
		var line logLine
		if err := json.Unmarshal(data, &line); err != nil {
			continue
		}

		// Usage data appears at the top level of assistant entries
		if line.Usage.InputTokens > 0 || line.Usage.OutputTokens > 0 {
			result.InputTokens += int64(line.Usage.InputTokens)
			result.OutputTokens += int64(line.Usage.OutputTokens)
			result.CacheRead += int64(line.Usage.CacheReadInputTokens)
			result.CacheCreate += int64(line.Usage.CacheCreationInputTokens)
			result.HasUsageData = true
		}

		// Capture model from the line
		if line.Model != "" {
			result.Model = line.Model
		}

		// Collect estimated tokens as fallback (only used if no usage data found)
		if line.Message != nil {
			tokens := estimateTokensFromMessage(line.Message)
			switch line.Type {
			case "user":
				estimatedInput += int64(tokens)
			case "assistant":
				estimatedOutput += int64(tokens)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return result, fmt.Errorf("scanning log file: %w", err)
	}

	// If no real usage data was found, use estimated tokens
	if !result.HasUsageData {
		result.InputTokens = estimatedInput
		result.OutputTokens = estimatedOutput
	}

	// Calculate cost
	pricing := pricingForModel(result.Model)
	result.EstimatedCost = calculateCost(result.InputTokens, result.OutputTokens, pricing)

	return result, nil
}

// logLine is the minimal structure for extracting usage and model data.
type logLine struct {
	Type    string          `json:"type"`
	Model   string          `json:"model,omitempty"`
	Usage   usageData       `json:"usage,omitempty"`
	Message json.RawMessage `json:"message,omitempty"`
}

type usageData struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
}

// estimateTokensFromMessage approximates token count from message content.
// Rough approximation: ~4 characters per token.
func estimateTokensFromMessage(raw json.RawMessage) int {
	// Try as string first
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return len(s) / charsPerToken
	}

	// Try as message envelope with content
	var msg struct {
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(raw, &msg); err != nil {
		return len(raw) / charsPerToken
	}

	// Content might be a string
	var contentStr string
	if err := json.Unmarshal(msg.Content, &contentStr); err == nil {
		return len(contentStr) / charsPerToken
	}

	// Content might be an array of blocks
	var blocks []struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(msg.Content, &blocks); err == nil {
		total := 0
		for _, b := range blocks {
			total += len(b.Text)
		}
		return total / charsPerToken
	}

	return len(raw) / charsPerToken
}

// pricingForModel returns the pricing for a given model string.
// It does a prefix match to handle versioned model names.
func pricingForModel(model string) ModelPricing {
	if model == "" {
		return defaultPricing
	}

	// Exact match first
	if p, ok := modelPricing[model]; ok {
		return p
	}

	// Prefix match for versioned model names
	lower := strings.ToLower(model)
	for key, p := range modelPricing {
		if strings.HasPrefix(lower, strings.TrimSuffix(key, key[strings.LastIndex(key, "-"):])) {
			return p
		}
	}

	// Keyword match
	if strings.Contains(lower, "opus") {
		return modelPricing["claude-opus-4-20250514"]
	}
	if strings.Contains(lower, "haiku") {
		return modelPricing["claude-haiku-3-5-20241022"]
	}
	if strings.Contains(lower, "sonnet") {
		return modelPricing["claude-sonnet-4-20250514"]
	}

	return defaultPricing
}

// calculateCost computes the cost in USD from token counts and pricing.
func calculateCost(inputTokens, outputTokens int64, pricing ModelPricing) float64 {
	inputCost := float64(inputTokens) / tokensPerMillion * pricing.InputPerMTok
	outputCost := float64(outputTokens) / tokensPerMillion * pricing.OutputPerMTok
	return inputCost + outputCost
}
