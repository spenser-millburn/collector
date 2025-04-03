package processors

import (
	"testing"
	"time"

	"github.com/sliink/collector/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestParserInitialize(t *testing.T) {
	t.Run("Successfully initializes with valid patterns", func(t *testing.T) {
		parser := NewParser("test_parser")
		parser.Configure(map[string]interface{}{
			"patterns": []interface{}{
				`(?P<level>INFO|ERROR|WARN) - (?P<message>.*)`,
				`\[(?P<timestamp>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z)\] (?P<level>\w+): (?P<message>.*)`,
			},
		})

		result := parser.Initialize()
		assert.True(t, result)
		assert.Equal(t, 2, len(parser.patterns))
		assert.Equal(t, model.StatusInitialized, parser.GetStatus())
	})

	t.Run("Returns false when no patterns are provided", func(t *testing.T) {
		parser := NewParser("test_parser")
		parser.Configure(map[string]interface{}{
			"patterns": []interface{}{},
		})

		result := parser.Initialize()
		assert.False(t, result)
	})
}

func TestParserValidate(t *testing.T) {
	t.Run("Validates with patterns configured", func(t *testing.T) {
		parser := NewParser("test_parser")
		parser.Configure(map[string]interface{}{
			"patterns": []interface{}{
				`(?P<level>INFO|ERROR|WARN) - (?P<message>.*)`,
			},
		})

		result := parser.Validate()
		assert.True(t, result)
	})

	t.Run("Returns false when patterns are missing", func(t *testing.T) {
		parser := NewParser("test_parser")
		parser.Configure(map[string]interface{}{})

		result := parser.Validate()
		assert.False(t, result)
	})
}

func TestParserProcess(t *testing.T) {
	parser := NewParser("test_parser")
	parser.Configure(map[string]interface{}{
		"patterns": []interface{}{
			`(?P<level>INFO|ERROR|WARN) - (?P<message>.*)`,
			`\[(?P<timestamp>[\d-]+T[\d:]+Z)\] (?P<level>\w+): (?P<message>.*)`,
		},
	})
	parser.Initialize()
	parser.Start()

	t.Run("Processes log points with pattern matches", func(t *testing.T) {
		// Create a batch with log points
		batch := model.NewDataBatch(model.LogTelemetryType)
		
		// Log point matching first pattern
		logPoint1 := &model.LogPoint{
			BaseDataPoint: model.BaseDataPoint{
				Timestamp: time.Now(),
			},
			Message:    "INFO - User logged in successfully",
			Level:      "",
			Attributes: map[string]interface{}{"source": "auth"},
		}
		batch.AddPoint(logPoint1)
		
		// Log point matching second pattern
		timestamp := "2023-01-01T12:00:00Z"
		ts, _ := time.Parse(time.RFC3339, timestamp)
		logPoint2 := &model.LogPoint{
			BaseDataPoint: model.BaseDataPoint{
				Timestamp: time.Now(),
			},
			Message:    "[2023-01-01T12:00:00Z] ERROR: Database connection failed",
			Level:      "",
			Attributes: map[string]interface{}{"source": "db"},
		}
		batch.AddPoint(logPoint2)

		// Process the batch
		resultBatch := parser.Process(batch)
		
		// Verify results
		assert.Equal(t, 2, resultBatch.Size())
		
		// Check first processed point
		point1, ok := resultBatch.Points[0].(*model.LogPoint)
		assert.True(t, ok)
		assert.Equal(t, "INFO", point1.Level)
		assert.Equal(t, "User logged in successfully", point1.Attributes["message"])
		assert.Equal(t, "auth", point1.Attributes["source"])
		
		// Check second processed point
		point2, ok := resultBatch.Points[1].(*model.LogPoint)
		assert.True(t, ok)
		assert.Equal(t, "ERROR", point2.Level)
		assert.Equal(t, "Database connection failed", point2.Attributes["message"])
		assert.Equal(t, ts, point2.Timestamp)
		assert.Equal(t, "db", point2.Attributes["source"])
	})

	t.Run("Handles empty batch gracefully", func(t *testing.T) {
		emptyBatch := model.NewDataBatch(model.LogTelemetryType)
		resultBatch := parser.Process(emptyBatch)
		assert.Equal(t, 0, resultBatch.Size())
	})

	t.Run("Returns batch as-is when stopped", func(t *testing.T) {
		parser.Stop()
		
		batch := model.NewDataBatch(model.LogTelemetryType)
		logPoint := &model.LogPoint{
			Message: "INFO - Test message",
		}
		batch.AddPoint(logPoint)
		
		resultBatch := parser.Process(batch)
		assert.Equal(t, batch, resultBatch)
	})
}