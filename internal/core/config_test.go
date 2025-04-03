package core

import (
	"encoding/json"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/sliink/collector/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestNewConfigManager(t *testing.T) {
	manager := NewConfigManager()
	
	assert.NotNil(t, manager)
	assert.NotNil(t, manager.config)
	assert.NotNil(t, manager.watchers)
	assert.Equal(t, "config_manager", manager.ID())
	assert.Equal(t, "Configuration Manager", manager.Name())
}

func TestConfigManagerLifecycle(t *testing.T) {
	manager := NewConfigManager()
	
	t.Run("Initialize sets correct status", func(t *testing.T) {
		success := manager.Initialize()
		assert.True(t, success)
		assert.Equal(t, model.StatusInitialized, manager.GetStatus())
	})
	
	t.Run("Start sets correct status", func(t *testing.T) {
		success := manager.Start()
		assert.True(t, success)
		assert.Equal(t, model.StatusRunning, manager.GetStatus())
	})
	
	t.Run("Stop clears watchers and sets correct status", func(t *testing.T) {
		// Add a watcher first
		manager.WatchConfig("test_path", func(interface{}) {
			// We don't need to do anything here, just registering a watcher
		})
		assert.NotEmpty(t, manager.watchers)
		
		// Now stop and verify watchers are cleared
		success := manager.Stop()
		assert.True(t, success)
		assert.Empty(t, manager.watchers)
		assert.Equal(t, model.StatusStopped, manager.GetStatus())
	})
}

func TestLoadConfig(t *testing.T) {
	manager := NewConfigManager()
	manager.Initialize()
	
	// Create a temporary config file
	configContent := `{
		"system": {
			"id": "test-collector",
			"version": "1.0.0"
		},
		"plugins": {
			"inputs": [
				{
					"id": "file-input",
					"type": "file",
					"paths": ["/var/log/*.log"]
				}
			]
		}
	}`
	
	tempFile, err := os.CreateTemp("", "config-*.json")
	assert.NoError(t, err)
	defer os.Remove(tempFile.Name())
	
	_, err = tempFile.Write([]byte(configContent))
	assert.NoError(t, err)
	tempFile.Close()
	
	t.Run("LoadConfig loads valid JSON file", func(t *testing.T) {
		err := manager.LoadConfig(tempFile.Name())
		assert.NoError(t, err)
		
		// Verify config was loaded correctly
		system := manager.GetConfig("system", nil)
		assert.NotNil(t, system)
		
		systemMap, ok := system.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "test-collector", systemMap["id"])
		
		// Check nested path
		version := manager.GetConfig("system.version", nil)
		assert.Equal(t, "1.0.0", version)
	})
	
	t.Run("LoadConfig returns error for nonexistent file", func(t *testing.T) {
		err := manager.LoadConfig("nonexistent-file.json")
		assert.Error(t, err)
	})
	
	t.Run("LoadConfig returns error for invalid JSON", func(t *testing.T) {
		invalidFile, err := os.CreateTemp("", "invalid-*.json")
		assert.NoError(t, err)
		defer os.Remove(invalidFile.Name())
		
		_, err = invalidFile.Write([]byte("{invalid json"))
		assert.NoError(t, err)
		invalidFile.Close()
		
		err = manager.LoadConfig(invalidFile.Name())
		assert.Error(t, err)
	})
	
	t.Run("LoadConfig notifies root path watchers", func(t *testing.T) {
		manager := NewConfigManager()
		
		var wg sync.WaitGroup
		wg.Add(1)
		
		var notifiedConfig map[string]interface{}
		manager.WatchConfig("", func(value interface{}) {
			if config, ok := value.(map[string]interface{}); ok {
				notifiedConfig = config
				wg.Done()
			}
		})
		
		err := manager.LoadConfig(tempFile.Name())
		assert.NoError(t, err)
		
		// Wait for notification with timeout
		waitDone := make(chan struct{})
		go func() {
			wg.Wait()
			close(waitDone)
		}()
		
		select {
		case <-waitDone:
			// Continue with assertions
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Timed out waiting for watcher notification")
		}
		
		// Verify watcher was notified with correct config
		assert.NotNil(t, notifiedConfig)
		assert.Contains(t, notifiedConfig, "system")
	})
}

