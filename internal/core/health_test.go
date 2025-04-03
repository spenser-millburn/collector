package core

import (
	"testing"

	"github.com/sliink/collector/internal/model"
	"github.com/stretchr/testify/assert"
)

// mockComponent implements the Component interface for testing
type mockComponent struct {
	id     string
	name   string
	status model.ComponentStatus
}

func (m *mockComponent) Initialize() bool {
	m.status = model.StatusInitialized
	return true
}

func (m *mockComponent) Start() bool {
	m.status = model.StatusRunning
	return true
}

func (m *mockComponent) Stop() bool {
	m.status = model.StatusStopped
	return true
}

func (m *mockComponent) GetStatus() model.ComponentStatus {
	return m.status
}

func (m *mockComponent) SetStatus(status model.ComponentStatus) {
	m.status = status
}

func (m *mockComponent) Configure(config map[string]interface{}) bool {
	return true
}

func (m *mockComponent) ID() string {
	return m.id
}

func (m *mockComponent) Name() string {
	return m.name
}

func newMockComponent(id, name string, status model.ComponentStatus) *mockComponent {
	return &mockComponent{
		id:     id,
		name:   name,
		status: status,
	}
}

func TestNewHealthMonitor(t *testing.T) {
	monitor := NewHealthMonitor()
	
	assert.NotNil(t, monitor)
	assert.NotNil(t, monitor.components)
	assert.NotNil(t, monitor.metrics)
	assert.Equal(t, "health_monitor", monitor.ID())
	assert.Equal(t, "Health Monitor", monitor.Name())
}

func TestHealthMonitorLifecycle(t *testing.T) {
	monitor := NewHealthMonitor()
	
	t.Run("Initialize sets correct status", func(t *testing.T) {
		success := monitor.Initialize()
		assert.True(t, success)
		assert.Equal(t, model.StatusInitialized, monitor.GetStatus())
	})
	
	t.Run("Start sets correct status", func(t *testing.T) {
		success := monitor.Start()
		assert.True(t, success)
		assert.Equal(t, model.StatusRunning, monitor.GetStatus())
	})
	
	t.Run("Stop clears metrics and sets correct status", func(t *testing.T) {
		// Add a metric first
		monitor.AddMetric("test_metric", 42, nil)
		assert.NotEmpty(t, monitor.metrics)
		
		// Now stop and verify metrics are cleared
		success := monitor.Stop()
		assert.True(t, success)
		assert.Empty(t, monitor.metrics)
		assert.Equal(t, model.StatusStopped, monitor.GetStatus())
	})
}

func TestRegisterComponent(t *testing.T) {
	monitor := NewHealthMonitor()
	
	t.Run("RegisterComponent adds component to tracked components", func(t *testing.T) {
		comp1 := newMockComponent("comp1", "Component 1", model.StatusRunning)
		comp2 := newMockComponent("comp2", "Component 2", model.StatusInitialized)
		
		monitor.RegisterComponent(comp1)
		monitor.RegisterComponent(comp2)
		
		assert.Len(t, monitor.components, 2)
		assert.Contains(t, monitor.components, "comp1")
		assert.Contains(t, monitor.components, "comp2")
	})
	
	t.Run("RegisterComponent overwrites existing component with same ID", func(t *testing.T) {
		comp1a := newMockComponent("comp1", "Component 1 Updated", model.StatusError)
		
		monitor.RegisterComponent(comp1a)
		
		assert.Len(t, monitor.components, 2)
		assert.Equal(t, comp1a, monitor.components["comp1"])
	})
}

func TestAddMetric(t *testing.T) {
	monitor := NewHealthMonitor()
	
	t.Run("AddMetric with nil metadata creates default metadata", func(t *testing.T) {
		monitor.AddMetric("metric1", 42, nil)
		
		metric, exists := monitor.GetMetric("metric1")
		assert.True(t, exists)
		
		metricMap, ok := metric.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, 42, metricMap["value"])
		assert.NotNil(t, metricMap["timestamp"])
	})
	
	t.Run("AddMetric with metadata adds value and timestamp", func(t *testing.T) {
		metadata := map[string]interface{}{
			"unit":        "bytes",
			"description": "Memory usage",
		}
		
		monitor.AddMetric("metric2", 1024, metadata)
		
		metric, exists := monitor.GetMetric("metric2")
		assert.True(t, exists)
		
		metricMap, ok := metric.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, 1024, metricMap["value"])
		assert.Equal(t, "bytes", metricMap["unit"])
		assert.Equal(t, "Memory usage", metricMap["description"])
		assert.NotNil(t, metricMap["timestamp"])
	})
}

