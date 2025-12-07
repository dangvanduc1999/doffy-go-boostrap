package core

import (
	"github.com/gin-gonic/gin"
)

// RouteHandler defines a handler function that has access to the DI container
type RouteHandler func(c *gin.Context, container DIContainer)

// RouteConfig contains configuration options for a route
type RouteConfig struct {
	Path            string
	IsAuth          *bool
	SchemaValidator interface{}
	Options         map[string]interface{}
}

// Router wraps gin.Engine and provides dependency injection support
type Router struct {
	engine    *gin.Engine
	container DIContainer
}

// NewRouter creates a new router helper
func NewRouter(engine *gin.Engine, container DIContainer) *Router {
	return &Router{
		engine:    engine,
		container: container,
	}
}

// Group creates a new route group
func (r *Router) Group(relativePath string, handlers ...gin.HandlerFunc) *RouterGroup {
	return &RouterGroup{
		group:  r.engine.Group(relativePath, handlers...),
		router: r,
	}
}

// GET registers a GET route
func (r *Router) GET(config RouteConfig, handler RouteHandler) {
	r.triggerOnRoute(&config)
	r.engine.GET(config.Path, r.wrapHandler(handler))
}

// POST registers a POST route
func (r *Router) POST(config RouteConfig, handler RouteHandler) {
	r.triggerOnRoute(&config)
	r.engine.POST(config.Path, r.wrapHandler(handler))
}

// PUT registers a PUT route
func (r *Router) PUT(config RouteConfig, handler RouteHandler) {
	r.triggerOnRoute(&config)
	r.engine.PUT(config.Path, r.wrapHandler(handler))
}

// PATCH registers a PATCH route
func (r *Router) PATCH(config RouteConfig, handler RouteHandler) {
	r.triggerOnRoute(&config)
	r.engine.PATCH(config.Path, r.wrapHandler(handler))
}

// DELETE registers a DELETE route
func (r *Router) DELETE(config RouteConfig, handler RouteHandler) {
	r.triggerOnRoute(&config)
	r.engine.DELETE(config.Path, r.wrapHandler(handler))
}

// OPTIONS registers an OPTIONS route
func (r *Router) OPTIONS(config RouteConfig, handler RouteHandler) {
	r.triggerOnRoute(&config)
	r.engine.OPTIONS(config.Path, r.wrapHandler(handler))
}

// HEAD registers a HEAD route
func (r *Router) HEAD(config RouteConfig, handler RouteHandler) {
	r.triggerOnRoute(&config)
	r.engine.HEAD(config.Path, r.wrapHandler(handler))
}

// Any registers a route that matches all HTTP methods
func (r *Router) Any(config RouteConfig, handler RouteHandler) {
	r.triggerOnRoute(&config)
	r.engine.Any(config.Path, r.wrapHandler(handler))
}

// buildOptions converts RouteConfig to options map
func (r *Router) buildOptions(config RouteConfig) map[string]interface{} {
	options := make(map[string]interface{})

	if config.Options != nil {
		for k, v := range config.Options {
			options[k] = v
		}
	}

	if config.IsAuth != nil {
		options["isAuth"] = *config.IsAuth
	}

	if config.SchemaValidator != nil {
		options["schema"] = config.SchemaValidator
	}

	return options
}

// Static registers a static file server
func (r *Router) Static(relativePath, root string) {
	r.engine.Static(relativePath, root)
}

// StaticFile registers a single static file
func (r *Router) StaticFile(relativePath, filepath string) {
	r.engine.StaticFile(relativePath, filepath)
}

// wrapHandler wraps a RouteHandler to provide access to the DI container
func (r *Router) wrapHandler(handler RouteHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get container from context
		container, exists := c.Get("container")
		if !exists {
			c.JSON(500, gin.H{"error": "DI container not found"})
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

		// Call the handler with the container
		handler(c, container.(DIContainer))
	}
}

// triggerOnRoute triggers the OnRoute hook
func (r *Router) triggerOnRoute(config *RouteConfig) {
	if pm, err := r.container.Resolve("pluginManager"); err == nil {
		if pluginManager, ok := pm.(*PluginManager); ok {
			pluginManager.ExecuteOnRoute(config)
		}
	}
}

// RouterGroup provides helper methods for route groups
type RouterGroup struct {
	group  *gin.RouterGroup
	router *Router
}

// Group creates a nested route group
func (rg *RouterGroup) Group(relativePath string, handlers ...gin.HandlerFunc) *RouterGroup {
	return &RouterGroup{
		group:  rg.group.Group(relativePath, handlers...),
		router: rg.router,
	}
}

// GET registers a GET route in the group
func (rg *RouterGroup) GET(config RouteConfig, handler RouteHandler) {
	rg.router.triggerOnRoute(&config)
	rg.group.GET(config.Path, rg.router.wrapHandler(handler))
}

// POST registers a POST route in the group
func (rg *RouterGroup) POST(config RouteConfig, handler RouteHandler) {
	rg.router.triggerOnRoute(&config)
	rg.group.POST(config.Path, rg.router.wrapHandler(handler))
}

// PUT registers a PUT route in the group
func (rg *RouterGroup) PUT(config RouteConfig, handler RouteHandler) {
	rg.router.triggerOnRoute(&config)
	rg.group.PUT(config.Path, rg.router.wrapHandler(handler))
}

// PATCH registers a PATCH route in the group
func (rg *RouterGroup) PATCH(config RouteConfig, handler RouteHandler) {
	rg.router.triggerOnRoute(&config)
	rg.group.PATCH(config.Path, rg.router.wrapHandler(handler))
}

// DELETE registers a DELETE route in the group
func (rg *RouterGroup) DELETE(config RouteConfig, handler RouteHandler) {
	rg.router.triggerOnRoute(&config)
	rg.group.DELETE(config.Path, rg.router.wrapHandler(handler))
}

// OPTIONS registers an OPTIONS route in the group
func (rg *RouterGroup) OPTIONS(config RouteConfig, handler RouteHandler) {
	rg.router.triggerOnRoute(&config)
	rg.group.OPTIONS(config.Path, rg.router.wrapHandler(handler))
}

// HEAD registers a HEAD route in the group
func (rg *RouterGroup) HEAD(config RouteConfig, handler RouteHandler) {
	rg.router.triggerOnRoute(&config)
	rg.group.HEAD(config.Path, rg.router.wrapHandler(handler))
}

// Any registers a route that matches all HTTP methods in the group
func (rg *RouterGroup) Any(config RouteConfig, handler RouteHandler) {
	rg.router.triggerOnRoute(&config)
	rg.group.Any(config.Path, rg.router.wrapHandler(handler))
}

// Static registers a static file server in the group
func (rg *RouterGroup) Static(relativePath, root string) {
	rg.group.Static(relativePath, root)
}

// StaticFile registers a single static file in the group
func (rg *RouterGroup) StaticFile(relativePath, filepath string) {
	rg.group.StaticFile(relativePath, filepath)
}

// Use adds middleware to the group
func (rg *RouterGroup) Use(middleware ...gin.HandlerFunc) {
	rg.group.Use(middleware...)
}
