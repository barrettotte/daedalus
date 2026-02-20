package daedalus

import (
	"archive/zip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Export structs -- used for JSON marshaling when exporting a board.

// ExportCard is a single card with its full body content.
type ExportCard struct {
	ID       int          `json:"id"`
	Title    string       `json:"title"`
	Metadata CardMetadata `json:"metadata"`
	Body     string       `json:"body"`
}

// ExportList is a list directory with its cards.
type ExportList struct {
	Dir   string       `json:"dir"`
	Title string       `json:"title"`
	Cards []ExportCard `json:"cards"`
}

// ExportIcon is a custom icon with its content (raw SVG or base64 data URI for PNG).
type ExportIcon struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// ExportBoard is the top-level export structure for a board.
type ExportBoard struct {
	Title      string       `json:"title"`
	ExportedAt time.Time    `json:"exportedAt"`
	Config     *BoardConfig `json:"config"`
	Lists      []ExportList `json:"lists"`
	Icons      []ExportIcon `json:"icons"`
}

// BuildExportBoard walks the board state and builds an ExportBoard with full card bodies and icons.
// iconsDir is the path to the board's _assets/icons/ directory.
func BuildExportBoard(state *BoardState, iconsDir string) (ExportBoard, error) {
	board := ExportBoard{
		Title:      state.Config.Title,
		ExportedAt: time.Now(),
		Config:     state.Config,
	}

	// Build ordered list of lists from config (preserves display order).
	for _, entry := range state.Config.Lists {
		cards := state.Lists[entry.Dir]
		el := ExportList{
			Dir:   entry.Dir,
			Title: entry.Title,
		}
		if el.Title == "" {
			el.Title = entry.Dir
		}
		for _, card := range cards {
			body, err := ReadCardContent(card.FilePath)
			if err != nil {
				slog.Warn("export: failed to read card body", "path", card.FilePath, "error", err)
				body = ""
			}
			el.Cards = append(el.Cards, ExportCard{
				ID:       card.Metadata.ID,
				Title:    card.Metadata.Title,
				Metadata: card.Metadata,
				Body:     body,
			})
		}
		board.Lists = append(board.Lists, el)
	}

	// Collect icons from the icons directory.
	board.Icons = readExportIcons(iconsDir)

	return board, nil
}

// readExportIcons reads icon files from iconsDir and returns them as ExportIcon slices.
// Returns nil if the directory doesn't exist.
func readExportIcons(iconsDir string) []ExportIcon {
	entries, err := os.ReadDir(iconsDir)
	if err != nil {
		return nil
	}

	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && IsIconExt(entry.Name()) {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)

	var icons []ExportIcon
	for _, name := range names {
		content, err := readIconContent(filepath.Join(iconsDir, name))
		if err != nil {
			slog.Warn("export: failed to read icon", "name", name, "error", err)
			continue
		}
		icons = append(icons, ExportIcon{Name: name, Content: content})
	}
	return icons
}

// readIconContent reads an icon file and returns its content as a string.
// SVG files are returned as raw text; PNG files are returned as base64 data URIs.
func readIconContent(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".svg":
		return string(data), nil
	case ".png":
		encoded := base64.StdEncoding.EncodeToString(data)
		return "data:image/png;base64," + encoded, nil
	default:
		return "", fmt.Errorf("unsupported icon type: %s", ext)
	}
}

// WriteExportJSON marshals an ExportBoard to indented JSON and writes it to a file.
func WriteExportJSON(board ExportBoard, path string) error {
	data, err := json.MarshalIndent(board, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling export: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing export file: %w", err)
	}
	return nil
}

// WriteExportZip creates a zip archive containing board.yaml, all card files, and icons.
func WriteExportZip(rootPath string, state *BoardState, iconsDir string, path string) error {
	outFile, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating zip file: %w", err)
	}

	zw := zip.NewWriter(outFile)

	// Add board.yaml.
	if err := addFileToZip(zw, filepath.Join(rootPath, "board.yaml"), "board.yaml"); err != nil {
		zw.Close()
		outFile.Close()
		return fmt.Errorf("adding board.yaml: %w", err)
	}

	// Add card files from each list directory.
	for _, entry := range state.Config.Lists {
		for _, card := range state.Lists[entry.Dir] {
			relPath := entry.Dir + "/" + filepath.Base(card.FilePath)
			if err := addFileToZip(zw, card.FilePath, relPath); err != nil {
				slog.Warn("export: failed to add card to zip", "path", card.FilePath, "error", err)
			}
		}
	}

	// Add icons from iconsDir if present.
	if entries, err := os.ReadDir(iconsDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() || !IsIconExt(entry.Name()) {
				continue
			}
			srcPath := filepath.Join(iconsDir, entry.Name())
			relPath := "_assets/icons/" + entry.Name()
			if err := addFileToZip(zw, srcPath, relPath); err != nil {
				slog.Warn("export: failed to add icon to zip", "name", entry.Name(), "error", err)
			}
		}
	}

	if err := zw.Close(); err != nil {
		outFile.Close()
		return fmt.Errorf("finalizing zip: %w", err)
	}
	if err := outFile.Close(); err != nil {
		return fmt.Errorf("closing zip file: %w", err)
	}
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
