package core

import (
	"github.com/sliink/collector/internal/model"
)

// Component represents a core system component with lifecycle management
type Component interface {
	// Initialize prepares the component for operation
	Initialize() bool
	
	// Start begins component operation
	Start() bool
	
	// Stop halts component operation
	Stop() bool
	
	// GetStatus returns the current component status
	GetStatus() model.ComponentStatus
	
	// SetStatus updates the component status
	SetStatus(status model.ComponentStatus)
	
	// Configure applies configuration to the component
	Configure(config map[string]interface{}) bool

	// ID returns the component's unique identifier
	ID() string

	// Name returns the component's human-readable name
	Name() string
}

// BaseComponent provides common functionality for all components
type BaseComponent struct {
	id     string
	name   string
	status model.ComponentStatus
	config map[string]interface{}
}

// NewBaseComponent creates a new base component
func NewBaseComponent(id, name string) BaseComponent {
	return BaseComponent{
		id:     id,
		name:   name,
		status: model.StatusUninitialized,
		config: make(map[string]interface{}),
	}
}

// ID returns the component's unique identifier
func (c *BaseComponent) ID() string {
	return c.id
}

// Name returns the component's human-readable name
func (c *BaseComponent) Name() string {
	return c.name
}

// GetStatus returns the current component status
func (c *BaseComponent) GetStatus() model.ComponentStatus {
	return c.status
}

// SetStatus updates the component status
func (c *BaseComponent) SetStatus(status model.ComponentStatus) {
	c.status = status
}

// Configure applies configuration to the component
func (c *BaseComponent) Configure(config map[string]interface{}) bool {
	if config == nil {
		return false
	}
	c.config = config
	return true
}