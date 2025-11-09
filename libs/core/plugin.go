package core

import (
	"github.com/gin-gonic/gin"
)

// Plugin defines the interface that all plugins must implement
type Plugin interface {
	// Name returns the unique name of the plugin
	Name() string
	// Version returns the version of the plugin
	Version() string
	// Register registers the plugin's services with the DI container
	Register(container DIContainer) error
	// Hooks returns the lifecycle hooks provided by this plugin
	Hooks() []LifecycleHook
	// Routes registers the plugin's routes (optional)
	Routes(router *gin.Engine) error
	// Init is called after all plugins are registered (optional)
	Init(app *DoffApp) error
	// Shutdown is called when the application is shutting down (optional)
	Shutdown() error
}

// PluginConfig holds configuration for a plugin
type PluginConfig struct {
	Name   string                 `json:"name"`
	Config map[string]interface{} `json:"config"`
}

// PluginManager manages plugin registration and lifecycle
type PluginManager struct {
	plugins   map[string]Plugin
	app       *DoffApp
	container DIContainer
	lifecycle *LifecycleManager
}

// NewPluginManager creates a new plugin manager
func NewPluginManager(app *DoffApp, container DIContainer) *PluginManager {
	return &PluginManager{
		plugins:   make(map[string]Plugin),
		app:       app,
		container: container,
		lifecycle: NewLifecycleManager(),
	}
}

// RegisterPlugin registers a plugin
func (pm *PluginManager) RegisterPlugin(plugin Plugin) error {
	if plugin == nil {
		return ErrPluginNil
	}

	name := plugin.Name()
	if _, exists := pm.plugins[name]; exists {
		return ErrPluginAlreadyRegistered
	}

	// Register plugin services
	if err := plugin.Register(pm.container); err != nil {
		return ErrPluginRegistrationFailed
	}

	// Store plugin
	pm.plugins[name] = plugin

	// Add hooks to lifecycle manager
	for _, hook := range plugin.Hooks() {
		pm.lifecycle.AddHook(hook)
	}

	return nil
}

// RegisterPluginByName registers a plugin by name (for dynamic loading)
func (pm *PluginManager) RegisterPluginByName(name string, config map[string]interface{}) error {
	// This would be implemented for dynamic plugin loading
	// For now, we'll return an error
	return ErrPluginNotFound
}

// GetPlugin returns a plugin by name
func (pm *PluginManager) GetPlugin(name string) (Plugin, bool) {
	plugin, exists := pm.plugins[name]
	return plugin, exists
}

// GetPlugins returns all registered plugins
func (pm *PluginManager) GetPlugins() map[string]Plugin {
	// Return a copy to prevent external modification
	result := make(map[string]Plugin)
	for name, plugin := range pm.plugins {
		result[name] = plugin
	}
	return result
}

// InitializePlugins initializes all registered plugins
func (pm *PluginManager) InitializePlugins() error {
	for _, plugin := range pm.plugins {
		if err := plugin.Init(pm.app); err != nil {
			return err
		}
	}
	return nil
}

// RegisterRoutes registers routes for all plugins
func (pm *PluginManager) RegisterRoutes(router *gin.Engine) error {
	for _, plugin := range pm.plugins {
		if err := plugin.Routes(router); err != nil {
			return err
		}
	}
	return nil
}

// ShutdownPlugins shuts down all registered plugins
func (pm *PluginManager) ShutdownPlugins() error {
	for _, plugin := range pm.plugins {
		if err := plugin.Shutdown(); err != nil {
			return err
		}
	}
	return nil
}

// GetLifecycleManager returns the lifecycle manager
func (pm *PluginManager) GetLifecycleManager() *LifecycleManager {
	return pm.lifecycle
}

// Plugin errors
var (
	ErrPluginNil                  = newError("plugin cannot be nil")
	ErrPluginAlreadyRegistered    = newError("plugin is already registered")
	ErrPluginNotFound             = newError("plugin not found")
	ErrPluginRegistrationFailed   = newError("plugin registration failed")
	ErrPluginInitializationFailed = newError("plugin initialization failed")
)

// BasePlugin provides a default implementation for optional plugin methods
type BasePlugin struct{}

// Routes provides a default empty implementation
func (bp *BasePlugin) Routes(router *gin.Engine) error {
	return nil
}

// Init provides a default empty implementation
func (bp *BasePlugin) Init(app *DoffApp) error {
	return nil
}

// Shutdown provides a default empty implementation
func (bp *BasePlugin) Shutdown() error {
	return nil
}

// Helper function to create errors
func newError(message string) error {
	return &pluginError{message: message}
}

type pluginError struct {
	message string
}

func (e *pluginError) Error() string {
	return e.message
}
