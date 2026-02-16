package main

import (
	"context"
	"daedalus/pkg/daedalus"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	defaultBoardPath      = "./tmp/kanban"
	httpClientTimeout     = 10 * time.Second
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

	return &BoardResponse{
		Lists:     state.Lists,
		Config:    state.Config,
		BoardPath: absRoot,
		Profile:   profile,
	}
}

// SaveListConfig updates the config for a single list and persists to board.yaml.
func (a *App) SaveListConfig(dirName string, title string, limit int) error {
	if a.board == nil {
		return fmt.Errorf("board not loaded")
	}

	idx := daedalus.FindListEntry(a.board.Config.Lists, dirName)
	if idx >= 0 {
		a.board.Config.Lists[idx].Title = title
		a.board.Config.Lists[idx].Limit = limit
	} else {
		a.board.Config.Lists = append(a.board.Config.Lists, daedalus.ListEntry{
			Dir:   dirName,
			Title: title,
			Limit: limit,
		})
	}

	if err := daedalus.SaveBoardConfig(a.board.RootPath, a.board.Config); err != nil {
		slog.Error("failed to save list config", "dir", dirName, "error", err)
		return err
	}
	slog.Info("list config saved", "dir", dirName, "title", title, "limit", limit)
	return nil
}

// SaveLabelsExpanded persists the label collapsed/expanded state to board.yaml.
func (a *App) SaveLabelsExpanded(expanded bool) error {
	if a.board == nil {
		return fmt.Errorf("board not loaded")
	}
	a.board.Config.LabelsExpanded = &expanded

	if err := daedalus.SaveBoardConfig(a.board.RootPath, a.board.Config); err != nil {
		slog.Error("failed to save labels expanded", "error", err)
		return err
	}
	slog.Debug("labels expanded state saved", "expanded", expanded)
	return nil
}

// SaveShowYearProgress persists the year progress bar visibility to board.yaml.
func (a *App) SaveShowYearProgress(show bool) error {
	if a.board == nil {
		return fmt.Errorf("board not loaded")
	}
	a.board.Config.ShowYearProgress = &show

	if err := daedalus.SaveBoardConfig(a.board.RootPath, a.board.Config); err != nil {
		slog.Error("failed to save year progress", "error", err)
		return err
	}
	slog.Debug("year progress state saved", "show", show)
	return nil
}

// SaveLabelColors persists custom label color overrides to board.yaml.
func (a *App) SaveLabelColors(colors map[string]string) error {
	if a.board == nil {
		return fmt.Errorf("board not loaded")
	}
	a.board.Config.LabelColors = colors

	if err := daedalus.SaveBoardConfig(a.board.RootPath, a.board.Config); err != nil {
		slog.Error("failed to save label colors", "error", err)
		return err
	}
	slog.Debug("label colors saved", "count", len(colors))
	return nil
}

// updateCardsWithLabel finds every card containing the given label, applies transformFn to modify
// the card's labels, writes the updated card to disk, and returns the count of affected cards.
func (a *App) updateCardsWithLabel(label string, transformFn func(labels []string, idx int) []string) (int, error) {
	affected := 0
	for listKey, cards := range a.board.Lists {
		for i, card := range cards {
			idx := -1
			for j, l := range card.Metadata.Labels {
				if l == label {
					idx = j
					break
				}
			}
			if idx == -1 {
				continue
			}

			card.Metadata.Labels = transformFn(card.Metadata.Labels, idx)
			now := time.Now()
			card.Metadata.Updated = &now

			body, err := daedalus.ReadCardContent(card.FilePath)
			if err != nil {
				return affected, fmt.Errorf("reading card %s: %w", card.FilePath, err)
			}
			if err := daedalus.WriteCardFile(card.FilePath, card.Metadata, body); err != nil {
				return affected, fmt.Errorf("writing card %s: %w", card.FilePath, err)
			}

			a.board.Lists[listKey][i] = card
			affected++
		}
	}
	return affected, nil
}

