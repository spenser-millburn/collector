package core

import (
	"testing"

	"github.com/sliink/collector/internal/model"
	"github.com/stretchr/testify/assert"
)

// mockInputPlugin implements the InputPlugin interface for testing
type mockInputPlugin struct {
	id     string
	name   string
	status model.ComponentStatus
}

func (m *mockInputPlugin) ID() string {
	return m.id
}

func (m *mockInputPlugin) Name() string {
	return m.name
}

func (m *mockInputPlugin) GetType() model.PluginType {
	return model.InputPluginType
}

func (m *mockInputPlugin) GetStatus() model.ComponentStatus {
	return m.status
}

func (m *mockInputPlugin) SetStatus(status model.ComponentStatus) {
	m.status = status
}

func (m *mockInputPlugin) Configure(config map[string]interface{}) bool {
	return true
}

func (m *mockInputPlugin) Initialize() bool {
	m.status = model.StatusInitialized
	return true
}

func (m *mockInputPlugin) Start() bool {
	m.status = model.StatusRunning
	return true
}

func (m *mockInputPlugin) Stop() bool {
	m.status = model.StatusStopped
	return true
}

func (m *mockInputPlugin) Validate() bool {
	return true
}

func (m *mockInputPlugin) RegisterWithCore(core model.CoreAPI) bool {
	return true
}

func (m *mockInputPlugin) Collect() []*model.DataBatch {
	return nil
}

// mockOutputPlugin implements the OutputPlugin interface for testing
type mockOutputPlugin struct {
	id     string
	name   string
	status model.ComponentStatus
}

func (m *mockOutputPlugin) ID() string {
	return m.id
}

func (m *mockOutputPlugin) Name() string {
	return m.name
}

func (m *mockOutputPlugin) GetType() model.PluginType {
	return model.OutputPluginType
}

func (m *mockOutputPlugin) GetStatus() model.ComponentStatus {
	return m.status
}

func (m *mockOutputPlugin) SetStatus(status model.ComponentStatus) {
	m.status = status
}

func (m *mockOutputPlugin) Configure(config map[string]interface{}) bool {
	return true
}

func (m *mockOutputPlugin) Initialize() bool {
	m.status = model.StatusInitialized
	return true
}

func (m *mockOutputPlugin) Start() bool {
	m.status = model.StatusRunning
	return true
}

func (m *mockOutputPlugin) Stop() bool {
	m.status = model.StatusStopped
	return true
}

func (m *mockOutputPlugin) Validate() bool {
	return true
}

func (m *mockOutputPlugin) RegisterWithCore(core model.CoreAPI) bool {
	return true
}

func (m *mockOutputPlugin) Send(batch *model.DataBatch) bool {
	return true
}

func TestNewPluginRegistry(t *testing.T) {
	registry := NewPluginRegistry()
	
	assert.NotNil(t, registry)
	assert.NotNil(t, registry.plugins)
	assert.Equal(t, "plugin_registry", registry.ID())
	assert.Equal(t, "Plugin Registry", registry.Name())
}

func TestPluginRegistryLifecycle(t *testing.T) {
	registry := NewPluginRegistry()
	
	t.Run("Initialize sets correct status", func(t *testing.T) {
		success := registry.Initialize()
		assert.True(t, success)
		assert.Equal(t, model.StatusInitialized, registry.GetStatus())
	})
	
	t.Run("Start sets correct status", func(t *testing.T) {
		success := registry.Start()
		assert.True(t, success)
		assert.Equal(t, model.StatusRunning, registry.GetStatus())
	})
	
	t.Run("Stop calls stop on all plugins", func(t *testing.T) {
		// Add some plugins
		input := &mockInputPlugin{
			id:     "input",
			name:   "Input Plugin",
			status: model.StatusRunning,
		}
		
		output := &mockOutputPlugin{
			id:     "output",
			name:   "Output Plugin",
			status: model.StatusRunning,
		}
		
		registry.plugins["input"] = input
		registry.plugins["output"] = output
		
		// Stop the registry
		success := registry.Stop()
		assert.True(t, success)
		assert.Equal(t, model.StatusStopped, registry.GetStatus())
		
		// Verify all plugins were stopped
		assert.Equal(t, model.StatusStopped, input.status)
		assert.Equal(t, model.StatusStopped, output.status)
	})
}

func TestRegisterPlugin(t *testing.T) {
	registry := NewPluginRegistry()
	registry.Initialize()
	
	input := &mockInputPlugin{
		id:     "input1",
		name:   "Input Plugin 1",
		status: model.StatusUninitialized,
	}
	
	output := &mockOutputPlugin{
		id:     "output1",
		name:   "Output Plugin 1",
		status: model.StatusUninitialized,
	}
	
	processor := newMockProcessorPlugin("processor1", "Processor 1", nil)
	
	t.Run("RegisterPlugin adds plugin to registry", func(t *testing.T) {
		success := registry.RegisterPlugin(input)
		assert.True(t, success)
		assert.Len(t, registry.plugins, 1)
		assert.Contains(t, registry.plugins, "input1")
		
		success = registry.RegisterPlugin(output)
		assert.True(t, success)
		assert.Len(t, registry.plugins, 2)
		assert.Contains(t, registry.plugins, "output1")
		
		success = registry.RegisterPlugin(processor)
		assert.True(t, success)
		assert.Len(t, registry.plugins, 3)
		assert.Contains(t, registry.plugins, "processor1")
	})
	
	t.Run("RegisterPlugin fails for duplicate ID", func(t *testing.T) {
		duplicateInput := &mockInputPlugin{
			id:   "input1",
			name: "Duplicate Input Plugin",
		}
		
		success := registry.RegisterPlugin(duplicateInput)
		assert.False(t, success)
		
		// Verify original wasn't replaced
		plugin, exists := registry.GetPlugin("input1")
		assert.True(t, exists)
		assert.Equal(t, "Input Plugin 1", plugin.Name())
	})
}

