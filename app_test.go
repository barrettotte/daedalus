package main

import (
	"daedalus/pkg/daedalus"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupTestBoardMulti creates a board with 3 cards in 00___open and an empty 10___done list.
func setupTestBoardMulti(t *testing.T) (*App, string) {
	t.Helper()
	root := t.TempDir()
	openList := filepath.Join(root, "00___open")
	doneList := filepath.Join(root, "10___done")

	os.Mkdir(openList, 0755)
	os.Mkdir(doneList, 0755)
	os.WriteFile(filepath.Join(openList, "1.md"), []byte("---\ntitle: \"Card A\"\nid: 1\nlist_order: 1\n---\n# Card A\n\nBody A.\n"), 0644)
	os.WriteFile(filepath.Join(openList, "2.md"), []byte("---\ntitle: \"Card B\"\nid: 2\nlist_order: 2\n---\n# Card B\n\nBody B.\n"), 0644)
	os.WriteFile(filepath.Join(openList, "3.md"), []byte("---\ntitle: \"Card C\"\nid: 3\nlist_order: 3\n---\n# Card C\n\nBody C.\n"), 0644)

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

// SaveHalfCollapsedLists should persist the list to board.yaml and reload correctly.
func TestSaveHalfCollapsedLists_Success(t *testing.T) {
	app, root := setupTestBoard(t)

	lists := []string{"00___test", "10___done"}
	if err := app.SaveHalfCollapsedLists(lists); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	config, err := daedalus.LoadBoardConfig(root)
	if err != nil {
		t.Fatalf("error loading config: %v", err)
	}
	if len(config.HalfCollapsedLists) != 2 {
		t.Fatalf("expected 2 half-collapsed lists, got %d", len(config.HalfCollapsedLists))
	}
	if config.HalfCollapsedLists[0] != "00___test" || config.HalfCollapsedLists[1] != "10___done" {
		t.Errorf("unexpected half-collapsed lists: %v", config.HalfCollapsedLists)
	}

	// Clear and verify empty
	if err := app.SaveHalfCollapsedLists(nil); err != nil {
		t.Fatalf("unexpected error clearing: %v", err)
	}

	config, err = daedalus.LoadBoardConfig(root)
	if err != nil {
		t.Fatalf("error loading config: %v", err)
	}
	if len(config.HalfCollapsedLists) != 0 {
		t.Errorf("expected empty half-collapsed lists, got %v", config.HalfCollapsedLists)
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
	card, err := app.CreateCard("00___test", "New Card", "Some body text", "top")
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
	_, err := app.CreateCard("00___test", "", "", "top")
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

	_, err := app.CreateCard("99___nonexistent", "", "", "top")
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

	card, err := app.CreateCard("00___test", "", "some body", "top")
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
	card, err := app.CreateCard("00___test", "Title", longBody, "top")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(card.PreviewText) != 150 {
		t.Errorf("PreviewText length: got %d, want 150", len(card.PreviewText))
	}
}

// CreateCard with position "bottom" should place the new card after all existing cards.
func TestCreateCard_Bottom(t *testing.T) {
	app, root := setupTestBoard(t)

	card, err := app.CreateCard("00___test", "Bottom Card", "bottom body", "bottom")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// File should exist on disk
	expectedPath := filepath.Join(root, "00___test", fmt.Sprintf("%d.md", card.Metadata.ID))
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Error("expected card file to exist on disk")
	}

	// Card should be last in the list
	cards := app.board.Lists["00___test"]
	if len(cards) < 2 {
		t.Fatalf("expected at least 2 cards, got %d", len(cards))
	}
	last := cards[len(cards)-1]
	if last.Metadata.ID != card.Metadata.ID {
		t.Errorf("new card should be last in list, got ID %d", last.Metadata.ID)
	}

	// New card's list_order should be greater than the previous last card's
	prev := cards[len(cards)-2]
	if last.Metadata.ListOrder <= prev.Metadata.ListOrder {
		t.Errorf("new card list_order (%f) should be greater than previous last (%f)",
			last.Metadata.ListOrder, prev.Metadata.ListOrder)
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

// CreateCard with a numeric position "1" should insert between the first and second cards.
func TestCreateCard_NumericMiddle(t *testing.T) {
	app, _ := setupTestBoardMulti(t)

	card, err := app.CreateCard("00___open", "Middle Card", "body", "1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// list_order should be midpoint of cards[0] (1.0) and cards[1] (2.0)
	if card.Metadata.ListOrder != 1.5 {
		t.Errorf("ListOrder: got %f, want 1.5", card.Metadata.ListOrder)
	}

	cards := app.board.Lists["00___open"]
	if len(cards) != 4 {
		t.Fatalf("expected 4 cards, got %d", len(cards))
	}
	if cards[1].Metadata.ID != card.Metadata.ID {
		t.Errorf("new card should be at index 1, got ID %d", cards[1].Metadata.ID)
	}
}

// CreateCard with numeric position "0" should behave like top.
func TestCreateCard_NumericZero(t *testing.T) {
	app, _ := setupTestBoardMulti(t)

	card, err := app.CreateCard("00___open", "Zero Card", "body", "0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cards := app.board.Lists["00___open"]
	if cards[0].Metadata.ID != card.Metadata.ID {
		t.Errorf("new card should be first, got ID %d", cards[0].Metadata.ID)
	}
	if card.Metadata.ListOrder >= cards[1].Metadata.ListOrder {
		t.Errorf("list_order (%f) should be less than first existing (%f)",
			card.Metadata.ListOrder, cards[1].Metadata.ListOrder)
	}
}

// CreateCard with a numeric position beyond the end should behave like bottom.
func TestCreateCard_NumericBeyondEnd(t *testing.T) {
	app, _ := setupTestBoardMulti(t)

	card, err := app.CreateCard("00___open", "Beyond Card", "body", "99")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cards := app.board.Lists["00___open"]
	last := cards[len(cards)-1]
	if last.Metadata.ID != card.Metadata.ID {
		t.Errorf("new card should be last, got ID %d", last.Metadata.ID)
	}
	prev := cards[len(cards)-2]
	if card.Metadata.ListOrder <= prev.Metadata.ListOrder {
		t.Errorf("list_order (%f) should be greater than previous last (%f)",
			card.Metadata.ListOrder, prev.Metadata.ListOrder)
	}
}

// CreateCard with a numeric position in an empty list should default to list_order 0.
func TestCreateCard_NumericEmptyList(t *testing.T) {
	app, _ := setupTestBoardMulti(t)

	card, err := app.CreateCard("10___done", "Empty List Card", "body", "5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if card.Metadata.ListOrder != 0 {
		t.Errorf("ListOrder: got %f, want 0", card.Metadata.ListOrder)
	}

	cards := app.board.Lists["10___done"]
	if len(cards) != 1 {
		t.Fatalf("expected 1 card, got %d", len(cards))
	}
}

// MoveCard should reorder a card within the same list by updating list_order.
func TestMoveCard_SameList(t *testing.T) {
	app, _ := setupTestBoardMulti(t)

	// Move card 3 (list_order=3) to between card A and card B (list_order=1.5)
	cardPath := app.board.Lists["00___open"][2].FilePath
	result, err := app.MoveCard(cardPath, "00___open", 1.5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Metadata.ListOrder != 1.5 {
		t.Errorf("ListOrder: got %f, want 1.5", result.Metadata.ListOrder)
	}

	// Verify in-memory order: A(1), C(1.5), B(2)
	cards := app.board.Lists["00___open"]
	if len(cards) != 3 {
		t.Fatalf("expected 3 cards, got %d", len(cards))
	}
	if cards[0].Metadata.ID != 1 || cards[1].Metadata.ID != 3 || cards[2].Metadata.ID != 2 {
		t.Errorf("unexpected order: IDs %d, %d, %d", cards[0].Metadata.ID, cards[1].Metadata.ID, cards[2].Metadata.ID)
	}
}

// MoveCard should move a card between lists, renaming the file on disk.
func TestMoveCard_CrossList(t *testing.T) {
	app, root := setupTestBoardMulti(t)

	cardPath := app.board.Lists["00___open"][0].FilePath
	result, err := app.MoveCard(cardPath, "10___done", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// File should exist in new directory
	newPath := filepath.Join(root, "10___done", "1.md")
	if _, err := os.Stat(newPath); os.IsNotExist(err) {
		t.Error("expected file to exist in target directory")
	}

	// File should be gone from old directory
	if _, err := os.Stat(cardPath); !os.IsNotExist(err) {
		t.Error("expected file to be removed from source directory")
	}

	// Source list should have 2 cards, target should have 1
	if len(app.board.Lists["00___open"]) != 2 {
		t.Errorf("source list: got %d cards, want 2", len(app.board.Lists["00___open"]))
	}
	if len(app.board.Lists["10___done"]) != 1 {
		t.Errorf("target list: got %d cards, want 1", len(app.board.Lists["10___done"]))
	}

	if result.FilePath != newPath {
		t.Errorf("FilePath: got %q, want %q", result.FilePath, newPath)
	}
}

// MoveCard should reject paths that escape the board root.
func TestMoveCard_PathTraversal(t *testing.T) {
	app, root := setupTestBoardMulti(t)
	outsidePath := filepath.Join(root, "..", "evil.md")

	_, err := app.MoveCard(outsidePath, "00___open", 0)
	if err == nil {
		t.Fatal("expected error for path traversal")
	}
	if err.Error() != "path outside board directory" {
		t.Errorf("unexpected error message: %v", err)
	}
}

// MoveCard should return an error when no board has been loaded.
func TestMoveCard_BoardNotLoaded(t *testing.T) {
	app := NewApp()
	_, err := app.MoveCard("/some/path.md", "00___open", 0)
	if err == nil {
		t.Fatal("expected error when board not loaded")
	}
	if err.Error() != "board not loaded" {
		t.Errorf("unexpected error message: %v", err)
	}
}

// MoveCard should return an error for a nonexistent target list.
func TestMoveCard_InvalidTargetList(t *testing.T) {
	app, _ := setupTestBoardMulti(t)
	cardPath := app.board.Lists["00___open"][0].FilePath

	_, err := app.MoveCard(cardPath, "99___nonexistent", 0)
	if err == nil {
		t.Fatal("expected error for invalid target list")
	}
	if !strings.Contains(err.Error(), "target list not found") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// SaveListOrder should persist the order to board.yaml and update in-memory config.
func TestSaveListOrder_Success(t *testing.T) {
	app, root := setupTestBoardMulti(t)

	order := []string{"10___done", "00___open"}
	if err := app.SaveListOrder(order); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(app.board.Config.ListOrder) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(app.board.Config.ListOrder))
	}
	if app.board.Config.ListOrder[0] != "10___done" || app.board.Config.ListOrder[1] != "00___open" {
		t.Errorf("unexpected in-memory order: %v", app.board.Config.ListOrder)
	}

	// Verify persisted to disk
	config, err := daedalus.LoadBoardConfig(root)
	if err != nil {
		t.Fatalf("error loading config: %v", err)
	}
	if len(config.ListOrder) != 2 {
		t.Fatalf("expected 2 persisted entries, got %d", len(config.ListOrder))
	}
	if config.ListOrder[0] != "10___done" {
		t.Errorf("unexpected persisted order: %v", config.ListOrder)
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

// DeleteList should remove the directory, cards, and all config references.
func TestDeleteList_Success(t *testing.T) {
	app, root := setupTestBoardMulti(t)

	// Verify list exists before delete
	if _, ok := app.board.Lists["00___open"]; !ok {
		t.Fatal("expected 00___open to exist before delete")
	}
	bytesBefore := app.board.TotalFileBytes

	err := app.DeleteList("00___open")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Directory should be gone from disk
	dirPath := filepath.Join(root, "00___open")
	if _, err := os.Stat(dirPath); !os.IsNotExist(err) {
		t.Error("expected directory to be removed from disk")
	}

	// List should be gone from in-memory state
	if _, ok := app.board.Lists["00___open"]; ok {
		t.Error("expected 00___open to be removed from board.Lists")
	}

	// TotalFileBytes should have decreased
	if app.board.TotalFileBytes >= bytesBefore {
		t.Errorf("TotalFileBytes should have decreased: before=%d, after=%d", bytesBefore, app.board.TotalFileBytes)
	}
}

// DeleteList should return an error for a nonexistent list.
func TestDeleteList_NotFound(t *testing.T) {
	app, _ := setupTestBoardMulti(t)

	err := app.DeleteList("99___nonexistent")
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
	err := app.DeleteList("00___open")
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

// DeleteList should remove the list from ListOrder in config.
func TestDeleteList_CleansListOrder(t *testing.T) {
	app, _ := setupTestBoardMulti(t)

	// Set up a custom list order that includes the list we'll delete
	app.board.Config.ListOrder = []string{"00___open", "10___done"}

	err := app.DeleteList("00___open")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(app.board.Config.ListOrder) != 1 {
		t.Fatalf("expected 1 entry in ListOrder, got %d", len(app.board.Config.ListOrder))
	}
	if app.board.Config.ListOrder[0] != "10___done" {
		t.Errorf("unexpected ListOrder: %v", app.board.Config.ListOrder)
	}
}

// MoveCard should return an error when the card is not found in any list.
func TestMoveCard_CardNotFound(t *testing.T) {
	app, root := setupTestBoardMulti(t)
	// Valid path within board root but not a card in any list
	fakePath := filepath.Join(root, "00___open", "999.md")

	_, err := app.MoveCard(fakePath, "00___open", 0)
	if err == nil {
		t.Fatal("expected error for card not found")
	}
	if !strings.Contains(err.Error(), "card not found") {
		t.Errorf("unexpected error message: %v", err)
	}
}
