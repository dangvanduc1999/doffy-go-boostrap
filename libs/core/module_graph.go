package core

import (
	"fmt"
	"sort"
)

// ModuleGraph manages module dependencies and initialization order
type ModuleGraph struct {
	modules map[string]*Module
	edges   map[string][]string // module name -> dependency names
}

// NewModuleGraph creates a new module graph
func NewModuleGraph() *ModuleGraph {
	return &ModuleGraph{
		modules: make(map[string]*Module),
		edges:   make(map[string][]string),
	}
}

// AddModule registers a module and its dependencies
func (g *ModuleGraph) AddModule(module *Module) error {
	if module == nil {
		return fmt.Errorf("module cannot be nil")
	}

	// Validate module before adding
	if err := module.Validate(); err != nil {
		return fmt.Errorf("module validation failed: %w", err)
	}

	if _, exists := g.modules[module.Name]; exists {
		return fmt.Errorf("module '%s' already registered", module.Name)
	}

	g.modules[module.Name] = module

	// Build dependency edges
	dependencies := make([]string, len(module.Imports))
	for i, dep := range module.Imports {
		dependencies[i] = dep.Name
	}
	g.edges[module.Name] = dependencies

	return nil
}

// GetModule returns a module by name
func (g *ModuleGraph) GetModule(name string) (*Module, bool) {
	module, exists := g.modules[name]
	return module, exists
}

// GetAllModules returns all registered modules
func (g *ModuleGraph) GetAllModules() []*Module {
	modules := make([]*Module, 0, len(g.modules))
	for _, module := range g.modules {
		modules = append(modules, module)
	}
	return modules
}

// TopologicalSort returns modules in dependency order
// Dependencies come before dependents in the result
func (g *ModuleGraph) TopologicalSort() ([]*Module, error) {
	visited := make(map[string]bool)
	temp := make(map[string]bool)
	var postOrder []*Module

	var visit func(name string) error
	visit = func(name string) error {
		if temp[name] {
			// Build cycle path for better error message
			path := g.buildCyclePath(name, temp)
			return fmt.Errorf("circular dependency detected: %s", path)
		}
		if visited[name] {
			return nil
		}

		temp[name] = true

		// Visit all dependencies first
		for _, dep := range g.edges[name] {
			if _, exists := g.modules[dep]; !exists {
				return fmt.Errorf("module '%s' depends on non-existent module '%s'", name, dep)
			}
			if err := visit(dep); err != nil {
				return err
			}
		}

		temp[name] = false
		visited[name] = true
		// Add to post-order list (dependencies first)
		postOrder = append(postOrder, g.modules[name])
		return nil
	}

	// Visit all modules in dependency order
	// Build a list of modules ordered by their dependencies
	moduleOrder := make([]string, 0, len(g.modules))

	// Find modules with no dependencies first
	for name := range g.modules {
		if len(g.edges[name]) == 0 {
			moduleOrder = append(moduleOrder, name)
		}
	}

	// Then add modules with dependencies
	for name := range g.modules {
		if len(g.edges[name]) > 0 {
			moduleOrder = append(moduleOrder, name)
		}
	}

	for _, name := range moduleOrder {
		if !visited[name] {
			if err := visit(name); err != nil {
				return nil, err
			}
		}
	}

	// Post-order gives us dependents first, dependencies last
	// Reverse to get dependencies first
	result := make([]*Module, len(postOrder))
	for i, module := range postOrder {
		result[len(postOrder)-1-i] = module
	}

	return result, nil
}

// buildCyclePath constructs a readable path for circular dependency error
func (g *ModuleGraph) buildCyclePath(start string, recStack map[string]bool) string {
	// For simplicity, just return the start module and indicate a cycle
	// In a more complex implementation, we could trace the full cycle path
	return start
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// GetSortedModuleNames returns module names sorted alphabetically
func (g *ModuleGraph) GetSortedModuleNames() []string {
	names := make([]string, 0, len(g.modules))
	for name := range g.modules {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetDependencies returns direct dependencies of a module
func (g *ModuleGraph) GetDependencies(moduleName string) ([]*Module, error) {
	module, exists := g.modules[moduleName]
	if !exists {
		return nil, fmt.Errorf("module '%s' not found", moduleName)
	}

	dependencies := make([]*Module, len(module.Imports))
	for i, dep := range module.Imports {
		dependencies[i] = dep
	}

	return dependencies, nil
}

// GetDependents returns modules that depend on the given module
func (g *ModuleGraph) GetDependents(moduleName string) ([]*Module, error) {
	if _, exists := g.modules[moduleName]; !exists {
		return nil, fmt.Errorf("module '%s' not found", moduleName)
	}

	var dependents []*Module
	for name, edges := range g.edges {
		for _, dep := range edges {
			if dep == moduleName {
				dependents = append(dependents, g.modules[name])
				break
			}
		}
	}

	return dependents, nil
}

// ValidateGraph checks for common issues like missing dependencies
func (g *ModuleGraph) ValidateGraph() error {
	// Check for missing dependencies
	for name, edges := range g.edges {
		for _, dep := range edges {
			if _, exists := g.modules[dep]; !exists {
				return fmt.Errorf("module '%s' depends on non-existent module '%s'", name, dep)
			}
		}
	}

	// Check for circular dependencies
	_, err := g.TopologicalSort()
	if err != nil {
		return fmt.Errorf("graph validation failed: %w", err)
	}

	return nil
}

// ValidateImports checks all imported modules exist and are registered
func (g *ModuleGraph) ValidateImports(module *Module) error {
	for _, imported := range module.Imports {
		if _, exists := g.modules[imported.Name]; !exists {
			return fmt.Errorf(
				"module '%s' imports non-existent module '%s'",
				module.Name,
				imported.Name,
			)
		}
	}
	return nil
}

// ValidateExportAccess checks module only accesses exported providers from imports
// This is a static analysis helper; actual enforcement in ResolveWithContext
func (g *ModuleGraph) ValidateExportAccess(module *Module, providerName string) error {
	// Check if provider exists in any imported module
	for _, imported := range module.Imports {
		importedModule := g.modules[imported.Name]
		if importedModule.IsExported(providerName) || importedModule.Global {
			return nil // Valid access
		}
	}

	return fmt.Errorf(
		"module '%s' attempts to access provider '%s' not exported by any imported module",
		module.Name,
		providerName,
	)
}

// Clone creates a deep copy of the module graph
func (g *ModuleGraph) Clone() *ModuleGraph {
	clone := NewModuleGraph()

	// Clone modules
	for name, module := range g.modules {
		cloneModule := &Module{
			Name:        module.Name,
			Version:     module.Version,
			Description: module.Description,
			Imports:     make([]*Module, len(module.Imports)),
			Providers:   make([]Provider, len(module.Providers)),
			Exports:     make([]string, len(module.Exports)),
			Controllers: make([]Controller, len(module.Controllers)),
			Prefix:      module.Prefix,
			Global:      module.Global,
		}

		// Copy slices
		copy(cloneModule.Imports, module.Imports)
		copy(cloneModule.Providers, module.Providers)
		copy(cloneModule.Exports, module.Exports)
		copy(cloneModule.Controllers, module.Controllers)

		clone.modules[name] = cloneModule
	}

	// Clone edges
	for name, edges := range g.edges {
		cloneEdges := make([]string, len(edges))
		copy(cloneEdges, edges)
		clone.edges[name] = cloneEdges
	}

	return clone
}