package main

import (
	"testing"
)

func TestParseNewTasks(t *testing.T) {
	tests := []struct {
		name     string
		response string
		want     string
	}{
		{
			name:     "valid response",
			response: "Some text\n---NEW_TASKS---\n[{\"id\":\"T001\"}]",
			want:     `[{"id":"T001"}]`,
		},
		{
			name:     "with code fence",
			response: "Some text\n---NEW_TASKS---\n```json\n[{\"id\":\"T001\"}]\n```",
			want:     `[{"id":"T001"}]`,
		},
		{
			name:     "no marker",
			response: "Some text without marker",
			want:     "",
		},
		{
			name:     "empty after marker",
			response: "---NEW_TASKS---\n",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseNewTasks(tt.response)
			if got != tt.want {
				t.Errorf("parseNewTasks() = %q, want %q", got, tt.want)
			}
		})
	}
}
