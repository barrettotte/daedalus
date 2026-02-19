package main

import (
	"archive/zip"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExportJSON(t *testing.T) {
	app, _ := setupTestBoardMulti(t)

	// Create a card so there's fresh data to export.
	card, err := app.CreateCard("open", "Test Card", "Some body content", "bottom")
	if err != nil {
		t.Fatalf("failed to create card: %v", err)
	}

	outPath := filepath.Join(t.TempDir(), "export.json")
	if err := app.ExportJSON(outPath); err != nil {
		t.Fatalf("ExportJSON failed: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read export file: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("export is not valid JSON: %v", err)
	}

	if result["title"] == nil {
		t.Error("export missing title field")
	}
	if result["exportedAt"] == nil {
		t.Error("export missing exportedAt field")
	}

	lists, ok := result["lists"].([]any)
	if !ok || len(lists) == 0 {
		t.Fatal("export missing lists")
	}

	// Verify the card's body was included.
	found := false
	for _, l := range lists {
		lm := l.(map[string]any)
		cards, ok := lm["cards"].([]any)
		if !ok {
			continue
		}
		for _, c := range cards {
			cm := c.(map[string]any)
			if int(cm["id"].(float64)) == card.Metadata.ID {
				if cm["body"] == nil || cm["body"] == "" {
					t.Error("card body is empty in export")
				}
				found = true
			}
		}
	}
	if !found {
		t.Error("created card not found in export")
	}
}

func TestExportJSON_BoardNotLoaded(t *testing.T) {
	app := NewApp()
	outPath := filepath.Join(t.TempDir(), "export.json")
	err := app.ExportJSON(outPath)
	if err == nil {
		t.Fatal("expected error when board not loaded")
	}
	if err.Error() != "board not loaded" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestExportZip(t *testing.T) {
	app, _ := setupTestBoardMulti(t)

	// Create a card so there's data to export.
	_, err := app.CreateCard("open", "Zip Card", "Zip body", "bottom")
	if err != nil {
		t.Fatalf("failed to create card: %v", err)
	}

	outPath := filepath.Join(t.TempDir(), "export.zip")
	if err := app.ExportZip(outPath); err != nil {
		t.Fatalf("ExportZip failed: %v", err)
	}

	// Open the zip and verify contents.
	zr, err := zip.OpenReader(outPath)
	if err != nil {
		t.Fatalf("failed to open zip: %v", err)
	}
	defer zr.Close()

	fileNames := make(map[string]bool)
	for _, f := range zr.File {
		fileNames[f.Name] = true
	}

	if !fileNames["board.yaml"] {
		t.Error("zip missing board.yaml")
	}

	foundCard := false
	for name := range fileNames {
		if strings.HasPrefix(name, "open/") && strings.HasSuffix(name, ".md") {
			foundCard = true
		}
	}
	if !foundCard {
		t.Error("zip missing card files in open/ directory")
	}
}

func TestExportZip_BoardNotLoaded(t *testing.T) {
	app := NewApp()
	outPath := filepath.Join(t.TempDir(), "export.zip")
	err := app.ExportZip(outPath)
	if err == nil {
		t.Fatal("expected error when board not loaded")
	}
}
