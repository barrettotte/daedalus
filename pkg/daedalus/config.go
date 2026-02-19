package daedalus

import (
	"log/slog"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// ListEntry holds per-list settings. Array order in BoardConfig.Lists = display order.
type ListEntry struct {
	Dir           string `yaml:"dir" json:"dir"`
	Title         string `yaml:"title,omitempty" json:"title,omitempty"`
	Limit         int    `yaml:"limit,omitempty" json:"limit,omitempty"`
	Collapsed     bool   `yaml:"collapsed,omitempty" json:"collapsed,omitempty"`
	HalfCollapsed bool   `yaml:"half_collapsed,omitempty" json:"halfCollapsed,omitempty"`
	Locked        bool   `yaml:"locked,omitempty" json:"locked,omitempty"`
	Pinned        string `yaml:"pinned,omitempty" json:"pinned,omitempty"`
	Color         string `yaml:"color,omitempty" json:"color,omitempty"`
	Icon          string `yaml:"icon,omitempty" json:"icon,omitempty"`
}

// BoardConfig holds board-level configuration loaded from board.yaml.
type BoardConfig struct {
	Title            string            `yaml:"title,omitempty" json:"title,omitempty"`
	Lists            []ListEntry       `yaml:"lists,omitempty" json:"lists,omitempty"`
	LabelColors      map[string]string `yaml:"label_colors,omitempty" json:"labelColors,omitempty"`
	Templates        []CardTemplate    `yaml:"templates,omitempty" json:"templates,omitempty"`
	LabelsExpanded   *bool             `yaml:"labels_expanded,omitempty" json:"labelsExpanded,omitempty"`
	ShowYearProgress *bool             `yaml:"show_year_progress,omitempty" json:"showYearProgress,omitempty"`
	DarkMode         *bool             `yaml:"dark_mode,omitempty" json:"darkMode,omitempty"`
	MinimalView      *bool             `yaml:"minimal_view,omitempty" json:"minimalView,omitempty"`
	Zoom             *float64          `yaml:"zoom,omitempty" json:"zoom,omitempty"`
}

// LoadBoardConfig reads board.yaml from rootPath. Returns empty config if file is missing.
func LoadBoardConfig(rootPath string) (*BoardConfig, error) {
	config := &BoardConfig{}

	data, err := os.ReadFile(filepath.Join(rootPath, "board.yaml"))
	if err != nil {
		if os.IsNotExist(err) {
			slog.Debug("board.yaml not found, using empty config", "path", rootPath)
			return config, nil
		}
		slog.Error("failed to read board.yaml", "path", rootPath, "error", err)
		return nil, err
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		slog.Error("failed to parse board.yaml", "path", rootPath, "error", err)
		return nil, err
	}

	slog.Debug("board config loaded", "path", rootPath, "lists", len(config.Lists))
	return config, nil
}

// SaveBoardConfig writes the config to board.yaml in rootPath.
func SaveBoardConfig(rootPath string, config *BoardConfig) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		slog.Error("failed to marshal board config", "error", err)
		return err
	}
	if err := os.WriteFile(filepath.Join(rootPath, "board.yaml"), data, 0644); err != nil {
		slog.Error("failed to write board.yaml", "path", rootPath, "error", err)
		return err
	}
	slog.Debug("board config saved", "path", rootPath)
	return nil
}

// FindListEntry returns the index of the entry with the given dir, or -1.
func FindListEntry(lists []ListEntry, dir string) int {
	for i, entry := range lists {
		if entry.Dir == dir {
			return i
		}
	}
	return -1
}

// InitBoardDir ensures a board.yaml exists in the given directory, creating
// both the directory and an empty config file if needed.
func InitBoardDir(path string) error {
	boardYaml := filepath.Join(path, "board.yaml")
	if _, err := os.Stat(boardYaml); err == nil {
		return nil // already exists
	}

	if err := os.MkdirAll(path, 0755); err != nil {
		slog.Error("failed to create board directory", "path", path, "error", err)
		return err
	}
	if err := SaveBoardConfig(path, &BoardConfig{}); err != nil {
		slog.Error("failed to initialize board.yaml", "path", path, "error", err)
		return err
	}
	slog.Info("initialized new board", "path", path)
	return nil
}

// MergeListEntries reconciles the config's list entries with the directories
// actually found on disk. Existing entries are kept in order, newly discovered
// dirs are appended alphabetically, and stale entries (dirs that no longer
// exist on disk) are removed.
func MergeListEntries(config *BoardConfig, diskDirs map[string]bool) {
	// Keep existing entries that still exist on disk.
	var merged []ListEntry
	for _, entry := range config.Lists {
		if diskDirs[entry.Dir] {
			merged = append(merged, entry)
			delete(diskDirs, entry.Dir)
		}
	}

	// Append newly discovered dirs alphabetically.
	var newDirs []string
	for dir := range diskDirs {
		newDirs = append(newDirs, dir)
	}
	sort.Strings(newDirs)
	for _, dir := range newDirs {
		merged = append(merged, ListEntry{Dir: dir})
	}
	if len(newDirs) > 0 {
		slog.Debug("discovered new list directories", "dirs", newDirs)
	}

	config.Lists = merged
}
