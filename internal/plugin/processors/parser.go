package processors

import (
	"regexp"
	"time"

	"github.com/sliink/collector/internal/model"
	"github.com/sliink/collector/internal/plugin"
)

// Parser processes raw logs into structured format
type Parser struct {
	plugin.BasePlugin
	patterns []*regexp.Regexp
}

// NewParser creates a new parser plugin
func NewParser(id string) *Parser {
	return &Parser{
		BasePlugin: plugin.NewBasePlugin(id, "Parser", model.ProcessorPluginType),
		patterns:   make([]*regexp.Regexp, 0),
	}
}

// Initialize prepares the parser for operation
func (p *Parser) Initialize() bool {
	// Get patterns from configuration
	if patterns, ok := p.Config["patterns"].([]interface{}); ok {
		for _, pat := range patterns {
			if patStr, ok := pat.(string); ok {
				if regex, err := regexp.Compile(patStr); err == nil {
					p.patterns = append(p.patterns, regex)
				}
			}
		}
	}

	p.SetStatus(model.StatusInitialized)
	return len(p.patterns) > 0
}

// Start begins parser operation
func (p *Parser) Start() bool {
	p.SetStatus(model.StatusRunning)
	return true
}

// Stop halts parser operation
func (p *Parser) Stop() bool {
	p.SetStatus(model.StatusStopped)
	return true
}

// Validate checks if the parser is properly configured
func (p *Parser) Validate() bool {
	// Check if patterns are configured
	if patterns, ok := p.Config["patterns"].([]interface{}); !ok || len(patterns) == 0 {
		return false
	}

	return true
}

// Process transforms a log batch into structured format
func (p *Parser) Process(batch *model.DataBatch) *model.DataBatch {
	if batch == nil || batch.Size() == 0 || batch.BatchType != model.LogTelemetryType {
		return batch
	}

	if p.GetStatus() != model.StatusRunning {
		return batch
	}

	resultBatch := model.NewDataBatch(model.LogTelemetryType)
	resultBatch.Metadata = batch.Metadata

	for _, point := range batch.Points {
		logPoint, ok := point.(*model.LogPoint)
		if !ok {
			continue
		}

		// Process the log message
		processedPoint := p.processLogPoint(logPoint)
		resultBatch.AddPoint(processedPoint)
	}

	return resultBatch
}

// processLogPoint parses a log message into structured data
func (p *Parser) processLogPoint(logPoint *model.LogPoint) *model.LogPoint {
	// Create a new log point with the same base data
	processed := &model.LogPoint{
		BaseDataPoint: logPoint.BaseDataPoint,
		Message:       logPoint.Message,
		Level:         logPoint.Level,
		Attributes:    make(map[string]interface{}),
	}

	// Copy existing attributes
	for k, v := range logPoint.Attributes {
		processed.Attributes[k] = v
	}

	// Apply patterns
	for _, pattern := range p.patterns {
		matches := pattern.FindStringSubmatch(logPoint.Message)
		if len(matches) > 0 {
			// Extract named capture groups
			for i, name := range pattern.SubexpNames() {
				if i > 0 && name != "" {
					processed.Attributes[name] = matches[i]
					
					// Special handling for common fields
					if name == "level" {
						processed.Level = matches[i]
					} else if name == "timestamp" {
						// Try to parse timestamp
						if ts, err := time.Parse(time.RFC3339, matches[i]); err == nil {
							processed.Timestamp = ts
						}
					}
				}
			}
			break
		}
	}

	return processed
}