package main

import (
	"os"
	"path/filepath"
	"testing"
)

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

// setupTestBoardMulti creates a board with 3 cards in "open" and an empty "done" list.
func setupTestBoardMulti(t *testing.T) (*App, string) {
	t.Helper()
	root := t.TempDir()
	openList := filepath.Join(root, "open")
	doneList := filepath.Join(root, "done")

	mustMkdir(t, openList)
	mustMkdir(t, doneList)
	mustWrite(t, filepath.Join(openList, "1.md"), []byte("---\ntitle: \"Card A\"\nid: 1\nlist_order: 1\n---\n# Card A\n\nBody A.\n"))
	mustWrite(t, filepath.Join(openList, "2.md"), []byte("---\ntitle: \"Card B\"\nid: 2\nlist_order: 2\n---\n# Card B\n\nBody B.\n"))
	mustWrite(t, filepath.Join(openList, "3.md"), []byte("---\ntitle: \"Card C\"\nid: 3\nlist_order: 3\n---\n# Card C\n\nBody C.\n"))

	app := NewApp()
	resp := app.LoadBoard(root)
	if resp == nil {
		t.Fatal("LoadBoard returned nil")
	}
	return app, root
}

func setupTestBoard(t *testing.T) (*App, string) {
	t.Helper()
	root := t.TempDir()
	list := filepath.Join(root, "test")

	mustMkdir(t, list)
	mustWrite(t, filepath.Join(list, "1.md"), []byte("---\ntitle: \"Test\"\nid: 1\n---\n# Hello\n\nBody content.\n"))

	app := NewApp()
	resp := app.LoadBoard(root)

	if resp == nil {
		t.Fatal("LoadBoard returned nil")
	}
	return app, root
}
