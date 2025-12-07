package core

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestService is a simple test service
type TestService struct {
	Value string
}

// TestService2 is another test service
type TestService2 struct {
	Service *TestService
}

func TestFactoryProvider(t *testing.T) {
	container := NewDIContainer()

	// Register a factory provider
	factory := func(c DIContainer) (interface{}, error) {
		return &TestService{Value: "test"}, nil
	}

	provider := NewFactoryProvider("testService", factory, Singleton)
	err := container.RegisterProvider(provider)
	if err != nil {
		t.Errorf("RegisterProvider failed: %v", err)
	}

	// Resolve the service
	_, err = container.Resolve("testService")
	if err != nil {
		t.Errorf("Resolve failed: %v", err)
	}
}

func TestClassProvider(t *testing.T) {
	container := NewDIContainer()

	// Register a class provider
	provider := NewClassProviderByType[TestService]("testService", Singleton)
	err := container.RegisterProvider(provider)
	if err != nil {
		t.Errorf("RegisterProvider failed: %v", err)
	}

	// Resolve the service
	service, err := container.Resolve("testService")
	if err != nil {
		t.Errorf("Resolve failed: %v", err)
	}

	_, ok := service.(*TestService)
	if !ok {
		t.Error("Service is not of type *TestService")
	}
}

func TestValueProvider(t *testing.T) {
	container := NewDIContainer()

	// Create a test service instance
	testService := &TestService{Value: "pre-created"}

	// Register a value provider
	provider := NewValueProvider("testService", testService)
	err := container.RegisterProvider(provider)
	if err != nil {
		t.Errorf("RegisterProvider failed: %v", err)
	}

	// Resolve the service
	service, err := container.Resolve("testService")
	if err != nil {
		t.Errorf("Resolve failed: %v", err)
	}

	resolvedService, ok := service.(*TestService)
	if !ok {
		t.Error("Service is not of type *TestService")
	}

	// Should be the same instance
	if resolvedService != testService {
		t.Error("Value provider did not return the same instance")
	}

	if resolvedService.Value != "pre-created" {
		t.Errorf("Expected value 'pre-created', got '%s'", resolvedService.Value)
	}
}

