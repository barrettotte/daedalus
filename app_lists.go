package main

import (
	"daedalus/pkg/daedalus"
	"fmt"
	"log/slog"
)

// CreateList creates a new empty list directory and adds it to the board config.
func (a *App) CreateList(name string) error {
	if _, err := a.prepareWrite(); err != nil {
		return err
	}

	// Validate and clean list name.
	name, err := daedalus.ValidateListName(name)
	if err != nil {
		return err
	}

	// Check for duplicates
	if _, exists := a.board.Lists[name]; exists {
		return fmt.Errorf("list already exists: %s", name)
	}

	if err := daedalus.CreateListOnDisk(a.board.RootPath, name, a.board.Config); err != nil {
		return err
	}

	a.board.Lists[name] = []daedalus.KanbanCard{}
	slog.Info("list created", "name", name)
	return nil
}

// DeleteList removes an entire list directory and cleans up all config references.
func (a *App) DeleteList(listDirName string) error {
	if _, err := a.prepareWrite(); err != nil {
		return err
	}

	// Validate list name.
	listDirName, err := daedalus.ValidateListName(listDirName)
	if err != nil {
		return err
	}

	// Block if list is locked.
	if daedalus.IsListLocked(a.board.Config, listDirName) {
		return fmt.Errorf("list is locked")
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

	if err := daedalus.DeleteListOnDisk(a.board.RootPath, listDirName, a.board.Config); err != nil {
		return err
	}

	// Update in-memory state
	a.board.TotalFileBytes -= totalBytes
	delete(a.board.Lists, listDirName)

	slog.Info("list deleted", "name", listDirName, "cardsRemoved", len(cards), "bytesFreed", totalBytes)
	return nil
}

// saveListBoolFlags builds a set from dirs, applies setFn to each list config entry, and persists to board.yaml.
func (a *App) saveListBoolFlags(dirs []string, setFn func(*daedalus.ListEntry, bool)) error {
	if _, err := a.prepareWrite(); err != nil {
		return err
	}

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
	if _, err := a.prepareWrite(); err != nil {
		return err
	}

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
	slog.Debug("locked lists saved", "count", len(locked))
	return nil
}

// SaveListConfig updates the config for a single list and persists to board.yaml.
func (a *App) SaveListConfig(dirName string, title string, limit int, color string, icon string) error {
	if _, err := a.prepareWrite(); err != nil {
		return err
	}

	idx := daedalus.FindListEntry(a.board.Config.Lists, dirName)
	if idx >= 0 {
		a.board.Config.Lists[idx].Title = title
		a.board.Config.Lists[idx].Limit = limit
		a.board.Config.Lists[idx].Color = color
		a.board.Config.Lists[idx].Icon = icon
	} else {
		a.board.Config.Lists = append(a.board.Config.Lists, daedalus.ListEntry{
			Dir:   dirName,
			Title: title,
			Limit: limit,
			Color: color,
			Icon:  icon,
		})
	}

	if err := daedalus.SaveBoardConfig(a.board.RootPath, a.board.Config); err != nil {
		slog.Error("failed to save list config", "dir", dirName, "error", err)
		return err
	}
	slog.Info("list config saved", "dir", dirName, "title", title, "limit", limit)
	return nil
}

// SaveListOrder reorders the config Lists array to match the given order and persists to board.yaml.
func (a *App) SaveListOrder(order []string) error {
	if _, err := a.prepareWrite(); err != nil {
		return err
	}

	// Build a map of dir -> entry for quick lookup
	entryMap := make(map[string]daedalus.ListEntry)
	for _, entry := range a.board.Config.Lists {
		entryMap[entry.Dir] = entry
	}

	// Reassemble in new order
	var reordered []daedalus.ListEntry
	used := make(map[string]bool)
	for _, dir := range order {
		if entry, ok := entryMap[dir]; ok {
			reordered = append(reordered, entry)
			used[dir] = true
		}
	}

	// Append any stragglers not in the order array
	for _, entry := range a.board.Config.Lists {
		if !used[entry.Dir] {
			reordered = append(reordered, entry)
		}
	}

	a.board.Config.Lists = reordered
	if err := daedalus.SaveBoardConfig(a.board.RootPath, a.board.Config); err != nil {
		slog.Error("failed to save list order", "error", err)
		return err
	}
	slog.Info("list order saved", "count", len(reordered))
	return nil
}
