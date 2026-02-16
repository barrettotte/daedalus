package daedalus

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	pollInterval = 5 * time.Second
)

// FileWatcher polls a board directory for external file changes and calls a
// callback when modifications are detected. Uses file modification times
// rather than OS-level filesystem events, so it has zero external dependencies.
type FileWatcher struct {
	rootPath string
	onChange func()
	done     chan struct{}
	mu       sync.Mutex
	snapshot map[string]time.Time // filePath -> modTime
}

// NewFileWatcher creates and starts a polling file watcher for the given board root.
// The onChange callback fires when any relevant file is created, modified, or deleted.
func NewFileWatcher(rootPath string, onChange func()) *FileWatcher {
	fw := &FileWatcher{
		rootPath: rootPath,
		onChange: onChange,
		done:     make(chan struct{}),
		snapshot: make(map[string]time.Time),
	}

	fw.snapshot = fw.scan()
	go fw.run()

	slog.Info("file watcher started", "path", rootPath, "files", len(fw.snapshot))
	return fw
}

// Close stops the file watcher.
func (fw *FileWatcher) Close() {
	close(fw.done)
	slog.Info("file watcher stopped")
}

// run is the main polling loop.
func (fw *FileWatcher) run() {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-fw.done:
			return
		case <-ticker.C:
			fw.poll()
		}
	}
}

// poll takes a new snapshot and compares it against the previous one.
func (fw *FileWatcher) poll() {
	current := fw.scan()

	fw.mu.Lock()
	prev := fw.snapshot
	fw.snapshot = current
	fw.mu.Unlock()

	if fw.hasChanged(prev, current) {
		slog.Debug("file watcher detected changes")
		fw.onChange()
	}
}

// hasChanged returns true if any file was added, removed, or modified.
func (fw *FileWatcher) hasChanged(prev, current map[string]time.Time) bool {
	if len(prev) != len(current) {
		return true
	}
	for path, modTime := range current {
		if prevTime, ok := prev[path]; !ok || !prevTime.Equal(modTime) {
			return true
		}
	}
	return false
}

// scan walks the board directory and returns a snapshot of all relevant file modification times.
func (fw *FileWatcher) scan() map[string]time.Time {
	result := make(map[string]time.Time)

	// Track the root directory itself (detects new/deleted list dirs).
	if info, err := os.Stat(fw.rootPath); err == nil {
		result[fw.rootPath] = info.ModTime()
	}

	// Track board.yaml.
	configPath := filepath.Join(fw.rootPath, "board.yaml")
	if info, err := os.Stat(configPath); err == nil {
		result[configPath] = info.ModTime()
	}

	// Scan each list subdirectory for .md card files.
	entries, err := os.ReadDir(fw.rootPath)
	if err != nil {
		slog.Warn("file watcher: failed to read root dir", "error", err)
		return result
	}

	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") || entry.Name() == "assets" {
			continue
		}
		listDir := filepath.Join(fw.rootPath, entry.Name())

		// Track the list directory itself (detects new/deleted cards).
		if info, err := os.Stat(listDir); err == nil {
			result[listDir] = info.ModTime()
		}

		files, err := os.ReadDir(listDir)
		if err != nil {
			slog.Warn("file watcher: failed to read list dir", "dir", listDir, "error", err)
			continue
		}

		for _, file := range files {
			if file.IsDir() || !strings.HasSuffix(file.Name(), ".md") {
				continue
			}
			filePath := filepath.Join(listDir, file.Name())
			if info, err := file.Info(); err == nil {
				result[filePath] = info.ModTime()
			}
		}
	}

	return result
}
