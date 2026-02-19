package main

import (
	"daedalus/pkg/daedalus"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// DeleteList should remove the directory, cards, and all config references.
func TestDeleteList_Success(t *testing.T) {
	app, root := setupTestBoardMulti(t)

	// Verify list exists before delete
	if _, ok := app.board.Lists["open"]; !ok {
		t.Fatal("expected open to exist before delete")
	}
	bytesBefore := app.board.TotalFileBytes

	err := app.DeleteList("open")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Directory should be gone from disk
	dirPath := filepath.Join(root, "open")
	if _, err := os.Stat(dirPath); !os.IsNotExist(err) {
		t.Error("expected directory to be removed from disk")
	}

	// List should be gone from in-memory state
	if _, ok := app.board.Lists["open"]; ok {
		t.Error("expected open to be removed from board.Lists")
	}

	// TotalFileBytes should have decreased
	if app.board.TotalFileBytes >= bytesBefore {
		t.Errorf("TotalFileBytes should have decreased: before=%d, after=%d", bytesBefore, app.board.TotalFileBytes)
	}
}

// DeleteList should return an error for a nonexistent list.
func TestDeleteList_NotFound(t *testing.T) {
	app, _ := setupTestBoardMulti(t)

	err := app.DeleteList("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent list")
	}
	if !strings.Contains(err.Error(), "list not found") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// DeleteList should return an error when no board has been loaded.
func TestDeleteList_BoardNotLoaded(t *testing.T) {
	app := NewApp()
	err := app.DeleteList("open")
	if err == nil {
		t.Fatal("expected error when board not loaded")
	}
	if err.Error() != "board not loaded" {
		t.Errorf("unexpected error message: %v", err)
	}
}

// DeleteList should reject names with path traversal characters.
func TestDeleteList_PathTraversal(t *testing.T) {
	app, _ := setupTestBoardMulti(t)

	for _, name := range []string{"../etc", "foo/bar", "..\\evil"} {
		err := app.DeleteList(name)
		if err == nil {
			t.Errorf("expected error for path traversal name %q", name)
		}
		if err != nil && err.Error() != "invalid list name" {
			t.Errorf("unexpected error for %q: %v", name, err)
		}
	}
}

// DeleteList should remove the list entry from the config Lists array.
func TestDeleteList_CleansConfigLists(t *testing.T) {
	app, _ := setupTestBoardMulti(t)

	err := app.DeleteList("open")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(app.board.Config.Lists) != 1 {
		t.Fatalf("expected 1 entry in Lists, got %d", len(app.board.Config.Lists))
	}
	if app.board.Config.Lists[0].Dir != "done" {
		t.Errorf("unexpected Lists: %v", app.board.Config.Lists)
	}
}

// CreateList should create the directory, update in-memory state, and persist to config.
func TestCreateList_Success(t *testing.T) {
	app, root := setupTestBoardMulti(t)

	err := app.CreateList("backlog")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Directory should exist on disk
	dirPath := filepath.Join(root, "backlog")
	info, err := os.Stat(dirPath)
	if err != nil {
		t.Fatalf("expected directory to exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected a directory, not a file")
	}

	// In-memory state should have the new list
	cards, ok := app.board.Lists["backlog"]
	if !ok {
		t.Fatal("expected backlog in board.Lists")
	}
	if len(cards) != 0 {
		t.Errorf("expected 0 cards, got %d", len(cards))
	}

	// Config should have the new entry
	idx := daedalus.FindListEntry(app.board.Config.Lists, "backlog")
	if idx < 0 {
		t.Fatal("expected config entry for backlog")
	}

	// Verify persisted to disk
	config, err := daedalus.LoadBoardConfig(root)
	if err != nil {
		t.Fatalf("error loading saved config: %v", err)
	}
	savedIdx := daedalus.FindListEntry(config.Lists, "backlog")
	if savedIdx < 0 {
		t.Fatal("expected persisted config entry for backlog")
	}
}

// CreateList should reject various invalid names.
func TestCreateList_InvalidName(t *testing.T) {
	app, _ := setupTestBoardMulti(t)

	invalidNames := []string{"../etc", "foo/bar", ".hidden", "_assets", "", "  "}
	for _, name := range invalidNames {
		err := app.CreateList(name)
		if err == nil {
			t.Errorf("expected error for invalid name %q", name)
		}
	}
}

// CreateList should reject a name that already exists.
func TestCreateList_Duplicate(t *testing.T) {
	app, _ := setupTestBoardMulti(t)

	err := app.CreateList("open")
	if err == nil {
		t.Fatal("expected error for duplicate list name")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// CreateList should return an error when no board has been loaded.
func TestCreateList_BoardNotLoaded(t *testing.T) {
	app := NewApp()
	err := app.CreateList("new-list")
	if err == nil {
		t.Fatal("expected error when board not loaded")
	}
	if err.Error() != "board not loaded" {
		t.Errorf("unexpected error message: %v", err)
	}
}
