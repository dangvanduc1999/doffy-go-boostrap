package main

import (
	"app/libs/core"
	"app/libs/plugins/logger"
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

// MyServicePlugin implements the Plugin interface
type MyServicePlugin struct {
	core.BasePlugin
}

// NewMyServicePlugin creates a new MyService plugin
func NewMyServicePlugin() *MyServicePlugin {
	return &MyServicePlugin{}
}

// Name returns the plugin name
func (p *MyServicePlugin) Name() string {
	return "my-service"
}

// Version returns the plugin version
func (p *MyServicePlugin) Version() string {
	return "1.0.0"
}

// Register registers the service with the DI container
func (p *MyServicePlugin) Register(container core.DIContainer) error {
	return container.RegisterSingleton("myService", func(c core.DIContainer) (interface{}, error) {
		return &MyService{}, nil
	})
}

// Hooks returns the lifecycle hooks
func (p *MyServicePlugin) Hooks() []core.LifecycleHook {
	return []core.LifecycleHook{}
}

// MyService represents a simple service
type MyService struct{}

// DoSomething does something
func (s *MyService) DoSomething() string {
	return "Hello from MyService!"
}

func main() {
	config := &core.AppOptions{
		Name:       "MyService",
		Mode:       "debug",
		UseLogger:  true,
		Port:       8080,
		ConfigPath: "config.json",
		Cors: &core.CorsOptions{
			AllowOrigins:     []string{"*"},
			AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
			AllowCredentials: false,
			MaxAge:           86400,
		},
	}
	
	app := core.CreateDoffApp(config)
	
	// Register plugins
	app.RegisterPlugin(logger.NewLoggerPlugin())
	app.RegisterPlugin(NewMyServicePlugin())
	
	// Get router and register routes
	router := app.(*core.DoffApp).GetRouter()
	
	// Health check endpoint
	router.GET("/health", func(c *gin.Context, container core.DIContainer) {
		c.JSON(200, gin.H{
			"status": "ok",
			"service": "MyService",
			"timestamp": time.Now().UTC(),
		})
	})
	
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