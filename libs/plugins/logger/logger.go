package logger

import (
	"fmt"
	"github.com/dangvanduc1999/doffy-go-boostrap/libs/core"
	"time"

	"github.com/gin-gonic/gin"
)

// LoggerPlugin implements the Plugin interface for request logging
type LoggerPlugin struct {
	core.BasePlugin
}

// NewLoggerPlugin creates a new logger plugin
func NewLoggerPlugin() *LoggerPlugin {
	return &LoggerPlugin{}
}

// Name returns the plugin name
func (p *LoggerPlugin) Name() string {
	return "logger"
}

// Version returns the plugin version
func (p *LoggerPlugin) Version() string {
	return "1.0.0"
}

// Register registers the logger service with the DI container
func (p *LoggerPlugin) Register(container core.DIContainer) error {
	return container.RegisterSingleton("requestLogger", func(c core.DIContainer) (interface{}, error) {
		logger, _ := c.Resolve("logger")
		return NewRequestLogger(logger.(core.Logger)), nil
	})
}

// Hooks returns the lifecycle hooks for logging
func (p *LoggerPlugin) Hooks() []core.LifecycleHook {
	return []core.LifecycleHook{
		NewLoggerHook(),
	}
}

// RequestLogger provides request logging functionality
type RequestLogger struct {
	logger core.Logger
}

// NewRequestLogger creates a new request logger
func NewRequestLogger(logger core.Logger) *RequestLogger {
	return &RequestLogger{
		logger: logger,
	}
}

// LogRequest logs a request
func (l *RequestLogger) LogRequest(c *gin.Context, start time.Time) {
	duration := time.Since(start)

	l.logger.Infor(&core.LoggerItem{
		Event:    "Request",
		Messages: fmt.Sprintf("%s %s", c.Request.Method, c.Request.URL.Path),
		Data: struct {
			Method     string        `json:"method"`
			Path       string        `json:"path"`
			StatusCode int           `json:"status_code"`
			Duration   time.Duration `json:"duration"`
			ClientIP   string        `json:"client_ip"`
			UserAgent  string        `json:"user_agent"`
		}{
			Method:     c.Request.Method,
			Path:       c.Request.URL.Path,
			StatusCode: c.Writer.Status(),
			Duration:   duration,
			ClientIP:   c.ClientIP(),
			UserAgent:  c.GetHeader("User-Agent"),
		},
	})
}

// LoggerHook implements the LifecycleHook interface for request logging
type LoggerHook struct {
	requestLogger *RequestLogger
}

// NewLoggerHook creates a new logger hook
func NewLoggerHook() *LoggerHook {
	return &LoggerHook{}
}

// OnRequest implements the LifecycleHook interface
func (h *LoggerHook) OnRequest(c *gin.Context) {
	// Store start time in context
	c.Set("start_time", time.Now())
	c.Next()
}

// PreHandler implements the LifecycleHook interface
func (h *LoggerHook) PreHandler(c *gin.Context) {
	// No pre-handler logic needed for request logging
}

// OnResponse implements the LifecycleHook interface
func (h *LoggerHook) OnResponse(c *gin.Context, response interface{}) {
	// Log the request after it's processed
	if startTime, exists := c.Get("start_time"); exists {
		if start, ok := startTime.(time.Time); ok {
			// Get request logger from container
			requestLogger, err := c.MustGet("container").(core.DIContainer).Resolve("requestLogger")
			if err == nil {
				if logger, ok := requestLogger.(*RequestLogger); ok {
					logger.LogRequest(c, start)
				}
			}
		}
	}
}

// OnError implements the LifecycleHook interface
func (h *LoggerHook) OnError(c *gin.Context, err error) {
	// Log the error
	logger, _ := c.MustGet("container").(core.DIContainer).Resolve("logger")
	if l, ok := logger.(core.Logger); ok {
		l.Infor(&core.LoggerItem{
			Event:    "Error",
			Messages: fmt.Sprintf("Error handling %s %s: %v", c.Request.Method, c.Request.URL.Path, err),
			Error:    err,
		})
	}
}