func TestUnregisterPlugin(t *testing.T) {
	registry := NewPluginRegistry()
	registry.Initialize()
	
	input := &mockInputPlugin{
		id:     "input1",
		name:   "Input Plugin 1",
		status: model.StatusUninitialized,
	}
	
	output := &mockOutputPlugin{
		id:     "output1",
		name:   "Output Plugin 1",
		status: model.StatusUninitialized,
	}
	
	registry.RegisterPlugin(input)
	registry.RegisterPlugin(output)
	
	t.Run("UnregisterPlugin removes existing plugin", func(t *testing.T) {
		success := registry.UnregisterPlugin("input1")
		assert.True(t, success)
		assert.Len(t, registry.plugins, 1)
		assert.NotContains(t, registry.plugins, "input1")
	})
	
	t.Run("UnregisterPlugin fails for nonexistent plugin", func(t *testing.T) {
		success := registry.UnregisterPlugin("nonexistent")
		assert.False(t, success)
	})
}

func TestGetPlugin(t *testing.T) {
	registry := NewPluginRegistry()
	registry.Initialize()
	
	input := &mockInputPlugin{
		id:     "input1",
		name:   "Input Plugin 1",
		status: model.StatusUninitialized,
	}
	
	registry.RegisterPlugin(input)
	
	t.Run("GetPlugin returns existing plugin", func(t *testing.T) {
		plugin, exists := registry.GetPlugin("input1")
		assert.True(t, exists)
		assert.Equal(t, input, plugin)
	})
	
	t.Run("GetPlugin returns false for nonexistent plugin", func(t *testing.T) {
		_, exists := registry.GetPlugin("nonexistent")
		assert.False(t, exists)
	})
}

func TestGetPluginsByType(t *testing.T) {
	registry := NewPluginRegistry()
	registry.Initialize()
	
	input1 := &mockInputPlugin{id: "input1", name: "Input 1"}
	input2 := &mockInputPlugin{id: "input2", name: "Input 2"}
	output1 := &mockOutputPlugin{id: "output1", name: "Output 1"}
	processor1 := newMockProcessorPlugin("processor1", "Processor 1", nil)
	
	registry.RegisterPlugin(input1)
	registry.RegisterPlugin(input2)
	registry.RegisterPlugin(output1)
	registry.RegisterPlugin(processor1)
	
	t.Run("GetPluginsByType returns all plugins of specified type", func(t *testing.T) {
		inputs := registry.GetPluginsByType(model.InputPluginType)
		assert.Len(t, inputs, 2)
		
		outputs := registry.GetPluginsByType(model.OutputPluginType)
		assert.Len(t, outputs, 1)
		
		processors := registry.GetPluginsByType(model.ProcessorPluginType)
		assert.Len(t, processors, 1)
	})
	
	t.Run("GetPluginsByType returns empty slice for nonexistent type", func(t *testing.T) {
		plugins := registry.GetPluginsByType("nonexistent")
		assert.Empty(t, plugins)
	})
}

func TestGetTypedPlugins(t *testing.T) {
	registry := NewPluginRegistry()
	registry.Initialize()
	
	input1 := &mockInputPlugin{id: "input1", name: "Input 1"}
	input2 := &mockInputPlugin{id: "input2", name: "Input 2"}
	output1 := &mockOutputPlugin{id: "output1", name: "Output 1"}
	processor1 := newMockProcessorPlugin("processor1", "Processor 1", nil)
	
	registry.RegisterPlugin(input1)
	registry.RegisterPlugin(input2)
	registry.RegisterPlugin(output1)
	registry.RegisterPlugin(processor1)
	
	t.Run("GetInputPlugins returns all input plugins", func(t *testing.T) {
		inputs := registry.GetInputPlugins()
		assert.Len(t, inputs, 2)
		
		// Verify correct type
		for _, plugin := range inputs {
			_, ok := plugin.(*mockInputPlugin)
			assert.True(t, ok)
		}
	})
	
	t.Run("GetOutputPlugins returns all output plugins", func(t *testing.T) {
		outputs := registry.GetOutputPlugins()
		assert.Len(t, outputs, 1)
		
		// Verify correct type
		for _, plugin := range outputs {
			_, ok := plugin.(*mockOutputPlugin)
			assert.True(t, ok)
		}
	})
	
	t.Run("GetProcessorPlugins returns all processor plugins", func(t *testing.T) {
		processors := registry.GetProcessorPlugins()
		assert.Len(t, processors, 1)
		
		// Verify correct type
		for _, plugin := range processors {
			_, ok := plugin.(*mockProcessorPlugin)
			assert.True(t, ok)
		}
	})
}