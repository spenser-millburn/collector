package inputs

import (
	"bufio"
	"bytes"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/sliink/collector/internal/model"
	"github.com/sliink/collector/internal/plugin"
)

// DockerComposeInput reads log data from docker-compose services
type DockerComposeInput struct {
	plugin.BasePlugin
	projectName      string
	services         []string
	follow           bool
	tailLines        string
	timestamps       bool
	composeFiles     []string
	refreshInterval  time.Duration
	containerMapping map[string]string // Map of container ID to service name
	mutex            sync.RWMutex
}

// NewDockerComposeInput creates a new docker-compose input plugin
func NewDockerComposeInput(id string) *DockerComposeInput {
	return &DockerComposeInput{
		BasePlugin:       plugin.NewBasePlugin(id, "Docker Compose Input", model.InputPluginType),
		services:         []string{},
		follow:           true,
		tailLines:        "10",
		timestamps:       true,
		composeFiles:     []string{},
		refreshInterval:  time.Minute,
		containerMapping: make(map[string]string),
	}
}

// Initialize prepares the docker-compose input for operation
func (d *DockerComposeInput) Initialize() bool {
	// Get project name from configuration
	if projectName, ok := d.Config["project_name"].(string); ok {
		d.projectName = projectName
	}

	// Get services from configuration
	if services, ok := d.Config["services"].([]interface{}); ok {
		for _, s := range services {
			if service, ok := s.(string); ok {
				d.services = append(d.services, service)
			}
		}
	}

	// Get follow flag from configuration
	if follow, ok := d.Config["follow"].(bool); ok {
		d.follow = follow
	}

	// Get tail lines from configuration
	if tailLines, ok := d.Config["tail"].(string); ok {
		d.tailLines = tailLines
	}

	// Get timestamps flag from configuration
	if timestamps, ok := d.Config["timestamps"].(bool); ok {
		d.timestamps = timestamps
	}

	// Get compose files from configuration
	if composeFiles, ok := d.Config["compose_files"].([]interface{}); ok {
		for _, cf := range composeFiles {
			if composeFile, ok := cf.(string); ok {
				d.composeFiles = append(d.composeFiles, composeFile)
			}
		}
	}

	// Get refresh interval from configuration
	if refreshStr, ok := d.Config["refresh_interval"].(string); ok {
		if duration, err := time.ParseDuration(refreshStr); err == nil {
			d.refreshInterval = duration
		}
	}

	// Initialize container mapping
	d.refreshContainerMapping()

	d.SetStatus(model.StatusInitialized)
	return true
}

// Start begins docker-compose input operation
func (d *DockerComposeInput) Start() bool {
	d.SetStatus(model.StatusRunning)
	return true
}

// Stop halts docker-compose input operation
func (d *DockerComposeInput) Stop() bool {
	d.SetStatus(model.StatusStopped)
	return true
}

// Validate checks if the docker-compose input is properly configured
func (d *DockerComposeInput) Validate() bool {
	// Since we can run docker-compose logs without any of these parameters,
	// we'll return true by default
	return true
}

// refreshContainerMapping updates the mapping from container IDs to service names
func (d *DockerComposeInput) refreshContainerMapping() {
	var cmd *exec.Cmd
	
	// Build docker-compose command
	args := []string{"ps", "--format", "{{.ID}},{{.Service}}"}

	// Add project name if specified
	if d.projectName != "" {
		args = append([]string{"-p", d.projectName}, args...)
	}

	// Add compose files if specified
	for _, file := range d.composeFiles {
		args = append([]string{"-f", file}, args...)
	}

	// Create the docker-compose command
	cmd = exec.Command("docker-compose", args...)

	// Execute the command
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return
	}

	// Parse the output and update container mapping
	d.mutex.Lock()
	defer d.mutex.Unlock()

	scanner := bufio.NewScanner(&out)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ",")
		if len(parts) == 2 {
			containerID := parts[0]
			serviceName := parts[1]
			d.containerMapping[containerID] = serviceName
		}
	}
}

// Collect gathers log data from docker-compose services
func (d *DockerComposeInput) Collect() []*model.DataBatch {
	if d.GetStatus() != model.StatusRunning {
		return nil
	}

	// Refresh container mapping if enough time has passed
	d.refreshContainerMapping()

	var results []*model.DataBatch
	batch := model.NewDataBatch(model.LogTelemetryType)

	// Build docker-compose logs command
	args := []string{"logs"}

	// Add follow flag if enabled
	if d.follow {
		args = append(args, "--follow")
	}

	// Add tail lines
	args = append(args, "--tail", d.tailLines)

	// Add timestamps flag if enabled
	if d.timestamps {
		args = append(args, "--timestamps")
	}

	// Add project name if specified
	if d.projectName != "" {
		args = append([]string{"-p", d.projectName}, args...)
	}

	// Add compose files if specified
	for _, file := range d.composeFiles {
		args = append([]string{"-f", file}, args...)
	}

	// Add services if specified, otherwise collect from all services
	if len(d.services) > 0 {
		args = append(args, d.services...)
	}

	// Create the docker-compose command
	cmd := exec.Command("docker-compose", args...)

	// Execute the command
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil
	}

	err = cmd.Start()
	if err != nil {
		return nil
	}

	// Process the output
	scanner := bufio.NewScanner(stdout)
	
	for scanner.Scan() {
		line := scanner.Text()
		
		// Parse the log line
		// Format: [service_name] YYYY-MM-DDTHH:MM:SS.sssZ message
		serviceName := ""
		message := line
		timestamp := time.Now()

		// Extract service name if present
		if strings.HasPrefix(line, "|") {
			parts := strings.SplitN(line, "|", 3)
			if len(parts) >= 3 {
				serviceName = strings.TrimSpace(parts[0])
				timestampStr := strings.TrimSpace(parts[1])
				message = strings.TrimSpace(parts[2])

				// Parse timestamp if present
				if t, err := time.Parse(time.RFC3339Nano, timestampStr); err == nil {
					timestamp = t
				}
			}
		}

		// Create a log point
		logPoint := &model.LogPoint{
			BaseDataPoint: model.BaseDataPoint{
				Timestamp: timestamp,
				Origin:    "docker-compose",
				Labels: map[string]string{
					"source":  "docker-compose",
					"service": serviceName,
				},
			},
			Message:    message,
			Level:      "INFO", // Default level, would be parsed from content
			Attributes: map[string]interface{}{},
		}
		
		batch.AddPoint(logPoint)
		
		// Create a new batch if current one is full
		if batch.Size() >= 1000 { // Configurable batch size
			results = append(results, batch)
			batch = model.NewDataBatch(model.LogTelemetryType)
		}

		// If not following logs, limit the number of log lines
		if !d.follow && batch.Size() > 10000 {
			break
		}
	}

	// Add the last batch if it has any points
	if batch.Size() > 0 {
		results = append(results, batch)
	}

	// Kill the process if we're not following logs
	if !d.follow {
		_ = cmd.Process.Kill()
	}

	return results
}