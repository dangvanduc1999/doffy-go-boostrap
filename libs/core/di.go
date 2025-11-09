package core

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
)

// Lifetime defines the lifetime of a service in the DI container
type Lifetime int

const (
	// Singleton creates one instance for the entire application lifetime
	Singleton Lifetime = iota
	// Transient creates a new instance every time it's requested
	Transient
	// Scoped creates one instance per request/scope
	Scoped
)

// Factory is a function that creates a service instance
type Factory func(container DIContainer) (interface{}, error)

// ServiceDefinition holds information about a registered service
type ServiceDefinition struct {
	Factory  Factory
	Lifetime Lifetime
	Instance interface{} // For singleton instances
}

// DIContainer manages service registration and resolution
type DIContainer interface {
	Register(name string, factory Factory, lifetime Lifetime) error
	RegisterSingleton(name string, factory Factory) error
	RegisterTransient(name string, factory Factory) error
	RegisterScoped(name string, factory Factory) error
	Resolve(name string) (interface{}, error)
	ResolveAs(name string, target interface{}) error
	Has(name string) bool
	CreateScope() DIContainer
}

// diContainer is the default implementation of DIContainer
type diContainer struct {
	services map[string]*ServiceDefinition
	mu       sync.RWMutex
	parent   DIContainer // For scoped containers
}

// NewDIContainer creates a new dependency injection container
func NewDIContainer() DIContainer {
	return &diContainer{
		services: make(map[string]*ServiceDefinition),
	}
}

// Register registers a service with the specified lifetime
func (c *diContainer) Register(name string, factory Factory, lifetime Lifetime) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.services[name]; exists {
		return fmt.Errorf("service '%s' is already registered", name)
	}

	if factory == nil {
		return fmt.Errorf("factory cannot be nil for service '%s'", name)
	}

	c.services[name] = &ServiceDefinition{
		Factory:  factory,
		Lifetime: lifetime,
	}

	return nil
}

// RegisterSingleton registers a singleton service
func (c *diContainer) RegisterSingleton(name string, factory Factory) error {
	return c.Register(name, factory, Singleton)
}

// RegisterTransient registers a transient service
func (c *diContainer) RegisterTransient(name string, factory Factory) error {
	return c.Register(name, factory, Transient)
}

// RegisterScoped registers a scoped service
func (c *diContainer) RegisterScoped(name string, factory Factory) error {
	return c.Register(name, factory, Scoped)
}

// Resolve resolves a service by name
func (c *diContainer) Resolve(name string) (interface{}, error) {
	c.mu.RLock()
	service, exists := c.services[name]
	c.mu.RUnlock()

	if !exists {
		// Check parent container if this is a scoped container
		if c.parent != nil {
			return c.parent.Resolve(name)
		}
		return nil, fmt.Errorf("service '%s' is not registered", name)
	}

	switch service.Lifetime {
	case Singleton:
		if service.Instance != nil {
			return service.Instance, nil
		}

		// Create singleton instance
		instance, err := service.Factory(c)
		if err != nil {
			return nil, fmt.Errorf("failed to create singleton service '%s': %w", name, err)
		}

		// Store the instance
		c.mu.Lock()
		service.Instance = instance
		c.mu.Unlock()

		return instance, nil

	case Transient:
		return service.Factory(c)

	case Scoped:
		// For scoped services, always create a new instance in the current scope
		return service.Factory(c)

	default:
		return nil, fmt.Errorf("unknown lifetime for service '%s'", name)
	}
}

// ResolveAs resolves a service and assigns it to the target pointer
func (c *diContainer) ResolveAs(name string, target interface{}) error {
	instance, err := c.Resolve(name)
	if err != nil {
		return err
	}

	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() != reflect.Ptr {
		return errors.New("target must be a pointer")
	}

	instanceValue := reflect.ValueOf(instance)
	if !instanceValue.Type().AssignableTo(targetValue.Elem().Type()) {
		return fmt.Errorf("service '%s' cannot be assigned to target type", name)
	}

	targetValue.Elem().Set(instanceValue)
	return nil
}

// Has checks if a service is registered
func (c *diContainer) Has(name string) bool {
	c.mu.RLock()
	_, exists := c.services[name]
	c.mu.RUnlock()

	if !exists && c.parent != nil {
		return c.parent.Has(name)
	}

	return exists
}

// CreateScope creates a new scoped container
func (c *diContainer) CreateScope() DIContainer {
	return &diContainer{
		services: make(map[string]*ServiceDefinition),
		parent:   c,
	}
}
