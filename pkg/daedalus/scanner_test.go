package daedalus

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
  label: "Tasks"
  items:
    - { desc: "Task A", done: true }
    - { desc: "Task B", done: false }
---
# Checklist Card
`)
	meta, _, err := parseFileHeader(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Checklist == nil {
		t.Fatal("expected checklist to be non-nil")
	}
	if len(meta.Checklist.Items) != 2 {
		t.Fatalf("checklist items length: got %d, want 2", len(meta.Checklist.Items))
	}
	if meta.Checklist.Label != "Tasks" {
		t.Errorf("checklist label: got %q, want %q", meta.Checklist.Label, "Tasks")
	}
	if meta.Checklist.Items[0].Desc != "Task A" || !meta.Checklist.Items[0].Done {
		t.Errorf("checklist items[0]: got %+v", meta.Checklist.Items[0])
	}
	if meta.Checklist.Items[1].Desc != "Task B" || meta.Checklist.Items[1].Done {
		t.Errorf("checklist items[1]: got %+v", meta.Checklist.Items[1])
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
// and track the global MaxID.
func TestScanBoard_ListDiscovery(t *testing.T) {
	root := t.TempDir()

	list1 := filepath.Join(root, "open")
	list2 := filepath.Join(root, "in_progress")
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

	openCards := state.Lists["open"]
	if len(openCards) != 2 {
		t.Fatalf("expected 2 cards in open, got %d", len(openCards))
	}

	ipCards := state.Lists["in_progress"]
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
	list := filepath.Join(root, "test")
	os.Mkdir(list, 0755)

	writeTestCard(t, list, "1.md", "---\ntitle: \"Third\"\nid: 1\nlist_order: 30\n---\n")
	writeTestCard(t, list, "2.md", "---\ntitle: \"First\"\nid: 2\nlist_order: 10\n---\n")
	writeTestCard(t, list, "3.md", "---\ntitle: \"Second\"\nid: 3\nlist_order: 20\n---\n")

	state, err := ScanBoard(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cards := state.Lists["test"]
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
	os.Mkdir(filepath.Join(root, "visible"), 0755)

	writeTestCard(t, filepath.Join(root, ".hidden"), "1.md", "---\ntitle: \"Hidden\"\nid: 1\n---\n")
	writeTestCard(t, filepath.Join(root, "visible"), "2.md", "---\ntitle: \"Visible\"\nid: 2\n---\n")

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
	list := filepath.Join(root, "test")
	os.Mkdir(list, 0755)

	writeTestCard(t, list, "42.md", "---\ntitle: \"No ID\"\nlist_order: 1\n---\n")

	state, err := ScanBoard(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cards := state.Lists["test"]
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
	list := filepath.Join(root, "test")
	os.Mkdir(list, 0755)

	writeTestCard(t, list, "1.md", "---\ntitle: \"Card\"\nid: 1\n---\n")
	writeTestCard(t, list, "notes.txt", "not a card")
	writeTestCard(t, list, "data.json", "{}")

	state, err := ScanBoard(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cards := state.Lists["test"]
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
	list := filepath.Join(root, "test")
	os.Mkdir(list, 0755)
	writeTestCard(t, list, "7.md", "---\ntitle: \"Path Test\"\nid: 7\n---\nBody\n")

	state, err := ScanBoard(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cards := state.Lists["test"]
	expectedPath := filepath.Join(list, "7.md")
	if cards[0].FilePath != expectedPath {
		t.Errorf("filePath: got %q, want %q", cards[0].FilePath, expectedPath)
	}
}

// Writing a card and reading it back should produce matching metadata and body.
func TestWriteCardFile_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "1.md")
	now := time.Now().Truncate(time.Second)
	meta := CardMetadata{
		ID:        1,
		Title:     "Round Trip",
		Created:   &now,
		ListOrder: 5.5,
		Labels:    []string{"test", "roundtrip"},
		Checklist: &Checklist{
			Label: "Steps",
			Items: []CheckListItem{
				{Idx: 0, Desc: "Step 1", Done: true},
				{Idx: 1, Desc: "Step 2", Done: false},
			},
		},
	}
	body := "# Round Trip\n\nSome description.\n"

	if err := WriteCardFile(path, meta, body); err != nil {
		t.Fatalf("WriteCardFile error: %v", err)
	}

	// Read back metadata
	readMeta, preview, err := parseFileHeader(path)
	if err != nil {
		t.Fatalf("parseFileHeader error: %v", err)
	}
	if readMeta.Title != "Round Trip" {
		t.Errorf("title: got %q, want %q", readMeta.Title, "Round Trip")
	}
	if readMeta.ID != 1 {
		t.Errorf("id: got %d, want 1", readMeta.ID)
	}
	if readMeta.ListOrder != 5.5 {
		t.Errorf("list_order: got %f, want 5.5", readMeta.ListOrder)
	}
	if len(readMeta.Labels) != 2 {
		t.Errorf("labels: got %v, want 2 items", readMeta.Labels)
	}
	if readMeta.Checklist == nil || len(readMeta.Checklist.Items) != 2 {
		t.Errorf("checklist: got %+v, want 2 items", readMeta.Checklist)
	}
	if preview == "" {
		t.Error("expected non-empty preview")
	}

	// Read back body
	readBody, err := ReadCardContent(path)
	if err != nil {
		t.Fatalf("ReadCardContent error: %v", err)
	}
	if readBody != body {
		t.Errorf("body: got %q, want %q", readBody, body)
	}
}

// Unknown YAML fields like trello_data should survive a WriteCardFile round-trip.
func TestWriteCardFile_PreservesUnknownFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "2.md")

	// Write initial file with trello_data field
	initial := "---\ntitle: \"Trello Card\"\nid: 2\nlist_order: 1\ntrello_data:\n  board_id: \"abc123\"\n  card_id: \"def456\"\n---\n# Trello Card\n\nImported from Trello.\n"
	if err := os.WriteFile(path, []byte(initial), 0644); err != nil {
		t.Fatal(err)
	}

	// Update via WriteCardFile
	meta := CardMetadata{
		ID:        2,
		Title:     "Updated Trello Card",
		ListOrder: 1,
	}
	if err := WriteCardFile(path, meta, "# Updated Trello Card\n\nNew body.\n"); err != nil {
		t.Fatalf("WriteCardFile error: %v", err)
	}

	// Read raw file and check trello_data is still there
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "trello_data") {
		t.Error("trello_data field was not preserved")
	}
	if !strings.Contains(string(content), "abc123") {
		t.Error("trello_data board_id value was not preserved")
	}

	// Verify updated title was written
	readMeta, _, err := parseFileHeader(path)
	if err != nil {
		t.Fatal(err)
	}
	if readMeta.Title != "Updated Trello Card" {
		t.Errorf("title: got %q, want %q", readMeta.Title, "Updated Trello Card")
	}
}

// TimeSeries data should survive a WriteCardFile round-trip: write, read back,
// and verify label and entries are preserved.
func TestWriteCardFile_TimeSeries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ts.md")
	now := time.Now().Truncate(time.Second)
	meta := CardMetadata{
		ID:        100,
		Title:     "TS Card",
		Created:   &now,
		ListOrder: 1,
		TimeSeries: &TimeSeries{
			Label: "Weight",
			Entries: []TimeSeriesEntry{
				{Time: "2026-01-01", Value: 215},
				{Time: "2026-01-02", Value: 214.5},
			},
		},
	}
	body := "# TS Card\n\nTracking weight.\n"

	if err := WriteCardFile(path, meta, body); err != nil {
		t.Fatalf("WriteCardFile error: %v", err)
	}

	// Verify YAML file contains timeseries data
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	raw := string(content)
	if !strings.Contains(raw, "timeseries") {
		t.Error("timeseries key not found in file")
	}
	if !strings.Contains(raw, "Weight") {
		t.Error("timeseries label not found in file")
	}

	// Read back metadata and verify
	readMeta, _, err := parseFileHeader(path)
	if err != nil {
		t.Fatalf("parseFileHeader error: %v", err)
	}
	if readMeta.TimeSeries == nil {
		t.Fatal("TimeSeries is nil after read-back")
	}
	if readMeta.TimeSeries.Label != "Weight" {
		t.Errorf("label: got %q, want %q", readMeta.TimeSeries.Label, "Weight")
	}
	if len(readMeta.TimeSeries.Entries) != 2 {
		t.Fatalf("entries: got %d, want 2", len(readMeta.TimeSeries.Entries))
	}
	if readMeta.TimeSeries.Entries[0].Time != "2026-01-01" {
		t.Errorf("entries[0].t: got %q", readMeta.TimeSeries.Entries[0].Time)
	}
	if readMeta.TimeSeries.Entries[0].Value != 215 {
		t.Errorf("entries[0].v: got %f", readMeta.TimeSeries.Entries[0].Value)
	}
}

// TimeSeries with empty label and no entries should still persist and read back as non-nil.
func TestWriteCardFile_TimeSeriesEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ts-empty.md")
	now := time.Now().Truncate(time.Second)
	meta := CardMetadata{
		ID:         101,
		Title:      "Empty TS",
		Created:    &now,
		ListOrder:  1,
		TimeSeries: &TimeSeries{Label: "", Entries: []TimeSeriesEntry{}},
	}

	if err := WriteCardFile(path, meta, "# Empty TS\n"); err != nil {
		t.Fatalf("WriteCardFile error: %v", err)
	}

	content, _ := os.ReadFile(path)
	if !strings.Contains(string(content), "timeseries") {
		t.Error("timeseries key not found in file for empty time series")
	}

	readMeta, _, err := parseFileHeader(path)
	if err != nil {
		t.Fatalf("parseFileHeader error: %v", err)
	}
	if readMeta.TimeSeries == nil {
		t.Fatal("TimeSeries should be non-nil after read-back of empty time series")
	}
	if readMeta.TimeSeries.Label != "" {
		t.Errorf("label: got %q, want empty", readMeta.TimeSeries.Label)
	}
}

// JSON round-trip for CardMetadata with TimeSeries (simulates Wails RPC serialization).
func TestCardMetadata_JSONRoundTrip(t *testing.T) {
	ts := &TimeSeries{
		Label: "Steps",
		Entries: []TimeSeriesEntry{
			{Time: "2026-02-01", Value: 8000},
		},
	}
	meta := CardMetadata{
		ID:         50,
		Title:      "JSON Test",
		ListOrder:  1,
		TimeSeries: ts,
	}

	data, err := json.Marshal(meta)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	// Verify timeseries is in the JSON output
	raw := string(data)
	if !strings.Contains(raw, `"timeseries"`) {
		t.Error("timeseries key not found in JSON")
	}
	if !strings.Contains(raw, `"Steps"`) {
		t.Error("timeseries label not found in JSON")
	}

	var decoded CardMetadata
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded.TimeSeries == nil {
		t.Fatal("TimeSeries is nil after JSON round-trip")
	}
	if decoded.TimeSeries.Label != "Steps" {
		t.Errorf("label: got %q, want %q", decoded.TimeSeries.Label, "Steps")
	}
	if len(decoded.TimeSeries.Entries) != 1 {
		t.Fatalf("entries: got %d, want 1", len(decoded.TimeSeries.Entries))
	}
	if decoded.TimeSeries.Entries[0].Value != 8000 {
		t.Errorf("entries[0].v: got %f, want 8000", decoded.TimeSeries.Entries[0].Value)
	}
}

// Clearing an omitempty field (like due) should remove it from the output file.
func TestWriteCardFile_ClearsOmitemptyFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "3.md")
	due := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	// First write with due date set
	meta := CardMetadata{
		ID:        3,
		Title:     "Due Card",
		ListOrder: 1,
		Due:       &due,
	}
	if err := WriteCardFile(path, meta, "# Due Card\n"); err != nil {
		t.Fatalf("first WriteCardFile error: %v", err)
	}

	// Verify due date is in the file
	content, _ := os.ReadFile(path)
	if !strings.Contains(string(content), "due") {
		t.Fatal("due field should be present after first write")
	}

	// Second write with nil due date
	meta.Due = nil
	if err := WriteCardFile(path, meta, "# Due Card\n"); err != nil {
		t.Fatalf("second WriteCardFile error: %v", err)
	}

	// Verify due date is gone
	content, _ = os.ReadFile(path)
	if strings.Contains(string(content), "due") {
		t.Error("due field should be removed when set to nil")
	}
}
