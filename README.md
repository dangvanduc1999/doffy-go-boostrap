# Doffy Go Framework

A Go server framework inspired by Fastify's plugin pattern and NestJS's dependency injection pattern, built on top of Gin-Gonic.

## Features

- **Plugin System**: Modular architecture with lifecycle hooks
- **Dependency Injection**: Service registration and resolution
- **Lifecycle Hooks**: onRequest, preHandler, onResponse, onError
- **Router Helper**: Easy route registration with DI support
- **Graceful Shutdown**: Clean server shutdown with context timeout
- **CORS Support**: Built-in CORS plugin
- **Logger Support**: Built-in logging with customizable implementation

## Quick Start

```go
package main

import (
    "app/libs/core"
    "app/libs/plugins/logger"
    "context"
    "os"
    "os/signal"
    "syscall"
    "time"
)

func main() {
    config := &core.AppOptions{
        Name:      "My API",
        Mode:      "debug",
        UseLogger: true,
        Port:      8080,
        Cors: &core.CorsOptions{
            AllowOrigins: []string{"*"},
            AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
            AllowHeaders: []string{"Origin", "Content-Type", "Authorization"},
            AllowCredentials: false,
            MaxAge: 86400,
        },
    }
    
    app := core.CreateDoffApp(config)
    
    // Register plugins
    app.RegisterPlugin(logger.NewLoggerPlugin())
    
    // Start server
    go app.Listen()
    
    // Graceful shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    app.Shutdown(ctx)
}
```

## Creating a Plugin

```go
package myplugin

import (
    "app/libs/core"
    "github.com/gin-gonic/gin"
)

// MyPlugin implements the Plugin interface
type MyPlugin struct {
    core.BasePlugin
}

// Name returns the plugin name
func (p *MyPlugin) Name() string {
    return "my-plugin"
}

// Version returns the plugin version
func (p *MyPlugin) Version() string {
    return "1.0.0"
}

// Register registers services with the DI container
func (p *MyPlugin) Register(container core.DIContainer) error {
    return container.RegisterSingleton("myService", func(c core.DIContainer) (interface{}, error) {
        return NewMyService(), nil
    })
}

// Hooks returns lifecycle hooks
func (p *MyPlugin) Hooks() []core.LifecycleHook {
    return []core.LifecycleHook{
        core.NewOnRequestHook(func(c *gin.Context) {
            // Request logic here
        }),
    }
}

// Routes registers plugin routes
func (p *MyPlugin) Routes(router *gin.Engine) error {
    router.GET("/my-endpoint", func(c *gin.Context) {
        // Get service from container
        container := c.MustGet("container").(core.DIContainer)
        myService, _ := container.Resolve("myService")
        
        // Use service
        c.JSON(200, gin.H{"message": "Hello from my plugin!"})
    })
    
    return nil
}
```

## Dependency Injection

The framework provides a powerful dependency injection container with three lifetime options:

1. **Singleton**: One instance for the entire application
2. **Transient**: New instance every time it's requested
3. **Scoped**: One instance per request/scope

### Registering Services

```go
// Singleton
container.RegisterSingleton("userService", func(c core.DIContainer) (interface{}, error) {
    return NewUserService(), nil
})

// Transient
container.RegisterTransient("requestService", func(c core.DIContainer) (interface{}, error) {
    return NewRequestService(), nil
})

// Scoped
container.RegisterScoped("contextService", func(c core.DIContainer) (interface{}, error) {
    return NewContextService(c), nil
})
```

### Resolving Services

```go
// In route handlers
func MyHandler(c *gin.Context, container core.DIContainer) {
    userService, err := container.Resolve("userService")
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    // Use service
    user := userService.(*UserService).GetUser("123")
    c.JSON(200, user)
}
```

## Router Helper

The framework provides a router helper that makes it easy to register routes with DI support:

```go
// Get router from app
router := app.GetRouter()

// Register routes
router.GET("/users", func(c *gin.Context, container core.DIContainer) {
    userService, _ := container.Resolve("userService")
    users := userService.(*UserService).ListUsers()
    c.JSON(200, users)
})

// Route groups
api := router.Group("/api/v1")
{
    api.GET("/users", getUsersHandler)
    api.POST("/users", createUserHandler)
}
```

## Lifecycle Hooks

Plugins can register lifecycle hooks that run at different points in the request lifecycle:

1. **OnRequest**: Runs when a request is received
2. **PreHandler**: Runs before the route handler
3. **OnResponse**: Runs after the response is sent
4. **OnError**: Runs when an error occurs

```go
// Creating hooks
hook := core.NewOnRequestHook(func(c *gin.Context) {
    // Log request
})

// Or implement the full interface
type MyHook struct{}

func (h *MyHook) OnRequest(c *gin.Context) {
    // Request logic
}

func (h *MyHook) PreHandler(c *gin.Context) {
    // Pre-handler logic
}

func (h *MyHook) OnResponse(c *gin.Context, response interface{}) {
    // Response logic
}

func (h *MyHook) OnError(c *gin.Context, err error) {
    // Error handling
}
```

## Project Structure

```
your-project/
├── main.go
├── go.mod
├── libs/
│   └── core/
│       ├── app.go
│       ├── di.go
│       ├── plugin.go
│       ├── lifecycle.go
│       ├── router.go
│       ├── logger.go
│       └── cors.go
├── plugins/
│   ├── auth/
│   │   └── auth.go
│   └── database/
│       └── db.go
└── examples/
    └── user-service/
        ├── main.go
        └── user.go
```

## Running the Examples

1. User Service Example:
   ```bash
   cd examples/user-service
   go run main.go user.go
   ```

2. Test the API:
   ```bash
   # Create a user
   curl -X POST http://localhost:8080/api/v1/users \
     -H "Content-Type: application/json" \
     -d '{"name": "John Doe", "email": "john@example.com"}'
   
   # Get all users
   curl http://localhost:8080/api/v1/users
   
   # Get a specific user
   curl http://localhost:8080/api/v1/users/user-1
   ```
## Example
```go
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



```
## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

MIT License