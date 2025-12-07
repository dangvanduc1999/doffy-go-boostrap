package core

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestModuleIsExported tests the IsExported method
func TestModuleIsExported(t *testing.T) {
	module := DefaultModule("test", "1.0.0")
	module.Exports = []string{"serviceA", "serviceB"}

	assert.True(t, module.IsExported("serviceA"))
	assert.True(t, module.IsExported("serviceB"))
	assert.False(t, module.IsExported("serviceC"))
}

// TestModuleGlobalFlag tests that Global modules bypass encapsulation
func TestModuleGlobalFlag(t *testing.T) {
	// Save original mode
	originalMode := GetEncapsulationMode()
	defer SetEncapsulationMode(originalMode)

	// Set to enforce mode
	SetEncapsulationMode(EncapsulationEnforce)

	// Create root container
	rootContainer := NewDIContainer()

	// Create parent module
	parentModule := DefaultModule("parent", "1.0.0")
	parentModule.Global = false // Not global
	parentModule.Exports = []string{} // No exports
	parentContainer := NewModuleContainer(parentModule, rootContainer)

	// Register a private service in parent module container
	parentContainer.RegisterSingleton("privateService", func(container DIContainer) (interface{}, error) {
		return "parent-private", nil
	})

	// Create child module that is NOT global
	childModule := DefaultModule("child", "1.0.0")
	childModule.Global = false // Not global
	childContainer := NewModuleContainer(childModule, parentContainer)

	// Child should NOT be able to access parent's private service (not exported)
	_, err := childContainer.Resolve("privateService")
	if err == nil {
		t.Error("Expected error when accessing unexported service")
	} else {
		// Check if it's the expected error type
		t.Logf("Got error: %v", err)
	}

	// Now make the child module global
	childModule.Global = true

	// Global child should be able to access parent's private service
	service, err := childContainer.Resolve("privateService")
	if err != nil {
		t.Errorf("Global module should access private service, got error: %v", err)
	} else {
		assert.Equal(t, "parent-private", service)
	}
}

// TestSiblingModuleIsolation tests that sibling modules cannot access each other's private services
func TestSiblingModuleIsolation(t *testing.T) {
	// Save original mode
	originalMode := GetEncapsulationMode()
	defer SetEncapsulationMode(originalMode)

	// Set to enforce mode
	SetEncapsulationMode(EncapsulationEnforce)

	// Create root container
	rootContainer := NewDIContainer()

	// Create ModuleA with private service
	moduleA := DefaultModule("moduleA", "1.0.0")
	moduleA.Exports = []string{} // No exports
	containerA := NewModuleContainer(moduleA, rootContainer)

	// Create ModuleB that tries to access ModuleA's private service
	moduleB := DefaultModule("moduleB", "1.0.0")
	containerB := NewModuleContainer(moduleB, rootContainer)

	// Register ModuleA's service in its own container
	containerA.RegisterSingleton("privateService", func(container DIContainer) (interface{}, error) {
		return "moduleA-private", nil
	})

	// ModuleB should NOT be able to access ModuleA's private service
	_, err := containerB.Resolve("privateService")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not registered in module 'moduleB'")
}

// TestExportedServiceAccess tests that exported services are accessible to child modules
func TestExportedServiceAccess(t *testing.T) {
	// Save original mode
	originalMode := GetEncapsulationMode()
	defer SetEncapsulationMode(originalMode)

	// Set to enforce mode
	SetEncapsulationMode(EncapsulationEnforce)

	// Create root container
	rootContainer := NewDIContainer()

	// Create parent module with exported service
	parentModule := DefaultModule("parent", "1.0.0")
	parentModule.Providers = []Provider{
		NewFactoryProvider("exportedService", func(container DIContainer) (interface{}, error) {
			return "parent-exported", nil
		}, Singleton),
	}
	parentModule.Exports = []string{"exportedService"}
	parentContainer := NewModuleContainer(parentModule, rootContainer)

	// Create child module
	childModule := DefaultModule("child", "1.0.0")
	childContainer := NewModuleContainer(childModule, parentContainer)

	// Register the exported service
	parentContainer.RegisterSingleton("exportedService", func(container DIContainer) (interface{}, error) {
		return "parent-exported", nil
	})

	// Child should be able to access parent's exported service
	service, err := childContainer.Resolve("exportedService")
	assert.NoError(t, err)
	assert.Equal(t, "parent-exported", service)
}