// RemoveLabel strips a label from every card that has it, writing each affected card to disk,
// and removes any custom color for that label from board.yaml.
func (a *App) RemoveLabel(label string) error {
	if a.board == nil {
		return fmt.Errorf("board not loaded")
	}
	slog.Info("removing label from all cards", "label", label)

	affected, err := a.updateCardsWithLabel(label, func(labels []string, idx int) []string {
		return append(labels[:idx], labels[idx+1:]...)
	})
	if err != nil {
		slog.Error("failed during label removal", "label", label, "error", err)
		return err
	}

	// Remove custom color if set
	if a.board.Config.LabelColors != nil {
		delete(a.board.Config.LabelColors, label)
		if err := daedalus.SaveBoardConfig(a.board.RootPath, a.board.Config); err != nil {
			slog.Error("failed to save config after label removal", "label", label, "error", err)
			return fmt.Errorf("saving board config: %w", err)
		}
	}

	slog.Info("label removed", "label", label, "cardsAffected", affected)
	return nil
}

// RenameLabel replaces oldName with newName in every card's labels, writing each affected card
// to disk, and migrates any custom color from the old name to the new name in board.yaml.
func (a *App) RenameLabel(oldName string, newName string) error {
	if a.board == nil {
		return fmt.Errorf("board not loaded")
	}
	if oldName == "" || newName == "" || oldName == newName {
		slog.Warn("invalid label rename parameters", "old", oldName, "new", newName)
		return fmt.Errorf("invalid label names")
	}
	slog.Info("renaming label", "old", oldName, "new", newName)

	affected, err := a.updateCardsWithLabel(oldName, func(labels []string, idx int) []string {
		labels[idx] = newName
		return labels
	})
	if err != nil {
		slog.Error("failed during label rename", "old", oldName, "new", newName, "error", err)
		return err
	}

	// Migrate custom color if set
	if a.board.Config.LabelColors != nil {
		if color, ok := a.board.Config.LabelColors[oldName]; ok {
			delete(a.board.Config.LabelColors, oldName)
			a.board.Config.LabelColors[newName] = color

			if err := daedalus.SaveBoardConfig(a.board.RootPath, a.board.Config); err != nil {
				slog.Error("failed to save config after label rename", "error", err)
				return fmt.Errorf("saving board config: %w", err)
			}
		}
	}

	slog.Info("label renamed", "old", oldName, "new", newName, "cardsAffected", affected)
	return nil
}

// SaveDarkMode persists the dark mode preference to board.yaml.
func (a *App) SaveDarkMode(dark bool) error {
	if a.board == nil {
		return fmt.Errorf("board not loaded")
	}
	a.board.Config.DarkMode = &dark
	if err := daedalus.SaveBoardConfig(a.board.RootPath, a.board.Config); err != nil {
		slog.Error("failed to save dark mode", "error", err)
		return err
	}
	slog.Debug("dark mode saved", "dark", dark)
	return nil
}

// SaveBoardTitle sets the board display title and persists to board.yaml.
func (a *App) SaveBoardTitle(title string) error {
	if a.board == nil {
		return fmt.Errorf("board not loaded")
	}
	a.board.Config.Title = title

	if err := daedalus.SaveBoardConfig(a.board.RootPath, a.board.Config); err != nil {
		slog.Error("failed to save board title", "error", err)
		return err
	}
	slog.Info("board title saved", "title", title)
	return nil
}

// SaveListOrder reorders the config Lists array to match the given order and persists to board.yaml.
func (a *App) SaveListOrder(order []string) error {
	if a.board == nil {
		return fmt.Errorf("board not loaded")
	}

	// Build a map of dir -> entry for quick lookup
	entryMap := make(map[string]daedalus.ListEntry)
	for _, entry := range a.board.Config.Lists {
		entryMap[entry.Dir] = entry
	}

	// Reassemble in new order
	var reordered []daedalus.ListEntry
	used := make(map[string]bool)
	for _, dir := range order {
		if entry, ok := entryMap[dir]; ok {
			reordered = append(reordered, entry)
			used[dir] = true
		}
	}

	// Append any stragglers not in the order array
	for _, entry := range a.board.Config.Lists {
		if !used[entry.Dir] {
			reordered = append(reordered, entry)
		}
	}

	a.board.Config.Lists = reordered
	if err := daedalus.SaveBoardConfig(a.board.RootPath, a.board.Config); err != nil {
		slog.Error("failed to save list order", "error", err)
		return err
	}
	slog.Info("list order saved", "count", len(reordered))
	return nil
}

