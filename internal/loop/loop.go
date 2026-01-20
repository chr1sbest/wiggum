package loop

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/chr1sbest/wiggum/internal/agent"
	"github.com/chr1sbest/wiggum/internal/config"
	"github.com/chr1sbest/wiggum/internal/logger"
	"github.com/chr1sbest/wiggum/internal/loop/steps"
	"github.com/chr1sbest/wiggum/internal/resilience"
	"github.com/chr1sbest/wiggum/internal/status"
	"github.com/chr1sbest/wiggum/internal/tracker"
)

// Status represents the current state of the loop.
type Status string

const (
	StatusRunning  Status = "RUNNING"
	StatusComplete Status = "COMPLETE"
	StatusBlocked  Status = "BLOCKED"
	StatusError    Status = "ERROR"
)

// State holds the current loop execution state.
type State struct {
	LoopNumber    int
	StartTime     time.Time
	CurrentStep   string
	PreviousStep  string
	FilesModified int
	TestsStatus   string
	Status        Status
}

// StepResult captures the outcome of a step execution.
type StepResult struct {
	StepName     string
	Success      bool
	Duration     time.Duration
	Output       string
	Error        error
	RetryAttempt int  // Number of retries attempted
	CircuitOpen  bool // True if skipped due to open circuit
}

// Loop is the main execution engine.
type Loop struct {
	config          *config.Config
	registry        *StepRegistry
	logger          logger.Logger
	status          *status.Writer
	state           State
	stepDelay       time.Duration
	circuitBreakers *resilience.CircuitBreakerRegistry
	trackerWriter   *tracker.Writer
	runID           string
	runStartedAt    time.Time

	// Task loop tracking for max_loops_per_task
	currentTaskID string
	loopsOnTask   int
	prdPath       string // Path to prd.json for marking tasks failed
}

// NewLoop creates a new loop executor.
func NewLoop(cfg *config.Config, registry *StepRegistry, log logger.Logger) *Loop {
	return &Loop{
		config:          cfg,
		registry:        registry,
		logger:          log,
		status:          status.New(),
		stepDelay:       500 * time.Millisecond,
		circuitBreakers: resilience.NewCircuitBreakerRegistry(resilience.DefaultCircuitBreakerConfig()),
		state: State{
			Status:      StatusRunning,
			TestsStatus: "NOT_RUN",
		},
	}
}

// EnableRunTracking enables writing run_state.json to the given directory.
// This is used to support abrupt stop + restart by persisting in-flight state.
func (l *Loop) EnableRunTracking(runID, dir string) {
	l.trackerWriter = tracker.NewWriter(dir)
	l.runID = runID
	l.runStartedAt = time.Now()
}

func (l *Loop) writeRunState(status string, currentStep string, stepStartedAt time.Time, lastSuccessfulStep string, lastErr error) {
	if l.trackerWriter == nil {
		return
	}

	rs := tracker.RunState{
		RunID:              l.runID,
		PID:                os.Getpid(),
		StartedAt:          l.runStartedAt,
		UpdatedAt:          time.Now(),
		LoopNumber:         l.state.LoopNumber,
		CurrentStep:        currentStep,
		StepStartedAt:      stepStartedAt,
		LastSuccessfulStep: lastSuccessfulStep,
		Status:             status,
	}
	if lastErr != nil {
		rs.LastError = lastErr.Error()
	}

	// Add current task info from prd.json
	if l.prdPath != "" {
		if prdStatus, _ := agent.LoadPRDStatus(l.prdPath); prdStatus != nil {
			rs.CurrentTaskID = prdStatus.CurrentTaskID
			rs.CurrentTask = prdStatus.CurrentTask
		}
	}

	_ = l.trackerWriter.WriteRunState(rs)
}

// SetStepDelay sets the delay between steps.
func (l *Loop) SetStepDelay(d time.Duration) {
	l.stepDelay = d
}

// SetPRDPath sets the path to prd.json for task tracking.
func (l *Loop) SetPRDPath(path string) {
	l.prdPath = path
}

// SetConfig updates the loop configuration (for hot-reload).
func (l *Loop) SetConfig(cfg *config.Config) {
	l.config = cfg
}

// State returns the current loop state.
func (l *Loop) State() State {
	return l.state
}

