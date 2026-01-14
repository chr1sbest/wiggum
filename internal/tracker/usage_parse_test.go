package tracker

import "testing"

func TestParseClaudeUsageFromOutput(t *testing.T) {
	out := `{"usage":{"input_tokens":12,"output_tokens":34,"total_tokens":46},"total_cost_usd":0.123}`
	d, ok := ParseClaudeUsageFromOutput(out)
	if !ok {
		t.Fatalf("expected ok")
	}
	if d.InputTokens != 12 || d.OutputTokens != 34 || d.TotalTokens != 46 {
		t.Fatalf("unexpected tokens: %+v", d)
	}
	if d.CostUSD != 0.123 {
		t.Fatalf("unexpected cost: %+v", d)
	}
}

func TestParseClaudeUsageWithCacheTokens(t *testing.T) {
	// Real Claude output with prompt caching
	out := `{"usage":{"input_tokens":2,"cache_creation_input_tokens":6843,"cache_read_input_tokens":91983,"output_tokens":931},"total_cost_usd":0.112}`
	d, ok := ParseClaudeUsageFromOutput(out)
	if !ok {
		t.Fatalf("expected ok")
	}
	// input = 2 + 6843 + 91983 = 98828
	expectedInput := 2 + 6843 + 91983
	if d.InputTokens != expectedInput {
		t.Fatalf("expected input tokens %d, got %d", expectedInput, d.InputTokens)
	}
	if d.OutputTokens != 931 {
		t.Fatalf("expected output tokens 931, got %d", d.OutputTokens)
	}
}
