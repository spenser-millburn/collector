package outputs

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/sliink/collector/internal/model"
	"github.com/sliink/collector/internal/plugin"
)

// StdoutOutput writes data to standard output
type StdoutOutput struct {
	plugin.BasePlugin
	colorize bool
	format   string
}

// NewStdoutOutput creates a new stdout output plugin
func NewStdoutOutput(id string) *StdoutOutput {
	return &StdoutOutput{
		BasePlugin: plugin.NewBasePlugin(id, "Stdout Output", model.OutputPluginType),
		colorize:   false,
		format:     "text",
	}
}

// Initialize prepares the stdout output for operation
func (s *StdoutOutput) Initialize() bool {
	// Get configuration
	if colorize, ok := s.Config["colorize"].(bool); ok {
		s.colorize = colorize
	}

	if format, ok := s.Config["format"].(string); ok {
		s.format = format
	}

	s.SetStatus(model.StatusInitialized)
	return true
}

// Start begins stdout output operation
func (s *StdoutOutput) Start() bool {
	s.SetStatus(model.StatusRunning)
	return true
}

// Stop halts stdout output operation
func (s *StdoutOutput) Stop() bool {
	s.SetStatus(model.StatusStopped)
	return true
}

// Validate checks if the stdout output is properly configured
func (s *StdoutOutput) Validate() bool {
	// No specific validation required
	return true
}

// Send exports a data batch to stdout
func (s *StdoutOutput) Send(batch *model.DataBatch) bool {
	if batch == nil || batch.Size() == 0 {
		return true
	}

	if s.GetStatus() != model.StatusRunning {
		return false
	}

	// Output each point
	for _, point := range batch.Points {
		if s.format == "json" {
			s.outputJSON(point)
		} else {
			s.outputText(point)
		}
	}

	return true
}

// outputJSON formats and prints a data point as JSON
func (s *StdoutOutput) outputJSON(point model.DataPoint) {
	data, err := json.Marshal(point.ToMap())
	if err != nil {
		return
	}

	fmt.Println(string(data))
}

// outputText formats and prints a data point as text
func (s *StdoutOutput) outputText(point model.DataPoint) {
	switch p := point.(type) {
	case *model.LogPoint:
		s.outputLogPoint(p)
	case *model.MetricPoint:
		s.outputMetricPoint(p)
	case *model.TracePoint:
		s.outputTracePoint(p)
	default:
		fmt.Printf("Unknown point type: %T\n", point)
	}
}

// outputLogPoint formats and prints a log point
func (s *StdoutOutput) outputLogPoint(point *model.LogPoint) {
	timestamp := point.Timestamp.Format(time.RFC3339)
	
	// Apply color if enabled
	level := point.Level
	if s.colorize {
		switch point.Level {
		case "ERROR", "FATAL":
			level = "\033[31m" + level + "\033[0m" // Red
		case "WARN", "WARNING":
			level = "\033[33m" + level + "\033[0m" // Yellow
		case "INFO":
			level = "\033[32m" + level + "\033[0m" // Green
		case "DEBUG":
			level = "\033[36m" + level + "\033[0m" // Cyan
		case "TRACE":
			level = "\033[35m" + level + "\033[0m" // Magenta
		}
	}
	
	fmt.Printf("[%s] %s: %s\n", timestamp, level, point.Message)
	
	// Output attributes if any
	if len(point.Attributes) > 0 {
		attributesJSON, _ := json.Marshal(point.Attributes)
		fmt.Printf("  %s\n", string(attributesJSON))
	}
}

// outputMetricPoint formats and prints a metric point
func (s *StdoutOutput) outputMetricPoint(point *model.MetricPoint) {
	timestamp := point.Timestamp.Format(time.RFC3339)
	fmt.Printf("[%s] METRIC %s: %f\n", timestamp, point.Name, point.Value)
	
	// Output dimensions if any
	if len(point.Dimensions) > 0 {
		dimensionsJSON, _ := json.Marshal(point.Dimensions)
		fmt.Printf("  %s\n", string(dimensionsJSON))
	}
}

// outputTracePoint formats and prints a trace point
func (s *StdoutOutput) outputTracePoint(point *model.TracePoint) {
	timestamp := point.Timestamp.Format(time.RFC3339)
	duration := point.EndTime.Sub(point.StartTime).Milliseconds()
	
	fmt.Printf("[%s] TRACE %s (span: %s): %dms\n", 
		timestamp, point.TraceID, point.SpanID, duration)
}