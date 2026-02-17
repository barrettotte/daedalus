package main

import (
	"daedalus/pkg/daedalus"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

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
	a.pauseWatcher()

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
	a.pauseWatcher()

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
	a.pauseWatcher()

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
	a.pauseWatcher()

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

// MoveAllCards moves every card from sourceDir into targetDir, appending after existing cards.
func (a *App) MoveAllCards(sourceDir, targetDir string) error {
	if a.board == nil {
		return fmt.Errorf("board not loaded")
	}
	if sourceDir == targetDir {
		return fmt.Errorf("source and target lists are the same")
	}

	// Validate both lists exist.
	if _, ok := a.board.Lists[sourceDir]; !ok {
		return fmt.Errorf("source list not found: %s", sourceDir)
	}
	if _, ok := a.board.Lists[targetDir]; !ok {
		return fmt.Errorf("target list not found: %s", targetDir)
	}

	// Block if either list is locked.
	if isListLocked(a.board.Config, sourceDir) {
		return fmt.Errorf("source list is locked")
	}
	if isListLocked(a.board.Config, targetDir) {
		return fmt.Errorf("target list is locked")
	}

	srcCards := a.board.Lists[sourceDir]
	if len(srcCards) == 0 {
		return nil
	}

	a.pauseWatcher()

	// Find max ListOrder in target to append after existing cards.
	maxOrder := 0.0
	if targetCards := a.board.Lists[targetDir]; len(targetCards) > 0 {
		maxOrder = targetCards[len(targetCards)-1].Metadata.ListOrder
	}

	now := time.Now()

	for i, card := range srcCards {
		body, err := daedalus.ReadCardContent(card.FilePath)
		if err != nil {
			slog.Error("failed to read card content for move-all", "path", card.FilePath, "error", err)
			return fmt.Errorf("reading card %d: %w", card.Metadata.ID, err)
		}

		card.Metadata.Updated = &now
		card.Metadata.ListOrder = maxOrder + float64(i) + 1.0

		filename := filepath.Base(card.FilePath)
		newPath := filepath.Join(a.board.RootPath, targetDir, filename)

		if err := os.Rename(card.FilePath, newPath); err != nil {
			slog.Error("failed to move card file", "from", card.FilePath, "to", newPath, "error", err)
			return fmt.Errorf("moving card %d: %w", card.Metadata.ID, err)
		}

		card.FilePath = newPath
		card.ListName = targetDir

		if err := daedalus.WriteCardFile(card.FilePath, card.Metadata, body); err != nil {
			slog.Error("failed to write card after move-all", "path", card.FilePath, "error", err)
			return fmt.Errorf("writing card %d: %w", card.Metadata.ID, err)
		}

		a.board.Lists[targetDir] = append(a.board.Lists[targetDir], card)
	}

	a.board.Lists[sourceDir] = []daedalus.KanbanCard{}
	slog.Info("moved all cards", "from", sourceDir, "to", targetDir, "count", len(srcCards))
	return nil
}

// DeleteAllCards removes every card file in a list directory while keeping the list itself intact.
func (a *App) DeleteAllCards(listDir string) error {
	if a.board == nil {
		return fmt.Errorf("board not loaded")
	}

	cards, ok := a.board.Lists[listDir]
	if !ok {
		return fmt.Errorf("list not found: %s", listDir)
	}

	if isListLocked(a.board.Config, listDir) {
		return fmt.Errorf("list is locked")
	}

	if len(cards) == 0 {
		return nil
	}

	a.pauseWatcher()

	var totalBytes int64
	for _, card := range cards {
		totalBytes += getFileSize(card.FilePath)
		if err := os.Remove(card.FilePath); err != nil {
			slog.Error("failed to remove card file", "path", card.FilePath, "error", err)
			return fmt.Errorf("removing card %d: %w", card.Metadata.ID, err)
		}
	}

	a.board.Lists[listDir] = []daedalus.KanbanCard{}
	a.board.TotalFileBytes -= totalBytes

	slog.Info("deleted all cards in list", "list", listDir, "count", len(cards), "bytesFreed", totalBytes)
	return nil
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
