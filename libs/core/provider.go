package core

import (
	"context"
	"fmt"
	"reflect"
	"time"
)

// Provider defines the interface for all provider types
type Provider interface {
	// GetName returns the service name for DI resolution
	GetName() string

	// GetLifetime returns the service lifetime (Singleton/Transient/Scoped)
	GetLifetime() Lifetime

	// Resolve creates the service instance
	// ctx enables cancellation for async providers
	Resolve(container DIContainer, ctx context.Context) (interface{}, error)

	// IsAsync indicates if this provider requires async initialization
	IsAsync() bool
}

// FactoryProvider wraps existing Factory functions (backward compatible)
type FactoryProvider struct {
	Name     string
	Factory  Factory  // Existing func(DIContainer) (interface{}, error)
	Lifetime Lifetime
}

func (p *FactoryProvider) GetName() string { return p.Name }
func (p *FactoryProvider) GetLifetime() Lifetime { return p.Lifetime }
func (p *FactoryProvider) IsAsync() bool { return false }
func (p *FactoryProvider) Resolve(container DIContainer, ctx context.Context) (interface{}, error) {
	return p.Factory(container)
}

// NewFactoryProvider creates a new FactoryProvider
func NewFactoryProvider(name string, factory Factory, lifetime Lifetime) *FactoryProvider {
	return &FactoryProvider{
		Name:     name,
		Factory:  factory,
		Lifetime: lifetime,
	}
}

// ClassProvider creates instances via reflection (struct type)
type ClassProvider struct {
	Name     string
	Type     reflect.Type  // e.g., reflect.TypeOf((*UserService)(nil)).Elem()
	Lifetime Lifetime
}

func (p *ClassProvider) GetName() string { return p.Name }
func (p *ClassProvider) GetLifetime() Lifetime { return p.Lifetime }
func (p *ClassProvider) IsAsync() bool { return false }
func (p *ClassProvider) Resolve(container DIContainer, ctx context.Context) (interface{}, error) {
	// Phase 2: Simple struct instantiation
	// Phase 3: Add constructor injection via reflection

	// Check if we have a pointer type (most common case for services)
	if p.Type.Kind() == reflect.Ptr {
		// For pointer types, create a new instance each time
		instance := reflect.New(p.Type.Elem())
		return instance.Interface(), nil
	}

	// For struct types (non-pointer), create a new instance
	if p.Type.Kind() == reflect.Struct {
		instance := reflect.New(p.Type)
		return instance.Interface(), nil
	}

	// For interface types, we need a concrete implementation
	return nil, fmt.Errorf("cannot create instance of interface type %s, use a concrete type", p.Type)
}

// NewClassProvider creates a new ClassProvider
func NewClassProvider(name string, typ reflect.Type, lifetime Lifetime) *ClassProvider {
	return &ClassProvider{
		Name:     name,
		Type:     typ,
		Lifetime: lifetime,
	}
}

// NewClassProviderByType creates a ClassProvider from a type parameter
func NewClassProviderByType[T any](name string, lifetime Lifetime) *ClassProvider {
	var zero T
	typ := reflect.TypeOf(zero)

	// If we have a pointer type, store the pointer type (we'll create new pointers in Resolve)
	// This ensures transient providers create new instances each time
	return &ClassProvider{
		Name:     name,
		Type:     typ,
		Lifetime: lifetime,
	}
}

// ValueProvider registers pre-instantiated values
type ValueProvider struct {
	Name  string
	Value interface{}
}

func (p *ValueProvider) GetName() string { return p.Name }
func (p *ValueProvider) GetLifetime() Lifetime { return Singleton }
func (p *ValueProvider) IsAsync() bool { return false }
func (p *ValueProvider) Resolve(container DIContainer, ctx context.Context) (interface{}, error) {
	return p.Value, nil
}

// NewValueProvider creates a new ValueProvider
func NewValueProvider(name string, value interface{}) *ValueProvider {
	return &ValueProvider{
		Name:  name,
		Value: value,
	}
}

// AsyncFactory creates services with async initialization
type AsyncFactory func(container DIContainer, ctx context.Context) (interface{}, error)

// AsyncProvider for services requiring async setup (DB, external APIs)
type AsyncProvider struct {
	Name     string
	Factory  AsyncFactory
	Lifetime Lifetime
	Timeout  time.Duration  // Default 30s if not set
}

func (p *AsyncProvider) GetName() string { return p.Name }
func (p *AsyncProvider) GetLifetime() Lifetime { return p.Lifetime }
func (p *AsyncProvider) IsAsync() bool { return true }
func (p *AsyncProvider) Resolve(container DIContainer, ctx context.Context) (interface{}, error) {
	timeout := p.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return p.Factory(container, ctx)
}

// NewAsyncProvider creates a new AsyncProvider with default timeout
func NewAsyncProvider(name string, factory AsyncFactory, lifetime Lifetime) *AsyncProvider {
	return &AsyncProvider{
		Name:     name,
		Factory:  factory,
		Lifetime: lifetime,
		Timeout:  30 * time.Second,
	}
}

// NewAsyncProviderWithTimeout creates a new AsyncProvider with custom timeout
func NewAsyncProviderWithTimeout(name string, factory AsyncFactory, lifetime Lifetime, timeout time.Duration) *AsyncProvider {
	return &AsyncProvider{
		Name:     name,
		Factory:  factory,
		Lifetime: lifetime,
		Timeout:  timeout,
	}
}