package daedalus

import (
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// PlatformOpen opens a file or URI with the system default handler.
func PlatformOpen(target string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", target).Start()
	case "windows":
		return exec.Command("cmd", "/c", "start", "", target).Start()
	default:
		return exec.Command("xdg-open", target).Start()
	}
}

// GetFileSize returns the size of a file in bytes, or 0 if the file cannot be stat'd.
func GetFileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		slog.Warn("failed to stat file for size", "path", path, "error", err)
		return 0
	}
	return info.Size()
}

// TruncatePreview returns a body preview truncated to PreviewMaxLen characters.
func TruncatePreview(body string) string {
	if len(body) > PreviewMaxLen {
		return body[:PreviewMaxLen]
	}
	return body
}

// IsIconExt returns true if the file extension is a supported icon type (.svg or .png).
func IsIconExt(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".svg" || ext == ".png"
}

// IsListLocked returns true if the given list directory is marked as locked in the config.
func IsListLocked(config *BoardConfig, dir string) bool {
	idx := FindListEntry(config.Lists, dir)
	return idx >= 0 && config.Lists[idx].Locked
}
