package core

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type AppOptions struct {
	Name          string         `json:"name"`
	Mode          string         `json:"mode"`
	Port          int16          `json:"port"`
	Cors          any            `json:"cors,omitempty"`
	UseLogger     bool           `json:"useLogger"`
	Logger        Logger         `json:"logger,omitempty"`
	Plugins       []PluginConfig `json:"plugins,omitempty"`
	ConfigPath    string         `json:"configPath,omitempty"`
	Authenticator any            `json:"authenticator,omitempty"`
}

type DoffServer interface {
	Listen()
	Shutdown(ctx context.Context) error
	RegisterPlugin(plugin Plugin) error
	GetContainer() DIContainer
	GetEngine() *gin.Engine
}

type config struct {
	Port int16
}

type DoffApp struct {
	server           *gin.Engine
	config           config
	name             string
	mode             string
	logger           Logger
	container        DIContainer         // Root container
	moduleContainers  map[string]*ModuleContainer  // Module-scoped containers
	pluginManager    *PluginManager
	httpServer       *http.Server
	configManager     ConfigManager
	decoratorManager  *DecoratorManager       // Decorator API
}

func (d *DoffApp) initServer() *DoffApp {
	gin.SetMode(d.mode)
	d.server = gin.New()

	// Add app and DI container to context
	d.server.Use(func(c *gin.Context) {
		c.Set("app", d)
		c.Set("container", d.container)
		c.Next()
	})

	// Add lifecycle middleware
	lifecycleManager := d.pluginManager.GetLifecycleManager()

	d.server.Use(func(c *gin.Context) {
		// Execute OnRequest hooks
		lifecycleManager.ExecuteOnRequest(c)
		if c.IsAborted() {
			return
		}
		c.Next()
	})

	return d
}

func (d *DoffApp) initConfig(configPath string) *DoffApp {
	d.configManager = NewConfigManager()
	if err := d.configManager.Load(configPath); err != nil {
		// Can't log yet since logger might not be initialized
		fmt.Printf("Failed to load configuration: %v\n", err)
	}
	return d
}

func (d *DoffApp) initLogger(useLogger bool, customLogger Logger) *DoffApp {
	if useLogger && customLogger != nil {
		d.logger = customLogger
	} else {
		d.logger = DefaultLogger()
	}

	// Register logger in DI container
	if d.container != nil {
		d.container.RegisterSingleton("logger", func(container DIContainer) (interface{}, error) {
			return d.logger, nil
		})
	}

	return d
}

func (d *DoffApp) initDIContainer() *DoffApp {
	d.container = NewDIContainer()
	d.pluginManager = NewPluginManager(d, d.container)

	// Register config manager in DI container
	d.container.RegisterSingleton("configManager", func(container DIContainer) (interface{}, error) {
		return d.configManager, nil
	})

	// Register plugin manager in DI container
	d.container.RegisterSingleton("pluginManager", func(container DIContainer) (interface{}, error) {
		return d.pluginManager, nil
	})

	// Set the global service locator
	SetGlobalContainer(d.container)

	return d
}

func (d *DoffApp) initAuthenticator(authenticator interface{}) *DoffApp {
	if d.container != nil {
		d.container.RegisterSingleton("authenticator", func(container DIContainer) (interface{}, error) {
			return authenticator, nil
		})
	}
	return d
}

func (d *DoffApp) Listen() {
	if d.logger == nil {
		panic("logger is not initialized")
	}

	addr := fmt.Sprintf(":%v", d.config.Port)

	// Execute OnReady hooks (serial, blocks startup)
	if d.pluginManager != nil {
		if err := d.pluginManager.GetLifecycleManager().ExecuteOnReady(d); err != nil {
			d.logger.Infor(&LoggerItem{
				Event:    "OnReadyError",
				Messages: "Failed to execute OnReady hooks",
				Error:    err,
			})
			panic(err)
		}
	}

	// Initialize all plugins
	if d.pluginManager != nil {
		if err := d.pluginManager.InitializePlugins(); err != nil {
			d.logger.Infor(&LoggerItem{
				Event:    "PluginInitializationError",
				Messages: "Failed to initialize plugins",
				Error:    err,
			})
			panic(err)
		}

		// Register plugin routes
		if err := d.pluginManager.RegisterRoutes(d.server); err != nil {
			d.logger.Infor(&LoggerItem{
				Event:    "PluginRouteRegistrationError",
				Messages: "Failed to register plugin routes",
				Error:    err,
			})
			panic(err)
		}
	}

	// Add CORS if configured
	if d.config.Port != 0 {
		// This will be handled by the CORS plugin
	}

	// Create HTTP server
	d.httpServer = &http.Server{
		Addr:    addr,
		Handler: d.server,
	}

	payload := &LoggerItem{
		Event:    "StartServer",
		Messages: fmt.Sprintf("%s is starting.....", d.name),
		Data: struct {
			CreatedAT time.Time `json:"created_at"`
			Addr      string    `json:"address"`
		}{
			CreatedAT: time.Now().UTC(),
			Addr:      addr,
		},
	}
	d.logger.Infor(payload)

	// Execute OnListen hooks (async)
	go func() {
		// Wait a brief moment to ensure server is actually up
		time.Sleep(100 * time.Millisecond)
		if d.pluginManager != nil {
			d.pluginManager.GetLifecycleManager().ExecuteOnListen(addr)
		}
	}()

	if err := d.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		panic(err)
	}
}

