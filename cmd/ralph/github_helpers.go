package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

type GitHubIssue struct {
	Number int      `json:"number"`
	Title  string   `json:"title"`
	Body   string   `json:"body"`
	State  string   `json:"state"`
	Labels []string `json:"labels"`
	URL    string   `json:"url"`
}

func getGitHubRepo() (string, error) {
	if _, err := exec.LookPath("git"); err != nil {
		return "", fmt.Errorf("git not found in PATH")
	}

	cmd := exec.Command("git", "remote", "get-url", "origin")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get git remote: %v", err)
	}

	remote := strings.TrimSpace(stdout.String())
	return parseGitHubRepo(remote)
}

func parseGitHubRepo(remote string) (string, error) {
	// Handle SSH: git@github.com:owner/repo.git
	sshRe := regexp.MustCompile(`git@github\.com:([^/]+)/(.+?)(\.git)?$`)
	if m := sshRe.FindStringSubmatch(remote); m != nil {
		return m[1] + "/" + strings.TrimSuffix(m[2], ".git"), nil
	}

	// Handle HTTPS: https://github.com/owner/repo.git
	httpsRe := regexp.MustCompile(`https://github\.com/([^/]+)/(.+?)(\.git)?$`)
	if m := httpsRe.FindStringSubmatch(remote); m != nil {
		return m[1] + "/" + strings.TrimSuffix(m[2], ".git"), nil
	}

	return "", fmt.Errorf("could not parse GitHub repo from remote: %s", remote)
}

func checkGitHubAuth() error {
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("gh CLI not found in PATH (install: https://cli.github.com)")
	}

	cmd := exec.Command("gh", "auth", "status")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gh CLI not authenticated: run 'gh auth login'")
	}
	return nil
}

func checkClaudeAvailable() error {
	if _, err := exec.LookPath("claude"); err != nil {
		return fmt.Errorf("claude not found in PATH")
	}

	// Quick check: run claude with minimal args to see if it's working
	cmd := exec.Command("claude", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("claude CLI not working: %v", err)
	}
	return nil
}

func fetchGitHubIssue(repo string, issueNum int) (*GitHubIssue, error) {
	cmd := exec.Command("gh", "issue", "view", fmt.Sprintf("%d", issueNum),
		"--repo", repo,
		"--json", "number,title,body,state,labels,url")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = err.Error()
		}
		return nil, fmt.Errorf("failed to fetch issue #%d: %s", issueNum, errMsg)
	}

	var raw struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		Body   string `json:"body"`
		State  string `json:"state"`
		Labels []struct {
			Name string `json:"name"`
		} `json:"labels"`
		URL string `json:"url"`
	}

	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		return nil, fmt.Errorf("failed to parse issue JSON: %v", err)
	}

	labels := make([]string, len(raw.Labels))
	for i, l := range raw.Labels {
		labels[i] = l.Name
	}

	return &GitHubIssue{
		Number: raw.Number,
		Title:  raw.Title,
		Body:   raw.Body,
		State:  raw.State,
		Labels: labels,
		URL:    raw.URL,
	}, nil
}

func formatIssueAsWork(issue *GitHubIssue) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# GitHub Issue #%d: %s\n\n", issue.Number, issue.Title))
	sb.WriteString(fmt.Sprintf("**URL:** %s\n", issue.URL))
	if len(issue.Labels) > 0 {
		sb.WriteString(fmt.Sprintf("**Labels:** %s\n", strings.Join(issue.Labels, ", ")))
	}
	sb.WriteString("\n## Description\n\n")
	if issue.Body != "" {
		sb.WriteString(issue.Body)
	} else {
		sb.WriteString("(No description provided)")
	}
	return sb.String()
}
