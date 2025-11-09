package core

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type AppOptions struct {
	Name       string         `json:"name"`
	Mode       string         `json:"mode"`
	Port       int16          `json:"port"`
	Cors       any            `json:"cors,omitempty"`
	UseLogger  bool           `json:"useLogger"`
	Logger     Logger         `json:"logger,omitempty"`
	Plugins    []PluginConfig `json:"plugins,omitempty"`
	ConfigPath string         `json:"configPath,omitempty"`
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
	server        *gin.Engine
	config        config
	name          string
	mode          string
	logger        Logger
	container     DIContainer
	pluginManager *PluginManager
	httpServer    *http.Server
	configManager ConfigManager
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

	// Set the global service locator
	SetGlobalContainer(d.container)

	return d
}

func (d *DoffApp) Listen() {
	if d.logger == nil {
		panic("logger is not initialized")
	}

	addr := fmt.Sprintf(":%v", d.config.Port)

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

	// Shutdown plugins
	if err := d.pluginManager.ShutdownPlugins(); err != nil {
		d.logger.Infor(&LoggerItem{
			Event:    "PluginShutdownError",
			Messages: "Error during plugin shutdown",
			Error:    err,
		})
	}

	// Shutdown HTTP server
	return d.httpServer.Shutdown(ctx)
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
	}

	// Initialize configuration first
	app.initConfig(options.ConfigPath)

	// Initialize DI container and plugin manager
	app.initDIContainer()

	// Initialize logger
	app.initLogger(options.UseLogger, options.Logger)

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
