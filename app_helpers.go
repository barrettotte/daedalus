package main

import (
	"daedalus/pkg/daedalus"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// getFileSize returns the size of a file in bytes, or 0 if the file cannot be stat'd.
func getFileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		slog.Warn("failed to stat file for size", "path", path, "error", err)
		return 0
	}
	return info.Size()
}

// truncatePreview returns a body preview truncated to PreviewMaxLen characters.
func truncatePreview(body string) string {
	if len(body) > daedalus.PreviewMaxLen {
		return body[:daedalus.PreviewMaxLen]
	}
	return body
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
	if !strings.HasPrefix(absPath, absRoot+string(filepath.Separator)) {
		slog.Warn("path traversal rejected", "path", absPath, "root", absRoot)
		return "", fmt.Errorf("path outside board directory")
	}
	return absPath, nil
}

// platformOpen opens a target (file path or URI) with the system default handler.
func platformOpen(target string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", target)
	case "darwin":
		cmd = exec.Command("open", target)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", target)
	default:
		return fmt.Errorf("unsupported platform")
	}
	return cmd.Start()
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
	if err := platformOpen(absPath); err != nil {
		slog.Error("failed to open file externally", "path", absPath, "error", err)
		return err
	}
	slog.Debug("opened file externally", "path", absPath)
	return nil
}

// OpenURI opens an arbitrary URI with the system handler (xdg-open, open, start).
func (a *App) OpenURI(uri string) error {
	if uri == "" {
		return fmt.Errorf("empty URI")
	}
	if err := platformOpen(uri); err != nil {
		slog.Error("failed to open URI", "uri", uri, "error", err)
		return err
	}
	slog.Debug("opened URI", "uri", uri)
	return nil
}
