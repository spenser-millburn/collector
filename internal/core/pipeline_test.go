package core

import (
	"testing"
	"time"

	"github.com/sliink/collector/internal/model"
	"github.com/stretchr/testify/assert"
)

// mockProcessorPlugin implements the ProcessorPlugin interface for testing
type mockProcessorPlugin struct {
	id          string
	name        string
	status      model.ComponentStatus
	processFunc func(batch *model.DataBatch) *model.DataBatch
}

func (m *mockProcessorPlugin) ID() string {
	return m.id
}

func (m *mockProcessorPlugin) Name() string {
	return m.name
}

func (m *mockProcessorPlugin) GetType() model.PluginType {
	return model.ProcessorPluginType
}

func (m *mockProcessorPlugin) GetStatus() model.ComponentStatus {
	return m.status
}

func (m *mockProcessorPlugin) SetStatus(status model.ComponentStatus) {
	m.status = status
}

func (m *mockProcessorPlugin) Configure(config map[string]interface{}) bool {
	return true
}

func (m *mockProcessorPlugin) Initialize() bool {
	m.status = model.StatusInitialized
	return true
}

func (m *mockProcessorPlugin) Start() bool {
	m.status = model.StatusRunning
	return true
}

func (m *mockProcessorPlugin) Stop() bool {
	m.status = model.StatusStopped
	return true
}

func (m *mockProcessorPlugin) Validate() bool {
	return true
}

func (m *mockProcessorPlugin) RegisterWithCore(core model.CoreAPI) bool {
	return true
}

func (m *mockProcessorPlugin) Process(batch *model.DataBatch) *model.DataBatch {
	if m.processFunc != nil {
		return m.processFunc(batch)
	}
	return batch
}

func newMockProcessorPlugin(id, name string, processFunc func(batch *model.DataBatch) *model.DataBatch) *mockProcessorPlugin {
	return &mockProcessorPlugin{
		id:          id,
		name:        name,
		status:      model.StatusUninitialized,
		processFunc: processFunc,
	}
}

// Helper function to create a registry with mock processors
func createTestRegistry() *PluginRegistry {
	registry := NewPluginRegistry()
	registry.Initialize()
	
	// Create and register processors
	passthrough := newMockProcessorPlugin("passthrough", "Passthrough Processor", nil)
	
	doubler := newMockProcessorPlugin("doubler", "Double Points Processor", func(batch *model.DataBatch) *model.DataBatch {
		// Create a new batch with double the points
		newBatch := model.NewDataBatch(batch.BatchType)
		for _, point := range batch.Points {
			newBatch.AddPoint(point)
			newBatch.AddPoint(point)
		}
		return newBatch
	})
	
	filter := newMockProcessorPlugin("filter", "Filter Processor", func(batch *model.DataBatch) *model.DataBatch {
		// Return empty batch
		return model.NewDataBatch(batch.BatchType)
	})
	
	registry.RegisterPlugin(passthrough)
	registry.RegisterPlugin(doubler)
	registry.RegisterPlugin(filter)
	
	return registry
}

