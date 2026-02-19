package daedalus

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadAppConfig_MissingFile(t *testing.T) {
	dir := t.TempDir()
	cfg, err := LoadAppConfig(dir)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.DefaultBoard != "" {
		t.Errorf("expected empty DefaultBoard, got %q", cfg.DefaultBoard)
	}
	if len(cfg.RecentBoards) != 0 {
		t.Errorf("expected 0 recent boards, got %d", len(cfg.RecentBoards))
	}
}

func TestSaveAndLoadAppConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := &AppConfig{
		DefaultBoard: "/home/test/boards/main",
		RecentBoards: []RecentBoard{
			{Path: "/home/test/boards/main", LastOpened: time.Now().UTC().Truncate(time.Second)},
		},
	}
	if err := SaveAppConfig(dir, cfg); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadAppConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.DefaultBoard != cfg.DefaultBoard {
		t.Errorf("DefaultBoard: got %q, want %q", loaded.DefaultBoard, cfg.DefaultBoard)
	}
	if len(loaded.RecentBoards) != 1 {
		t.Fatalf("expected 1 recent board, got %d", len(loaded.RecentBoards))
	}
	if loaded.RecentBoards[0].Path != cfg.RecentBoards[0].Path {
		t.Errorf("Path: got %q, want %q", loaded.RecentBoards[0].Path, cfg.RecentBoards[0].Path)
	}
	if !loaded.RecentBoards[0].LastOpened.Equal(cfg.RecentBoards[0].LastOpened) {
		t.Errorf("LastOpened: got %v, want %v", loaded.RecentBoards[0].LastOpened, cfg.RecentBoards[0].LastOpened)
	}
}

func TestAddRecentBoard(t *testing.T) {
	cfg := &AppConfig{}
	now := time.Now().UTC()

	AddRecentBoard(cfg, "/boards/a", "Board A", now)
	if len(cfg.RecentBoards) != 1 {
		t.Fatalf("expected 1 board, got %d", len(cfg.RecentBoards))
	}
	if cfg.RecentBoards[0].Title != "Board A" {
		t.Errorf("expected title %q, got %q", "Board A", cfg.RecentBoards[0].Title)
	}

	// Adding the same path updates timestamp and title, doesn't duplicate.
	later := now.Add(time.Hour)
	AddRecentBoard(cfg, "/boards/a", "Board A Renamed", later)
	if len(cfg.RecentBoards) != 1 {
		t.Fatalf("expected 1 board after re-add, got %d", len(cfg.RecentBoards))
	}
	if !cfg.RecentBoards[0].LastOpened.Equal(later) {
		t.Error("timestamp should be updated")
	}
	if cfg.RecentBoards[0].Title != "Board A Renamed" {
		t.Errorf("title should be updated, got %q", cfg.RecentBoards[0].Title)
	}
}

func TestAddRecentBoard_MaxTen(t *testing.T) {
	cfg := &AppConfig{}
	now := time.Now().UTC()
	for i := range 12 {
		AddRecentBoard(cfg, "/boards/"+string(rune('a'+i)), "", now.Add(time.Duration(i)*time.Minute))
	}

	if len(cfg.RecentBoards) != 10 {
		t.Errorf("expected max 10 recent boards, got %d", len(cfg.RecentBoards))
	}
	// Most recent should be first.
	if cfg.RecentBoards[0].Path != "/boards/l" {
		t.Errorf("first board should be most recent, got %q", cfg.RecentBoards[0].Path)
	}
}

func TestRemoveRecentBoard(t *testing.T) {
	cfg := &AppConfig{
		RecentBoards: []RecentBoard{
			{Path: "/boards/a"},
			{Path: "/boards/b"},
		},
	}
	RemoveRecentBoard(cfg, "/boards/a")
	if len(cfg.RecentBoards) != 1 {
		t.Fatalf("expected 1 board, got %d", len(cfg.RecentBoards))
	}
	if cfg.RecentBoards[0].Path != "/boards/b" {
		t.Error("wrong board remaining")
	}
}

func TestPruneInvalidBoards(t *testing.T) {
	existing := t.TempDir()
	cfg := &AppConfig{
		DefaultBoard: "/nonexistent/board",
		RecentBoards: []RecentBoard{
			{Path: existing},
			{Path: "/nonexistent/board"},
		},
	}
	PruneInvalidBoards(cfg)

	if cfg.DefaultBoard != "" {
		t.Errorf("expected default board cleared, got %q", cfg.DefaultBoard)
	}
	if len(cfg.RecentBoards) != 1 {
		t.Fatalf("expected 1 board after prune, got %d", len(cfg.RecentBoards))
	}
	if cfg.RecentBoards[0].Path != existing {
		t.Error("wrong board survived prune")
	}
}

func TestSaveAppConfig_CreatesDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "config")
	cfg := &AppConfig{DefaultBoard: "/test"}
	if err := SaveAppConfig(dir, cfg); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "config.yaml")); err != nil {
		t.Error("config.yaml should exist after save")
	}
}
