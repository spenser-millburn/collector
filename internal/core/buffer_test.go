package core

import (
	"testing"
	"time"

	"github.com/sliink/collector/internal/model"
	"github.com/stretchr/testify/assert"
)

// mockDataPoint implements the DataPoint interface for testing
type mockDataPoint struct {
	timestamp time.Time
	origin    string
	labels    map[string]string
}

func (m *mockDataPoint) GetTimestamp() time.Time {
	return m.timestamp
}

func (m *mockDataPoint) GetOrigin() string {
	return m.origin
}

func (m *mockDataPoint) GetLabels() map[string]string {
	return m.labels
}

func (m *mockDataPoint) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"timestamp": m.timestamp,
		"origin":    m.origin,
		"labels":    m.labels,
	}
}

// Helper to create a test batch
func createTestBatch(size int) *model.DataBatch {
	batch := model.NewDataBatch(model.LogTelemetryType)
	for i := 0; i < size; i++ {
		point := &mockDataPoint{
			timestamp: time.Now(),
			origin:    "test",
			labels:    map[string]string{"test": "value"},
		}
		batch.AddPoint(point)
	}
	return batch
}

func TestNewBufferManager(t *testing.T) {
	t.Run("Creates with valid max queue size", func(t *testing.T) {
		manager := NewBufferManager(500)
		assert.Equal(t, 500, manager.maxQueueSize)
		assert.NotNil(t, manager.buffers)
		assert.NotNil(t, manager.status)
	})

	t.Run("Creates with default queue size when invalid", func(t *testing.T) {
		manager := NewBufferManager(0)
		assert.Equal(t, 1000, manager.maxQueueSize)

		manager = NewBufferManager(-10)
		assert.Equal(t, 1000, manager.maxQueueSize)
	})
}

func TestBufferManagerLifecycle(t *testing.T) {
	manager := NewBufferManager(10)

	t.Run("Initialize sets correct status", func(t *testing.T) {
		success := manager.Initialize()
		assert.True(t, success)
		assert.Equal(t, model.StatusInitialized, manager.GetStatus())
	})

	t.Run("Start sets correct status", func(t *testing.T) {
		success := manager.Start()
		assert.True(t, success)
		assert.Equal(t, model.StatusRunning, manager.GetStatus())
	})

	t.Run("Stop clears buffers and sets correct status", func(t *testing.T) {
		// Add a batch to the buffer first
		outputID := "test_output"
		batch := createTestBatch(5)
		success := manager.Buffer(outputID, batch)
		assert.True(t, success)
		assert.NotEmpty(t, manager.buffers)

		// Now stop and verify everything is cleared
		success = manager.Stop()
		assert.True(t, success)
		assert.Empty(t, manager.buffers)
		assert.Empty(t, manager.status)
		assert.Equal(t, model.StatusStopped, manager.GetStatus())
	})
}

func TestBufferManagerBuffer(t *testing.T) {
	manager := NewBufferManager(5)
	manager.Initialize()
	manager.Start()
	outputID := "test_output"

	t.Run("Buffer handles nil batch", func(t *testing.T) {
		success := manager.Buffer(outputID, nil)
		assert.True(t, success)
		assert.Empty(t, manager.buffers)
	})

	t.Run("Buffer handles empty batch", func(t *testing.T) {
		batch := model.NewDataBatch(model.LogTelemetryType)
		success := manager.Buffer(outputID, batch)
		assert.True(t, success)
		assert.Empty(t, manager.buffers)
	})

	t.Run("Buffer adds batch to queue", func(t *testing.T) {
		batch := createTestBatch(3)
		success := manager.Buffer(outputID, batch)
		assert.True(t, success)
		
		// Verify the buffer contains the batch
		assert.Len(t, manager.buffers[outputID], 1)
		assert.Equal(t, batch, manager.buffers[outputID][0])
		
		// Verify status is updated
		status := manager.status[outputID]
		assert.Equal(t, outputID, status.BufferID)
		assert.Equal(t, 1, status.QueueSize)
		assert.Equal(t, 3, status.TotalItems)
		assert.False(t, status.IsFull)
	})

	t.Run("Buffer fails when queue is full", func(t *testing.T) {
		// Add batches until the buffer is full (we already have 1)
		for i := 0; i < 4; i++ {
			batch := createTestBatch(1)
			success := manager.Buffer(outputID, batch)
			assert.True(t, success)
		}
		
		// Verify the buffer is now full
		assert.Len(t, manager.buffers[outputID], 5)
		
		// Try to add one more batch
		batch := createTestBatch(1)
		success := manager.Buffer(outputID, batch)
		assert.False(t, success)
		
		// Verify status shows full
		status := manager.status[outputID]
		assert.True(t, status.IsFull)
	})

	t.Run("Buffer fails when not running", func(t *testing.T) {
		manager.SetStatus(model.StatusStopped)
		batch := createTestBatch(1)
		success := manager.Buffer("new_output", batch)
		assert.False(t, success)
	})
}