// DeleteList removes an entire list directory and cleans up all config references.
func (a *App) DeleteList(listDirName string) error {
	if a.board == nil {
		return fmt.Errorf("board not loaded")
	}

	// Reject names with path separators or traversal
	if strings.ContainsAny(listDirName, "/\\") || strings.Contains(listDirName, "..") {
		slog.Warn("rejected invalid list name", "name", listDirName)
		return fmt.Errorf("invalid list name")
	}

	// Verify list exists in memory
	cards, ok := a.board.Lists[listDirName]
	if !ok {
		slog.Warn("attempted to delete non-existent list", "name", listDirName)
		return fmt.Errorf("list not found: %s", listDirName)
	}

	slog.Info("deleting list", "name", listDirName, "cards", len(cards))

	// Sum file bytes for metrics update
	var totalBytes int64
	for _, card := range cards {
		totalBytes += getFileSize(card.FilePath)
	}

	// Remove directory from disk
	dirPath := filepath.Join(a.board.RootPath, listDirName)
	if err := os.RemoveAll(dirPath); err != nil {
		slog.Error("failed to remove list directory", "name", listDirName, "path", dirPath, "error", err)
		return fmt.Errorf("removing list directory: %w", err)
	}

	// Update metrics
	a.board.TotalFileBytes -= totalBytes

	// Clean up in-memory state
	delete(a.board.Lists, listDirName)

	// Remove from config Lists array
	idx := daedalus.FindListEntry(a.board.Config.Lists, listDirName)
	if idx >= 0 {
		a.board.Config.Lists = append(a.board.Config.Lists[:idx], a.board.Config.Lists[idx+1:]...)
	}

	if err := daedalus.SaveBoardConfig(a.board.RootPath, a.board.Config); err != nil {
		slog.Error("failed to save config after list deletion", "name", listDirName, "error", err)
		return err
	}
	slog.Info("list deleted", "name", listDirName, "cardsRemoved", len(cards), "bytesFreed", totalBytes)
	return nil
}

// saveListBoolFlags builds a set from dirs, applies setFn to each list config entry, and persists to board.yaml.
func (a *App) saveListBoolFlags(dirs []string, setFn func(*daedalus.ListEntry, bool)) error {
	if a.board == nil {
		return fmt.Errorf("board not loaded")
	}
	set := make(map[string]bool, len(dirs))
	for _, dir := range dirs {
		set[dir] = true
	}
	for i := range a.board.Config.Lists {
		setFn(&a.board.Config.Lists[i], set[a.board.Config.Lists[i].Dir])
	}
	return daedalus.SaveBoardConfig(a.board.RootPath, a.board.Config)
}

// SaveCollapsedLists sets the Collapsed flag on matching entries and persists to board.yaml.
func (a *App) SaveCollapsedLists(collapsed []string) error {
	if err := a.saveListBoolFlags(collapsed, func(e *daedalus.ListEntry, v bool) { e.Collapsed = v }); err != nil {
		slog.Error("failed to save collapsed lists", "error", err)
		return err
	}
	slog.Debug("collapsed lists saved", "count", len(collapsed))
	return nil
}

// SaveHalfCollapsedLists sets the HalfCollapsed flag on matching entries and persists to board.yaml.
func (a *App) SaveHalfCollapsedLists(halfCollapsed []string) error {
	if err := a.saveListBoolFlags(halfCollapsed, func(e *daedalus.ListEntry, v bool) { e.HalfCollapsed = v }); err != nil {
		slog.Error("failed to save half-collapsed lists", "error", err)
		return err
	}
	slog.Debug("half-collapsed lists saved", "count", len(halfCollapsed))
	return nil
}

// SavePinnedLists sets the Pinned field on matching entries and persists to board.yaml.
func (a *App) SavePinnedLists(left []string, right []string) error {
	if a.board == nil {
		return fmt.Errorf("board not loaded")
	}

	leftSet := make(map[string]bool, len(left))
	for _, dir := range left {
		leftSet[dir] = true
	}
	rightSet := make(map[string]bool, len(right))
	for _, dir := range right {
		rightSet[dir] = true
	}

	for i := range a.board.Config.Lists {
		dir := a.board.Config.Lists[i].Dir
		if leftSet[dir] {
			a.board.Config.Lists[i].Pinned = "left"
		} else if rightSet[dir] {
			a.board.Config.Lists[i].Pinned = "right"
		} else {
			a.board.Config.Lists[i].Pinned = ""
		}
	}

	if err := daedalus.SaveBoardConfig(a.board.RootPath, a.board.Config); err != nil {
		slog.Error("failed to save pinned lists", "error", err)
		return err
	}
	slog.Debug("pinned lists saved", "left", len(left), "right", len(right))
	return nil
}

