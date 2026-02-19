package main

import (
	"daedalus/pkg/daedalus"
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// iconsDir returns the path to the board's icons directory.
func (a *App) iconsDir() string {
	return filepath.Join(a.board.RootPath, "_assets", "icons")
}

// ListIcons returns a sorted list of icon filenames from {boardRoot}/_assets/icons/.
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
		if daedalus.IsIconExt(entry.Name()) {
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

	iconPath := filepath.Join(a.board.RootPath, "_assets", "icons", name)
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

// SaveCustomIcon saves an uploaded icon file to {boardRoot}/_assets/icons/.
// For .svg files, content is raw SVG text. For .png files, content is base64-encoded.
func (a *App) SaveCustomIcon(name string, content string) error {
	if a.board == nil {
		return fmt.Errorf("board not loaded")
	}
	a.pauseWatcher()

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
	switch ext {
	case ".svg":
		if !strings.Contains(content, "<svg") {
			return fmt.Errorf("invalid SVG content")
		}
		if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
			slog.Error("failed to write SVG icon", "name", name, "error", err)
			return fmt.Errorf("writing icon: %w", err)
		}
	case ".png":
		data, err := base64.StdEncoding.DecodeString(content)
		if err != nil {
			return fmt.Errorf("invalid base64 content: %w", err)
		}
		if err := os.WriteFile(absPath, data, 0644); err != nil {
			slog.Error("failed to write PNG icon", "name", name, "error", err)
			return fmt.Errorf("writing icon: %w", err)
		}
	default:
		return fmt.Errorf("unsupported icon type: %s", ext)
	}

	slog.Info("custom icon saved", "name", name)
	return nil
}

// DeleteIcon removes an icon file from {boardRoot}/_assets/icons/,
// clears the icon field on any card that references it, and removes
// the display name entry from board.yaml.
func (a *App) DeleteIcon(name string) error {
	if a.board == nil {
		return fmt.Errorf("board not loaded")
	}
	a.pauseWatcher()

	if strings.ContainsAny(name, "/\\") || strings.Contains(name, "..") {
		slog.Warn("rejected invalid icon name", "name", name)
		return fmt.Errorf("invalid icon name")
	}

	iconPath := filepath.Join(a.iconsDir(), name)
	absPath, err := a.validatePath(iconPath)
	if err != nil {
		return err
	}

	if err := os.Remove(absPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("icon not found: %s", name)
		}
		slog.Error("failed to delete icon", "name", name, "error", err)
		return fmt.Errorf("deleting icon: %w", err)
	}

	// Strip icon from any card that references it.
	affected := 0
	for listKey, cards := range a.board.Lists {
		for i, card := range cards {
			if card.Metadata.Icon != name {
				continue
			}

			card.Metadata.Icon = ""
			now := time.Now()
			card.Metadata.Updated = &now

			body, err := daedalus.ReadCardContent(card.FilePath)
			if err != nil {
				slog.Error("failed to read card for icon cleanup", "path", card.FilePath, "error", err)
				continue
			}
			if err := daedalus.WriteCardFile(card.FilePath, card.Metadata, body); err != nil {
				slog.Error("failed to write card for icon cleanup", "path", card.FilePath, "error", err)
				continue
			}

			a.board.Lists[listKey][i] = card
			affected++
		}
	}

	slog.Info("icon deleted", "name", name, "cardsAffected", affected)
	return nil
}
