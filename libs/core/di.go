package core

import (
	"context"
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
	Provider Provider  // Changed from Factory
	Instance interface{} // Cached singleton instance
}

// DIContainer manages service registration and resolution
type DIContainer interface {
	// Legacy methods for backward compatibility
	Register(name string, factory Factory, lifetime Lifetime) error
	RegisterSingleton(name string, factory Factory) error
	RegisterTransient(name string, factory Factory) error
	RegisterScoped(name string, factory Factory) error

	// New provider-based methods
	RegisterProvider(provider Provider) error
	RegisterProviderSingleton(provider Provider) error
	RegisterProviderTransient(provider Provider) error
	RegisterProviderScoped(provider Provider) error

	// Resolution methods
	Resolve(name string) (interface{}, error)
	ResolveWithContext(name string, ctx context.Context) (interface{}, error)
	ResolveAs(name string, target interface{}) error
	ResolveAsWithContext(name string, ctx context.Context, target interface{}) error

	// Utility methods
	Has(name string) bool
	CreateScope() DIContainer

	// Module-scoped container creation
	CreateModuleScope(module *Module) DIContainer
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

// Register registers a service with the specified lifetime (backward compatibility)
func (c *diContainer) Register(name string, factory Factory, lifetime Lifetime) error {
	// Wrap Factory in FactoryProvider for backward compatibility
	provider := &FactoryProvider{
		Name:     name,
		Factory:  factory,
		Lifetime: lifetime,
	}
	return c.RegisterProvider(provider)
}

// RegisterProvider registers a provider (new primary method)
func (c *diContainer) RegisterProvider(provider Provider) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	name := provider.GetName()
	if _, exists := c.services[name]; exists {
		return fmt.Errorf("service '%s' is already registered", name)
	}

	if provider == nil {
		return fmt.Errorf("provider cannot be nil")
	}

	c.services[name] = &ServiceDefinition{
		Provider: provider,
	}

	return nil
}

// RegisterProviderSingleton registers a singleton provider
func (c *diContainer) RegisterProviderSingleton(provider Provider) error {
	// Create a wrapper provider with Singleton lifetime
	return c.RegisterProvider(&singletonLifetimeWrapper{Provider: provider})
}

// RegisterProviderTransient registers a transient provider
func (c *diContainer) RegisterProviderTransient(provider Provider) error {
	// Create a wrapper provider with Transient lifetime
	return c.RegisterProvider(&transientLifetimeWrapper{Provider: provider})
}

// RegisterProviderScoped registers a scoped provider
func (c *diContainer) RegisterProviderScoped(provider Provider) error {
	// Create a wrapper provider with Scoped lifetime
	return c.RegisterProvider(&scopedLifetimeWrapper{Provider: provider})
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
	return c.ResolveWithContext(name, context.Background())
}

// ResolveWithContext enables async resolution
func (c *diContainer) ResolveWithContext(name string, ctx context.Context) (interface{}, error) {
	c.mu.RLock()
	service, exists := c.services[name]
	c.mu.RUnlock()

	if !exists {
		// Check parent container if this is a scoped container
		if c.parent != nil {
			if parentWithCtx, ok := c.parent.(*diContainer); ok {
				return parentWithCtx.ResolveWithContext(name, ctx)
			}
			return c.parent.Resolve(name)
		}
		return nil, fmt.Errorf("service '%s' is not registered", name)
	}

	provider := service.Provider

	switch provider.GetLifetime() {
	case Singleton:
		if service.Instance != nil {
			return service.Instance, nil
		}

		// Create singleton instance
		instance, err := provider.Resolve(c, ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create singleton service '%s': %w", name, err)
		}

		// Store the instance
		c.mu.Lock()
		service.Instance = instance
		c.mu.Unlock()

		return instance, nil

	case Transient:
		return provider.Resolve(c, ctx)

	case Scoped:
		// For scoped services, always create a new instance in the current scope
		return provider.Resolve(c, ctx)

	default:
		return nil, fmt.Errorf("unknown lifetime for service '%s'", name)
	}
}

// ResolveAs resolves a service and assigns it to the target pointer
func (c *diContainer) ResolveAs(name string, target interface{}) error {
	return c.ResolveAsWithContext(name, context.Background(), target)
}

// ResolveAsWithContext resolves a service with context and assigns it to the target pointer
func (c *diContainer) ResolveAsWithContext(name string, ctx context.Context, target interface{}) error {
	instance, err := c.ResolveWithContext(name, ctx)
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

// Lifetime wrapper providers for RegisterProviderSingleton/Transient/Scoped

type singletonLifetimeWrapper struct {
	Provider
}

func (w *singletonLifetimeWrapper) GetLifetime() Lifetime {
	return Singleton
}

type transientLifetimeWrapper struct {
	Provider
}

func (w *transientLifetimeWrapper) GetLifetime() Lifetime {
	return Transient
}

type scopedLifetimeWrapper struct {
	Provider
}

func (w *scopedLifetimeWrapper) GetLifetime() Lifetime {
	return Scoped
}

// CreateModuleScope creates a new ModuleContainer for the given module
func (c *diContainer) CreateModuleScope(module *Module) DIContainer {
	return NewModuleContainer(module, c)
}
