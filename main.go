package main

import (
	"embed"
	"fmt"
	"kube-go/kfiles"
	knet "kube-go/kubenet"
	RFG "kube-go/refrigirator"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {

	RFG := RFG.CreateRefrigirator("1.0", "no index")
	Registry, NRegErr := knet.NewNodeRegistry("/Users/nominsendinu/DEWILL/CODE/Projects/kube-go/kfiles")
	if NRegErr != nil {
		println("Error:", NRegErr.Error())
	}

	fileBrowser, fbErr := kfiles.LaunchFileBrowser()
	if fbErr != nil {
		log.Fatalf("Error initializing File Browser: %v", fbErr)
	}
	fmt.Printf("File Browser initialized. KubeLoads directory: %s\n", fileBrowser.KubeLoadsDir)

	app := NewApp()
	app.register = Registry
	app.fileBrowser = fileBrowser
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
			Registry,
			fileBrowser,
			RFG,
		},
	})

	if err != nil {
		println("Error Launching the UI:", err.Error())
	}

}
