package main

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/vba-excel/sharepoint-gui/internal/spgui"
)

type App struct {
	ctx   context.Context
	SP    *spgui.SPGUI
}

func NewApp() *App {
	return &App{
		SP: spgui.NewSPGUI(spgui.Config{
			ConfigPath: "private.json",
			// SiteURL: "", // podes forçar override aqui se quiseres
		}),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	runtime.LogInfo(ctx, "App started")
}
