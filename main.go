package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// ctx, cancel := context.WithCancel(context.Background())
	// NReg, NRegErr := knet.NewNodeRegistry("/Users/nominsendinu/DEWILL/CODE/Projects/kube-go/kfiles")
	// if NRegErr != nil {
	// 	println("Error:", NRegErr.Error())
	// }
	// fmt.Printf("N-Reg: %+v \n", *NReg)
	// discover_err := NReg.Start(ctx)
	// if discover_err != nil {
	// 	println("Discovery Error", discover_err.Error())
	// }
	// cancel()

	//TMP: stopped with bool temporary

	app := NewApp()
	err := wails.Run(&options.App{
		Title:  "kube-go",
		Width:  1024,
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
		println("Error:", err.Error())
	}

}
