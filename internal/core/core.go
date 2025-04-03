package core

import (
	"context"
	"fmt"
	"time"

	"github.com/sliink/collector/internal/model"
)

// Core is the central coordinator of the system
type Core struct {
	eventBus       *EventBus
	registry       *PluginRegistry
	pipeline       *DataPipeline
	bufferManager  *BufferManager
	configManager  *ConfigManager
	healthMonitor  *HealthMonitor
	inputChannels  map[string]chan *model.DataBatch
	outputChannels map[string]chan *model.DataBatch
	ctx            context.Context
	cancel         context.CancelFunc
	BaseComponent
}

// GetComponent returns a component by ID
func (c *Core) GetComponent(id string) (Component, bool) {
	// Check if this is a core component first
	switch id {
	case "event_bus":
		return c.eventBus, true
	case "plugin_registry":
		return c.registry, true
	case "data_pipeline":
		return c.pipeline, true
	case "buffer_manager":
		return c.bufferManager, true
	case "config_manager":
		return c.configManager, true
	case "health_monitor":
		return c.healthMonitor, true
	case "core":
		return c, true
	}
	
	// Check if it's a plugin
	if c.registry != nil {
		plugin, exists := c.registry.GetPlugin(id)
		if exists {
			return plugin, true
		}
	}
	
	return nil, false
}

// GetDataPipeline returns the data pipeline component
func (c *Core) GetDataPipeline() *DataPipeline {
	return c.pipeline
}

// GetConfigManager returns the configuration manager component
func (c *Core) GetConfigManager() *ConfigManager {
	return c.configManager
}

// NewCore creates a new core system
func NewCore() *Core {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &Core{
		inputChannels:  make(map[string]chan *model.DataBatch),
		outputChannels: make(map[string]chan *model.DataBatch),
		ctx:            ctx,
		cancel:         cancel,
		BaseComponent:  NewBaseComponent("core", "Core System"),
	}
}

// Initialize prepares the core system for operation
func (c *Core) Initialize() bool {
	// Create core components
	c.eventBus = NewEventBus()
	c.registry = NewPluginRegistry()
	c.configManager = NewConfigManager()
	c.healthMonitor = NewHealthMonitor()
	c.bufferManager = NewBufferManager(1000) // Default buffer size
	
	// Initialize each component
	if !c.eventBus.Initialize() {
		return false
	}
	
	if !c.registry.Initialize() {
		return false
	}
	
	if !c.configManager.Initialize() {
		return false
	}
	
	if !c.healthMonitor.Initialize() {
		return false
	}
	
	if !c.bufferManager.Initialize() {
		return false
	}
	
	// Create pipeline after registry is initialized
	c.pipeline = NewDataPipeline(c.registry)
	if !c.pipeline.Initialize() {
		return false
	}
	
	// Register core components with health monitor
	c.healthMonitor.RegisterComponent(c)
	c.healthMonitor.RegisterComponent(c.eventBus)
	c.healthMonitor.RegisterComponent(c.registry)
	c.healthMonitor.RegisterComponent(c.configManager)
	c.healthMonitor.RegisterComponent(c.pipeline)
	c.healthMonitor.RegisterComponent(c.bufferManager)
	
	c.SetStatus(model.StatusInitialized)
	return true
}

// Start begins core system operation
func (c *Core) Start() bool {
	// Start each component
	if !c.eventBus.Start() {
		return false
	}
	
	if !c.registry.Start() {
		return false
	}
	
	if !c.configManager.Start() {
		return false
	}
	
	if !c.healthMonitor.Start() {
		return false
	}
	
	if !c.bufferManager.Start() {
		return false
	}
	
	if !c.pipeline.Start() {
		return false
	}
	
	// Start input plugins
	inputPlugins := c.registry.GetInputPlugins()
	for _, input := range inputPlugins {
		if err := c.startInputPlugin(input); err != nil {
			c.PublishEvent(model.EventError, c.ID(), err)
			return false
		}
	}
	
	// Start output plugins
	outputPlugins := c.registry.GetOutputPlugins()
	for _, output := range outputPlugins {
		if err := c.startOutputPlugin(output); err != nil {
			c.PublishEvent(model.EventError, c.ID(), err)
			return false
		}
	}
	
	c.SetStatus(model.StatusRunning)
	c.PublishEvent(model.EventComponentStatusChange, c.ID(), c.GetStatus())
	
	return true
}

// Stop halts core system operation
func (c *Core) Stop() bool {
	// Cancel all goroutines
	c.cancel()
	
	// Stop each component in reverse order
	c.pipeline.Stop()
	c.bufferManager.Stop()
	c.healthMonitor.Stop()
	c.configManager.Stop()
	c.registry.Stop()
	c.eventBus.Stop()
	
	// Close all channels
	for _, ch := range c.inputChannels {
		close(ch)
	}
	
	for _, ch := range c.outputChannels {
		close(ch)
	}
	
	c.SetStatus(model.StatusStopped)
	return true
}

