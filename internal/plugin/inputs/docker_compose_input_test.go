package inputs

import (
	"testing"

	"github.com/sliink/collector/internal/model"
)

func TestNewDockerComposeInput(t *testing.T) {
	input := NewDockerComposeInput("test_docker_compose")
	
	if input == nil {
		t.Fatal("Expected non-nil input")
	}
	
	if input.ID() != "test_docker_compose" {
		t.Errorf("Expected ID to be test_docker_compose, got %s", input.ID())
	}
	
	if input.GetType() != model.InputPluginType {
		t.Errorf("Expected type to be InputPluginType, got %v", input.GetType())
	}
}

func TestDockerComposeInput_Initialize(t *testing.T) {
	input := NewDockerComposeInput("test_docker_compose")
	
	// Test with empty config
	config := map[string]interface{}{}
	input.Configure(config)
	
	result := input.Initialize()
	if !result {
		t.Error("Initialize should return true even with empty config")
	}
	
	if input.GetStatus() != model.StatusInitialized {
		t.Errorf("Expected status to be StatusInitialized, got %v", input.GetStatus())
	}
	
	// Test with config
	input = NewDockerComposeInput("test_docker_compose")
	config = map[string]interface{}{
		"project_name": "test_project",
		"services": []interface{}{"service1", "service2"},
		"follow": false,
		"tail": "50",
		"timestamps": true,
		"compose_files": []interface{}{"docker-compose.yml"},
		"refresh_interval": "30s",
	}
	input.Configure(config)
	
	result = input.Initialize()
	if !result {
		t.Error("Initialize should return true with valid config")
	}
	
	if input.projectName != "test_project" {
		t.Errorf("Expected projectName to be test_project, got %s", input.projectName)
	}
	
	if len(input.services) != 2 || input.services[0] != "service1" || input.services[1] != "service2" {
		t.Errorf("Expected services to be [service1, service2], got %v", input.services)
	}
	
	if input.follow != false {
		t.Error("Expected follow to be false")
	}
	
	if input.tailLines != "50" {
		t.Errorf("Expected tailLines to be 50, got %s", input.tailLines)
	}
}

func TestDockerComposeInput_Validate(t *testing.T) {
	input := NewDockerComposeInput("test_docker_compose")
	
	// Test with no project name or compose files
	config := map[string]interface{}{}
	input.Configure(config)
	input.Initialize()
	
	if input.Validate() {
		t.Error("Validate should return false with no project name or compose files")
	}
	
	// Test with project name
	input = NewDockerComposeInput("test_docker_compose")
	config = map[string]interface{}{
		"project_name": "test_project",
	}
	input.Configure(config)
	input.Initialize()
	
	if !input.Validate() {
		t.Error("Validate should return true with project name")
	}
	
	// Test with compose files
	input = NewDockerComposeInput("test_docker_compose")
	config = map[string]interface{}{
		"compose_files": []interface{}{"docker-compose.yml"},
	}
	input.Configure(config)
	input.Initialize()
	
	if !input.Validate() {
		t.Error("Validate should return true with compose files")
	}
}

func TestDockerComposeInput_Lifecycle(t *testing.T) {
	input := NewDockerComposeInput("test_docker_compose")
	
	// Configure and initialize
	config := map[string]interface{}{
		"project_name": "test_project",
	}
	input.Configure(config)
	input.Initialize()
	
	// Start
	result := input.Start()
	if !result {
		t.Error("Start should return true")
	}
	
	if input.GetStatus() != model.StatusRunning {
		t.Errorf("Expected status to be StatusRunning, got %v", input.GetStatus())
	}
	
	// Collect (not testing actual collection, just checking it doesn't crash)
	batches := input.Collect()
	if batches != nil && len(batches) > 0 {
		// This would only happen if docker-compose was actually running with matching services
		t.Logf("Found %d batches", len(batches))
	}
	
	// Stop
	result = input.Stop()
	if !result {
		t.Error("Stop should return true")
	}
	
	if input.GetStatus() != model.StatusStopped {
		t.Errorf("Expected status to be StatusStopped, got %v", input.GetStatus())
	}
}