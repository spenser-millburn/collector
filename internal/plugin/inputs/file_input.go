package inputs

import (
	"bufio"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sliink/collector/internal/model"
	"github.com/sliink/collector/internal/plugin"
)

// FileInput reads log data from files
type FileInput struct {
	plugin.BasePlugin
	paths           []string
	filePositions   map[string]int64
	multilineConfig map[string]interface{}
	mutex           sync.RWMutex
}

// NewFileInput creates a new file input plugin
func NewFileInput(id string) *FileInput {
	return &FileInput{
		BasePlugin:      plugin.NewBasePlugin(id, "File Input", model.InputPluginType),
		filePositions:   make(map[string]int64),
		multilineConfig: make(map[string]interface{}),
	}
}

// Initialize prepares the file input for operation
func (f *FileInput) Initialize() bool {
	// Get paths from configuration
	if paths, ok := f.Config["paths"].([]interface{}); ok {
		for _, p := range paths {
			if path, ok := p.(string); ok {
				f.paths = append(f.paths, path)
			}
		}
	}

	// Get multiline configuration if any
	if multiline, ok := f.Config["multiline"].(map[string]interface{}); ok {
		f.multilineConfig = multiline
	}

	f.SetStatus(model.StatusInitialized)
	return len(f.paths) > 0
}

// Start begins file input operation
func (f *FileInput) Start() bool {
	f.SetStatus(model.StatusRunning)
	return true
}

// Stop halts file input operation
func (f *FileInput) Stop() bool {
	f.SetStatus(model.StatusStopped)
	return true
}

// Validate checks if the file input is properly configured
func (f *FileInput) Validate() bool {
	// Check if paths are configured
	if paths, ok := f.Config["paths"].([]interface{}); !ok || len(paths) == 0 {
		return false
	}

	return true
}

// Collect gathers log data from files
func (f *FileInput) Collect() []*model.DataBatch {
	if f.GetStatus() != model.StatusRunning {
		return nil
	}

	var results []*model.DataBatch
	batch := model.NewDataBatch(model.LogTelemetryType)

	// Process each configured path
	for _, pathPattern := range f.paths {
		// Expand glob patterns
		matches, err := filepath.Glob(pathPattern)
		if err != nil {
			continue
		}

		for _, path := range matches {
			// Process each file
			logPoints := f.processFile(path)
			
			// Add log points to batch
			for _, point := range logPoints {
				batch.AddPoint(point)
				
				// Create a new batch if current one is full
				if batch.Size() >= 1000 { // Configurable batch size
					results = append(results, batch)
					batch = model.NewDataBatch(model.LogTelemetryType)
				}
			}
		}
	}

	// Add the last batch if it has any points
	if batch.Size() > 0 {
		results = append(results, batch)
	}

	return results
}

// processFile reads a file and creates log points
func (f *FileInput) processFile(path string) []*model.LogPoint {
	f.mutex.Lock()
	lastPosition, exists := f.filePositions[path]
	f.mutex.Unlock()

	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	// Seek to last position if known
	if exists {
		_, err = file.Seek(lastPosition, 0)
		if err != nil {
			return nil
		}
	}

	var logPoints []*model.LogPoint
	scanner := bufio.NewScanner(file)
	
	for scanner.Scan() {
		line := scanner.Text()
		
		// Create a log point
		logPoint := &model.LogPoint{
			BaseDataPoint: model.BaseDataPoint{
				Timestamp: time.Now(),
				Origin:    path,
				Labels: map[string]string{
					"source": "file",
					"path":   path,
				},
			},
			Message:    line,
			Level:      "INFO", // Default level, would be parsed from content
			Attributes: map[string]interface{}{},
		}
		
		logPoints = append(logPoints, logPoint)
	}

	// Update file position
	currentPos, _ := file.Seek(0, 1) // Get current position
	
	f.mutex.Lock()
	f.filePositions[path] = currentPos
	f.mutex.Unlock()

	return logPoints
}