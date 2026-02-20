package main

import (
	"daedalus/pkg/daedalus"
	"fmt"
	"log/slog"
	"path/filepath"
	"runtime"
	"strings"
)

// requireBoard returns the loaded board state or an error if no board is loaded.
func (a *App) requireBoard() (*daedalus.BoardState, error) {
	if a.board == nil {
		return nil, fmt.Errorf("board not loaded")
	}
	return a.board, nil
}

// prepareWrite checks that a board is loaded and pauses the file watcher.
func (a *App) prepareWrite() (*daedalus.BoardState, error) {
	board, err := a.requireBoard()
	if err != nil {
		return nil, err
	}
	a.pauseWatcher()
	return board, nil
}

// validateIconName checks that an icon filename has no path traversal characters.
func validateIconName(name string) error {
	if strings.ContainsAny(name, "/\\") || strings.Contains(name, "..") {
		slog.Warn("rejected invalid icon name", "name", name)
		return fmt.Errorf("invalid icon name")
	}
	return nil
}

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
	prefix := absRoot + string(filepath.Separator)
	// Windows and macOS use case-insensitive filesystems.
	hasPrefix := false
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		hasPrefix = strings.HasPrefix(strings.ToLower(absPath), strings.ToLower(prefix))
	} else {
		hasPrefix = strings.HasPrefix(absPath, prefix)
	}
	if !hasPrefix {
		slog.Warn("path traversal rejected", "path", absPath, "root", absRoot)
		return "", fmt.Errorf("path outside board directory")
	}
	return absPath, nil
}

// OpenFileExternal opens a file in the system default application.
func (a *App) OpenFileExternal(filePath string) error {
	if _, err := a.requireBoard(); err != nil {
		return err
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
