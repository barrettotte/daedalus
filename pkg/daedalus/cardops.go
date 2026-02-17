package daedalus

import (
	"sort"
	"strconv"
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
