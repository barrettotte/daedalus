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
		t.Errorf("expected empty lists map, got %d entries", len(config.Lists))
	}
}

// Loading a valid board.yaml should parse list configs correctly.
func TestLoadBoardConfig_ValidFile(t *testing.T) {
	root := t.TempDir()
	yaml := `lists:
  10___in_progress:
    title: "Doing"
    limit: 15
  09___to_do:
    title: ""
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

	ip := config.Lists["10___in_progress"]
	if ip.Title != "Doing" || ip.Limit != 15 {
		t.Errorf("in_progress: got title=%q limit=%d, want title=\"Doing\" limit=15", ip.Title, ip.Limit)
	}

	todo := config.Lists["09___to_do"]
	if todo.Title != "" || todo.Limit != 20 {
		t.Errorf("to_do: got title=%q limit=%d, want title=\"\" limit=20", todo.Title, todo.Limit)
	}
}

// Saving and loading config should produce identical values.
func TestBoardConfig_SaveRoundTrip(t *testing.T) {
	root := t.TempDir()

	original := &BoardConfig{
		Lists: map[string]ListConfig{
			"00___open": {Title: "Open Items", Limit: 50},
			"10___wip":  {Title: "", Limit: 5},
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

	open := loaded.Lists["00___open"]
	if open.Title != "Open Items" || open.Limit != 50 {
		t.Errorf("open: got %+v", open)
	}

	wip := loaded.Lists["10___wip"]
	if wip.Title != "" || wip.Limit != 5 {
		t.Errorf("wip: got %+v", wip)
	}
}

// UpdateListConfig should set the entry when title or limit is non-zero.
func TestUpdateListConfig_SetsValues(t *testing.T) {
	config := &BoardConfig{Lists: make(map[string]ListConfig)}

	config.UpdateListConfig("00___open", ListConfig{Title: "My Open", Limit: 10})

	lc, ok := config.Lists["00___open"]
	if !ok {
		t.Fatal("expected entry for 00___open")
	}
	if lc.Title != "My Open" || lc.Limit != 10 {
		t.Errorf("got %+v", lc)
	}
}

// UpdateListConfig should remove the entry when both title and limit are zero-value.
func TestUpdateListConfig_CleansEmptyEntry(t *testing.T) {
	config := &BoardConfig{
		Lists: map[string]ListConfig{
			"00___open": {Title: "Custom", Limit: 5},
		},
	}

	config.UpdateListConfig("00___open", ListConfig{Title: "", Limit: 0})

	if _, ok := config.Lists["00___open"]; ok {
		t.Error("expected entry to be removed when both fields are zero-value")
	}
}
