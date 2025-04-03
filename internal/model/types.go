package model

import "time"

// ComponentStatus represents the current status of a component
type ComponentStatus string

const (
	// StatusUninitialized indicates the component has not been initialized
	StatusUninitialized ComponentStatus = "UNINITIALIZED"
	// StatusInitialized indicates the component has been initialized but not started
	StatusInitialized ComponentStatus = "INITIALIZED"
	// StatusRunning indicates the component is currently running
	StatusRunning ComponentStatus = "RUNNING"
	// StatusStopped indicates the component has been stopped
	StatusStopped ComponentStatus = "STOPPED"
	// StatusError indicates the component is in an error state
	StatusError ComponentStatus = "ERROR"
)

// PluginType represents the type of plugin
type PluginType string

const (
	// InputPluginType represents plugins that collect data
	InputPluginType PluginType = "INPUT"
	// ProcessorPluginType represents plugins that transform data
	ProcessorPluginType PluginType = "PROCESSOR"
	// OutputPluginType represents plugins that export data
	OutputPluginType PluginType = "OUTPUT"
)

// TelemetryType represents the type of telemetry data
type TelemetryType string

const (
	// LogTelemetryType represents log data
	LogTelemetryType TelemetryType = "LOG"
	// MetricTelemetryType represents metric data
	MetricTelemetryType TelemetryType = "METRIC"
	// TraceTelemetryType represents trace data
	TraceTelemetryType TelemetryType = "TRACE"
)

// EventType represents the type of system event
type EventType string

const (
	// EventComponentStatusChange indicates a component status has changed
	EventComponentStatusChange EventType = "COMPONENT_STATUS_CHANGE"
	// EventConfigChange indicates a configuration has changed
	EventConfigChange EventType = "CONFIG_CHANGE"
	// EventDataReceived indicates data has been received
	EventDataReceived EventType = "DATA_RECEIVED"
	// EventDataProcessed indicates data has been processed
	EventDataProcessed EventType = "DATA_PROCESSED"
	// EventDataSent indicates data has been sent
	EventDataSent EventType = "DATA_SENT"
	// EventError indicates an error has occurred
	EventError EventType = "ERROR"
)

// HealthStatus represents the health status of the system or a component
type HealthStatus struct {
	Status    ComponentStatus     `json:"status"`
	Timestamp time.Time           `json:"timestamp"`
	Message   string              `json:"message,omitempty"`
	Details   map[string]any      `json:"details,omitempty"`
	Components map[string]HealthStatus `json:"components,omitempty"`
}

// BufferStatus represents the status of a buffer
type BufferStatus struct {
	BufferID   string    `json:"buffer_id"`
	QueueSize  int       `json:"queue_size"`
	TotalItems int       `json:"total_items"`
	IsFull     bool      `json:"is_full"`
	LastUpdate time.Time `json:"last_update"`
}