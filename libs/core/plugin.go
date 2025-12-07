package core

import (
	"context"
	"fmt"
	"sync"
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

// RouteInfo contains information about a registered route
type RouteInfo struct {
	Method  string
	Path    string
	Options map[string]interface{}
}

// RouteAwarePlugin defines the interface for plugins that want to be notified about route registration
type RouteAwarePlugin interface {
	Plugin
	// OnRoute is called when a route is registered
	OnRoute(config *RouteConfig)
}

// PluginConfig holds configuration for a plugin
type PluginConfig struct {
	Name   string                 `json:"name"`
	Config map[string]interface{} `json:"config"`
}

// ModuleProvider extends Plugin interface to expose module metadata
type ModuleProvider interface {
	Plugin
	// Module returns the module definition for this plugin
	// If nil, plugin is wrapped in DefaultModule
	Module() *Module
}

// PluginManager manages plugin registration and lifecycle
type PluginManager struct {
	plugins      map[string]Plugin
	modules      *ModuleGraph
	app          *DoffApp
	container    DIContainer
	lifecycle    *LifecycleManager
	modulePrefixes map[string]string // Track module prefixes for route registration
}

// NewPluginManager creates a new plugin manager
func NewPluginManager(app *DoffApp, container DIContainer) *PluginManager {
	return &PluginManager{
		plugins:       make(map[string]Plugin),
		modules:       NewModuleGraph(),
		app:           app,
		container:     container,
		lifecycle:     NewLifecycleManager(),
		modulePrefixes: make(map[string]string),
	}
}

// ApplicationHookProvider defines the interface for plugins that provide application hooks
type ApplicationHookProvider interface {
	AppHooks() []ApplicationHook
}

// RegisterPlugin registers a plugin and its module
func (pm *PluginManager) RegisterPlugin(plugin Plugin) error {
	if plugin == nil {
		return ErrPluginNil
	}

	name := plugin.Name()
	if _, exists := pm.plugins[name]; exists {
		return ErrPluginAlreadyRegistered
	}

	// Extract or create module
	var module *Module
	if moduleProvider, ok := plugin.(ModuleProvider); ok {
		module = moduleProvider.Module()
	}

	if module == nil {
		// Backward compatibility: wrap in default module
		module = DefaultModule(plugin.Name(), plugin.Version())
	}

	// NEW: Validate module exports
	if err := module.ValidateExports(); err != nil {
		return fmt.Errorf("export validation failed: %w", err)
	}

	// Register module in dependency graph
	if err := pm.modules.AddModule(module); err != nil {
		return fmt.Errorf("module registration failed: %w", err)
	}

	// NEW: Validate module imports
	if err := pm.modules.ValidateImports(module); err != nil {
		return fmt.Errorf("import validation failed: %w", err)
	}

	// Track module prefix for route registration
	pm.modulePrefixes[module.Name] = module.GetFullPrefix()

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

	// Add application hooks if provided
	if appHookProvider, ok := plugin.(ApplicationHookProvider); ok {
		for _, hook := range appHookProvider.AppHooks() {
			pm.lifecycle.AddAppHook(hook)
		}
	}

	// Notify OnRegister hooks
	pm.lifecycle.ExecuteOnRegister(plugin)

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

// InitializePlugins executes plugins in dependency order with async support
func (pm *PluginManager) InitializePlugins() error {
	// Phase 1: Get initialization order from module graph
	orderedPlugins, err := pm.GetInitializationOrder()
	if err != nil {
		return fmt.Errorf("failed to resolve module dependencies: %w", err)
	}

	// Phase 2: Initialize async providers
	ctx := context.Background()
	if err := pm.initializeAsyncProviders(ctx, orderedPlugins); err != nil {
		return fmt.Errorf("async provider initialization failed: %w", err)
	}

	// Phase 3: Call plugin Init() methods (existing logic)
	for _, plugin := range orderedPlugins {
		if err := plugin.Init(pm.app); err != nil {
			return fmt.Errorf("plugin '%s' init failed: %w", plugin.Name(), err)
		}
	}

	return nil
}

// initializeAsyncProviders pre-initializes all async providers
func (pm *PluginManager) initializeAsyncProviders(ctx context.Context, plugins []Plugin) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(plugins))
	semaphore := make(chan struct{}, 10) // Limit parallel initialization to 10

	// Group providers by module dependencies
	for _, plugin := range plugins {
		moduleProvider, ok := plugin.(ModuleProvider)
		if !ok {
			continue
		}

		module := moduleProvider.Module()
		if module == nil {
			continue
		}

		// Initialize async providers for this module
		for _, provider := range module.Providers {
			if !provider.IsAsync() {
				continue
			}

			wg.Add(1)
			go func(p Provider, moduleName string) {
				defer wg.Done()

				// Acquire semaphore to limit parallelism
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				name := p.GetName()
				if _, err := pm.container.(*diContainer).ResolveWithContext(name, ctx); err != nil {
					errChan <- fmt.Errorf("async provider '%s' in module '%s' failed: %w",
						name, moduleName, err)
					return
				}
			}(provider, module.Name)
		}
	}

	// Wait for all async providers to complete
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// Collect any errors
	var errors []string
	for err := range errChan {
		if err != nil {
			errors = append(errors, err.Error())
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("async initialization errors: %v", errors)
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

// GetAppHooks returns all registered application hooks
func (pm *PluginManager) GetAppHooks() []ApplicationHook {
	return pm.lifecycle.appHooks
}

// ExecuteOnRoute notifies all RouteAwarePlugins about a new route
func (pm *PluginManager) ExecuteOnRoute(config *RouteConfig) {
	// Notify plugins implementing RouteAwarePlugin
	for _, plugin := range pm.plugins {
		if routeAware, ok := plugin.(RouteAwarePlugin); ok {
			routeAware.OnRoute(config)
		}
	}

	// Also execute registered OnRoute hooks from LifecycleManager
	pm.lifecycle.ExecuteOnRoute(config)
}

// GetModuleGraph returns the module dependency graph
func (pm *PluginManager) GetModuleGraph() *ModuleGraph {
	return pm.modules
}

// GetInitializationOrder returns plugins sorted by module dependencies
func (pm *PluginManager) GetInitializationOrder() ([]Plugin, error) {
	sortedModules, err := pm.modules.TopologicalSort()
	if err != nil {
		return nil, err
	}

	result := make([]Plugin, 0, len(sortedModules))
	for _, module := range sortedModules {
		if plugin, exists := pm.plugins[module.Name]; exists {
			result = append(result, plugin)
		}
	}

	return result, nil
}

// GetEnhancedRouterForModule creates an EnhancedRouter with the module's prefix
func (pm *PluginManager) GetEnhancedRouterForModule(moduleName string) *EnhancedRouter {
	prefix, exists := pm.modulePrefixes[moduleName]
	if !exists {
		prefix = ""
	}
	return NewEnhancedRouterWithPrefix(pm.app.server, pm.container, prefix)
}

// GetModulePrefix returns the prefix for a given module
func (pm *PluginManager) GetModulePrefix(moduleName string) string {
	prefix, exists := pm.modulePrefixes[moduleName]
	if !exists {
		return ""
	}
	return prefix
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
