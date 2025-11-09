package core

import (
	"fmt"
	"reflect"
	"sync"
)

// ServiceLocator provides a global way to access services
type ServiceLocator interface {
	GetContainer() DIContainer
	SetContainer(container DIContainer)
	GetByType(serviceType reflect.Type) (interface{}, error)
	GetService(name string) (interface{}, error)
	GetServiceAs(name string, target interface{}) error
	Has(name string) bool
}

// serviceLocator implements ServiceLocator
type serviceLocator struct {
	container DIContainer
	mu        sync.RWMutex
}

// Global service locator instance
var GlobalLocator ServiceLocator = &serviceLocator{}

func init() {
	GlobalLocator = &serviceLocator{}
}

// GetContainer returns the DI container
func (sl *serviceLocator) GetContainer() DIContainer {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	return sl.container
}

// SetContainer sets the DI container
func (sl *serviceLocator) SetContainer(container DIContainer) {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	sl.container = container
}

// GetByType retrieves a service by type using reflection
func (sl *serviceLocator) GetByType(serviceType reflect.Type) (interface{}, error) {
	sl.mu.RLock()
	defer sl.mu.RUnlock()

	if sl.container == nil {
		return nil, fmt.Errorf("container not set")
	}

	// Get the type name
	typeName := serviceType.String()

	// Try to resolve by type name
	service, err := sl.container.Resolve(typeName)
	if err != nil {
		// If type name resolution fails, try with a generic naming convention
		// For example: *UserService -> userService
		typeName = toServiceName(serviceType)
		service, err = sl.container.Resolve(typeName)
		if err != nil {
			return nil, err
		}
	}

	return service, nil
}

// GetService retrieves a service by name
func (sl *serviceLocator) GetService(name string) (interface{}, error) {
	sl.mu.RLock()
	defer sl.mu.RUnlock()

	if sl.container == nil {
		return nil, fmt.Errorf("container not set")
	}

	return sl.container.Resolve(name)
}

// GetServiceAs retrieves a service by name and assigns it to the target pointer
func (sl *serviceLocator) GetServiceAs(name string, target interface{}) error {
	sl.mu.RLock()
	defer sl.mu.RUnlock()

	if sl.container == nil {
		return fmt.Errorf("container not set")
	}

	return sl.container.ResolveAs(name, target)
}

// Has checks if a service exists
func (sl *serviceLocator) Has(name string) bool {
	sl.mu.RLock()
	defer sl.mu.RUnlock()

	if sl.container == nil {
		return false
	}

	return sl.container.Has(name)
}

// Helper functions for global access

// GetService retrieves a service by type using generics from the global locator
func GetService[T any]() T {
	var t T
	serviceType := reflect.TypeOf(t)
	service, err := GlobalLocator.GetByType(serviceType)
	if err != nil {
		var zero T
		return zero
	}

	if service, ok := service.(T); ok {
		return service
	}

	var zero T
	return zero
}

// GetServiceByName retrieves a service by name from the global locator
func GetServiceByName(name string) (interface{}, error) {
	return GlobalLocator.GetService(name)
}

// GetServiceByNameAs retrieves a service by name and assigns it to the target pointer from the global locator
func GetServiceByNameAs(name string, target interface{}) error {
	return GlobalLocator.GetServiceAs(name, target)
}

// HasService checks if a service exists in the global locator
func HasService(name string) bool {
	return GlobalLocator.Has(name)
}

// SetGlobalContainer sets the container in the global locator
func SetGlobalContainer(container DIContainer) {
	GlobalLocator.SetContainer(container)
}

// toServiceName converts a type to a service name
func toServiceName(t reflect.Type) string {
	// If it's a pointer, get the element type
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Keep the original type name (e.g., UserController)
	return t.Name()
}

// RegisterByType registers a service by its type for easier resolution
func RegisterByType[T any](container DIContainer, factory Factory, lifetime Lifetime) error {
	var t T
	typeName := reflect.TypeOf(t).String()
	return container.Register(typeName, factory, lifetime)
}

// RegisterSingletonByType registers a singleton service by its type
func RegisterSingletonByType[T any](container DIContainer, factory Factory) error {
	return RegisterByType[T](container, factory, Singleton)
}

// RegisterTransientByType registers a transient service by its type
func RegisterTransientByType[T any](container DIContainer, factory Factory) error {
	return RegisterByType[T](container, factory, Transient)
}

// RegisterScopedByType registers a scoped service by its type
func RegisterScopedByType[T any](container DIContainer, factory Factory) error {
	return RegisterByType[T](container, factory, Scoped)
}
