package eval

import (
	"strings"
	"testing"
)

func TestNewRunConfig(t *testing.T) {
	config := NewRunConfig("flask", "ralph", "sonnet")

	if config.SuiteName != "flask" {
		t.Errorf("expected suite name 'flask', got '%s'", config.SuiteName)
	}
	if config.Approach != "ralph" {
		t.Errorf("expected approach 'ralph', got '%s'", config.Approach)
	}
	if config.Model != "sonnet" {
		t.Errorf("expected model 'sonnet', got '%s'", config.Model)
	}
	if config.TimeoutSeconds != DefaultTimeoutSeconds {
		t.Errorf("expected timeout %d, got %d", DefaultTimeoutSeconds, config.TimeoutSeconds)
	}
	if config.OutputDir != "" {
		t.Errorf("expected empty output dir, got '%s'", config.OutputDir)
	}
}

func TestValidateApproach(t *testing.T) {
	tests := []struct {
		name      string
		approach  string
		wantErr   bool
		normalized string
	}{
		{"valid ralph lowercase", "ralph", false, "ralph"},
		{"valid ralph uppercase", "RALPH", false, "ralph"},
		{"valid ralph mixed case", "Ralph", false, "ralph"},
		{"valid oneshot lowercase", "oneshot", false, "oneshot"},
		{"valid oneshot uppercase", "ONESHOT", false, "oneshot"},
		{"valid oneshot mixed case", "OneShot", false, "oneshot"},
		{"valid ralph with spaces", "  ralph  ", false, "ralph"},
		{"valid oneshot with spaces", "  oneshot  ", false, "oneshot"},
		{"invalid approach", "invalid", true, ""},
		{"empty approach", "", true, ""},
		{"random string", "random", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &RunConfig{
				SuiteName:      "test",
				Approach:       tt.approach,
				Model:          "sonnet",
				TimeoutSeconds: DefaultTimeoutSeconds,
			}

			err := config.ValidateApproach()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateApproach() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && config.Approach != tt.normalized {
				t.Errorf("expected normalized approach '%s', got '%s'", tt.normalized, config.Approach)
			}

			if tt.wantErr && err != nil {
				if !strings.Contains(err.Error(), "invalid approach") {
					t.Errorf("expected error to contain 'invalid approach', got: %v", err)
				}
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *RunConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: &RunConfig{
				SuiteName:      "flask",
				Approach:       "ralph",
				Model:          "sonnet",
				TimeoutSeconds: DefaultTimeoutSeconds,
			},
			wantErr: false,
		},
		{
			name: "empty suite name",
			config: &RunConfig{
				SuiteName:      "",
				Approach:       "ralph",
				Model:          "sonnet",
				TimeoutSeconds: DefaultTimeoutSeconds,
			},
			wantErr: true,
			errMsg:  "suite name cannot be empty",
		},
		{
			name: "invalid approach",
			config: &RunConfig{
				SuiteName:      "flask",
				Approach:       "invalid",
				Model:          "sonnet",
				TimeoutSeconds: DefaultTimeoutSeconds,
			},
			wantErr: true,
			errMsg:  "invalid approach",
		},
		{
			name: "empty model",
			config: &RunConfig{
				SuiteName:      "flask",
				Approach:       "ralph",
				Model:          "",
				TimeoutSeconds: DefaultTimeoutSeconds,
			},
			wantErr: true,
			errMsg:  "model cannot be empty",
		},
		{
			name: "zero timeout",
			config: &RunConfig{
				SuiteName:      "flask",
				Approach:       "ralph",
				Model:          "sonnet",
				TimeoutSeconds: 0,
			},
			wantErr: true,
			errMsg:  "timeout must be positive",
		},
		{
			name: "negative timeout",
			config: &RunConfig{
				SuiteName:      "flask",
				Approach:       "ralph",
				Model:          "sonnet",
				TimeoutSeconds: -100,
			},
			wantErr: true,
			errMsg:  "timeout must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error to contain '%s', got: %v", tt.errMsg, err)
				}
			}
		})
	}
}

func TestIsRalphApproach(t *testing.T) {
	tests := []struct {
		name     string
		approach string
		want     bool
	}{
		{"ralph lowercase", "ralph", true},
		{"ralph uppercase", "RALPH", true},
		{"ralph mixed case", "Ralph", true},
		{"ralph with spaces", "  ralph  ", true},
		{"oneshot", "oneshot", false},
		{"invalid", "invalid", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &RunConfig{Approach: tt.approach}
			if got := config.IsRalphApproach(); got != tt.want {
				t.Errorf("IsRalphApproach() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsOneshotApproach(t *testing.T) {
	tests := []struct {
		name     string
		approach string
		want     bool
	}{
		{"oneshot lowercase", "oneshot", true},
		{"oneshot uppercase", "ONESHOT", true},
		{"oneshot mixed case", "OneShot", true},
		{"oneshot with spaces", "  oneshot  ", true},
		{"ralph", "ralph", false},
		{"invalid", "invalid", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &RunConfig{Approach: tt.approach}
			if got := config.IsOneshotApproach(); got != tt.want {
				t.Errorf("IsOneshotApproach() = %v, want %v", got, tt.want)
			}
		})
	}
}
