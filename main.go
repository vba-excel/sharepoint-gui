// sharepoint-gui/main.go
package main

import (
	"context"
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	// ⚠️ Ajusta este import ao teu module path se for diferente
	"github.com/vba-excel/sharepoint-gui/internal/spgui"
)

//go:embed all:frontend/dist
var assets embed.FS

// App contém os serviços expostos ao frontend (Wails Bind).
type App struct {
	SP *spgui.SPGUI
}

func NewApp() *App {
	// Defaults seguros; o frontend pode mudar via SetConfig
	sp := spgui.NewSPGUI(spgui.Config{
		ConfigPath:               "private.json",
		SiteURL:                  "",
		GlobalTimeoutSec:         60,
		CleanOutput:              false,
		HTTPIdleConnTimeoutSec:   20, // < timeout do proxy
		HTTPMaxIdleConns:         40,
		HTTPMaxIdleConnsPerHost:  4,
		HTTPDisableKeepAlives:    false,
		HTTPTLSHandshakeTOsec:    10,
	})
	return &App{SP: sp}
}

// Chamado no OnStartup do Wails
func (a *App) startup(ctx context.Context) {
	if a.SP != nil {
		a.SP.SetContext(ctx)
		runtime.LogInfo(ctx, "App started")
	}
}

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:  "sharepoint-gui",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup: func(ctx context.Context) {
			app.startup(ctx)
		},
		OnShutdown: func(ctx context.Context) {
			// Cancela qualquer operação SharePoint em curso (seguro/idempotente)
			if app.SP != nil {
				app.SP.CancelCurrent()
			}
			runtime.LogInfo(ctx, "Shutdown: cancelada eventual operação em curso")
		},
		Bind: []interface{}{
			app,    // (opcional) expõe métodos do “shell” se precisares
			app.SP, // serviço principal exposto ao frontend
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
