package core

import (
	"sync"

	"github.com/sliink/collector/internal/model"
)

// PluginRegistry keeps track of available plugins
type PluginRegistry struct {
	plugins map[string]model.Plugin
	mutex   sync.RWMutex
	BaseComponent
}

// NewPluginRegistry creates a new plugin registry
func NewPluginRegistry() *PluginRegistry {
	return &PluginRegistry{
		plugins:       make(map[string]model.Plugin),
		BaseComponent: NewBaseComponent("plugin_registry", "Plugin Registry"),
	}
}

// Initialize prepares the plugin registry for operation
func (r *PluginRegistry) Initialize() bool {
	r.SetStatus(model.StatusInitialized)
	return true
}

// Start begins plugin registry operation
func (r *PluginRegistry) Start() bool {
	r.SetStatus(model.StatusRunning)
	return true
}

// Stop halts plugin registry operation
func (r *PluginRegistry) Stop() bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Stop all plugins
	for _, p := range r.plugins {
		p.Stop()
	}
	
	r.SetStatus(model.StatusStopped)
	return true
}

// RegisterPlugin adds a plugin to the registry
func (r *PluginRegistry) RegisterPlugin(p model.Plugin) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.plugins[p.ID()]; exists {
		return false
	}

	r.plugins[p.ID()] = p
	return true
}

// UnregisterPlugin removes a plugin from the registry
func (r *PluginRegistry) UnregisterPlugin(pluginID string) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.plugins[pluginID]; !exists {
		return false
	}

	delete(r.plugins, pluginID)
	return true
}

// GetPlugin retrieves a plugin by ID
func (r *PluginRegistry) GetPlugin(pluginID string) (model.Plugin, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	p, exists := r.plugins[pluginID]
	return p, exists
}

// GetPluginsByType retrieves all plugins of a specific type
func (r *PluginRegistry) GetPluginsByType(pluginType model.PluginType) []model.Plugin {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var result []model.Plugin
	
	for _, p := range r.plugins {
		if p.GetType() == pluginType {
			result = append(result, p)
		}
	}
	
	return result
}

// GetInputPlugins retrieves all input plugins
func (r *PluginRegistry) GetInputPlugins() []model.InputPlugin {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var result []model.InputPlugin
	
	for _, p := range r.plugins {
		if p.GetType() == model.InputPluginType {
			if inputPlugin, ok := p.(model.InputPlugin); ok {
				result = append(result, inputPlugin)
			}
		}
	}
	
	return result
}

// GetProcessorPlugins retrieves all processor plugins
func (r *PluginRegistry) GetProcessorPlugins() []model.ProcessorPlugin {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var result []model.ProcessorPlugin
	
	for _, p := range r.plugins {
		if p.GetType() == model.ProcessorPluginType {
			if processorPlugin, ok := p.(model.ProcessorPlugin); ok {
				result = append(result, processorPlugin)
			}
		}
	}
	
	return result
}

// GetOutputPlugins retrieves all output plugins
func (r *PluginRegistry) GetOutputPlugins() []model.OutputPlugin {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var result []model.OutputPlugin
	
	for _, p := range r.plugins {
		if p.GetType() == model.OutputPluginType {
			if outputPlugin, ok := p.(model.OutputPlugin); ok {
				result = append(result, outputPlugin)
			}
		}
	}
	
	return result
}

// GetAllPlugins retrieves all registered plugins
func (r *PluginRegistry) GetAllPlugins() []model.Plugin {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var result []model.Plugin
	
	for _, p := range r.plugins {
		result = append(result, p)
	}
	
	return result
}