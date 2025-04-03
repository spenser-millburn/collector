package core

import (
	"sync"
	"time"

	"github.com/sliink/collector/internal/model"
)

// HealthMonitor tracks system and component health
type HealthMonitor struct {
	components map[string]Component
	metrics    map[string]interface{}
	mutex      sync.RWMutex
	BaseComponent
}

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor() *HealthMonitor {
	return &HealthMonitor{
		components:    make(map[string]Component),
		metrics:       make(map[string]interface{}),
		BaseComponent: NewBaseComponent("health_monitor", "Health Monitor"),
	}
}

// Initialize prepares the health monitor for operation
func (h *HealthMonitor) Initialize() bool {
	h.SetStatus(model.StatusInitialized)
	return true
}

// Start begins health monitor operation
func (h *HealthMonitor) Start() bool {
	h.SetStatus(model.StatusRunning)
	return true
}

// Stop halts health monitor operation
func (h *HealthMonitor) Stop() bool {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	// Clear all metrics
	h.metrics = make(map[string]interface{})
	
	h.SetStatus(model.StatusStopped)
	return true
}

// RegisterComponent adds a component to be monitored
func (h *HealthMonitor) RegisterComponent(component Component) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.components[component.ID()] = component
}

// AddMetric adds a metric value with optional metadata
func (h *HealthMonitor) AddMetric(name string, value interface{}, metadata map[string]interface{}) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	
	metadata["value"] = value
	metadata["timestamp"] = time.Now()
	
	h.metrics[name] = metadata
}

// GetMetric retrieves a metric value
func (h *HealthMonitor) GetMetric(name string) (interface{}, bool) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	metric, exists := h.metrics[name]
	return metric, exists
}

// GetAllMetrics retrieves all metrics
func (h *HealthMonitor) GetAllMetrics() map[string]interface{} {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	// Create a copy to avoid concurrent map access
	metrics := make(map[string]interface{}, len(h.metrics))
	for k, v := range h.metrics {
		metrics[k] = v
	}
	
	return metrics
}

// GetHealthStatus retrieves the health status of the system
func (h *HealthMonitor) GetHealthStatus() model.HealthStatus {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	// Build component status map
	components := make(map[string]model.HealthStatus)
	for id, component := range h.components {
		components[id] = model.HealthStatus{
			Status:    component.GetStatus(),
			Timestamp: time.Now(),
			Message:   component.Name() + " status: " + string(component.GetStatus()),
		}
	}

	// Determine system status based on component statuses
	systemStatus := model.StatusRunning
	var statusMessage string
	
	// Count components in each status
	statusCounts := make(map[model.ComponentStatus]int)
	for _, health := range components {
		statusCounts[health.Status]++
	}
	
	// Check for error components
	if statusCounts[model.StatusError] > 0 {
		systemStatus = model.StatusError
		statusMessage = "System has errors: " + string(statusCounts[model.StatusError]) + " components in ERROR state"
	} else if statusCounts[model.StatusStopped] > 0 && statusCounts[model.StatusStopped] == len(components) {
		systemStatus = model.StatusStopped
		statusMessage = "System is stopped"
	} else if statusCounts[model.StatusRunning] == 0 {
		systemStatus = model.StatusInitialized
		statusMessage = "System is initializing"
	} else if statusCounts[model.StatusRunning] < len(components) {
		statusMessage = "System is partially running: " + string(statusCounts[model.StatusRunning]) + " of " + string(len(components)) + " components running"
	} else {
		statusMessage = "System is healthy: all components running"
	}

	return model.HealthStatus{
		Status:     systemStatus,
		Timestamp:  time.Now(),
		Message:    statusMessage,
		Components: components,
		Details:    h.GetAllMetrics(),
	}
}