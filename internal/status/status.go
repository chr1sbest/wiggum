package status

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/chr1sbest/wiggum/internal/agent"
)

// ANSI escape codes
const (
	clearLine  = "\033[2K"
	moveUp     = "\033[A"
	moveToCol0 = "\r"
	reset      = "\033[0m"
	bold       = "\033[1m"
	dim        = "\033[2m"
	green      = "\033[32m"
	yellow     = "\033[33m"
	cyan       = "\033[36m"
	red        = "\033[31m"
)

// Progress bar characters
const (
	barFilled = "█"
	barEmpty  = "░"
	barWidth  = 20
)

// Writer handles in-place status updates to the terminal
type Writer struct {
	w            io.Writer
	mu           sync.Mutex
	linesWritten int
	startTime    time.Time
}

// animatedDots returns a cycling dot pattern based on elapsed time
func animatedDots() string {
	ms := time.Now().UnixMilli()
	phase := (ms / 500) % 3 // cycle every 500ms through 3 phases
	switch phase {
	case 0:
		return "."
	case 1:
		return ".."
	default:
		return "..."
	}
}

// New creates a status writer that outputs to stdout
func New() *Writer {
	return &Writer{w: os.Stdout}
}

// NewWithWriter creates a status writer with a custom output
func NewWithWriter(w io.Writer) *Writer {
	return &Writer{w: w}
}

// Clear erases any previously written status lines
func (s *Writer) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := 0; i < s.linesWritten; i++ {
		fmt.Fprint(s.w, moveUp+clearLine)
	}
	fmt.Fprint(s.w, moveToCol0)
	s.linesWritten = 0
}

// Update clears previous status and writes new status
func (s *Writer) Update(lines ...string) {
	s.Clear()
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, line := range lines {
		fmt.Fprintln(s.w, line)
	}
	s.linesWritten = len(lines)
}

// progressBar generates a progress bar string
func progressBar(completed, total int) string {
	if total == 0 {
		return strings.Repeat(barEmpty, barWidth)
	}

	filled := (completed * barWidth) / total
	if filled > barWidth {
		filled = barWidth
	}

	return green + strings.Repeat(barFilled, filled) + reset +
		dim + strings.Repeat(barEmpty, barWidth-filled) + reset
}

// Step displays the current step with progress bar
func (s *Writer) Step(loopNum, stepNum, totalSteps int, stepName string) {
	s.StepWithRetry(loopNum, stepNum, totalSteps, stepName, 0, 0)
}

// StepWithRetry displays step with retry information
func (s *Writer) StepWithRetry(loopNum, stepNum, totalSteps int, stepName string, attempt, maxRetries int) {
	prdStatus, _ := agent.LoadPRDStatus(".ralph/prd.json")
	completed := 0
	total := 0
	current := ""
	if prdStatus != nil {
		completed = prdStatus.CompletedTasks
		total = prdStatus.TotalTasks
		current = prdStatus.CurrentTask
	}
	bar := progressBar(completed, total)
	dots := animatedDots()

	_ = loopNum
	_ = stepNum
	_ = totalSteps
	_ = stepName
	_ = attempt
	_ = maxRetries

	var lines []string
	if current != "" {
		lines = []string{
			fmt.Sprintf("%s %s%d/%d%s %s%s%s", bar, dim, completed, total, reset, bold, current, reset),
			fmt.Sprintf("%s%sWorking%s%s", green, bold, dots, reset),
		}
	} else if completed == total && total > 0 {
		lines = []string{
			fmt.Sprintf("%s %s%d/%d%s", bar, dim, completed, total, reset),
			fmt.Sprintf("%s%sWrapping up%s%s", green, bold, dots, reset),
		}
	} else {
		lines = []string{
			fmt.Sprintf("%s %s%d/%d%s", bar, dim, completed, total, reset),
			fmt.Sprintf("%s%sWorking%s%s", green, bold, dots, reset),
		}
	}

	s.Update(lines...)
}

// Complete shows completion status
func (s *Writer) Complete(loopNum, totalSteps int) {
	prdStatus, _ := agent.LoadPRDStatus(".ralph/prd.json")
	completed := totalSteps
	total := totalSteps
	if prdStatus != nil {
		completed = prdStatus.CompletedTasks
		total = prdStatus.TotalTasks
	}

	bar := progressBar(completed, total)
	lines := []string{
		fmt.Sprintf("%s %s%d/%d%s", bar, dim, completed, total, reset),
		fmt.Sprintf("%s✓ Complete%s", green+bold, reset),
	}

	s.Update(lines...)
}

// Error shows error status
func (s *Writer) Error(loopNum, stepNum, totalSteps int, stepName string, err error) {
	s.Clear()
	s.mu.Lock()
	defer s.mu.Unlock()

	completed := stepNum - 1
	bar := progressBar(completed, totalSteps)

	_ = loopNum
	// Print error state (don't track - let it persist)
	fmt.Fprintln(s.w, fmt.Sprintf("%s %s%d/%d%s", bar, dim, stepNum, totalSteps, reset))
	fmt.Fprintln(s.w, fmt.Sprintf("%s✗ %s failed%s", red+bold, stepName, reset))
	fmt.Fprintln(s.w, fmt.Sprintf("%s%v%s", dim, err, reset))

	s.linesWritten = 0 // don't clear error messages
}

// Waiting shows waiting status between loops
func (s *Writer) Waiting(loopNum, totalSteps int) {
	bar := progressBar(totalSteps, totalSteps)
	_ = loopNum

	lines := []string{
		fmt.Sprintf("%s %s%d/%d%s", bar, dim, totalSteps, totalSteps, reset),
		fmt.Sprintf("%s⏳ Waiting for next iteration...%s", dim, reset),
		"",
	}

	s.Update(lines...)
}

// CircuitOpen shows when a step's circuit breaker has opened
func (s *Writer) CircuitOpen(loopNum, stepNum, totalSteps int, stepName string) {
	completed := stepNum - 1
	bar := progressBar(completed, totalSteps)
	_ = loopNum

	lines := []string{
		fmt.Sprintf("%s %s%d/%d%s", bar, dim, stepNum, totalSteps, reset),
		fmt.Sprintf("%s⚡ %s circuit open%s", yellow+bold, stepName, reset),
		fmt.Sprintf("%sSkipping due to recent failures%s", dim, reset),
	}

	s.Update(lines...)
}
