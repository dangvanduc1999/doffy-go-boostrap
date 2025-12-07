package core

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

// Module represents a logical grouping of providers, controllers, and dependencies
type Module struct {
	// Name is the unique identifier for this module
	Name string

	// Version follows semantic versioning (e.g., "1.0.0")
	Version string

	// Description provides human-readable module purpose
	Description string

	// Imports lists modules this module depends on
	// Services exported by imported modules become available in this module's container
	Imports []*Module

	// Providers are services registered in this module's DI container
	// Changed from []Factory to []Provider in Phase 2
	Providers []Provider

	// Exports lists provider names accessible to importing modules
	// Non-exported providers are private to this module
	Exports []string

	// Controllers are HTTP request handlers registered by this module
	// Initially unused; populated in Phase 5 for route prefixing
	Controllers []Controller

	// Prefix for all routes registered by this module (Phase 5)
	Prefix string

	// Global flag breaks encapsulation (fastify-plugin pattern)
	// If true, all providers registered in root container
	Global bool
}

// Controller placeholder (defined in Phase 5)
type Controller interface{}

// NewModule creates a new module with the given name and version
func NewModule(name, version string) *Module {
	return &Module{
		Name:        name,
		Version:     version,
		Imports:     make([]*Module, 0),
		Providers:   make([]Provider, 0),
		Exports:     make([]string, 0),
		Controllers: make([]Controller, 0),
		Global:      false,
	}
}

// WithImports adds import dependencies to the module
func (m *Module) WithImports(imports ...*Module) *Module {
	m.Imports = append(m.Imports, imports...)
	return m
}

// WithProviders adds providers to the module
func (m *Module) WithProviders(providers ...Provider) *Module {
	m.Providers = append(m.Providers, providers...)
	return m
}

// WithFactoryProviders adds Factory providers for backward compatibility
func (m *Module) WithFactoryProviders(providers ...Factory) *Module {
	for _, factory := range providers {
		provider := &FactoryProvider{
			Name:     "", // Will be set by caller
			Factory:  factory,
			Lifetime: Singleton, // Default lifetime
		}
		m.Providers = append(m.Providers, provider)
	}
	return m
}

// AddProvider adds a single provider to the module
func (m *Module) AddProvider(provider Provider) *Module {
	m.Providers = append(m.Providers, provider)
	return m
}

// AddFactoryProvider adds a Factory provider with a name and lifetime
func (m *Module) AddFactoryProvider(name string, factory Factory, lifetime Lifetime) *Module {
	provider := NewFactoryProvider(name, factory, lifetime)
	m.Providers = append(m.Providers, provider)
	return m
}

// AddClassProvider adds a Class provider with reflection
func (m *Module) AddClassProvider(name string, typ reflect.Type, lifetime Lifetime) *Module {
	provider := NewClassProvider(name, typ, lifetime)
	m.Providers = append(m.Providers, provider)
	return m
}

// AddValueProvider adds a Value provider
func (m *Module) AddValueProvider(name string, value interface{}) *Module {
	provider := NewValueProvider(name, value)
	m.Providers = append(m.Providers, provider)
	return m
}

// AddAsyncProvider adds an Async provider with timeout
func (m *Module) AddAsyncProvider(name string, factory AsyncFactory, lifetime Lifetime, timeout time.Duration) *Module {
	provider := NewAsyncProviderWithTimeout(name, factory, lifetime, timeout)
	m.Providers = append(m.Providers, provider)
	return m
}

// WithExports marks provider names as exported
func (m *Module) WithExports(exports ...string) *Module {
	m.Exports = append(m.Exports, exports...)
	return m
}

// WithControllers adds controllers to the module
func (m *Module) WithControllers(controllers ...Controller) *Module {
	m.Controllers = append(m.Controllers, controllers...)
	return m
}

// WithPrefix sets the route prefix for the module
func (m *Module) WithPrefix(prefix string) *Module {
	m.Prefix = prefix
	return m
}

// AsGlobal marks the module as global (breaks encapsulation)
func (m *Module) AsGlobal() *Module {
	m.Global = true
	return m
}

// Validate checks if the module configuration is valid
func (m *Module) Validate() error {
	if m.Name == "" {
		return fmt.Errorf("module name cannot be empty")
	}
	if m.Version == "" {
		return fmt.Errorf("module version cannot be empty")
	}

	// Check for duplicate provider names in exports
	seen := make(map[string]bool)
	for _, export := range m.Exports {
		if seen[export] {
			return fmt.Errorf("duplicate export '%s' in module '%s'", export, m.Name)
		}
		seen[export] = true
	}

	// Check for duplicate provider names
	providerNames := make(map[string]bool)
	for _, provider := range m.Providers {
		if provider == nil {
			return fmt.Errorf("provider cannot be nil in module '%s'", m.Name)
		}
		name := provider.GetName()
		if name == "" {
			return fmt.Errorf("provider name cannot be empty in module '%s'", m.Name)
		}
		if providerNames[name] {
			return fmt.Errorf("duplicate provider name '%s' in module '%s'", name, m.Name)
		}
		providerNames[name] = true
	}

	// Check that all exported providers exist
	for _, export := range m.Exports {
		if !providerNames[export] {
			return fmt.Errorf("exported provider '%s' not found in module '%s'", export, m.Name)
		}
	}

	// Validate module prefix
	if err := m.ValidatePrefix(); err != nil {
		return err
	}

	return nil
}

// IsExported checks if a provider is exported by this module
func (m *Module) IsExported(providerName string) bool {
	for _, exportName := range m.Exports {
		if exportName == providerName {
			return true
		}
	}
	return false
}

// ValidateExports checks all exported provider names exist in module (alias for Validate consistency)
func (m *Module) ValidateExports() error {
	return m.Validate()
}

// GetFullPrefix computes full prefix including parent prefixes
// For now, returns the module's own prefix
// TODO: In future phases, concatenate with parent module prefixes
func (m *Module) GetFullPrefix() string {
	if m.Prefix == "" {
		return ""
	}
	// Normalize prefix: ensure it starts with / and doesn't end with /
	prefix := m.Prefix
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	return strings.TrimSuffix(prefix, "/")
}

// ValidatePrefix checks if the module prefix is valid
func (m *Module) ValidatePrefix() error {
	if m.Prefix == "" {
		return nil // Empty prefix is valid
	}

	// Prefix must start with /
	if !strings.HasPrefix(m.Prefix, "/") {
		return fmt.Errorf("module prefix '%s' must start with '/'", m.Prefix)
	}

	// Prevent path traversal attempts
	if strings.Contains(m.Prefix, "..") {
		return fmt.Errorf("module prefix '%s' contains invalid path traversal", m.Prefix)
	}

	return nil
}

// GetImportNames returns the names of all imported modules
func (m *Module) GetImportNames() []string {
	names := make([]string, len(m.Imports))
	for i, imp := range m.Imports {
		names[i] = imp.Name
	}
	return names
}

// HasExport checks if a provider name is exported by this module
func (m *Module) HasExport(name string) bool {
	for _, export := range m.Exports {
		if export == name {
			return true
		}
	}
	return false
}

// DefaultModule creates a default module wrapper for legacy plugins
func DefaultModule(name, version string) *Module {
	return &Module{
		Name:        name,
		Version:     version,
		Imports:     make([]*Module, 0),
		Providers:   make([]Provider, 0),
		Exports:     make([]string, 0),
		Controllers: make([]Controller, 0),
		Global:      true, // Maintain existing global behavior
	}
}