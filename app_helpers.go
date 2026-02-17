package main

import (
	"daedalus/pkg/daedalus"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
)

// validatePath resolves a file path to absolute and verifies it is within the board root.
func (a *App) validatePath(filePath string) (string, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		slog.Warn("path resolution failed", "path", filePath, "error", err)
		return "", fmt.Errorf("invalid path")
	}
	absRoot, err := filepath.Abs(a.board.RootPath)
	if err != nil {
		slog.Error("board root path resolution failed", "root", a.board.RootPath, "error", err)
		return "", fmt.Errorf("invalid root path")
	}
	if !strings.HasPrefix(absPath, absRoot+string(filepath.Separator)) {
		slog.Warn("path traversal rejected", "path", absPath, "root", absRoot)
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
	if err := daedalus.PlatformOpen(absPath); err != nil {
		slog.Error("failed to open file externally", "path", absPath, "error", err)
		return err
	}
	slog.Debug("opened file externally", "path", absPath)
	return nil
}

// OpenURI opens a URI with the system default handler.
func (a *App) OpenURI(uri string) error {
	if uri == "" {
		return fmt.Errorf("empty URI")
	}
	if err := daedalus.PlatformOpen(uri); err != nil {
		slog.Error("failed to open URI", "uri", uri, "error", err)
		return err
	}
	slog.Debug("opened URI", "uri", uri)
	return nil
}
