package core

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// CorsOptions defines CORS configuration
type CorsOptions struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           int
}

type CorsPlugin struct {
	BasePlugin
	options interface{}
}

// NewCorsPlugin creates a new CORS plugin
func NewCorsPlugin(options interface{}) *CorsPlugin {
	return &CorsPlugin{
		options: options,
	}
}

// Name returns the plugin name
func (p *CorsPlugin) Name() string {
	return "cors"
}

// Version returns the plugin version
func (p *CorsPlugin) Version() string {
	return "1.0.0"
}

// Register registers the CORS service with the DI container
func (p *CorsPlugin) Register(container DIContainer) error {
	return container.RegisterSingleton("corsService", func(c DIContainer) (interface{}, error) {
		return NewCorsService(p.options), nil
	})
}

// Hooks returns the lifecycle hooks for CORS
func (p *CorsPlugin) Hooks() []LifecycleHook {
	return []LifecycleHook{
		NewCorsHook(),
	}
}

// CorsService provides CORS functionality
type CorsService struct {
	options *CorsOptions
}

// NewCorsService creates a new CORS service
func NewCorsService(options interface{}) *CorsService {
	var corsOptions *CorsOptions

	if options != nil {
		var ok bool
		corsOptions, ok = options.(*CorsOptions)
		if !ok {
			// Try to convert from map[string]interface{}
			if optMap, ok := options.(map[string]interface{}); ok {
				corsOptions = &CorsOptions{}
				if origins, ok := optMap["allowOrigins"].([]string); ok {
					corsOptions.AllowOrigins = origins
				}
				if methods, ok := optMap["allowMethods"].([]string); ok {
					corsOptions.AllowMethods = methods
				}
				if headers, ok := optMap["allowHeaders"].([]string); ok {
					corsOptions.AllowHeaders = headers
				}
				if exposeHeaders, ok := optMap["exposeHeaders"].([]string); ok {
					corsOptions.ExposeHeaders = exposeHeaders
				}
				if credentials, ok := optMap["allowCredentials"].(bool); ok {
					corsOptions.AllowCredentials = credentials
				}
				if maxAge, ok := optMap["maxAge"].(int); ok {
					corsOptions.MaxAge = maxAge
				}
			}
		}
	}

	defaultOptions := &CorsOptions{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{},
		AllowCredentials: false,
		MaxAge:           86400,
	}

	if corsOptions != nil {
		if len(corsOptions.AllowOrigins) > 0 {
			defaultOptions.AllowOrigins = corsOptions.AllowOrigins
		}
		if len(corsOptions.AllowMethods) > 0 {
			defaultOptions.AllowMethods = corsOptions.AllowMethods
		}
		if len(corsOptions.AllowHeaders) > 0 {
			defaultOptions.AllowHeaders = corsOptions.AllowHeaders
		}
		if len(corsOptions.ExposeHeaders) > 0 {
			defaultOptions.ExposeHeaders = corsOptions.ExposeHeaders
		}
		defaultOptions.AllowCredentials = corsOptions.AllowCredentials
		if corsOptions.MaxAge > 0 {
			defaultOptions.MaxAge = corsOptions.MaxAge
		}
	}

	return &CorsService{
		options: defaultOptions,
	}
}

// Handle handles the CORS middleware
func (s *CorsService) Handle(c *gin.Context) {
	c.Header("Access-Control-Allow-Origin", strings.Join(s.options.AllowOrigins, ","))
	c.Header("Access-Control-Allow-Methods", strings.Join(s.options.AllowMethods, ","))
	c.Header("Access-Control-Allow-Headers", strings.Join(s.options.AllowHeaders, ","))
	c.Header("Access-Control-Expose-Headers", strings.Join(s.options.ExposeHeaders, ","))
	if s.options.AllowCredentials {
		c.Header("Access-Control-Allow-Credentials", "true")
	}
	c.Header("Access-Control-Max-Age", strconv.Itoa(s.options.MaxAge))

	if c.Request.Method == "OPTIONS" {
		c.AbortWithStatus(204)
		return
	}

	c.Next()
}

// CorsHook implements the LifecycleHook interface for CORS
type CorsHook struct{}

// NewCorsHook creates a new CORS hook
func NewCorsHook() *CorsHook {
	return &CorsHook{}
}

// OnRequest implements the LifecycleHook interface
func (h *CorsHook) OnRequest(c *gin.Context) {
	// Get CORS service from container
	corsService, err := c.MustGet("container").(DIContainer).Resolve("corsService")
	if err != nil {
		c.Next()
		return
	}

	if service, ok := corsService.(*CorsService); ok {
		service.Handle(c)
	} else {
		c.Next()
	}
}

// PreHandler implements the LifecycleHook interface
func (h *CorsHook) PreHandler(c *gin.Context) {
	// No pre-handler logic needed for CORS
}

// OnResponse implements the LifecycleHook interface
func (h *CorsHook) OnResponse(c *gin.Context, response interface{}) {
	// No response logic needed for CORS
}

// OnError implements the LifecycleHook interface
func (h *CorsHook) OnError(c *gin.Context, err error) {
	// No error handling needed for CORS
}

// DefaultCors is kept for backward compatibility
func DefaultCors(instance *gin.Engine, corsOptions interface{}) gin.HandlerFunc {
	service := NewCorsService(corsOptions)
	return service.Handle
}
