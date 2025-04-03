package plugin

import (
	"fmt"

	"github.com/sliink/collector/internal/model"
)

// PluginFactory creates plugins based on their type and name
type PluginFactory struct {
	inputCreators    map[string]func(id string) model.InputPlugin
	processorCreators map[string]func(id string) model.ProcessorPlugin
	outputCreators   map[string]func(id string) model.OutputPlugin
}

// NewPluginFactory creates a new plugin factory
func NewPluginFactory() *PluginFactory {
	return &PluginFactory{
		inputCreators:    make(map[string]func(id string) model.InputPlugin),
		processorCreators: make(map[string]func(id string) model.ProcessorPlugin),
		outputCreators:   make(map[string]func(id string) model.OutputPlugin),
	}
}

// RegisterInputPlugin registers an input plugin creator
func (f *PluginFactory) RegisterInputPlugin(name string, creator func(id string) model.InputPlugin) {
	f.inputCreators[name] = creator
}

// RegisterProcessorPlugin registers a processor plugin creator
func (f *PluginFactory) RegisterProcessorPlugin(name string, creator func(id string) model.ProcessorPlugin) {
	f.processorCreators[name] = creator
}

// RegisterOutputPlugin registers an output plugin creator
func (f *PluginFactory) RegisterOutputPlugin(name string, creator func(id string) model.OutputPlugin) {
	f.outputCreators[name] = creator
}

// CreatePlugin creates a plugin based on its type and name
func (f *PluginFactory) CreatePlugin(pluginType model.PluginType, name, id string) (model.Plugin, error) {
	switch pluginType {
	case model.InputPluginType:
		creator, exists := f.inputCreators[name]
		if !exists {
			return nil, fmt.Errorf("unknown input plugin: %s", name)
		}
		return creator(id), nil
		
	case model.ProcessorPluginType:
		creator, exists := f.processorCreators[name]
		if !exists {
			return nil, fmt.Errorf("unknown processor plugin: %s", name)
		}
		return creator(id), nil
		
	case model.OutputPluginType:
		creator, exists := f.outputCreators[name]
		if !exists {
			return nil, fmt.Errorf("unknown output plugin: %s", name)
		}
		return creator(id), nil
		
	default:
		return nil, fmt.Errorf("unknown plugin type: %s", pluginType)
	}
}