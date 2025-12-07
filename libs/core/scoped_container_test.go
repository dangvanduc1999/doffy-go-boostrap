package core

import (
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModuleContainer_CreateRequestScope(t *testing.T) {
	// Create module and module container
	module := DefaultModule("test", "1.0.0")
	parentContainer := NewDIContainer()
	moduleContainer := NewModuleContainer(module, parentContainer)

	// Register a service in module container
	moduleContainer.RegisterSingleton("service", func(container DIContainer) (interface{}, error) {
		return "module-service", nil
	})

	// Create request scope
	requestContainer := moduleContainer.CreateRequestScope()

	// Test that request container can resolve module service
	service, err := requestContainer.Resolve("service")
	require.NoError(t, err)
	assert.Equal(t, "module-service", service)

	// Test that request container has access to parent
	assert.Same(t, moduleContainer, requestContainer.GetModule())
}

func TestRequestContainer_DecorateRequest(t *testing.T) {
	module := DefaultModule("test", "1.0.0")
	moduleContainer := NewModuleContainer(module, NewDIContainer())
	requestContainer := NewRequestContainer(moduleContainer)

	// Decorate request
	requestContainer.DecorateRequest("userID", 123)

	// Retrieve decoration
	value, exists := requestContainer.GetRequestData("userID")
	require.True(t, exists)
	assert.Equal(t, 123, value)
}

func TestRequestContainer_DecorateReply(t *testing.T) {
	module := DefaultModule("test", "1.0.0")
	moduleContainer := NewModuleContainer(module, NewDIContainer())
	requestContainer := NewRequestContainer(moduleContainer)

	// Decorate reply helper
	replyFn := func(data interface{}) interface{} {
		return map[string]interface{}{
			"success": true,
			"data":    data,
		}
	}
	requestContainer.DecorateReply("successResponse", replyFn)

	// Retrieve reply helper
	helper, exists := requestContainer.GetReplyHelper("successResponse")
	require.True(t, exists)
	assert.NotNil(t, helper)
}

func TestRequestContainer_ResolveOrder(t *testing.T) {
	module := DefaultModule("test", "1.0.0")
	parentContainer := NewDIContainer()
	moduleContainer := NewModuleContainer(module, parentContainer)

	// Register service in parent
	parentContainer.RegisterSingleton("service", func(container DIContainer) (interface{}, error) {
		return "parent-service", nil
	})

	// Register service in module
	moduleContainer.RegisterSingleton("service", func(container DIContainer) (interface{}, error) {
		return "module-service", nil
	})

	requestContainer := NewRequestContainer(moduleContainer)

	// Decorate request with same name
	requestContainer.DecorateRequest("service", "request-service")

	// Resolve should return request decoration first
	service, err := requestContainer.Resolve("service")
	require.NoError(t, err)
	assert.Equal(t, "request-service", service)

	// Clear and resolve should return module service
	requestContainer.Clear()
	service, err = requestContainer.Resolve("service")
	require.NoError(t, err)
	assert.Equal(t, "module-service", service)
}

func TestDecoratorManager_InstanceDecorators(t *testing.T) {
	dm := NewDecoratorManager()

	// Register instance decorator
	err := dm.Decorate("config", map[string]string{"env": "test"})
	require.NoError(t, err)

	// Retrieve instance decorator
	value, exists := dm.GetInstanceDecorator("config")
	require.True(t, exists)
	assert.Equal(t, map[string]string{"env": "test"}, value)

	// Test duplicate registration error
	err = dm.Decorate("config", "another-value")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestDecoratorManager_RequestDecorators(t *testing.T) {
	dm := NewDecoratorManager()

	// Register request decorator with default
	err := dm.DecorateRequest("timeout", 30)
	require.NoError(t, err)

	// Retrieve request decorator
	value, exists := dm.GetRequestDecorator("timeout")
	require.True(t, exists)
	assert.Equal(t, 30, value)
}

func TestDecoratorManager_ReplyDecorators(t *testing.T) {
	dm := NewDecoratorManager()

	// Register reply helper
	err := dm.DecorateReply("jsonResponse", func(data interface{}) map[string]interface{} {
		return map[string]interface{}{
			"status": "success",
			"data":   data,
		}
	})
	require.NoError(t, err)

	// Retrieve reply helper
	helper, exists := dm.GetReplyDecorator("jsonResponse")
	require.True(t, exists)
	assert.NotNil(t, helper)
}

func TestRequestContainer_InitializeFromManager(t *testing.T) {
	dm := NewDecoratorManager()

	// Register decorators in manager
	dm.DecorateRequest("correlationID", "test-123")
	dm.DecorateReply("errorResponse", func(msg string) map[string]interface{} {
		return map[string]interface{}{
			"status": "error",
			"message": msg,
		}
	})

	// Create request container and initialize from manager
	module := DefaultModule("test", "1.0.0")
	moduleContainer := NewModuleContainer(module, NewDIContainer())
	requestContainer := NewRequestContainer(moduleContainer)

	// Initialize decorators from manager
	dm.InitializeRequestContainer(requestContainer)
	dm.InitializeReplyHelpers(requestContainer)

	// Check initialized decorators
	value, exists := requestContainer.GetRequestData("correlationID")
	require.True(t, exists)
	assert.Equal(t, "test-123", value)

	helper, exists := requestContainer.GetReplyHelper("errorResponse")
	require.True(t, exists)
	assert.NotNil(t, helper)
}

func TestRequestContainer_ConcurrentAccess(t *testing.T) {
	module := DefaultModule("test", "1.0.0")
	moduleContainer := NewModuleContainer(module, NewDIContainer())
	requestContainer := NewRequestContainer(moduleContainer)

	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrently decorate request data
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			requestContainer.DecorateRequest("key", id)
		}(i)
	}

	// Concurrently decorate reply helpers
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			requestContainer.DecorateReply("helper", id)
		}(i)
	}

	wg.Wait()

	// Verify final state
	_, exists := requestContainer.GetRequestData("key")
	assert.True(t, exists)

	_, exists = requestContainer.GetReplyHelper("helper")
	assert.True(t, exists)
}

func TestDoffApp_DecoratorMethods(t *testing.T) {
	gin.SetMode(gin.TestMode)

	app := &DoffApp{
		name:            "test-app",
		mode:            gin.TestMode,
		moduleContainers: make(map[string]*ModuleContainer),
		decoratorManager: NewDecoratorManager(),
	}

	// Test instance decorator
	err := app.Decorate("instanceKey", "instanceValue")
	require.NoError(t, err)

	value, exists := app.GetDecoratorManager().GetInstanceDecorator("instanceKey")
	require.True(t, exists)
	assert.Equal(t, "instanceValue", value)

	// Test request decorator
	err = app.DecorateRequest("requestKey", "requestValue")
	require.NoError(t, err)

	value, exists = app.GetDecoratorManager().GetRequestDecorator("requestKey")
	require.True(t, exists)
	assert.Equal(t, "requestValue", value)

	// Test reply decorator
	err = app.DecorateReply("replyKey", func(data interface{}) interface{} { return data })
	require.NoError(t, err)

	helper, exists := app.GetDecoratorManager().GetReplyDecorator("replyKey")
	require.True(t, exists)
	assert.NotNil(t, helper)
}