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
	hooks    []LifecycleHook
	appHooks []ApplicationHook
}

// NewLifecycleManager creates a new lifecycle manager
func NewLifecycleManager() *LifecycleManager {
	return &LifecycleManager{
		hooks:    make([]LifecycleHook, 0),
		appHooks: make([]ApplicationHook, 0),
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

// ApplicationHook defines the interface for application-level lifecycle hooks
type ApplicationHook interface {
	// OnRoute is called when a route is registered
	OnRoute(config *RouteConfig)
	// OnRegister is called when a plugin is registered
	OnRegister(plugin interface{})
	// OnReady is called before the server starts listening
	OnReady(app interface{}) error
	// OnListen is called after the server starts listening
	OnListen(addr string)
	// PreClose is called before the server shuts down
	PreClose(ctx interface{})
	// OnClose is called after the server shuts down
	OnClose() error
}

// ApplicationHookFunc is a helper type to create application hooks from functions
type ApplicationHookFunc struct {
	OnRouteFunc    func(config *RouteConfig)
	OnRegisterFunc func(plugin interface{})
	OnReadyFunc    func(app interface{}) error
	OnListenFunc   func(addr string)
	PreCloseFunc   func(ctx interface{})
	OnCloseFunc    func() error
}

// OnRoute implements ApplicationHook
func (h *ApplicationHookFunc) OnRoute(config *RouteConfig) {
	if h.OnRouteFunc != nil {
		h.OnRouteFunc(config)
	}
}

// OnRegister implements ApplicationHook
func (h *ApplicationHookFunc) OnRegister(plugin interface{}) {
	if h.OnRegisterFunc != nil {
		h.OnRegisterFunc(plugin)
	}
}

// OnReady implements ApplicationHook
func (h *ApplicationHookFunc) OnReady(app interface{}) error {
	if h.OnReadyFunc != nil {
		return h.OnReadyFunc(app)
	}
	return nil
}

// OnListen implements ApplicationHook
func (h *ApplicationHookFunc) OnListen(addr string) {
	if h.OnListenFunc != nil {
		h.OnListenFunc(addr)
	}
}

// PreClose implements ApplicationHook
func (h *ApplicationHookFunc) PreClose(ctx interface{}) {
	if h.PreCloseFunc != nil {
		h.PreCloseFunc(ctx)
	}
}

// OnClose implements ApplicationHook
func (h *ApplicationHookFunc) OnClose() error {
	if h.OnCloseFunc != nil {
		return h.OnCloseFunc()
	}
	return nil
}

// AddAppHook adds an application lifecycle hook
func (lm *LifecycleManager) AddAppHook(hook ApplicationHook) {
	if hook != nil {
		lm.appHooks = append(lm.appHooks, hook)
	}
}

// ExecuteOnRoute executes all OnRoute hooks
func (lm *LifecycleManager) ExecuteOnRoute(config *RouteConfig) {
	for _, hook := range lm.appHooks {
		hook.OnRoute(config)
	}
}

// ExecuteOnRegister executes all OnRegister hooks
func (lm *LifecycleManager) ExecuteOnRegister(plugin interface{}) {
	for _, hook := range lm.appHooks {
		hook.OnRegister(plugin)
	}
}

// ExecuteOnReady executes all OnReady hooks
func (lm *LifecycleManager) ExecuteOnReady(app interface{}) error {
	for _, hook := range lm.appHooks {
		if err := hook.OnReady(app); err != nil {
			return err
		}
	}
	return nil
}

// ExecuteOnListen executes all OnListen hooks
func (lm *LifecycleManager) ExecuteOnListen(addr string) {
	for _, hook := range lm.appHooks {
		hook.OnListen(addr)
	}
}

// ExecutePreClose executes all PreClose hooks
func (lm *LifecycleManager) ExecutePreClose(ctx interface{}) {
	for _, hook := range lm.appHooks {
		hook.PreClose(ctx)
	}
}

// ExecuteOnClose executes all OnClose hooks
func (lm *LifecycleManager) ExecuteOnClose() error {
	var lastErr error
	for _, hook := range lm.appHooks {
		if err := hook.OnClose(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}
