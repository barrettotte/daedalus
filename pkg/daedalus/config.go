package daedalus

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ListConfig holds per-list settings like display title override and card limit.
type ListConfig struct {
	Title string `yaml:"title" json:"title"`
	Limit int    `yaml:"limit" json:"limit"`
}

// BoardConfig holds board-level configuration loaded from board.yaml.
type BoardConfig struct {
	Lists          map[string]ListConfig `yaml:"lists" json:"lists"`
	LabelsExpanded *bool                 `yaml:"labels_expanded,omitempty" json:"labelsExpanded,omitempty"`
	CollapsedLists []string              `yaml:"collapsed_lists,omitempty" json:"collapsedLists,omitempty"`
}

// LoadBoardConfig reads board.yaml from rootPath. Returns empty config if file is missing.
func LoadBoardConfig(rootPath string) (*BoardConfig, error) {
	config := &BoardConfig{
		Lists: make(map[string]ListConfig),
	}

	data, err := os.ReadFile(filepath.Join(rootPath, "board.yaml"))
	if err != nil {
		if os.IsNotExist(err) {
			return config, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, err
	}

	if config.Lists == nil {
		config.Lists = make(map[string]ListConfig)
	}

	return config, nil
}

// SaveBoardConfig writes the config to board.yaml in rootPath.
func SaveBoardConfig(rootPath string, config *BoardConfig) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(rootPath, "board.yaml"), data, 0644)
}

// UpdateListConfig sets a list's config entry. If both title and limit are zero-value,
// the entry is removed to keep board.yaml clean.
func (c *BoardConfig) UpdateListConfig(dirName string, lc ListConfig) {
	if lc.Title == "" && lc.Limit == 0 {
		delete(c.Lists, dirName)
	} else {
		c.Lists[dirName] = lc
	}
}