// RunOnce executes all steps in the config once.
func (l *Loop) RunOnce(ctx context.Context) error {
	l.state.LoopNumber++
	l.state.StartTime = time.Now()
	l.state.Status = StatusRunning

	// Mark loop start
	l.writeRunState("running", l.state.CurrentStep, time.Time{}, l.state.PreviousStep, nil)

	l.logger.Debug("Starting loop iteration", logger.F("loop", l.state.LoopNumber))

	// Count enabled steps for progress display
	enabledSteps := l.countEnabledSteps()
	stepNum := 0

	for _, stepCfg := range l.config.Steps {
		if !stepCfg.IsEnabled() {
			l.logger.Debug("Skipping disabled step", logger.F("step", stepCfg.Name))
			continue
		}

		select {
		case <-ctx.Done():
			l.state.Status = StatusBlocked
			l.writeRunState("blocked", l.state.CurrentStep, time.Time{}, l.state.PreviousStep, ctx.Err())
			return ctx.Err()
		default:
		}

		stepNum++
		stepStart := time.Now()
		l.writeRunState("running", stepCfg.Name, stepStart, l.state.PreviousStep, nil)

		// Update status display
		l.status.Step(l.state.LoopNumber, stepNum, enabledSteps, stepCfg.Name)

		result := l.executeStepWithResilience(ctx, stepCfg, stepNum, enabledSteps)

		if result.CircuitOpen {
			// Step was skipped due to open circuit, continue to next step
			l.logger.Debug("Step skipped (circuit open)",
				logger.F("step", stepCfg.Name),
			)
			continue
		}

		if !result.Success {
			// Graceful completion signaled by the agent step.
			if exitErr, ok := steps.IsAgentExitError(result.Error); ok {
				l.state.Status = StatusComplete
				l.status.Complete(l.state.LoopNumber, enabledSteps)
				l.writeRunState("complete", l.state.CurrentStep, time.Time{}, l.state.CurrentStep, nil)
				return exitErr
			}

			if stepCfg.ContinueOnError {
				l.logger.Debug("Step failed but continuing",
					logger.F("step", stepCfg.Name),
					logger.F("error", result.Error),
				)
				l.writeRunState("error", stepCfg.Name, stepStart, l.state.PreviousStep, result.Error)
				continue
			}

			l.state.Status = StatusError
			l.status.Error(l.state.LoopNumber, stepNum, enabledSteps, stepCfg.Name, result.Error)
			l.writeRunState("error", stepCfg.Name, stepStart, l.state.PreviousStep, result.Error)
			l.logger.Debug("Step failed",
				logger.F("step", stepCfg.Name),
				logger.F("error", result.Error),
				logger.F("retries", result.RetryAttempt),
			)
			return result.Error
		}

		l.state.PreviousStep = l.state.CurrentStep
		l.state.CurrentStep = stepCfg.Name
		l.writeRunState("running", l.state.CurrentStep, stepStart, l.state.PreviousStep, nil)

		// Delay between steps
		if l.stepDelay > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(l.stepDelay):
			}
		}
	}

	l.state.Status = StatusComplete
	l.status.Complete(l.state.LoopNumber, enabledSteps)
	l.writeRunState("complete", l.state.CurrentStep, time.Time{}, l.state.CurrentStep, nil)
	l.logger.Debug("Loop iteration complete",
		logger.F("loop", l.state.LoopNumber),
		logger.F("duration", time.Since(l.state.StartTime)),
	)

	return nil
}

func (l *Loop) countEnabledSteps() int {
	count := 0
	for _, s := range l.config.Steps {
		if s.IsEnabled() {
			count++
		}
	}
	return count
}

// Run executes the loop continuously until context is cancelled.
func (l *Loop) Run(ctx context.Context) error {
	backoff := l.stepDelay
	const maxBackoff = 30 * time.Second

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Preflight: check if all tasks are complete before running Claude
		if l.prdPath != "" {
			prdStatus, _ := agent.LoadPRDStatus(l.prdPath)
			if prdStatus != nil && prdStatus.TotalTasks > 0 && !prdStatus.HasActionableTasks() {
				l.state.Status = StatusComplete
				l.status.Complete(l.state.LoopNumber, l.countEnabledSteps())
				l.writeRunState("complete", "", time.Time{}, "", nil)
				return &steps.AgentExitError{Reason: agent.ExitReasonNoActionableTasks}
			}
		}

		// Check max_loops_per_task limit before running
		if l.config.MaxLoopsPerTask > 0 && l.prdPath != "" {
			prdStatus, _ := agent.LoadPRDStatus(l.prdPath)
			if prdStatus != nil && prdStatus.CurrentTaskID != "" {
				// Track which task we're working on
				if prdStatus.CurrentTaskID != l.currentTaskID {
					// New task, reset counter
					l.currentTaskID = prdStatus.CurrentTaskID
					l.loopsOnTask = 0
				}
				l.loopsOnTask++

				// Check if we've exceeded max loops for this task
				if l.loopsOnTask > l.config.MaxLoopsPerTask {
					l.logger.Debug("Max loops per task reached, marking task as failed",
						logger.F("task_id", l.currentTaskID),
						logger.F("loops", l.loopsOnTask),
						logger.F("max", l.config.MaxLoopsPerTask),
					)
					// Print visible notification
					fmt.Printf("\n⚠️  Task %s failed after %d loops - moving to next task\n", l.currentTaskID, l.config.MaxLoopsPerTask)
					if err := agent.MarkTaskFailed(l.prdPath, l.currentTaskID); err != nil {
						l.logger.Debug("Failed to mark task as failed", logger.F("error", err))
					}
					// Reset counter and continue to next task
					l.currentTaskID = ""
					l.loopsOnTask = 0
					continue
				}
			}
		}

		if err := l.RunOnce(ctx); err != nil {
			// Graceful completion signaled by the agent step should stop the loop.
			if exitErr, ok := steps.IsAgentExitError(err); ok {
				return exitErr
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			l.logger.Debug("Loop iteration failed", logger.F("error", err), logger.F("backoff", backoff))
			// On error, wait with exponential backoff before retrying
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
			// Increase backoff for next failure (capped at maxBackoff)
			backoff = time.Duration(float64(backoff) * 1.5)
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		} else {
			// Reset backoff on success
			backoff = l.stepDelay
		}
	}
}