func TestBufferManagerFlush(t *testing.T) {
	manager := NewBufferManager(10)
	manager.Initialize()
	manager.Start()
	outputID := "test_output"

	t.Run("Flush returns nil when buffer doesn't exist", func(t *testing.T) {
		result := manager.Flush("nonexistent", 5)
		assert.Nil(t, result)
	})

	t.Run("Flush returns nil when buffer is empty", func(t *testing.T) {
		// Initialize empty buffer
		manager.buffers[outputID] = []*model.DataBatch{}
		manager.status[outputID] = model.BufferStatus{
			BufferID:   outputID,
			QueueSize:  0,
			TotalItems: 0,
			IsFull:     false,
			LastUpdate: time.Now(),
		}
		
		result := manager.Flush(outputID, 5)
		assert.Nil(t, result)
	})

	t.Run("Flush returns all batches when maxBatches is 0", func(t *testing.T) {
		// Add 3 batches
		for i := 0; i < 3; i++ {
			batch := createTestBatch(2)
			manager.Buffer(outputID, batch)
		}
		
		result := manager.Flush(outputID, 0)
		assert.Len(t, result, 3)
		assert.Empty(t, manager.buffers[outputID])
		
		// Verify status is updated
		status := manager.status[outputID]
		assert.Equal(t, 0, status.QueueSize)
		assert.Equal(t, 0, status.TotalItems)
		assert.False(t, status.IsFull)
	})

	t.Run("Flush returns limited batches when maxBatches > 0", func(t *testing.T) {
		// Add 5 batches
		for i := 0; i < 5; i++ {
			batch := createTestBatch(1)
			manager.Buffer(outputID, batch)
		}
		
		// Flush only 2 batches
		result := manager.Flush(outputID, 2)
		assert.Len(t, result, 2)
		assert.Len(t, manager.buffers[outputID], 3)
		
		// Verify status is updated
		status := manager.status[outputID]
		assert.Equal(t, 3, status.QueueSize)
		assert.Equal(t, 3, status.TotalItems)
		assert.False(t, status.IsFull)
	})

	t.Run("Flush returns nil when not running", func(t *testing.T) {
		manager.SetStatus(model.StatusStopped)
		result := manager.Flush(outputID, 1)
		assert.Nil(t, result)
	})
}

func TestBufferManagerGetBufferStatus(t *testing.T) {
	manager := NewBufferManager(10)
	manager.Initialize()
	manager.Start()
	
	outputID1 := "output1"
	outputID2 := "output2"
	
	// Add batches to different outputs
	batch1 := createTestBatch(3)
	batch2 := createTestBatch(5)
	
	manager.Buffer(outputID1, batch1)
	manager.Buffer(outputID2, batch2)
	
	t.Run("GetBufferStatus returns a copy of all statuses", func(t *testing.T) {
		statuses := manager.GetBufferStatus()
		
		assert.Len(t, statuses, 2)
		assert.Contains(t, statuses, outputID1)
		assert.Contains(t, statuses, outputID2)
		
		assert.Equal(t, 1, statuses[outputID1].QueueSize)
		assert.Equal(t, 3, statuses[outputID1].TotalItems)
		
		assert.Equal(t, 1, statuses[outputID2].QueueSize)
		assert.Equal(t, 5, statuses[outputID2].TotalItems)
	})
	
	t.Run("Modifying returned status doesn't affect internal state", func(t *testing.T) {
		statuses := manager.GetBufferStatus()
		originalStatus := statuses[outputID1]
		
		// Modify the returned status
		statuses[outputID1] = model.BufferStatus{
			BufferID:   outputID1,
			QueueSize:  999,
			TotalItems: 999,
			IsFull:     true,
			LastUpdate: time.Now(),
		}
		
		// Get status again and verify it's unchanged
		newStatuses := manager.GetBufferStatus()
		assert.Equal(t, originalStatus.QueueSize, newStatuses[outputID1].QueueSize)
		assert.Equal(t, originalStatus.TotalItems, newStatuses[outputID1].TotalItems)
		assert.Equal(t, originalStatus.IsFull, newStatuses[outputID1].IsFull)
	})
}