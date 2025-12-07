package main

import (
	"app/libs/core"
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

// DatabasePlugin demonstrates async provider for DB connection
type DatabasePlugin struct {
	core.BasePlugin
}

// NewDatabasePlugin creates a new database plugin
func NewDatabasePlugin() *DatabasePlugin {
	return &DatabasePlugin{}
}

// Name returns the plugin name
func (p *DatabasePlugin) Name() string {
	return "database"
}

// Version returns the plugin version
func (p *DatabasePlugin) Version() string {
	return "1.0.0"
}

// Module returns the module definition for the database plugin
func (p *DatabasePlugin) Module() *core.Module {
	return core.NewModule("database", "1.0.0").
		WithDescription("Database connection plugin with async initialization").
		WithProviders(
			// Async provider for database connection
			core.NewAsyncProvider("db", func(container core.DIContainer, ctx context.Context) (interface{}, error) {
				// Simulate database connection setup
				connStr := "postgres://user:password@localhost/dbname?sslmode=disable"

				// In a real app, you'd get connection string from config
				if val := os.Getenv("DATABASE_URL"); val != "" {
					connStr = val
				}

				// Connect with timeout
				db, err := sql.Open("postgres", connStr)
				if err != nil {
					return nil, fmt.Errorf("failed to open database: %w", err)
				}

				// Ping to verify connection
				if err := db.PingContext(ctx); err != nil {
					db.Close()
					return nil, fmt.Errorf("failed to ping database: %w", err)
				}

				log.Println("Database connection established successfully")
				return db, nil
			}, core.Singleton, 10*time.Second),
		).
		WithExports("db")
}

// Register registers the database provider
func (p *DatabasePlugin) Register(container core.DIContainer) error {
	module := p.Module()

	// Register all providers from module
	for _, provider := range module.Providers {
		if err := container.RegisterProvider(provider); err != nil {
			return err
		}
	}

	return nil
}

// Hooks returns lifecycle hooks
func (p *DatabasePlugin) Hooks() []core.LifecycleHook {
	return []core.LifecycleHook{
		core.NewOnRequestHook(func(c *gin.Context) {
			// Example: Set request ID in context
			requestID := c.GetHeader("X-Request-ID")
			if requestID == "" {
				requestID = fmt.Sprintf("req-%d", time.Now().UnixNano())
			}
			c.Set("requestID", requestID)
		}),
	}
}

// Shutdown closes database connections
func (p *DatabasePlugin) Shutdown() error {
	// In a real implementation, you'd close the DB connection here
	// This is just a placeholder
	log.Println("Database plugin shutdown complete")
	return nil
}

// Routes registers any database-related routes
func (p *DatabasePlugin) Routes(router *gin.Engine) error {
	// Example health check endpoint
	router.GET("/health/db", func(c *gin.Context) {
		container := c.MustGet("container").(core.DIContainer)

		db, err := container.Resolve("db")
		if err != nil {
			c.JSON(500, gin.H{"status": "error", "message": err.Error()})
			return
		}

		// Ping database
		sqlDB := db.(*sql.DB)
		if err := sqlDB.Ping(); err != nil {
			c.JSON(500, gin.H{"status": "unhealthy", "error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"status": "healthy"})
	})

	return nil
}

func main() {
	config := &core.AppOptions{
		Name:      "Database Example",
		Mode:      "debug",
		UseLogger: true,
		Port:      8080,
		Cors: &core.CorsOptions{
			AllowOrigins:     []string{"*"},
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
			AllowCredentials: false,
			MaxAge:           86400,
		},
	}

	app := core.CreateDoffApp(config)

	// Register plugins
	app.RegisterPlugin(NewDatabasePlugin())

	// Start server in a goroutine
	go func() {
		if err := app.Listen(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	log.Println("Server started on :8080")
	log.Println("Health check: http://localhost:8080/health/db")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := app.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}