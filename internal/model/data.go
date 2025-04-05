package model

import (
	"time"
)

// DataPoint represents a single telemetry data point
type DataPoint interface {
	// GetTimestamp returns the time the data point was created
	GetTimestamp() time.Time
	
	// GetOrigin returns the source of the data point
	GetOrigin() string
	
	// GetLabels returns metadata labels for the data point
	GetLabels() map[string]string
	
	// ToMap converts the data point to a map representation
	ToMap() map[string]interface{}
}

// BaseDataPoint provides common functionality for all data points
type BaseDataPoint struct {
	Timestamp time.Time
	Origin    string
	Labels    map[string]string
}

// GetTimestamp returns the time the data point was created
func (p *BaseDataPoint) GetTimestamp() time.Time {
	return p.Timestamp
}

// GetOrigin returns the source of the data point
func (p *BaseDataPoint) GetOrigin() string {
	return p.Origin
}

// GetLabels returns metadata labels for the data point
func (p *BaseDataPoint) GetLabels() map[string]string {
	return p.Labels
}

// LogPoint represents a log entry
type LogPoint struct {
	BaseDataPoint
	Message    string
	Level      string
	Attributes map[string]interface{}
}

// ToMap converts the log point to a map representation
func (p *LogPoint) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"timestamp":  p.Timestamp,
		"origin":     p.Origin,
		"labels":     p.Labels,
		"message":    p.Message,
		"level":      p.Level,
		"attributes": p.Attributes,
	}
}

// MetricPoint represents a metric measurement
type MetricPoint struct {
	BaseDataPoint
	Name       string
	Value      float64
	MetricType string
	Dimensions map[string]string
}

// ToMap converts the metric point to a map representation
func (p *MetricPoint) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"timestamp":   p.Timestamp,
		"origin":      p.Origin,
		"labels":      p.Labels,
		"name":        p.Name,
		"value":       p.Value,
		"metric_type": p.MetricType,
		"dimensions":  p.Dimensions,
	}
}

// TracePoint represents a trace span
type TracePoint struct {
	BaseDataPoint
	TraceID      string
	SpanID       string
	ParentSpanID string
	StartTime    time.Time
	EndTime      time.Time
}

// ToMap converts the trace point to a map representation
func (p *TracePoint) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"timestamp":      p.Timestamp,
		"origin":         p.Origin,
		"labels":         p.Labels,
		"trace_id":       p.TraceID,
		"span_id":        p.SpanID,
		"parent_span_id": p.ParentSpanID,
		"start_time":     p.StartTime,
		"end_time":       p.EndTime,
	}
}

// DataBatch is a collection of data points of the same type
type DataBatch struct {
	SourceID    string
	BatchType   TelemetryType
	Points      []DataPoint
	Records     []Record
	Timestamp   time.Time
	Attributes  map[string]interface{}
}

// NewDataBatch creates a new data batch of the specified type
func NewDataBatch(batchType TelemetryType) *DataBatch {
	return &DataBatch{
		BatchType:   batchType,
		Points:      make([]DataPoint, 0),
		Records:     make([]Record, 0),
		Timestamp:   time.Now(),
		Attributes:  make(map[string]interface{}),
	}
}

// AddPoint adds a data point to the batch
func (b *DataBatch) AddPoint(point DataPoint) {
	b.Points = append(b.Points, point)
}

// Size returns the number of data points in the batch
func (b *DataBatch) Size() int {
	return len(b.Points)
}

// ToMap converts the data batch to a map representation
func (b *DataBatch) ToMap() map[string]interface{} {
	points := make([]map[string]interface{}, len(b.Points))
	for i, point := range b.Points {
		points[i] = point.ToMap()
	}
	
	// Convert records to a simple format
	recordsData := make([]map[string]interface{}, len(b.Records))
	for i, record := range b.Records {
		recordsData[i] = map[string]interface{}{
			"source":     record.Source,
			"timestamp":  record.Timestamp,
			"attributes": record.Attributes,
			"data":       string(record.RawData), // Convert binary data to string for display
		}
	}
	
	return map[string]interface{}{
		"source_id":   b.SourceID,
		"batch_type":  b.BatchType,
		"timestamp":   b.Timestamp,
		"points":      points,
		"records":     recordsData,
		"attributes":  b.Attributes,
	}
}