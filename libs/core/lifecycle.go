package core

import (
	"github.com/gin-gonic/gin"
)

// LifecycleHook defines the interface for lifecycle hooks
type LifecycleHook interface {
	// OnRequest is called when a request is received
	OnRequest(c *gin.Context)
	// PreHandler is called before the route handler
	PreHandler(c *gin.Context)
	// OnResponse is called after the response is sent
	OnResponse(c *gin.Context, response interface{})
	// OnError is called when an error occurs
	OnError(c *gin.Context, err error)
}

// LifecycleHookFunc is a helper type to create hooks from functions
type LifecycleHookFunc struct {
	OnRequestFunc  func(c *gin.Context)
	PreHandlerFunc func(c *gin.Context)
	OnResponseFunc func(c *gin.Context, response interface{})
	OnErrorFunc    func(c *gin.Context, err error)
}

// OnRequest implements LifecycleHook
func (h *LifecycleHookFunc) OnRequest(c *gin.Context) {
	if h.OnRequestFunc != nil {
		h.OnRequestFunc(c)
	}
}

// PreHandler implements LifecycleHook
func (h *LifecycleHookFunc) PreHandler(c *gin.Context) {
	if h.PreHandlerFunc != nil {
		h.PreHandlerFunc(c)
	}
}

// OnResponse implements LifecycleHook
func (h *LifecycleHookFunc) OnResponse(c *gin.Context, response interface{}) {
	if h.OnResponseFunc != nil {
		h.OnResponseFunc(c, response)
	}
}

// OnError implements LifecycleHook
func (h *LifecycleHookFunc) OnError(c *gin.Context, err error) {
	if h.OnErrorFunc != nil {
		h.OnErrorFunc(c, err)
	}
}

// LifecycleManager manages the execution of lifecycle hooks
type LifecycleManager struct {
	hooks []LifecycleHook
}

// NewLifecycleManager creates a new lifecycle manager
func NewLifecycleManager() *LifecycleManager {
	return &LifecycleManager{
		hooks: make([]LifecycleHook, 0),
	}
}

// AddHook adds a lifecycle hook
func (lm *LifecycleManager) AddHook(hook LifecycleHook) {
	if hook != nil {
		lm.hooks = append(lm.hooks, hook)
	}
}

// ExecuteOnRequest executes all OnRequest hooks
func (lm *LifecycleManager) ExecuteOnRequest(c *gin.Context) {
	for _, hook := range lm.hooks {
		hook.OnRequest(c)
		if c.IsAborted() {
			return
		}
	}
}

// ExecutePreHandler executes all PreHandler hooks
func (lm *LifecycleManager) ExecutePreHandler(c *gin.Context) {
	for _, hook := range lm.hooks {
		hook.PreHandler(c)
		if c.IsAborted() {
			return
		}
	}
}

// ExecuteOnResponse executes all OnResponse hooks
func (lm *LifecycleManager) ExecuteOnResponse(c *gin.Context, response interface{}) {
	for _, hook := range lm.hooks {
		hook.OnResponse(c, response)
	}
}

// ExecuteOnError executes all OnError hooks
func (lm *LifecycleManager) ExecuteOnError(c *gin.Context, err error) {
	for _, hook := range lm.hooks {
		hook.OnError(c, err)
	}
}

// Helper functions to create specific hooks

// NewOnRequestHook creates a hook that only implements OnRequest
func NewOnRequestHook(fn func(c *gin.Context)) LifecycleHook {
	return &LifecycleHookFunc{
		OnRequestFunc: fn,
	}
}

// NewPreHandlerHook creates a hook that only implements PreHandler
func NewPreHandlerHook(fn func(c *gin.Context)) LifecycleHook {
	return &LifecycleHookFunc{
		PreHandlerFunc: fn,
	}
}

// NewOnResponseHook creates a hook that only implements OnResponse
func NewOnResponseHook(fn func(c *gin.Context, response interface{})) LifecycleHook {
	return &LifecycleHookFunc{
		OnResponseFunc: fn,
	}
}

// NewOnErrorHook creates a hook that only implements OnError
func NewOnErrorHook(fn func(c *gin.Context, err error)) LifecycleHook {
	return &LifecycleHookFunc{
		OnErrorFunc: fn,
	}
}
