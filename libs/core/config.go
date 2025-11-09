package core

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strconv"
	"strings"
)

// ConfigManager manages application configuration
type ConfigManager interface {
	Load(configPath string) error
	Get(key string) interface{}
	GetString(key string) string
	GetInt(key string) int
	GetBool(key string) bool
	GetFloat(key string) float64
	Set(key string, value interface{})
	Has(key string) bool
	Unmarshal(target interface{}) error
}

// configManager implements ConfigManager
type configManager struct {
	data map[string]interface{}
}

// NewConfigManager creates a new configuration manager
func NewConfigManager() ConfigManager {
	return &configManager{
		data: make(map[string]interface{}),
	}
}

// Load loads configuration from a file
func (cm *configManager) Load(configPath string) error {
	if configPath == "" {
		// Try to load default config files
		for _, path := range []string{"config.json", "config/config.json"} {
			if _, err := os.Stat(path); err == nil {
				configPath = path
				break
			}
		}
	}

	if configPath == "" {
		// No config file found, use environment variables only
		return cm.loadFromEnv()
	}

	// Read config file
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse JSON
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// Flatten nested config
	cm.data = cm.flatten(config)

	// Override with environment variables
	return cm.loadFromEnv()
}

// loadFromEnv loads configuration from environment variables
func (cm *configManager) loadFromEnv() error {
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := parts[1]

		// Only process environment variables with a specific prefix
		if strings.HasPrefix(key, "DOFFY_") {
			configKey := strings.TrimPrefix(key, "DOFFY_")
			configKey = strings.ToLower(configKey)
			configKey = strings.ReplaceAll(configKey, "_", ".")
			cm.data[configKey] = value
		}
	}

	return nil
}

// flatten flattens a nested map
func (cm *configManager) flatten(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for k, v := range m {
		switch child := v.(type) {
		case map[string]interface{}:
			nested := cm.flatten(child)
			for nk, nv := range nested {
				result[k+"."+nk] = nv
			}
		default:
			result[k] = v
		}
	}

	return result
}

// Get returns a configuration value
func (cm *configManager) Get(key string) interface{} {
	return cm.data[key]
}

// GetString returns a configuration value as string
func (cm *configManager) GetString(key string) string {
	if value, exists := cm.data[key]; exists {
		return fmt.Sprintf("%v", value)
	}
	return ""
}

// GetInt returns a configuration value as int
func (cm *configManager) GetInt(key string) int {
	if value, exists := cm.data[key]; exists {
		switch v := value.(type) {
		case int:
			return v
		case float64:
			return int(v)
		case string:
			if i, err := strconv.Atoi(v); err == nil {
				return i
			}
		}
	}
	return 0
}

// GetBool returns a configuration value as bool
func (cm *configManager) GetBool(key string) bool {
	if value, exists := cm.data[key]; exists {
		switch v := value.(type) {
		case bool:
			return v
		case string:
			return strings.ToLower(v) == "true" || v == "1"
		case int:
			return v != 0
		case float64:
			return v != 0
		}
	}
	return false
}

// GetFloat returns a configuration value as float64
func (cm *configManager) GetFloat(key string) float64 {
	if value, exists := cm.data[key]; exists {
		switch v := value.(type) {
		case float64:
			return v
		case int:
			return float64(v)
		case string:
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				return f
			}
		}
	}
	return 0
}

// Set sets a configuration value
func (cm *configManager) Set(key string, value interface{}) {
	cm.data[key] = value
}

// Has checks if a configuration key exists
func (cm *configManager) Has(key string) bool {
	_, exists := cm.data[key]
	return exists
}

// Unmarshal unmarshals the configuration into a struct
func (cm *configManager) Unmarshal(target interface{}) error {
	// Convert flat map to nested map
	nested := cm.nest(cm.data)

	data, err := json.Marshal(nested)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, target)
}

// nest converts a flat map to a nested map
func (cm *configManager) nest(flat map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range flat {
		parts := strings.Split(key, ".")
		current := result

		for i, part := range parts {
			if i == len(parts)-1 {
				// Last part, set the value
				current[part] = value
			} else {
				// Create nested map if it doesn't exist
				if _, exists := current[part]; !exists {
					current[part] = make(map[string]interface{})
				}
				current = current[part].(map[string]interface{})
			}
		}
	}

	return result
}

// LoadConfigWithDefaults loads configuration with default values
func LoadConfigWithDefaults(configPath string, defaults interface{}) (ConfigManager, error) {
	cm := NewConfigManager()

	// Set defaults
	if defaults != nil {
		v := reflect.ValueOf(defaults)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}

		if v.Kind() == reflect.Struct {
			t := v.Type()
			for i := 0; i < v.NumField(); i++ {
				field := t.Field(i)
				jsonTag := field.Tag.Get("json")
				if jsonTag != "" && jsonTag != "-" {
					cm.Set(jsonTag, v.Field(i).Interface())
				}
			}
		}
	}

	// Load from file and environment
	if err := cm.Load(configPath); err != nil {
		return nil, err
	}

	return cm, nil
}
