package core

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
)

// ControllerFunc represents a function that receives an injected controller
type ControllerFunc[T any] func(c *gin.Context, controller T)

// EnhancedRouter provides automatic controller injection with module prefix support
type EnhancedRouter struct {
	*Router
	modulePrefix string // Current module's prefix for auto-prefixing
}

// NewEnhancedRouter creates a new enhanced router
func NewEnhancedRouter(engine *gin.Engine, container DIContainer) *EnhancedRouter {
	return &EnhancedRouter{
		Router:       NewRouter(engine, container),
		modulePrefix: "",
	}
}

// NewEnhancedRouterWithPrefix creates a router with module prefix
func NewEnhancedRouterWithPrefix(engine *gin.Engine, container DIContainer, prefix string) *EnhancedRouter {
	return &EnhancedRouter{
		Router:       NewRouter(engine, container),
		modulePrefix: strings.TrimSuffix(prefix, "/"),
	}
}

// applyPrefix applies module prefix to relative paths
func (r *EnhancedRouter) applyPrefix(path string) string {
	// Absolute paths bypass prefixing if no module prefix is set
	if strings.HasPrefix(path, "/") && r.modulePrefix == "" {
		return path
	}

	// Absolute paths with custom prefix bypass auto-prefixing
	if strings.HasPrefix(path, "/") && r.modulePrefix != "" && !strings.HasPrefix(path, r.modulePrefix) {
		return path
	}

	// Relative path or empty module prefix: return as is
	if r.modulePrefix == "" {
		return path
	}

	// Relative path: apply module prefix
	if !strings.HasPrefix(path, "/") {
		return r.modulePrefix + "/" + path
	}

	// Path starts with module prefix, return as is (already prefixed)
	return path
}

// GET registers a GET route with automatic controller injection
func (r *EnhancedRouter) GET(config RouteConfig, handler interface{}) {
	prefixedPath := r.applyPrefix(config.Path)
	config.Path = prefixedPath

	r.triggerOnRoute(&config)
	r.engine.GET(prefixedPath, r.withController(handler))
}

// POST registers a POST route with automatic controller injection
func (r *EnhancedRouter) POST(config RouteConfig, handler interface{}) {
	prefixedPath := r.applyPrefix(config.Path)
	config.Path = prefixedPath

	r.triggerOnRoute(&config)
	r.engine.POST(prefixedPath, r.withController(handler))
}

// PUT registers a PUT route with automatic controller injection
func (r *EnhancedRouter) PUT(config RouteConfig, handler interface{}) {
	prefixedPath := r.applyPrefix(config.Path)
	config.Path = prefixedPath

	r.triggerOnRoute(&config)
	r.engine.PUT(prefixedPath, r.withController(handler))
}

// PATCH registers a PATCH route with automatic controller injection
func (r *EnhancedRouter) PATCH(config RouteConfig, handler interface{}) {
	prefixedPath := r.applyPrefix(config.Path)
	config.Path = prefixedPath

	r.triggerOnRoute(&config)
	r.engine.PATCH(prefixedPath, r.withController(handler))
}

// DELETE registers a DELETE route with automatic controller injection
func (r *EnhancedRouter) DELETE(config RouteConfig, handler interface{}) {
	prefixedPath := r.applyPrefix(config.Path)
	config.Path = prefixedPath

	r.triggerOnRoute(&config)
	r.engine.DELETE(prefixedPath, r.withController(handler))
}

// OPTIONS registers an OPTIONS route with automatic controller injection
func (r *EnhancedRouter) OPTIONS(config RouteConfig, handler interface{}) {
	prefixedPath := r.applyPrefix(config.Path)
	config.Path = prefixedPath

	r.triggerOnRoute(&config)
	r.engine.OPTIONS(prefixedPath, r.withController(handler))
}

// HEAD registers a HEAD route with automatic controller injection
func (r *EnhancedRouter) HEAD(config RouteConfig, handler interface{}) {
	prefixedPath := r.applyPrefix(config.Path)
	config.Path = prefixedPath

	r.triggerOnRoute(&config)
	r.engine.HEAD(prefixedPath, r.withController(handler))
}

// Any registers a route that matches all HTTP methods with automatic controller injection
func (r *EnhancedRouter) Any(config RouteConfig, handler interface{}) {
	prefixedPath := r.applyPrefix(config.Path)
	config.Path = prefixedPath

	r.triggerOnRoute(&config)
	r.engine.Any(prefixedPath, r.withController(handler))
}

