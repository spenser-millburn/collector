package core

import (
	"sync"
	"testing"
	"time"

	"github.com/sliink/collector/internal/model"
	"github.com/stretchr/testify/assert"
)

// mockInvalidPlugin implements model.Plugin for testing validation and registration failures
type mockInvalidPlugin struct {
	id                    string
	name                  string
	status                model.ComponentStatus
	validationResult      bool
	coreRegistrationResult bool
}

func (m *mockInvalidPlugin) ID() string {
	return m.id
}

func (m *mockInvalidPlugin) Name() string {
	return m.name
}

func (m *mockInvalidPlugin) GetType() model.PluginType {
	return model.InputPluginType
}

func (m *mockInvalidPlugin) GetStatus() model.ComponentStatus {
	return m.status
}

func (m *mockInvalidPlugin) SetStatus(status model.ComponentStatus) {
	m.status = status
}

func (m *mockInvalidPlugin) Configure(config map[string]interface{}) bool {
	return true
}

func (m *mockInvalidPlugin) Initialize() bool {
	m.status = model.StatusInitialized
	return true
}

func (m *mockInvalidPlugin) Start() bool {
	m.status = model.StatusRunning
	return true
}

func (m *mockInvalidPlugin) Stop() bool {
	m.status = model.StatusStopped
	return true
}

func (m *mockInvalidPlugin) Validate() bool {
	return m.validationResult
}

func (m *mockInvalidPlugin) RegisterWithCore(core model.CoreAPI) bool {
	return m.coreRegistrationResult
}

func TestNewCore(t *testing.T) {
	core := NewCore()
	
	assert.NotNil(t, core)
	assert.NotNil(t, core.inputChannels)
	assert.NotNil(t, core.outputChannels)
	assert.NotNil(t, core.ctx)
	assert.Equal(t, "core", core.ID())
	assert.Equal(t, "Core System", core.Name())
}

func TestCoreInitialize(t *testing.T) {
	core := NewCore()
	
	success := core.Initialize()
	assert.True(t, success)
	
	// Check that all core components were created
	assert.NotNil(t, core.eventBus)
	assert.NotNil(t, core.registry)
	assert.NotNil(t, core.configManager)
	assert.NotNil(t, core.healthMonitor)
	assert.NotNil(t, core.bufferManager)
	assert.NotNil(t, core.pipeline)
	
	// Check component status
	assert.Equal(t, model.StatusInitialized, core.GetStatus())
}

func TestCoreStartStop(t *testing.T) {
	core := NewCore()
	core.Initialize()
	
	t.Run("Start initializes and starts all components", func(t *testing.T) {
		success := core.Start()
		assert.True(t, success)
		assert.Equal(t, model.StatusRunning, core.GetStatus())
		
		// Check all core components are running
		assert.Equal(t, model.StatusRunning, core.eventBus.GetStatus())
		assert.Equal(t, model.StatusRunning, core.registry.GetStatus())
		assert.Equal(t, model.StatusRunning, core.configManager.GetStatus())
		assert.Equal(t, model.StatusRunning, core.healthMonitor.GetStatus())
		assert.Equal(t, model.StatusRunning, core.bufferManager.GetStatus())
		assert.Equal(t, model.StatusRunning, core.pipeline.GetStatus())
	})
	
	t.Run("Stop halts all components", func(t *testing.T) {
		success := core.Stop()
		assert.True(t, success)
		assert.Equal(t, model.StatusStopped, core.GetStatus())
		
		// Check all core components are stopped
		assert.Equal(t, model.StatusStopped, core.eventBus.GetStatus())
		assert.Equal(t, model.StatusStopped, core.registry.GetStatus())
		assert.Equal(t, model.StatusStopped, core.configManager.GetStatus())
		assert.Equal(t, model.StatusStopped, core.healthMonitor.GetStatus())
		assert.Equal(t, model.StatusStopped, core.bufferManager.GetStatus())
		assert.Equal(t, model.StatusStopped, core.pipeline.GetStatus())
	})
}

func TestCoreGetComponent(t *testing.T) {
	core := NewCore()
	core.Initialize()
	
	t.Run("GetComponent returns core components", func(t *testing.T) {
		// Check each core component
		component, exists := core.GetComponent("event_bus")
		assert.True(t, exists)
		assert.Equal(t, core.eventBus, component)
		
		component, exists = core.GetComponent("plugin_registry")
		assert.True(t, exists)
		assert.Equal(t, core.registry, component)
		
		component, exists = core.GetComponent("data_pipeline")
		assert.True(t, exists)
		assert.Equal(t, core.pipeline, component)
		
		component, exists = core.GetComponent("buffer_manager")
		assert.True(t, exists)
		assert.Equal(t, core.bufferManager, component)
		
		component, exists = core.GetComponent("config_manager")
		assert.True(t, exists)
		assert.Equal(t, core.configManager, component)
		
		component, exists = core.GetComponent("health_monitor")
		assert.True(t, exists)
		assert.Equal(t, core.healthMonitor, component)
		
		component, exists = core.GetComponent("core")
		assert.True(t, exists)
		assert.Equal(t, core, component)
	})
	
	t.Run("GetComponent returns registered plugins", func(t *testing.T) {
		// Register a plugin
		plugin := &mockInputPlugin{id: "test_plugin", name: "Test Plugin"}
		core.registry.RegisterPlugin(plugin)
		
		component, exists := core.GetComponent("test_plugin")
		assert.True(t, exists)
		assert.Equal(t, plugin, component)
	})
	
	t.Run("GetComponent returns false for nonexistent component", func(t *testing.T) {
		_, exists := core.GetComponent("nonexistent")
		assert.False(t, exists)
	})
}