func (d *DoffApp) Shutdown(ctx context.Context) error {
	d.logger.Infor(&LoggerItem{
		Event:    "ShutdownServer",
		Messages: fmt.Sprintf("%s is shutting down.....", d.name),
		Data: struct {
			ShutdownAt time.Time `json:"shutdown_at"`
		}{
			ShutdownAt: time.Now().UTC(),
		},
	})

	// Execute PreClose hooks (notify shutdown)
	if d.pluginManager != nil {
		d.pluginManager.GetLifecycleManager().ExecutePreClose(ctx)
	}

	// Shutdown HTTP server
	err := d.httpServer.Shutdown(ctx)

	// Execute OnClose hooks (final cleanup)
	if d.pluginManager != nil {
		if closeErr := d.pluginManager.GetLifecycleManager().ExecuteOnClose(); closeErr != nil {
			d.logger.Infor(&LoggerItem{
				Event:    "OnCloseError",
				Messages: "Error during OnClose hooks",
				Error:    closeErr,
			})
		}
	}

	// Shutdown plugins
	if pluginErr := d.pluginManager.ShutdownPlugins(); pluginErr != nil {
		d.logger.Infor(&LoggerItem{
			Event:    "PluginShutdownError",
			Messages: "Error during plugin shutdown",
			Error:    pluginErr,
		})
	}

	return err
}

func (d *DoffApp) RegisterPlugin(plugin Plugin) error {
	return d.pluginManager.RegisterPlugin(plugin)
}

func (d *DoffApp) GetContainer() DIContainer {
	return d.container
}

func (d *DoffApp) GetEngine() *gin.Engine {
	return d.server
}

// GetPluginManager returns the plugin manager
func (d *DoffApp) GetPluginManager() *PluginManager {
	return d.pluginManager
}

// GetRouter returns a router helper with DI support
func (d *DoffApp) GetRouter() *Router {
	return NewRouter(d.server, d.container)
}

func CreateDoffApp(options *AppOptions) DoffServer {
	app := &DoffApp{
		name: options.Name,
		mode: options.Mode,
		config: config{
			Port: options.Port,
		},
		moduleContainers:  make(map[string]*ModuleContainer),
		decoratorManager:  NewDecoratorManager(),
	}

	// Initialize configuration first
	app.initConfig(options.ConfigPath)

	// Initialize DI container and plugin manager
	app.initDIContainer()

	// Initialize logger
	app.initLogger(options.UseLogger, options.Logger)

	// Initialize authenticator
	app.initAuthenticator(options.Authenticator)

	// Initialize server
	app.initServer()

	// Register CORS plugin if configured
	if options.Cors != nil {
		corsPlugin := NewCorsPlugin(options.Cors)
		app.RegisterPlugin(corsPlugin)
	}

	return app
}

// GetConfigManager returns the configuration manager
func (d *DoffApp) GetConfigManager() ConfigManager {
	return d.configManager
}

// Decorate registers an instance-level decorator
func (d *DoffApp) Decorate(name string, value interface{}) error {
	return d.decoratorManager.Decorate(name, value)
}

// DecorateRequest registers a request-scoped decorator with default value
func (d *DoffApp) DecorateRequest(name string, defaultValue interface{}) error {
	return d.decoratorManager.DecorateRequest(name, defaultValue)
}

// DecorateReply registers a reply helper function
func (d *DoffApp) DecorateReply(name string, fn interface{}) error {
	return d.decoratorManager.DecorateReply(name, fn)
}

// GetDecoratorManager returns the decorator manager
func (d *DoffApp) GetDecoratorManager() *DecoratorManager {
	return d.decoratorManager
}
