package request

import (
	"fmt"
	"net/http"

	"github.com/dangvanduc1999/doffy-go-boostrap/libs/core"
	"github.com/gin-gonic/gin"
)

type RequestAuthentication struct {
	core.BasePlugin
	publicRoutes map[string]bool
}

func NewRequestAuthentication() *RequestAuthentication {
	return &RequestAuthentication{
		publicRoutes: make(map[string]bool),
	}
}

func (p *RequestAuthentication) Name() string {
	return "request-authentication"
}

func (p *RequestAuthentication) Version() string {
	return "1.0.0"
}

func (p *RequestAuthentication) Register(container core.DIContainer) error {
	return nil
}

func (p *RequestAuthentication) Hooks() []core.LifecycleHook {
	return []core.LifecycleHook{
		NewRequestAuthenticationHook(p),
	}
}

// OnRoute implements core.RouteAwarePlugin
func (p *RequestAuthentication) OnRoute(info core.RouteInfo) {
	fmt.Printf("[RequestAuthentication] Route registered: %s %s\n", info.Method, info.Path)

	// Check if isAuth option is present and false
	if info.Options != nil {
		if isAuth, ok := info.Options["isAuth"]; ok {
			if isAuthBool, ok := isAuth.(bool); ok && !isAuthBool {
				key := fmt.Sprintf("%s:%s", info.Method, info.Path)
				p.publicRoutes[key] = true
				fmt.Printf("[RequestAuthentication] Route marked as public: %s\n", key)
			}
		}
	}
}

type RequestAuthenticationHook struct {
	plugin *RequestAuthentication
}

func NewRequestAuthenticationHook(plugin *RequestAuthentication) *RequestAuthenticationHook {
	return &RequestAuthenticationHook{
		plugin: plugin,
	}
}

// OnRequest implements core.LifecycleHook
func (h *RequestAuthenticationHook) OnRequest(c *gin.Context) {
	// Check if route is public
	// Note: c.FullPath() returns the matched path pattern (e.g. /users/:id)
	key := fmt.Sprintf("%s:%s", c.Request.Method, c.FullPath())

	if h.plugin.publicRoutes[key] {
		// Public route, skip auth
		return
	}

	// Perform authentication
	// For demonstration, we just check for a header
	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		c.Abort()
		return
	}
}

// PreHandler implements core.LifecycleHook
func (h *RequestAuthenticationHook) PreHandler(c *gin.Context) {
}

// OnResponse implements core.LifecycleHook
func (h *RequestAuthenticationHook) OnResponse(c *gin.Context, response interface{}) {
}

// OnError implements core.LifecycleHook
func (h *RequestAuthenticationHook) OnError(c *gin.Context, err error) {
}
