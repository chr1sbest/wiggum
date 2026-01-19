package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
)

var version = "dev"

var commit = "none"

var date = "unknown"

func versionLine() string {
	installMethod := detectInstallMethod()
	if version != "dev" {
		if installMethod != "" {
			return fmt.Sprintf("ralph version %s (%s)", version, installMethod)
		}
		return fmt.Sprintf("ralph version %s", version)
	}

	c := strings.TrimSpace(commit)
	d := strings.TrimSpace(date)

	if (c == "" || c == "none") || (d == "" || d == "unknown") {
		if bi, ok := debug.ReadBuildInfo(); ok {
			for _, s := range bi.Settings {
				switch s.Key {
				case "vcs.revision":
					if (c == "" || c == "none") && strings.TrimSpace(s.Value) != "" {
						c = strings.TrimSpace(s.Value)
					}
				case "vcs.time":
					if (d == "" || d == "unknown") && strings.TrimSpace(s.Value) != "" {
						d = strings.TrimSpace(s.Value)
					}
				}
			}
		}
	}

	if c != "" && c != "none" {
		if len(c) > 7 {
			c = c[:7]
		}
	}

	var suffix string
	if installMethod != "" {
		suffix = ", " + installMethod
	}

	if (c == "" || c == "none") && (d == "" || d == "unknown") {
		if installMethod != "" {
			return fmt.Sprintf("ralph version dev (%s)", installMethod)
		}
		return "ralph version dev"
	}
	if c == "" || c == "none" {
		return fmt.Sprintf("ralph version dev (built %s%s)", d, suffix)
	}
	if d == "" || d == "unknown" {
		return fmt.Sprintf("ralph version dev (commit %s%s)", c, suffix)
	}
	return fmt.Sprintf("ralph version dev (commit %s, built %s%s)", c, d, suffix)
}

func compareSemver(a, b string) int {
	pa := parseSemver(a)
	pb := parseSemver(b)
	for i := 0; i < 3; i++ {
		if pa[i] > pb[i] {
			return 1
		}
		if pa[i] < pb[i] {
			return -1
		}
	}
	return 0
}

func parseSemver(v string) [3]int {
	v = strings.TrimSpace(strings.TrimPrefix(v, "v"))
	parts := strings.Split(v, ".")
	out := [3]int{}
	for i := 0; i < 3 && i < len(parts); i++ {
		n, err := strconv.Atoi(strings.TrimSpace(parts[i]))
		if err != nil {
			return [3]int{}
		}
		out[i] = n
	}
	return out
}

func detectInstallMethod() string {
	exePath, err := os.Executable()
	if err != nil {
		return ""
	}
	if resolved, err := filepath.EvalSymlinks(exePath); err == nil {
		exePath = resolved
	}
	if looksLikeHomebrewInstall(exePath) {
		return "brew"
	}
	exeDir := filepath.Dir(exePath)
	if looksLikeGoInstall(exeDir) {
		return "go install"
	}
	return ""
}
