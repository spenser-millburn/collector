package core

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/sliink/collector/internal/model"
)

// ConfigManager handles loading, storing, and accessing configuration
type ConfigManager struct {
	config    map[string]interface{}
	watchers  map[string][]func(interface{})
	mutex     sync.RWMutex
	configFile string
	BaseComponent
}

// NewConfigManager creates a new configuration manager
func NewConfigManager() *ConfigManager {
	return &ConfigManager{
		config:        make(map[string]interface{}),
		watchers:      make(map[string][]func(interface{})),
		BaseComponent: NewBaseComponent("config_manager", "Configuration Manager"),
	}
}

// Initialize prepares the configuration manager for operation
func (m *ConfigManager) Initialize() bool {
	m.SetStatus(model.StatusInitialized)
	return true
}

// Start begins configuration manager operation
func (m *ConfigManager) Start() bool {
	m.SetStatus(model.StatusRunning)
	return true
}

// Stop halts configuration manager operation
func (m *ConfigManager) Stop() bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Clear all watchers
	m.watchers = make(map[string][]func(interface{}))
	
	m.SetStatus(model.StatusStopped)
	return true
}

// LoadConfig loads configuration from a file
func (m *ConfigManager) LoadConfig(configFile string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Read file
	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("error reading config file: %w", err)
	}

	// Parse JSON
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("error parsing config file: %w", err)
	}

	// Store config
	m.config = config
	m.configFile = configFile

	// Notify watchers for root path
	if watchers, exists := m.watchers[""]; exists {
		for _, callback := range watchers {
			go callback(m.config)
		}
	}

	return nil
}

// SaveConfig saves the current configuration to a file
func (m *ConfigManager) SaveConfig(configFile string) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// If no file specified, use the one we loaded from
	if configFile == "" {
		configFile = m.configFile
	}

	// If still no file, error
	if configFile == "" {
		return fmt.Errorf("no config file specified")
	}

	// Marshal JSON
	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return fmt.Errorf("error encoding config: %w", err)
	}

	// Write file
	if err := os.WriteFile(configFile, data, 0644); err != nil {
		return fmt.Errorf("error writing config file: %w", err)
	}

	return nil
}

// GetConfig retrieves a configuration value
func (m *ConfigManager) GetConfig(path string, defaultValue interface{}) interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// If no path, return entire config
	if path == "" {
		return m.config
	}

	// Split path into parts
	parts := strings.Split(path, ".")
	
	// Navigate through the config
	current := m.config
	
	for i, part := range parts {
		// Check if current is a map
		v, ok := current[part]
		if !ok {
			return defaultValue
		}

		// If we're at the last part, return the value
		if i == len(parts)-1 {
			return v
		}

		// Otherwise, ensure the value is a map and navigate deeper
		current, ok = v.(map[string]interface{})
		if !ok {
			return defaultValue
		}
	}

	return defaultValue
}

// SetConfig sets a configuration value
func (m *ConfigManager) SetConfig(path string, value interface{}) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// If no path, replace entire config
	if path == "" {
		if newConfig, ok := value.(map[string]interface{}); ok {
			m.config = newConfig
			
			// Notify watchers for root path
			if watchers, exists := m.watchers[""]; exists {
				for _, callback := range watchers {
					go callback(m.config)
				}
			}
			
			return nil
		}
		return fmt.Errorf("cannot set root config to non-map value")
	}

	// Split path into parts
	parts := strings.Split(path, ".")
	
	// Navigate to the parent of the target path
	current := m.config
	
	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]
		
		// Check if the current part exists and is a map
		v, exists := current[part]
		if !exists {
			// Create a new map for this part
			newMap := make(map[string]interface{})
			current[part] = newMap
			current = newMap
		} else {
			// Ensure the existing value is a map
			nextMap, ok := v.(map[string]interface{})
			if !ok {
				// Replace with a new map
				newMap := make(map[string]interface{})
				current[part] = newMap
				current = newMap
			} else {
				current = nextMap
			}
		}
	}

	// Set the value at the target path
	lastPart := parts[len(parts)-1]
	current[lastPart] = value
	
	// Notify watchers for this path and parent paths
	for i := 0; i <= len(parts); i++ {
		subPath := strings.Join(parts[:i], ".")
		if watchers, exists := m.watchers[subPath]; exists {
			var watchValue interface{}
			if i == 0 {
				watchValue = m.config
			} else {
				watchValue = m.GetConfig(subPath, nil)
			}
			
			for _, callback := range watchers {
				go callback(watchValue)
			}
		}
	}
	
	return nil
}

// WatchConfig registers a callback for configuration changes
func (m *ConfigManager) WatchConfig(path string, callback func(interface{})) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.watchers[path] == nil {
		m.watchers[path] = make([]func(interface{}), 0)
	}
	
	m.watchers[path] = append(m.watchers[path], callback)
	
	// Immediately call the callback with the current value
	go callback(m.GetConfig(path, nil))
}