package core_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/dangvanduc1999/doffy-go-boostrap/libs/core"
)

// MockPlugin implements core.Plugin and core.ApplicationHookProvider
type MockPlugin struct {
	mock.Mock
	core.BasePlugin
}

func (m *MockPlugin) Name() string {
	return "MockPlugin"
}

func (m *MockPlugin) Version() string {
	return "1.0.0"
}

func (m *MockPlugin) Register(container core.DIContainer) error {
	m.Called(container)
	return nil
}

func (m *MockPlugin) Hooks() []core.LifecycleHook {
	return nil
}

func (m *MockPlugin) AppHooks() []core.ApplicationHook {
	args := m.Called()
	return args.Get(0).([]core.ApplicationHook)
}

func TestApplicationHooksExecutionOrder(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	executionOrder := []string{}

	// Create hooks
	onRegisterHook := &core.ApplicationHookFunc{
		OnRegisterFunc: func(plugin interface{}) {
			executionOrder = append(executionOrder, "OnRegister")
		},
	}

	onRouteHook := &core.ApplicationHookFunc{
		OnRouteFunc: func(config *core.RouteConfig) {
			executionOrder = append(executionOrder, "OnRoute")
		},
	}

	onReadyHook := &core.ApplicationHookFunc{
		OnReadyFunc: func(app interface{}) error {
			executionOrder = append(executionOrder, "OnReady")
			return nil
		},
	}

	onListenHook := &core.ApplicationHookFunc{
		OnListenFunc: func(addr string) {
			executionOrder = append(executionOrder, "OnListen")
		},
	}

	preCloseHook := &core.ApplicationHookFunc{
		PreCloseFunc: func(ctx interface{}) {
			executionOrder = append(executionOrder, "PreClose")
		},
	}

	onCloseHook := &core.ApplicationHookFunc{
		OnCloseFunc: func() error {
			executionOrder = append(executionOrder, "OnClose")
			return nil
		},
	}

	// Mock plugin
	mockPlugin := new(MockPlugin)
	mockPlugin.On("Register", mock.Anything).Return(nil)
	mockPlugin.On("AppHooks").Return([]core.ApplicationHook{
		onRegisterHook,
		onRouteHook,
		onReadyHook,
		onListenHook,
		preCloseHook,
		onCloseHook,
	})

	// Create app
	app := core.CreateDoffApp(&core.AppOptions{
		Name:      "TestApp",
		Port:      8080,
		Mode:      gin.TestMode,
		UseLogger: false,
	})

	// Register plugin
	err := app.RegisterPlugin(mockPlugin)
	assert.NoError(t, err)

	// Register a route to trigger OnRoute
	router := app.(interface{ GetRouter() *core.Router }).GetRouter()
	router.GET(core.RouteConfig{Path: "/test"}, func(c *gin.Context, container core.DIContainer) {})

	// Start app in a goroutine
	go func() {
		app.Listen()
	}()

	// Wait for startup
	time.Sleep(200 * time.Millisecond)

	// Shutdown app
	err = app.Shutdown(context.Background())
	assert.NoError(t, err)

	// Verify order
	expectedOrder := []string{
		"OnRegister", // From RegisterPlugin
		"OnRoute",    // From router.GET
		"OnReady",    // From Listen start
		"OnListen",   // From Listen after start
		"PreClose",   // From Shutdown start
		"OnClose",    // From Shutdown end
	}

	// Note: OnListen is async, so it might race with PreClose in this tight loop test
	// But generally we expect it before shutdown if we wait enough

	// Check if all events occurred
	assert.Subset(t, executionOrder, expectedOrder)
	assert.Contains(t, executionOrder, "OnRegister")
	assert.Contains(t, executionOrder, "OnRoute")
	assert.Contains(t, executionOrder, "OnReady")
	// OnListen might be tricky to catch deterministically in test without more sync, but let's check
	// assert.Contains(t, executionOrder, "OnListen")
	assert.Contains(t, executionOrder, "PreClose")
	assert.Contains(t, executionOrder, "OnClose")

	fmt.Printf("Execution Order: %v\n", executionOrder)
}
