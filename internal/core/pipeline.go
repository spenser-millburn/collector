package core

import (
	"errors"
	"sync"

	"github.com/sliink/collector/internal/model"
)

// PipelineStage represents a single processing step
type PipelineStage struct {
	Processor model.ProcessorPlugin
	NextStage *PipelineStage
}

// Process executes the processing stage on a data batch
func (s *PipelineStage) Process(batch *model.DataBatch) *model.DataBatch {
	if s == nil || batch == nil {
		return batch
	}

	// Process the batch
	processed := s.Processor.Process(batch)
	
	// If the processor returns nil or the batch has no points, stop the pipeline
	if processed == nil || processed.Size() == 0 {
		return nil
	}

	// Pass to next stage if any
	if s.NextStage != nil {
		return s.NextStage.Process(processed)
	}
	
	return processed
}

// DataPipeline manages the processing pipeline
type DataPipeline struct {
	pipelines map[model.TelemetryType]*PipelineStage
	registry  *PluginRegistry
	mutex     sync.RWMutex
	BaseComponent
}

// NewDataPipeline creates a new data pipeline
func NewDataPipeline(registry *PluginRegistry) *DataPipeline {
	return &DataPipeline{
		pipelines:     make(map[model.TelemetryType]*PipelineStage),
		registry:      registry,
		BaseComponent: NewBaseComponent("data_pipeline", "Data Pipeline"),
	}
}

// Initialize prepares the data pipeline for operation
func (p *DataPipeline) Initialize() bool {
	if p.registry == nil {
		return false
	}
	
	p.SetStatus(model.StatusInitialized)
	return true
}

// Start begins data pipeline operation
func (p *DataPipeline) Start() bool {
	p.SetStatus(model.StatusRunning)
	return true
}

// Stop halts data pipeline operation
func (p *DataPipeline) Stop() bool {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Clear all pipelines
	p.pipelines = make(map[model.TelemetryType]*PipelineStage)
	
	p.SetStatus(model.StatusStopped)
	return true
}

// CreatePipeline builds a processing pipeline for a telemetry type
func (p *DataPipeline) CreatePipeline(telemetryType model.TelemetryType, processorIDs []string) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if len(processorIDs) == 0 {
		return errors.New("no processors specified for pipeline")
	}

	var firstStage *PipelineStage
	var currentStage *PipelineStage

	// Create processing stages
	for _, processorID := range processorIDs {
		plugin, exists := p.registry.GetPlugin(processorID)
		if !exists {
			return errors.New("processor plugin not found: " + processorID)
		}

		processor, ok := plugin.(model.ProcessorPlugin)
		if !ok {
			return errors.New("plugin is not a processor: " + processorID)
		}

		stage := &PipelineStage{
			Processor: processor,
		}

		if firstStage == nil {
			firstStage = stage
			currentStage = stage
		} else {
			currentStage.NextStage = stage
			currentStage = stage
		}
	}

	p.pipelines[telemetryType] = firstStage
	return nil
}

// Process sends a data batch through the pipeline
func (p *DataPipeline) Process(batch *model.DataBatch) *model.DataBatch {
	if batch == nil || batch.Size() == 0 {
		return nil
	}

	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if p.GetStatus() != model.StatusRunning {
		return nil
	}

	// Find pipeline for batch type
	pipeline, exists := p.pipelines[batch.BatchType]
	if !exists || pipeline == nil {
		// No processing needed, return original batch
		return batch
	}

	// Process the batch through the pipeline
	return pipeline.Process(batch)
}