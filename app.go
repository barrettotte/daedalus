package main

import (
	"context"
	"daedalus/pkg/daedalus"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

// BoardResponse is the structure returned to the frontend from LoadBoard.
type BoardResponse struct {
	Lists     map[string][]daedalus.KanbanCard `json:"lists"`
	Config    *daedalus.BoardConfig            `json:"config"`
	BoardPath string                           `json:"boardPath"`
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

	// Merge discovered list dirs into config array:
	// keep existing entries in order, append new dirs alphabetically, remove stale entries.
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

	state.Config.Lists = merged

	absRoot, _ := filepath.Abs(state.RootPath)

	return &BoardResponse{
		Lists:     state.Lists,
		Config:    state.Config,
		BoardPath: absRoot,
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

// SaveShowYearProgress persists the year progress bar visibility to board.yaml.
func (a *App) SaveShowYearProgress(show bool) error {
	if a.board == nil {
		return fmt.Errorf("board not loaded")
	}
	a.board.Config.ShowYearProgress = &show
	return daedalus.SaveBoardConfig(a.board.RootPath, a.board.Config)
}

// SaveLabelColors persists custom label color overrides to board.yaml.
func (a *App) SaveLabelColors(colors map[string]string) error {
	if a.board == nil {
		return fmt.Errorf("board not loaded")
	}
	a.board.Config.LabelColors = colors
	return daedalus.SaveBoardConfig(a.board.RootPath, a.board.Config)
}

// RemoveLabel strips a label from every card that has it, writing each affected card to disk,
// and removes any custom color for that label from board.yaml.
func (a *App) RemoveLabel(label string) error {
	if a.board == nil {
		return fmt.Errorf("board not loaded")
	}

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

			// Remove label from slice
			card.Metadata.Labels = append(card.Metadata.Labels[:idx], card.Metadata.Labels[idx+1:]...)
			now := time.Now()
			card.Metadata.Updated = &now

			body, err := daedalus.ReadCardContent(card.FilePath)
			if err != nil {
				return fmt.Errorf("reading card %s: %w", card.FilePath, err)
			}
			if err := daedalus.WriteCardFile(card.FilePath, card.Metadata, body); err != nil {
				return fmt.Errorf("writing card %s: %w", card.FilePath, err)
			}

			a.board.Lists[listKey][i] = card
		}
	}

	// Remove custom color if set
	if a.board.Config.LabelColors != nil {
		delete(a.board.Config.LabelColors, label)
		if err := daedalus.SaveBoardConfig(a.board.RootPath, a.board.Config); err != nil {
			return fmt.Errorf("saving board config: %w", err)
		}
	}

	return nil
}

// RenameLabel replaces oldName with newName in every card's labels, writing each affected card
// to disk, and migrates any custom color from the old name to the new name in board.yaml.
func (a *App) RenameLabel(oldName string, newName string) error {
	if a.board == nil {
		return fmt.Errorf("board not loaded")
	}
	if oldName == "" || newName == "" || oldName == newName {
		return fmt.Errorf("invalid label names")
	}

	for listKey, cards := range a.board.Lists {
		for i, card := range cards {
			idx := -1

			for j, l := range card.Metadata.Labels {
				if l == oldName {
					idx = j
					break
				}
			}
			if idx == -1 {
				continue
			}

			card.Metadata.Labels[idx] = newName
			now := time.Now()
			card.Metadata.Updated = &now

			body, err := daedalus.ReadCardContent(card.FilePath)
			if err != nil {
				return fmt.Errorf("reading card %s: %w", card.FilePath, err)
			}
			if err := daedalus.WriteCardFile(card.FilePath, card.Metadata, body); err != nil {
				return fmt.Errorf("writing card %s: %w", card.FilePath, err)
			}

			a.board.Lists[listKey][i] = card
		}
	}

	// Migrate custom color if set
	if a.board.Config.LabelColors != nil {
		if color, ok := a.board.Config.LabelColors[oldName]; ok {
			delete(a.board.Config.LabelColors, oldName)
			a.board.Config.LabelColors[newName] = color

			if err := daedalus.SaveBoardConfig(a.board.RootPath, a.board.Config); err != nil {
				return fmt.Errorf("saving board config: %w", err)
			}
		}
	}

	return nil
}

// SaveDarkMode persists the dark mode preference to board.yaml.
func (a *App) SaveDarkMode(dark bool) error {
	if a.board == nil {
		return fmt.Errorf("board not loaded")
	}
	a.board.Config.DarkMode = &dark
	return daedalus.SaveBoardConfig(a.board.RootPath, a.board.Config)
}

// SaveBoardTitle sets the board display title and persists to board.yaml.
func (a *App) SaveBoardTitle(title string) error {
	if a.board == nil {
		return fmt.Errorf("board not loaded")
	}
	a.board.Config.Title = title
	return daedalus.SaveBoardConfig(a.board.RootPath, a.board.Config)
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
	return daedalus.SaveBoardConfig(a.board.RootPath, a.board.Config)
}

// DeleteList removes an entire list directory and cleans up all config references.
func (a *App) DeleteList(listDirName string) error {
	if a.board == nil {
		return fmt.Errorf("board not loaded")
	}

	// Reject names with path separators or traversal
	if strings.ContainsAny(listDirName, "/\\") || strings.Contains(listDirName, "..") {
		return fmt.Errorf("invalid list name")
	}

	// Verify list exists in memory
	cards, ok := a.board.Lists[listDirName]
	if !ok {
		return fmt.Errorf("list not found: %s", listDirName)
	}

	// Sum file bytes for metrics update
	var totalBytes int64
	for _, card := range cards {
		if info, err := os.Stat(card.FilePath); err == nil {
			totalBytes += info.Size()
		}
	}

	// Remove directory from disk
	dirPath := filepath.Join(a.board.RootPath, listDirName)
	if err := os.RemoveAll(dirPath); err != nil {
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

	return daedalus.SaveBoardConfig(a.board.RootPath, a.board.Config)
}

// SaveCollapsedLists sets the Collapsed flag on matching entries and persists to board.yaml.
func (a *App) SaveCollapsedLists(collapsed []string) error {
	if a.board == nil {
		return fmt.Errorf("board not loaded")
	}

	set := make(map[string]bool, len(collapsed))
	for _, dir := range collapsed {
		set[dir] = true
	}
	for i := range a.board.Config.Lists {
		a.board.Config.Lists[i].Collapsed = set[a.board.Config.Lists[i].Dir]
	}

	return daedalus.SaveBoardConfig(a.board.RootPath, a.board.Config)
}

// SaveHalfCollapsedLists sets the HalfCollapsed flag on matching entries and persists to board.yaml.
func (a *App) SaveHalfCollapsedLists(halfCollapsed []string) error {
	if a.board == nil {
		return fmt.Errorf("board not loaded")
	}

	set := make(map[string]bool, len(halfCollapsed))
	for _, dir := range halfCollapsed {
		set[dir] = true
	}
	for i := range a.board.Config.Lists {
		a.board.Config.Lists[i].HalfCollapsed = set[a.board.Config.Lists[i].Dir]
	}

	return daedalus.SaveBoardConfig(a.board.RootPath, a.board.Config)
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

	return daedalus.SaveBoardConfig(a.board.RootPath, a.board.Config)
}

// SaveLockedLists sets the Locked flag on matching entries and persists to board.yaml.
func (a *App) SaveLockedLists(locked []string) error {
	if a.board == nil {
		return fmt.Errorf("board not loaded")
	}

	set := make(map[string]bool, len(locked))
	for _, dir := range locked {
		set[dir] = true
	}
	for i := range a.board.Config.Lists {
		a.board.Config.Lists[i].Locked = set[a.board.Config.Lists[i].Dir]
	}

	return daedalus.SaveBoardConfig(a.board.RootPath, a.board.Config)
}

// isListLocked returns true if the given list directory is marked as locked in the config.
func isListLocked(config *daedalus.BoardConfig, dir string) bool {
	idx := daedalus.FindListEntry(config.Lists, dir)
	return idx >= 0 && config.Lists[idx].Locked
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
		return nil, fmt.Errorf("list not found: %s", listDirName)
	}

	a.board.MaxID++
	newID := a.board.MaxID

	// Determine list_order and insertion index based on position.
	// Numeric strings (e.g. "3") insert at that 0-based index with a midpoint list_order.
	listOrder := 0.0
	insertIdx := 0
	if len(cards) > 0 {
		if position == "bottom" {
			listOrder = cards[len(cards)-1].Metadata.ListOrder + 1
			insertIdx = len(cards)
		} else if idx, err := strconv.Atoi(position); err == nil {
			if idx <= 0 {
				listOrder = cards[0].Metadata.ListOrder - 1
				insertIdx = 0
			} else if idx >= len(cards) {
				listOrder = cards[len(cards)-1].Metadata.ListOrder + 1
				insertIdx = len(cards)
			} else {
				listOrder = (cards[idx-1].Metadata.ListOrder + cards[idx].Metadata.ListOrder) / 2
				insertIdx = idx
			}
		} else {
			// "top" or any unrecognized value
			listOrder = cards[0].Metadata.ListOrder - 1
			insertIdx = 0
		}
	}

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
		return nil, fmt.Errorf("writing new card: %w", err)
	}

	// Update cached total file size
	if info, err := os.Stat(filePath); err == nil {
		a.board.TotalFileBytes += info.Size()
	}

	// Generate preview from body (first ~150 chars)
	preview := body
	if len(preview) > 150 {
		preview = preview[:150]
	}

	card := daedalus.KanbanCard{
		FilePath:    filePath,
		ListName:    listDirName,
		Metadata:    meta,
		PreviewText: preview,
	}

	// Insert card at the computed index
	updated := make([]daedalus.KanbanCard, 0, len(cards)+1)
	updated = append(updated, cards[:insertIdx]...)
	updated = append(updated, card)
	updated = append(updated, cards[insertIdx:]...)
	a.board.Lists[listDirName] = updated

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

	// Capture file size before removal for TotalFileBytes bookkeeping
	var fileSize int64
	if info, err := os.Stat(absPath); err == nil {
		fileSize = info.Size()
	}

	if err := os.Remove(absPath); err != nil {
		return fmt.Errorf("removing card file: %w", err)
	}

	a.board.TotalFileBytes -= fileSize

	// Remove card from in-memory lists
	for listName, cards := range a.board.Lists {
		for i, card := range cards {
			if card.FilePath == absPath {
				a.board.Lists[listName] = append(cards[:i], cards[i+1:]...)
				return nil
			}
		}
	}

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
		return nil, fmt.Errorf("target list not found: %s", targetListDirName)
	}

	// Find card in memory
	var sourceListKey string
	var sourceIdx int
	var found bool
	for listKey, cards := range a.board.Lists {
		for i, card := range cards {
			if card.FilePath == absPath {
				sourceListKey = listKey
				sourceIdx = i
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("card not found in any list")
	}

	// Block moves into or out of locked lists.
	if isListLocked(a.board.Config, sourceListKey) {
		return nil, fmt.Errorf("source list is locked")
	}
	if isListLocked(a.board.Config, targetListDirName) {
		return nil, fmt.Errorf("target list is locked")
	}

	card := a.board.Lists[sourceListKey][sourceIdx]

	// Read card body from disk
	body, err := daedalus.ReadCardContent(absPath)
	if err != nil {
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
			return nil, fmt.Errorf("moving card file: %w", err)
		}
		card.FilePath = newPath
		card.ListName = targetListDirName
	}

	// Write updated frontmatter
	if err := daedalus.WriteCardFile(card.FilePath, card.Metadata, body); err != nil {
		return nil, fmt.Errorf("writing card file: %w", err)
	}

	// Update in-memory state: remove from source
	srcCards := a.board.Lists[sourceListKey]
	a.board.Lists[sourceListKey] = append(srcCards[:sourceIdx], srcCards[sourceIdx+1:]...)

	// Insert at sorted position in target list
	a.board.Lists[targetListDirName] = insertSorted(a.board.Lists[targetListDirName], card)

	return &card, nil
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

// readProcessRSS reads the resident set size from /proc/self/statm in megabytes.
func readProcessRSS() float64 {
	data, err := os.ReadFile("/proc/self/statm")
	if err != nil {
		return 0
	}
	var size, resident int64
	if _, err := fmt.Sscanf(string(data), "%d %d", &size, &resident); err != nil {
		return 0
	}
	return float64(resident*int64(os.Getpagesize())) / 1024 / 1024
}

// readProcessCPUTicks reads utime + stime from /proc/self/stat in clock ticks.
func readProcessCPUTicks() int64 {
	data, err := os.ReadFile("/proc/self/stat")
	if err != nil {
		return 0
	}
	s := string(data)
	// Skip past comm field (may contain spaces/parens) by finding last ")"
	i := strings.LastIndex(s, ")")
	if i < 0 || i+2 >= len(s) {
		return 0
	}
	fields := strings.Fields(s[i+2:])
	if len(fields) < 13 {
		return 0
	}
	utime, _ := strconv.ParseInt(fields[11], 10, 64)
	stime, _ := strconv.ParseInt(fields[12], 10, 64)
	return utime + stime
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
			// Convert tick delta to seconds (100 ticks/sec on Linux) then to percentage
			cpuDelta := float64(cpuTicks-a.prevCPUTicks) / 100.0
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
