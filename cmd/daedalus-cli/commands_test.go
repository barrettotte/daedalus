package main

import (
	"archive/zip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"daedalus/pkg/daedalus"
)

// setupTestBoard creates a temp board directory with one list ("open")
// containing one card (id=1).
func setupTestBoard(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := daedalus.InitBoardDir(dir); err != nil {
		t.Fatalf("InitBoardDir: %v", err)
	}

	// Create list directory
	listDir := filepath.Join(dir, "open")
	if err := os.MkdirAll(listDir, 0755); err != nil {
		t.Fatalf("creating list dir: %v", err)
	}

	// Write a card file
	meta := daedalus.CardMetadata{
		ID:        1,
		Title:     "Test Card",
		ListOrder: 1.0,
	}
	cardPath := filepath.Join(listDir, "1.md")
	if err := daedalus.WriteCardFile(cardPath, meta, "# Test Card\n\nCard body.\n"); err != nil {
		t.Fatalf("WriteCardFile: %v", err)
	}

	// Write board.yaml with title and list entry
	config := &daedalus.BoardConfig{
		Title: "Test Board",
		Lists: []daedalus.ListEntry{{Dir: "open"}},
	}
	if err := daedalus.SaveBoardConfig(dir, config); err != nil {
		t.Fatalf("SaveBoardConfig: %v", err)
	}

	return dir
}

// captureStdout redirects os.Stdout to a pipe and returns the captured output
// after calling fn. This allows testing functions that print to stdout.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("reading pipe: %v", err)
	}
	return string(data)
}

func TestCmdBoard(t *testing.T) {
	dir := setupTestBoard(t)
	output := captureStdout(t, func() {
		if err := cmdBoard(dir); err != nil {
			t.Fatalf("cmdBoard: %v", err)
		}
	})

	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, output)
	}

	if result["title"] != "Test Board" {
		t.Errorf("title: got %v, want %q", result["title"], "Test Board")
	}
	// JSON numbers are float64
	if result["cards"] != float64(1) {
		t.Errorf("cards: got %v, want 1", result["cards"])
	}
	if result["lists"] != float64(1) {
		t.Errorf("lists: got %v, want 1", result["lists"])
	}
}

func TestCmdLists(t *testing.T) {
	dir := setupTestBoard(t)
	output := captureStdout(t, func() {
		if err := cmdLists(dir); err != nil {
			t.Fatalf("cmdLists: %v", err)
		}
	})

	var result []map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, output)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 list, got %d", len(result))
	}
	if result[0]["dir"] != "open" {
		t.Errorf("dir: got %v, want %q", result[0]["dir"], "open")
	}
	if result[0]["cards"] != float64(1) {
		t.Errorf("cards: got %v, want 1", result[0]["cards"])
	}
}

func TestCmdCards(t *testing.T) {
	dir := setupTestBoard(t)
	output := captureStdout(t, func() {
		if err := cmdCards(dir, []string{"open"}); err != nil {
			t.Fatalf("cmdCards: %v", err)
		}
	})

	var result []map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, output)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 card, got %d", len(result))
	}
	if result[0]["id"] != float64(1) {
		t.Errorf("id: got %v, want 1", result[0]["id"])
	}
	if result[0]["title"] != "Test Card" {
		t.Errorf("title: got %v, want %q", result[0]["title"], "Test Card")
	}
}

func TestCmdCards_InvalidList(t *testing.T) {
	dir := setupTestBoard(t)
	err := cmdCards(dir, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for invalid list")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found', got: %v", err)
	}
}

func TestCmdCardGet(t *testing.T) {
	dir := setupTestBoard(t)
	output := captureStdout(t, func() {
		if err := cmdCardGet(dir, []string{"1"}); err != nil {
			t.Fatalf("cmdCardGet: %v", err)
		}
	})

	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, output)
	}

	if result["id"] != float64(1) {
		t.Errorf("id: got %v, want 1", result["id"])
	}
	if result["title"] != "Test Card" {
		t.Errorf("title: got %v, want %q", result["title"], "Test Card")
	}
	body, ok := result["body"].(string)
	if !ok || body == "" {
		t.Error("expected non-empty body")
	}
}

func TestCmdCardGet_NotFound(t *testing.T) {
	dir := setupTestBoard(t)
	err := cmdCardGet(dir, []string{"999"})
	if err == nil {
		t.Fatal("expected error for missing card")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found', got: %v", err)
	}
}

func TestCmdCardCreate(t *testing.T) {
	dir := setupTestBoard(t)
	output := captureStdout(t, func() {
		if err := cmdCardCreate(dir, []string{"open", "New Card"}); err != nil {
			t.Fatalf("cmdCardCreate: %v", err)
		}
	})

	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, output)
	}

	if result["id"] != float64(2) {
		t.Errorf("id: got %v, want 2", result["id"])
	}
	if result["title"] != "New Card" {
		t.Errorf("title: got %v, want %q", result["title"], "New Card")
	}
	if result["list"] != "open" {
		t.Errorf("list: got %v, want %q", result["list"], "open")
	}

	// Verify file exists on disk
	cardPath := filepath.Join(dir, "open", "2.md")
	if _, err := os.Stat(cardPath); os.IsNotExist(err) {
		t.Errorf("card file not created at %s", cardPath)
	}
}

