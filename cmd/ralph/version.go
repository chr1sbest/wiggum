package main

import (
	"fmt"
	"runtime/debug"
	"strconv"
	"strings"
)

var version = "dev"

var commit = "none"

var date = "unknown"

func versionLine() string {
	if version != "dev" {
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

	if (c == "" || c == "none") && (d == "" || d == "unknown") {
		return "ralph version dev"
	}
	if c == "" || c == "none" {
		return fmt.Sprintf("ralph version dev (built %s)", d)
	}
	if d == "" || d == "unknown" {
		return fmt.Sprintf("ralph version dev (commit %s)", c)
	}
	return fmt.Sprintf("ralph version dev (commit %s, built %s)", c, d)
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
