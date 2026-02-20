package daedalus

import (
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"time"

	"gopkg.in/yaml.v3"
)

const maxRecentBoards = 10

// RecentBoard is a board that was recently opened by the user.
type RecentBoard struct {
	Path       string    `yaml:"path" json:"path"`
	Title      string    `yaml:"title,omitempty" json:"title"`
	LastOpened time.Time `yaml:"last_opened" json:"lastOpened"`
}

// AppConfig holds app-level configuration stored in ~/.config/daedalus/config.yaml.
type AppConfig struct {
	DefaultBoard string        `yaml:"default_board,omitempty" json:"defaultBoard"`
	RecentBoards []RecentBoard `yaml:"recent_boards,omitempty" json:"recentBoards"`
}

// LoadAppConfig reads config.yaml from configDir. Returns empty config if file is missing.
func LoadAppConfig(configDir string) (*AppConfig, error) {
	cfg := &AppConfig{}
	data, err := os.ReadFile(filepath.Join(configDir, "config.yaml"))
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		slog.Error("failed to read app config", "error", err)
		return nil, err
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		slog.Error("failed to parse app config", "error", err)
		return nil, err
	}
	return cfg, nil
}

// SaveAppConfig writes config.yaml to configDir, creating the directory if needed.
func SaveAppConfig(configDir string, cfg *AppConfig) error {
	if err := os.MkdirAll(configDir, 0755); err != nil {
		slog.Error("failed to create config dir", "error", err)
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		slog.Error("failed to marshal app config", "error", err)
		return err
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), data, 0644); err != nil {
		slog.Error("failed to write app config", "error", err)
		return err
	}
	return nil
}

// AddRecentBoard adds or updates a board in the recent list, sorted by recency, capped at 10.
func AddRecentBoard(cfg *AppConfig, boardPath string, title string, opened time.Time) {
	// Remove existing entry for this path if present.
	filtered := make([]RecentBoard, 0, len(cfg.RecentBoards))
	for _, rb := range cfg.RecentBoards {
		if rb.Path != boardPath {
			filtered = append(filtered, rb)
		}
	}

	// Add the new/updated entry and sort by most recent first.
	filtered = append(filtered, RecentBoard{Path: boardPath, Title: title, LastOpened: opened})
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].LastOpened.After(filtered[j].LastOpened)
	})

	// Cap at max.
	if len(filtered) > maxRecentBoards {
		filtered = filtered[:maxRecentBoards]
	}
	cfg.RecentBoards = filtered
}

// RemoveRecentBoard removes a board from the recent list by path.
func RemoveRecentBoard(cfg *AppConfig, boardPath string) {
	filtered := make([]RecentBoard, 0, len(cfg.RecentBoards))
	for _, rb := range cfg.RecentBoards {
		if rb.Path != boardPath {
			filtered = append(filtered, rb)
		}
	}
	cfg.RecentBoards = filtered
}

// PruneInvalidBoards removes boards that no longer exist on disk from the recent list,
// and clears the default board if its path is invalid. Returns true if any entries were removed.
func PruneInvalidBoards(cfg *AppConfig) bool {
	changed := false
	if cfg.DefaultBoard != "" {
		if _, err := os.Stat(cfg.DefaultBoard); err != nil {
			cfg.DefaultBoard = ""
			changed = true
		}
	}
	filtered := make([]RecentBoard, 0, len(cfg.RecentBoards))
	for _, rb := range cfg.RecentBoards {
		if _, err := os.Stat(rb.Path); err == nil {
			filtered = append(filtered, rb)
		}
	}
	if len(filtered) != len(cfg.RecentBoards) {
		changed = true
	}
	cfg.RecentBoards = filtered
	return changed
}
