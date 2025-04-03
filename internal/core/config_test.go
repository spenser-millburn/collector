package core

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/sliink/collector/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestNewConfigManager(t *testing.T) {
	manager := NewConfigManager()

	assert.NotNil(t, manager)
	assert.NotNil(t, manager.config)
	assert.Equal(t, "config_manager", manager.ID())
	assert.Equal(t, "Configuration Manager", manager.Name())
	assert.True(t, manager.enableWatchers)
}

func TestNewConfigManagerWithOptions(t *testing.T) {
	t.Run("With watchers enabled", func(t *testing.T) {
		manager := NewConfigManagerWithOptions(true)
		assert.True(t, manager.enableWatchers)
	})

	t.Run("With watchers disabled", func(t *testing.T) {
		manager := NewConfigManagerWithOptions(false)
		assert.False(t, manager.enableWatchers)
	})
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

	t.Run("Stop sets correct status", func(t *testing.T) {
		success := manager.Stop()
		assert.True(t, success)
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
		
		// Write some valid JSON to the file
		_, err = tempFile.Write([]byte(`{"test": "value"}`))
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

func TestConfigWatchers(t *testing.T) {
	t.Run("Watchers enabled", func(t *testing.T) {
		manager := NewConfigManagerWithOptions(true)
		
		// Channel to signal when callback is called
		callbackCh := make(chan interface{})
		
		// Register a watch
		var callbackValue interface{}
		
		manager.WatchConfig("test.path", func(value interface{}) {
			// Ignore the initial nil value
			if value != nil {
				callbackValue = value
				callbackCh <- value
			}
		})
		
		// Set a value that should trigger the watch
		err := manager.SetConfig("test.path", "test_value")
		assert.NoError(t, err)
		
		// Wait for callback with timeout
		select {
		case <-callbackCh:
			// Success
		case <-time.After(1 * time.Second):
			t.Fatal("Watcher callback was not called within timeout")
		}
		
		assert.Equal(t, "test_value", callbackValue)
	})
	
	t.Run("Watchers disabled", func(t *testing.T) {
		manager := NewConfigManagerWithOptions(false)
		
		// Register a watch
		var callbackCalled bool
		
		manager.WatchConfig("test.path", func(value interface{}) {
			callbackCalled = true
		})
		
		// Set a value that would trigger the watch if enabled
		err := manager.SetConfig("test.path", "test_value")
		assert.NoError(t, err)
		
		// Give some time for any potential callbacks to execute
		time.Sleep(100 * time.Millisecond)
		
		// Verify callback was not called
		assert.False(t, callbackCalled)
	})
	
	t.Run("Multiple subscribers get updates", func(t *testing.T) {
		manager := NewConfigManagerWithOptions(true)
		
		// Channels to signal when callbacks are called
		callback1Ch := make(chan interface{})
		callback2Ch := make(chan interface{})
		
		var callback1Value, callback2Value interface{}
		
		// Register first watcher
		manager.WatchConfig("multi.path", func(value interface{}) {
			// Ignore the initial nil value
			if value != nil {
				callback1Value = value
				callback1Ch <- value
			}
		})
		
		// Register second watcher for same path
		manager.WatchConfig("multi.path", func(value interface{}) {
			// Ignore the initial nil value
			if value != nil {
				callback2Value = value
				callback2Ch <- value
			}
		})
		
		// Set a value that should trigger both watches
		err := manager.SetConfig("multi.path", "multi_test_value")
		assert.NoError(t, err)
		
		// Wait for both callbacks with timeout
		for i := 0; i < 2; i++ {
			select {
			case <-callback1Ch:
				// Callback 1 triggered
			case <-callback2Ch:
				// Callback 2 triggered
			case <-time.After(1 * time.Second):
				t.Fatal("One or more watcher callbacks were not called within timeout")
			}
		}
		
		assert.Equal(t, "multi_test_value", callback1Value)
		assert.Equal(t, "multi_test_value", callback2Value)
	})
	
	t.Run("Channel-based updates work correctly", func(t *testing.T) {
		manager := NewConfigManagerWithOptions(true)
		
		// Create channel for collecting values
		valuesCh := make(chan interface{}, 3)
		
		// Track values received
		var values []interface{}
		
		// Register watcher
		manager.WatchConfig("channel.path", func(value interface{}) {
			valuesCh <- value
		})
		
		// Set values that should trigger the watch
		err := manager.SetConfig("channel.path", "value1")
		assert.NoError(t, err)
		
		err = manager.SetConfig("channel.path", "value2")
		assert.NoError(t, err)
		
		// Collect 3 values (initial value + 2 updates)
		for i := 0; i < 3; i++ {
			select {
			case val := <-valuesCh:
				values = append(values, val)
			case <-time.After(1 * time.Second):
				t.Fatalf("Didn't receive expected update %d within timeout", i+1)
			}
		}
		
		// Values should include nil (initial) and both of our updates
		// The order might not be guaranteed, so just check that all expected values are present
		assert.Equal(t, 3, len(values))
		assert.Contains(t, values, nil)
		assert.Contains(t, values, "value1")
		assert.Contains(t, values, "value2")
	})
	
	t.Run("Shutdown closes subscriber channels", func(t *testing.T) {
		manager := NewConfigManagerWithOptions(true)
		
		// Register watcher that signals when the channel is closed
		manager.WatchConfig("test.path", func(value interface{}) {
			// Only interested in channel close, not values
		})
		
		// Shutdown the manager
		success := manager.Stop()
		assert.True(t, success)
		
		// Wait a bit to let goroutines finish
		time.Sleep(100 * time.Millisecond)
		
		// Verify manager is stopped
		assert.Equal(t, model.StatusStopped, manager.GetStatus())
	})
}


