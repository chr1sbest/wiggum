package loop

// Outcome represents the result of a step execution.
// This separates success signals from errors and provides clear retry semantics.
type Outcome struct {
	// Status indicates the outcome type
	Status OutcomeStatus

	// Error is set when Status is OutcomeError
	Error error

	// Reason provides context for Complete/Error outcomes
	Reason string
}

// OutcomeStatus represents the type of step outcome
type OutcomeStatus int

const (
	// OutcomeSuccess indicates the step completed successfully and the loop should continue
	OutcomeSuccess OutcomeStatus = iota

	// OutcomeComplete indicates the work is done and the loop should exit gracefully
	// This is a SUCCESS condition, not an error
	OutcomeComplete

	// OutcomeError indicates the step failed
	// The error may be retryable (transient) or permanent
	OutcomeError
)

// Success returns an outcome indicating the step succeeded
func Success() Outcome {
	return Outcome{Status: OutcomeSuccess}
}

// Complete returns an outcome indicating the plan is complete (success exit)
func Complete(reason string) Outcome {
	return Outcome{Status: OutcomeComplete, Reason: reason}
}

// Error returns an outcome indicating a failure
func Error(err error) Outcome {
	return Outcome{Status: OutcomeError, Error: err}
}

// IsSuccess returns true if the outcome is a success
func (o Outcome) IsSuccess() bool {
	return o.Status == OutcomeSuccess
}

// IsComplete returns true if the work is done (exit gracefully)
func (o Outcome) IsComplete() bool {
	return o.Status == OutcomeComplete
}

// IsError returns true if the step failed
func (o Outcome) IsError() bool {
	return o.Status == OutcomeError
}

// ShouldExit returns true if the loop should stop (either complete or fatal error)
func (o Outcome) ShouldExit() bool {
	return o.Status == OutcomeComplete
}
