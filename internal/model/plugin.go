package model

// CoreAPI is an interface for core functions needed by plugins
type CoreAPI interface {
	// ProcessBatch processes a data batch through the pipeline
	ProcessBatch(batch *DataBatch) *DataBatch
	
	// PublishEvent publishes an event to the event bus
	PublishEvent(eventType EventType, sourceID string, data interface{})
}

// Plugin is the base interface for all plugins
type Plugin interface {
	// Initialize prepares the plugin for operation
	Initialize() bool
	
	// Start begins plugin operation
	Start() bool
	
	// Stop halts plugin operation
	Stop() bool
	
	// GetStatus returns the current plugin status
	GetStatus() ComponentStatus
	
	// SetStatus updates the plugin status
	SetStatus(status ComponentStatus)
	
	// Configure applies configuration to the plugin
	Configure(config map[string]interface{}) bool
	
	// ID returns the plugin's unique identifier
	ID() string
	
	// Name returns the plugin's human-readable name
	Name() string
	
	// GetType returns the plugin type
	GetType() PluginType
	
	// Validate checks if the plugin is properly configured
	Validate() bool
	
	// RegisterWithCore registers the plugin with the core system
	RegisterWithCore(core CoreAPI) bool
}

// InputPlugin collects data from sources
type InputPlugin interface {
	Plugin
	
	// Collect gathers data from the source
	Collect() []*DataBatch
}

// ProcessorPlugin transforms data
type ProcessorPlugin interface {
	Plugin
	
	// Process transforms a data batch
	Process(batch *DataBatch) *DataBatch
}

// OutputPlugin exports data to destinations
type OutputPlugin interface {
	Plugin
	
	// Send exports a data batch
	Send(batch *DataBatch) bool
}