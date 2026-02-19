package main

import (
	"daedalus/pkg/daedalus"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// CreateList creates a new empty list directory and adds it to the board config.
func (a *App) CreateList(name string) error {
	if a.board == nil {
		return fmt.Errorf("board not loaded")
	}
	a.pauseWatcher()

	// Validate name: reject empty, path separators, traversal, hidden dirs, and reserved names
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("list name cannot be empty")
	}
	if strings.ContainsAny(name, "/\\") || strings.Contains(name, "..") {
		slog.Warn("rejected invalid list name", "name", name)
		return fmt.Errorf("invalid list name")
	}
	if strings.HasPrefix(name, ".") {
		return fmt.Errorf("list name cannot start with a dot")
	}
	if name == "_assets" {
		return fmt.Errorf("'_assets' is a reserved name")
	}

	// Check for duplicates
	if _, exists := a.board.Lists[name]; exists {
		return fmt.Errorf("list already exists: %s", name)
	}

	// Create directory on disk
	dirPath := filepath.Join(a.board.RootPath, name)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		slog.Error("failed to create list directory", "name", name, "path", dirPath, "error", err)
		return fmt.Errorf("creating list directory: %w", err)
	}

	// Update in-memory state
	a.board.Lists[name] = []daedalus.KanbanCard{}
	a.board.Config.Lists = append(a.board.Config.Lists, daedalus.ListEntry{Dir: name})

	if err := daedalus.SaveBoardConfig(a.board.RootPath, a.board.Config); err != nil {
		slog.Error("failed to save config after list creation", "name", name, "error", err)
		return err
	}

	slog.Info("list created", "name", name)
	return nil
}

// DeleteList removes an entire list directory and cleans up all config references.
func (a *App) DeleteList(listDirName string) error {
	if a.board == nil {
		return fmt.Errorf("board not loaded")
	}
	a.pauseWatcher()

	// Reject names with path separators or traversal
	if strings.ContainsAny(listDirName, "/\\") || strings.Contains(listDirName, "..") {
		slog.Warn("rejected invalid list name", "name", listDirName)
		return fmt.Errorf("invalid list name")
	}

	// Verify list exists in memory
	cards, ok := a.board.Lists[listDirName]
	if !ok {
		slog.Warn("attempted to delete non-existent list", "name", listDirName)
		return fmt.Errorf("list not found: %s", listDirName)
	}

	slog.Info("deleting list", "name", listDirName, "cards", len(cards))

	// Sum file bytes for metrics update
	var totalBytes int64
	for _, card := range cards {
		totalBytes += daedalus.GetFileSize(card.FilePath)
	}

	// Remove directory from disk
	dirPath := filepath.Join(a.board.RootPath, listDirName)
	if err := os.RemoveAll(dirPath); err != nil {
		slog.Error("failed to remove list directory", "name", listDirName, "path", dirPath, "error", err)
		return fmt.Errorf("removing list directory: %w", err)
	}

	// Update metrics
	a.board.TotalFileBytes -= totalBytes

	// Clean up in-memory state
	delete(a.board.Lists, listDirName)

	// Remove from config Lists array
	idx := daedalus.FindListEntry(a.board.Config.Lists, listDirName)
	if idx >= 0 {
		a.board.Config.Lists = append(a.board.Config.Lists[:idx], a.board.Config.Lists[idx+1:]...)
	}

	if err := daedalus.SaveBoardConfig(a.board.RootPath, a.board.Config); err != nil {
		slog.Error("failed to save config after list deletion", "name", listDirName, "error", err)
		return err
	}
	slog.Info("list deleted", "name", listDirName, "cardsRemoved", len(cards), "bytesFreed", totalBytes)
	return nil
}

// saveListBoolFlags builds a set from dirs, applies setFn to each list config entry, and persists to board.yaml.
func (a *App) saveListBoolFlags(dirs []string, setFn func(*daedalus.ListEntry, bool)) error {
	if a.board == nil {
		return fmt.Errorf("board not loaded")
	}
	a.pauseWatcher()

	set := make(map[string]bool, len(dirs))
	for _, dir := range dirs {
		set[dir] = true
	}
	for i := range a.board.Config.Lists {
		setFn(&a.board.Config.Lists[i], set[a.board.Config.Lists[i].Dir])
	}
	return daedalus.SaveBoardConfig(a.board.RootPath, a.board.Config)
}

// SaveCollapsedLists sets the Collapsed flag on matching entries and persists to board.yaml.
func (a *App) SaveCollapsedLists(collapsed []string) error {
	if err := a.saveListBoolFlags(collapsed, func(e *daedalus.ListEntry, v bool) { e.Collapsed = v }); err != nil {
		slog.Error("failed to save collapsed lists", "error", err)
		return err
	}
	slog.Debug("collapsed lists saved", "count", len(collapsed))
	return nil
}

// SaveHalfCollapsedLists sets the HalfCollapsed flag on matching entries and persists to board.yaml.
func (a *App) SaveHalfCollapsedLists(halfCollapsed []string) error {
	if err := a.saveListBoolFlags(halfCollapsed, func(e *daedalus.ListEntry, v bool) { e.HalfCollapsed = v }); err != nil {
		slog.Error("failed to save half-collapsed lists", "error", err)
		return err
	}
	slog.Debug("half-collapsed lists saved", "count", len(halfCollapsed))
	return nil
}

// SavePinnedLists sets the Pinned field on matching entries and persists to board.yaml.
func (a *App) SavePinnedLists(left []string, right []string) error {
	if a.board == nil {
		return fmt.Errorf("board not loaded")
	}
	a.pauseWatcher()

	leftSet := make(map[string]bool, len(left))
	for _, dir := range left {
		leftSet[dir] = true
	}
	rightSet := make(map[string]bool, len(right))
	for _, dir := range right {
		rightSet[dir] = true
	}

	for i := range a.board.Config.Lists {
		dir := a.board.Config.Lists[i].Dir
		if leftSet[dir] {
			a.board.Config.Lists[i].Pinned = "left"
		} else if rightSet[dir] {
			a.board.Config.Lists[i].Pinned = "right"
		} else {
			a.board.Config.Lists[i].Pinned = ""
		}
	}

	if err := daedalus.SaveBoardConfig(a.board.RootPath, a.board.Config); err != nil {
		slog.Error("failed to save pinned lists", "error", err)
		return err
	}
	slog.Debug("pinned lists saved", "left", len(left), "right", len(right))
	return nil
}

// SaveLockedLists sets the Locked flag on matching entries and persists to board.yaml.
func (a *App) SaveLockedLists(locked []string) error {
	if err := a.saveListBoolFlags(locked, func(e *daedalus.ListEntry, v bool) { e.Locked = v }); err != nil {
		slog.Error("failed to save locked lists", "error", err)
		return err
	}
	slog.Info("locked lists saved", "count", len(locked))
	return nil
}