// executeStepWithResilience executes a step with retry and circuit breaker support.
func (l *Loop) executeStepWithResilience(ctx context.Context, stepCfg config.StepConfig, stepNum, totalSteps int) StepResult {
	start := time.Now()

	// Get step implementation
	step, err := l.registry.Get(stepCfg.Type)
	if err != nil {
		return StepResult{
			StepName: stepCfg.Name,
			Success:  false,
			Duration: time.Since(start),
			Error:    err,
		}
	}

	// Get or create circuit breaker for this step
	var cbConfig *resilience.CircuitBreakerConfig
	if stepCfg.CircuitBreaker != nil {
		cbConfig = &resilience.CircuitBreakerConfig{
			Threshold:  stepCfg.CircuitBreaker.Threshold,
			ResetAfter: stepCfg.GetCircuitBreakerResetAfter(),
		}
	}
	cb := l.circuitBreakers.Get(stepCfg.Name, cbConfig)

	// Check circuit breaker state
	if cb.State() == resilience.CircuitOpen {
		l.status.CircuitOpen(l.state.LoopNumber, stepNum, totalSteps, stepCfg.Name)
		return StepResult{
			StepName:    stepCfg.Name,
			Success:     false,
			Duration:    time.Since(start),
			CircuitOpen: true,
		}
	}

	// Build retry config
	retryCfg := resilience.RetryConfig{
		MaxRetries: stepCfg.MaxRetries,
		InitDelay:  stepCfg.GetRetryDelay(),
		MaxDelay:   30 * time.Second,
		Multiplier: 2.0,
		Jitter:     0.1,
	}

	var retryAttempt int
	var lastErr error

	// Execute step - check for AgentExitError which is a success signal, not a failure
	execFunc := func(execCtx context.Context) error {
		err := step.Execute(execCtx, stepCfg.Config)
		// AgentExitError is a success signal (plan complete), not a failure to retry
		// Mark it as permanent so retry logic doesn't treat it as transient
		if _, isExit := steps.IsAgentExitError(err); isExit {
			return resilience.NewPermanentError(err)
		}
		return err
	}

	// Wrap with timeout if configured
	timeout := stepCfg.GetTimeout()
	if timeout > 0 {
		originalFunc := execFunc
		execFunc = func(execCtx context.Context) error {
			timeoutCtx, cancel := context.WithTimeout(execCtx, timeout)
			defer cancel()
			return originalFunc(timeoutCtx)
		}
	}

	// Execute through circuit breaker
	cbErr := cb.Execute(ctx, func(cbCtx context.Context) error {
		// Update display callback for retries
		callback := func(attempt int, err error, nextDelay time.Duration) {
			retryAttempt = attempt
			l.status.StepWithRetry(l.state.LoopNumber, stepNum, totalSteps, stepCfg.Name, attempt, stepCfg.MaxRetries)
			l.logger.Debug("Retrying step",
				logger.F("step", stepCfg.Name),
				logger.F("attempt", attempt),
				logger.F("error", err),
				logger.F("next_delay", nextDelay),
			)
		}

		lastErr = resilience.RetryWithCallback(cbCtx, retryCfg, execFunc, callback)
		return lastErr
	})

	// Handle circuit breaker open error
	if cbErr == resilience.ErrCircuitOpen {
		l.status.CircuitOpen(l.state.LoopNumber, stepNum, totalSteps, stepCfg.Name)
		return StepResult{
			StepName:    stepCfg.Name,
			Success:     false,
			Duration:    time.Since(start),
			CircuitOpen: true,
		}
	}

	l.logger.Debug("Executed step",
		logger.F("step", stepCfg.Name),
		logger.F("type", stepCfg.Type),
		logger.F("duration", time.Since(start)),
		logger.F("retries", retryAttempt),
	)

	return StepResult{
		StepName:     stepCfg.Name,
		Success:      cbErr == nil,
		Duration:     time.Since(start),
		Error:        cbErr,
		RetryAttempt: retryAttempt,
	}
}
