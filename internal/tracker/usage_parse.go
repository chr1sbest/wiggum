package tracker

import (
	"encoding/json"
	"strconv"
	"strings"
)

// ParseClaudeUsageFromOutput best-effort parses Claude CLI JSON output and extracts usage.
// It returns ok=false when no usage information could be found.
func ParseClaudeUsageFromOutput(output string) (UsageDelta, bool) {
	// Strip any appended stderr section.
	if idx := strings.Index(output, "\n--- STDERR ---\n"); idx >= 0 {
		output = output[:idx]
	}
	output = strings.TrimSpace(output)
	if output == "" {
		return UsageDelta{}, false
	}

	// Try to parse the entire output as JSON first
	var v any
	if err := json.Unmarshal([]byte(output), &v); err != nil {
		// If that fails, try to extract JSON objects from the text
		v = extractLastJSON(output)
		if v == nil {
			return UsageDelta{}, false
		}
	}

	// Claude uses prompt caching, so input tokens are split across multiple fields.
	// Sum them all to get the true input token count.
	input := findInt(v, []string{"input_tokens", "prompt_tokens"})
	cacheCreation := findInt(v, []string{"cache_creation_input_tokens"})
	cacheRead := findInt(v, []string{"cache_read_input_tokens"})
	input += cacheCreation + cacheRead

	out := findInt(v, []string{"output_tokens", "completion_tokens"})
	total := findInt(v, []string{"total_tokens", "tokens"})
	cost := findFloat(v, []string{"total_cost", "cost", "total_cost_usd", "cost_usd"})
	turns := findInt(v, []string{"num_turns"})

	if total == 0 {
		if input > 0 || out > 0 {
			total = input + out
		}
	}

	// Check if we found any of the relevant keys at all
	// If none of the keys exist in the JSON, return false
	// But if they exist with zero values, that's valid
	hasAnyKey := hasAnyKey(v, []string{
		"input_tokens", "prompt_tokens", "output_tokens", "completion_tokens",
		"total_tokens", "tokens", "total_cost", "cost", "total_cost_usd", "cost_usd",
		"cache_creation_input_tokens", "cache_read_input_tokens",
	})
	if !hasAnyKey {
		return UsageDelta{}, false
	}

	return UsageDelta{InputTokens: input, OutputTokens: out, TotalTokens: total, CostUSD: cost, Turns: turns}, true
}

// extractLastJSON finds and parses the last valid JSON object in the text.
// It scans each line and tries to parse it as JSON, keeping the last successful parse.
func extractLastJSON(text string) any {
	var lastValid any
	lines := strings.Split(text, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "{") {
			continue
		}

		var v any
		if err := json.Unmarshal([]byte(line), &v); err == nil {
			lastValid = v
		}
	}

	return lastValid
}

// hasAnyKey checks if any of the given keys exist in the JSON structure
func hasAnyKey(v any, keys []string) bool {
	keySet := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		keySet[k] = struct{}{}
	}

	var walk func(any) bool
	walk = func(x any) bool {
		switch t := x.(type) {
		case map[string]any:
			for k := range t {
				kl := strings.ToLower(k)
				if _, ok := keySet[kl]; ok {
					return true
				}
			}
			for _, vv := range t {
				if walk(vv) {
					return true
				}
			}
		case []any:
			for _, vv := range t {
				if walk(vv) {
					return true
				}
			}
		}
		return false
	}

	return walk(v)
}

func findInt(v any, keys []string) int {
	found, ok := findNumber(v, keys)
	if !ok {
		return 0
	}
	return int(found)
}

func findFloat(v any, keys []string) float64 {
	found, ok := findNumber(v, keys)
	if !ok {
		return 0
	}
	return found
}

func findNumber(v any, keys []string) (float64, bool) {
	keySet := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		keySet[k] = struct{}{}
	}

	var walk func(any) (float64, bool)
	walk = func(x any) (float64, bool) {
		switch t := x.(type) {
		case map[string]any:
			for k, vv := range t {
				kl := strings.ToLower(k)
				if _, ok := keySet[kl]; ok {
					if n, ok := toFloat(vv); ok {
						return n, true
					}
				}
			}
			for _, vv := range t {
				if n, ok := walk(vv); ok {
					return n, true
				}
			}
		case []any:
			for _, vv := range t {
				if n, ok := walk(vv); ok {
					return n, true
				}
			}
		}
		return 0, false
	}

	return walk(v)
}

func toFloat(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	case json.Number:
		f, err := t.Float64()
		return f, err == nil
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(t), 64)
		return f, err == nil
	default:
		return 0, false
	}
}
