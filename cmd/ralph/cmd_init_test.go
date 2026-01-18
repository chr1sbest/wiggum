package main

import (
	"testing"
)

func TestParseGeneratedPRD(t *testing.T) {
	tests := []struct {
		name     string
		response string
		want     string
	}{
		{
			name:     "valid response",
			response: "Some text\n---FILE: prd.json---\n{\"version\":1,\"tasks\":[]}",
			want:     `{"version":1,"tasks":[]}`,
		},
		{
			name:     "with code fence",
			response: "Some text\n---FILE: prd.json---\n```json\n{\"version\":1}\n```",
			want:     `{"version":1}`,
		},
		{
			name:     "no marker",
			response: "Some text without marker",
			want:     "",
		},
		{
			name:     "empty after marker",
			response: "---FILE: prd.json---\n",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseGeneratedPRD(tt.response)
			if got != tt.want {
				t.Errorf("parseGeneratedPRD() = %q, want %q", got, tt.want)
			}
		})
	}
}
