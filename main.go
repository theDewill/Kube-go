package main

import (
	"embed"
	"fmt"
	"kube-go/kfiles"
	knet "kube-go/kubenet"
	RFG "kube-go/refrigirator"
	security "kube-go/security"
	"log"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {

	//FILE PATHS
	platformPath, err := kfiles.GetPlatformSpecificPath()
	dbPath := filepath.Join(platformPath, "fileidx.sqlite")
	modelsPath := filepath.Join(platformPath, "models")

	Registry, NRegErr := knet.NewNodeRegistry("/Users/nominsendinu/DEWILL/CODE/Projects/kube-go/kfiles")
	if NRegErr != nil {
		println("Error:", NRegErr.Error())
	}

	fileBrowser, fbErr := kfiles.LaunchFileBrowser(Registry)
	RFG := RFG.CreateRefrigirator("1.0", "no index", fileBrowser)
	if fbErr != nil {
		log.Fatalf("Error initializing File Browser: %v", fbErr)
	}
	fmt.Printf("File Browser initialized. KubeLoads directory: %s\n", fileBrowser.KubeLoadsDir, fileBrowser.KubeRestsDir)

	facialSystem, fsErr := security.NewFacialSystem(dbPath, modelsPath)
	if fsErr != nil {
		log.Printf("Warning: Failed to initialize facial recognition system: %v", fsErr)
		// We don't fatal here because the app can still function without facial recognition
	} else {
		fmt.Println("Facial recognition system initialized successfully")
		fmt.Printf("Using models from: %s\n", modelsPath)
		fmt.Printf("User data stored in SQLite database: %s\n", dbPath)

		// Get the number of registered users
		userCount, err := facialSystem.GetRegisteredUserCount()
		if err != nil {
			log.Printf("Warning: Failed to get registered user count: %v", err)
		} else {
			fmt.Printf("Number of registered users: %d\n", userCount)
		}
	}
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
			facialSystem,
		},
		// OnShutdown: func(ctx) {

		// 	if facialSystem != nil {
		// 		facialSystem.Close()
		// 	}
		// },
	})

	if err != nil {
		println("Error Launching the UI:", err.Error())
	}

}
