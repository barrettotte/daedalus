package main

import (
	"embed"
	"log/slog"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Set up structured logging. DAEDALUS_DEBUG=1 enables debug-level output.
	level := slog.LevelInfo
	if os.Getenv("DAEDALUS_DEBUG") == "1" {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))

	app := NewApp()
	err := wails.Run(&options.App{
		Title:     "daedalus",
		Width:     1024,
		Height:    768,
		MinWidth:  800,
		MinHeight: 600,
		MaxWidth:  3840,
		MaxHeight: 2160,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		slog.Error("wails runtime failed", "error", err)
	}
}
