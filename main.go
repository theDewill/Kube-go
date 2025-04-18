package main

import (
	"context"
	"embed"
	"fmt"
	knet "kube-go/kubenet"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	Registry, NRegErr := knet.NewNodeRegistry("/Users/nominsendinu/DEWILL/CODE/Projects/kube-go/kfiles")
	if NRegErr != nil {
		println("Error:", NRegErr.Error())
	}
	fmt.Printf("N-Reg: %+v \n", *Registry)
	discover_err := Registry.Start(ctx)
	if discover_err != nil {
		println("Discovery Error", discover_err.Error())
	}

	app := NewApp()
	err := wails.Run(&options.App{
		Title:  "Kube",
		Width:  1280,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error Launching the UI:", err.Error())
	}

	cancel()

}
