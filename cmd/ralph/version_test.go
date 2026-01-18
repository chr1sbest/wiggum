package main

import (
	"testing"
)

func TestParseSemver(t *testing.T) {
	tests := []struct {
		input string
		want  [3]int
	}{
		{"1.0.0", [3]int{1, 0, 0}},
		{"v1.0.0", [3]int{1, 0, 0}},
		{"2.3.4", [3]int{2, 3, 4}},
		{"v10.20.30", [3]int{10, 20, 30}},
		{"1.2", [3]int{1, 2, 0}},
		{"1", [3]int{1, 0, 0}},
		{"", [3]int{0, 0, 0}},
		{"invalid", [3]int{0, 0, 0}},
		{"  v1.2.3  ", [3]int{0, 0, 0}}, // whitespace inside parts causes parse failure
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseSemver(tt.input)
			if got != tt.want {
				t.Errorf("parseSemver(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestCompareSemver(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.1", "1.0.0", 1},
		{"1.0.0", "1.0.1", -1},
		{"2.0.0", "1.9.9", 1},
		{"1.9.9", "2.0.0", -1},
		{"1.2.3", "1.2.3", 0},
		{"v1.0.0", "1.0.0", 0},
		{"1.0.6", "1.0.5", 1},
		{"1.1.0", "1.0.9", 1},
	}

	for _, tt := range tests {
		name := tt.a + "_vs_" + tt.b
		t.Run(name, func(t *testing.T) {
			got := compareSemver(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("compareSemver(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestVersionConstant(t *testing.T) {
	if version == "" {
		t.Error("version constant should not be empty")
	}
	parts := parseSemver(version)
	if parts[0] == 0 && parts[1] == 0 && parts[2] == 0 {
		t.Errorf("version %q should parse to valid semver", version)
	}
}