// SaveLockedLists sets the Locked flag on matching entries and persists to board.yaml.
func (a *App) SaveLockedLists(locked []string) error {
	if err := a.saveListBoolFlags(locked, func(e *daedalus.ListEntry, v bool) { e.Locked = v }); err != nil {
		slog.Error("failed to save locked lists", "error", err)
		return err
	}
	slog.Info("locked lists saved", "count", len(locked))
	return nil
}

// isListLocked returns true if the given list directory is marked as locked in the config.
func isListLocked(config *daedalus.BoardConfig, dir string) bool {
	idx := daedalus.FindListEntry(config.Lists, dir)
	return idx >= 0 && config.Lists[idx].Locked
}

// getFileSize returns the size of a file in bytes, or 0 if the file cannot be stat'd.
func getFileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		slog.Warn("failed to stat file for size", "path", path, "error", err)
		return 0
	}
	return info.Size()
}

// truncatePreview returns a body preview truncated to PreviewMaxLen characters.
func truncatePreview(body string) string {
	if len(body) > daedalus.PreviewMaxLen {
		return body[:daedalus.PreviewMaxLen]
	}
	return body
}

// iconsDir returns the path to the board's icons directory.
func (a *App) iconsDir() string {
	return filepath.Join(a.board.RootPath, "assets", "icons")
}

// isIconExt returns true if the file extension is a supported icon type (.svg or .png).
func isIconExt(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".svg" || ext == ".png"
}

// validatePath resolves a file path to absolute and verifies it is within the board root.
func (a *App) validatePath(filePath string) (string, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		slog.Warn("path resolution failed", "path", filePath, "error", err)
		return "", fmt.Errorf("invalid path")
	}
	absRoot, err := filepath.Abs(a.board.RootPath)
	if err != nil {
		slog.Error("board root path resolution failed", "root", a.board.RootPath, "error", err)
		return "", fmt.Errorf("invalid root path")
	}
	if !strings.HasPrefix(absPath, absRoot+string(filepath.Separator)) {
		slog.Warn("path traversal rejected", "path", absPath, "root", absRoot)
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
		slog.Error("unsupported platform for external open", "os", runtime.GOOS)
		return fmt.Errorf("unsupported platform")
	}

	if err := cmd.Start(); err != nil {
		slog.Error("failed to open file externally", "path", absPath, "error", err)
		return err
	}
	slog.Debug("opened file externally", "path", absPath)
	return nil
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

	oldSize := getFileSize(absPath)

	if err := daedalus.WriteCardFile(absPath, metadata, body); err != nil {
		slog.Error("failed to write card", "id", metadata.ID, "file", absPath, "error", err)
		return nil, fmt.Errorf("writing card file: %w", err)
	}

	a.board.TotalFileBytes += getFileSize(absPath) - oldSize

	updatedCard := daedalus.KanbanCard{
		FilePath:    absPath,
		Metadata:    metadata,
		PreviewText: truncatePreview(body),
	}

	// Update card in-place in board lists
	if listKey, idx, found := a.findCardByPath(absPath); found {
		updatedCard.ListName = a.board.Lists[listKey][idx].ListName
		a.board.Lists[listKey][idx] = updatedCard
		slog.Info("card saved", "id", metadata.ID, "list", listKey, "title", metadata.Title)
	} else {
		slog.Info("card saved", "id", metadata.ID, "title", metadata.Title)
	}
	return &updatedCard, nil
}

