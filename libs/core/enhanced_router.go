package core

import (
	"fmt"
	"net/http"
	"reflect"

	"github.com/gin-gonic/gin"
)

// ControllerFunc represents a function that receives an injected controller
type ControllerFunc[T any] func(c *gin.Context, controller T)

// EnhancedRouter provides automatic controller injection
type EnhancedRouter struct {
	*Router
}

// NewEnhancedRouter creates a new enhanced router
func NewEnhancedRouter(engine *gin.Engine, container DIContainer) *EnhancedRouter {
	return &EnhancedRouter{
		Router: NewRouter(engine, container),
	}
}

// GET registers a GET route with automatic controller injection
func (r *EnhancedRouter) GET(path string, handler interface{}) {
	r.engine.GET(path, r.withController(handler))
}

// POST registers a POST route with automatic controller injection
func (r *EnhancedRouter) POST(path string, handler interface{}) {
	r.engine.POST(path, r.withController(handler))
}

// PUT registers a PUT route with automatic controller injection
func (r *EnhancedRouter) PUT(path string, handler interface{}) {
	r.engine.PUT(path, r.withController(handler))
}

// PATCH registers a PATCH route with automatic controller injection
func (r *EnhancedRouter) PATCH(path string, handler interface{}) {
	r.engine.PATCH(path, r.withController(handler))
}

// DELETE registers a DELETE route with automatic controller injection
func (r *EnhancedRouter) DELETE(path string, handler interface{}) {
	r.engine.DELETE(path, r.withController(handler))
}

// OPTIONS registers an OPTIONS route with automatic controller injection
func (r *EnhancedRouter) OPTIONS(path string, handler interface{}) {
	r.engine.OPTIONS(path, r.withController(handler))
}

// HEAD registers a HEAD route with automatic controller injection
func (r *EnhancedRouter) HEAD(path string, handler interface{}) {
	r.engine.HEAD(path, r.withController(handler))
}

// Any registers a route that matches all HTTP methods with automatic controller injection
func (r *EnhancedRouter) Any(path string, handler interface{}) {
	r.engine.Any(path, r.withController(handler))
}

// Group creates a new route group with enhanced capabilities
func (r *EnhancedRouter) Group(relativePath string, handlers ...gin.HandlerFunc) *EnhancedRouterGroup {
	group := r.engine.Group(relativePath, handlers...)
	return &EnhancedRouterGroup{
		group:  group,
		router: r,
	}
}

// withController creates a middleware that automatically injects the controller
func (r *EnhancedRouter) withController(handler interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get handler value and type
		handlerValue := reflect.ValueOf(handler)
		handlerType := handlerValue.Type()

		// Check if it's a function with the right signature
		if handlerType.Kind() != reflect.Func || handlerType.NumIn() != 2 {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Invalid handler signature",
			})
			return
		}

		// Get controller type from the handler's second parameter
		controllerType := handlerType.In(1)

		// Try to resolve by type name
		typeName := controllerType.String()
		service, err := r.container.Resolve(typeName)
		if err != nil {
			// Try with naming convention
			typeName = toServiceName(controllerType)
			service, err = r.container.Resolve(typeName)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": fmt.Sprintf("Failed to resolve controller: %v", err),
				})
				return
			}
		}

		// Execute pre-handler hooks
		if app, exists := c.Get("app"); exists {
			if doffApp, ok := app.(*DoffApp); ok {
				doffApp.pluginManager.GetLifecycleManager().ExecutePreHandler(c)
				if c.IsAborted() {
					return
				}
			}
		}

		// Call the handler with injected controller
		args := []reflect.Value{
			reflect.ValueOf(c),
			reflect.ValueOf(service),
		}
		handlerValue.Call(args)
	}
}

// EnhancedRouterGroup provides enhanced route groups
type EnhancedRouterGroup struct {
	group  *gin.RouterGroup
	router *EnhancedRouter
}

// Group creates a nested enhanced route group
func (rg *EnhancedRouterGroup) Group(relativePath string, handlers ...gin.HandlerFunc) *EnhancedRouterGroup {
	return &EnhancedRouterGroup{
		group:  rg.group.Group(relativePath, handlers...),
		router: rg.router,
	}
}

// GET registers a GET route in the group with automatic controller injection
func (rg *EnhancedRouterGroup) GET(path string, handler interface{}) {
	rg.group.GET(path, rg.router.withController(handler))
}

// POST registers a POST route in the group with automatic controller injection
func (rg *EnhancedRouterGroup) POST(path string, handler interface{}) {
	rg.group.POST(path, rg.router.withController(handler))
}

// PUT registers a PUT route in the group with automatic controller injection
func (rg *EnhancedRouterGroup) PUT(path string, handler interface{}) {
	rg.group.PUT(path, rg.router.withController(handler))
}

// PATCH registers a PATCH route in the group with automatic controller injection
func (rg *EnhancedRouterGroup) PATCH(path string, handler interface{}) {
	rg.group.PATCH(path, rg.router.withController(handler))
}

// DELETE registers a DELETE route in the group with automatic controller injection
func (rg *EnhancedRouterGroup) DELETE(path string, handler interface{}) {
	rg.group.DELETE(path, rg.router.withController(handler))
}

// OPTIONS registers an OPTIONS route in the group with automatic controller injection
func (rg *EnhancedRouterGroup) OPTIONS(path string, handler interface{}) {
	rg.group.OPTIONS(path, rg.router.withController(handler))
}

// HEAD registers a HEAD route in the group with automatic controller injection
func (rg *EnhancedRouterGroup) HEAD(path string, handler interface{}) {
	rg.group.HEAD(path, rg.router.withController(handler))
}

// Any registers a route that matches all HTTP methods in the group with automatic controller injection
func (rg *EnhancedRouterGroup) Any(path string, handler interface{}) {
	rg.group.Any(path, rg.router.withController(handler))
}

// Use adds middleware to the group
func (rg *EnhancedRouterGroup) Use(middleware ...gin.HandlerFunc) {
	rg.group.Use(middleware...)
}

// Static registers a static file server in the group
func (rg *EnhancedRouterGroup) Static(relativePath, root string) {
	rg.group.Static(relativePath, root)
}

// StaticFile registers a single static file in the group
func (rg *EnhancedRouterGroup) StaticFile(relativePath, filepath string) {
	rg.group.StaticFile(relativePath, filepath)
}

// Helper function to get the enhanced router from DoffApp
func (d *DoffApp) GetEnhancedRouter() *EnhancedRouter {
	return NewEnhancedRouter(d.server, d.container)
}
