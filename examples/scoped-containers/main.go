package main

import (
	"fmt"
	"net/http"

	"github.com/dangvanduc1999/doffy-go-boostrap/libs/core"
	"github.com/gin-gonic/gin"
)

// DemoPlugin demonstrates module-scoped containers
type DemoPlugin struct {
	core.BasePlugin
}

// Name returns the plugin name
func (p *DemoPlugin) Name() string {
	return "demo-plugin"
}

// Version returns the plugin version
func (p *DemoPlugin) Version() string {
	return "1.0.0"
}

// Module returns the module definition
func (p *DemoPlugin) Module() *core.Module {
	return core.DefaultModule("demo-module", "1.0.0").
		WithProviders(
			core.NewFactoryProvider("simpleService", func(container core.DIContainer) (interface{}, error) {
				return NewSimpleService("ModuleScoped"), nil
			}, core.Singleton),
		).
		WithExports("simpleService")
}

// Register registers the plugin with the DI container
func (p *DemoPlugin) Register(container core.DIContainer) error {
	module := p.Module()
	for _, provider := range module.Providers {
		if err := container.RegisterProvider(provider); err != nil {
			return err
		}
	}
	return nil
}

// Hooks returns lifecycle hooks
func (p *DemoPlugin) Hooks() []core.LifecycleHook {
	return []core.LifecycleHook{}
}

// Routes registers plugin routes
func (p *DemoPlugin) Routes(engine *gin.Engine) error {
	return nil
}

// SimpleService represents a basic service
type SimpleService struct {
	name string
}

func NewSimpleService(name string) *SimpleService {
	return &SimpleService{name: name}
}

func (s *SimpleService) Process(data string) string {
	return fmt.Sprintf("%s processed: %s", s.name, data)
}

// RequestScopedService demonstrates per-request service
type RequestScopedService struct {
	requestID string
}

func NewRequestScopedService(requestID string) *RequestScopedService {
	return &RequestScopedService{requestID: requestID}
}

func (s *RequestScopedService) GetRequestID() string {
	return s.requestID
}

func main() {
	// Create app with scoped container support
	app := core.CreateDoffApp(&core.AppOptions{
		Name:      "Scoped Containers Demo",
		Mode:      gin.DebugMode,
		UseLogger: true,
		Port:      8080,
	})

	// Type assert to DoffApp to access decorator methods
	doffApp := app.(*core.DoffApp)

	// Register global decorators
	doffApp.DecorateRequest("correlationID", "global-correlation")
	doffApp.DecorateReply("standardResponse", func(data interface{}) map[string]interface{} {
		return map[string]interface{}{
			"success": true,
			"data":    data,
			"version": "1.0",
		}
	})

	// Get router and add middleware for request container
	router := app.GetEngine()

	// Add request container middleware
	router.Use(func(c *gin.Context) {
		// Create module container (using root as parent)
		moduleContainer := app.GetContainer().CreateModuleScope(
			core.DefaultModule("request-scope", "1.0.0"),
		)

		// Create request-scoped container
		requestContainer := core.NewRequestContainer(moduleContainer)

		// Decorate request with correlation ID from header or use default
		corrID := c.GetHeader("X-Correlation-ID")
		if corrID == "" {
			if defCorrID, exists := doffApp.GetDecoratorManager().GetRequestDecorator("correlationID"); exists {
				corrID = defCorrID.(string)
			}
		}
		requestContainer.DecorateRequest("correlationID", corrID)

		// Register request-scoped service
		requestContainer.DecorateRequest("requestScopedService",
			NewRequestScopedService(corrID))

		// Initialize decorators from app's decorator manager
		doffApp.GetDecoratorManager().InitializeRequestContainer(requestContainer)
		doffApp.GetDecoratorManager().InitializeReplyHelpers(requestContainer)

		// Set request container in context
		c.Set("requestContainer", requestContainer)

		c.Next()
	})

	// Register routes that use scoped containers
	router.GET("/demo", func(c *gin.Context) {
		// Get request container
		rc, exists := c.Get("requestContainer")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "request container not found"})
			return
		}
		requestContainer := rc.(*core.RequestContainer)

		// Get correlation ID from request decorators
		if corrID, exists := requestContainer.GetRequestData("correlationID"); exists {
			c.Header("X-Correlation-ID", corrID.(string))
		}

		// Get request-scoped service
		if service, exists := requestContainer.GetRequestData("requestScopedService"); exists {
			if reqService, ok := service.(*RequestScopedService); ok {
				c.JSON(http.StatusOK, gin.H{
					"message":     "Request scoped service accessed",
					"requestID":   reqService.GetRequestID(),
					"correlation": reqService.requestID,
				})
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{"message": "Hello from scoped containers demo!"})
	})

	router.GET("/standard-response", func(c *gin.Context) {
		// Get request container
		rc, exists := c.Get("requestContainer")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "request container not found"})
			return
		}
		requestContainer := rc.(*core.RequestContainer)

		// Use standard response decorator
		if helper, exists := requestContainer.GetReplyHelper("standardResponse"); exists {
			if responseFn, ok := helper.(func(interface{}) map[string]interface{}); ok {
				response := responseFn(gin.H{
					"message": "This response uses a decorator",
					"timestamp": "2024-01-01T00:00:00Z",
				})
				c.JSON(http.StatusOK, response)
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{"message": "Standard response decorator not found"})
	})

	router.POST("/decorate", func(c *gin.Context) {
		var req struct {
			Key   string      `json:"key"`
			Value interface{} `json:"value"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Get request container and add custom decoration
		rc, exists := c.Get("requestContainer")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "request container not found"})
			return
		}
		requestContainer := rc.(*core.RequestContainer)

		// Add custom request decoration
		requestContainer.DecorateRequest(req.Key, req.Value)

		// Retrieve it back to demonstrate
		if value, exists := requestContainer.GetRequestData(req.Key); exists {
			c.JSON(http.StatusOK, gin.H{
				"message": "Decoration added successfully",
				"key":     req.Key,
				"value":   value,
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve decoration"})
		}
	})

	// Register a demo plugin
	demoPlugin := &DemoPlugin{}
	app.RegisterPlugin(demoPlugin)

	// Add route that uses module-scoped service
	router.GET("/module-service", func(c *gin.Context) {
		// Get request container
		rc, exists := c.Get("requestContainer")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "request container not found"})
			return
		}
		requestContainer := rc.(*core.RequestContainer)

		// Try to resolve from request container (will check parent module container)
		if service, err := requestContainer.Resolve("simpleService"); err == nil {
			if simpleService, ok := service.(*SimpleService); ok {
				c.JSON(http.StatusOK, gin.H{
					"message": "Module-scoped service accessed",
					"result":  simpleService.Process("test data"),
				})
				return
			}
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Module service not found"})
	})

	fmt.Println("Starting Scoped Containers Demo on :8080")
	fmt.Println("Available endpoints:")
	fmt.Println("  GET /demo - Demonstrates request-scoped services")
	fmt.Println("  GET /standard-response - Shows response decorators")
	fmt.Println("  POST /decorate - Add custom request decorations")
	fmt.Println("  GET /module-service - Uses module-scoped services")

	app.Listen()
}