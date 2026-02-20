package daedalus

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ComputeInsertPosition determines list_order and insertion index for a new card.
// Position "bottom" appends, a numeric string inserts at that 0-based index,
// and anything else (including "top") prepends.
func ComputeInsertPosition(cards []KanbanCard, position string) (float64, int) {
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

// InsertSorted inserts a card into a sorted slice at the correct position by ListOrder then ID.
func InsertSorted(cards []KanbanCard, card KanbanCard) []KanbanCard {
	idx := sort.Search(len(cards), func(i int) bool {
		if cards[i].Metadata.ListOrder != card.Metadata.ListOrder {
			return cards[i].Metadata.ListOrder > card.Metadata.ListOrder
		}
		return cards[i].Metadata.ID > card.Metadata.ID
	})
	cards = append(cards, KanbanCard{})
	copy(cards[idx+1:], cards[idx:])
	cards[idx] = card
	return cards
}

// CreateCardOnDisk computes the new card ID, builds metadata, writes the file to disk,
// and returns the metadata, file path, and insertion index. The caller is responsible
// for updating any in-memory state.
func CreateCardOnDisk(
	boardPath, listDir, title, body, position string,
	cards []KanbanCard, maxID int,
) (CardMetadata, string, int, error) {
	newID := maxID + 1

	listOrder, insertIdx := ComputeInsertPosition(cards, position)

	if strings.TrimSpace(title) == "" {
		title = fmt.Sprintf("%d", newID)
	}

	now := time.Now()
	meta := CardMetadata{
		ID:        newID,
		Title:     title,
		Created:   &now,
		Updated:   &now,
		ListOrder: listOrder,
	}

	fullBody := fmt.Sprintf("# %s\n\n%s", title, body)
	filePath := filepath.Join(boardPath, listDir, fmt.Sprintf("%d.md", newID))

	if err := WriteCardFile(filePath, meta, fullBody); err != nil {
		return CardMetadata{}, "", 0, fmt.Errorf("writing card file: %w", err)
	}

	return meta, filePath, insertIdx, nil
}
