package plugin

import (
	"github.com/sliink/collector/internal/model"
)

// BasePlugin provides common functionality for all plugins
type BasePlugin struct {
	id         string
	name       string
	pluginType model.PluginType
	status     model.ComponentStatus
	Config     map[string]interface{}
	core       model.CoreAPI
}

// NewBasePlugin creates a new base plugin
func NewBasePlugin(id, name string, pluginType model.PluginType) BasePlugin {
	return BasePlugin{
		id:         id,
		name:       name,
		pluginType: pluginType,
		status:     model.StatusUninitialized,
		Config:     make(map[string]interface{}),
	}
}

// ID returns the plugin's unique identifier
func (p *BasePlugin) ID() string {
	return p.id
}

// Name returns the plugin's human-readable name
func (p *BasePlugin) Name() string {
	return p.name
}

// GetType returns the plugin type
func (p *BasePlugin) GetType() model.PluginType {
	return p.pluginType
}

// GetStatus returns the current plugin status
func (p *BasePlugin) GetStatus() model.ComponentStatus {
	return p.status
}

// SetStatus updates the plugin status
func (p *BasePlugin) SetStatus(status model.ComponentStatus) {
	p.status = status
}

// Configure applies configuration to the plugin
func (p *BasePlugin) Configure(config map[string]interface{}) bool {
	if config == nil {
		return false
	}
	p.Config = config
	return true
}

// RegisterWithCore registers the plugin with the core system
func (p *BasePlugin) RegisterWithCore(core model.CoreAPI) bool {
	p.core = core
	return true
}

// Validate checks if the plugin is properly configured
func (p *BasePlugin) Validate() bool {
	// Base implementation assumes valid, derived plugins should override
	return true
}