package inputs

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/sliink/collector/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestNewFileInput(t *testing.T) {
	t.Run("Creates FileInput with correct properties", func(t *testing.T) {
		input := NewFileInput("file_input")
		
		assert.Equal(t, "file_input", input.ID())
		assert.Equal(t, "File Input", input.Name())
		assert.Equal(t, model.InputPluginType, input.GetType())
		assert.Equal(t, model.StatusUninitialized, input.GetStatus())
		assert.NotNil(t, input.filePositions)
		assert.NotNil(t, input.multilineConfig)
	})
}

func TestFileInputLifecycle(t *testing.T) {
	input := NewFileInput("file_input")
	
	t.Run("Initialize fails when no paths configured", func(t *testing.T) {
		success := input.Initialize()
		assert.False(t, success)
	})
	
	t.Run("Initialize succeeds with paths configured", func(t *testing.T) {
		input.Config = map[string]interface{}{
			"paths": []interface{}{
				"/var/log/test.log",
				"/var/log/app/*.log",
			},
		}
		
		success := input.Initialize()
		assert.True(t, success)
		assert.Equal(t, model.StatusInitialized, input.GetStatus())
		assert.Len(t, input.paths, 2)
	})
	
	t.Run("Initialize with multiline config", func(t *testing.T) {
		input := NewFileInput("file_input")
		input.Config = map[string]interface{}{
			"paths": []interface{}{"/var/log/test.log"},
			"multiline": map[string]interface{}{
				"pattern": "^[0-9]",
				"negate":  true,
			},
		}
		
		success := input.Initialize()
		assert.True(t, success)
		assert.NotEmpty(t, input.multilineConfig)
	})
	
	t.Run("Start sets correct status", func(t *testing.T) {
		success := input.Start()
		assert.True(t, success)
		assert.Equal(t, model.StatusRunning, input.GetStatus())
	})
	
	t.Run("Stop sets correct status", func(t *testing.T) {
		success := input.Stop()
		assert.True(t, success)
		assert.Equal(t, model.StatusStopped, input.GetStatus())
	})
}

func TestFileInputValidate(t *testing.T) {
	input := NewFileInput("file_input")
	
	t.Run("Validate fails with no paths", func(t *testing.T) {
		success := input.Validate()
		assert.False(t, success)
	})
	
	t.Run("Validate succeeds with paths", func(t *testing.T) {
		input.Config = map[string]interface{}{
			"paths": []interface{}{"/var/log/test.log"},
		}
		
		success := input.Validate()
		assert.True(t, success)
	})
	
	t.Run("Validate fails with empty paths array", func(t *testing.T) {
		input.Config = map[string]interface{}{
			"paths": []interface{}{},
		}
		
		success := input.Validate()
		assert.False(t, success)
	})
}

func TestFileInputCollect(t *testing.T) {
	t.Run("Collect returns nil when not running", func(t *testing.T) {
		input := NewFileInput("file_input")
		input.Config = map[string]interface{}{
			"paths": []interface{}{"/var/log/test.log"},
		}
		input.Initialize()
		
		// Not started, status is still Initialized
		batches := input.Collect()
		assert.Nil(t, batches)
	})
	
	t.Run("Collect returns data from test file", func(t *testing.T) {
		// Create a temporary test file
		tempDir, err := os.MkdirTemp("", "file_input_test")
		assert.NoError(t, err)
		defer os.RemoveAll(tempDir)
		
		testFilePath := filepath.Join(tempDir, "test.log")
		testFileContent := "line1\nline2\nline3\n"
		
		err = os.WriteFile(testFilePath, []byte(testFileContent), 0644)
		assert.NoError(t, err)
		
		// Create the input plugin
		input := NewFileInput("file_input")
		input.Config = map[string]interface{}{
			"paths": []interface{}{filepath.Join(tempDir, "*.log")},
		}
		input.Initialize()
		input.Start()
		
		// Collect data
		batches := input.Collect()
		
		// Verify results
		assert.NotNil(t, batches)
		assert.Len(t, batches, 1) // Should be one batch
		assert.Equal(t, model.LogTelemetryType, batches[0].BatchType)
		assert.Equal(t, 3, batches[0].Size())
		
		// Check log content
		points := batches[0].Points
		assert.Equal(t, "line1", points[0].(*model.LogPoint).Message)
		assert.Equal(t, "line2", points[1].(*model.LogPoint).Message)
		assert.Equal(t, "line3", points[2].(*model.LogPoint).Message)
	})
	
	t.Run("Collect handles large files with batching", func(t *testing.T) {
		// Create a temporary test file with many lines
		tempDir, err := os.MkdirTemp("", "file_input_test_large")
		assert.NoError(t, err)
		defer os.RemoveAll(tempDir)
		
		testFilePath := filepath.Join(tempDir, "large.log")
		
		// Create file with 2500 lines (should create multiple batches)
		f, err := os.Create(testFilePath)
		assert.NoError(t, err)
		
		for i := 0; i < 2500; i++ {
			_, err := f.WriteString(fmt.Sprintf("line%d\n", i))
			assert.NoError(t, err)
		}
		f.Close()
		
		// Create the input plugin
		input := NewFileInput("file_input")
		input.Config = map[string]interface{}{
			"paths": []interface{}{testFilePath},
		}
		input.Initialize()
		input.Start()
		
		// Collect data
		batches := input.Collect()
		
		// Verify results - should have multiple batches
		assert.NotNil(t, batches)
		assert.GreaterOrEqual(t, len(batches), 2) // At least 2 batches (with default 1000 batch size)
		
		// Count total points
		totalPoints := 0
		for _, batch := range batches {
			totalPoints += batch.Size()
		}
		assert.Equal(t, 2500, totalPoints)
	})
	
	t.Run("Collect only reads new lines on subsequent calls", func(t *testing.T) {
		// Create a temporary test file
		tempDir, err := os.MkdirTemp("", "file_input_test_append")
		assert.NoError(t, err)
		defer os.RemoveAll(tempDir)
		
		testFilePath := filepath.Join(tempDir, "append.log")
		
		// Write initial content
		err = os.WriteFile(testFilePath, []byte("line1\nline2\n"), 0644)
		assert.NoError(t, err)
		
		// Create the input plugin
		input := NewFileInput("file_input")
		input.Config = map[string]interface{}{
			"paths": []interface{}{testFilePath},
		}
		input.Initialize()
		input.Start()
		
		// First collection
		batches := input.Collect()
		assert.Len(t, batches, 1)
		assert.Equal(t, 2, batches[0].Size())
		
		// Append more content
		f, err := os.OpenFile(testFilePath, os.O_APPEND|os.O_WRONLY, 0644)
		assert.NoError(t, err)
		_, err = f.WriteString("line3\nline4\n")
		assert.NoError(t, err)
		f.Close()
		
		// Second collection should only get new lines
		batches = input.Collect()
		assert.Len(t, batches, 1)
		assert.Equal(t, 2, batches[0].Size())
		
		// Check content of new batch
		points := batches[0].Points
		assert.Equal(t, "line3", points[0].(*model.LogPoint).Message)
		assert.Equal(t, "line4", points[1].(*model.LogPoint).Message)
	})
}