func TestAsyncProvider(t *testing.T) {
	container := NewDIContainer()

	// Register an async provider
	factory := func(c DIContainer, ctx context.Context) (interface{}, error) {
		select {
		case <-time.After(100 * time.Millisecond):
			return &TestService{Value: "async"}, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	provider := NewAsyncProvider("testService", factory, Singleton)
	err := container.RegisterProvider(provider)
	if err != nil {
		t.Errorf("RegisterProvider failed: %v", err)
	}

	// Resolve the service
	ctx := context.Background()
	service, err := container.ResolveWithContext("testService", ctx)
	if err != nil {
		t.Errorf("Resolve failed: %v", err)
	}

	testService, ok := service.(*TestService)
	if !ok {
		t.Error("Service is not of type *TestService")
	}

	if testService.Value != "async" {
		t.Errorf("Expected value 'async', got '%s'", testService.Value)
	}
}

func TestAsyncProviderTimeout(t *testing.T) {
	container := NewDIContainer()

	// Register an async provider that takes longer than its timeout
	factory := func(c DIContainer, ctx context.Context) (interface{}, error) {
		select {
		case <-time.After(2 * time.Second):
			return &TestService{Value: "async"}, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// Provider with 100ms timeout
	provider := NewAsyncProviderWithTimeout("testService", factory, Singleton, 100*time.Millisecond)
	err := container.RegisterProvider(provider)
	if err != nil {
		t.Errorf("RegisterProvider failed: %v", err)
	}

	// Resolve should fail due to timeout
	ctx := context.Background()
	_, err = container.ResolveWithContext("testService", ctx)
	if err == nil {
		t.Error("Expected timeout error, but got none")
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected DeadlineExceeded error, got: %v", err)
	}
}

func TestProviderLifetimeSingleton(t *testing.T) {
	container := NewDIContainer()

	// Register a singleton provider
	provider := NewClassProviderByType[TestService]("testService", Singleton)
	err := container.RegisterProviderSingleton(provider)
	if err != nil {
		t.Errorf("RegisterProviderSingleton failed: %v", err)
	}

	// Resolve twice
	service1, err1 := container.Resolve("testService")
	service2, err2 := container.Resolve("testService")

	if err1 != nil || err2 != nil {
		t.Errorf("Resolve failed: err1=%v, err2=%v", err1, err2)
	}

	// Should be the same instance
	if service1 != service2 {
		t.Error("Singleton provider returned different instances")
	}
}

func TestProviderLifetimeTransient(t *testing.T) {
	container := NewDIContainer()

	// Register a transient provider
	provider := NewClassProviderByType[TestService]("testService", Transient)
	err := container.RegisterProviderTransient(provider)
	if err != nil {
		t.Errorf("RegisterProviderTransient failed: %v", err)
	}

	// Resolve twice
	service1, err1 := container.Resolve("testService")
	service2, err2 := container.Resolve("testService")

	if err1 != nil || err2 != nil {
		t.Errorf("Resolve failed: err1=%v, err2=%v", err1, err2)
	}

	// Should be different instances
	if service1 == service2 {
		t.Error("Transient provider returned the same instance")
	}
}

func TestModuleWithProviders(t *testing.T) {
	// Create providers
	factoryProvider := NewFactoryProvider("factoryService", func(c DIContainer) (interface{}, error) {
		return &TestService{Value: "factory"}, nil
	}, Singleton)

	classProvider := NewClassProviderByType[TestService2]("classService", Singleton)

	valueProvider := NewValueProvider("valueService", &TestService{Value: "value"})

	// Create module with providers
	module := NewModule("testModule", "1.0.0").
		WithProviders(factoryProvider, classProvider, valueProvider).
		WithExports("factoryService", "classService")

	// Validate module
	err := module.Validate()
	if err != nil {
		t.Errorf("Module validation failed: %v", err)
	}

	// Check providers count
	if len(module.Providers) != 3 {
		t.Errorf("Expected 3 providers, got %d", len(module.Providers))
	}
}

func TestModuleProviderValidation(t *testing.T) {
	// Test duplicate provider names
	provider1 := NewValueProvider("testService", &TestService{Value: "1"})
	provider2 := NewValueProvider("testService", &TestService{Value: "2"})

	module := NewModule("testModule", "1.0.0").
		WithProviders(provider1, provider2)

	err := module.Validate()
	if err == nil {
		t.Error("Expected error for duplicate provider names")
	}

	if err.Error() != "duplicate provider name 'testService' in module 'testModule'" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestModuleExportValidation(t *testing.T) {
	// Test export of non-existent provider
	provider := NewValueProvider("testService", &TestService{Value: "test"})

	module := NewModule("testModule", "1.0.0").
		WithProviders(provider).
		WithExports("nonExistentService")

	err := module.Validate()
	if err == nil {
		t.Error("Expected error for exporting non-existent provider")
	}

	if err.Error() != "exported provider 'nonExistentService' not found in module 'testModule'" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestBackwardCompatibility(t *testing.T) {
	container := NewDIContainer()

	// Register using old factory method
	err := container.RegisterSingleton("testService", func(c DIContainer) (interface{}, error) {
		return &TestService{Value: "legacy"}, nil
	})
	if err != nil {
		t.Errorf("RegisterSingleton failed: %v", err)
	}

	// Should still resolve correctly
	service, err := container.Resolve("testService")
	if err != nil {
		t.Errorf("Resolve failed: %v", err)
	}

	testService, ok := service.(*TestService)
	if !ok {
		t.Error("Service is not of type *TestService")
	}

	if testService.Value != "legacy" {
		t.Errorf("Expected value 'legacy', got '%s'", testService.Value)
	}
}

// BenchmarkProviderResolution benchmarks provider resolution performance
func BenchmarkProviderResolution(b *testing.B) {
	container := NewDIContainer()

	// Register different provider types
	container.RegisterSingleton("factory", func(c DIContainer) (interface{}, error) {
		return &TestService{Value: "test"}, nil
	})

	container.RegisterProvider(NewClassProviderByType[TestService]("class", Singleton))
	container.RegisterProvider(NewValueProvider("value", &TestService{Value: "test"}))

	b.ResetTimer()

	b.Run("Factory", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = container.Resolve("factory")
		}
	})

	b.Run("Class", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = container.Resolve("class")
		}
	})

	b.Run("Value", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = container.Resolve("value")
		}
	})
}

// BenchmarkAsyncProvider benchmarks async provider initialization
func BenchmarkAsyncProvider(b *testing.B) {
	container := NewDIContainer()

	factory := func(c DIContainer, ctx context.Context) (interface{}, error) {
		select {
		case <-time.After(10 * time.Millisecond):
			return &TestService{Value: "async"}, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	container.RegisterProvider(NewAsyncProvider("async", factory, Singleton))

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = container.ResolveWithContext("async", context.Background())
		}
	})
}