package agent

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCheckPRDTasks_NoTasks(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "prd.json")
	if err := os.WriteFile(p, []byte(`{"version":1,"tasks":[]}`), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	has, all, err := CheckPRDTasks(p)
	if !errors.Is(err, ErrPRDNoTasks) {
		t.Fatalf("expected ErrPRDNoTasks, got %v", err)
	}
	if has {
		t.Fatalf("expected hasTasks=false")
	}
	if all {
		t.Fatalf("expected allComplete=false")
	}
}

func TestCheckPRDTasks_AllComplete(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "prd.json")
	if err := os.WriteFile(p, []byte(`{"version":1,"tasks":[{"id":"T1","title":"a","status":"done"},{"id":"T2","title":"b","status":"done"}]}`), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	has, all, err := CheckPRDTasks(p)
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if !has {
		t.Fatalf("expected hasTasks=true")
	}
	if !all {
		t.Fatalf("expected allComplete=true")
	}
}

func TestCheckPRDTasks_Incomplete(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "prd.json")
	if err := os.WriteFile(p, []byte(`{"version":1,"tasks":[{"id":"T1","title":"a","status":"done"},{"id":"T2","title":"b","status":"todo"}]}`), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	has, all, err := CheckPRDTasks(p)
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if !has {
		t.Fatalf("expected hasTasks=true")
	}
	if all {
		t.Fatalf("expected allComplete=false")
	}
}

func TestCheckPRDTasks_FencedJSON(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "prd.json")
	contents := "```json\n{\"version\":1,\"tasks\":[{\"id\":\"T1\",\"title\":\"a\",\"status\":\"done\"}]}\n```\n"
	if err := os.WriteFile(p, []byte(contents), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	has, all, err := CheckPRDTasks(p)
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if !has || !all {
		t.Fatalf("expected hasTasks=true and allComplete=true, got has=%v all=%v", has, all)
	}
}
