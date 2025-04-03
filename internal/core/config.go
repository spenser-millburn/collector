package core

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/sliink/collector/internal/model"
)

// configUpdate represents an update to the configuration
type configUpdate struct {
	path  string
	value interface{}
}

// ConfigManager handles loading, storing, and accessing configuration
type ConfigManager struct {
	config         map[string]interface{}
	configFile     string
	enableWatchers bool
	updateChan     chan configUpdate
	subscribers    map[string][]chan interface{}
	mutex          sync.RWMutex // Still needed for config map access
	done           chan struct{}
	BaseComponent
}

// NewConfigManager creates a new configuration manager
func NewConfigManager() *ConfigManager {
	cm := &ConfigManager{
		config:         make(map[string]interface{}),
		subscribers:    make(map[string][]chan interface{}),
		enableWatchers: true,
		updateChan:     make(chan configUpdate, 100),
		done:           make(chan struct{}),
		BaseComponent:  NewBaseComponent("config_manager", "Configuration Manager"),
	}
	
	// Start the update processor
	go cm.processUpdates()
	
	return cm
}

// NewConfigManagerWithOptions creates a new configuration manager with specified options
func NewConfigManagerWithOptions(enableWatchers bool) *ConfigManager {
	cm := &ConfigManager{
		config:         make(map[string]interface{}),
		subscribers:    make(map[string][]chan interface{}),
		enableWatchers: enableWatchers,
		updateChan:     make(chan configUpdate, 100),
		done:           make(chan struct{}),
		BaseComponent:  NewBaseComponent("config_manager", "Configuration Manager"),
	}
	
	// Start the update processor if watchers are enabled
	if enableWatchers {
		go cm.processUpdates()
	}
	
	return cm
}

// processUpdates handles configuration updates and notifies subscribers
func (m *ConfigManager) processUpdates() {
	for {
		select {
		case update := <-m.updateChan:
			// Apply the update
			m.applyUpdate(update)
			
			// Notify subscribers about the update
			m.notifySubscribers(update.path)
		case <-m.done:
			return
		}
	}
}

// applyUpdate applies a configuration update
func (m *ConfigManager) applyUpdate(update configUpdate) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	path := update.path
	value := update.value
	
	// If no path, replace entire config
	if path == "" {
		if newConfig, ok := value.(map[string]interface{}); ok {
			m.config = newConfig
		}
		return
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
}

// notifySubscribers notifies subscribers about configuration changes
func (m *ConfigManager) notifySubscribers(path string) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	// Split path into parts
	parts := strings.Split(path, ".")
	
	// Notify subscribers for this path and parent paths
	for i := 0; i <= len(parts); i++ {
		subPath := strings.Join(parts[:i], ".")
		
		if subscribers, exists := m.subscribers[subPath]; exists {
			var watchValue interface{}
			if i == 0 {
				watchValue = m.config
			} else {
				// Need to get the value at this subpath
				watchValue = m.getConfigInternal(subPath, nil)
			}
			
			// Send to all subscribers of this path
			for _, ch := range subscribers {
				// Use a goroutine to avoid blocking if a subscriber's channel is full
				go func(subscriber chan interface{}, val interface{}) {
					select {
					case subscriber <- val:
						// Value sent
					default:
						// Channel is full, drop the update
					}
				}(ch, watchValue)
			}
		}
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
	// Signal the update processor to stop
	close(m.done)
	
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	// Close all subscriber channels
	for path, subscribers := range m.subscribers {
		for _, ch := range subscribers {
			close(ch)
		}
		delete(m.subscribers, path)
	}
	
	m.SetStatus(model.StatusStopped)
	return true
}

// LoadConfig loads configuration from a file
func (m *ConfigManager) LoadConfig(configFile string) error {
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
	
	m.mutex.Lock()
	m.configFile = configFile
	// Update config directly to ensure tests can access it right away
	m.config = config
	m.mutex.Unlock()
	
	// Send update for the entire config
	if m.enableWatchers {
		m.updateChan <- configUpdate{
			path:  "",
			value: config,
		}
	}
	
	return nil
}

// SaveConfig saves the current configuration to a file
func (m *ConfigManager) SaveConfig(configFile string) error {
	m.mutex.RLock()
	
	// If no file specified, use the one we loaded from
	if configFile == "" {
		configFile = m.configFile
	}
	
	// If still no file, error
	if configFile == "" {
		m.mutex.RUnlock()
		return fmt.Errorf("no config file specified")
	}
	
	// Make a copy of the config to avoid holding the lock during I/O
	configCopy := make(map[string]interface{})
	for k, v := range m.config {
		configCopy[k] = v
	}
	m.mutex.RUnlock()
	
	// Marshal JSON
	data, err := json.MarshalIndent(configCopy, "", "  ")
	if err != nil {
		return fmt.Errorf("error encoding config: %w", err)
	}
	
	// Write file
	if err := os.WriteFile(configFile, data, 0644); err != nil {
		return fmt.Errorf("error writing config file: %w", err)
	}
	
	return nil
}

// getConfigInternal retrieves a configuration value (internal use)
func (m *ConfigManager) getConfigInternal(path string, defaultValue interface{}) interface{} {
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

// GetConfig retrieves a configuration value
func (m *ConfigManager) GetConfig(path string, defaultValue interface{}) interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	return m.getConfigInternal(path, defaultValue)
}

// SetConfig sets a configuration value
func (m *ConfigManager) SetConfig(path string, value interface{}) error {
	// Validate root config updates
	if path == "" {
		if _, ok := value.(map[string]interface{}); !ok {
			return fmt.Errorf("cannot set root config to non-map value")
		}
	}
	
	// First update directly to ensure tests can access it right away
	m.mutex.Lock()
	if path == "" {
		if newConfig, ok := value.(map[string]interface{}); ok {
			m.config = newConfig
		}
	} else {
		// Split path and navigate to create the path
		parts := strings.Split(path, ".")
		current := m.config
		
		for i := 0; i < len(parts)-1; i++ {
			part := parts[i]
			
			// Create or navigate the path
			v, exists := current[part]
			if !exists {
				newMap := make(map[string]interface{})
				current[part] = newMap
				current = newMap
			} else {
				nextMap, ok := v.(map[string]interface{})
				if !ok {
					newMap := make(map[string]interface{})
					current[part] = newMap
					current = newMap
				} else {
					current = nextMap
				}
			}
		}
		
		// Set the value
		lastPart := parts[len(parts)-1]
		current[lastPart] = value
	}
	m.mutex.Unlock()
	
	// Then send update via channel if watchers are enabled
	if m.enableWatchers {
		m.updateChan <- configUpdate{
			path:  path,
			value: value,
		}
	}
	
	return nil
}

// WatchConfig registers a callback for configuration changes
func (m *ConfigManager) WatchConfig(path string, callback func(interface{})) {
	if !m.enableWatchers {
		return
	}
	
	// Create a channel for this subscription
	ch := make(chan interface{}, 10)
	
	// Register the subscriber
	m.mutex.Lock()
	if m.subscribers[path] == nil {
		m.subscribers[path] = make([]chan interface{}, 0)
	}
	m.subscribers[path] = append(m.subscribers[path], ch)
	
	// Get initial value
	initialValue := m.getConfigInternal(path, nil)
	m.mutex.Unlock()
	
	// Start a goroutine to handle updates for this subscriber
	go func() {
		// Send initial value
		callback(initialValue)
		
		// Process future updates
		for value := range ch {
			callback(value)
		}
	}()
}