func TestCoreGetters(t *testing.T) {
	core := NewCore()
	core.Initialize()
	
	t.Run("GetDataPipeline returns pipeline", func(t *testing.T) {
		pipeline := core.GetDataPipeline()
		assert.Equal(t, core.pipeline, pipeline)
	})
	
	t.Run("GetConfigManager returns config manager", func(t *testing.T) {
		configManager := core.GetConfigManager()
		assert.Equal(t, core.configManager, configManager)
	})
}

func TestCoreRegisterPlugin(t *testing.T) {
	core := NewCore()
	core.Initialize()
	
	t.Run("RegisterPlugin fails with nil plugin", func(t *testing.T) {
		err := core.RegisterPlugin(nil)
		assert.Error(t, err)
	})
	
	t.Run("RegisterPlugin fails when validation fails", func(t *testing.T) {
		plugin := &mockInvalidPlugin{
			id: "invalid", 
			name: "Invalid Plugin", 
			validationResult: false,
			coreRegistrationResult: true,
		}
		
		err := core.RegisterPlugin(plugin)
		assert.Error(t, err)
	})
	
	t.Run("RegisterPlugin fails when registration with core fails", func(t *testing.T) {
		plugin := &mockInvalidPlugin{
			id: "invalid", 
			name: "Invalid Plugin",
			validationResult: true, 
			coreRegistrationResult: false,
		}
		
		err := core.RegisterPlugin(plugin)
		assert.Error(t, err)
	})
	
	t.Run("RegisterPlugin succeeds with valid plugin", func(t *testing.T) {
		plugin := &mockInputPlugin{id: "valid", name: "Valid Plugin"}
		
		err := core.RegisterPlugin(plugin)
		assert.NoError(t, err)
		
		// Verify plugin is in registry
		p, exists := core.registry.GetPlugin("valid")
		assert.True(t, exists)
		assert.Equal(t, plugin, p)
		
		// Verify plugin is registered with health monitor
		health := core.healthMonitor.GetHealthStatus()
		assert.Contains(t, health.Components, "valid")
	})
}

func TestCoreProcessBatch(t *testing.T) {
	core := NewCore()
	core.Initialize()
	core.Start()
	
	// Create a test batch
	batch := createTestBatch(3)
	
	t.Run("ProcessBatch handles nil batch", func(t *testing.T) {
		result := core.ProcessBatch(nil)
		assert.Nil(t, result)
	})
	
	t.Run("ProcessBatch handles empty batch", func(t *testing.T) {
		emptyBatch := model.NewDataBatch(model.LogTelemetryType)
		result := core.ProcessBatch(emptyBatch)
		assert.Nil(t, result)
	})
	
	t.Run("ProcessBatch publishes events", func(t *testing.T) {
		// Subscribe to events
		var receivedReceived, receivedProcessed bool
		var wg sync.WaitGroup
		wg.Add(2)
		
		core.eventBus.Subscribe(model.EventDataReceived, "test", func(event Event) {
			receivedReceived = true
			wg.Done()
		})
		
		core.eventBus.Subscribe(model.EventDataProcessed, "test", func(event Event) {
			receivedProcessed = true
			wg.Done()
		})
		
		// Process a batch
		result := core.ProcessBatch(batch)
		assert.Equal(t, batch, result) // No processors registered, should return original
		
		// Wait for events with timeout
		waitDone := make(chan struct{})
		go func() {
			wg.Wait()
			close(waitDone)
		}()
		
		select {
		case <-waitDone:
			// Continue with assertions
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Timed out waiting for event notifications")
		}
		
		assert.True(t, receivedReceived)
		assert.True(t, receivedProcessed)
	})
	
	t.Run("ProcessBatch uses pipeline", func(t *testing.T) {
		// Create a processor that doubles batch size
		processor := newMockProcessorPlugin("doubler", "Doubler", func(batch *model.DataBatch) *model.DataBatch {
			newBatch := model.NewDataBatch(batch.BatchType)
			for _, point := range batch.Points {
				newBatch.AddPoint(point)
				newBatch.AddPoint(point)
			}
			return newBatch
		})
		
		// Register the processor
		err := core.RegisterPlugin(processor)
		assert.NoError(t, err)
		
		// Create a pipeline using the processor
		err = core.pipeline.CreatePipeline(model.LogTelemetryType, []string{"doubler"})
		assert.NoError(t, err)
		
		// Process a batch through the pipeline
		result := core.ProcessBatch(batch)
		assert.NotNil(t, result)
		assert.Equal(t, 6, result.Size()) // Doubled by processor
	})
}

func TestPublishEvent(t *testing.T) {
	core := NewCore()
	core.Initialize()
	
	t.Run("PublishEvent does nothing with nil eventBus", func(t *testing.T) {
		// Save and then nil out the eventBus
		eventBus := core.eventBus
		core.eventBus = nil
		
		// This should not panic
		core.PublishEvent(model.EventError, "test", "data")
		
		// Restore the eventBus
		core.eventBus = eventBus
	})
	
	t.Run("PublishEvent sends event to eventBus", func(t *testing.T) {
		var receivedEvent Event
		var wg sync.WaitGroup
		wg.Add(1)
		
		core.eventBus.Subscribe(model.EventError, "test", func(event Event) {
			receivedEvent = event
			wg.Done()
		})
		
		core.PublishEvent(model.EventError, "source", "test_data")
		
		// Wait for event with timeout
		waitDone := make(chan struct{})
		go func() {
			wg.Wait()
			close(waitDone)
		}()
		
		select {
		case <-waitDone:
			// Continue with assertions
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Timed out waiting for event notification")
		}
		
		assert.Equal(t, model.EventError, receivedEvent.Type)
		assert.Equal(t, "source", receivedEvent.SourceID)
		assert.Equal(t, "test_data", receivedEvent.Data)
	})
}