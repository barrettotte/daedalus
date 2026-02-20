package main

import (
	"daedalus/pkg/daedalus"
	"fmt"
	"log/slog"
	"time"
)

// SaveLabelsExpanded persists the label collapsed/expanded state to board.yaml.
func (a *App) SaveLabelsExpanded(expanded bool) error {
	if _, err := a.prepareWrite(); err != nil {
		return err
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
	if _, err := a.prepareWrite(); err != nil {
		return err
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
	if _, err := a.prepareWrite(); err != nil {
		return err
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
	if _, err := a.prepareWrite(); err != nil {
		return err
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
	if _, err := a.prepareWrite(); err != nil {
		return err
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
	if _, err := a.prepareWrite(); err != nil {
		return err
	}
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
	if _, err := a.prepareWrite(); err != nil {
		return err
	}

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
	if _, err := a.prepareWrite(); err != nil {
		return err
	}
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
	if _, err := a.prepareWrite(); err != nil {
		return err
	}
	a.board.Config.Title = title

	if err := daedalus.SaveBoardConfig(a.board.RootPath, a.board.Config); err != nil {
		slog.Error("failed to save board title", "error", err)
		return err
	}
	slog.Info("board title saved", "title", title)
	return nil
}

// SaveTemplates replaces the full templates array in board.yaml and persists.
func (a *App) SaveTemplates(templates []daedalus.CardTemplate) error {
	slog.Debug("SaveTemplates called", "count", len(templates))
	for i, t := range templates {
		slog.Debug("template received",
			"index", i,
			"name", t.Name,
			"labels", t.Labels,
			"icon", t.Icon,
			"hasEstimate", t.Estimate != nil,
			"hasCounter", t.Counter != nil,
			"hasChecklist", t.Checklist != nil,
		)
	}

	if _, err := a.prepareWrite(); err != nil {
		return err
	}
	a.board.Config.Templates = templates

	if err := daedalus.SaveBoardConfig(a.board.RootPath, a.board.Config); err != nil {
		slog.Error("failed to save templates", "error", err)
		return err
	}
	slog.Debug("templates saved", "count", len(templates))
	return nil
}
