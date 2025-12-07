package core

import (
	"fmt"
	"sync"
)

// DecoratorManager manages application-level decorators
type DecoratorManager struct {
	instanceDecorators map[string]interface{}
	requestDecorators  map[string]interface{}  // Default values
	replyDecorators    map[string]interface{}
	mu                 sync.RWMutex
}

// NewDecoratorManager creates a new decorator manager
func NewDecoratorManager() *DecoratorManager {
	return &DecoratorManager{
		instanceDecorators: make(map[string]interface{}),
		requestDecorators:  make(map[string]interface{}),
		replyDecorators:    make(map[string]interface{}),
	}
}

// Decorate registers an instance-level decorator
func (dm *DecoratorManager) Decorate(name string, value interface{}) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if _, exists := dm.instanceDecorators[name]; exists {
		return fmt.Errorf("decorator '%s' already registered", name)
	}

	dm.instanceDecorators[name] = value
	return nil
}

// DecorateRequest registers a request-scoped decorator with default value
func (dm *DecoratorManager) DecorateRequest(name string, defaultValue interface{}) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if _, exists := dm.requestDecorators[name]; exists {
		return fmt.Errorf("request decorator '%s' already registered", name)
	}

	dm.requestDecorators[name] = defaultValue
	return nil
}

// DecorateReply registers a reply helper function
func (dm *DecoratorManager) DecorateReply(name string, fn interface{}) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if _, exists := dm.replyDecorators[name]; exists {
		return fmt.Errorf("reply decorator '%s' already registered", name)
	}

	dm.replyDecorators[name] = fn
	return nil
}

// GetInstanceDecorator retrieves an instance decorator
func (dm *DecoratorManager) GetInstanceDecorator(name string) (interface{}, bool) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	value, exists := dm.instanceDecorators[name]
	return value, exists
}

// GetRequestDecorator retrieves a request decorator default value
func (dm *DecoratorManager) GetRequestDecorator(name string) (interface{}, bool) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	value, exists := dm.requestDecorators[name]
	return value, exists
}

// GetReplyDecorator retrieves a reply helper function
func (dm *DecoratorManager) GetReplyDecorator(name string) (interface{}, bool) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	value, exists := dm.replyDecorators[name]
	return value, exists
}

// InitializeRequestContainer initializes request decorators in container
func (dm *DecoratorManager) InitializeRequestContainer(rc *RequestContainer) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	for name, defaultValue := range dm.requestDecorators {
	// Only set if not already present
		if _, exists := rc.GetRequestData(name); !exists {
			rc.DecorateRequest(name, defaultValue)
		}
	}
}

// InitializeReplyHelpers initializes reply decorators in container
func (dm *DecoratorManager) InitializeReplyHelpers(rc *RequestContainer) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	for name, fn := range dm.replyDecorators {
	// Only set if not already present
		if _, exists := rc.GetReplyHelper(name); !exists {
			rc.DecorateReply(name, fn)
		}
	}
}

// RemoveInstanceDecorator removes an instance decorator
func (dm *DecoratorManager) RemoveInstanceDecorator(name string) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	delete(dm.instanceDecorators, name)
}

// RemoveRequestDecorator removes a request decorator
func (dm *DecoratorManager) RemoveRequestDecorator(name string) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	delete(dm.requestDecorators, name)
}

// RemoveReplyDecorator removes a reply decorator
func (dm *DecoratorManager) RemoveReplyDecorator(name string) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	delete(dm.replyDecorators, name)
}

// ClearInstanceDecorators clears all instance decorators
func (dm *DecoratorManager) ClearInstanceDecorators() {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	dm.instanceDecorators = make(map[string]interface{})
}

// ClearRequestDecorators clears all request decorators
func (dm *DecoratorManager) ClearRequestDecorators() {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	dm.requestDecorators = make(map[string]interface{})
}

// ClearReplyDecorators clears all reply decorators
func (dm *DecoratorManager) ClearReplyDecorators() {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	dm.replyDecorators = make(map[string]interface{})
}

// GetDecoratorStats returns statistics about registered decorators
func (dm *DecoratorManager) GetDecoratorStats() (instanceCount int, requestCount int, replyCount int) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	return len(dm.instanceDecorators), len(dm.requestDecorators), len(dm.replyDecorators)
}

// ListInstanceDecorators returns all instance decorator names
func (dm *DecoratorManager) ListInstanceDecorators() []string {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	names := make([]string, 0, len(dm.instanceDecorators))
	for name := range dm.instanceDecorators {
		names = append(names, name)
	}
	return names
}

// ListRequestDecorators returns all request decorator names
func (dm *DecoratorManager) ListRequestDecorators() []string {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	names := make([]string, 0, len(dm.requestDecorators))
	for name := range dm.requestDecorators {
		names = append(names, name)
	}
	return names
}

// ListReplyDecorators returns all reply decorator names
func (dm *DecoratorManager) ListReplyDecorators() []string {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	names := make([]string, 0, len(dm.replyDecorators))
	for name := range dm.replyDecorators {
		names = append(names, name)
	}
	return names
}