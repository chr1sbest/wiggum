package agent

// ExitReason provides a human-readable exit reason
type ExitReason string

const (
	ExitReasonNone              ExitReason = ""
	ExitReasonPlanComplete      ExitReason = "plan_complete"
	ExitReasonNoProgress        ExitReason = "no_progress"
	ExitReasonNoActionableTasks ExitReason = "no_actionable_tasks"
)

// ExitDetector tracks exit conditions across loops
type ExitDetector struct {
	consecutiveNoProgress int
	lastCompletedCount    int // Track prd.json completed task count

	// Thresholds
	NoProgressThreshold int
}

// NewExitDetector creates an exit detector with default thresholds
func NewExitDetector() *ExitDetector {
	return &ExitDetector{
		NoProgressThreshold: 2,
	}
}

// Check evaluates exit conditions and returns a reason if should exit.
// planComplete should reflect whether all tasks in the current plan (prd.json) are done.
// completedCount is the current number of completed tasks from prd.json.
func (d *ExitDetector) Check(planComplete bool, completedCount int) ExitReason {
	// Check if plan is complete
	if planComplete {
		return ExitReasonPlanComplete
	}

	// Track no progress based on prd.json task completion
	// Only count as no-progress if we've seen this count before (prevents double-counting per loop)
	hasPrdProgress := completedCount > d.lastCompletedCount
	if hasPrdProgress {
		d.lastCompletedCount = completedCount
		d.consecutiveNoProgress = 0
	}
	// Note: consecutiveNoProgress is incremented by MarkLoopComplete, not here

	if d.consecutiveNoProgress >= d.NoProgressThreshold {
		return ExitReasonNoProgress
	}

	return ExitReasonNone
}

// MarkLoopComplete should be called once per loop iteration to track no-progress
func (d *ExitDetector) MarkLoopComplete(completedCount int) {
	if completedCount <= d.lastCompletedCount {
		d.consecutiveNoProgress++
	}
}

// Reset clears all counters
func (d *ExitDetector) Reset() {
	d.consecutiveNoProgress = 0
	d.lastCompletedCount = 0
}

// SetInitialCompletedCount sets the baseline for tracking progress
func (d *ExitDetector) SetInitialCompletedCount(count int) {
	d.lastCompletedCount = count
}