// CreateCard creates a new card in the given list directory, writes it to disk,
// updates in-memory state, and returns the new KanbanCard.
// Position "bottom" appends, a numeric string inserts at that 0-based index,
// and anything else (including "top") prepends.
func (a *App) CreateCard(listDirName string, title string, body string, position string) (*daedalus.KanbanCard, error) {
	if a.board == nil {
		return nil, fmt.Errorf("board not loaded")
	}

	cards, ok := a.board.Lists[listDirName]
	if !ok {
		slog.Error("cannot create card in non-existent list", "list", listDirName)
		return nil, fmt.Errorf("list not found: %s", listDirName)
	}

	a.board.MaxID++
	newID := a.board.MaxID
	slog.Debug("assigning new card ID", "id", newID, "previousMaxID", newID-1)

	listOrder, insertIdx := computeInsertPosition(cards, position)

	// Use provided title, falling back to ID string if empty
	if strings.TrimSpace(title) == "" {
		title = fmt.Sprintf("%d", newID)
	}

	now := time.Now()
	meta := daedalus.CardMetadata{
		ID:        newID,
		Title:     title,
		Created:   &now,
		Updated:   &now,
		ListOrder: listOrder,
	}

	// Construct full file body matching SaveCard pattern
	fullBody := fmt.Sprintf("# %s\n\n%s", title, body)

	filePath := filepath.Join(a.board.RootPath, listDirName, fmt.Sprintf("%d.md", newID))
	if err := daedalus.WriteCardFile(filePath, meta, fullBody); err != nil {
		slog.Error("failed to write new card", "id", newID, "list", listDirName, "error", err)
		return nil, fmt.Errorf("writing new card: %w", err)
	}

	a.board.TotalFileBytes += getFileSize(filePath)

	card := daedalus.KanbanCard{
		FilePath:    filePath,
		ListName:    listDirName,
		Metadata:    meta,
		PreviewText: truncatePreview(body),
	}

	// Insert card at the computed index
	updated := make([]daedalus.KanbanCard, 0, len(cards)+1)
	updated = append(updated, cards[:insertIdx]...)
	updated = append(updated, card)
	updated = append(updated, cards[insertIdx:]...)
	a.board.Lists[listDirName] = updated

	slog.Info("card created", "id", newID, "list", listDirName, "title", title, "position", position)
	return &card, nil
}

// DeleteCard removes a card file from disk and from the in-memory board state.
// MaxID is intentionally not decremented (high-water mark for unique IDs).
func (a *App) DeleteCard(filePath string) error {
	if a.board == nil {
		return fmt.Errorf("board not loaded")
	}

	absPath, err := a.validatePath(filePath)
	if err != nil {
		return err
	}

	removedBytes := getFileSize(absPath)

	if err := os.Remove(absPath); err != nil {
		slog.Error("failed to remove card file", "path", absPath, "error", err)
		return fmt.Errorf("removing card file: %w", err)
	}

	a.board.TotalFileBytes -= removedBytes

	// Remove card from in-memory lists
	for listName, cards := range a.board.Lists {
		for i, card := range cards {
			if card.FilePath == absPath {
				a.board.Lists[listName] = append(cards[:i], cards[i+1:]...)
				slog.Info("card deleted", "id", card.Metadata.ID, "list", listName, "bytesFreed", removedBytes)
				return nil
			}
		}
	}

	slog.Warn("deleted card file not found in memory", "path", absPath)
	return nil
}

