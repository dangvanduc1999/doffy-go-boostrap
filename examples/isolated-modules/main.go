package main

import (
	"fmt"
	"net/http"

	"github.com/dangvanduc1999/doffy-go-boostrap/libs/core"
	"github.com/gin-gonic/gin"
)

// ModuleA has a private service that should not be accessible to other modules
type ModuleA struct {
	core.BasePlugin
}

type PrivateService struct {
	name string
}

func (s *PrivateService) SecretOperation(data string) string {
	return fmt.Sprintf("Secret from ModuleA: %s", data)
}

type ExportedService struct {
	name string
}

func (s *ExportedService) PublicOperation(data string) string {
	return fmt.Sprintf("Public from ModuleA: %s", data)
}

func (p *ModuleA) Name() string {
	return "module-a"
}

func (p *ModuleA) Version() string {
	return "1.0.0"
}

func (p *ModuleA) Module() *core.Module {
	module := core.DefaultModule("module-a", "1.0.0")
	module.Providers = []core.Provider{
		// Private service - not exported
		core.NewFactoryProvider("privateService", func(container core.DIContainer) (interface{}, error) {
			return &PrivateService{name: "private"}, nil
		}, core.Singleton),
		// Exported service
		core.NewFactoryProvider("exportedService", func(container core.DIContainer) (interface{}, error) {
			return &ExportedService{name: "exported"}, nil
		}, core.Singleton),
	}
	module.Exports = []string{"exportedService"} // Only export 'exportedService'
	return module
}

func (p *ModuleA) Register(container core.DIContainer) error {
	module := p.Module()
	for _, provider := range module.Providers {
		if err := container.RegisterProvider(provider); err != nil {
			return err
		}
	}
	return nil
}

func (p *ModuleA) Hooks() []core.LifecycleHook {
	return []core.LifecycleHook{}
}