func TestGetMetric(t *testing.T) {
	monitor := NewHealthMonitor()
	
	t.Run("GetMetric returns false for non-existent metric", func(t *testing.T) {
		_, exists := monitor.GetMetric("nonexistent")
		assert.False(t, exists)
	})
	
	t.Run("GetMetric returns true and metric for existing metric", func(t *testing.T) {
		monitor.AddMetric("test_metric", 42, nil)
		
		metric, exists := monitor.GetMetric("test_metric")
		assert.True(t, exists)
		assert.NotNil(t, metric)
	})
}

func TestGetAllMetrics(t *testing.T) {
	monitor := NewHealthMonitor()
	
	t.Run("GetAllMetrics returns empty map when no metrics", func(t *testing.T) {
		metrics := monitor.GetAllMetrics()
		assert.Empty(t, metrics)
	})
	
	t.Run("GetAllMetrics returns all metrics", func(t *testing.T) {
		monitor.AddMetric("metric1", 42, nil)
		monitor.AddMetric("metric2", "test", nil)
		
		metrics := monitor.GetAllMetrics()
		assert.Len(t, metrics, 2)
		assert.Contains(t, metrics, "metric1")
		assert.Contains(t, metrics, "metric2")
	})
	
	t.Run("Modifying returned metrics doesn't affect internal state", func(t *testing.T) {
		metrics := monitor.GetAllMetrics()
		metrics["metric3"] = "should not be added"
		
		newMetrics := monitor.GetAllMetrics()
		assert.Len(t, newMetrics, 2)
		assert.NotContains(t, newMetrics, "metric3")
	})
}

func TestGetHealthStatus(t *testing.T) {
	monitor := NewHealthMonitor()
	monitor.Initialize()
	
	t.Run("Empty monitor returns initialized status", func(t *testing.T) {
		health := monitor.GetHealthStatus()
		
		assert.Equal(t, model.StatusInitialized, health.Status)
		assert.Contains(t, health.Message, "initializing")
		assert.Empty(t, health.Components)
	})
	
	t.Run("All components running returns running status", func(t *testing.T) {
		comp1 := newMockComponent("comp1", "Component 1", model.StatusRunning)
		comp2 := newMockComponent("comp2", "Component 2", model.StatusRunning)
		
		monitor.RegisterComponent(comp1)
		monitor.RegisterComponent(comp2)
		
		health := monitor.GetHealthStatus()
		
		assert.Equal(t, model.StatusRunning, health.Status)
		assert.Contains(t, health.Message, "healthy")
		assert.Len(t, health.Components, 2)
	})
	
	t.Run("Some components running returns partially running status", func(t *testing.T) {
		comp1 := newMockComponent("comp1", "Component 1", model.StatusRunning)
		comp2 := newMockComponent("comp2", "Component 2", model.StatusInitialized)
		
		monitor.RegisterComponent(comp1)
		monitor.RegisterComponent(comp2)
		
		health := monitor.GetHealthStatus()
		
		assert.Equal(t, model.StatusInitialized, health.Status)
		assert.Contains(t, health.Message, "partially running")
	})
	
	t.Run("Any component in error returns error status", func(t *testing.T) {
		comp1 := newMockComponent("comp1", "Component 1", model.StatusRunning)
		comp2 := newMockComponent("comp2", "Component 2", model.StatusError)
		
		monitor.RegisterComponent(comp1)
		monitor.RegisterComponent(comp2)
		
		health := monitor.GetHealthStatus()
		
		assert.Equal(t, model.StatusError, health.Status)
		assert.Contains(t, health.Message, "error")
	})
	
	t.Run("All components stopped returns stopped status", func(t *testing.T) {
		comp1 := newMockComponent("comp1", "Component 1", model.StatusStopped)
		comp2 := newMockComponent("comp2", "Component 2", model.StatusStopped)
		
		monitor.RegisterComponent(comp1)
		monitor.RegisterComponent(comp2)
		
		health := monitor.GetHealthStatus()
		
		assert.Equal(t, model.StatusStopped, health.Status)
		assert.Contains(t, health.Message, "stopped")
	})
	
	t.Run("Health status includes metrics", func(t *testing.T) {
		monitor.AddMetric("metric1", 42, nil)
		
		health := monitor.GetHealthStatus()
		
		assert.NotNil(t, health.Details)
		assert.Contains(t, health.Details, "metric1")
	})
}