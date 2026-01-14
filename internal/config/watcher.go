package config

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// ConfigEvent represents a configuration change event.
type ConfigEvent struct {
	Path   string
	Config *Config
	Error  error
}

// Watcher monitors a directory for configuration file changes.
type Watcher struct {
	loader   *Loader
	watchDir string
	watcher  *fsnotify.Watcher
	events   chan ConfigEvent
	debounce time.Duration
	mu       sync.RWMutex
	configs  map[string]*Config
}

// NewWatcher creates a new config file watcher.
func NewWatcher(loader *Loader, watchDir string) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}

	return &Watcher{
		loader:   loader,
		watchDir: watchDir,
		watcher:  fsWatcher,
		events:   make(chan ConfigEvent, 10),
		debounce: 100 * time.Millisecond,
		configs:  make(map[string]*Config),
	}, nil
}

// Events returns the channel that receives config change events.
func (w *Watcher) Events() <-chan ConfigEvent {
	return w.events
}

// Start begins watching the directory for config changes.
func (w *Watcher) Start(ctx context.Context) error {
	// Load existing configs first
	if err := w.loadExisting(); err != nil {
		return fmt.Errorf("failed to load existing configs: %w", err)
	}

	// Add the watch directory
	if err := w.watcher.Add(w.watchDir); err != nil {
		return fmt.Errorf("failed to watch directory %s: %w", w.watchDir, err)
	}

	go w.run(ctx)
	return nil
}

// Stop closes the watcher and cleans up resources.
func (w *Watcher) Stop() error {
	close(w.events)
	return w.watcher.Close()
}

// GetConfig returns a loaded config by filename.
func (w *Watcher) GetConfig(name string) (*Config, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	cfg, ok := w.configs[name]
	return cfg, ok
}

// GetAllConfigs returns all currently loaded configs.
func (w *Watcher) GetAllConfigs() map[string]*Config {
	w.mu.RLock()
	defer w.mu.RUnlock()
	result := make(map[string]*Config, len(w.configs))
	for k, v := range w.configs {
		result[k] = v
	}
	return result
}

func (w *Watcher) loadExisting() error {
	configs, err := w.loader.LoadDirectory(w.watchDir)
	if err != nil {
		return err
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	for _, cfg := range configs {
		w.configs[cfg.Name] = cfg
	}

	return nil
}

func (w *Watcher) run(ctx context.Context) {
	// Debounce map to avoid multiple events for same file
	pending := make(map[string]time.Time)
	ticker := time.NewTicker(w.debounce)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			// Only process JSON files
			if !strings.HasSuffix(event.Name, ".json") {
				continue
			}

			// Track the pending event with timestamp
			if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
				pending[event.Name] = time.Now()
			} else if event.Op&fsnotify.Remove != 0 {
				w.handleRemove(event.Name)
			}

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			w.events <- ConfigEvent{Error: err}

		case <-ticker.C:
			// Process debounced events
			now := time.Now()
			for path, timestamp := range pending {
				if now.Sub(timestamp) >= w.debounce {
					w.handleUpdate(path)
					delete(pending, path)
				}
			}
		}
	}
}

func (w *Watcher) handleUpdate(path string) {
	cfg, err := w.loader.LoadFile(path)
	if err != nil {
		w.events <- ConfigEvent{
			Path:  path,
			Error: fmt.Errorf("failed to load config %s: %w", path, err),
		}
		return
	}

	w.mu.Lock()
	w.configs[cfg.Name] = cfg
	w.mu.Unlock()

	w.events <- ConfigEvent{
		Path:   path,
		Config: cfg,
	}
}

func (w *Watcher) handleRemove(path string) {
	name := strings.TrimSuffix(filepath.Base(path), ".json")

	w.mu.Lock()
	delete(w.configs, name)
	w.mu.Unlock()

	w.events <- ConfigEvent{
		Path:  path,
		Error: fmt.Errorf("config removed: %s", path),
	}
}
