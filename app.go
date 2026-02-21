package main

import (
	"context"
	"daedalus/pkg/daedalus"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
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
	appConfigDir string
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// GetVersion returns the app version set at build time.
func (a *App) GetVersion() string {
	return version
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// resolveConfigDir returns the app config directory, falling back to os.UserConfigDir.
func (a *App) resolveConfigDir() string {
	if a.appConfigDir != "" {
		return a.appConfigDir
	}

	base, err := os.UserConfigDir()
	if err != nil {
		slog.Error("failed to get user config dir", "error", err)
		return ""
	}
	return filepath.Join(base, "daedalus")
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

// GetAppConfig returns the app-level configuration (default board, recent boards).
func (a *App) GetAppConfig() *daedalus.AppConfig {
	dir := a.resolveConfigDir()
	if dir == "" {
		return &daedalus.AppConfig{}
	}

	cfg, err := daedalus.LoadAppConfig(dir)
	if err != nil {
		slog.Error("failed to load app config", "error", err)
		return &daedalus.AppConfig{}
	}
	if daedalus.PruneInvalidBoards(cfg) {
		if saveErr := daedalus.SaveAppConfig(dir, cfg); saveErr != nil {
			slog.Error("failed to save pruned app config", "error", saveErr)
		}
	}
	return cfg
}

// SetDefaultBoard sets or clears the default board path in app config.
func (a *App) SetDefaultBoard(path string) error {
	dir := a.resolveConfigDir()
	if dir == "" {
		return fmt.Errorf("unable to resolve config directory")
	}

	cfg, err := daedalus.LoadAppConfig(dir)
	if err != nil {
		return err
	}
	cfg.DefaultBoard = path
	return daedalus.SaveAppConfig(dir, cfg)
}

// RemoveRecentBoard removes a board from the recent boards list.
func (a *App) RemoveRecentBoard(path string) error {
	dir := a.resolveConfigDir()
	if dir == "" {
		return fmt.Errorf("unable to resolve config directory")
	}

	cfg, err := daedalus.LoadAppConfig(dir)
	if err != nil {
		return err
	}
	daedalus.RemoveRecentBoard(cfg, path)
	return daedalus.SaveAppConfig(dir, cfg)
}

// OpenDirectoryDialog opens a native OS directory picker and returns the selected path.
func (a *App) OpenDirectoryDialog() string {
	path, err := wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Select Board Directory",
	})
	if err != nil {
		slog.Error("directory dialog failed", "error", err)
		return ""
	}
	return path
}

// SaveFileDialog opens a native OS save-file dialog and returns the selected path.
func (a *App) SaveFileDialog(defaultFilename string) string {
	path, err := wailsRuntime.SaveFileDialog(a.ctx, wailsRuntime.SaveDialogOptions{
		DefaultFilename:      defaultFilename,
		CanCreateDirectories: true,
		Title:                "Export Board",
	})
	if err != nil {
		slog.Error("save dialog failed", "error", err)
		return ""
	}
	return path
}

// updateRecentBoards loads the app config, adds the board to the recent list, and saves.
// Errors are logged internally -- this is best-effort and should not block board loading.
func (a *App) updateRecentBoards(absRoot, title string) {
	cfgDir := a.resolveConfigDir()
	if cfgDir == "" {
		return
	}
	appCfg, err := daedalus.LoadAppConfig(cfgDir)
	if err != nil {
		slog.Error("failed to load app config for recent boards", "error", err)
		return
	}
	daedalus.AddRecentBoard(appCfg, absRoot, title, time.Now().UTC())
	if saveErr := daedalus.SaveAppConfig(cfgDir, appCfg); saveErr != nil {
		slog.Error("failed to save app config after board load", "error", saveErr)
	}
}

// buildLoadProfile constructs and logs timing data for each phase of board loading.
func buildLoadProfile(state *daedalus.BoardState, mergeDuration time.Duration, totalStart time.Time) LoadProfile {
	profile := LoadProfile{
		ConfigMs: float64(state.ConfigLoadTime.Microseconds()) / 1000,
		ScanMs:   float64(state.ScanTime.Microseconds()) / 1000,
		MergeMs:  float64(mergeDuration.Microseconds()) / 1000,
		TotalMs:  float64(time.Since(totalStart).Microseconds()) / 1000,
	}
	slog.Info("load profile",
		"configMs", profile.ConfigMs,
		"scanMs", profile.ScanMs,
		"mergeMs", profile.MergeMs,
		"totalMs", profile.TotalMs,
	)
	return profile
}

// LoadBoard is what is exposed to the frontend
func (a *App) LoadBoard(path string) *BoardResponse {
	if path == "" {
		slog.Error("LoadBoard called with empty path")
		return nil
	}

	// Ensure the board directory and board.yaml exist so ScanBoard succeeds.
	if err := daedalus.InitBoardDir(path); err != nil {
		return nil
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

	// Merge discovered list dirs into config (preserves order, appends new, removes stale).
	mergeStart := time.Now()
	diskDirs := make(map[string]bool)
	for dirName := range state.Lists {
		diskDirs[dirName] = true
	}
	daedalus.MergeListEntries(state.Config, diskDirs)
	mergeDuration := time.Since(mergeStart)
	absRoot, _ := filepath.Abs(state.RootPath)

	a.updateRecentBoards(absRoot, state.Config.Title)
	profile := buildLoadProfile(state, mergeDuration, start)

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
	processRSS := daedalus.ReadProcessRSS()
	processCPU := 0.0
	cpuTicks := daedalus.ReadProcessCPUTicks()
	now := time.Now()
	if a.prevCPUTicks > 0 && !a.prevWallTime.IsZero() {
		wallDelta := now.Sub(a.prevWallTime).Seconds()
		if wallDelta > 0 {
			// Convert tick delta to seconds then to percentage
			cpuDelta := float64(cpuTicks-a.prevCPUTicks) / daedalus.ClockTicksPerSec
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
