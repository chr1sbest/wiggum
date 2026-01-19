package agent

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

// Task represents a single task from @fix_plan.md
type Task struct {
	Description string
	Completed   bool
	LineNumber  int
}

// FixPlanStatus holds the parsed state of @fix_plan.md
type FixPlanStatus struct {
	TotalTasks      int
	CompletedTasks  int
	IncompleteTasks int
	NextTask        string
	Tasks           []Task
}

var (
	// Matches "- [ ] task description" or "- [x] task description"
	taskPattern = regexp.MustCompile(`^[\s]*-\s*\[([ xX])\]\s*(.+)$`)
)

// ParseFixPlan reads and parses @fix_plan.md
func ParseFixPlan(path string) (*FixPlanStatus, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			// No fix plan is OK - return empty status
			return &FixPlanStatus{}, nil
		}
		return nil, err
	}
	defer file.Close()

	status := &FixPlanStatus{
		Tasks: make([]Task, 0),
	}

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		matches := taskPattern.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		checkbox := matches[1]
		description := strings.TrimSpace(matches[2])

		task := Task{
			Description: description,
			Completed:   checkbox == "x" || checkbox == "X",
			LineNumber:  lineNum,
		}

		status.Tasks = append(status.Tasks, task)
		status.TotalTasks++

		if task.Completed {
			status.CompletedTasks++
		} else {
			status.IncompleteTasks++
			// Track first incomplete task
			if status.NextTask == "" {
				status.NextTask = description
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return status, nil
}

// IsComplete returns true if all tasks are done
func (s *FixPlanStatus) IsComplete() bool {
	return s.TotalTasks > 0 && s.IncompleteTasks == 0
}

// Progress returns a string like "3/10"
func (s *FixPlanStatus) Progress() string {
	if s.TotalTasks == 0 {
		return "0/0"
	}
	return strings.TrimSpace(strings.Join([]string{
		itoa(s.CompletedTasks),
		"/",
		itoa(s.TotalTasks),
	}, ""))
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	result := ""
	for i > 0 {
		result = string(rune('0'+i%10)) + result
		i /= 10
	}
	return result
}
