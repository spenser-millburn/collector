package plugin

import (
	"testing"

	"github.com/sliink/collector/internal/model"
	"github.com/stretchr/testify/assert"
)

// mockCoreAPI implements model.CoreAPI for testing
type mockCoreAPI struct {
	processBatchCalled bool
	publishEventCalled bool
	lastEventType      model.EventType
	lastSourceID       string
	lastData           interface{}
}

func (m *mockCoreAPI) ProcessBatch(batch *model.DataBatch) *model.DataBatch {
	m.processBatchCalled = true
	return batch
}

func (m *mockCoreAPI) PublishEvent(eventType model.EventType, sourceID string, data interface{}) {
	m.publishEventCalled = true
	m.lastEventType = eventType
	m.lastSourceID = sourceID
	m.lastData = data
}

func TestNewBasePlugin(t *testing.T) {
	t.Run("Creates plugin with correct properties", func(t *testing.T) {
		plugin := NewBasePlugin("test_id", "Test Plugin", model.InputPluginType)
		
		assert.Equal(t, "test_id", plugin.id)
		assert.Equal(t, "Test Plugin", plugin.name)
		assert.Equal(t, model.InputPluginType, plugin.pluginType)
		assert.Equal(t, model.StatusUninitialized, plugin.status)
		assert.NotNil(t, plugin.Config)
	})
}

func TestBasePluginAccessors(t *testing.T) {
	plugin := NewBasePlugin("test_id", "Test Plugin", model.ProcessorPluginType)
	
	t.Run("ID returns correct identifier", func(t *testing.T) {
		assert.Equal(t, "test_id", plugin.ID())
	})
	
	t.Run("Name returns correct name", func(t *testing.T) {
		assert.Equal(t, "Test Plugin", plugin.Name())
	})
	
	t.Run("GetType returns correct plugin type", func(t *testing.T) {
		assert.Equal(t, model.ProcessorPluginType, plugin.GetType())
	})
	
	t.Run("GetStatus returns current status", func(t *testing.T) {
		assert.Equal(t, model.StatusUninitialized, plugin.GetStatus())
	})
	
	t.Run("SetStatus updates status", func(t *testing.T) {
		plugin.SetStatus(model.StatusRunning)
		assert.Equal(t, model.StatusRunning, plugin.GetStatus())
	})
}

func TestBasePluginConfigure(t *testing.T) {
	plugin := NewBasePlugin("test_id", "Test Plugin", model.OutputPluginType)
	
	t.Run("Configure with nil config returns false", func(t *testing.T) {
		result := plugin.Configure(nil)
		assert.False(t, result)
	})
	
	t.Run("Configure with valid config returns true", func(t *testing.T) {
		config := map[string]interface{}{
			"test_key": "test_value",
			"nested": map[string]interface{}{
				"key": 42,
			},
		}
		
		result := plugin.Configure(config)
		assert.True(t, result)
		assert.Equal(t, config, plugin.Config)
	})
}

func TestBasePluginRegisterWithCore(t *testing.T) {
	plugin := NewBasePlugin("test_id", "Test Plugin", model.InputPluginType)
	core := &mockCoreAPI{}
	
	t.Run("RegisterWithCore sets core reference", func(t *testing.T) {
		result := plugin.RegisterWithCore(core)
		assert.True(t, result)
		assert.Equal(t, core, plugin.core)
	})
}

func TestBasePluginValidate(t *testing.T) {
	plugin := NewBasePlugin("test_id", "Test Plugin", model.InputPluginType)
	
	t.Run("Validate returns true by default", func(t *testing.T) {
		result := plugin.Validate()
		assert.True(t, result)
	})
}

// Test for FileInput plugin
type TestBasePlugin struct {
	BasePlugin
}

func TestBasePluginExtension(t *testing.T) {
	t.Run("Can be embedded in derived plugin", func(t *testing.T) {
		basePlugin := NewBasePlugin("derived", "Derived Plugin", model.InputPluginType)
		derived := TestBasePlugin{
			BasePlugin: basePlugin,
		}
		
		assert.Equal(t, "derived", derived.ID())
		assert.Equal(t, "Derived Plugin", derived.Name())
		assert.Equal(t, model.InputPluginType, derived.GetType())
	})
}