// TestEncapsulationModeWarn tests that Warn mode logs warnings but allows access
func TestEncapsulationModeWarn(t *testing.T) {
	// Save original mode and logger
	originalMode := GetEncapsulationMode()
	originalLogger := encapsulationViolationLogger
	defer func() {
		SetEncapsulationMode(originalMode)
		encapsulationViolationLogger = originalLogger
	}()

	// Set to warn mode
	SetEncapsulationMode(EncapsulationWarn)

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	encapsulationViolationLogger = w

	// Create containers
	rootContainer := NewDIContainer()
	parentModule := DefaultModule("parent", "1.0.0")
	parentModule.Providers = []Provider{
		NewFactoryProvider("privateService", func(container DIContainer) (interface{}, error) {
			return "private", nil
		}, Singleton),
	}
	parentModule.Exports = []string{} // No exports
	parentContainer := NewModuleContainer(parentModule, rootContainer)
	parentContainer.RegisterSingleton("privateService", func(container DIContainer) (interface{}, error) {
		return "private", nil
	})

	childModule := DefaultModule("child", "1.0.0")
	childContainer := NewModuleContainer(childModule, parentContainer)

	// Child tries to access parent's private service
	service, err := childContainer.Resolve("privateService")

	// Close pipe and restore stderr
	w.Close()
	os.Stderr = oldStderr

	// Read warning
	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	warning := string(buf[:n])

	// Should succeed with warning
	assert.NoError(t, err)
	assert.Equal(t, "private", service)
	assert.Contains(t, warning, "WARNING:")
	assert.Contains(t, warning, "cannot access unexported provider")
}

// TestEncapsulationModeDisabled tests that Disabled mode bypasses all checks
func TestEncapsulationModeDisabled(t *testing.T) {
	// Save original mode
	originalMode := GetEncapsulationMode()
	defer SetEncapsulationMode(originalMode)

	// Set to disabled mode
	SetEncapsulationMode(EncapsulationDisabled)

	// Create containers
	rootContainer := NewDIContainer()
	parentModule := DefaultModule("parent", "1.0.0")
	parentModule.Providers = []Provider{
		NewFactoryProvider("privateService", func(container DIContainer) (interface{}, error) {
			return "private", nil
		}, Singleton),
	}
	parentModule.Exports = []string{} // No exports
	parentContainer := NewModuleContainer(parentModule, rootContainer)
	parentContainer.RegisterSingleton("privateService", func(container DIContainer) (interface{}, error) {
		return "private", nil
	})

	childModule := DefaultModule("child", "1.0.0")
	childContainer := NewModuleContainer(childModule, parentContainer)

	// Child should be able to access parent's private service
	service, err := childContainer.Resolve("privateService")
	assert.NoError(t, err)
	assert.Equal(t, "private", service)
}

// TestValidateExports tests export validation
func TestValidateExports(t *testing.T) {
	module := DefaultModule("test", "1.0.0")
	module.Providers = []Provider{
		NewFactoryProvider("service1", func(container DIContainer) (interface{}, error) {
			return nil, nil
		}, Singleton),
		NewFactoryProvider("service2", func(container DIContainer) (interface{}, error) {
			return nil, nil
		}, Singleton),
	}
	module.Exports = []string{"service1", "service2"} // Valid exports

	// Should pass validation
	err := module.ValidateExports()
	assert.NoError(t, err)

	// Add invalid export
	module.Exports = []string{"service1", "service2", "nonExistent"}
	err = module.ValidateExports()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exports non-existent provider 'nonExistent'")
}

