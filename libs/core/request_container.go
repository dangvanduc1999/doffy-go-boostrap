package core

import (
	"context"
	"fmt"
	"sync"
)

// RequestContainer is a per-request scoped DI container
type RequestContainer struct {
	*diContainer  // Embed base container

	module       DIContainer
	requestData  map[string]interface{}  // Request decorators
	replyHelpers map[string]interface{}  // Reply decorators
	mu           sync.RWMutex
}

// NewRequestContainer creates a request-scoped container
func NewRequestContainer(moduleContainer DIContainer) *RequestContainer {
	return &RequestContainer{
		diContainer: &diContainer{
			services: make(map[string]*ServiceDefinition),
			parent:   moduleContainer,
		},
		module:       moduleContainer,
		requestData:  make(map[string]interface{}),
		replyHelpers: make(map[string]interface{}),
	}
}

// DecorateRequest adds request-scoped data
func (rc *RequestContainer) DecorateRequest(name string, value interface{}) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.requestData[name] = value
}

// GetRequestData retrieves request-scoped data
func (rc *RequestContainer) GetRequestData(name string) (interface{}, bool) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	value, exists := rc.requestData[name]
	return value, exists
}

// DecorateReply adds reply helper function
func (rc *RequestContainer) DecorateReply(name string, fn interface{}) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.replyHelpers[name] = fn
}

// GetReplyHelper retrieves reply helper function
func (rc *RequestContainer) GetReplyHelper(name string) (interface{}, bool) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	helper, exists := rc.replyHelpers[name]
	return helper, exists
}

// GetModule returns the module container that created this request container
func (rc *RequestContainer) GetModule() DIContainer {
	return rc.module
}

// Resolve resolves a service by name using request-scoped resolution
func (rc *RequestContainer) Resolve(name string) (interface{}, error) {
	return rc.ResolveWithContext(name, context.Background())
}

// ResolveWithContext overrides parent resolution to check request data first
func (rc *RequestContainer) ResolveWithContext(name string, ctx context.Context) (interface{}, error) {
	// Check request-scoped data first
	if value, exists := rc.GetRequestData(name); exists {
		return value, nil
	}

	// Check reply helpers
	if helper, exists := rc.GetReplyHelper(name); exists {
		return helper, nil
	}

	// Fall back to parent resolution
	rc.mu.RLock()
	service, exists := rc.services[name]
	rc.mu.RUnlock()

	if exists {
		provider := service.Provider

		switch provider.GetLifetime() {
		case Singleton:
			// For request containers, we don't cache singletons
			// Each request should get a fresh instance if requested
			return provider.Resolve(rc, ctx)

		case Transient:
			return provider.Resolve(rc, ctx)

		case Scoped:
			// For request containers, scoped means "per request"
			// So we always create a new instance
			return provider.Resolve(rc, ctx)

		default:
			return nil, fmt.Errorf("unknown lifetime for service '%s'", name)
		}
	}

	// Check parent container (module container)
	if rc.module != nil {
		if moduleWithCtx, ok := rc.module.(interface{ ResolveWithContext(string, context.Context) (interface{}, error) }); ok {
			return moduleWithCtx.ResolveWithContext(name, ctx)
		}
		return rc.module.Resolve(name)
	}

	return nil, fmt.Errorf("service '%s' is not registered", name)
}

// Clear clears all request-scoped data (useful for cleanup)
func (rc *RequestContainer) Clear() {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	// Clear request data
	for key := range rc.requestData {
		delete(rc.requestData, key)
	}

	// Clear reply helpers
	for key := range rc.replyHelpers {
		delete(rc.replyHelpers, key)
	}
}

// Size returns the number of registered decorators
func (rc *RequestContainer) Size() (requestCount int, replyCount int) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	return len(rc.requestData), len(rc.replyHelpers)
}

// ListRequestData returns all request-scoped data keys
func (rc *RequestContainer) ListRequestData() []string {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	keys := make([]string, 0, len(rc.requestData))
	for key := range rc.requestData {
		keys = append(keys, key)
	}
	return keys
}

// ListReplyHelpers returns all reply helper keys
func (rc *RequestContainer) ListReplyHelpers() []string {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	keys := make([]string, 0, len(rc.replyHelpers))
	for key := range rc.replyHelpers {
		keys = append(keys, key)
	}
	return keys
}