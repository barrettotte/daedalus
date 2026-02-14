package daedalus

import (
	"os"
	"path/filepath"
	"testing"
)

// Loading config when board.yaml doesn't exist should return empty config without error.
func TestLoadBoardConfig_MissingFile(t *testing.T) {
	root := t.TempDir()

	config, err := LoadBoardConfig(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(config.Lists) != 0 {
		t.Errorf("expected empty lists slice, got %d entries", len(config.Lists))
	}
}

// Loading a valid board.yaml should parse list entries correctly.
func TestLoadBoardConfig_ValidFile(t *testing.T) {
	root := t.TempDir()
	yaml := `lists:
  - dir: in_progress
    title: "Doing"
    limit: 15
  - dir: to_do
    limit: 20
`
	os.WriteFile(filepath.Join(root, "board.yaml"), []byte(yaml), 0644)

	config, err := LoadBoardConfig(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(config.Lists) != 2 {
		t.Fatalf("expected 2 list entries, got %d", len(config.Lists))
	}

	ip := config.Lists[0]
	if ip.Dir != "in_progress" || ip.Title != "Doing" || ip.Limit != 15 {
		t.Errorf("lists[0]: got dir=%q title=%q limit=%d", ip.Dir, ip.Title, ip.Limit)
	}

	todo := config.Lists[1]
	if todo.Dir != "to_do" || todo.Title != "" || todo.Limit != 20 {
		t.Errorf("lists[1]: got dir=%q title=%q limit=%d", todo.Dir, todo.Title, todo.Limit)
	}
}

// Saving and loading config should produce identical values.
func TestBoardConfig_SaveRoundTrip(t *testing.T) {
	root := t.TempDir()

	original := &BoardConfig{
		Lists: []ListEntry{
			{Dir: "open", Title: "Open Items", Limit: 50},
			{Dir: "wip", Limit: 5},
		},
	}

	if err := SaveBoardConfig(root, original); err != nil {
		t.Fatalf("save error: %v", err)
	}

	loaded, err := LoadBoardConfig(root)
	if err != nil {
		t.Fatalf("load error: %v", err)
	}

	if len(loaded.Lists) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(loaded.Lists))
	}

	open := loaded.Lists[0]
	if open.Dir != "open" || open.Title != "Open Items" || open.Limit != 50 {
		t.Errorf("open: got %+v", open)
	}

	wip := loaded.Lists[1]
	if wip.Dir != "wip" || wip.Title != "" || wip.Limit != 5 {
		t.Errorf("wip: got %+v", wip)
	}
}

// Collapsed and half-collapsed flags should survive a round trip.
func TestBoardConfig_CollapseRoundTrip(t *testing.T) {
	root := t.TempDir()

	original := &BoardConfig{
		Lists: []ListEntry{
			{Dir: "open"},
			{Dir: "archive", Collapsed: true},
			{Dir: "backlog", HalfCollapsed: true},
		},
	}

	if err := SaveBoardConfig(root, original); err != nil {
		t.Fatalf("save error: %v", err)
	}

	loaded, err := LoadBoardConfig(root)
	if err != nil {
		t.Fatalf("load error: %v", err)
	}

	if len(loaded.Lists) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(loaded.Lists))
	}
	if loaded.Lists[0].Collapsed || loaded.Lists[0].HalfCollapsed {
		t.Errorf("open should not be collapsed")
	}
	if !loaded.Lists[1].Collapsed {
		t.Errorf("archive should be collapsed")
	}
	if !loaded.Lists[2].HalfCollapsed {
		t.Errorf("backlog should be half-collapsed")
	}
}

// FindListEntry should return the correct index or -1.
func TestFindListEntry(t *testing.T) {
	lists := []ListEntry{
		{Dir: "open"},
		{Dir: "wip"},
		{Dir: "done"},
	}

	if idx := FindListEntry(lists, "wip"); idx != 1 {
		t.Errorf("expected index 1 for 'wip', got %d", idx)
	}
	if idx := FindListEntry(lists, "nonexistent"); idx != -1 {
		t.Errorf("expected -1 for nonexistent, got %d", idx)
	}
	if idx := FindListEntry(lists, "open"); idx != 0 {
		t.Errorf("expected index 0 for 'open', got %d", idx)
	}
}

// Array order in Lists is the display order -- no separate list_order field needed.
func TestBoardConfig_ArrayOrderIsDisplayOrder(t *testing.T) {
	root := t.TempDir()

	original := &BoardConfig{
		Lists: []ListEntry{
			{Dir: "done"},
			{Dir: "open"},
			{Dir: "wip"},
		},
	}

	if err := SaveBoardConfig(root, original); err != nil {
		t.Fatalf("save error: %v", err)
	}

	loaded, err := LoadBoardConfig(root)
	if err != nil {
		t.Fatalf("load error: %v", err)
	}

	if loaded.Lists[0].Dir != "done" || loaded.Lists[1].Dir != "open" || loaded.Lists[2].Dir != "wip" {
		t.Errorf("array order not preserved: %v", loaded.Lists)
	}
}
