package main

import (
	"daedalus/pkg/daedalus"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestBoard(t *testing.T) (*App, string) {
	t.Helper()
	root := t.TempDir()
	list := filepath.Join(root, "00___test")

	os.Mkdir(list, 0755)
	os.WriteFile(filepath.Join(list, "1.md"), []byte("---\ntitle: \"Test\"\nid: 1\n---\n# Hello\n\nBody content.\n"), 0644)

	app := NewApp()
	resp := app.LoadBoard(root)

	if resp == nil {
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
	resp := app.LoadBoard("/nonexistent/path/that/does/not/exist")
	if resp != nil {
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
	list := filepath.Join(root, "00___open")
	os.Mkdir(list, 0755)
	os.WriteFile(filepath.Join(list, "5.md"), []byte("---\ntitle: \"Card Five\"\nid: 5\nlist_order: 1\n---\nBody\n"), 0644)

	app := NewApp()
	resp := app.LoadBoard(root)
	if resp == nil {
		t.Fatal("LoadBoard returned nil")
	}

	cards, ok := resp.Lists["00___open"]
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

// SaveListConfig should update the in-memory config and persist to board.yaml.
func TestSaveListConfig_Success(t *testing.T) {
	app, root := setupTestBoard(t)

	err := app.SaveListConfig("00___test", "My Test List", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lc, ok := app.board.Config.Lists["00___test"]
	if !ok {
		t.Fatal("expected config entry for 00___test")
	}
	if lc.Title != "My Test List" || lc.Limit != 10 {
		t.Errorf("got title=%q limit=%d, want title=\"My Test List\" limit=10", lc.Title, lc.Limit)
	}

	// Verify file was written
	config, err := daedalus.LoadBoardConfig(root)
	if err != nil {
		t.Fatalf("error loading saved config: %v", err)
	}
	saved := config.Lists["00___test"]
	if saved.Title != "My Test List" || saved.Limit != 10 {
		t.Errorf("saved config: got %+v", saved)
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

// SaveListConfig should return an error when no board has been loaded.
func TestSaveListConfig_BoardNotLoaded(t *testing.T) {
	app := NewApp()
	err := app.SaveListConfig("00___test", "Title", 5)
	if err == nil {
		t.Fatal("expected error when board not loaded")
	}
	if err.Error() != "board not loaded" {
		t.Errorf("unexpected error message: %v", err)
	}
}

// LoadBoard response should include a non-nil config even without a board.yaml file.
func TestLoadBoard_IncludesConfig(t *testing.T) {
	root := t.TempDir()
	list := filepath.Join(root, "00___test")
	os.Mkdir(list, 0755)
	os.WriteFile(filepath.Join(list, "1.md"), []byte("---\ntitle: \"T\"\nid: 1\n---\n"), 0644)

	app := NewApp()
	resp := app.LoadBoard(root)
	if resp == nil {
		t.Fatal("LoadBoard returned nil")
	}
	if resp.Config == nil {
		t.Fatal("expected non-nil Config in response")
	}
	if resp.Config.Lists == nil {
		t.Fatal("expected non-nil Lists map in Config")
	}
}

// LoadBoard should include config values from an existing board.yaml.
func TestLoadBoard_WithConfigFile(t *testing.T) {
	root := t.TempDir()
	list := filepath.Join(root, "00___test")
	os.Mkdir(list, 0755)
	os.WriteFile(filepath.Join(list, "1.md"), []byte("---\ntitle: \"T\"\nid: 1\n---\n"), 0644)
	os.WriteFile(filepath.Join(root, "board.yaml"), []byte("lists:\n  00___test:\n    title: \"Custom\"\n    limit: 3\n"), 0644)

	app := NewApp()
	resp := app.LoadBoard(root)
	if resp == nil {
		t.Fatal("LoadBoard returned nil")
	}

	lc, ok := resp.Config.Lists["00___test"]
	if !ok {
		t.Fatal("expected config entry for 00___test")
	}
	if lc.Title != "Custom" || lc.Limit != 3 {
		t.Errorf("got title=%q limit=%d, want title=\"Custom\" limit=3", lc.Title, lc.Limit)
	}
}

// CreateCard should increment MaxID, write the file to disk, and prepend the card to the list.
func TestCreateCard_Success(t *testing.T) {
	app, root := setupTestBoard(t)

	oldMaxID := app.board.MaxID
	card, err := app.CreateCard("00___test", "New Card", "Some body text")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if card.Metadata.ID != oldMaxID+1 {
		t.Errorf("ID: got %d, want %d", card.Metadata.ID, oldMaxID+1)
	}
	if app.board.MaxID != oldMaxID+1 {
		t.Errorf("MaxID: got %d, want %d", app.board.MaxID, oldMaxID+1)
	}

	// File should exist on disk
	expectedPath := filepath.Join(root, "00___test", fmt.Sprintf("%d.md", card.Metadata.ID))
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Error("expected card file to exist on disk")
	}

	// Card should be prepended (first in list)
	cards := app.board.Lists["00___test"]
	if len(cards) < 2 {
		t.Fatalf("expected at least 2 cards, got %d", len(cards))
	}
	if cards[0].Metadata.ID != card.Metadata.ID {
		t.Errorf("new card should be first in list, got ID %d", cards[0].Metadata.ID)
	}

	// New card's list_order should be less than the original card's
	if cards[0].Metadata.ListOrder >= cards[1].Metadata.ListOrder {
		t.Errorf("new card list_order (%f) should be less than existing (%f)",
			cards[0].Metadata.ListOrder, cards[1].Metadata.ListOrder)
	}

	// ListName should be set
	if card.ListName != "00___test" {
		t.Errorf("ListName: got %q, want %q", card.ListName, "00___test")
	}

	// Title should match the provided title
	if card.Metadata.Title != "New Card" {
		t.Errorf("Title: got %q, want %q", card.Metadata.Title, "New Card")
	}

	// PreviewText should match the provided body
	if card.PreviewText != "Some body text" {
		t.Errorf("PreviewText: got %q, want %q", card.PreviewText, "Some body text")
	}
}

// CreateCard should return an error when no board has been loaded.
func TestCreateCard_BoardNotLoaded(t *testing.T) {
	app := NewApp()
	_, err := app.CreateCard("00___test", "", "")
	if err == nil {
		t.Fatal("expected error when board not loaded")
	}
	if err.Error() != "board not loaded" {
		t.Errorf("unexpected error message: %v", err)
	}
}

// CreateCard should return an error for a nonexistent list directory.
func TestCreateCard_InvalidList(t *testing.T) {
	app, _ := setupTestBoard(t)

	_, err := app.CreateCard("99___nonexistent", "", "")
	if err == nil {
		t.Fatal("expected error for invalid list")
	}
	if !strings.Contains(err.Error(), "list not found") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// CreateCard with an empty title should fall back to using the ID as the title.
func TestCreateCard_EmptyTitle(t *testing.T) {
	app, _ := setupTestBoard(t)

	card, err := app.CreateCard("00___test", "", "some body")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := fmt.Sprintf("%d", card.Metadata.ID)
	if card.Metadata.Title != expected {
		t.Errorf("Title: got %q, want %q", card.Metadata.Title, expected)
	}
}

// CreateCard should truncate the preview text at 150 characters for long bodies.
func TestCreateCard_LongBodyPreview(t *testing.T) {
	app, _ := setupTestBoard(t)

	longBody := strings.Repeat("x", 300)
	card, err := app.CreateCard("00___test", "Title", longBody)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(card.PreviewText) != 150 {
		t.Errorf("PreviewText length: got %d, want 150", len(card.PreviewText))
	}
}

// DeleteCard should remove the file from disk, remove it from in-memory lists, and leave MaxID unchanged.
func TestDeleteCard_Success(t *testing.T) {
	app, root := setupTestBoard(t)

	cardPath := filepath.Join(root, "00___test", "1.md")
	oldMaxID := app.board.MaxID

	err := app.DeleteCard(cardPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// File should be gone from disk
	if _, err := os.Stat(cardPath); !os.IsNotExist(err) {
		t.Error("expected card file to be deleted from disk")
	}

	// Card should be removed from in-memory lists
	for _, cards := range app.board.Lists {
		for _, card := range cards {
			if card.FilePath == cardPath {
				t.Error("card should not exist in board lists after delete")
			}
		}
	}

	// MaxID should NOT be decremented
	if app.board.MaxID != oldMaxID {
		t.Errorf("MaxID changed: got %d, want %d", app.board.MaxID, oldMaxID)
	}
}

// DeleteCard should reject paths that escape the board root directory.
func TestDeleteCard_PathTraversal(t *testing.T) {
	app, root := setupTestBoard(t)
	outsidePath := filepath.Join(root, "..", "evil.md")

	err := app.DeleteCard(outsidePath)
	if err == nil {
		t.Fatal("expected error for path traversal")
	}
	if err.Error() != "path outside board directory" {
		t.Errorf("unexpected error message: %v", err)
	}
}

// DeleteCard should return an error when no board has been loaded.
func TestDeleteCard_BoardNotLoaded(t *testing.T) {
	app := NewApp()
	err := app.DeleteCard("/some/path.md")
	if err == nil {
		t.Fatal("expected error when board not loaded")
	}
	if err.Error() != "board not loaded" {
		t.Errorf("unexpected error message: %v", err)
	}
}

// DeleteCard should return an error when the file does not exist.
func TestDeleteCard_NonexistentFile(t *testing.T) {
	app, root := setupTestBoard(t)
	badPath := filepath.Join(root, "00___test", "999.md")

	err := app.DeleteCard(badPath)
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
	if !strings.Contains(err.Error(), "removing card file") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// DeleteCard should decrement TotalFileBytes by the size of the removed file.
func TestDeleteCard_UpdatesTotalFileBytes(t *testing.T) {
	app, root := setupTestBoard(t)

	cardPath := filepath.Join(root, "00___test", "1.md")
	info, err := os.Stat(cardPath)
	if err != nil {
		t.Fatalf("could not stat card file: %v", err)
	}
	fileSize := info.Size()
	bytesBefore := app.board.TotalFileBytes

	err = app.DeleteCard(cardPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := bytesBefore - fileSize
	if app.board.TotalFileBytes != expected {
		t.Errorf("TotalFileBytes: got %d, want %d", app.board.TotalFileBytes, expected)
	}
}

// SaveCard should write updated title and body to disk and return the updated card.
func TestSaveCard_Success(t *testing.T) {
	app, root := setupTestBoard(t)
	cardPath := filepath.Join(root, "00___test", "1.md")

	meta := daedalus.CardMetadata{
		ID:        1,
		Title:     "Updated Title",
		ListOrder: 1,
	}
	result, err := app.SaveCard(cardPath, meta, "# Updated Title\n\nNew body.\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Metadata.Title != "Updated Title" {
		t.Errorf("title: got %q, want %q", result.Metadata.Title, "Updated Title")
	}

	// Read file back to verify
	content, err := os.ReadFile(cardPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "Updated Title") {
		t.Error("file should contain updated title")
	}
	if !strings.Contains(string(content), "New body.") {
		t.Error("file should contain new body")
	}
}

// SaveCard should reject paths that escape the board root directory.
func TestSaveCard_PathTraversal(t *testing.T) {
	app, root := setupTestBoard(t)
	outsidePath := filepath.Join(root, "..", "evil.md")

	meta := daedalus.CardMetadata{ID: 1, Title: "Evil"}
	_, err := app.SaveCard(outsidePath, meta, "hacked")
	if err == nil {
		t.Fatal("expected error for path traversal")
	}
	if err.Error() != "path outside board directory" {
		t.Errorf("unexpected error message: %v", err)
	}
}

// SaveCard should return an error when the board has not been loaded.
func TestSaveCard_BoardNotLoaded(t *testing.T) {
	app := NewApp()
	meta := daedalus.CardMetadata{ID: 1, Title: "Test"}
	_, err := app.SaveCard("/some/path.md", meta, "body")
	if err == nil {
		t.Fatal("expected error when board not loaded")
	}
	if err.Error() != "board not loaded" {
		t.Errorf("unexpected error message: %v", err)
	}
}

// After SaveCard, the in-memory board state should reflect the updated card data.
func TestSaveCard_UpdatesInMemory(t *testing.T) {
	app, root := setupTestBoard(t)
	cardPath := filepath.Join(root, "00___test", "1.md")

	meta := daedalus.CardMetadata{
		ID:        1,
		Title:     "Memory Update",
		ListOrder: 1,
	}

	_, err := app.SaveCard(cardPath, meta, "# Memory Update\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cards := app.board.Lists["00___test"]
	found := false
	for _, card := range cards {
		if card.FilePath == cardPath {
			found = true
			if card.Metadata.Title != "Memory Update" {
				t.Errorf("in-memory title: got %q, want %q", card.Metadata.Title, "Memory Update")
			}
			if card.Metadata.Updated == nil {
				t.Error("expected Updated timestamp to be set")
			}
		}
	}
	if !found {
		t.Error("card not found in board lists after save")
	}
}