// MoveCard moves a card to a target list at a given list_order. Handles both same-list
// reordering and cross-list moves (file rename to new directory).
func (a *App) MoveCard(filePath string, targetListDirName string, newListOrder float64) (*daedalus.KanbanCard, error) {
	if a.board == nil {
		return nil, fmt.Errorf("board not loaded")
	}

	absPath, err := a.validatePath(filePath)
	if err != nil {
		return nil, err
	}

	// Validate target list exists
	if _, ok := a.board.Lists[targetListDirName]; !ok {
		slog.Error("move target list not found", "target", targetListDirName)
		return nil, fmt.Errorf("target list not found: %s", targetListDirName)
	}

	// Find card in memory
	sourceListKey, sourceIdx, found := a.findCardByPath(absPath)
	if !found {
		slog.Error("card not found in any list", "path", absPath)
		return nil, fmt.Errorf("card not found in any list")
	}

	// Block moves into or out of locked lists.
	if isListLocked(a.board.Config, sourceListKey) {
		slog.Warn("move blocked by locked source list", "source", sourceListKey)
		return nil, fmt.Errorf("source list is locked")
	}
	if isListLocked(a.board.Config, targetListDirName) {
		slog.Warn("move blocked by locked target list", "target", targetListDirName)
		return nil, fmt.Errorf("target list is locked")
	}

	card := a.board.Lists[sourceListKey][sourceIdx]

	// Read card body from disk
	body, err := daedalus.ReadCardContent(absPath)
	if err != nil {
		slog.Error("failed to read card content for move", "path", absPath, "error", err)
		return nil, fmt.Errorf("reading card content: %w", err)
	}

	// Update metadata
	now := time.Now()
	card.Metadata.Updated = &now
	card.Metadata.ListOrder = newListOrder

	// Determine new file path
	filename := filepath.Base(absPath)
	newPath := filepath.Join(a.board.RootPath, targetListDirName, filename)

	crossList := sourceListKey != targetListDirName

	if crossList {
		// Move file to new directory
		if err := os.Rename(absPath, newPath); err != nil {
			slog.Error("failed to move card file", "from", absPath, "to", newPath, "error", err)
			return nil, fmt.Errorf("moving card file: %w", err)
		}
		card.FilePath = newPath
		card.ListName = targetListDirName
	}

	// Write updated frontmatter
	if err := daedalus.WriteCardFile(card.FilePath, card.Metadata, body); err != nil {
		slog.Error("failed to write card after move", "path", card.FilePath, "error", err)
		return nil, fmt.Errorf("writing card file: %w", err)
	}

	// Update in-memory state: remove from source
	srcCards := a.board.Lists[sourceListKey]
	a.board.Lists[sourceListKey] = append(srcCards[:sourceIdx], srcCards[sourceIdx+1:]...)

	// Insert at sorted position in target list
	a.board.Lists[targetListDirName] = insertSorted(a.board.Lists[targetListDirName], card)

	if crossList {
		slog.Info("card moved", "id", card.Metadata.ID, "from", sourceListKey, "to", targetListDirName)
	} else {
		slog.Debug("card reordered", "id", card.Metadata.ID, "list", sourceListKey, "order", newListOrder)
	}
	return &card, nil
}

// findCardByPath searches all board lists for a card with the given file path.
// Returns the list key, index within that list, and whether the card was found.
func (a *App) findCardByPath(absPath string) (string, int, bool) {
	for listKey, cards := range a.board.Lists {
		for i, card := range cards {
			if card.FilePath == absPath {
				return listKey, i, true
			}
		}
	}
	return "", 0, false
}

// computeInsertPosition determines list_order and insertion index for a new card.
// Position "bottom" appends, a numeric string inserts at that 0-based index,
// and anything else (including "top") prepends.
func computeInsertPosition(cards []daedalus.KanbanCard, position string) (float64, int) {
	if len(cards) == 0 {
		return 0, 0
	}
	if position == "bottom" {
		return cards[len(cards)-1].Metadata.ListOrder + 1, len(cards)
	}
	if idx, err := strconv.Atoi(position); err == nil {
		if idx <= 0 {
			return cards[0].Metadata.ListOrder - 1, 0
		}
		if idx >= len(cards) {
			return cards[len(cards)-1].Metadata.ListOrder + 1, len(cards)
		}
		return (cards[idx-1].Metadata.ListOrder + cards[idx].Metadata.ListOrder) / 2, idx
	}
	// "top" or any unrecognized value
	return cards[0].Metadata.ListOrder - 1, 0
}

// insertSorted inserts a card into a sorted slice at the correct position by ListOrder then ID.
func insertSorted(cards []daedalus.KanbanCard, card daedalus.KanbanCard) []daedalus.KanbanCard {
	idx := sort.Search(len(cards), func(i int) bool {
		if cards[i].Metadata.ListOrder != card.Metadata.ListOrder {
			return cards[i].Metadata.ListOrder > card.Metadata.ListOrder
		}
		return cards[i].Metadata.ID > card.Metadata.ID
	})
	cards = append(cards, daedalus.KanbanCard{})
	copy(cards[idx+1:], cards[idx:])
	cards[idx] = card
	return cards
}

