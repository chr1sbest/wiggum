package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLooksLikeGoInstall(t *testing.T) {
	tests := []struct {
		name   string
		exeDir string
		gobin  string
		gopath string
		want   bool
	}{
		{
			name:   "matches GOBIN",
			exeDir: "/home/user/go/bin",
			gobin:  "/home/user/go/bin",
			gopath: "",
			want:   true,
		},
		{
			name:   "matches GOPATH/bin",
			exeDir: "/home/user/go/bin",
			gobin:  "",
			gopath: "/home/user/go",
			want:   true,
		},
		{
			name:   "no match",
			exeDir: "/usr/local/bin",
			gobin:  "",
			gopath: "/home/user/go",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldGobin := os.Getenv("GOBIN")
			oldGopath := os.Getenv("GOPATH")
			defer func() {
				os.Setenv("GOBIN", oldGobin)
				os.Setenv("GOPATH", oldGopath)
			}()

			os.Setenv("GOBIN", tt.gobin)
			os.Setenv("GOPATH", tt.gopath)

			got := looksLikeGoInstall(tt.exeDir)
			if got != tt.want {
				t.Errorf("looksLikeGoInstall(%q) = %v, want %v", tt.exeDir, got, tt.want)
			}
		})
	}
}

func TestSamePath(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"/foo/bar", "/foo/bar", true},
		{"/foo/bar/", "/foo/bar", true},
		{"/foo/bar", "/foo/baz", false},
		{"/foo/../foo/bar", "/foo/bar", true},
	}

	for _, tt := range tests {
		name := tt.a + "_vs_" + tt.b
		t.Run(name, func(t *testing.T) {
			got := samePath(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("samePath(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestCheckWritable(t *testing.T) {
	// Create a temp file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	// Test writable file
	if err := checkWritable(tmpFile); err != nil {
		t.Errorf("checkWritable() returned error for writable file: %v", err)
	}

	// Test non-existent file
	nonExistent := filepath.Join(tmpDir, "nonexistent.txt")
	if err := checkWritable(nonExistent); err == nil {
		t.Error("checkWritable() should return error for non-existent file")
	}
}