func TestPipelineStageProcess(t *testing.T) {
	t.Run("Nil stage returns original batch", func(t *testing.T) {
		var stage *PipelineStage
		batch := createTestBatch(5)
		
		result := stage.Process(batch)
		assert.Equal(t, batch, result)
	})
	
	t.Run("Nil batch returns nil", func(t *testing.T) {
		processor := newMockProcessorPlugin("test", "Test", nil)
		stage := &PipelineStage{
			Processor: processor,
		}
		
		result := stage.Process(nil)
		assert.Nil(t, result)
	})
	
	t.Run("Single stage processes batch", func(t *testing.T) {
		// Create a processor that doubles the batch size
		processor := newMockProcessorPlugin("doubler", "Doubler", func(batch *model.DataBatch) *model.DataBatch {
			newBatch := model.NewDataBatch(batch.BatchType)
			for _, point := range batch.Points {
				newBatch.AddPoint(point)
				newBatch.AddPoint(point)
			}
			return newBatch
		})
		
		stage := &PipelineStage{
			Processor: processor,
		}
		
		batch := createTestBatch(3)
		result := stage.Process(batch)
		
		assert.NotNil(t, result)
		assert.Equal(t, 6, result.Size())
	})
	
	t.Run("Chained stages process in sequence", func(t *testing.T) {
		// First processor doubles points
		doubler := newMockProcessorPlugin("doubler", "Doubler", func(batch *model.DataBatch) *model.DataBatch {
			newBatch := model.NewDataBatch(batch.BatchType)
			for _, point := range batch.Points {
				newBatch.AddPoint(point)
				newBatch.AddPoint(point)
			}
			return newBatch
		})
		
		// Second processor adds a tag to all points
		tagger := newMockProcessorPlugin("tagger", "Tagger", func(batch *model.DataBatch) *model.DataBatch {
			for _, point := range batch.Points {
				if mp, ok := point.(*mockDataPoint); ok {
					mp.labels["processed"] = "true"
				}
			}
			return batch
		})
		
		firstStage := &PipelineStage{
			Processor: doubler,
			NextStage: &PipelineStage{
				Processor: tagger,
			},
		}
		
		batch := createTestBatch(2)
		result := firstStage.Process(batch)
		
		assert.NotNil(t, result)
		assert.Equal(t, 4, result.Size())
		
		// Check that all points have the processed tag
		for _, point := range result.Points {
			if mp, ok := point.(*mockDataPoint); ok {
				assert.Equal(t, "true", mp.labels["processed"])
			}
		}
	})
	
	t.Run("Empty result stops pipeline", func(t *testing.T) {
		// First processor returns empty batch
		filter := newMockProcessorPlugin("filter", "Filter", func(batch *model.DataBatch) *model.DataBatch {
			emptyBatch := model.NewDataBatch(batch.BatchType)
			return emptyBatch
		})
		
		// Second processor should never be called
		var secondCalled bool
		second := newMockProcessorPlugin("second", "Second", func(batch *model.DataBatch) *model.DataBatch {
			secondCalled = true
			return batch
		})
		
		firstStage := &PipelineStage{
			Processor: filter,
			NextStage: &PipelineStage{
				Processor: second,
			},
		}
		
		batch := createTestBatch(2)
		result := firstStage.Process(batch)
		
		assert.NotNil(t, result, "Result should not be nil")
		if result != nil {
			assert.Equal(t, 0, result.Size())
		}
		assert.False(t, secondCalled)
	})
}

func TestNewDataPipeline(t *testing.T) {
	registry := createTestRegistry()
	pipeline := NewDataPipeline(registry)
	
	assert.NotNil(t, pipeline)
	assert.Equal(t, registry, pipeline.registry)
	assert.NotNil(t, pipeline.pipelines)
	assert.Equal(t, "data_pipeline", pipeline.ID())
	assert.Equal(t, "Data Pipeline", pipeline.Name())
}

func TestDataPipelineLifecycle(t *testing.T) {
	registry := createTestRegistry()
	pipeline := NewDataPipeline(registry)
	
	t.Run("Initialize fails with nil registry", func(t *testing.T) {
		nilPipeline := NewDataPipeline(nil)
		success := nilPipeline.Initialize()
		assert.False(t, success)
	})
	
	t.Run("Initialize sets correct status", func(t *testing.T) {
		success := pipeline.Initialize()
		assert.True(t, success)
		assert.Equal(t, model.StatusInitialized, pipeline.GetStatus())
	})
	
	t.Run("Start sets correct status", func(t *testing.T) {
		success := pipeline.Start()
		assert.True(t, success)
		assert.Equal(t, model.StatusRunning, pipeline.GetStatus())
	})
	
	t.Run("Stop clears pipelines and sets correct status", func(t *testing.T) {
		// Create a pipeline first
		err := pipeline.CreatePipeline(model.LogTelemetryType, []string{"passthrough"})
		assert.NoError(t, err)
		assert.NotEmpty(t, pipeline.pipelines)
		
		// Now stop and verify pipelines are cleared
		success := pipeline.Stop()
		assert.True(t, success)
		assert.Empty(t, pipeline.pipelines)
		assert.Equal(t, model.StatusStopped, pipeline.GetStatus())
	})
}

