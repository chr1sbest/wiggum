package tracker

import "time"

type RunState struct {
	RunID              string    `json:"run_id"`
	PID                int       `json:"pid"`
	StartedAt          time.Time `json:"started_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	LoopNumber         int       `json:"loop_number"`
	CurrentStep        string    `json:"current_step,omitempty"`
	StepStartedAt      time.Time `json:"step_started_at,omitempty"`
	LastSuccessfulStep string    `json:"last_successful_step,omitempty"`
	Status             string    `json:"status"`
	LastError          string    `json:"last_error,omitempty"`
}