func TestSaveConfig(t *testing.T) {
	manager := NewConfigManager()
	manager.Initialize()
	
	// Set some config values
	config := map[string]interface{}{
		"system": map[string]interface{}{
			"id":      "test-collector",
			"version": "1.0.0",
		},
	}
	manager.SetConfig("", config)
	
	t.Run("SaveConfig writes config to file", func(t *testing.T) {
		tempFile, err := os.CreateTemp("", "saved-config-*.json")
		assert.NoError(t, err)
		tempFile.Close()
		defer os.Remove(tempFile.Name())
		
		err = manager.SaveConfig(tempFile.Name())
		assert.NoError(t, err)
		
		// Read the file back and verify content
		data, err := os.ReadFile(tempFile.Name())
		assert.NoError(t, err)
		
		var savedConfig map[string]interface{}
		err = json.Unmarshal(data, &savedConfig)
		assert.NoError(t, err)
		
		assert.Contains(t, savedConfig, "system")
		systemMap, ok := savedConfig["system"].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "test-collector", systemMap["id"])
	})
	
	t.Run("SaveConfig uses loaded file path if not specified", func(t *testing.T) {
		tempFile, err := os.CreateTemp("", "loaded-config-*.json")
		assert.NoError(t, err)
		tempFile.Close()
		defer os.Remove(tempFile.Name())
		
		// First load the config
		err = manager.LoadConfig(tempFile.Name())
		assert.NoError(t, err)
		
		// Then save without specifying path
		err = manager.SaveConfig("")
		assert.NoError(t, err)
		
		// Read the file back and verify content
		data, err := os.ReadFile(tempFile.Name())
		assert.NoError(t, err)
		assert.NotEmpty(t, data)
	})
	
	t.Run("SaveConfig returns error with no path", func(t *testing.T) {
		// New manager with no loaded config
		manager := NewConfigManager()
		
		err := manager.SaveConfig("")
		assert.Error(t, err)
	})
}

func TestGetConfig(t *testing.T) {
	manager := NewConfigManager()
	
	// Set up test config
	config := map[string]interface{}{
		"system": map[string]interface{}{
			"id":      "test-collector",
			"version": "1.0.0",
			"nested": map[string]interface{}{
				"deeply": map[string]interface{}{
					"property": "value",
				},
			},
		},
		"array_value": []interface{}{1, 2, 3},
	}
	manager.config = config
	
	t.Run("GetConfig with empty path returns entire config", func(t *testing.T) {
		result := manager.GetConfig("", nil)
		assert.Equal(t, config, result)
	})
	
	t.Run("GetConfig with valid path returns correct value", func(t *testing.T) {
		result := manager.GetConfig("system.id", nil)
		assert.Equal(t, "test-collector", result)
	})
	
	t.Run("GetConfig with nested path returns correct value", func(t *testing.T) {
		result := manager.GetConfig("system.nested.deeply.property", nil)
		assert.Equal(t, "value", result)
	})
	
	t.Run("GetConfig with non-existent path returns default value", func(t *testing.T) {
		result := manager.GetConfig("nonexistent", "default")
		assert.Equal(t, "default", result)
	})
	
	t.Run("GetConfig with partial path returns default value", func(t *testing.T) {
		result := manager.GetConfig("system.nonexistent", "default")
		assert.Equal(t, "default", result)
	})
	
	t.Run("GetConfig with array path returns default value", func(t *testing.T) {
		// Non-map values can't be traversed
		result := manager.GetConfig("array_value.0", "default")
		assert.Equal(t, "default", result)
	})
}

func TestSetConfig(t *testing.T) {
	manager := NewConfigManager()
	
	t.Run("SetConfig with empty path sets entire config", func(t *testing.T) {
		newConfig := map[string]interface{}{
			"system": map[string]interface{}{
				"id": "new-id",
			},
		}
		
		err := manager.SetConfig("", newConfig)
		assert.NoError(t, err)
		assert.Equal(t, newConfig, manager.config)
	})
	
	t.Run("SetConfig with non-map value at root returns error", func(t *testing.T) {
		err := manager.SetConfig("", "string-value")
		assert.Error(t, err)
		
		err = manager.SetConfig("", 42)
		assert.Error(t, err)
	})
	
	t.Run("SetConfig with simple path creates path and sets value", func(t *testing.T) {
		err := manager.SetConfig("new_key", "new_value")
		assert.NoError(t, err)
		
		result := manager.GetConfig("new_key", nil)
		assert.Equal(t, "new_value", result)
	})
	
	t.Run("SetConfig with nested path creates intermediate paths", func(t *testing.T) {
		err := manager.SetConfig("deeply.nested.path", 42)
		assert.NoError(t, err)
		
		result := manager.GetConfig("deeply.nested.path", nil)
		assert.Equal(t, 42, result)
		
		// Check intermediate maps were created
		deeply := manager.GetConfig("deeply", nil)
		deeplyMap, ok := deeply.(map[string]interface{})
		assert.True(t, ok)
		assert.Contains(t, deeplyMap, "nested")
	})
	
	t.Run("SetConfig overwrites non-map values in path", func(t *testing.T) {
		// First set a simple value
		err := manager.SetConfig("to_be_overwritten", "simple_value")
		assert.NoError(t, err)
		
		// Now set a nested path that would require this to be a map
		err = manager.SetConfig("to_be_overwritten.nested", "nested_value")
		assert.NoError(t, err)
		
		// The simple value should be replaced with a map
		result := manager.GetConfig("to_be_overwritten", nil)
		resultMap, ok := result.(map[string]interface{})
		assert.True(t, ok)
		assert.Contains(t, resultMap, "nested")
	})
}

