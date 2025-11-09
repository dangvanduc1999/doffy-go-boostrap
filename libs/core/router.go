package core

import (
	"github.com/gin-gonic/gin"
)

// RouteHandler defines a handler function that has access to the DI container
type RouteHandler func(c *gin.Context, container DIContainer)

// Router provides helper methods for registering routes with DI support
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
func (r *Router) GET(path string, handler RouteHandler) {
	r.engine.GET(path, r.wrapHandler(handler))
}

// POST registers a POST route
func (r *Router) POST(path string, handler RouteHandler) {
	r.engine.POST(path, r.wrapHandler(handler))
}

// PUT registers a PUT route
func (r *Router) PUT(path string, handler RouteHandler) {
	r.engine.PUT(path, r.wrapHandler(handler))
}

// PATCH registers a PATCH route
func (r *Router) PATCH(path string, handler RouteHandler) {
	r.engine.PATCH(path, r.wrapHandler(handler))
}

// DELETE registers a DELETE route
func (r *Router) DELETE(path string, handler RouteHandler) {
	r.engine.DELETE(path, r.wrapHandler(handler))
}

// OPTIONS registers an OPTIONS route
func (r *Router) OPTIONS(path string, handler RouteHandler) {
	r.engine.OPTIONS(path, r.wrapHandler(handler))
}

// HEAD registers a HEAD route
func (r *Router) HEAD(path string, handler RouteHandler) {
	r.engine.HEAD(path, r.wrapHandler(handler))
}

// Any registers a route that matches all HTTP methods
func (r *Router) Any(path string, handler RouteHandler) {
	r.engine.Any(path, r.wrapHandler(handler))
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
func (rg *RouterGroup) GET(path string, handler RouteHandler) {
	rg.group.GET(path, rg.router.wrapHandler(handler))
}

// POST registers a POST route in the group
func (rg *RouterGroup) POST(path string, handler RouteHandler) {
	rg.group.POST(path, rg.router.wrapHandler(handler))
}

// PUT registers a PUT route in the group
func (rg *RouterGroup) PUT(path string, handler RouteHandler) {
	rg.group.PUT(path, rg.router.wrapHandler(handler))
}

// PATCH registers a PATCH route in the group
func (rg *RouterGroup) PATCH(path string, handler RouteHandler) {
	rg.group.PATCH(path, rg.router.wrapHandler(handler))
}

// DELETE registers a DELETE route in the group
func (rg *RouterGroup) DELETE(path string, handler RouteHandler) {
	rg.group.DELETE(path, rg.router.wrapHandler(handler))
}

// OPTIONS registers an OPTIONS route in the group
func (rg *RouterGroup) OPTIONS(path string, handler RouteHandler) {
	rg.group.OPTIONS(path, rg.router.wrapHandler(handler))
}

// HEAD registers a HEAD route in the group
func (rg *RouterGroup) HEAD(path string, handler RouteHandler) {
	rg.group.HEAD(path, rg.router.wrapHandler(handler))
}

// Any registers a route that matches all HTTP methods in the group
func (rg *RouterGroup) Any(path string, handler RouteHandler) {
	rg.group.Any(path, rg.router.wrapHandler(handler))
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
