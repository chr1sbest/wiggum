package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWatcher(t *testing.T) {
	// Create a temp directory
	dir, err := os.MkdirTemp("", "watcher-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	// Write initial config
	configPath := filepath.Join(dir, "test.json")
	initialConfig := `{"name": "test-config", "steps": []}`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("failed to write initial config: %v", err)
	}

	// Create watcher
	loader := NewLoader(dir)
	watcher, err := NewWatcher(loader, dir)
	if err != nil {
		t.Fatalf("failed to create watcher: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := watcher.Start(ctx); err != nil {
		t.Fatalf("failed to start watcher: %v", err)
	}
	defer watcher.Stop()

	// Verify initial config was loaded
	cfg, ok := watcher.GetConfig("test-config")
	if !ok {
		t.Fatal("initial config not loaded")
	}
	if cfg.Name != "test-config" {
		t.Errorf("expected name 'test-config', got %q", cfg.Name)
	}

	// Update the config and watch for event
	updatedConfig := `{"name": "test-config", "description": "updated", "steps": []}`
	if err := os.WriteFile(configPath, []byte(updatedConfig), 0644); err != nil {
		t.Fatalf("failed to write updated config: %v", err)
	}

	// Wait for event with timeout
	select {
	case event := <-watcher.Events():
		if event.Error != nil {
			t.Errorf("unexpected error: %v", event.Error)
		}
		if event.Config == nil {
			t.Error("expected config in event")
		} else if event.Config.Description != "updated" {
			t.Errorf("expected description 'updated', got %q", event.Config.Description)
		}
	case <-time.After(2 * time.Second):
		t.Error("timed out waiting for config event")
	}
}

func TestWatcherNewFile(t *testing.T) {
	// Create a temp directory
	dir, err := os.MkdirTemp("", "watcher-newfile-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	// Create watcher on empty directory
	loader := NewLoader(dir)
	watcher, err := NewWatcher(loader, dir)
	if err != nil {
		t.Fatalf("failed to create watcher: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := watcher.Start(ctx); err != nil {
		t.Fatalf("failed to start watcher: %v", err)
	}
	defer watcher.Stop()

	// Create a new config file
	configPath := filepath.Join(dir, "new.json")
	newConfig := `{"name": "new-config", "steps": []}`
	if err := os.WriteFile(configPath, []byte(newConfig), 0644); err != nil {
		t.Fatalf("failed to write new config: %v", err)
	}

	// Wait for event
	select {
	case event := <-watcher.Events():
		if event.Error != nil {
			t.Errorf("unexpected error: %v", event.Error)
		}
		if event.Config == nil {
			t.Error("expected config in event")
		} else if event.Config.Name != "new-config" {
			t.Errorf("expected name 'new-config', got %q", event.Config.Name)
		}
	case <-time.After(2 * time.Second):
		t.Error("timed out waiting for new config event")
	}

	// Verify it's accessible via GetConfig
	cfg, ok := watcher.GetConfig("new-config")
	if !ok {
		t.Fatal("new config not found in watcher")
	}
	if cfg.Name != "new-config" {
		t.Errorf("expected name 'new-config', got %q", cfg.Name)
	}
}

func TestWatcherGetAllConfigs(t *testing.T) {
	dir, err := os.MkdirTemp("", "watcher-getall-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	// Write two configs
	if err := os.WriteFile(filepath.Join(dir, "a.json"), []byte(`{"name": "config-a", "steps": []}`), 0644); err != nil {
		t.Fatalf("failed to write config a: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.json"), []byte(`{"name": "config-b", "steps": []}`), 0644); err != nil {
		t.Fatalf("failed to write config b: %v", err)
	}

	loader := NewLoader(dir)
	watcher, err := NewWatcher(loader, dir)
	if err != nil {
		t.Fatalf("failed to create watcher: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := watcher.Start(ctx); err != nil {
		t.Fatalf("failed to start watcher: %v", err)
	}
	defer watcher.Stop()

	configs := watcher.GetAllConfigs()
	if len(configs) != 2 {
		t.Errorf("expected 2 configs, got %d", len(configs))
	}
	if _, ok := configs["config-a"]; !ok {
		t.Error("config-a not found")
	}
	if _, ok := configs["config-b"]; !ok {
		t.Error("config-b not found")
	}
}