// Group creates a new route group with enhanced capabilities
func (r *EnhancedRouter) Group(relativePath string, handlers ...gin.HandlerFunc) *EnhancedRouterGroup {
	fullPrefix := r.applyPrefix(relativePath)
	group := r.engine.Group(relativePath, handlers...)

	return &EnhancedRouterGroup{
		group:       group,
		router:      r,
		groupPrefix: fullPrefix,  // Track full prefix for this group
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

		// Get request container from context
		var service interface{}
		var err error

		if rc, exists := c.Get("requestContainer"); exists {
			// Resolve from request container
			requestContainer := rc.(*RequestContainer)
			typeName := controllerType.String()
			service, err = requestContainer.Resolve(typeName)
			if err != nil {
				// Try with naming convention
				typeName = toServiceName(controllerType)
				service, err = requestContainer.Resolve(typeName)
			}
		} else {
			// Fallback to global container (should not happen with proper middleware setup)
			typeName := controllerType.String()
			service, err = r.container.Resolve(typeName)
			if err != nil {
				// Try with naming convention
				typeName = toServiceName(controllerType)
				service, err = r.container.Resolve(typeName)
			}
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to resolve controller: %v", err),
			})
			return
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
	group       *gin.RouterGroup
	router      *EnhancedRouter
	groupPrefix string  // Full prefix for this group
}

// Group creates a nested enhanced route group
func (rg *EnhancedRouterGroup) Group(relativePath string, handlers ...gin.HandlerFunc) *EnhancedRouterGroup {
	fullPrefix := rg.groupPrefix
	if relativePath != "" {
		fullPrefix = fullPrefix + "/" + strings.TrimPrefix(relativePath, "/")
	}
	group := rg.group.Group(relativePath, handlers...)

	return &EnhancedRouterGroup{
		group:       group,
		router:      rg.router,
		groupPrefix: fullPrefix,
	}
}

// applyGroupPrefix applies group prefix to a path
func (rg *EnhancedRouterGroup) applyGroupPrefix(path string) string {
	if rg.groupPrefix == "" {
		return path
	}

	if !strings.HasPrefix(path, "/") {
		return rg.groupPrefix + "/" + path
	}

	// Path is already absolute, check if it already has the prefix
	if strings.HasPrefix(path, rg.groupPrefix) {
		return path
	}

	// Return path as is (absolute path without group prefix)
	return path
}

// GET registers a GET route in the group with automatic controller injection
func (rg *EnhancedRouterGroup) GET(config RouteConfig, handler interface{}) {
	// Apply group prefix to the path
	prefixedPath := rg.applyGroupPrefix(config.Path)
	config.Path = prefixedPath

	rg.router.triggerOnRoute(&config)
	rg.group.GET(config.Path, rg.router.withController(handler))
}

// POST registers a POST route in the group with automatic controller injection
func (rg *EnhancedRouterGroup) POST(config RouteConfig, handler interface{}) {
	// Apply group prefix to the path
	prefixedPath := rg.applyGroupPrefix(config.Path)
	config.Path = prefixedPath

	rg.router.triggerOnRoute(&config)
	rg.group.POST(config.Path, rg.router.withController(handler))
}

// PUT registers a PUT route in the group with automatic controller injection
func (rg *EnhancedRouterGroup) PUT(config RouteConfig, handler interface{}) {
	// Apply group prefix to the path
	prefixedPath := rg.applyGroupPrefix(config.Path)
	config.Path = prefixedPath

	rg.router.triggerOnRoute(&config)
	rg.group.PUT(config.Path, rg.router.withController(handler))
}

// PATCH registers a PATCH route in the group with automatic controller injection
func (rg *EnhancedRouterGroup) PATCH(config RouteConfig, handler interface{}) {
	// Apply group prefix to the path
	prefixedPath := rg.applyGroupPrefix(config.Path)
	config.Path = prefixedPath

	rg.router.triggerOnRoute(&config)
	rg.group.PATCH(config.Path, rg.router.withController(handler))
}

// DELETE registers a DELETE route in the group with automatic controller injection
func (rg *EnhancedRouterGroup) DELETE(config RouteConfig, handler interface{}) {
	// Apply group prefix to the path
	prefixedPath := rg.applyGroupPrefix(config.Path)
	config.Path = prefixedPath

	rg.router.triggerOnRoute(&config)
	rg.group.DELETE(config.Path, rg.router.withController(handler))
}

// OPTIONS registers an OPTIONS route in the group with automatic controller injection
func (rg *EnhancedRouterGroup) OPTIONS(config RouteConfig, handler interface{}) {
	// Apply group prefix to the path
	prefixedPath := rg.applyGroupPrefix(config.Path)
	config.Path = prefixedPath

	rg.router.triggerOnRoute(&config)
	rg.group.OPTIONS(config.Path, rg.router.withController(handler))
}

// HEAD registers a HEAD route in the group with automatic controller injection
func (rg *EnhancedRouterGroup) HEAD(config RouteConfig, handler interface{}) {
	// Apply group prefix to the path
	prefixedPath := rg.applyGroupPrefix(config.Path)
	config.Path = prefixedPath

	rg.router.triggerOnRoute(&config)
	rg.group.HEAD(config.Path, rg.router.withController(handler))
}

// Any registers a route that matches all HTTP methods in the group with automatic controller injection
func (rg *EnhancedRouterGroup) Any(config RouteConfig, handler interface{}) {
	// Apply group prefix to the path
	prefixedPath := rg.applyGroupPrefix(config.Path)
	config.Path = prefixedPath

	rg.router.triggerOnRoute(&config)
	rg.group.Any(config.Path, rg.router.withController(handler))
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
