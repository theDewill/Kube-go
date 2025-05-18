package main

import (
	"context"
	"fmt"
	"kube-go/kfiles"
	"kube-go/kubenet"
	knet "kube-go/kubenet"
	"log"
	"os"
	"path/filepath"
)

// App struct
type App struct {
	ctx         context.Context
	ctx_cancel  context.CancelFunc
	register    *knet.NodeRegistry
	fileBrowser *kfiles.FileBrowser
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	Done := make(chan struct{})
	go func() {
		<-ctx.Done()
		close(Done)
	}()
	discover_err := a.register.Start(Done)
	if discover_err != nil {
		println("Reg Discovery Error", discover_err.Error())
	}
}

func (a *App) shutdown(ctx context.Context) {
	// Set node status to offline on shutdown
	if a.register != nil {
		if err := a.register.SetNodeStatus(kubenet.StatusOffline); err != nil {
			log.Printf("Failed to set node status to offline: %v", err)
		}
	}
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

// getApplicationDirectory returns the platform-specific application directory
func getApplicationDirectory() (string, error) {
	var baseDir string

	// Get user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	// Set paths based on OS
	switch os.Getenv("GOOS") {
	case "windows":
		baseDir = filepath.Join(os.Getenv("LOCALAPPDATA"), "Kube")
	case "darwin":
		baseDir = filepath.Join(homeDir, "Library", "Application Support", "Kube")
	default: // Linux and other Unix-like systems
		xdgDataHome := os.Getenv("XDG_DATA_HOME")
		if xdgDataHome != "" {
			baseDir = filepath.Join(xdgDataHome, "Kube")
		} else {
			baseDir = filepath.Join(homeDir, ".local", "share", "Kube")
		}
	}

	// Create the directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create application directory: %w", err)
	}

	return baseDir, nil
}

// GetNodeRegistry exposes the registry to the frontend
func (a *App) GetNodeRegistry() *kubenet.NodeRegistry {
	return a.register
}

// GetFileBrowser exposes the file browser to the frontend
func (a *App) GetFileBrowser() *kfiles.FileBrowser {
	return a.fileBrowser
}
