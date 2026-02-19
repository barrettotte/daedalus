package main

import (
	"encoding/base64"
	"path/filepath"
	"strings"
	"testing"
)

// ListIcons should return sorted filenames when icons directory has files.
func TestListIcons_Success(t *testing.T) {
	app, root := setupTestBoard(t)

	iconsDir := filepath.Join(root, "_assets", "icons")
	mustMkdir(t, iconsDir)
	mustWrite(t, filepath.Join(iconsDir, "zebra.svg"), []byte("<svg></svg>"))
	mustWrite(t, filepath.Join(iconsDir, "arrow.svg"), []byte("<svg></svg>"))
	mustWrite(t, filepath.Join(iconsDir, "logo.png"), []byte("PNG"))

	names, err := app.ListIcons()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(names) != 3 {
		t.Fatalf("expected 3 icons, got %d", len(names))
	}
	if names[0] != "arrow.svg" || names[1] != "logo.png" || names[2] != "zebra.svg" {
		t.Errorf("unexpected order: %v", names)
	}
}

// ListIcons should return an empty list when the icons directory doesn't exist.
func TestListIcons_EmptyDir(t *testing.T) {
	app, _ := setupTestBoard(t)

	names, err := app.ListIcons()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 0 {
		t.Errorf("expected 0 icons, got %d", len(names))
	}
}

// GetIconContent should return raw SVG markup for .svg files.
func TestGetIconContent_SVG(t *testing.T) {
	app, root := setupTestBoard(t)

	iconsDir := filepath.Join(root, "_assets", "icons")
	mustMkdir(t, iconsDir)
	svgContent := `<svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="10"/></svg>`
	mustWrite(t, filepath.Join(iconsDir, "test.svg"), []byte(svgContent))

	content, err := app.GetIconContent("test.svg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content != svgContent {
		t.Errorf("got %q, want %q", content, svgContent)
	}
}

// GetIconContent should return a base64 data URI for .png files.
func TestGetIconContent_PNG(t *testing.T) {
	app, root := setupTestBoard(t)

	iconsDir := filepath.Join(root, "_assets", "icons")
	mustMkdir(t, iconsDir)
	pngData := []byte{0x89, 0x50, 0x4E, 0x47} // PNG magic bytes
	mustWrite(t, filepath.Join(iconsDir, "test.png"), pngData)

	content, err := app.GetIconContent("test.png")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedPrefix := "data:image/png;base64,"
	if !strings.HasPrefix(content, expectedPrefix) {
		t.Errorf("expected data URI prefix, got %q", content[:30])
	}

	encoded := strings.TrimPrefix(content, expectedPrefix)
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("failed to decode base64: %v", err)
	}
	if len(decoded) != len(pngData) {
		t.Errorf("decoded length: got %d, want %d", len(decoded), len(pngData))
	}
}

// GetIconContent should reject names with path traversal characters.
func TestGetIconContent_PathTraversal(t *testing.T) {
	app, _ := setupTestBoard(t)

	for _, name := range []string{"../etc/passwd", "foo/bar.svg", "..\\evil.svg"} {
		_, err := app.GetIconContent(name)
		if err == nil {
			t.Errorf("expected error for name %q", name)
		}
		if err != nil && err.Error() != "invalid icon name" {
			t.Errorf("unexpected error for %q: %v", name, err)
		}
	}
}

// GetIconContent should return an error for a nonexistent icon.
func TestGetIconContent_NotFound(t *testing.T) {
	app, root := setupTestBoard(t)

	mustMkdir(t, filepath.Join(root, "_assets", "icons"))

	_, err := app.GetIconContent("nonexistent.svg")
	if err == nil {
		t.Fatal("expected error for nonexistent icon")
	}
	if !strings.Contains(err.Error(), "icon not found") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// SaveCustomIcon should save an SVG file and read it back correctly.
func TestSaveCustomIcon_SVG(t *testing.T) {
	app, _ := setupTestBoard(t)

	svgContent := `<svg viewBox="0 0 24 24"><rect width="24" height="24"/></svg>`
	err := app.SaveCustomIcon("new-icon.svg", svgContent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := app.GetIconContent("new-icon.svg")
	if err != nil {
		t.Fatalf("unexpected error reading back: %v", err)
	}
	if content != svgContent {
		t.Errorf("got %q, want %q", content, svgContent)
	}
}

// SaveCustomIcon should save a PNG file (base64 input) and read it back as a data URI.
func TestSaveCustomIcon_PNG(t *testing.T) {
	app, _ := setupTestBoard(t)

	pngData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	encoded := base64.StdEncoding.EncodeToString(pngData)

	err := app.SaveCustomIcon("logo.png", encoded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := app.GetIconContent("logo.png")
	if err != nil {
		t.Fatalf("unexpected error reading back: %v", err)
	}

	if !strings.HasPrefix(content, "data:image/png;base64,") {
		t.Errorf("expected data URI, got %q", content)
	}
}

// SaveCustomIcon should reject .svg files that don't contain <svg.
func TestSaveCustomIcon_InvalidSVG(t *testing.T) {
	app, _ := setupTestBoard(t)

	err := app.SaveCustomIcon("bad.svg", "<div>not an svg</div>")
	if err == nil {
		t.Fatal("expected error for invalid SVG content")
	}
	if !strings.Contains(err.Error(), "invalid SVG") {
		t.Errorf("unexpected error message: %v", err)
	}
}
