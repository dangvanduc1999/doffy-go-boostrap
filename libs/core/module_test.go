package core

import (
	"testing"
)

func TestModule_NewModule(t *testing.T) {
	module := NewModule("test-module", "1.0.0")

	if module.Name != "test-module" {
		t.Errorf("Expected module name 'test-module', got '%s'", module.Name)
	}

	if module.Version != "1.0.0" {
		t.Errorf("Expected module version '1.0.0', got '%s'", module.Version)
	}

	if len(module.Imports) != 0 {
		t.Errorf("Expected empty imports, got %d", len(module.Imports))
	}

	if len(module.Providers) != 0 {
		t.Errorf("Expected empty providers, got %d", len(module.Providers))
	}

	if len(module.Exports) != 0 {
		t.Errorf("Expected empty exports, got %d", len(module.Exports))
	}

	if module.Global {
		t.Errorf("Expected Global to be false, got true")
	}
}

func TestModule_WithMethods(t *testing.T) {
	// Create test modules for imports
	dep1 := NewModule("dep1", "1.0.0")
	dep2 := NewModule("dep2", "1.0.0")

	// Create a factory for testing
	factory := func(container DIContainer) (interface{}, error) {
		return "test-service", nil
	}

	module := NewModule("test", "1.0.0").
		WithImports(dep1, dep2).
		WithFactoryProviders(factory).
		WithExports("service1", "service2").
		WithPrefix("/api/v1").
		AsGlobal()

	// Test imports
	if len(module.Imports) != 2 {
		t.Errorf("Expected 2 imports, got %d", len(module.Imports))
	}

	// Test providers
	if len(module.Providers) != 1 {
		t.Errorf("Expected 1 provider, got %d", len(module.Providers))
	}

	// Test exports
	if len(module.Exports) != 2 {
		t.Errorf("Expected 2 exports, got %d", len(module.Exports))
	}

	// Test prefix
	if module.Prefix != "/api/v1" {
		t.Errorf("Expected prefix '/api/v1', got '%s'", module.Prefix)
	}

	// Test global flag
	if !module.Global {
		t.Errorf("Expected Global to be true, got false")
	}
}

