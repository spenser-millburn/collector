package plugin

import (
	"github.com/sliink/collector/internal/model"
)

// RegisterStandardPlugins registers all standard plugins with the factory
func RegisterStandardPlugins(factory *PluginFactory) {
	// Register standard plugins
	// Implementations will be imported by specific modules
}

// CreateStandardPlugins creates a set of standard plugins from configuration
func CreateStandardPlugins(config map[string]interface{}) ([]model.Plugin, error) {
	factory := NewPluginFactory()
	RegisterStandardPlugins(factory)

	var plugins []model.Plugin

	// Process input plugins
	if inputsConf, ok := config["inputs"].([]interface{}); ok {
		for _, inputConf := range inputsConf {
			if inputMap, ok := inputConf.(map[string]interface{}); ok {
				id, _ := inputMap["id"].(string)
				typeName, _ := inputMap["type"].(string)
				pluginConf, _ := inputMap["config"].(map[string]interface{})

				plugin, err := factory.CreatePlugin(model.InputPluginType, typeName, id)
				if err != nil {
					continue
				}

				plugin.Configure(pluginConf)
				plugins = append(plugins, plugin)
			}
		}
	}

	// Process processor plugins
	if processorsConf, ok := config["processors"].([]interface{}); ok {
		for _, processorConf := range processorsConf {
			if processorMap, ok := processorConf.(map[string]interface{}); ok {
				id, _ := processorMap["id"].(string)
				typeName, _ := processorMap["type"].(string)
				pluginConf, _ := processorMap["config"].(map[string]interface{})

				plugin, err := factory.CreatePlugin(model.ProcessorPluginType, typeName, id)
				if err != nil {
					continue
				}

				plugin.Configure(pluginConf)
				plugins = append(plugins, plugin)
			}
		}
	}

	// Process output plugins
	if outputsConf, ok := config["outputs"].([]interface{}); ok {
		for _, outputConf := range outputsConf {
			if outputMap, ok := outputConf.(map[string]interface{}); ok {
				id, _ := outputMap["id"].(string)
				typeName, _ := outputMap["type"].(string)
				pluginConf, _ := outputMap["config"].(map[string]interface{})

				plugin, err := factory.CreatePlugin(model.OutputPluginType, typeName, id)
				if err != nil {
					continue
				}

				plugin.Configure(pluginConf)
				plugins = append(plugins, plugin)
			}
		}
	}

	return plugins, nil
}