// RegisterPlugin registers a plugin with the core system
func (c *Core) RegisterPlugin(p model.Plugin) error {
	if p == nil {
		return fmt.Errorf("cannot register nil plugin")
	}
	
	// Validate plugin
	if !p.Validate() {
		return fmt.Errorf("plugin validation failed: %s", p.ID())
	}
	
	// Register with core
	if !p.RegisterWithCore(c) {
		return fmt.Errorf("plugin failed to register with core: %s", p.ID())
	}
	
	// Register with registry
	if !c.registry.RegisterPlugin(p) {
		return fmt.Errorf("plugin registration failed: %s", p.ID())
	}
	
	// Register with health monitor
	c.healthMonitor.RegisterComponent(p)
	
	return nil
}

// startInputPlugin starts a goroutine for an input plugin
func (c *Core) startInputPlugin(input model.InputPlugin) error {
	if !input.Initialize() {
		return fmt.Errorf("failed to initialize input plugin: %s", input.ID())
	}
	
	// Create channel for this input
	c.inputChannels[input.ID()] = make(chan *model.DataBatch, 100)
	
	// Start the input goroutine
	go func(input model.InputPlugin, ch chan *model.DataBatch) {
		if !input.Start() {
			c.PublishEvent(model.EventError, input.ID(), fmt.Errorf("failed to start input plugin: %s", input.ID()))
			return
		}
		
		ticker := time.NewTicker(1 * time.Second) // Configurable interval
		defer ticker.Stop()
		
		for {
			select {
			case <-c.ctx.Done():
				input.Stop()
				return
			case <-ticker.C:
				// Collect data
				batches := input.Collect()
				
				// Process and buffer each batch
				for _, batch := range batches {
					if batch == nil || batch.Size() == 0 {
						continue
					}
					
					// Process the batch
					processed := c.ProcessBatch(batch)
					if processed == nil || processed.Size() == 0 {
						continue
					}
					
					// Send to channel for processing
					ch <- processed
				}
			}
		}
	}(input, c.inputChannels[input.ID()])
	
	// Start a goroutine to handle batches from this input
	go func(inputID string, ch chan *model.DataBatch) {
		for {
			select {
			case <-c.ctx.Done():
				return
			case batch := <-ch:
				if batch == nil {
					continue
				}
				
				// Get output plugins for this batch type
				outputs := c.getOutputsForBatchType(batch.BatchType)
				
				// Buffer for each output
				for _, output := range outputs {
					if !c.bufferManager.Buffer(output.ID(), batch) {
						c.PublishEvent(model.EventError, c.ID(), fmt.Errorf("buffer full for output: %s", output.ID()))
					}
				}
			}
		}
	}(input.ID(), c.inputChannels[input.ID()])
	
	return nil
}

// startOutputPlugin starts a goroutine for an output plugin
func (c *Core) startOutputPlugin(output model.OutputPlugin) error {
	if !output.Initialize() {
		return fmt.Errorf("failed to initialize output plugin: %s", output.ID())
	}
	
	// Create channel for this output
	c.outputChannels[output.ID()] = make(chan *model.DataBatch, 100)
	
	// Start the output goroutine
	go func(output model.OutputPlugin) {
		if !output.Start() {
			c.PublishEvent(model.EventError, output.ID(), fmt.Errorf("failed to start output plugin: %s", output.ID()))
			return
		}
		
		ticker := time.NewTicker(1 * time.Second) // Configurable interval
		defer ticker.Stop()
		
		for {
			select {
			case <-c.ctx.Done():
				output.Stop()
				return
			case <-ticker.C:
				// Flush batches from buffer
				batches := c.bufferManager.Flush(output.ID(), 10) // Configurable batch size
				
				// Send each batch
				for _, batch := range batches {
					if !output.Send(batch) {
						c.PublishEvent(model.EventError, output.ID(), fmt.Errorf("failed to send batch"))
					} else {
						c.PublishEvent(model.EventDataSent, output.ID(), map[string]interface{}{
							"batch_type": batch.BatchType,
							"batch_size": batch.Size(),
						})
					}
				}
			}
		}
	}(output)
	
	return nil
}

// getOutputsForBatchType returns all output plugins that should receive a batch type
func (c *Core) getOutputsForBatchType(batchType model.TelemetryType) []model.OutputPlugin {
	allOutputs := c.registry.GetOutputPlugins()
	
	// In a real implementation, check the configuration to determine which
	// outputs should receive which batch types. For now, return all outputs.
	return allOutputs
}

// PublishEvent publishes an event to the event bus
func (c *Core) PublishEvent(eventType model.EventType, sourceID string, data interface{}) {
	if c.eventBus == nil {
		return
	}
	
	event := NewEvent(eventType, sourceID, data)
	c.eventBus.Publish(event)
}

// ProcessBatch processes a data batch through the pipeline
func (c *Core) ProcessBatch(batch *model.DataBatch) *model.DataBatch {
	if batch == nil || batch.Size() == 0 || c.pipeline == nil {
		return batch
	}
	
	c.PublishEvent(model.EventDataReceived, c.ID(), map[string]interface{}{
		"batch_type": batch.BatchType,
		"batch_size": batch.Size(),
	})
	
	processed := c.pipeline.Process(batch)
	
	if processed != nil && processed.Size() > 0 {
		c.PublishEvent(model.EventDataProcessed, c.ID(), map[string]interface{}{
			"batch_type": processed.BatchType,
			"batch_size": processed.Size(),
		})
	}
	
	return processed
}