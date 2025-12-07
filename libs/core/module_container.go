package core

import (
	"context"
	"fmt"
	"sync"
)

// ModuleContainer is a scoped DI container for a module
type ModuleContainer struct {
	*diContainer  // Embed base container

	module       *Module
	parent       DIContainer
	children     map[string]*ModuleContainer
	decorators   map[string]interface{}  // Instance decorators
	mu           sync.RWMutex
}

// NewModuleContainer creates a scoped container for a module
func NewModuleContainer(module *Module, parent DIContainer) *ModuleContainer {
	return &ModuleContainer{
		diContainer: &diContainer{
			services: make(map[string]*ServiceDefinition),
			parent:   parent,
		},
		module:     module,
		parent:     parent,
		children:   make(map[string]*ModuleContainer),
		decorators: make(map[string]interface{}),
	}
}

// Decorate adds an instance-level decorator
func (mc *ModuleContainer) Decorate(name string, value interface{}) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if _, exists := mc.decorators[name]; exists {
		return fmt.Errorf("decorator '%s' already exists", name)
	}

	mc.decorators[name] = value
	return nil
}

// GetDecorator retrieves an instance decorator
func (mc *ModuleContainer) GetDecorator(name string) (interface{}, bool) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if value, exists := mc.decorators[name]; exists {
		return value, true
	}

	// Check parent container
	if parent, ok := mc.parent.(*ModuleContainer); ok {
		return parent.GetDecorator(name)
	}

	return nil, false
}

// CreateRequestScope creates a request-scoped container
func (mc *ModuleContainer) CreateRequestScope() *RequestContainer {
	return NewRequestContainer(mc)
}

// GetModule returns the module associated with this container
func (mc *ModuleContainer) GetModule() *Module {
	return mc.module
}

// GetParent returns the parent container
func (mc *ModuleContainer) GetParent() DIContainer {
	return mc.parent
}

// AddChild adds a child module container
func (mc *ModuleContainer) AddChild(name string, child *ModuleContainer) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.children[name] = child
}

// GetChild retrieves a child module container by name
func (mc *ModuleContainer) GetChild(name string) (*ModuleContainer, bool) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	child, exists := mc.children[name]
	return child, exists
}

// GetAllChildren returns all child containers
func (mc *ModuleContainer) GetAllChildren() map[string]*ModuleContainer {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	result := make(map[string]*ModuleContainer)
	for name, child := range mc.children {
		result[name] = child
	}
	return result
}

// ResolveWithContext overrides parent resolution to check decorators first
func (mc *ModuleContainer) ResolveWithContext(name string, ctx context.Context) (interface{}, error) {
	// Check decorators first
	if value, exists := mc.GetDecorator(name); exists {
		return value, nil
	}

	// Fall back to parent resolution
	mc.mu.RLock()
	service, exists := mc.services[name]
	mc.mu.RUnlock()

	if exists {
		provider := service.Provider

		switch provider.GetLifetime() {
		case Singleton:
			if service.Instance != nil {
				return service.Instance, nil
			}

			// Create singleton instance
			instance, err := provider.Resolve(mc, ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to create singleton service '%s': %w", name, err)
			}

			// Store the instance
			mc.mu.Lock()
			service.Instance = instance
			mc.mu.Unlock()

			return instance, nil

		case Transient:
			return provider.Resolve(mc, ctx)

		case Scoped:
			// For scoped services, always create a new instance
			return provider.Resolve(mc, ctx)

		default:
			return nil, fmt.Errorf("unknown lifetime for service '%s'", name)
		}
	}

	// Check parent container
	if mc.parent != nil {
		// If parent is another ModuleContainer, check encapsulation
		if parentModule, ok := mc.parent.(*ModuleContainer); ok {
			// Skip validation if either module is Global
			if !mc.module.Global && !parentModule.module.Global {
				// Check if the service is exported by parent module
				if !parentModule.module.IsExported(name) {
					// Check encapsulation mode
					allowed, err := CheckEncapsulationViolation(
						mc.module.Name,
						parentModule.module.Name,
						name,
					)
					if !allowed {
						return nil, err
					}
				}
			}
		}

		if parentWithCtx, ok := mc.parent.(interface{ ResolveWithContext(string, context.Context) (interface{}, error) }); ok {
			return parentWithCtx.ResolveWithContext(name, ctx)
		}
		return mc.parent.Resolve(name)
	}

	return nil, fmt.Errorf("service '%s' is not registered in module '%s'", name, mc.module.Name)
}

// Validate checks if the module container is valid
func (mc *ModuleContainer) Validate() error {
	if mc.module == nil {
		return fmt.Errorf("module container has no associated module")
	}

	// Validate module
	if err := mc.module.Validate(); err != nil {
		return fmt.Errorf("module validation failed: %w", err)
	}

	return nil
}