// ListIcons returns a sorted list of icon filenames from {boardRoot}/assets/icons/.
// Returns an empty list if the directory does not exist.
func (a *App) ListIcons() ([]string, error) {
	if a.board == nil {
		return nil, fmt.Errorf("board not loaded")
	}

	dir := a.iconsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		slog.Error("failed to read icons directory", "path", dir, "error", err)
		return nil, fmt.Errorf("reading icons directory: %w", err)
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if isIconExt(entry.Name()) {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

// GetIconContent returns the content of a custom icon file.
// For .svg files, returns the raw SVG markup.
// For .png files, returns a base64 data URI.
func (a *App) GetIconContent(name string) (string, error) {
	if a.board == nil {
		return "", fmt.Errorf("board not loaded")
	}

	if strings.ContainsAny(name, "/\\") || strings.Contains(name, "..") {
		slog.Warn("rejected invalid icon name", "name", name)
		return "", fmt.Errorf("invalid icon name")
	}

	iconPath := filepath.Join(a.board.RootPath, "assets", "icons", name)
	absPath, err := a.validatePath(iconPath)
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("icon not found: %s", name)
		}
		slog.Error("failed to read icon file", "name", name, "error", err)
		return "", fmt.Errorf("reading icon: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(name))
	if ext == ".svg" {
		return string(data), nil
	}
	if ext == ".png" {
		encoded := base64.StdEncoding.EncodeToString(data)
		return "data:image/png;base64," + encoded, nil
	}
	return "", fmt.Errorf("unsupported icon type: %s", ext)
}

// SaveCustomIcon saves an uploaded icon file to {boardRoot}/assets/icons/.
// For .svg files, content is raw SVG text. For .png files, content is base64-encoded.
func (a *App) SaveCustomIcon(name string, content string) error {
	if a.board == nil {
		return fmt.Errorf("board not loaded")
	}

	if strings.ContainsAny(name, "/\\") || strings.Contains(name, "..") {
		slog.Warn("rejected invalid icon name", "name", name)
		return fmt.Errorf("invalid icon name")
	}

	dir := a.iconsDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		slog.Error("failed to create icons directory", "path", dir, "error", err)
		return fmt.Errorf("creating icons directory: %w", err)
	}

	iconPath := filepath.Join(dir, name)
	absPath, err := a.validatePath(iconPath)
	if err != nil {
		return err
	}

	ext := strings.ToLower(filepath.Ext(name))
	if ext == ".svg" {
		if !strings.Contains(content, "<svg") {
			return fmt.Errorf("invalid SVG content")
		}
		if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
			slog.Error("failed to write SVG icon", "name", name, "error", err)
			return fmt.Errorf("writing icon: %w", err)
		}
	} else if ext == ".png" {
		data, err := base64.StdEncoding.DecodeString(content)
		if err != nil {
			return fmt.Errorf("invalid base64 content: %w", err)
		}
		if err := os.WriteFile(absPath, data, 0644); err != nil {
			slog.Error("failed to write PNG icon", "name", name, "error", err)
			return fmt.Errorf("writing icon: %w", err)
		}
	} else {
		return fmt.Errorf("unsupported icon type: %s", ext)
	}

	slog.Info("custom icon saved", "name", name)
	return nil
}

// DownloadIcon fetches an icon from a URL and saves it to {boardRoot}/assets/icons/.
// Only .svg and .png extensions are allowed. Returns the filename on success.
func (a *App) DownloadIcon(rawURL string) (string, error) {
	if a.board == nil {
		return "", fmt.Errorf("board not loaded")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		slog.Warn("invalid icon download URL", "url", rawURL)
		return "", fmt.Errorf("invalid URL")
	}

	// Extract filename from the URL path's last segment
	filename := filepath.Base(parsed.Path)
	if filename == "" || filename == "." || filename == "/" {
		return "", fmt.Errorf("could not determine filename from URL")
	}

	if !isIconExt(filename) {
		return "", fmt.Errorf("unsupported file type: %s (only .svg and .png)", filepath.Ext(filename))
	}

	client := &http.Client{Timeout: httpClientTimeout}
	resp, err := client.Get(rawURL)
	if err != nil {
		slog.Error("icon download failed", "url", rawURL, "error", err)
		return "", fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	// Validate content
	if strings.ToLower(filepath.Ext(filename)) == ".svg" {
		if !strings.Contains(string(body), "<svg") {
			return "", fmt.Errorf("invalid SVG content")
		}
	}

	// Save to icons directory
	dir := a.iconsDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		slog.Error("failed to create icons directory", "path", dir, "error", err)
		return "", fmt.Errorf("creating icons directory: %w", err)
	}

	iconPath := filepath.Join(dir, filename)
	absPath, err := a.validatePath(iconPath)
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(absPath, body, 0644); err != nil {
		slog.Error("failed to write downloaded icon", "name", filename, "error", err)
		return "", fmt.Errorf("writing icon: %w", err)
	}

	slog.Info("icon downloaded", "name", filename, "url", rawURL, "bytes", len(body))
	return filename, nil
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
