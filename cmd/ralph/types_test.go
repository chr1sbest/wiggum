package main

import (
	"encoding/json"
	"testing"
)

func TestJsonIntUnmarshal(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    jsonInt
		wantErr bool
	}{
		{"integer", `1`, 1, false},
		{"string integer", `"42"`, 42, false},
		{"null", `null`, 0, false},
		{"empty string", `""`, 0, true},
		{"zero", `0`, 0, false},
		{"negative", `-5`, -5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got jsonInt
			err := json.Unmarshal([]byte(tt.input), &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("jsonInt.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("jsonInt.UnmarshalJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPrdFileUnmarshal(t *testing.T) {
	input := `{
		"version": 1,
		"tasks": [
			{"id": "T001", "title": "First task", "status": "todo"},
			{"id": "T002", "title": "Second task", "status": "done"}
		]
	}`

	var prd prdFile
	if err := json.Unmarshal([]byte(input), &prd); err != nil {
		t.Fatalf("Failed to unmarshal prdFile: %v", err)
	}

	if prd.Version != 1 {
		t.Errorf("Version = %d, want 1", prd.Version)
	}
	if len(prd.Tasks) != 2 {
		t.Errorf("Tasks count = %d, want 2", len(prd.Tasks))
	}
	if prd.Tasks[0].ID != "T001" {
		t.Errorf("Tasks[0].ID = %s, want T001", prd.Tasks[0].ID)
	}
	if prd.Tasks[1].Status != "done" {
		t.Errorf("Tasks[1].Status = %s, want done", prd.Tasks[1].Status)
	}
}

func TestStripJSONFences(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no fences",
			input: `{"key": "value"}`,
			want:  `{"key": "value"}`,
		},
		{
			name:  "with json fence",
			input: "```json\n{\"key\": \"value\"}\n```",
			want:  `{"key": "value"}`,
		},
		{
			name:  "with plain fence",
			input: "```\n{\"key\": \"value\"}\n```",
			want:  `{"key": "value"}`,
		},
		{
			name:  "whitespace around",
			input: "  \n```json\n{\"key\": \"value\"}\n```\n  ",
			want:  `{"key": "value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripJSONFences(tt.input)
			if got != tt.want {
				t.Errorf("stripJSONFences() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"empty", "", nil},
		{"single line", "hello", []string{"hello"}},
		{"two lines", "hello\nworld", []string{"hello", "world"}},
		{"trailing newline", "hello\n", []string{"hello"}},
		{"multiple lines", "a\nb\nc", []string{"a", "b", "c"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitLines(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("splitLines() len = %d, want %d", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("splitLines()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
