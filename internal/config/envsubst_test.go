package config

import (
	"os"
	"testing"
)

func TestExpandEnvVars(t *testing.T) {
	// Set up test environment variables
	os.Setenv("TEST_VAR", "hello")
	os.Setenv("ANOTHER_VAR", "world")
	defer func() {
		os.Unsetenv("TEST_VAR")
		os.Unsetenv("ANOTHER_VAR")
	}()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no variables",
			input:    "plain text",
			expected: "plain text",
		},
		{
			name:     "simple variable",
			input:    "say ${TEST_VAR}",
			expected: "say hello",
		},
		{
			name:     "multiple variables",
			input:    "${TEST_VAR} ${ANOTHER_VAR}",
			expected: "hello world",
		},
		{
			name:     "unset variable becomes empty",
			input:    "value: ${UNSET_VAR}",
			expected: "value: ",
		},
		{
			name:     "default value used when unset",
			input:    "value: ${UNSET_VAR:-fallback}",
			expected: "value: fallback",
		},
		{
			name:     "default value ignored when set",
			input:    "value: ${TEST_VAR:-unused}",
			expected: "value: hello",
		},
		{
			name:     "empty default value",
			input:    "value: ${UNSET_VAR:-}",
			expected: "value: ",
		},
		{
			name:     "variable in JSON context",
			input:    `{"name": "${TEST_VAR}", "path": "${UNSET_PATH:-/default/path}"}`,
			expected: `{"name": "hello", "path": "/default/path"}`,
		},
		{
			name:     "variable with underscores",
			input:    "${TEST_VAR}",
			expected: "hello",
		},
		{
			name:     "variable with numbers",
			input:    "${VAR_123:-num}",
			expected: "num",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandEnvVars(tt.input)
			if result != tt.expected {
				t.Errorf("ExpandEnvVars(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExpandEnvVarsBytes(t *testing.T) {
	os.Setenv("TEST_VAR", "hello")
	defer os.Unsetenv("TEST_VAR")

	input := []byte("say ${TEST_VAR}")
	expected := []byte("say hello")

	result := ExpandEnvVarsBytes(input)
	if string(result) != string(expected) {
		t.Errorf("ExpandEnvVarsBytes(%q) = %q, want %q", input, result, expected)
	}
}