func TestWatchConfig(t *testing.T) {
	manager := NewConfigManager()
	manager.Initialize()
	
	t.Run("WatchConfig registers callback and calls immediately", func(t *testing.T) {
		manager.config = map[string]interface{}{
			"test_key": "test_value",
		}
		
		var wg sync.WaitGroup
		wg.Add(1)
		
		var notifiedValue interface{}
		manager.WatchConfig("test_key", func(value interface{}) {
			notifiedValue = value
			wg.Done()
		})
		
		// Wait for notification with timeout
		waitDone := make(chan struct{})
		go func() {
			wg.Wait()
			close(waitDone)
		}()
		
		select {
		case <-waitDone:
			// Continue with assertions
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Timed out waiting for watcher notification")
		}
		
		assert.Equal(t, "test_value", notifiedValue)
	})
	
	t.Run("SetConfig notifies watchers for path and parent paths", func(t *testing.T) {
		manager := NewConfigManager()
		manager.Initialize()
		
		var rootCalled, parentCalled, pathCalled bool
		var rootWg, parentWg, pathWg sync.WaitGroup
		rootWg.Add(1)
		parentWg.Add(1)
		pathWg.Add(1)
		
		manager.WatchConfig("", func(value interface{}) {
			rootCalled = true
			rootWg.Done()
		})
		
		manager.WatchConfig("parent", func(value interface{}) {
			parentCalled = true
			parentWg.Done()
		})
		
		manager.WatchConfig("parent.child", func(value interface{}) {
			pathCalled = true
			pathWg.Done()
		})
		
		// Set a value that should trigger all watchers
		err := manager.SetConfig("parent.child", "test_value")
		assert.NoError(t, err)
		
		// Wait for notifications with timeout
		timeout := 100 * time.Millisecond
		
		waitWithTimeout := func(wg *sync.WaitGroup, timeout time.Duration) bool {
			c := make(chan struct{})
			go func() {
				wg.Wait()
				close(c)
			}()
			
			select {
			case <-c:
				return true
			case <-time.After(timeout):
				return false
			}
		}
		
		assert.True(t, waitWithTimeout(&rootWg, timeout), "Root watcher not called")
		assert.True(t, waitWithTimeout(&parentWg, timeout), "Parent watcher not called")
		assert.True(t, waitWithTimeout(&pathWg, timeout), "Path watcher not called")
		
		assert.True(t, rootCalled)
		assert.True(t, parentCalled)
		assert.True(t, pathCalled)
	})
	
	t.Run("Multiple watchers for same path all get notified", func(t *testing.T) {
		manager := NewConfigManager()
		
		var called1, called2 bool
		var wg sync.WaitGroup
		wg.Add(2)
		
		manager.WatchConfig("test_path", func(value interface{}) {
			called1 = true
			wg.Done()
		})
		
		manager.WatchConfig("test_path", func(value interface{}) {
			called2 = true
			wg.Done()
		})
		
		// Set a value to trigger watchers
		err := manager.SetConfig("test_path", "test_value")
		assert.NoError(t, err)
		
		// Wait for notifications with timeout
		waitDone := make(chan struct{})
		go func() {
			wg.Wait()
			close(waitDone)
		}()
		
		select {
		case <-waitDone:
			// Continue with assertions
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Timed out waiting for watcher notifications")
		}
		
		assert.True(t, called1)
		assert.True(t, called2)
	})
}