package main

import (
	"archive/zip"
	"daedalus/pkg/daedalus"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// Export structs -- used only for JSON marshaling.

type exportCard struct {
	ID       int                   `json:"id"`
	Title    string                `json:"title"`
	Metadata daedalus.CardMetadata `json:"metadata"`
	Body     string                `json:"body"`
}

type exportList struct {
	Dir   string       `json:"dir"`
	Title string       `json:"title"`
	Cards []exportCard `json:"cards"`
}

type exportIcon struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

type exportBoard struct {
	Title      string                `json:"title"`
	ExportedAt time.Time             `json:"exportedAt"`
	Config     *daedalus.BoardConfig `json:"config"`
	Lists      []exportList          `json:"lists"`
	Icons      []exportIcon          `json:"icons"`
}

// ExportJSON writes the full board (config, cards with bodies, icons) to a JSON file.
func (a *App) ExportJSON(path string) error {
	if a.board == nil {
		return fmt.Errorf("board not loaded")
	}

	board := exportBoard{
		Title:      a.board.Config.Title,
		ExportedAt: time.Now(),
		Config:     a.board.Config,
	}

	// Build ordered list of lists from config (preserves display order).
	for _, entry := range a.board.Config.Lists {
		cards := a.board.Lists[entry.Dir]
		el := exportList{
			Dir:   entry.Dir,
			Title: entry.Title,
		}
		for _, card := range cards {
			body, err := daedalus.ReadCardContent(card.FilePath)
			if err != nil {
				slog.Warn("export: failed to read card body", "path", card.FilePath, "error", err)
				body = ""
			}
			el.Cards = append(el.Cards, exportCard{
				ID:       card.Metadata.ID,
				Title:    card.Metadata.Title,
				Metadata: card.Metadata,
				Body:     body,
			})
		}
		board.Lists = append(board.Lists, el)
	}

	// Collect icons.
	iconNames, err := a.ListIcons()
	if err == nil {
		for _, name := range iconNames {
			content, err := a.GetIconContent(name)
			if err != nil {
				slog.Warn("export: failed to read icon", "name", name, "error", err)
				continue
			}
			board.Icons = append(board.Icons, exportIcon{Name: name, Content: content})
		}
	}

	data, err := json.MarshalIndent(board, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling export: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing export file: %w", err)
	}

	slog.Info("board exported as JSON", "path", path)
	return nil
}

// ExportZip writes the full board directory (board.yaml, cards, icons) to a zip archive.
func (a *App) ExportZip(path string) error {
	if a.board == nil {
		return fmt.Errorf("board not loaded")
	}

	outFile, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating zip file: %w", err)
	}
	defer outFile.Close()

	zw := zip.NewWriter(outFile)
	defer zw.Close()

	root := a.board.RootPath

	// Add board.yaml.
	if err := addFileToZip(zw, filepath.Join(root, "board.yaml"), "board.yaml"); err != nil {
		return fmt.Errorf("adding board.yaml: %w", err)
	}

	// Add card files from each list directory.
	for _, entry := range a.board.Config.Lists {
		cards := a.board.Lists[entry.Dir]
		for _, card := range cards {
			relPath := entry.Dir + "/" + filepath.Base(card.FilePath)
			if err := addFileToZip(zw, card.FilePath, relPath); err != nil {
				slog.Warn("export: failed to add card to zip", "path", card.FilePath, "error", err)
			}
		}
	}

	// Add icons.
	iconsDir := a.iconsDir()
	iconNames, err := a.ListIcons()
	if err == nil {
		for _, name := range iconNames {
			srcPath := filepath.Join(iconsDir, name)
			relPath := "_assets/icons/" + name
			if err := addFileToZip(zw, srcPath, relPath); err != nil {
				slog.Warn("export: failed to add icon to zip", "name", name, "error", err)
			}
		}
	}

	slog.Info("board exported as zip", "path", path)
	return nil
}

// addFileToZip reads a file from disk and writes it into a zip archive at the given path.
func addFileToZip(zw *zip.Writer, srcPath string, zipPath string) error {
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}
	w, err := zw.Create(zipPath)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}
