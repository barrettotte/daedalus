package main

import (
	"daedalus/pkg/daedalus"
	"path/filepath"
	"testing"
)

// SaveListConfig should update the in-memory config and persist to board.yaml.
func TestSaveListConfig_Success(t *testing.T) {
	app, root := setupTestBoard(t)

	err := app.SaveListConfig("test", "My Test List", 10, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	idx := daedalus.FindListEntry(app.board.Config.Lists, "test")
	if idx < 0 {
		t.Fatal("expected config entry for test")
	}
	lc := app.board.Config.Lists[idx]
	if lc.Title != "My Test List" || lc.Limit != 10 {
		t.Errorf("got title=%q limit=%d, want title=\"My Test List\" limit=10", lc.Title, lc.Limit)
	}

	// Verify file was written
	config, err := daedalus.LoadBoardConfig(root)
	if err != nil {
		t.Fatalf("error loading saved config: %v", err)
	}
	savedIdx := daedalus.FindListEntry(config.Lists, "test")
	if savedIdx < 0 {
		t.Fatal("expected saved config entry for test")
	}
	saved := config.Lists[savedIdx]
	if saved.Title != "My Test List" || saved.Limit != 10 {
		t.Errorf("saved config: got %+v", saved)
	}
}

// SaveListConfig should return an error when no board has been loaded.
func TestSaveListConfig_BoardNotLoaded(t *testing.T) {
	app := NewApp()
	err := app.SaveListConfig("test", "Title", 5, "", "")
	if err == nil {
		t.Fatal("expected error when board not loaded")
	}
	if err.Error() != "board not loaded" {
		t.Errorf("unexpected error message: %v", err)
	}
}

// SaveLabelsExpanded should persist the value to board.yaml and reload correctly.
func TestSaveLabelsExpanded_Success(t *testing.T) {
	app, root := setupTestBoard(t)

	if err := app.SaveLabelsExpanded(false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	config, err := daedalus.LoadBoardConfig(root)
	if err != nil {
		t.Fatalf("error loading config: %v", err)
	}
	if config.LabelsExpanded == nil || *config.LabelsExpanded != false {
		t.Errorf("expected labelsExpanded=false, got %v", config.LabelsExpanded)
	}

	if err := app.SaveLabelsExpanded(true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	config, err = daedalus.LoadBoardConfig(root)
	if err != nil {
		t.Fatalf("error loading config: %v", err)
	}
	if config.LabelsExpanded == nil || *config.LabelsExpanded != true {
		t.Errorf("expected labelsExpanded=true, got %v", config.LabelsExpanded)
	}
}

// SaveHalfCollapsedLists should persist the flags to board.yaml and reload correctly.
func TestSaveHalfCollapsedLists_Success(t *testing.T) {
	app, root := setupTestBoard(t)

	lists := []string{"test"}
	if err := app.SaveHalfCollapsedLists(lists); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	config, err := daedalus.LoadBoardConfig(root)
	if err != nil {
		t.Fatalf("error loading config: %v", err)
	}
	idx := daedalus.FindListEntry(config.Lists, "test")
	if idx < 0 {
		t.Fatal("expected config entry for test")
	}
	if !config.Lists[idx].HalfCollapsed {
		t.Error("expected test to be half-collapsed")
	}

	// Clear and verify
	if err := app.SaveHalfCollapsedLists(nil); err != nil {
		t.Fatalf("unexpected error clearing: %v", err)
	}

	config, err = daedalus.LoadBoardConfig(root)
	if err != nil {
		t.Fatalf("error loading config: %v", err)
	}
	idx = daedalus.FindListEntry(config.Lists, "test")
	if idx < 0 {
		t.Fatal("expected config entry for test after clear")
	}
	if config.Lists[idx].HalfCollapsed {
		t.Error("expected test to not be half-collapsed after clearing")
	}
}

// SaveListOrder should reorder the Lists array and persist to board.yaml.
func TestSaveListOrder_Success(t *testing.T) {
	app, root := setupTestBoardMulti(t)

	order := []string{"done", "open"}
	if err := app.SaveListOrder(order); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(app.board.Config.Lists) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(app.board.Config.Lists))
	}
	if app.board.Config.Lists[0].Dir != "done" || app.board.Config.Lists[1].Dir != "open" {
		t.Errorf("unexpected in-memory order: %v", app.board.Config.Lists)
	}

	// Verify persisted to disk
	config, err := daedalus.LoadBoardConfig(root)
	if err != nil {
		t.Fatalf("error loading config: %v", err)
	}
	if len(config.Lists) != 2 {
		t.Fatalf("expected 2 persisted entries, got %d", len(config.Lists))
	}
	if config.Lists[0].Dir != "done" {
		t.Errorf("unexpected persisted order: %v", config.Lists)
	}
}

// SaveListOrder should return an error when no board has been loaded.
func TestSaveListOrder_BoardNotLoaded(t *testing.T) {
	app := NewApp()
	err := app.SaveListOrder([]string{"a", "b"})
	if err == nil {
		t.Fatal("expected error when board not loaded")
	}
	if err.Error() != "board not loaded" {
		t.Errorf("unexpected error message: %v", err)
	}
}

// SaveLockedLists should persist the Locked flag to board.yaml and reload correctly.
func TestSaveLockedLists_Success(t *testing.T) {
	app, root := setupTestBoardMulti(t)

	locked := []string{"open"}
	if err := app.SaveLockedLists(locked); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	config, err := daedalus.LoadBoardConfig(root)
	if err != nil {
		t.Fatalf("error loading config: %v", err)
	}
	idx := daedalus.FindListEntry(config.Lists, "open")
	if idx < 0 {
		t.Fatal("expected config entry for open")
	}
	if !config.Lists[idx].Locked {
		t.Error("expected open to be locked")
	}

	doneIdx := daedalus.FindListEntry(config.Lists, "done")
	if doneIdx >= 0 && config.Lists[doneIdx].Locked {
		t.Error("expected done to NOT be locked")
	}

	// Clear and verify
	if err := app.SaveLockedLists(nil); err != nil {
		t.Fatalf("unexpected error clearing: %v", err)
	}

	config, err = daedalus.LoadBoardConfig(root)
	if err != nil {
		t.Fatalf("error loading config: %v", err)
	}
	idx = daedalus.FindListEntry(config.Lists, "open")
	if idx < 0 {
		t.Fatal("expected config entry for open after clear")
	}
	if config.Lists[idx].Locked {
		t.Error("expected open to not be locked after clearing")
	}
}

// SaveLockedLists should return an error when no board has been loaded.
func TestSaveLockedLists_BoardNotLoaded(t *testing.T) {
	app := NewApp()
	err := app.SaveLockedLists([]string{"open"})
	if err == nil {
		t.Fatal("expected error when board not loaded")
	}
	if err.Error() != "board not loaded" {
		t.Errorf("unexpected error message: %v", err)
	}
}

// SavePinnedLists should persist left/right pin state and round-trip through reload.
func TestSavePinnedLists_Success(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"backlog", "open", "done"} {
		mustMkdir(t, filepath.Join(root, dir))
		mustWrite(t, filepath.Join(root, dir, "1.md"), []byte("---\ntitle: \"T\"\nid: 1\n---\n"))
	}

	app := NewApp()
	resp := app.LoadBoard(root)
	if resp == nil {
		t.Fatal("LoadBoard returned nil")
	}

	// Pin backlog left, done right
	if err := app.SavePinnedLists([]string{"backlog"}, []string{"done"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify in-memory state
	for _, entry := range app.board.Config.Lists {
		switch entry.Dir {
		case "backlog":
			if entry.Pinned != "left" {
				t.Errorf("backlog: got pinned=%q, want left", entry.Pinned)
			}
		case "done":
			if entry.Pinned != "right" {
				t.Errorf("done: got pinned=%q, want right", entry.Pinned)
			}
		case "open":
			if entry.Pinned != "" {
				t.Errorf("open: got pinned=%q, want empty", entry.Pinned)
			}
		}
	}

	// Verify round-trip: reload config from disk
	config, err := daedalus.LoadBoardConfig(root)
	if err != nil {
		t.Fatalf("error loading config: %v", err)
	}

	for _, entry := range config.Lists {
		switch entry.Dir {
		case "backlog":
			if entry.Pinned != "left" {
				t.Errorf("reloaded backlog: got pinned=%q, want left", entry.Pinned)
			}
		case "done":
			if entry.Pinned != "right" {
				t.Errorf("reloaded done: got pinned=%q, want right", entry.Pinned)
			}
		case "open":
			if entry.Pinned != "" {
				t.Errorf("reloaded open: got pinned=%q, want empty", entry.Pinned)
			}
		}
	}

	// Clear all pins and verify
	if err := app.SavePinnedLists(nil, nil); err != nil {
		t.Fatalf("unexpected error clearing: %v", err)
	}

	config, err = daedalus.LoadBoardConfig(root)
	if err != nil {
		t.Fatalf("error loading config after clear: %v", err)
	}
	for _, entry := range config.Lists {
		if entry.Pinned != "" {
			t.Errorf("after clear: %s got pinned=%q, want empty", entry.Dir, entry.Pinned)
		}
	}
}

// SavePinnedLists should return an error when no board has been loaded.
func TestSavePinnedLists_BoardNotLoaded(t *testing.T) {
	app := NewApp()
	err := app.SavePinnedLists([]string{"open"}, nil)
	if err == nil {
		t.Fatal("expected error when board not loaded")
	}
	if err.Error() != "board not loaded" {
		t.Errorf("unexpected error message: %v", err)
	}
}
