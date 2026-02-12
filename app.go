package main

import (
	"context"
	"daedalus/pkg/daedalus"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// BoardResponse is the structure returned to the frontend from LoadBoard.
type BoardResponse struct {
	Lists  map[string][]daedalus.KanbanCard `json:"lists"`
	Config *daedalus.BoardConfig            `json:"config"`
}

// AppMetrics holds runtime performance metrics for the frontend overlay
type AppMetrics struct {
	HeapAlloc  float64 `json:"heapAlloc"`
	Sys        float64 `json:"sys"`
	NumGC      uint32  `json:"numGC"`
	Goroutines int     `json:"goroutines"`
	NumCards   int     `json:"numCards"`
	NumLists   int     `json:"numLists"`
	MaxID      int     `json:"maxID"`
	FileSizeMB float64 `json:"fileSizeMB"`
}

// App struct
type App struct {
	ctx   context.Context
	board *daedalus.BoardState
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// LoadBoard is what is exposed to the frontend
func (a *App) LoadBoard(path string) *BoardResponse {
	if path == "" {
		path = "./tmp/kanban"
	}
	fmt.Printf("Scanning board at: %s\n", path)

	state, err := daedalus.ScanBoard(path)
	if err != nil {
		fmt.Printf("Error scanning board: %v\n", err)
		return nil
	}
	a.board = state
	fmt.Printf("Scan Complete. MaxID: %d\n", state.MaxID)

	return &BoardResponse{
		Lists:  state.Lists,
		Config: state.Config,
	}
}

// SaveListConfig updates the config for a single list and persists to board.yaml.
func (a *App) SaveListConfig(dirName string, title string, limit int) error {
	if a.board == nil {
		return fmt.Errorf("board not loaded")
	}

	a.board.Config.UpdateListConfig(dirName, daedalus.ListConfig{
		Title: title,
		Limit: limit,
	})

	return daedalus.SaveBoardConfig(a.board.RootPath, a.board.Config)
}

// SaveLabelsExpanded persists the label collapsed/expanded state to board.yaml.
func (a *App) SaveLabelsExpanded(expanded bool) error {
	if a.board == nil {
		return fmt.Errorf("board not loaded")
	}
	a.board.Config.LabelsExpanded = &expanded
	return daedalus.SaveBoardConfig(a.board.RootPath, a.board.Config)
}

// SaveCollapsedLists persists the set of collapsed list keys to board.yaml.
func (a *App) SaveCollapsedLists(lists []string) error {
	if a.board == nil {
		return fmt.Errorf("board not loaded")
	}
	a.board.Config.CollapsedLists = lists
	return daedalus.SaveBoardConfig(a.board.RootPath, a.board.Config)
}

// validatePath resolves a file path to absolute and verifies it is within the board root.
func (a *App) validatePath(filePath string) (string, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return "", fmt.Errorf("invalid path")
	}
	absRoot, err := filepath.Abs(a.board.RootPath)
	if err != nil {
		return "", fmt.Errorf("invalid root path")
	}
	if !strings.HasPrefix(absPath, absRoot+string(filepath.Separator)) {
		return "", fmt.Errorf("path outside board directory")
	}
	return absPath, nil
}

// OpenFileExternal opens a file in the system default application.
func (a *App) OpenFileExternal(filePath string) error {
	if a.board == nil {
		return fmt.Errorf("board not loaded")
	}

	absPath, err := a.validatePath(filePath)
	if err != nil {
		return err
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", absPath)
	case "darwin":
		cmd = exec.Command("open", absPath)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", absPath)
	default:
		return fmt.Errorf("unsupported platform")
	}

	return cmd.Start()
}

// GetCardContent returns the full markdown body of a card file
func (a *App) GetCardContent(filePath string) (string, error) {
	if a.board == nil {
		return "", fmt.Errorf("board not loaded")
	}

	absPath, err := a.validatePath(filePath)
	if err != nil {
		return "", err
	}
	return daedalus.ReadCardContent(absPath)
}

// SaveCard writes updated metadata and body to a card file, updates in-memory state, and returns the updated card
func (a *App) SaveCard(filePath string, metadata daedalus.CardMetadata, body string) (*daedalus.KanbanCard, error) {
	if a.board == nil {
		return nil, fmt.Errorf("board not loaded")
	}

	absPath, err := a.validatePath(filePath)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	metadata.Updated = &now
	if metadata.Created == nil {
		metadata.Created = &now
	}

	// Capture old file size before writing for incremental update
	var oldSize int64
	if info, err := os.Stat(absPath); err == nil {
		oldSize = info.Size()
	}

	if err := daedalus.WriteCardFile(absPath, metadata, body); err != nil {
		return nil, fmt.Errorf("writing card file: %w", err)
	}

	// Update cached total file size
	if info, err := os.Stat(absPath); err == nil {
		a.board.TotalFileBytes += info.Size() - oldSize
	}

	// Generate preview from body (first ~150 chars)
	preview := body
	if len(preview) > 150 {
		preview = preview[:150]
	}

	updatedCard := daedalus.KanbanCard{
		FilePath:    absPath,
		Metadata:    metadata,
		PreviewText: preview,
	}

	// Update card in-place in board lists
	for listName, cards := range a.board.Lists {
		for i, card := range cards {
			if card.FilePath == absPath {
				updatedCard.ListName = card.ListName
				a.board.Lists[listName][i] = updatedCard
				return &updatedCard, nil
			}
		}
	}
	return &updatedCard, nil
}

// GetMetrics returns runtime performance metrics
func (a *App) GetMetrics() AppMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	numCards := 0
	numLists := 0
	maxID := 0
	var totalBytes int64

	if a.board != nil {
		numLists = len(a.board.Lists)
		maxID = a.board.MaxID
		totalBytes = a.board.TotalFileBytes

		for _, cards := range a.board.Lists {
			numCards += len(cards)
		}
	}

	return AppMetrics{
		HeapAlloc:  float64(m.HeapAlloc) / 1024 / 1024,
		Sys:        float64(m.Sys) / 1024 / 1024,
		NumGC:      m.NumGC,
		Goroutines: runtime.NumGoroutine(),
		NumCards:   numCards,
		NumLists:   numLists,
		MaxID:      maxID,
		FileSizeMB: float64(totalBytes) / 1024 / 1024,
	}
}
