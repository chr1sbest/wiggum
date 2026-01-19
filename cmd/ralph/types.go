package main

import (
	"strconv"
	"strings"
)

type prdFile struct {
	Version jsonInt   `json:"version"`
	Tasks   []prdTask `json:"tasks"`
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

type prdTask struct {
	ID       string     `json:"id"`
	Title    string     `json:"title"`
	Details  string     `json:"details,omitempty"`
	Priority string     `json:"priority,omitempty"`
	Status   string     `json:"status,omitempty"`
	Tests    string     `json:"tests,omitempty"`
	Issue    *taskIssue `json:"issue,omitempty"`
}

type taskIssue struct {
	Number int    `json:"number"`
	URL    string `json:"url"`
}

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

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
