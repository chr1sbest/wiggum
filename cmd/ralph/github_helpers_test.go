package main

import (
	"testing"
)

func TestParseGitHubRepo(t *testing.T) {
	tests := []struct {
		name    string
		remote  string
		want    string
		wantErr bool
	}{
		{
			name:   "SSH format",
			remote: "git@github.com:owner/repo.git",
			want:   "owner/repo",
		},
		{
			name:   "SSH format without .git",
			remote: "git@github.com:owner/repo",
			want:   "owner/repo",
		},
		{
			name:   "HTTPS format",
			remote: "https://github.com/owner/repo.git",
			want:   "owner/repo",
		},
		{
			name:   "HTTPS format without .git",
			remote: "https://github.com/owner/repo",
			want:   "owner/repo",
		},
		{
			name:    "non-GitHub remote",
			remote:  "https://gitlab.com/owner/repo.git",
			wantErr: true,
		},
		{
			name:    "invalid format",
			remote:  "not-a-url",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseGitHubRepo(tt.remote)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseGitHubRepo() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("parseGitHubRepo() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("parseGitHubRepo() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseIssueURL(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		wantRepo   string
		wantNumber int
		wantNil    bool
	}{
		{
			name:       "valid issue URL",
			url:        "https://github.com/owner/repo/issues/42",
			wantRepo:   "owner/repo",
			wantNumber: 42,
		},
		{
			name:       "valid issue URL with trailing slash",
			url:        "https://github.com/owner/repo/issues/123/",
			wantRepo:   "owner/repo",
			wantNumber: 123,
		},
		{
			name:    "PR URL (not issue)",
			url:     "https://github.com/owner/repo/pull/42",
			wantNil: true,
		},
		{
			name:    "repo URL without issue",
			url:     "https://github.com/owner/repo",
			wantNil: true,
		},
		{
			name:    "non-GitHub URL",
			url:     "https://gitlab.com/owner/repo/issues/42",
			wantNil: true,
		},
		{
			name:    "not a URL",
			url:     "42",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseIssueURL(tt.url)
			if tt.wantNil {
				if got != nil {
					t.Errorf("parseIssueURL() = %+v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Errorf("parseIssueURL() = nil, want non-nil")
				return
			}
			if got.Repo != tt.wantRepo {
				t.Errorf("parseIssueURL().Repo = %q, want %q", got.Repo, tt.wantRepo)
			}
			if got.Number != tt.wantNumber {
				t.Errorf("parseIssueURL().Number = %d, want %d", got.Number, tt.wantNumber)
			}
		})
	}
}

func TestFormatIssueAsWork(t *testing.T) {
	issue := &GitHubIssue{
		Number: 42,
		Title:  "Fix the bug",
		Body:   "This is the bug description",
		Labels: []string{"bug", "high-priority"},
		URL:    "https://github.com/owner/repo/issues/42",
	}

	result := formatIssueAsWork(issue)

	if result == "" {
		t.Error("formatIssueAsWork() returned empty string")
	}

	// Check key parts are present
	if !contains(result, "# GitHub Issue #42") {
		t.Error("formatIssueAsWork() missing issue number header")
	}
	if !contains(result, "Fix the bug") {
		t.Error("formatIssueAsWork() missing title")
	}
	if !contains(result, "This is the bug description") {
		t.Error("formatIssueAsWork() missing body")
	}
	if !contains(result, "bug, high-priority") {
		t.Error("formatIssueAsWork() missing labels")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
