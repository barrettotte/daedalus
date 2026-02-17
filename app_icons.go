package main

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// iconsDir returns the path to the board's icons directory.
func (a *App) iconsDir() string {
	return filepath.Join(a.board.RootPath, "assets", "icons")
}

// isIconExt returns true if the file extension is a supported icon type (.svg or .png).
func isIconExt(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".svg" || ext == ".png"
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

// DeleteIcon removes an icon file from {boardRoot}/assets/icons/.
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

	slog.Info("icon deleted", "name", name)
	return nil
}
