package outputs

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sliink/collector/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestNewStdoutOutput(t *testing.T) {
	t.Run("Creates StdoutOutput with correct properties", func(t *testing.T) {
		output := NewStdoutOutput("stdout_output")
		
		assert.Equal(t, "stdout_output", output.ID())
		assert.Equal(t, "Stdout Output", output.Name())
		assert.Equal(t, model.OutputPluginType, output.GetType())
		assert.Equal(t, model.StatusUninitialized, output.GetStatus())
		assert.False(t, output.colorize)
		assert.Equal(t, "text", output.format)
	})
}

func TestStdoutOutputLifecycle(t *testing.T) {
	output := NewStdoutOutput("stdout_output")
	
	t.Run("Initialize sets default values", func(t *testing.T) {
		success := output.Initialize()
		assert.True(t, success)
		assert.Equal(t, model.StatusInitialized, output.GetStatus())
		assert.False(t, output.colorize)
		assert.Equal(t, "text", output.format)
	})
	
	t.Run("Initialize applies configuration", func(t *testing.T) {
		output := NewStdoutOutput("stdout_output")
		output.Config = map[string]interface{}{
			"colorize": true,
			"format":   "json",
		}
		
		success := output.Initialize()
		assert.True(t, success)
		assert.True(t, output.colorize)
		assert.Equal(t, "json", output.format)
	})
	
	t.Run("Start sets correct status", func(t *testing.T) {
		success := output.Start()
		assert.True(t, success)
		assert.Equal(t, model.StatusRunning, output.GetStatus())
	})
	
	t.Run("Stop sets correct status", func(t *testing.T) {
		success := output.Stop()
		assert.True(t, success)
		assert.Equal(t, model.StatusStopped, output.GetStatus())
	})
}

func TestStdoutOutputValidate(t *testing.T) {
	output := NewStdoutOutput("stdout_output")
	
	t.Run("Validate always returns true", func(t *testing.T) {
		success := output.Validate()
		assert.True(t, success)
	})
}

func captureStdout(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	
	f()
	
	w.Close()
	os.Stdout = old
	
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestStdoutOutputSend(t *testing.T) {
	t.Run("Send handles nil batch", func(t *testing.T) {
		output := NewStdoutOutput("stdout_output")
		output.Initialize()
		output.Start()
		
		success := output.Send(nil)
		assert.True(t, success)
	})
	
	t.Run("Send handles empty batch", func(t *testing.T) {
		output := NewStdoutOutput("stdout_output")
		output.Initialize()
		output.Start()
		
		batch := model.NewDataBatch(model.LogTelemetryType)
		success := output.Send(batch)
		assert.True(t, success)
	})
	
	t.Run("Send returns false when not running", func(t *testing.T) {
		output := NewStdoutOutput("stdout_output")
		output.Initialize()
		// Not started
		
		batch := model.NewDataBatch(model.LogTelemetryType)
		batch.AddPoint(&model.LogPoint{
			BaseDataPoint: model.BaseDataPoint{
				Timestamp: time.Now(),
				Origin:    "test",
			},
			Message: "Test message",
			Level:   "INFO",
		})
		
		success := output.Send(batch)
		assert.False(t, success)
	})
	
	t.Run("Send outputs text format correctly", func(t *testing.T) {
		output := NewStdoutOutput("stdout_output")
		output.Config = map[string]interface{}{
			"format": "text",
		}
		output.Initialize()
		output.Start()
		
		// Create a test log point
		now := time.Now()
		logPoint := &model.LogPoint{
			BaseDataPoint: model.BaseDataPoint{
				Timestamp: now,
				Origin:    "test",
				Labels:    map[string]string{"env": "test"},
			},
			Message:    "Test log message",
			Level:      "INFO",
			Attributes: map[string]interface{}{"requestId": "123"},
		}
		
		batch := model.NewDataBatch(model.LogTelemetryType)
		batch.AddPoint(logPoint)
		
		// Capture stdout during Send
		capturedOutput := captureStdout(func() {
			output.Send(batch)
		})
		
		// Verify output format
		assert.Contains(t, capturedOutput, now.Format(time.RFC3339))
		assert.Contains(t, capturedOutput, "INFO")
		assert.Contains(t, capturedOutput, "Test log message")
		assert.Contains(t, capturedOutput, "requestId")
		assert.Contains(t, capturedOutput, "123")
	})
	
	t.Run("Send outputs JSON format correctly", func(t *testing.T) {
		output := NewStdoutOutput("stdout_output")
		output.Config = map[string]interface{}{
			"format": "json",
		}
		output.Initialize()
		output.Start()
		
		// Create a test metric point
		now := time.Now()
		metricPoint := &model.MetricPoint{
			BaseDataPoint: model.BaseDataPoint{
				Timestamp: now,
				Origin:    "test",
				Labels:    map[string]string{"env": "test"},
			},
			Name:       "cpu_usage",
			Value:      0.75,
			MetricType: "gauge",
			Dimensions: map[string]string{"host": "server1"},
		}
		
		batch := model.NewDataBatch(model.MetricTelemetryType)
		batch.AddPoint(metricPoint)
		
		// Capture stdout during Send
		capturedOutput := captureStdout(func() {
			output.Send(batch)
		})
		
		// Verify output is valid JSON
		var result map[string]interface{}
		err := json.Unmarshal([]byte(strings.TrimSpace(capturedOutput)), &result)
		assert.NoError(t, err)
		
		// Check JSON content
		assert.Equal(t, "cpu_usage", result["name"])
		assert.Equal(t, 0.75, result["value"])
		assert.Equal(t, "gauge", result["metric_type"])
		assert.Equal(t, "test", result["origin"])
	})
	
	t.Run("Send handles different point types", func(t *testing.T) {
		output := NewStdoutOutput("stdout_output")
		output.Initialize()
		output.Start()
		
		now := time.Now()
		
		// Create different types of points
		logPoint := &model.LogPoint{
			BaseDataPoint: model.BaseDataPoint{
				Timestamp: now,
				Origin:    "test",
			},
			Message: "Log message",
			Level:   "INFO",
		}
		
		metricPoint := &model.MetricPoint{
			BaseDataPoint: model.BaseDataPoint{
				Timestamp: now,
				Origin:    "test",
			},
			Name:       "metric",
			Value:      1.0,
			MetricType: "counter",
		}
		
		tracePoint := &model.TracePoint{
			BaseDataPoint: model.BaseDataPoint{
				Timestamp: now,
				Origin:    "test",
			},
			TraceID:   "trace1",
			SpanID:    "span1",
			StartTime: now.Add(-1 * time.Second),
			EndTime:   now,
		}
		
		// Create batch with mixed points
		batch := model.NewDataBatch(model.LogTelemetryType)
		batch.AddPoint(logPoint)
		batch.AddPoint(metricPoint)
		batch.AddPoint(tracePoint)
		
		// This should handle all point types without error
		capturedOutput := captureStdout(func() {
			output.Send(batch)
		})
		
		// Verify all types were output
		assert.Contains(t, capturedOutput, "Log message")
		assert.Contains(t, capturedOutput, "METRIC metric")
		assert.Contains(t, capturedOutput, "TRACE trace1")
	})
}