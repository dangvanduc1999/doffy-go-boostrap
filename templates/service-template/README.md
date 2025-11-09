# Service Template

This is a template for creating new services using the Doffy Go framework.

## How to Use

1. Copy this template to a new directory
2. Rename the service from "MyService" to your desired service name
3. Update the configuration in `main.go`
4. Add your business logic to the service
5. Add routes as needed

## Structure

```
service-template/
├── main.go          # Main application entry point
└── README.md        # This file
```

## Customization

### Rename the Service

1. Replace all instances of "MyService" with your service name
2. Update the plugin name in the `Name()` method
3. Update the service name in the configuration

### Add Configuration

Create a `config.json` file:

```json
{
  "name": "MyService",
  "mode": "debug",
  "port": 8080,
  "database": {
    "host": "localhost",
    "port": 5432,
    "name": "myservice"
  }
}
```

Access configuration in your service:

```go
configManager := c.MustGet("container").(core.DIContainer).Resolve("configManager")
dbHost := configManager.(core.ConfigManager).GetString("database.host")
```

### Add Routes

```go
router.GET("/api/v1/my-endpoint", func(c *gin.Context, container core.DIContainer) {
    // Get service from container
    myService, _ := container.Resolve("myService")
    
    // Use service
    result := myService.(*MyService).DoSomething()
    
    c.JSON(200, gin.H{"result": result})
})
```

### Add Dependencies

Register additional services in your plugin's `Register` method:

```go
func (p *MyServicePlugin) Register(container core.DIContainer) error {
    // Register main service
    container.RegisterSingleton("myService", func(c core.DIContainer) (interface{}, error) {
        return &MyService{}, nil
    })
    
    // Register dependency
    container.RegisterSingleton("database", func(c core.DIContainer) (interface{}, error) {
        return NewDatabase(), nil
    })
    
    return nil
}
```

Then inject the dependency:

```go
func (p *MyServicePlugin) Register(container core.DIContainer) error {
    return container.RegisterSingleton("myService", func(c core.DIContainer) (interface{}, error) {
        db, err := c.Resolve("database")
        if err != nil {
            return nil, err
        }
        return &MyService{db: db.(Database)}, nil
    })
}
```

## Environment Variables

You can override configuration using environment variables with the `DOFFY_` prefix:

```bash
export DOFFY_PORT=9090
export DOFFY_MODE=production
export DOFFY_DATABASE_HOST=prod-db.example.com
```

## Running the Service

```bash
go run main.go
```

The service will start on the configured port (default: 8080).

## Health Check

The service includes a health check endpoint at `/health`:

```bash
curl http://localhost:8080/health
```

Response:
```json
{
  "status": "ok",
  "service": "MyService",
  "timestamp": "2023-01-01T12:00:00Z"
}