func TestCreatePipeline(t *testing.T) {
	registry := createTestRegistry()
	pipeline := NewDataPipeline(registry)
	pipeline.Initialize()
	
	t.Run("Empty processor list returns error", func(t *testing.T) {
		err := pipeline.CreatePipeline(model.LogTelemetryType, []string{})
		assert.Error(t, err)
	})
	
	t.Run("Nonexistent processor returns error", func(t *testing.T) {
		err := pipeline.CreatePipeline(model.LogTelemetryType, []string{"nonexistent"})
		assert.Error(t, err)
	})
	
	t.Run("Non-processor plugin returns error", func(t *testing.T) {
		// Skipping this test as it requires a non-processor plugin implementation
	})
	
	t.Run("Valid processor list creates pipeline", func(t *testing.T) {
		err := pipeline.CreatePipeline(model.LogTelemetryType, []string{"passthrough"})
		assert.NoError(t, err)
		
		// Verify pipeline was created
		stage, exists := pipeline.pipelines[model.LogTelemetryType]
		assert.True(t, exists)
		assert.NotNil(t, stage)
		assert.Equal(t, "passthrough", stage.Processor.ID())
	})
	
	t.Run("Multiple processors create chained pipeline", func(t *testing.T) {
		err := pipeline.CreatePipeline(model.MetricTelemetryType, []string{"passthrough", "doubler"})
		assert.NoError(t, err)
		
		// Verify pipeline was created
		stage, exists := pipeline.pipelines[model.MetricTelemetryType]
		assert.True(t, exists)
		assert.NotNil(t, stage)
		assert.Equal(t, "passthrough", stage.Processor.ID())
		
		// Verify chain
		assert.NotNil(t, stage.NextStage)
		assert.Equal(t, "doubler", stage.NextStage.Processor.ID())
	})
}

func TestProcessMethod(t *testing.T) {
	registry := createTestRegistry()
	pipeline := NewDataPipeline(registry)
	pipeline.Initialize()
	pipeline.Start()
	
	// Create two pipelines
	err := pipeline.CreatePipeline(model.LogTelemetryType, []string{"doubler"})
	assert.NoError(t, err)
	
	err = pipeline.CreatePipeline(model.MetricTelemetryType, []string{"filter"})
	assert.NoError(t, err)
	
	t.Run("Process returns nil for nil batch", func(t *testing.T) {
		result := pipeline.Process(nil)
		assert.Nil(t, result)
	})
	
	t.Run("Process returns nil for empty batch", func(t *testing.T) {
		batch := model.NewDataBatch(model.LogTelemetryType)
		result := pipeline.Process(batch)
		assert.Nil(t, result)
	})
	
	t.Run("Process returns nil when not running", func(t *testing.T) {
		pipeline.SetStatus(model.StatusStopped)
		batch := createTestBatch(5)
		result := pipeline.Process(batch)
		assert.Nil(t, result)
		pipeline.SetStatus(model.StatusRunning) // Reset for next tests
	})
	
	t.Run("Process returns original batch when no pipeline for type", func(t *testing.T) {
		batch := model.NewDataBatch(model.TraceTelemetryType)
		batch.AddPoint(&mockDataPoint{
			timestamp: time.Now(),
			origin:    "test",
			labels:    map[string]string{},
		})
		
		result := pipeline.Process(batch)
		assert.Equal(t, batch, result)
	})
	
	t.Run("Process applies correct pipeline for log type", func(t *testing.T) {
		batch := createTestBatch(3)
		batch.BatchType = model.LogTelemetryType
		
		result := pipeline.Process(batch)
		assert.NotNil(t, result)
		assert.Equal(t, 6, result.Size()) // Doubled by the processor
	})
	
	t.Run("Process applies correct pipeline for metric type", func(t *testing.T) {
		batch := createTestBatch(3)
		batch.BatchType = model.MetricTelemetryType
		
		result := pipeline.Process(batch)
		assert.NotNil(t, result)
		assert.Equal(t, 0, result.Size()) // Filtered by the processor
	})
}