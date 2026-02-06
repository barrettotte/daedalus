package daedalus

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTestCard(t *testing.T, dir, filename, content string) string {
	t.Helper()
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

// Verifies that ReadCardContent extracts the full markdown body after frontmatter.
func TestReadCardContent_Basic(t *testing.T) {
	dir := t.TempDir()
	path := writeTestCard(t, dir, "1.md", `---
title: "Test Card"
id: 1
---
# Test Card

Some body content here.
Second line.
`)
	body, err := ReadCardContent(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "# Test Card\n\nSome body content here.\nSecond line.\n"
	if body != expected {
		t.Errorf("got %q, want %q", body, expected)
	}
}

// A card with frontmatter but no body content should return an empty string.
func TestReadCardContent_EmptyBody(t *testing.T) {
	dir := t.TempDir()
	path := writeTestCard(t, dir, "2.md", `---
title: "Empty Body"
id: 2
---
`)
	body, err := ReadCardContent(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body != "" {
		t.Errorf("expected empty body, got %q", body)
	}
}

// A file without frontmatter delimiters should return an empty body,
// since the parser only emits content after the closing ---.
func TestReadCardContent_NoFrontmatter(t *testing.T) {
	dir := t.TempDir()
	path := writeTestCard(t, dir, "3.md", `# Just Markdown

No frontmatter here.
`)
	body, err := ReadCardContent(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body != "" {
		t.Errorf("expected empty body for no-frontmatter file, got %q", body)
	}
}

// Ensures all lines after frontmatter are captured, not just a truncated preview.
func TestReadCardContent_MultilineBody(t *testing.T) {
	dir := t.TempDir()
	content := "---\ntitle: \"Multi\"\nid: 4\n---\nLine 1\nLine 2\nLine 3\n"
	path := writeTestCard(t, dir, "4.md", content)

	body, err := ReadCardContent(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "Line 1\nLine 2\nLine 3\n"
	if body != expected {
		t.Errorf("got %q, want %q", body, expected)
	}
}

// Reading a nonexistent file should return an error.
func TestReadCardContent_FileNotFound(t *testing.T) {
	_, err := ReadCardContent("/nonexistent/path/card.md")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

// Verifies that parseFileHeader correctly deserializes title, id, list_order,
// and labels from YAML frontmatter, and produces a non-empty body preview.
func TestParseFileHeader_BasicMetadata(t *testing.T) {
	dir := t.TempDir()
	path := writeTestCard(t, dir, "10.md", `---
title: "Parse Test"
id: 10
list_order: 5.0
labels:
  - "bug"
  - "urgent"
---
# Parse Test

Body here.
`)
	meta, preview, err := parseFileHeader(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Title != "Parse Test" {
		t.Errorf("title: got %q, want %q", meta.Title, "Parse Test")
	}
	if meta.ID != 10 {
		t.Errorf("id: got %d, want %d", meta.ID, 10)
	}
	if meta.ListOrder != 5.0 {
		t.Errorf("list_order: got %f, want %f", meta.ListOrder, 5.0)
	}
	if len(meta.Labels) != 2 || meta.Labels[0] != "bug" || meta.Labels[1] != "urgent" {
		t.Errorf("labels: got %v, want [bug urgent]", meta.Labels)
	}
	if preview == "" {
		t.Error("expected non-empty preview")
	}
}

// Checklist items should be parsed with their desc and done status preserved.
func TestParseFileHeader_Checklist(t *testing.T) {
	dir := t.TempDir()
	path := writeTestCard(t, dir, "11.md", `---
title: "Checklist Card"
id: 11
checklist:
  - { desc: "Task A", done: true }
  - { desc: "Task B", done: false }
---
# Checklist Card
`)
	meta, _, err := parseFileHeader(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(meta.Checklist) != 2 {
		t.Fatalf("checklist length: got %d, want 2", len(meta.Checklist))
	}
	if meta.Checklist[0].Desc != "Task A" || !meta.Checklist[0].Done {
		t.Errorf("checklist[0]: got %+v", meta.Checklist[0])
	}
	if meta.Checklist[1].Desc != "Task B" || meta.Checklist[1].Done {
		t.Errorf("checklist[1]: got %+v", meta.Checklist[1])
	}
}

// Counter struct should be parsed with current, max, and label fields.
func TestParseFileHeader_Counter(t *testing.T) {
	dir := t.TempDir()
	path := writeTestCard(t, dir, "12.md", `---
title: "Counter Card"
id: 12
counter:
  current: 3
  max: 10
  label: "Progress"
---
Body.
`)
	meta, _, err := parseFileHeader(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Counter == nil {
		t.Fatal("expected counter to be non-nil")
	}
	if meta.Counter.Current != 3 || meta.Counter.Max != 10 || meta.Counter.Label != "Progress" {
		t.Errorf("counter: got %+v", meta.Counter)
	}
}

// The body preview from parseFileHeader should stop accumulating around 150 chars.
func TestParseFileHeader_PreviewTruncation(t *testing.T) {
	dir := t.TempDir()
	longLine := ""
	for i := 0; i < 200; i++ {
		longLine += "x"
	}
	path := writeTestCard(t, dir, "13.md", "---\ntitle: \"Long\"\nid: 13\n---\n"+longLine+"\n")

	_, preview, err := parseFileHeader(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(preview) > 210 {
		t.Errorf("preview too long: %d chars", len(preview))
	}
}

// ScanBoard should discover list directories, assign cards to the right lists,
// parse display names from the directory prefix, and track the global MaxID.
func TestScanBoard_ListDiscovery(t *testing.T) {
	root := t.TempDir()

	list1 := filepath.Join(root, "00___open")
	list2 := filepath.Join(root, "10___in_progress")
	os.Mkdir(list1, 0755)
	os.Mkdir(list2, 0755)

	writeTestCard(t, list1, "1.md", "---\ntitle: \"Card A\"\nid: 1\nlist_order: 1\n---\nBody A\n")
	writeTestCard(t, list1, "2.md", "---\ntitle: \"Card B\"\nid: 2\nlist_order: 2\n---\nBody B\n")
	writeTestCard(t, list2, "3.md", "---\ntitle: \"Card C\"\nid: 3\nlist_order: 1\n---\nBody C\n")

	state, err := ScanBoard(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(state.Lists) != 2 {
		t.Fatalf("expected 2 lists, got %d", len(state.Lists))
	}

	openCards := state.Lists["00___open"]
	if len(openCards) != 2 {
		t.Fatalf("expected 2 cards in open, got %d", len(openCards))
	}

	ipCards := state.Lists["10___in_progress"]
	if len(ipCards) != 1 {
		t.Fatalf("expected 1 card in in_progress, got %d", len(ipCards))
	}

	if openCards[0].ListName != "open" {
		t.Errorf("expected list name 'open', got %q", openCards[0].ListName)
	}

	if state.MaxID != 3 {
		t.Errorf("expected MaxID 3, got %d", state.MaxID)
	}
}

// Cards within a list should be sorted by list_order, with ID as tiebreaker.
func TestScanBoard_CardSortOrder(t *testing.T) {
	root := t.TempDir()
	list := filepath.Join(root, "00___test")
	os.Mkdir(list, 0755)

	writeTestCard(t, list, "1.md", "---\ntitle: \"Third\"\nid: 1\nlist_order: 30\n---\n")
	writeTestCard(t, list, "2.md", "---\ntitle: \"First\"\nid: 2\nlist_order: 10\n---\n")
	writeTestCard(t, list, "3.md", "---\ntitle: \"Second\"\nid: 3\nlist_order: 20\n---\n")

	state, err := ScanBoard(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cards := state.Lists["00___test"]
	if len(cards) != 3 {
		t.Fatalf("expected 3 cards, got %d", len(cards))
	}

	if cards[0].Metadata.Title != "First" {
		t.Errorf("cards[0]: got %q, want \"First\"", cards[0].Metadata.Title)
	}
	if cards[1].Metadata.Title != "Second" {
		t.Errorf("cards[1]: got %q, want \"Second\"", cards[1].Metadata.Title)
	}
	if cards[2].Metadata.Title != "Third" {
		t.Errorf("cards[2]: got %q, want \"Third\"", cards[2].Metadata.Title)
	}
}

// Directories starting with a dot should be skipped during board scanning.
func TestScanBoard_HiddenDirsIgnored(t *testing.T) {
	root := t.TempDir()
	os.Mkdir(filepath.Join(root, ".hidden"), 0755)
	os.Mkdir(filepath.Join(root, "00___visible"), 0755)
	writeTestCard(t, filepath.Join(root, ".hidden"), "1.md", "---\ntitle: \"Hidden\"\nid: 1\n---\n")
	writeTestCard(t, filepath.Join(root, "00___visible"), "2.md", "---\ntitle: \"Visible\"\nid: 2\n---\n")

	state, err := ScanBoard(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(state.Lists) != 1 {
		t.Fatalf("expected 1 list (hidden ignored), got %d", len(state.Lists))
	}
	if _, ok := state.Lists[".hidden"]; ok {
		t.Error("hidden directory should not appear as a list")
	}
}

// When frontmatter has no id field, the card ID should fall back to the filename.
func TestScanBoard_IDFromFilename(t *testing.T) {
	root := t.TempDir()
	list := filepath.Join(root, "00___test")
	os.Mkdir(list, 0755)

	writeTestCard(t, list, "42.md", "---\ntitle: \"No ID\"\nlist_order: 1\n---\n")

	state, err := ScanBoard(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cards := state.Lists["00___test"]
	if len(cards) != 1 {
		t.Fatalf("expected 1 card, got %d", len(cards))
	}
	if cards[0].Metadata.ID != 42 {
		t.Errorf("expected ID 42 from filename, got %d", cards[0].Metadata.ID)
	}
}

// Only .md files should be picked up as cards; other file types are ignored.
func TestScanBoard_NonMdFilesIgnored(t *testing.T) {
	root := t.TempDir()
	list := filepath.Join(root, "00___test")
	os.Mkdir(list, 0755)

	writeTestCard(t, list, "1.md", "---\ntitle: \"Card\"\nid: 1\n---\n")
	writeTestCard(t, list, "notes.txt", "not a card")
	writeTestCard(t, list, "data.json", "{}")

	state, err := ScanBoard(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cards := state.Lists["00___test"]
	if len(cards) != 1 {
		t.Errorf("expected 1 card (non-.md ignored), got %d", len(cards))
	}
}

// An empty root directory should produce zero lists and MaxID of 0.
func TestScanBoard_EmptyBoard(t *testing.T) {
	root := t.TempDir()

	state, err := ScanBoard(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(state.Lists) != 0 {
		t.Errorf("expected 0 lists, got %d", len(state.Lists))
	}
	if state.MaxID != 0 {
		t.Errorf("expected MaxID 0, got %d", state.MaxID)
	}
}

// Each card's FilePath should be the full absolute path to the .md file on disk.
func TestScanBoard_FilePaths(t *testing.T) {
	root := t.TempDir()
	list := filepath.Join(root, "00___test")
	os.Mkdir(list, 0755)
	writeTestCard(t, list, "7.md", "---\ntitle: \"Path Test\"\nid: 7\n---\nBody\n")

	state, err := ScanBoard(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cards := state.Lists["00___test"]
	expectedPath := filepath.Join(list, "7.md")
	if cards[0].FilePath != expectedPath {
		t.Errorf("filePath: got %q, want %q", cards[0].FilePath, expectedPath)
	}
}
