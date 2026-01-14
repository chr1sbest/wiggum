package agent

import (
	"regexp"
	"strconv"
	"strings"
)

// RalphStatus represents the parsed status block from Claude's output
type RalphStatus struct {
	Status                 string // IN_PROGRESS, COMPLETE, BLOCKED
	TasksCompletedThisLoop int
	FilesModified          int
	TestsStatus            string // PASSING, FAILING, NOT_RUN
	WorkType               string // IMPLEMENTATION, TESTING, DOCUMENTATION, REFACTORING
	ExitSignal             bool
	Recommendation         string
	Raw                    string // The raw status block
}

var (
	statusBlockPattern = regexp.MustCompile(`(?s)---RALPH_STATUS---(.+?)---END_RALPH_STATUS---`)
	statusLinePattern  = regexp.MustCompile(`^([A-Z_]+):\s*(.+)$`)
)

// ParseRalphStatus extracts and parses the RALPH_STATUS block from Claude's output
func ParseRalphStatus(output string) *RalphStatus {
	matches := statusBlockPattern.FindStringSubmatch(output)
	if matches == nil {
		return nil
	}

	status := &RalphStatus{
		Raw: matches[0],
	}

	// Parse each line in the status block
	lines := strings.Split(matches[1], "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		lineMatches := statusLinePattern.FindStringSubmatch(line)
		if lineMatches == nil {
			continue
		}

		key := lineMatches[1]
		value := strings.TrimSpace(lineMatches[2])

		switch key {
		case "STATUS":
			status.Status = value
		case "TASKS_COMPLETED_THIS_LOOP":
			if n, err := strconv.Atoi(value); err == nil {
				status.TasksCompletedThisLoop = n
			}
		case "FILES_MODIFIED":
			if n, err := strconv.Atoi(value); err == nil {
				status.FilesModified = n
			}
		case "TESTS_STATUS":
			status.TestsStatus = value
		case "WORK_TYPE":
			status.WorkType = value
		case "EXIT_SIGNAL":
			status.ExitSignal = strings.ToLower(value) == "true"
		case "RECOMMENDATION":
			status.Recommendation = value
		}
	}

	return status
}

// ShouldExit determines if the loop should exit based on this status
func (s *RalphStatus) ShouldExit() bool {
	if s == nil {
		return false
	}
	return s.ExitSignal || s.Status == "COMPLETE"
}

// HasProgress indicates if meaningful work was done
func (s *RalphStatus) HasProgress() bool {
	if s == nil {
		return false
	}
	return s.FilesModified > 0 || s.TasksCompletedThisLoop > 0
}

// ExitReason provides a human-readable exit reason
type ExitReason string

const (
	ExitReasonNone           ExitReason = ""
	ExitReasonExitSignal     ExitReason = "claude_exit_signal"
	ExitReasonPlanComplete   ExitReason = "plan_complete"
	ExitReasonStatusComplete ExitReason = "status_complete"
	ExitReasonNoProgress     ExitReason = "no_progress"
	ExitReasonTestSaturation ExitReason = "test_saturation"
)

// ExitDetector tracks exit conditions across loops
type ExitDetector struct {
	consecutiveNoProgress  int
	consecutiveTestLoops   int
	consecutiveDoneSignals int

	// Thresholds
	NoProgressThreshold int
	TestLoopThreshold   int
	DoneSignalThreshold int
}

// NewExitDetector creates an exit detector with default thresholds
func NewExitDetector() *ExitDetector {
	return &ExitDetector{
		NoProgressThreshold: 3,
		TestLoopThreshold:   3,
		DoneSignalThreshold: 2,
	}
}

// Check evaluates exit conditions and returns a reason if should exit.
// planComplete should reflect whether all tasks in the current plan (prd.json) are done.
func (d *ExitDetector) Check(status *RalphStatus, planComplete bool) ExitReason {
	// Check explicit exit signal from Claude
	if status != nil && status.ExitSignal {
		return ExitReasonExitSignal
	}

	// Check if plan is complete
	if planComplete {
		return ExitReasonPlanComplete
	}

	// Check status indicates complete
	if status != nil && status.Status == "COMPLETE" {
		d.consecutiveDoneSignals++
		if d.consecutiveDoneSignals >= d.DoneSignalThreshold {
			return ExitReasonStatusComplete
		}
	} else {
		d.consecutiveDoneSignals = 0
	}

	// Track no progress
	if status == nil || !status.HasProgress() {
		d.consecutiveNoProgress++
	} else {
		d.consecutiveNoProgress = 0
	}

	if d.consecutiveNoProgress >= d.NoProgressThreshold {
		return ExitReasonNoProgress
	}

	// Track test-only loops
	if status != nil && status.WorkType == "TESTING" {
		d.consecutiveTestLoops++
	} else {
		d.consecutiveTestLoops = 0
	}

	if d.consecutiveTestLoops >= d.TestLoopThreshold {
		return ExitReasonTestSaturation
	}

	return ExitReasonNone
}

// Reset clears all counters
func (d *ExitDetector) Reset() {
	d.consecutiveNoProgress = 0
	d.consecutiveTestLoops = 0
	d.consecutiveDoneSignals = 0
}
