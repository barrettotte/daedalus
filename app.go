package main

import (
	"context"
	"daedalus/pkg/daedalus"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// AppMetrics holds runtime performance metrics for the frontend overlay
type AppMetrics struct {
	HeapAlloc  float64 `json:"heapAlloc"`
	Sys        float64 `json:"sys"`
	NumGC      uint32  `json:"numGC"`
	Goroutines int     `json:"goroutines"`
	NumCards   int     `json:"numCards"`
	NumLists   int     `json:"numLists"`
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
func (a *App) LoadBoard(path string) map[string][]daedalus.KanbanCard {
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

	return state.Lists
}

// GetCardContent returns the full markdown body of a card file
func (a *App) GetCardContent(filePath string) (string, error) {
	if a.board == nil {
		return "", fmt.Errorf("board not loaded")
	}

	// Resolve to absolute and verify it's within the board root
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

	return daedalus.ReadCardContent(absPath)
}

// GetMetrics returns runtime performance metrics
func (a *App) GetMetrics() AppMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	numCards := 0
	numLists := 0
	var totalBytes int64
	if a.board != nil {
		numLists = len(a.board.Lists)
		for _, cards := range a.board.Lists {
			numCards += len(cards)
			for _, card := range cards {
				if info, err := os.Stat(card.FilePath); err == nil {
					totalBytes += info.Size()
				}
			}
		}
	}

	return AppMetrics{
		HeapAlloc:  float64(m.HeapAlloc) / 1024 / 1024,
		Sys:        float64(m.Sys) / 1024 / 1024,
		NumGC:      m.NumGC,
		Goroutines: runtime.NumGoroutine(),
		NumCards:    numCards,
		NumLists:   numLists,
		FileSizeMB: float64(totalBytes) / 1024 / 1024,
	}
}
