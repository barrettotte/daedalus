package main

import (
	"daedalus/pkg/daedalus"
	"path/filepath"
	"testing"
)

// Calling LoadBoard with an empty string should return nil.
func TestLoadBoard_DefaultPath(t *testing.T) {
	app := NewApp()
	resp := app.LoadBoard("")
	if resp != nil {
		t.Error("expected nil for empty path")
	}
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

func TestGetAppConfig_ReturnsConfig(t *testing.T) {
	app := NewApp()
	app.appConfigDir = t.TempDir()
	cfg := app.GetAppConfig()
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
}

func TestSetDefaultBoard(t *testing.T) {
	app := NewApp()
	app.appConfigDir = t.TempDir()

	// Use a real directory so PruneInvalidBoards doesn't clear it.
	boardDir := t.TempDir()
	err := app.SetDefaultBoard(boardDir)
	if err != nil {
		t.Fatal(err)
	}

	cfg := app.GetAppConfig()
	if cfg.DefaultBoard != boardDir {
		t.Errorf("expected %q, got %q", boardDir, cfg.DefaultBoard)
	}
}

func TestSetDefaultBoard_Clear(t *testing.T) {
	app := NewApp()
	app.appConfigDir = t.TempDir()

	boardDir := t.TempDir()
	app.SetDefaultBoard(boardDir)
	err := app.SetDefaultBoard("")
	if err != nil {
		t.Fatal(err)
	}

	cfg := app.GetAppConfig()
	if cfg.DefaultBoard != "" {
		t.Errorf("expected empty default, got %q", cfg.DefaultBoard)
	}
}

func TestGetAppConfig_PrunesInvalidBoards(t *testing.T) {
	app := NewApp()
	app.appConfigDir = t.TempDir()

	validDir := t.TempDir()
	cfg := &daedalus.AppConfig{
		DefaultBoard: "/nonexistent/path",
		RecentBoards: []daedalus.RecentBoard{
			{Path: validDir},
			{Path: "/another/nonexistent/path"},
		},
	}
	if err := daedalus.SaveAppConfig(app.appConfigDir, cfg); err != nil {
		t.Fatal(err)
	}

	result := app.GetAppConfig()
	if result.DefaultBoard != "" {
		t.Errorf("invalid default board should be cleared, got %q", result.DefaultBoard)
	}
	if len(result.RecentBoards) != 1 {
		t.Fatalf("expected 1 valid board after prune, got %d", len(result.RecentBoards))
	}
	if result.RecentBoards[0].Path != validDir {
		t.Errorf("valid board should remain, got %q", result.RecentBoards[0].Path)
	}
}

func TestRemoveRecentBoard(t *testing.T) {
	app := NewApp()
	app.appConfigDir = t.TempDir()

	// Seed a recent board via LoadBoard.
	root := t.TempDir()
	list := filepath.Join(root, "test")
	mustMkdir(t, list)
	mustWrite(t, filepath.Join(list, "1.md"), []byte("---\ntitle: \"T\"\nid: 1\n---\n"))
	app.LoadBoard(root)

	cfg := app.GetAppConfig()
	if len(cfg.RecentBoards) == 0 {
		t.Fatal("expected at least 1 recent board after LoadBoard")
	}

	absRoot, _ := filepath.Abs(root)
	err := app.RemoveRecentBoard(absRoot)
	if err != nil {
		t.Fatal(err)
	}

	cfg = app.GetAppConfig()
	for _, rb := range cfg.RecentBoards {
		if rb.Path == absRoot {
			t.Error("board should have been removed from recent list")
		}
	}
}

func TestLoadBoard_EmptyDir(t *testing.T) {
	app := NewApp()
	app.appConfigDir = t.TempDir()
	root := t.TempDir()
	resp := app.LoadBoard(root)
	if resp == nil {
		t.Fatal("LoadBoard should succeed on empty dir by initializing board.yaml")
	}
	if resp.Config == nil {
		t.Fatal("expected non-nil config")
	}
}

func TestLoadBoard_AddsToRecent(t *testing.T) {
	app := NewApp()
	app.appConfigDir = t.TempDir()

	root := t.TempDir()
	list := filepath.Join(root, "test")
	mustMkdir(t, list)
	mustWrite(t, filepath.Join(list, "1.md"), []byte("---\ntitle: \"T\"\nid: 1\n---\n"))

	resp := app.LoadBoard(root)
	if resp == nil {
		t.Fatal("LoadBoard returned nil")
	}

	cfg := app.GetAppConfig()
	absRoot, _ := filepath.Abs(root)
	found := false
	for _, rb := range cfg.RecentBoards {
		if rb.Path == absRoot {
			found = true
			break
		}
	}
	if !found {
		t.Error("board should appear in recent list after LoadBoard")
	}
}
