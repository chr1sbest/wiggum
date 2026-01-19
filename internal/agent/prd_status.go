package agent

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
)

func stripJSONFences(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		lines := strings.Split(s, "\n")
		if len(lines) >= 2 {
			lines = lines[1:]
		}
		if len(lines) > 0 {
			last := strings.TrimSpace(lines[len(lines)-1])
			if last == "```" {
				lines = lines[:len(lines)-1]
			}
		}
		s = strings.TrimSpace(strings.Join(lines, "\n"))
	}
	return s
}

type prdFile struct {
	Version jsonInt       `json:"version"`
	Tasks   []prdFileTask `json:"tasks"`
}

type jsonInt int

func (i *jsonInt) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	if s == "" || s == "null" {
		*i = 0
		return nil
	}
	if strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\"") {
		unq, err := strconv.Unquote(s)
		if err != nil {
			return err
		}
		s = strings.TrimSpace(unq)
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return err
	}
	*i = jsonInt(n)
	return nil
}

type prdFileTask struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Details  string `json:"details,omitempty"`
	Priority string `json:"priority,omitempty"`
	Status   string `json:"status,omitempty"`
}

// PRDStatus is a lightweight view of prd.json used for progress display and exit detection.
// Task selection and task updates are intentionally left to the LLM prompts.
type PRDStatus struct {
	TotalTasks      int
	CompletedTasks  int
	IncompleteTasks int
	TodoTasks       int
	FailedTasks     int
	CurrentTaskID   string
	CurrentTask     string
}

func (s *PRDStatus) IsComplete() bool {
	return s.TotalTasks > 0 && s.IncompleteTasks == 0
}

// HasActionableTasks returns true if there are tasks that can be worked on (status "todo")
func (s *PRDStatus) HasActionableTasks() bool {
	return s.TodoTasks > 0
}

func (s *PRDStatus) Progress() string {
	if s.TotalTasks == 0 {
		return "0/0"
	}
	return itoa(s.CompletedTasks) + "/" + itoa(s.TotalTasks)
}

// LoadPRDStatus reads prd.json and returns counts + current in-progress task (if any).
func LoadPRDStatus(path string) (*PRDStatus, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &PRDStatus{}, nil
		}
		return nil, err
	}
	clean := stripJSONFences(string(b))
	if clean == "" {
		return &PRDStatus{}, nil
	}

	var f prdFile
	if err := json.Unmarshal([]byte(clean), &f); err != nil {
		return nil, err
	}

	st := &PRDStatus{}
	st.TotalTasks = len(f.Tasks)
	for _, t := range f.Tasks {
		status := strings.ToLower(strings.TrimSpace(t.Status))
		id := strings.TrimSpace(t.ID)
		title := strings.TrimSpace(t.Title)
		switch status {
		case "done":
			st.CompletedTasks++
		case "todo":
			st.TodoTasks++
			st.IncompleteTasks++
		case "failed":
			st.FailedTasks++
			st.IncompleteTasks++
		default: // in_progress or other
			st.IncompleteTasks++
		}
		if st.CurrentTask == "" && status == "in_progress" && title != "" {
			st.CurrentTaskID = id
			st.CurrentTask = title
		}
	}
	if st.CurrentTask == "" {
		for _, t := range f.Tasks {
			status := strings.ToLower(strings.TrimSpace(t.Status))
			id := strings.TrimSpace(t.ID)
			title := strings.TrimSpace(t.Title)
			if status == "todo" && title != "" {
				st.CurrentTaskID = id
				st.CurrentTask = title
				break
			}
		}
	}

	return st, nil
}

// ResetFailedTasks changes all "failed" tasks back to "todo" so they can be retried.
// Returns the number of tasks reset.
func ResetFailedTasks(path string) (int, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	clean := stripJSONFences(string(b))
	if clean == "" {
		return 0, nil
	}

	var f prdFile
	if err := json.Unmarshal([]byte(clean), &f); err != nil {
		return 0, err
	}

	count := 0
	for i := range f.Tasks {
		if strings.ToLower(strings.TrimSpace(f.Tasks[i].Status)) == "failed" {
			f.Tasks[i].Status = "todo"
			count++
		}
	}

	if count == 0 {
		return 0, nil
	}

	out, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return 0, err
	}
	return count, os.WriteFile(path, out, 0644)
}

// MarkTaskFailed updates the status of a task to "failed" in prd.json.
// This prevents the task from being picked up again.
func MarkTaskFailed(path, taskID string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	clean := stripJSONFences(string(b))
	if clean == "" {
		return nil
	}

	var f prdFile
	if err := json.Unmarshal([]byte(clean), &f); err != nil {
		return err
	}

	// Find and update the task
	for i := range f.Tasks {
		if strings.TrimSpace(f.Tasks[i].ID) == strings.TrimSpace(taskID) {
			f.Tasks[i].Status = "failed"
			break
		}
	}

	// Write back
	out, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0644)
}
