package main

import (
	"context"
	"fmt"
	knet "kube-go/kubenet"
)

// App struct
type App struct {
	ctx        context.Context
	ctx_cancel context.CancelFunc
	register   *knet.NodeRegistry
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

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}
