package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"daedalus/pkg/daedalus"
)

func jsonOut(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func cmdBoard(boardPath string) error {
	state, err := daedalus.ScanBoard(boardPath)
	if err != nil {
		return err
	}

	totalCards := 0
	for _, cards := range state.Lists {
		totalCards += len(cards)
	}

	return jsonOut(map[string]any{
		"title": state.Config.Title,
		"lists": len(state.Config.Lists),
		"cards": totalCards,
		"path":  state.RootPath,
	})
}

func cmdLists(boardPath string) error {
	state, err := daedalus.ScanBoard(boardPath)
	if err != nil {
		return err
	}

	type listInfo struct {
		Dir   string `json:"dir"`
		Title string `json:"title"`
		Cards int    `json:"cards"`
	}

	result := []listInfo{}
	for _, entry := range state.Config.Lists {
		cards := state.Lists[entry.Dir]
		title := entry.Title
		if title == "" {
			title = entry.Dir
		}
		result = append(result, listInfo{
			Dir:   entry.Dir,
			Title: title,
			Cards: len(cards),
		})
	}

	return jsonOut(result)
}

func cmdCards(boardPath string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cards <list-name>")
	}
	listName := args[0]

	state, err := daedalus.ScanBoard(boardPath)
	if err != nil {
		return err
	}

	cards, ok := state.Lists[listName]
	if !ok {
		return fmt.Errorf("list %q not found", listName)
	}

	type cardInfo struct {
		ID        int      `json:"id"`
		Title     string   `json:"title"`
		Labels    []string `json:"labels"`
		ListOrder float64  `json:"listOrder"`
	}

	result := []cardInfo{}
	for _, c := range cards {
		labels := c.Metadata.Labels
		if labels == nil {
			labels = []string{}
		}
		result = append(result, cardInfo{
			ID:        c.Metadata.ID,
			Title:     c.Metadata.Title,
			Labels:    labels,
			ListOrder: c.Metadata.ListOrder,
		})
	}

	return jsonOut(result)
}

func cmdCardGet(boardPath string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: card-get <card-id>")
	}
	cardID, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid card ID %q: %w", args[0], err)
	}

	state, err := daedalus.ScanBoard(boardPath)
	if err != nil {
		return err
	}

	for _, cards := range state.Lists {
		for _, c := range cards {
			if c.Metadata.ID == cardID {
				body, err := daedalus.ReadCardContent(c.FilePath)
				if err != nil {
					return fmt.Errorf("reading card content: %w", err)
				}
				return jsonOut(map[string]any{
					"id":       c.Metadata.ID,
					"title":    c.Metadata.Title,
					"list":     c.ListName,
					"metadata": c.Metadata,
					"body":     body,
				})
			}
		}
	}

	return fmt.Errorf("card with ID %d not found", cardID)
}

func cmdCardCreate(boardPath string, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: card-create <list-name> <title>")
	}
	listName := args[0]
	title := args[1]

	state, err := daedalus.ScanBoard(boardPath)
	if err != nil {
		return err
	}

	cards, ok := state.Lists[listName]
	if !ok {
		return fmt.Errorf("list %q not found", listName)
	}

	meta, filePath, _, err := daedalus.CreateCardOnDisk(boardPath, listName, title, "", "bottom", cards, state.MaxID)
	if err != nil {
		return err
	}

	return jsonOut(map[string]any{
		"id":    meta.ID,
		"title": meta.Title,
		"list":  listName,
		"path":  filePath,
	})
}

func cmdCardDelete(boardPath string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: card-delete <card-id>")
	}
	cardID, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid card ID %q: %w", args[0], err)
	}

	state, err := daedalus.ScanBoard(boardPath)
	if err != nil {
		return err
	}

	for _, cards := range state.Lists {
		for _, c := range cards {
			if c.Metadata.ID == cardID {
				return os.Remove(c.FilePath)
			}
		}
	}

	return fmt.Errorf("card with ID %d not found", cardID)
}

func cmdListCreate(boardPath string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: list-create <name>")
	}

	name, err := daedalus.ValidateListName(args[0])
	if err != nil {
		return err
	}

	state, err := daedalus.ScanBoard(boardPath)
	if err != nil {
		return err
	}

	if _, exists := state.Lists[name]; exists {
		return fmt.Errorf("list %q already exists", name)
	}
	// Also check config entries for lists that exist but have no cards
	if daedalus.FindListEntry(state.Config.Lists, name) >= 0 {
		return fmt.Errorf("list %q already exists", name)
	}

	if err := daedalus.CreateListOnDisk(boardPath, name, state.Config); err != nil {
		return err
	}

	return jsonOut(map[string]any{
		"dir": name,
	})
}

func cmdListDelete(boardPath string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: list-delete <name>")
	}
	name := args[0]

	if strings.Contains(name, "/") || strings.Contains(name, "\\") || strings.Contains(name, "..") {
		return fmt.Errorf("invalid list name")
	}

	state, err := daedalus.ScanBoard(boardPath)
	if err != nil {
		return err
	}

	_, inLists := state.Lists[name]
	inConfig := daedalus.FindListEntry(state.Config.Lists, name) >= 0
	if !inLists && !inConfig {
		return fmt.Errorf("list %q not found", name)
	}

	return daedalus.DeleteListOnDisk(boardPath, name, state.Config)
}

func cmdExportJSON(boardPath string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: export-json <output-path>")
	}
	outputPath := args[0]

	state, err := daedalus.ScanBoard(boardPath)
	if err != nil {
		return err
	}

	iconsDir := filepath.Join(boardPath, "_assets", "icons")
	board, err := daedalus.BuildExportBoard(state, iconsDir)
	if err != nil {
		return err
	}

	return daedalus.WriteExportJSON(board, outputPath)
}

func cmdExportZip(boardPath string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: export-zip <output-path>")
	}
	outputPath := args[0]

	state, err := daedalus.ScanBoard(boardPath)
	if err != nil {
		return err
	}

	iconsDir := filepath.Join(boardPath, "_assets", "icons")
	return daedalus.WriteExportZip(boardPath, state, iconsDir, outputPath)
}
