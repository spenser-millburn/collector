package core

import (
	"sync"
	"time"

	"github.com/sliink/collector/internal/model"
)

// BufferManager handles data buffering and backpressure
type BufferManager struct {
	buffers      map[string][]*model.DataBatch
	maxQueueSize int
	status       map[string]model.BufferStatus
	mutex        sync.RWMutex
	BaseComponent
}

// NewBufferManager creates a new buffer manager
func NewBufferManager(maxQueueSize int) *BufferManager {
	if maxQueueSize <= 0 {
		maxQueueSize = 1000 // Default max queue size
	}

	return &BufferManager{
		buffers:       make(map[string][]*model.DataBatch),
		maxQueueSize:  maxQueueSize,
		status:        make(map[string]model.BufferStatus),
		BaseComponent: NewBaseComponent("buffer_manager", "Buffer Manager"),
	}
}

// Initialize prepares the buffer manager for operation
func (b *BufferManager) Initialize() bool {
	b.SetStatus(model.StatusInitialized)
	return true
}

// Start begins buffer manager operation
func (b *BufferManager) Start() bool {
	b.SetStatus(model.StatusRunning)
	return true
}

// Stop halts buffer manager operation
func (b *BufferManager) Stop() bool {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	// Clear all buffers
	b.buffers = make(map[string][]*model.DataBatch)
	b.status = make(map[string]model.BufferStatus)
	
	b.SetStatus(model.StatusStopped)
	return true
}

// Buffer adds a data batch to the buffer for a specific output
func (b *BufferManager) Buffer(outputID string, batch *model.DataBatch) bool {
	if batch == nil || batch.Size() == 0 {
		return true // Nothing to buffer
	}

	b.mutex.Lock()
	defer b.mutex.Unlock()

	if b.GetStatus() != model.StatusRunning {
		return false
	}

	// Initialize buffer for output if not exists
	if _, exists := b.buffers[outputID]; !exists {
		b.buffers[outputID] = make([]*model.DataBatch, 0)
		b.status[outputID] = model.BufferStatus{
			BufferID:   outputID,
			QueueSize:  0,
			TotalItems: 0,
			IsFull:     false,
			LastUpdate: time.Now(),
		}
	}

	// Check if buffer is full
	if len(b.buffers[outputID]) >= b.maxQueueSize {
		// Update status
		status := b.status[outputID]
		status.IsFull = true
		status.LastUpdate = time.Now()
		b.status[outputID] = status
		
		return false // Buffer is full
	}

	// Add batch to buffer
	b.buffers[outputID] = append(b.buffers[outputID], batch)
	
	// Update status
	status := b.status[outputID]
	status.QueueSize = len(b.buffers[outputID])
	status.TotalItems += batch.Size()
	status.IsFull = len(b.buffers[outputID]) >= b.maxQueueSize
	status.LastUpdate = time.Now()
	b.status[outputID] = status
	
	return true
}

// Flush retrieves batches from the buffer for a specific output
func (b *BufferManager) Flush(outputID string, maxBatches int) []*model.DataBatch {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if b.GetStatus() != model.StatusRunning {
		return nil
	}

	// Check if buffer exists for output
	if _, exists := b.buffers[outputID]; !exists {
		return nil
	}

	// Determine number of batches to return
	numBatches := len(b.buffers[outputID])
	if maxBatches > 0 && maxBatches < numBatches {
		numBatches = maxBatches
	}

	if numBatches == 0 {
		return nil
	}

	// Get batches to return
	result := b.buffers[outputID][:numBatches]
	
	// Update buffer
	b.buffers[outputID] = b.buffers[outputID][numBatches:]
	
	// Calculate total items in returned batches
	totalItems := 0
	for _, batch := range result {
		totalItems += batch.Size()
	}
	
	// Update status
	status := b.status[outputID]
	status.QueueSize = len(b.buffers[outputID])
	status.TotalItems -= totalItems
	status.IsFull = false // We just made room
	status.LastUpdate = time.Now()
	b.status[outputID] = status
	
	return result
}

// GetBufferStatus retrieves the status of all buffers
func (b *BufferManager) GetBufferStatus() map[string]model.BufferStatus {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	// Create a copy to avoid concurrent map access
	result := make(map[string]model.BufferStatus, len(b.status))
	for k, v := range b.status {
		result[k] = v
	}
	
	return result
}