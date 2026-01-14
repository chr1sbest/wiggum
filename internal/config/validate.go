package config

import (
	"fmt"
	"strings"
)

// ValidationError holds details about a configuration validation failure.
type ValidationError struct {
	Field   string
	Message string
	Context string
}

func (e ValidationError) Error() string {
	if e.Context != "" {
		return fmt.Sprintf("%s: %s (in %s)", e.Field, e.Message, e.Context)
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationErrors collects multiple validation errors.
type ValidationErrors []ValidationError

func (errs ValidationErrors) Error() string {
	if len(errs) == 0 {
		return "no validation errors"
	}
	var msgs []string
	for _, e := range errs {
		msgs = append(msgs, "  - "+e.Error())
	}
	return fmt.Sprintf("validation failed with %d error(s):\n%s", len(errs), strings.Join(msgs, "\n"))
}

// HasErrors returns true if there are any validation errors.
func (errs ValidationErrors) HasErrors() bool {
	return len(errs) > 0
}

// Validator validates configuration files.
type Validator struct {
	knownStepTypes []string
}

// NewValidator creates a new config validator.
func NewValidator(knownStepTypes []string) *Validator {
	return &Validator{knownStepTypes: knownStepTypes}
}

// Validate checks a config for errors and returns detailed validation errors.
func (v *Validator) Validate(cfg *Config) ValidationErrors {
	var errs ValidationErrors

	// Validate config name
	if cfg.Name == "" {
		errs = append(errs, ValidationError{
			Field:   "name",
			Message: "config name is required",
		})
	}

	// Validate steps
	if len(cfg.Steps) == 0 {
		errs = append(errs, ValidationError{
			Field:   "steps",
			Message: "at least one step is required",
		})
	}

	// Track step names for duplicate detection
	seenNames := make(map[string]bool)

	for i, step := range cfg.Steps {
		stepContext := fmt.Sprintf("step[%d]", i)

		// Validate step type
		if step.Type == "" {
			errs = append(errs, ValidationError{
				Field:   "type",
				Message: "step type is required",
				Context: stepContext,
			})
		} else if len(v.knownStepTypes) > 0 && !v.isKnownType(step.Type) {
			errs = append(errs, ValidationError{
				Field:   "type",
				Message: fmt.Sprintf("unknown step type %q, known types: %s", step.Type, strings.Join(v.knownStepTypes, ", ")),
				Context: stepContext,
			})
		}

		// Validate step name
		if step.Name == "" {
			errs = append(errs, ValidationError{
				Field:   "name",
				Message: "step name is required",
				Context: stepContext,
			})
		} else {
			// Check for duplicate names
			if seenNames[step.Name] {
				errs = append(errs, ValidationError{
					Field:   "name",
					Message: fmt.Sprintf("duplicate step name %q", step.Name),
					Context: stepContext,
				})
			}
			seenNames[step.Name] = true
		}
	}

	return errs
}

func (v *Validator) isKnownType(stepType string) bool {
	for _, t := range v.knownStepTypes {
		if t == stepType {
			return true
		}
	}
	return false
}

// ValidateConfig is a convenience function to validate a config with known step types.
func ValidateConfig(cfg *Config, knownStepTypes []string) error {
	validator := NewValidator(knownStepTypes)
	errs := validator.Validate(cfg)
	if errs.HasErrors() {
		return errs
	}
	return nil
}
