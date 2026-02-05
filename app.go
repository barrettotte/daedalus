package main

import (
	"context"
	"daedalus/pkg/daedalus"
	"fmt"
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
