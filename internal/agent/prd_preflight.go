package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

var ErrPRDNoTasks = errors.New("prd.json contains no tasks")

// CheckPRDTasks validates that prd.json contains tasks and reports whether all tasks are complete.
func CheckPRDTasks(path string) (hasTasks bool, allComplete bool, err error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return false, false, err
	}

	clean := stripJSONFences(string(b))
	if strings.TrimSpace(clean) == "" {
		return false, false, ErrPRDNoTasks
	}

	var f prdFile
	if err := json.Unmarshal([]byte(clean), &f); err != nil {
		return false, false, fmt.Errorf("failed to parse prd.json: %w", err)
	}

	if len(f.Tasks) == 0 {
		return false, false, ErrPRDNoTasks
	}

	incomplete := 0
	for _, t := range f.Tasks {
		status := strings.ToLower(strings.TrimSpace(t.Status))
		if status != "done" {
			incomplete++
		}
	}

	return true, incomplete == 0, nil
}
