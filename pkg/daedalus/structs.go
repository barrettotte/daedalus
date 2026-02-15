package daedalus

import "time"

// PreviewMaxLen is the maximum character length for card body previews.
const PreviewMaxLen = 150

// BoardState holds runtime data in memory
type BoardState struct {
	Lists          map[string][]KanbanCard
	MaxID          int
	RootPath       string
	Config         *BoardConfig
	TotalFileBytes int64
	ConfigLoadTime time.Duration
	ScanTime       time.Duration
}

// KanbanCard is the object sent to the frontend
type KanbanCard struct {
	FilePath    string       `json:"filePath"`
	ListName    string       `json:"listName"`
	Metadata    CardMetadata `json:"metadata"`
	PreviewText string       `json:"previewText"`
}

// CardMetadata is the YAML frontmatter structure in the .md files
type CardMetadata struct {
	ID        int             `yaml:"id" json:"id"`
	Title     string          `yaml:"title" json:"title"`
	Created   *time.Time      `yaml:"created,omitempty" json:"created,omitempty"`
	Updated   *time.Time      `yaml:"updated,omitempty" json:"updated,omitempty"`
	ListOrder float64         `yaml:"list_order" json:"list_order"`
	Due       *time.Time      `yaml:"due,omitempty" json:"due,omitempty"`
	Range     *DateRange      `yaml:"range,omitempty" json:"range,omitempty"`
	Labels    []string        `yaml:"labels,omitempty" json:"labels"`
	Icon      string          `yaml:"icon,omitempty" json:"icon"`
	Counter   *Counter        `yaml:"counter,omitempty" json:"counter,omitempty"`
	Checklist []CheckListItem `yaml:"checklist,omitempty" json:"checklist,omitempty"`
}

// DateRange is the date range a card will be active
type DateRange struct {
	Start time.Time `yaml:"start" json:"start"`
	End   time.Time `yaml:"end" json:"end"`
}

// Counter is an incrementable counter with a label.
// If start < max the counter counts up; if start > max it counts down.
type Counter struct {
	Current int    `yaml:"current" json:"current"`
	Max     int    `yaml:"max" json:"max"`
	Start   int    `yaml:"start,omitempty" json:"start"`
	Step    int    `yaml:"step,omitempty" json:"step"`
	Label   string `yaml:"label,omitempty" json:"label"`
}

// CheckListItem is an item in a checklist for a card
type CheckListItem struct {
	Idx  int    `yaml:"idx" json:"idx"`
	Desc string `yaml:"desc" json:"desc"`
	Done bool   `yaml:"done" json:"done"`
}