func TestCmdCardDelete(t *testing.T) {
	dir := setupTestBoard(t)
	cardPath := filepath.Join(dir, "open", "1.md")

	// Verify file exists before delete
	if _, err := os.Stat(cardPath); os.IsNotExist(err) {
		t.Fatalf("card file should exist before delete")
	}

	if err := cmdCardDelete(dir, []string{"1"}); err != nil {
		t.Fatalf("cmdCardDelete: %v", err)
	}

	// Verify file is removed
	if _, err := os.Stat(cardPath); !os.IsNotExist(err) {
		t.Error("card file should be removed after delete")
	}
}

func TestCmdCardDelete_NotFound(t *testing.T) {
	dir := setupTestBoard(t)
	err := cmdCardDelete(dir, []string{"999"})
	if err == nil {
		t.Fatal("expected error for missing card")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found', got: %v", err)
	}
}

func TestCmdListCreate(t *testing.T) {
	dir := setupTestBoard(t)
	output := captureStdout(t, func() {
		if err := cmdListCreate(dir, []string{"done"}); err != nil {
			t.Fatalf("cmdListCreate: %v", err)
		}
	})

	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, output)
	}

	if result["dir"] != "done" {
		t.Errorf("dir: got %v, want %q", result["dir"], "done")
	}

	// Verify directory exists
	listPath := filepath.Join(dir, "done")
	info, err := os.Stat(listPath)
	if os.IsNotExist(err) {
		t.Errorf("list directory not created at %s", listPath)
	} else if !info.IsDir() {
		t.Errorf("expected directory at %s", listPath)
	}

	// Verify config updated
	config, err := daedalus.LoadBoardConfig(dir)
	if err != nil {
		t.Fatalf("LoadBoardConfig: %v", err)
	}
	if daedalus.FindListEntry(config.Lists, "done") < 0 {
		t.Error("list 'done' not found in board config")
	}
}

func TestCmdListCreate_Duplicate(t *testing.T) {
	dir := setupTestBoard(t)
	err := cmdListCreate(dir, []string{"open"})
	if err == nil {
		t.Fatal("expected error for duplicate list")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error should mention 'already exists', got: %v", err)
	}
}

func TestCmdListDelete(t *testing.T) {
	dir := setupTestBoard(t)
	listPath := filepath.Join(dir, "open")

	if err := cmdListDelete(dir, []string{"open"}); err != nil {
		t.Fatalf("cmdListDelete: %v", err)
	}

	// Verify directory removed
	if _, err := os.Stat(listPath); !os.IsNotExist(err) {
		t.Error("list directory should be removed after delete")
	}

	// Verify config updated
	config, err := daedalus.LoadBoardConfig(dir)
	if err != nil {
		t.Fatalf("LoadBoardConfig: %v", err)
	}
	if daedalus.FindListEntry(config.Lists, "open") >= 0 {
		t.Error("list 'open' should not be in board config after delete")
	}
}

func TestCmdListDelete_Empty(t *testing.T) {
	dir := setupTestBoard(t)
	// Create an empty list (directory exists but no cards)
	emptyDir := filepath.Join(dir, "empty")
	if err := os.MkdirAll(emptyDir, 0755); err != nil {
		t.Fatalf("creating empty list dir: %v", err)
	}
	config, err := daedalus.LoadBoardConfig(dir)
	if err != nil {
		t.Fatalf("LoadBoardConfig: %v", err)
	}
	config.Lists = append(config.Lists, daedalus.ListEntry{Dir: "empty"})
	if err := daedalus.SaveBoardConfig(dir, config); err != nil {
		t.Fatalf("SaveBoardConfig: %v", err)
	}

	if err := cmdListDelete(dir, []string{"empty"}); err != nil {
		t.Fatalf("cmdListDelete empty list: %v", err)
	}

	if _, err := os.Stat(emptyDir); !os.IsNotExist(err) {
		t.Error("empty list directory should be removed")
	}
}

func TestCmdExportJSON(t *testing.T) {
	dir := setupTestBoard(t)
	outputPath := filepath.Join(t.TempDir(), "export.json")

	if err := cmdExportJSON(dir, []string{outputPath}); err != nil {
		t.Fatalf("cmdExportJSON: %v", err)
	}

	// Verify the file was written and contains valid JSON
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("reading export file: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("invalid JSON in export: %v", err)
	}

	if result["title"] != "Test Board" {
		t.Errorf("title: got %v, want %q", result["title"], "Test Board")
	}

	lists, ok := result["lists"].([]any)
	if !ok || len(lists) == 0 {
		t.Fatal("expected non-empty lists array in export")
	}
}

func TestCmdExportZip(t *testing.T) {
	dir := setupTestBoard(t)
	outputPath := filepath.Join(t.TempDir(), "export.zip")

	if err := cmdExportZip(dir, []string{outputPath}); err != nil {
		t.Fatalf("cmdExportZip: %v", err)
	}

	// Verify the zip file was written and contains board.yaml
	r, err := zip.OpenReader(outputPath)
	if err != nil {
		t.Fatalf("opening zip: %v", err)
	}
	defer r.Close()

	foundBoardYaml := false
	for _, f := range r.File {
		if f.Name == "board.yaml" {
			foundBoardYaml = true
			break
		}
	}
	if !foundBoardYaml {
		t.Error("zip should contain board.yaml")
	}
}
