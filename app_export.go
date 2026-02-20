package main

import (
	"daedalus/pkg/daedalus"
	"log/slog"
)

// ExportJSON writes the full board (config, cards with bodies, icons) to a JSON file.
func (a *App) ExportJSON(path string) error {
	if _, err := a.requireBoard(); err != nil {
		return err
	}

	board, err := daedalus.BuildExportBoard(a.board, a.iconsDir())
	if err != nil {
		return err
	}
	if err := daedalus.WriteExportJSON(board, path); err != nil {
		return err
	}

	slog.Info("board exported as JSON", "path", path)
	return nil
}

// ExportZip writes the full board directory (board.yaml, cards, icons) to a zip archive.
func (a *App) ExportZip(path string) error {
	if _, err := a.requireBoard(); err != nil {
		return err
	}

	if err := daedalus.WriteExportZip(a.board.RootPath, a.board, a.iconsDir(), path); err != nil {
		return err
	}

	slog.Info("board exported as zip", "path", path)
	return nil
}
