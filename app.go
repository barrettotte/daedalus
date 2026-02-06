package main

import (
	"context"
	"daedalus/pkg/daedalus"
	"fmt"
	"path/filepath"
	"strings"
)

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
