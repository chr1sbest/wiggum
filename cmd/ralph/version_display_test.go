package main

import (
	"strings"
	"testing"
)

func TestVersionLine(t *testing.T) {
	oldVersion := version
	oldCommit := commit
	oldDate := date
	defer func() {
		version = oldVersion
		commit = oldCommit
		date = oldDate
	}()

	t.Run("release version", func(t *testing.T) {
		version = "v1.2.3"
		commit = "none"
		date = "unknown"
		got := versionLine()
		if got != "ralph version v1.2.3" {
			t.Fatalf("versionLine() = %q", got)
		}
	})

	t.Run("dev no metadata", func(t *testing.T) {
		version = "dev"
		commit = "none"
		date = "unknown"
		got := versionLine()
		if got != "ralph version dev" {
			t.Fatalf("versionLine() = %q", got)
		}
	})

	t.Run("dev commit only", func(t *testing.T) {
		version = "dev"
		commit = "abcdef012345"
		date = "unknown"
		got := versionLine()
		if got != "ralph version dev (commit abcdef0)" {
			t.Fatalf("versionLine() = %q", got)
		}
	})

	t.Run("dev date only", func(t *testing.T) {
		version = "dev"
		commit = "none"
		date = "2026-01-18T16:00:00Z"
		got := versionLine()
		if got != "ralph version dev (built 2026-01-18T16:00:00Z)" {
			t.Fatalf("versionLine() = %q", got)
		}
	})

	t.Run("dev commit and date", func(t *testing.T) {
		version = "dev"
		commit = "abcdef012345"
		date = "2026-01-18T16:00:00Z"
		got := versionLine()
		if !strings.Contains(got, "commit abcdef0") || !strings.Contains(got, "built 2026-01-18T16:00:00Z") {
			t.Fatalf("versionLine() = %q", got)
		}
	})
}