func (p *ModuleA) Routes(engine *gin.Engine) error {
	// Route that can access private service (within same module)
	router := core.NewEnhancedRouter(engine, core.GlobalLocator.GetContainer())

	router.GET(core.RouteConfig{Path: "/api/module-a/private"}, func(c *gin.Context, ctrl struct{}) {
		// Get container from context
		if rc, exists := c.Get("requestContainer"); exists {
			requestContainer := rc.(*core.RequestContainer)

			// Try to resolve private service
			if service, err := requestContainer.Resolve("privateService"); err == nil {
				if privateSvc, ok := service.(*PrivateService); ok {
					c.JSON(http.StatusOK, gin.H{
						"message": "Accessed private service from within ModuleA",
						"result":  privateSvc.SecretOperation("test data"),
					})
					return
				}
			}
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Private service not accessible"})
	})

	router.GET(core.RouteConfig{Path: "/api/module-a/exported"}, func(c *gin.Context, ctrl struct{}) {
		// Get container from context
		if rc, exists := c.Get("requestContainer"); exists {
			requestContainer := rc.(*core.RequestContainer)

			// Try to resolve exported service
			if service, err := requestContainer.Resolve("exportedService"); err == nil {
				if exportedSvc, ok := service.(*ExportedService); ok {
					c.JSON(http.StatusOK, gin.H{
						"message": "Accessed exported service",
						"result":  exportedSvc.PublicOperation("test data"),
					})
					return
				}
			}
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Exported service not accessible"})
	})

	return nil
}

// ModuleB attempts to access ModuleA's private service
type ModuleB struct {
	core.BasePlugin
}

func (p *ModuleB) Name() string {
	return "module-b"
}

func (p *ModuleB) Version() string {
	return "1.0.0"
}

func (p *ModuleB) Module() *core.Module {
	module := core.DefaultModule("module-b", "1.0.0")
	// Import ModuleA
	module.Imports = []*core.Module{
		core.DefaultModule("module-a", "1.0.0"),
	}
	// This module doesn't register any services
	module.Providers = []core.Provider{}
	return module
}

func (p *ModuleB) Register(container core.DIContainer) error {
	module := p.Module()
	for _, provider := range module.Providers {
		if err := container.RegisterProvider(provider); err != nil {
			return err
		}
	}
	return nil
}

func (p *ModuleB) Hooks() []core.LifecycleHook {
	return []core.LifecycleHook{}
}

func (p *ModuleB) Routes(engine *gin.Engine) error {
	router := core.NewEnhancedRouter(engine, core.GlobalLocator.GetContainer())

	// This route tries to access ModuleA's private service (should fail)
	router.GET(core.RouteConfig{Path: "/api/module-b/try-private"}, func(c *gin.Context, ctrl struct{}) {
		// Get container from context
		if rc, exists := c.Get("requestContainer"); exists {
			requestContainer := rc.(*core.RequestContainer)

			// Try to resolve ModuleA's private service
			if _, err := requestContainer.Resolve("privateService"); err != nil {
				c.JSON(http.StatusForbidden, gin.H{
					"error":   "Access denied",
					"message": "Cannot access private service from ModuleA",
					"detail":  err.Error(),
				})
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{"message": "Unexpectedly accessed private service"})
	})

	// This route accesses ModuleA's exported service (should succeed)
	router.GET(core.RouteConfig{Path: "/api/module-b/access-exported"}, func(c *gin.Context, ctrl struct{}) {
		// Get container from context
		if rc, exists := c.Get("requestContainer"); exists {
			requestContainer := rc.(*core.RequestContainer)

			// Try to resolve ModuleA's exported service
			if service, err := requestContainer.Resolve("exportedService"); err == nil {
				if exportedSvc, ok := service.(*ExportedService); ok {
					c.JSON(http.StatusOK, gin.H{
						"message": "Successfully accessed exported service from ModuleA",
						"result":  exportedSvc.PublicOperation("from ModuleB"),
					})
					return
				}
			}
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to access exported service"})
	})

	return nil
}

// GlobalModule has Global flag set to bypass encapsulation
type GlobalModule struct {
	core.BasePlugin
}

type GlobalService struct{}

func (s *GlobalService) GlobalOperation() string {
	return "This service is globally accessible"
}

func (p *GlobalModule) Name() string {
	return "global-module"
}

func (p *GlobalModule) Version() string {
	return "1.0.0"
}

func (p *GlobalModule) Module() *core.Module {
	return &core.Module{
		Name:        "global-module",
		Version:     "1.0.0",
		Description: "Global module that bypasses encapsulation",
		Global:      true, // This flag bypasses all encapsulation checks
		Providers: []core.Provider{
			core.NewFactoryProvider("globalService", func(container core.DIContainer) (interface{}, error) {
				return &GlobalService{}, nil
			}, core.Singleton),
		},
		Exports: []string{"globalService"},
	}
}

func (p *GlobalModule) Register(container core.DIContainer) error {
	module := p.Module()
	for _, provider := range module.Providers {
		if err := container.RegisterProvider(provider); err != nil {
			return err
		}
	}
	return nil
}

func (p *GlobalModule) Hooks() []core.LifecycleHook {
	return []core.LifecycleHook{}
}

func (p *GlobalModule) Routes(engine *gin.Engine) error {
	return nil
}

func main() {
	// Set encapsulation mode to enforce violations
	core.SetEncapsulationMode(core.EncapsulationEnforce)

	gin.SetMode(gin.ReleaseMode)

	app := core.CreateDoffApp(&core.AppOptions{
		Name:      "Isolated Modules Demo",
		Mode:      gin.ReleaseMode,
		UseLogger: true,
		Port:      8080,
	})

	// Register ModuleA (with private and exported services)
	app.RegisterPlugin(&ModuleA{})

	// Register ModuleB (attempts to access ModuleA's private service)
	app.RegisterPlugin(&ModuleB{})

	// Register GlobalModule (bypasses encapsulation)
	app.RegisterPlugin(&GlobalModule{})

	// Add request container middleware
	router := app.GetEngine()
	router.Use(func(c *gin.Context) {
		// Create a simple module container for demonstration
		moduleContainer := core.NewModuleContainer(
			core.DefaultModule("request-scope", "1.0.0"),
			app.GetContainer(),
		)

		// Create request-scoped container
		requestContainer := core.NewRequestContainer(moduleContainer)
		c.Set("requestContainer", requestContainer)

		c.Next()
	})

	// Add a root route to demonstrate global service access
	router.GET("/api/global", func(c *gin.Context) {
		if rc, exists := c.Get("requestContainer"); exists {
			requestContainer := rc.(*core.RequestContainer)

			// Try to resolve global service
			if service, err := requestContainer.Resolve("globalService"); err == nil {
				if globalSvc, ok := service.(*GlobalService); ok {
					c.JSON(http.StatusOK, gin.H{
						"message": "Accessed global service",
						"result":  globalSvc.GlobalOperation(),
					})
					return
				}
			}
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Global service not accessible"})
	})

	// Add a status route
	router.GET("/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message":  "Isolated Modules Demo Running",
			"endpoints": gin.H{
				"ModuleA (self)": gin.H{
					"private":  "/api/module-a/private",
					"exported": "/api/module-a/exported",
				},
				"ModuleB (cross-module)": gin.H{
					"try_private":   "/api/module-b/try-private",
					"access_exported": "/api/module-b/access-exported",
				},
				"Global": "/api/global",
			},
		})
	})

	fmt.Println("Isolated Modules Demo starting on :8080")
	fmt.Println("\nEndpoints to test:")
	fmt.Println("  ✓ GET /status - Show available endpoints")
	fmt.Println("  ✓ GET /api/module-a/private - ModuleA accessing its private service")
	fmt.Println("  ✓ GET /api/module-a/exported - ModuleA accessing its exported service")
	fmt.Println("  ✓ GET /api/module-b/try-private - ModuleB trying to access ModuleA's private service (should fail)")
	fmt.Println("  ✓ GET /api/module-b/access-exported - ModuleB accessing ModuleA's exported service (should succeed)")
	fmt.Println("  ✓ GET /api/global - Accessing global service (always succeeds)")
	fmt.Println("\nEncapsulation Mode: ENFORCED")
	fmt.Println("  - Private services are isolated to their module")
	fmt.Println("  - Only exported services are accessible to other modules")
	fmt.Println("  - Global modules bypass encapsulation")

	app.Listen()
}