package main

import (
	"context"
	"daedalus/pkg/daedalus"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	defaultBoardPath      = "./tmp/kanban"
	linuxClockTicksPerSec = 100
)

// BoardResponse is the structure returned to the frontend from LoadBoard.
type BoardResponse struct {
	Lists     map[string][]daedalus.KanbanCard `json:"lists"`
	Config    *daedalus.BoardConfig            `json:"config"`
	BoardPath string                           `json:"boardPath"`
	Profile   LoadProfile                      `json:"profile"`
}

// LoadProfile holds timing data for each phase of board loading.
type LoadProfile struct {
	ConfigMs float64 `json:"configMs"`
	ScanMs   float64 `json:"scanMs"`
	MergeMs  float64 `json:"mergeMs"`
	TotalMs  float64 `json:"totalMs"`
}

// AppMetrics holds runtime performance metrics for the frontend overlay
type AppMetrics struct {
	PID        int     `json:"pid"`
	HeapAlloc  float64 `json:"heapAlloc"`
	Sys        float64 `json:"sys"`
	NumGC      uint32  `json:"numGC"`
	Goroutines int     `json:"goroutines"`
	NumCards   int     `json:"numCards"`
	NumLists   int     `json:"numLists"`
	MaxID      int     `json:"maxID"`
	FileSizeMB float64 `json:"fileSizeMB"`
	ProcessRSS float64 `json:"processRSS"`
	ProcessCPU float64 `json:"processCPU"`
}

// App struct
type App struct {
	ctx          context.Context
	board        *daedalus.BoardState
	watcher      *daedalus.FileWatcher
	prevCPUTicks int64
	prevWallTime time.Time
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// pauseWatcher suppresses the file watcher briefly so the app's own disk writes
// don't trigger a redundant board reload.
func (a *App) pauseWatcher() {
	if a.watcher != nil {
		a.watcher.Suppress(10 * time.Second)
	}
}

// shutdown is called when the app is closing
func (a *App) shutdown(ctx context.Context) {
	if a.watcher != nil {
		a.watcher.Close()
	}
}

// LoadBoard is what is exposed to the frontend
func (a *App) LoadBoard(path string) *BoardResponse {
	if path == "" {
		path = defaultBoardPath
	}
	slog.Info("scanning board", "path", path)

	start := time.Now()
	state, err := daedalus.ScanBoard(path)
	if err != nil {
		slog.Error("board scan failed", "path", path, "error", err)
		return nil
	}
	a.board = state

	numCards := 0
	for _, cards := range state.Lists {
		numCards += len(cards)
	}
	slog.Info("board scan complete",
		"duration", time.Since(start),
		"lists", len(state.Lists),
		"cards", numCards,
		"maxID", state.MaxID,
		"totalBytes", state.TotalFileBytes,
	)

	// Merge discovered list dirs into config array:
	// keep existing entries in order, append new dirs alphabetically, remove stale entries.
	mergeStart := time.Now()
	diskDirs := make(map[string]bool)
	for dirName := range state.Lists {
		diskDirs[dirName] = true
	}

	// Keep existing entries that still exist on disk
	var merged []daedalus.ListEntry
	for _, entry := range state.Config.Lists {
		if diskDirs[entry.Dir] {
			merged = append(merged, entry)
			delete(diskDirs, entry.Dir)
		}
	}

	// Append newly discovered dirs alphabetically
	var newDirs []string
	for dir := range diskDirs {
		newDirs = append(newDirs, dir)
	}
	sort.Strings(newDirs)
	for _, dir := range newDirs {
		merged = append(merged, daedalus.ListEntry{Dir: dir})
	}
	if len(newDirs) > 0 {
		slog.Debug("discovered new list directories", "dirs", newDirs)
	}

	state.Config.Lists = merged
	mergeDuration := time.Since(mergeStart)
	absRoot, _ := filepath.Abs(state.RootPath)

	profile := LoadProfile{
		ConfigMs: float64(state.ConfigLoadTime.Microseconds()) / 1000,
		ScanMs:   float64(state.ScanTime.Microseconds()) / 1000,
		MergeMs:  float64(mergeDuration.Microseconds()) / 1000,
		TotalMs:  float64(time.Since(start).Microseconds()) / 1000,
	}
	slog.Info("load profile",
		"configMs", profile.ConfigMs,
		"scanMs", profile.ScanMs,
		"mergeMs", profile.MergeMs,
		"totalMs", profile.TotalMs,
	)

	// (Re)start the file watcher for external edits.
	if a.watcher != nil {
		a.watcher.Close()
	}
	a.watcher = daedalus.NewFileWatcher(absRoot, func() {
		wailsRuntime.EventsEmit(a.ctx, "board:reload")
	})

	return &BoardResponse{
		Lists:     state.Lists,
		Config:    state.Config,
		BoardPath: absRoot,
		Profile:   profile,
	}
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

	// Process-level metrics from /proc/self
	processRSS := readProcessRSS()
	processCPU := 0.0
	cpuTicks := readProcessCPUTicks()
	now := time.Now()
	if a.prevCPUTicks > 0 && !a.prevWallTime.IsZero() {
		wallDelta := now.Sub(a.prevWallTime).Seconds()
		if wallDelta > 0 {
			// Convert tick delta to seconds then to percentage
			cpuDelta := float64(cpuTicks-a.prevCPUTicks) / linuxClockTicksPerSec
			processCPU = (cpuDelta / wallDelta) * 100
		}
	}
	a.prevCPUTicks = cpuTicks
	a.prevWallTime = now

	return AppMetrics{
		PID:        os.Getpid(),
		HeapAlloc:  float64(m.HeapAlloc) / 1024 / 1024,
		Sys:        float64(m.Sys) / 1024 / 1024,
		NumGC:      m.NumGC,
		Goroutines: runtime.NumGoroutine(),
		NumCards:   numCards,
		NumLists:   numLists,
		MaxID:      maxID,
		FileSizeMB: float64(totalBytes) / 1024 / 1024,
		ProcessRSS: processRSS,
		ProcessCPU: processCPU,
	}
}
