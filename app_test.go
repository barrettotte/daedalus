package main

import (
	"daedalus/pkg/daedalus"
	"os"
	"path/filepath"
	"testing"
)

func setupTestBoard(t *testing.T) (*App, string) {
	t.Helper()
	root := t.TempDir()
	list := filepath.Join(root, "00___test")

	os.Mkdir(list, 0755)
	os.WriteFile(filepath.Join(list, "1.md"), []byte("---\ntitle: \"Test\"\nid: 1\n---\n# Hello\n\nBody content.\n"), 0644)

	app := NewApp()
	result := app.LoadBoard(root)

	if result == nil {
		t.Fatal("LoadBoard returned nil")
	}
	return app, root
}

// GetCardContent should return the full markdown body for a valid card path.
func TestGetCardContent_Success(t *testing.T) {
	app, root := setupTestBoard(t)
	cardPath := filepath.Join(root, "00___test", "1.md")

	content, err := app.GetCardContent(cardPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "# Hello\n\nBody content.\n"
	if content != expected {
		t.Errorf("got %q, want %q", content, expected)
	}
}

// Paths that escape the board root via .. should be rejected.
func TestGetCardContent_PathTraversal(t *testing.T) {
	app, root := setupTestBoard(t)
	outsideFile := filepath.Join(root, "..", "etc", "passwd")

	_, err := app.GetCardContent(outsideFile)
	if err == nil {
		t.Fatal("expected error for path traversal attempt")
	}
	if err.Error() != "path outside board directory" {
		t.Errorf("unexpected error message: %v", err)
	}
}

// GetCardContent should fail when no board has been loaded yet.
func TestGetCardContent_BoardNotLoaded(t *testing.T) {
	app := NewApp()
	_, err := app.GetCardContent("/some/path.md")
	if err == nil {
		t.Fatal("expected error when board not loaded")
	}
	if err.Error() != "board not loaded" {
		t.Errorf("unexpected error message: %v", err)
	}
}

// Requesting a card file that doesn't exist on disk should return an error.
func TestGetCardContent_NonexistentFile(t *testing.T) {
	app, root := setupTestBoard(t)
	badPath := filepath.Join(root, "00___test", "999.md")

	_, err := app.GetCardContent(badPath)
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

// Paths using .. segments that resolve outside the board root should be blocked,
// even when they start inside a valid list directory.
func TestGetCardContent_DotDotInPath(t *testing.T) {
	app, root := setupTestBoard(t)
	traversal := filepath.Join(root, "00___test", "..", "..", "secret.md")

	_, err := app.GetCardContent(traversal)
	if err == nil {
		t.Fatal("expected error for .. path traversal")
	}
}

// Calling LoadBoard with an empty string should use the default path without panicking.
func TestLoadBoard_DefaultPath(t *testing.T) {
	app := NewApp()
	app.LoadBoard("")
}

// LoadBoard should return nil when given a nonexistent directory.
func TestLoadBoard_InvalidPath(t *testing.T) {
	app := NewApp()
	result := app.LoadBoard("/nonexistent/path/that/does/not/exist")
	if result != nil {
		t.Error("expected nil for invalid path")
	}
}

// After a successful LoadBoard, the internal board state should be populated
// with the correct RootPath.
func TestLoadBoard_SetsBoard(t *testing.T) {
	root := t.TempDir()
	list := filepath.Join(root, "00___test")
	os.Mkdir(list, 0755)
	os.WriteFile(filepath.Join(list, "1.md"), []byte("---\ntitle: \"T\"\nid: 1\n---\n"), 0644)

	app := NewApp()
	result := app.LoadBoard(root)
	if result == nil {
		t.Fatal("LoadBoard returned nil")
	}
	if app.board == nil {
		t.Fatal("board should be set after LoadBoard")
	}
	if app.board.RootPath != root {
		t.Errorf("RootPath: got %q, want %q", app.board.RootPath, root)
	}
}

// The map returned by LoadBoard should contain the correct list keys and card metadata.
func TestLoadBoard_ReturnedData(t *testing.T) {
	root := t.TempDir()
	list := filepath.Join(root, "00___open")
	os.Mkdir(list, 0755)
	os.WriteFile(filepath.Join(list, "5.md"), []byte("---\ntitle: \"Card Five\"\nid: 5\nlist_order: 1\n---\nBody\n"), 0644)

	app := NewApp()
	result := app.LoadBoard(root)
	if result == nil {
		t.Fatal("LoadBoard returned nil")
	}

	cards, ok := result["00___open"]
	if !ok {
		t.Fatal("expected 00___open key in result")
	}
	if len(cards) != 1 {
		t.Fatalf("expected 1 card, got %d", len(cards))
	}
	if cards[0].Metadata.Title != "Card Five" {
		t.Errorf("title: got %q, want %q", cards[0].Metadata.Title, "Card Five")
	}
}

// Passing the board root directory itself (not a file inside it) should be rejected.
func TestGetCardContent_ExactRootPath(t *testing.T) {
	app, root := setupTestBoard(t)
	_, err := app.GetCardContent(root)
	if err == nil {
		t.Fatal("expected error when path is the root itself")
	}
}

// A relative path that resolves outside the board root should be rejected.
func TestGetCardContent_RelativePath(t *testing.T) {
	root := t.TempDir()
	list := filepath.Join(root, "00___test")
	os.Mkdir(list, 0755)
	os.WriteFile(filepath.Join(list, "1.md"), []byte("---\ntitle: \"T\"\nid: 1\n---\nBody\n"), 0644)

	app := NewApp()
	app.board = &daedalus.BoardState{
		RootPath: root,
	}

	_, err := app.GetCardContent("./1.md")
	if err == nil {
		t.Fatal("expected error for relative path outside board")
	}
}
