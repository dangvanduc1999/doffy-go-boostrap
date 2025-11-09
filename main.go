package main

import (
	"app/libs/core"
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	config := &core.AppOptions{
		Name:      "Doffy server",
		Mode:      "debug",
		UseLogger: true,
		Port:      3037,
		Cors: &core.CorsOptions{
			AllowOrigins:     []string{"*"},
			AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
			AllowCredentials: false,
			MaxAge:           86400,
		},
	}
	app := core.CreateDoffApp(config)

	// Start server in a goroutine
	go func() {
		app.Listen()
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	println("Shutting down server...")

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := app.Shutdown(ctx); err != nil {
		println("Server forced to shutdown:", err.Error())
	}

	println("Server exiting")
}
