package steps

import (
	"fmt"
	"strings"
	"time"

	"github.com/chr1sbest/wiggum/internal/agent"
)

// refreshStatus updates the terminal status display with animated dots
func (s *AgentStep) refreshStatus(prdFile string, _ bool) {
	prdStatus, _ := agent.LoadPRDStatus(prdFile)
	completed := 0
	total := 0
	current := ""
	if prdStatus != nil {
		completed = prdStatus.CompletedTasks
		total = prdStatus.TotalTasks
		current = prdStatus.CurrentTask
	}

	// Progress bar
	const barWidth = 20
	filled := 0
	if total > 0 {
		filled = (completed * barWidth) / total
	}
	bar := "\033[32m" + strings.Repeat("█", filled) + "\033[0m" +
		"\033[2m" + strings.Repeat("░", barWidth-filled) + "\033[0m"

	// Animated dots
	phase := (time.Now().Unix()) % 3
	dots := strings.Repeat(".", int(phase)+1)

	// Build status line
	var line1, line2 string
	if current != "" {
		line1 = fmt.Sprintf("%s \033[2m%d/%d\033[0m %s", bar, completed, total, current)
	} else {
		line1 = fmt.Sprintf("%s \033[2m%d/%d\033[0m", bar, completed, total)
	}
	if completed == total && total > 0 {
		line2 = fmt.Sprintf("\033[32m\033[1mWrapping up%s\033[0m", dots)
	} else {
		line2 = fmt.Sprintf("\033[32m\033[1mWorking%s\033[0m", dots)
	}

	// Clear and rewrite (2 lines) - loop.go already printed initial status
	fmt.Print("\033[A\033[2K\033[A\033[2K\r")
	fmt.Println(line1)
	fmt.Println(line2)
}
