package main

import (
	"daedalus/pkg/daedalus"
	"path/filepath"
	"testing"
)

// Calling LoadBoard with an empty string should use the default path without panicking.
func TestLoadBoard_DefaultPath(t *testing.T) {
	app := NewApp()
	app.LoadBoard("")
}

// LoadBoard should return nil when given a nonexistent directory.
func TestLoadBoard_InvalidPath(t *testing.T) {
	app := NewApp()
	resp := app.LoadBoard("/nonexistent/path/that/does/not/exist")
	if resp != nil {
		t.Error("expected nil for invalid path")
	}
}

// After a successful LoadBoard, the internal board state should be populated
// with the correct RootPath.
func TestLoadBoard_SetsBoard(t *testing.T) {
	root := t.TempDir()
	list := filepath.Join(root, "test")
	mustMkdir(t, list)
	mustWrite(t, filepath.Join(list, "1.md"), []byte("---\ntitle: \"T\"\nid: 1\n---\n"))

	app := NewApp()
	resp := app.LoadBoard(root)
	if resp == nil {
		t.Fatal("LoadBoard returned nil")
	}
	if app.board == nil {
		t.Fatal("board should be set after LoadBoard")
	}
	if app.board.RootPath != root {
		t.Errorf("RootPath: got %q, want %q", app.board.RootPath, root)
	}
}

// The response from LoadBoard should contain the correct list keys and card metadata.
func TestLoadBoard_ReturnedData(t *testing.T) {
	root := t.TempDir()
	list := filepath.Join(root, "open")
	mustMkdir(t, list)
	mustWrite(t, filepath.Join(list, "5.md"), []byte("---\ntitle: \"Card Five\"\nid: 5\nlist_order: 1\n---\nBody\n"))

	app := NewApp()
	resp := app.LoadBoard(root)
	if resp == nil {
		t.Fatal("LoadBoard returned nil")
	}

	cards, ok := resp.Lists["open"]
	if !ok {
		t.Fatal("expected open key in result")
	}
	if len(cards) != 1 {
		t.Fatalf("expected 1 card, got %d", len(cards))
	}
	if cards[0].Metadata.Title != "Card Five" {
		t.Errorf("title: got %q, want %q", cards[0].Metadata.Title, "Card Five")
	}
}

// LoadBoard response should include a non-nil config even without a board.yaml file.
func TestLoadBoard_IncludesConfig(t *testing.T) {
	root := t.TempDir()
	list := filepath.Join(root, "test")
	mustMkdir(t, list)
	mustWrite(t, filepath.Join(list, "1.md"), []byte("---\ntitle: \"T\"\nid: 1\n---\n"))

	app := NewApp()
	resp := app.LoadBoard(root)
	if resp == nil {
		t.Fatal("LoadBoard returned nil")
	}
	if resp.Config == nil {
		t.Fatal("expected non-nil Config in response")
	}

	// Lists should be populated by LoadBoard's merge logic (discovered "test" dir)
	if len(resp.Config.Lists) != 1 {
		t.Fatalf("expected 1 list entry from merge, got %d", len(resp.Config.Lists))
	}
}

// LoadBoard should include config values from an existing board.yaml.
func TestLoadBoard_WithConfigFile(t *testing.T) {
	root := t.TempDir()
	list := filepath.Join(root, "test")
	mustMkdir(t, list)
	mustWrite(t, filepath.Join(list, "1.md"), []byte("---\ntitle: \"T\"\nid: 1\n---\n"))
	mustWrite(t, filepath.Join(root, "board.yaml"), []byte("lists:\n  - dir: test\n    title: \"Custom\"\n    limit: 3\n"))

	app := NewApp()
	resp := app.LoadBoard(root)
	if resp == nil {
		t.Fatal("LoadBoard returned nil")
	}

	idx := daedalus.FindListEntry(resp.Config.Lists, "test")
	if idx < 0 {
		t.Fatal("expected config entry for test")
	}

	lc := resp.Config.Lists[idx]
	if lc.Title != "Custom" || lc.Limit != 3 {
		t.Errorf("got title=%q limit=%d, want title=\"Custom\" limit=3", lc.Title, lc.Limit)
	}
}