// TestValidateImports tests import validation
func TestValidateImports(t *testing.T) {
	graph := NewModuleGraph()

	// Add modules
	module1 := DefaultModule("module1", "1.0.0")
	module2 := DefaultModule("module2", "1.0.0")
	module3 := DefaultModule("module3", "1.0.0")

	graph.AddModule(module1)
	graph.AddModule(module2)
	// module3 is not added

	// Module with valid imports
	module1.Imports = []*Module{module2}
	err := graph.ValidateImports(module1)
	assert.NoError(t, err)

	// Module with invalid imports
	module2.Imports = []*Module{module3}
	err = graph.ValidateImports(module2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "imports non-existent module 'module3'")
}

// TestValidateExportAccess tests export access validation
func TestValidateExportAccess(t *testing.T) {
	graph := NewModuleGraph()

	moduleA := DefaultModule("moduleA", "1.0.0")
	moduleA.Exports = []string{"serviceA"}

	moduleB := DefaultModule("moduleB", "1.0.0")
	moduleB.Imports = []*Module{moduleA}

	graph.AddModule(moduleA)
	graph.AddModule(moduleB)

	// Valid access to exported service
	err := graph.ValidateExportAccess(moduleB, "serviceA")
	assert.NoError(t, err)

	// Invalid access to non-exported service
	err = graph.ValidateExportAccess(moduleB, "privateService")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not exported by any imported module")
}

// TestConcurrentAccess tests thread safety of encapsulation checks
func TestConcurrentAccess(t *testing.T) {
	// Save original mode
	originalMode := GetEncapsulationMode()
	defer SetEncapsulationMode(originalMode)

	// Set to warn mode to test concurrent warnings
	SetEncapsulationMode(EncapsulationWarn)

	rootContainer := NewDIContainer()
	parentModule := DefaultModule("parent", "1.0.0")
	parentModule.Providers = []Provider{
		NewFactoryProvider("service", func(container DIContainer) (interface{}, error) {
			return "value", nil
		}, Singleton),
	}
	parentModule.Exports = []string{} // No exports
	parentContainer := NewModuleContainer(parentModule, rootContainer)
	parentContainer.RegisterSingleton("service", func(container DIContainer) (interface{}, error) {
		return "value", nil
	})

	childModule := DefaultModule("child", "1.0.0")
	childContainer := NewModuleContainer(childModule, parentContainer)

	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrent access attempts
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			childContainer.Resolve("service")
		}()
	}

	wg.Wait()
	// No assertion needed - just verify no race conditions or panics
}

// TestModuleContainerResolveWithContext tests resolution with context
func TestModuleContainerResolveWithContext(t *testing.T) {
	// Save original mode
	originalMode := GetEncapsulationMode()
	defer SetEncapsulationMode(originalMode)

	SetEncapsulationMode(EncapsulationEnforce)

	rootContainer := NewDIContainer()
	parentModule := DefaultModule("parent", "1.0.0")
	parentModule.Exports = []string{"exported"}
	parentContainer := NewModuleContainer(parentModule, rootContainer)

	childModule := DefaultModule("child", "1.0.0")
	childContainer := NewModuleContainer(childModule, parentContainer)

	// Register exported service in parent
	parentContainer.RegisterSingleton("exported", func(container DIContainer) (interface{}, error) {
		return "exported-value", nil
	})

	// Child should be able to resolve exported service with context
	ctx := context.Background()
	service, err := childContainer.ResolveWithContext("exported", ctx)
	assert.NoError(t, err)
	assert.Equal(t, "exported-value", service)

	// Child should NOT be able to resolve non-exported service
	parentContainer.RegisterSingleton("private", func(container DIContainer) (interface{}, error) {
		return "private-value", nil
	})
	_, err = childContainer.ResolveWithContext("private", ctx)
	assert.Error(t, err)
}