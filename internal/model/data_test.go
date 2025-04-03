package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBaseDataPoint(t *testing.T) {
	now := time.Now()
	labels := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}
	
	basePoint := BaseDataPoint{
		Timestamp: now,
		Origin:    "test-origin",
		Labels:    labels,
	}
	
	t.Run("GetTimestamp returns correct timestamp", func(t *testing.T) {
		assert.Equal(t, now, basePoint.GetTimestamp())
	})
	
	t.Run("GetOrigin returns correct origin", func(t *testing.T) {
		assert.Equal(t, "test-origin", basePoint.GetOrigin())
	})
	
	t.Run("GetLabels returns correct labels", func(t *testing.T) {
		assert.Equal(t, labels, basePoint.GetLabels())
	})
}

func TestLogPoint(t *testing.T) {
	now := time.Now()
	labels := map[string]string{"env": "test"}
	attributes := map[string]interface{}{
		"requestId": "123",
		"userId":    456,
	}
	
	logPoint := LogPoint{
		BaseDataPoint: BaseDataPoint{
			Timestamp: now,
			Origin:    "app-server",
			Labels:    labels,
		},
		Message:    "Test log message",
		Level:      "INFO",
		Attributes: attributes,
	}
	
	t.Run("Inherits BaseDataPoint methods", func(t *testing.T) {
		assert.Equal(t, now, logPoint.GetTimestamp())
		assert.Equal(t, "app-server", logPoint.GetOrigin())
		assert.Equal(t, labels, logPoint.GetLabels())
	})
	
	t.Run("ToMap includes all fields", func(t *testing.T) {
		result := logPoint.ToMap()
		
		assert.Equal(t, now, result["timestamp"])
		assert.Equal(t, "app-server", result["origin"])
		assert.Equal(t, labels, result["labels"])
		assert.Equal(t, "Test log message", result["message"])
		assert.Equal(t, "INFO", result["level"])
		assert.Equal(t, attributes, result["attributes"])
	})
}

func TestMetricPoint(t *testing.T) {
	now := time.Now()
	labels := map[string]string{"env": "test"}
	dimensions := map[string]string{
		"host":   "server1",
		"region": "us-west",
	}
	
	metricPoint := MetricPoint{
		BaseDataPoint: BaseDataPoint{
			Timestamp: now,
			Origin:    "monitoring",
			Labels:    labels,
		},
		Name:       "cpu_usage",
		Value:      0.75,
		MetricType: "gauge",
		Dimensions: dimensions,
	}
	
	t.Run("Inherits BaseDataPoint methods", func(t *testing.T) {
		assert.Equal(t, now, metricPoint.GetTimestamp())
		assert.Equal(t, "monitoring", metricPoint.GetOrigin())
		assert.Equal(t, labels, metricPoint.GetLabels())
	})
	
	t.Run("ToMap includes all fields", func(t *testing.T) {
		result := metricPoint.ToMap()
		
		assert.Equal(t, now, result["timestamp"])
		assert.Equal(t, "monitoring", result["origin"])
		assert.Equal(t, labels, result["labels"])
		assert.Equal(t, "cpu_usage", result["name"])
		assert.Equal(t, 0.75, result["value"])
		assert.Equal(t, "gauge", result["metric_type"])
		assert.Equal(t, dimensions, result["dimensions"])
	})
}

func TestTracePoint(t *testing.T) {
	now := time.Now()
	startTime := now.Add(-5 * time.Second)
	endTime := now
	labels := map[string]string{"env": "test"}
	
	tracePoint := TracePoint{
		BaseDataPoint: BaseDataPoint{
			Timestamp: now,
			Origin:    "api-service",
			Labels:    labels,
		},
		TraceID:      "trace-123",
		SpanID:       "span-456",
		ParentSpanID: "span-parent",
		StartTime:    startTime,
		EndTime:      endTime,
	}
	
	t.Run("Inherits BaseDataPoint methods", func(t *testing.T) {
		assert.Equal(t, now, tracePoint.GetTimestamp())
		assert.Equal(t, "api-service", tracePoint.GetOrigin())
		assert.Equal(t, labels, tracePoint.GetLabels())
	})
	
	t.Run("ToMap includes all fields", func(t *testing.T) {
		result := tracePoint.ToMap()
		
		assert.Equal(t, now, result["timestamp"])
		assert.Equal(t, "api-service", result["origin"])
		assert.Equal(t, labels, result["labels"])
		assert.Equal(t, "trace-123", result["trace_id"])
		assert.Equal(t, "span-456", result["span_id"])
		assert.Equal(t, "span-parent", result["parent_span_id"])
		assert.Equal(t, startTime, result["start_time"])
		assert.Equal(t, endTime, result["end_time"])
	})
}

func TestDataBatch(t *testing.T) {
	t.Run("NewDataBatch creates batch with correct type", func(t *testing.T) {
		batch := NewDataBatch(LogTelemetryType)
		
		assert.Equal(t, LogTelemetryType, batch.BatchType)
		assert.Empty(t, batch.Points)
		assert.NotNil(t, batch.Metadata)
	})
	
	t.Run("AddPoint adds points to batch", func(t *testing.T) {
		batch := NewDataBatch(MetricTelemetryType)
		
		point1 := &MetricPoint{
			BaseDataPoint: BaseDataPoint{
				Timestamp: time.Now(),
				Origin:    "test",
			},
			Name:       "test_metric",
			Value:      42.0,
			MetricType: "counter",
		}
		
		point2 := &MetricPoint{
			BaseDataPoint: BaseDataPoint{
				Timestamp: time.Now(),
				Origin:    "test",
			},
			Name:       "another_metric",
			Value:      99.9,
			MetricType: "gauge",
		}
		
		batch.AddPoint(point1)
		assert.Len(t, batch.Points, 1)
		
		batch.AddPoint(point2)
		assert.Len(t, batch.Points, 2)
		assert.Equal(t, point1, batch.Points[0])
		assert.Equal(t, point2, batch.Points[1])
	})
	
	t.Run("Size returns correct number of points", func(t *testing.T) {
		batch := NewDataBatch(TraceTelemetryType)
		assert.Equal(t, 0, batch.Size())
		
		point := &TracePoint{
			BaseDataPoint: BaseDataPoint{
				Timestamp: time.Now(),
				Origin:    "test",
			},
			TraceID: "test-trace",
			SpanID:  "test-span",
		}
		
		batch.AddPoint(point)
		assert.Equal(t, 1, batch.Size())
		
		batch.AddPoint(point)
		batch.AddPoint(point)
		assert.Equal(t, 3, batch.Size())
	})
	
	t.Run("ToMap includes all batch data", func(t *testing.T) {
		batch := NewDataBatch(LogTelemetryType)
		batch.Metadata["test_key"] = "test_value"
		
		point := &LogPoint{
			BaseDataPoint: BaseDataPoint{
				Timestamp: time.Now(),
				Origin:    "test",
			},
			Message: "Test message",
			Level:   "INFO",
		}
		
		batch.AddPoint(point)
		
		result := batch.ToMap()
		
		assert.Equal(t, LogTelemetryType, result["batch_type"])
		assert.Len(t, result["points"].([]map[string]interface{}), 1)
		assert.Equal(t, map[string]interface{}{"test_key": "test_value"}, result["metadata"])
		
		// Check that the point was properly converted
		pointMap := result["points"].([]map[string]interface{})[0]
		assert.Equal(t, "Test message", pointMap["message"])
		assert.Equal(t, "INFO", pointMap["level"])
	})
}