package main

import (
	"fmt"
	"github.com/dangvanduc1999/doffy-go-boostrap/libs/core"
	"net/http"

	"github.com/gin-gonic/gin"
)

// User represents a user entity
type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// UserService defines the interface for user service
type UserService interface {
	GetUser(id string) (*User, error)
	CreateUser(user *User) (*User, error)
	UpdateUser(id string, user *User) (*User, error)
	DeleteUser(id string) error
	ListUsers() ([]*User, error)
}

// userService implements UserService
type userService struct {
	users map[string]*User
}

// NewUserService creates a new user service
func NewUserService() UserService {
	return &userService{
		users: make(map[string]*User),
	}
}

// GetUser retrieves a user by ID
func (s *userService) GetUser(id string) (*User, error) {
	if user, exists := s.users[id]; exists {
		return user, nil
	}
	return nil, fmt.Errorf("user with ID %s not found", id)
}

// CreateUser creates a new user
func (s *userService) CreateUser(user *User) (*User, error) {
	if user.ID == "" {
		user.ID = fmt.Sprintf("user-%d", len(s.users)+1)
	}
	s.users[user.ID] = user
	return user, nil
}

// UpdateUser updates an existing user
func (s *userService) UpdateUser(id string, user *User) (*User, error) {
	if _, exists := s.users[id]; !exists {
		return nil, fmt.Errorf("user with ID %s not found", id)
	}
	user.ID = id
	s.users[id] = user
	return user, nil
}

// DeleteUser deletes a user by ID
func (s *userService) DeleteUser(id string) error {
	if _, exists := s.users[id]; !exists {
		return fmt.Errorf("user with ID %s not found", id)
	}
	delete(s.users, id)
	return nil
}

// ListUsers returns all users
func (s *userService) ListUsers() ([]*User, error) {
	users := make([]*User, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, user)
	}
	return users, nil
}

// UserController handles HTTP requests for users
type UserController struct {
	userService UserService
}

// NewUserController creates a new user controller
func NewUserController(userService UserService) *UserController {
	return &UserController{
		userService: userService,
	}
}

// GetUser handles GET /users/:id
func (ctrl *UserController) GetUser(c *gin.Context) {
	id := c.Param("id")
	user, err := ctrl.userService.GetUser(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, user)
}

// CreateUser handles POST /users
func (ctrl *UserController) CreateUser(c *gin.Context) {
	var user User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	createdUser, err := ctrl.userService.CreateUser(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, createdUser)
}

// UpdateUser handles PUT /users/:id
func (ctrl *UserController) UpdateUser(c *gin.Context) {
	id := c.Param("id")
	var user User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updatedUser, err := ctrl.userService.UpdateUser(id, &user)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, updatedUser)
}

// DeleteUser handles DELETE /users/:id
func (ctrl *UserController) DeleteUser(c *gin.Context) {
	id := c.Param("id")
	err := ctrl.userService.DeleteUser(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// ListUsers handles GET /users
func (ctrl *UserController) ListUsers(c *gin.Context) {
	users, err := ctrl.userService.ListUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
}

// UserPlugin implements the Plugin interface for user management
type UserPlugin struct {
	core.BasePlugin
}

// NewUserPlugin creates a new user plugin
func NewUserPlugin() *UserPlugin {
	return &UserPlugin{}
}

// Name returns the plugin name
func (p *UserPlugin) Name() string {
	return "user-service"
}

// Version returns the plugin version
func (p *UserPlugin) Version() string {
	return "1.0.0"
}

// Register registers the user service with the DI container
func (p *UserPlugin) Register(container core.DIContainer) error {
	// Register user service
	if err := container.RegisterSingleton("userService", func(c core.DIContainer) (interface{}, error) {
		return NewUserService(), nil
	}); err != nil {
		return err
	}

	// Register user controller by type name for automatic resolution
	if err := container.RegisterSingleton("UserController", func(c core.DIContainer) (interface{}, error) {
		userService, err := c.Resolve("userService")
		if err != nil {
			return nil, err
		}
		return NewUserController(userService.(UserService)), nil
	}); err != nil {
		return err
	}

	// Also register by the conventional name for backward compatibility
	return container.RegisterSingleton("userController", func(c core.DIContainer) (interface{}, error) {
		userService, err := c.Resolve("userService")
		if err != nil {
			return nil, err
		}
		return NewUserController(userService.(UserService)), nil
	})
}

// Hooks returns the lifecycle hooks for the user plugin
func (p *UserPlugin) Hooks() []core.LifecycleHook {
	// No specific hooks needed for the user plugin
	return []core.LifecycleHook{}
}

// Routes registers the user routes
func (p *UserPlugin) Routes(router *gin.Engine) error {
	// Get the container from context
	container := core.GlobalLocator.GetContainer()
	enhancedRouter := core.NewEnhancedRouter(router, container)

	// Register routes using the enhanced router with automatic controller injection
	api := enhancedRouter.Group("/api/v1")
	{
		api.GET("/users", func(c *gin.Context, ctrl *UserController) {
			ctrl.ListUsers(c)
		})

		api.GET("/users/:id", func(c *gin.Context, ctrl *UserController) {
			ctrl.GetUser(c)
		})

		api.POST("/users", func(c *gin.Context, ctrl *UserController) {
			ctrl.CreateUser(c)
		})

		api.PUT("/users/:id", func(c *gin.Context, ctrl *UserController) {
			ctrl.UpdateUser(c)
		})

		api.DELETE("/users/:id", func(c *gin.Context, ctrl *UserController) {
			ctrl.DeleteUser(c)
		})
	}

	return nil
}
