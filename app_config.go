package main

import (
	"daedalus/pkg/daedalus"
	"fmt"
	"log/slog"
	"time"
)

// SaveListConfig updates the config for a single list and persists to board.yaml.
func (a *App) SaveListConfig(dirName string, title string, limit int) error {
	if a.board == nil {
		return fmt.Errorf("board not loaded")
	}
	a.pauseWatcher()

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
	a.pauseWatcher()
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
	a.pauseWatcher()
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
	a.pauseWatcher()
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
	a.pauseWatcher()
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
	a.pauseWatcher()
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
	a.pauseWatcher()
	a.board.Config.DarkMode = &dark
	if err := daedalus.SaveBoardConfig(a.board.RootPath, a.board.Config); err != nil {
		slog.Error("failed to save dark mode", "error", err)
		return err
	}
	slog.Debug("dark mode saved", "dark", dark)
	return nil
}

// SaveMinimalView persists the minimal card view preference to board.yaml.
func (a *App) SaveMinimalView(minimal bool) error {
	if a.board == nil {
		return fmt.Errorf("board not loaded")
	}
	a.pauseWatcher()

	a.board.Config.MinimalView = &minimal
	if err := daedalus.SaveBoardConfig(a.board.RootPath, a.board.Config); err != nil {
		slog.Error("failed to save minimal view", "error", err)
		return err
	}
	slog.Debug("minimal view saved", "minimal", minimal)
	return nil
}

// SaveZoom persists the board zoom level to board.yaml.
func (a *App) SaveZoom(level float64) error {
	if a.board == nil {
		return fmt.Errorf("board not loaded")
	}
	a.pauseWatcher()
	a.board.Config.Zoom = &level
	if err := daedalus.SaveBoardConfig(a.board.RootPath, a.board.Config); err != nil {
		slog.Error("failed to save zoom level", "error", err)
		return err
	}
	slog.Debug("zoom level saved", "level", level)
	return nil
}

// SaveBoardTitle sets the board display title and persists to board.yaml.
func (a *App) SaveBoardTitle(title string) error {
	if a.board == nil {
		return fmt.Errorf("board not loaded")
	}
	a.pauseWatcher()
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
	a.pauseWatcher()

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