func TestModule_Validate(t *testing.T) {
	tests := []struct {
		name    string
		module  *Module
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid module",
			module: &Module{
				Name:    "test",
				Version: "1.0.0",
			},
			wantErr: false,
		},
		{
			name: "empty name",
			module: &Module{
				Name:    "",
				Version: "1.0.0",
			},
			wantErr: true,
			errMsg:  "module name cannot be empty",
		},
		{
			name: "empty version",
			module: &Module{
				Name:    "test",
				Version: "",
			},
			wantErr: true,
			errMsg:  "module version cannot be empty",
		},
		{
			name: "duplicate exports",
			module: &Module{
				Name:    "test",
				Version: "1.0.0",
				Exports: []string{"service1", "service1"},
			},
			wantErr: true,
			errMsg:  "duplicate export 'service1' in module 'test'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.module.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("Validate() error message = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestModule_HasExport(t *testing.T) {
	module := &Module{
		Name:    "test",
		Version: "1.0.0",
		Exports: []string{"service1", "service2"},
	}

	if !module.HasExport("service1") {
		t.Error("Expected HasExport to return true for 'service1'")
	}

	if module.HasExport("service3") {
		t.Error("Expected HasExport to return false for 'service3'")
	}
}

func TestDefaultModule(t *testing.T) {
	module := DefaultModule("test-plugin", "2.0.0")

	if module.Name != "test-plugin" {
		t.Errorf("Expected name 'test-plugin', got '%s'", module.Name)
	}

	if module.Version != "2.0.0" {
		t.Errorf("Expected version '2.0.0', got '%s'", module.Version)
	}

	if !module.Global {
		t.Error("Expected Global to be true for DefaultModule")
	}

	if len(module.Exports) != 0 {
		t.Error("Expected empty exports for DefaultModule")
	}
}

func TestModuleGraph_AddModule(t *testing.T) {
	graph := NewModuleGraph()
	module := NewModule("test", "1.0.0")

	err := graph.AddModule(module)
	if err != nil {
		t.Errorf("AddModule() error = %v", err)
	}

	// Test duplicate module
	err = graph.AddModule(module)
	if err == nil {
		t.Error("Expected error for duplicate module")
	}

	// Test nil module
	err = graph.AddModule(nil)
	if err == nil {
		t.Error("Expected error for nil module")
	}
}

func TestModuleGraph_TopologicalSort(t *testing.T) {
	graph := NewModuleGraph()

	// Create modules
	leaf := NewModule("leaf", "1.0.0")
	middle := NewModule("middle", "1.0.0")
	root := NewModule("root", "1.0.0")

	// Set up dependencies (WithImports returns a new module)
	middle = middle.WithImports(leaf)
	root = root.WithImports(middle)

	// Add modules to graph
	graph.AddModule(leaf)
	graph.AddModule(middle)
	graph.AddModule(root)

	// Test topological sort
	sorted, err := graph.TopologicalSort()
	if err != nil {
		t.Errorf("TopologicalSort() error = %v", err)
	}

	if len(sorted) != 3 {
		t.Errorf("Expected 3 modules, got %d", len(sorted))
	}

	// Create a map of module positions
	positions := make(map[string]int)
	for i, module := range sorted {
		positions[module.Name] = i
		t.Logf("Sorted[%d]: %s", i, module.Name)
	}

	// Verify dependencies are loaded in the graph
	t.Logf("Graph edges for root: %v", graph.edges["root"])
	t.Logf("Graph edges for middle: %v", graph.edges["middle"])
	t.Logf("Graph edges for leaf: %v", graph.edges["leaf"])

	// Check that dependencies come before dependents
	if positions["leaf"] >= positions["middle"] {
		t.Errorf("Module 'leaf' should come before 'middle', got leaf=%d, middle=%d", positions["leaf"], positions["middle"])
	}

	if positions["middle"] >= positions["root"] {
		t.Errorf("Module 'middle' should come before 'root', got middle=%d, root=%d", positions["middle"], positions["root"])
	}
}

func TestModuleGraph_CircularDependency(t *testing.T) {
	// Create circular dependency: a -> b -> c -> a
	c := NewModule("c", "1.0.0")
	b := NewModule("b", "1.0.0")
	a := NewModule("a", "1.0.0")

	// Create the cycle
	b = b.WithImports(c)
	a = a.WithImports(b)
	c = c.WithImports(a)

	graph := NewModuleGraph()
	graph.AddModule(a)
	graph.AddModule(b)
	graph.AddModule(c)

	_, err := graph.TopologicalSort()
	if err == nil {
		t.Error("Expected error for circular dependency")
	}

	if len(err.Error()) < 26 || err.Error()[:26] != "circular dependency detected" {
		t.Errorf("Expected circular dependency error, got: %v", err)
	}
}

func TestModuleGraph_GetModule(t *testing.T) {
	graph := NewModuleGraph()
	module := NewModule("test", "1.0.0")
	graph.AddModule(module)

	retrieved, exists := graph.GetModule("test")
	if !exists {
		t.Error("Expected module to exist")
	}

	if retrieved.Name != "test" {
		t.Errorf("Expected module name 'test', got '%s'", retrieved.Name)
	}

	_, exists = graph.GetModule("nonexistent")
	if exists {
		t.Error("Expected module to not exist")
	}
}

func TestModuleGraph_GetDependencies(t *testing.T) {
	graph := NewModuleGraph()

	dep := NewModule("dep", "1.0.0")
	module := NewModule("test", "1.0.0").WithImports(dep)

	graph.AddModule(dep)
	graph.AddModule(module)

	deps, err := graph.GetDependencies("test")
	if err != nil {
		t.Errorf("GetDependencies() error = %v", err)
	}

	if len(deps) != 1 {
		t.Errorf("Expected 1 dependency, got %d", len(deps))
	}

	if deps[0].Name != "dep" {
		t.Errorf("Expected dependency name 'dep', got '%s'", deps[0].Name)
	}

	// Test nonexistent module
	_, err = graph.GetDependencies("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent module")
	}
}

func TestModuleGraph_GetDependents(t *testing.T) {
	graph := NewModuleGraph()

	dep := NewModule("dep", "1.0.0")
	module := NewModule("test", "1.0.0").WithImports(dep)

	graph.AddModule(dep)
	graph.AddModule(module)

	dependents, err := graph.GetDependents("dep")
	if err != nil {
		t.Errorf("GetDependents() error = %v", err)
	}

	if len(dependents) != 1 {
		t.Errorf("Expected 1 dependent, got %d", len(dependents))
	}

	if dependents[0].Name != "test" {
		t.Errorf("Expected dependent name 'test', got '%s'", dependents[0].Name)
	}
}

func TestModuleGraph_ValidateGraph(t *testing.T) {
	graph := NewModuleGraph()

	// Test valid graph
	module1 := NewModule("module1", "1.0.0")
	module2 := NewModule("module2", "1.0.0").WithImports(module1)

	graph.AddModule(module1)
	graph.AddModule(module2)

	err := graph.ValidateGraph()
	if err != nil {
		t.Errorf("ValidateGraph() error = %v", err)
	}

	// Test missing dependency
	module3 := NewModule("module3", "1.0.0").WithImports(NewModule("missing", "1.0.0"))
	graph.AddModule(module3)

	err = graph.ValidateGraph()
	if err == nil {
		t.Error("Expected error for missing dependency")
	}
}

func TestModuleGraph_Clone(t *testing.T) {
	original := NewModuleGraph()
	module := NewModule("test", "1.0.0").
		WithFactoryProviders(func(container DIContainer) (interface{}, error) { return nil, nil }).
		WithExports("service1")
	original.AddModule(module)

	clone := original.Clone()

	// Verify they have the same modules
	originalModule, _ := original.GetModule("test")
	cloneModule, _ := clone.GetModule("test")

	if originalModule.Name != cloneModule.Name {
		t.Error("Cloned module should have same name")
	}

	// Modify clone and verify original is unchanged
	cloneModule.Description = "modified"

	if originalModule.Description != "" {
		t.Error("Original module should not be affected by clone modification")
	}
}

// Benchmark tests
func BenchmarkModuleGraph_AddModule(b *testing.B) {
	graph := NewModuleGraph()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		module := NewModule("module-"+string(rune(i)), "1.0.0")
		graph.AddModule(module)
	}
}

func BenchmarkModuleGraph_TopologicalSort(b *testing.B) {
	// Create a graph with 50 modules
	graph := NewModuleGraph()

	for i := 0; i < 50; i++ {
		name := "module-" + string(rune(i))
		module := NewModule(name, "1.0.0")

		// Add some dependencies
		if i > 0 {
			depName := "module-" + string(rune(i-1))
			if dep, exists := graph.GetModule(depName); exists {
				module.WithImports(dep)
			}
		}

		graph.AddModule(module)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		graph.TopologicalSort()
	}
}