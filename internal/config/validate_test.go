package config

import (
	"testing"
)

func TestValidator(t *testing.T) {
	knownTypes := []string{"command", "noop", "readme-check"}
	validator := NewValidator(knownTypes)

	tests := []struct {
		name       string
		config     *Config
		wantErrors int
		wantFields []string
	}{
		{
			name: "valid config",
			config: &Config{
				Name: "test",
				Steps: []StepConfig{
					{Type: "command", Name: "run-tests"},
				},
			},
			wantErrors: 0,
		},
		{
			name:       "missing config name",
			config:     &Config{Steps: []StepConfig{{Type: "noop", Name: "test"}}},
			wantErrors: 1,
			wantFields: []string{"name"},
		},
		{
			name:       "empty steps",
			config:     &Config{Name: "test", Steps: []StepConfig{}},
			wantErrors: 1,
			wantFields: []string{"steps"},
		},
		{
			name: "missing step type",
			config: &Config{
				Name:  "test",
				Steps: []StepConfig{{Name: "test-step"}},
			},
			wantErrors: 1,
			wantFields: []string{"type"},
		},
		{
			name: "unknown step type",
			config: &Config{
				Name:  "test",
				Steps: []StepConfig{{Type: "unknown", Name: "test-step"}},
			},
			wantErrors: 1,
			wantFields: []string{"type"},
		},
		{
			name: "missing step name",
			config: &Config{
				Name:  "test",
				Steps: []StepConfig{{Type: "noop"}},
			},
			wantErrors: 1,
			wantFields: []string{"name"},
		},
		{
			name: "duplicate step names",
			config: &Config{
				Name: "test",
				Steps: []StepConfig{
					{Type: "noop", Name: "same"},
					{Type: "command", Name: "same"},
				},
			},
			wantErrors: 1,
			wantFields: []string{"name"},
		},
		{
			name: "multiple errors",
			config: &Config{
				Steps: []StepConfig{
					{}, // missing type and name
				},
			},
			wantErrors: 3, // missing config name, step type, step name
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := validator.Validate(tt.config)

			if len(errs) != tt.wantErrors {
				t.Errorf("got %d errors, want %d: %v", len(errs), tt.wantErrors, errs)
			}

			// Check that expected fields are in errors
			for _, field := range tt.wantFields {
				found := false
				for _, e := range errs {
					if e.Field == field {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error for field %q, got errors: %v", field, errs)
				}
			}
		})
	}
}

func TestValidationErrorFormat(t *testing.T) {
	err := ValidationError{
		Field:   "type",
		Message: "step type is required",
		Context: "step[0]",
	}

	expected := "type: step type is required (in step[0])"
	if err.Error() != expected {
		t.Errorf("got %q, want %q", err.Error(), expected)
	}

	// Without context
	err.Context = ""
	expected = "type: step type is required"
	if err.Error() != expected {
		t.Errorf("got %q, want %q", err.Error(), expected)
	}
}

func TestValidateConfigConvenience(t *testing.T) {
	valid := &Config{
		Name:  "test",
		Steps: []StepConfig{{Type: "noop", Name: "test-step"}},
	}

	err := ValidateConfig(valid, []string{"noop"})
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	invalid := &Config{} // missing name and steps
	err = ValidateConfig(invalid, []string{"noop"})
	if err == nil {
		t.Error("expected validation error, got nil")
	}
}
