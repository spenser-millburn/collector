package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sliink/collector/internal/api"
	"github.com/sliink/collector/internal/core"
	"github.com/sliink/collector/internal/model"
	"github.com/sliink/collector/internal/plugin/inputs"
	"github.com/sliink/collector/internal/plugin/outputs"
	"github.com/sliink/collector/internal/plugin/processors"
	"github.com/spf13/cobra"
)

var (
	configFile string
	inputFile  string
	outputDir  string
	stdout     bool
	colorize   bool
	jsonFormat bool
	oneShot    bool
	apiEnabled bool
	apiPort    int
	apiHost    string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "collector",
		Short: "Observability Collector - Collect, process, and export telemetry data",
		Run:   runCollector,
	}

	// Common flags
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "Path to configuration file")
	rootCmd.PersistentFlags().StringVar(&inputFile, "input-file", "", "Process a specific input file")
	rootCmd.PersistentFlags().StringVar(&outputDir, "output-dir", "", "Output directory for file output")
	rootCmd.PersistentFlags().BoolVar(&stdout, "stdout", false, "Output to stdout instead of files")
	rootCmd.PersistentFlags().BoolVar(&colorize, "color", false, "Colorize stdout output")
	rootCmd.PersistentFlags().BoolVar(&jsonFormat, "json", false, "Output in JSON format")
	rootCmd.PersistentFlags().BoolVar(&oneShot, "one-shot", false, "Process files once and exit")
	
	// API server flags
	rootCmd.PersistentFlags().BoolVar(&apiEnabled, "api", true, "Enable the API server")
	rootCmd.PersistentFlags().IntVar(&apiPort, "api-port", 8080, "API server port")
	rootCmd.PersistentFlags().StringVar(&apiHost, "api-host", "localhost", "API server host")
	
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func runCollector(cmd *cobra.Command, args []string) {
	fmt.Println("Starting Observability Collector...")

	// Create the core system
	c := core.NewCore()

	// Initialize the core system
	if !c.Initialize() {
		fmt.Println("Failed to initialize core system")
		os.Exit(1)
	}
	
	// Load configuration if provided
	if configFile != "" {
		configManager := c.GetConfigManager()
		if configManager != nil {
			if err := configManager.LoadConfig(configFile); err != nil {
				fmt.Println("Failed to load configuration:", err)
				os.Exit(1)
			}
			fmt.Println("Loaded configuration from", configFile)
		}
	}

	// Create and register plugins
	if err := registerPlugins(c); err != nil {
		fmt.Println("Failed to register plugins:", err)
		os.Exit(1)
	}

	// Start the core system
	fmt.Println("Starting core system...")
	
	// Get all registered plugins for debug
	fmt.Println("Registered plugins:")
	if registry, exists := c.GetComponent("plugin_registry"); exists {
		if pluginRegistry, ok := registry.(*core.PluginRegistry); ok {
			for _, plugin := range pluginRegistry.GetAllPlugins() {
				fmt.Printf("- %s (type: %s, status: %s)\n", plugin.ID(), plugin.GetType(), plugin.GetStatus())
			}
		}
	}
	
	if !c.Start() {
		fmt.Println("Failed to start core system")
		
		// Try to get status of all registered plugins
		for _, pluginType := range []string{"input", "processor", "output"} {
			fmt.Printf("Checking %s plugins status:\n", pluginType)
			
			// Get all plugins of this type and check status
			if registry, exists := c.GetComponent("plugin_registry"); exists {
				if pluginRegistry, ok := registry.(*core.PluginRegistry); ok {
					var plugins []model.Plugin
					
					switch pluginType {
					case "input":
						for _, p := range pluginRegistry.GetInputPlugins() {
							plugins = append(plugins, p)
						}
					case "processor":
						for _, p := range pluginRegistry.GetProcessorPlugins() {
							plugins = append(plugins, p)
						}
					case "output":
						for _, p := range pluginRegistry.GetOutputPlugins() {
							plugins = append(plugins, p)
						}
					}
					
					if len(plugins) == 0 {
						fmt.Printf("  No %s plugins registered\n", pluginType)
						continue
					}
					
					for _, plugin := range plugins {
						fmt.Printf("  - %s status: %s\n", plugin.ID(), plugin.GetStatus())
					}
				}
			}
		}
		
		// Check core components status
		fmt.Println("Core components status:")
		components := []string{"event_bus", "plugin_registry", "data_pipeline", "buffer_manager", "config_manager", "health_monitor", "core"}
		for _, compID := range components {
			if comp, exists := c.GetComponent(compID); exists {
				fmt.Printf("  - %s: %s\n", compID, comp.GetStatus())
			}
		}
		
		os.Exit(1)
	}

	fmt.Println("Collector is running. Press Ctrl+C to stop.")
	
	// Start API server if enabled
	var apiServer *api.API
	if apiEnabled {
		// Initialize API with the core instance
		apiServer = api.NewAPI(c, apiPort, apiHost)
		
		// Start the API server in a goroutine
		go func() {
			fmt.Printf("Starting API server at %s:%d\n", apiHost, apiPort)
			if err := apiServer.Start(); err != nil && err != http.ErrServerClosed {
				fmt.Printf("API server error: %s\n", err)
			}
		}()
	}

	// Setup signal handling for graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal
	<-sigs
	
	// Shutdown API server if it was started
	if apiServer != nil {
		fmt.Println("Shutting down API server...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := apiServer.Stop(ctx); err != nil {
			fmt.Printf("API server shutdown error: %s\n", err)
		}
	}

	fmt.Println("\nShutting down...")

	// Stop the core system
	if !c.Stop() {
		fmt.Println("Failed to stop core system cleanly")
		os.Exit(1)
	}

	fmt.Println("Shutdown complete")
}

func configurePipeline(c *core.Core) error {
	// Convert Core to the appropriate interface for the pipeline
	// Get the data pipeline component
	pipeline := c.GetDataPipeline()
	if pipeline == nil {
		return fmt.Errorf("failed to get data pipeline component")
	}
	
	// Check if we have a config file with pipeline definitions
	configManager := c.GetConfigManager()
	
	if configManager != nil {
		pipelines, ok := configManager.GetConfig("pipelines", nil).(map[string]interface{})
		if ok {
			fmt.Println("Found pipeline configuration in config file")
			
			// Configure each pipeline
			for pipelineType, pipelineConfig := range pipelines {
				config, ok := pipelineConfig.(map[string]interface{})
				if !ok {
					continue
				}
				
				fmt.Printf("Configuring pipeline type: %s\n", pipelineType)
				
				// Get processors for this pipeline
				var processorIDs []string
				if processors, ok := config["processors"].([]interface{}); ok {
					for _, processorID := range processors {
						if id, ok := processorID.(string); ok {
							processorIDs = append(processorIDs, id)
							fmt.Printf("  Adding processor: %s\n", id)
						}
					}
				}
				
				// Create the pipeline
				var telemetryType model.TelemetryType
				switch pipelineType {
				case "logs":
					telemetryType = model.LogTelemetryType
				case "metrics":
					telemetryType = model.MetricTelemetryType
				case "traces":
					telemetryType = model.TraceTelemetryType
				default:
					continue
				}
				
				if len(processorIDs) > 0 {
					if err := pipeline.CreatePipeline(telemetryType, processorIDs); err != nil {
						return fmt.Errorf("failed to create %s pipeline: %w", pipelineType, err)
					}
					fmt.Printf("Created %s pipeline with %d processors\n", pipelineType, len(processorIDs))
				} else {
					fmt.Printf("No processors specified for %s pipeline\n", pipelineType)
				}
			}
			
			return nil
		} else {
			fmt.Println("No pipeline configuration found in config file, using default")
		}
	}
	
	// If no pipeline configuration was found, configure a simple log pipeline
	fmt.Println("Creating default log pipeline with log_parser")
	err := pipeline.CreatePipeline(model.LogTelemetryType, []string{"log_parser"})
	if err != nil {
		return fmt.Errorf("failed to create log pipeline: %w", err)
	}
	
	return nil
}

func registerPlugins(c *core.Core) error {
	configManager := c.GetConfigManager()
	
	// Create input plugins
	fileInput := inputs.NewFileInput("file_input")
	
	// Configure file input
	fileInputConfig := map[string]interface{}{
		"paths": []interface{}{},
		"enabled": inputFile != "",  // Only enable if input file is provided
	}
	
	// Override with config file if available
	if configFile != "" && configManager != nil {
		if fileConfig, ok := configManager.GetConfig("file_input", nil).(map[string]interface{}); ok {
			// Merge config
			for k, v := range fileConfig {
				fileInputConfig[k] = v
			}
			fmt.Println("Using file_input configuration from config file")
		}
	} else if inputFile != "" {
		fileInputConfig["paths"] = []interface{}{inputFile}
	}
	
	fileInput.Configure(fileInputConfig)
	
	// Create and configure Docker Compose input plugin
	dockerComposeInput := inputs.NewDockerComposeInput("docker_compose_input")
	dockerComposeConfig := map[string]interface{}{
		"project_name": "",
		"services":     []interface{}{},
		"follow":       true,
		"tail":         "100",
		"timestamps":   true,
		"enabled":      false, // Disabled by default
	}
	
	// Override with config file if available
	if configFile != "" && configManager != nil {
		if dockerConfig, ok := configManager.GetConfig("docker_compose_input", nil).(map[string]interface{}); ok {
			// Merge config
			for k, v := range dockerConfig {
				dockerComposeConfig[k] = v
			}
			fmt.Println("Using docker_compose_input configuration from config file")
		}
	}
	
	dockerComposeInput.Configure(dockerComposeConfig)
	
	// Create and configure Socket input plugin
	socketInput := inputs.NewSocketInput("socket_input")
	socketConfig := map[string]interface{}{
		"protocol": "tcp",
		"address":  "localhost:8888",
		"enabled":  true, // Enabled by default
	}
	
	// Override with config file if available
	if configFile != "" && configManager != nil {
		if socketConfig_, ok := configManager.GetConfig("socket_input", nil).(map[string]interface{}); ok {
			// Merge config
			for k, v := range socketConfig_ {
				socketConfig[k] = v
			}
			fmt.Println("Using socket_input configuration from config file")
		}
	}
	
	socketInput.Configure(socketConfig)
	
	// Create processor plugins
	parser := processors.NewParser("log_parser")
	
	// Configure parser
	parserConfig := map[string]interface{}{
		"patterns": []interface{}{
			// Simple log pattern for common log formats
			`^(?P<timestamp>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.\d+Z) (?P<level>[A-Z]+) (?P<message>.*)$`,
			// Fallback pattern
			`^(?P<message>.*)$`,
		},
	}
	
	// Override with config file if available
	if configFile != "" && configManager != nil {
		if parserConfig_, ok := configManager.GetConfig("log_parser", nil).(map[string]interface{}); ok {
			// Merge config
			for k, v := range parserConfig_ {
				parserConfig[k] = v
			}
			fmt.Println("Using log_parser configuration from config file")
		}
	}
	
	parser.Configure(parserConfig)
	
	// Create output plugins
	// Use stdout output
	stdoutOutput := outputs.NewStdoutOutput("stdout_output")
	
	// Configure stdout output
	stdoutOutputConfig := map[string]interface{}{
		"colorize": colorize,
		"format":   "text",
	}
	
	if jsonFormat {
		stdoutOutputConfig["format"] = "json"
	}
	
	// Override with config file if available
	if configFile != "" && configManager != nil {
		if stdoutConfig, ok := configManager.GetConfig("stdout_output", nil).(map[string]interface{}); ok {
			// Merge config
			for k, v := range stdoutConfig {
				stdoutOutputConfig[k] = v
			}
			fmt.Println("Using stdout_output configuration from config file")
		}
	}
	
	stdoutOutput.Configure(stdoutOutputConfig)
	
	// Register plugins with core
	fmt.Println("Registering plugins with core:")
	
	fmt.Println("- Registering file_input")
	if err := c.RegisterPlugin(fileInput); err != nil {
		return fmt.Errorf("failed to register file_input: %w", err)
	}
	
	fmt.Println("- Registering docker_compose_input")
	if err := c.RegisterPlugin(dockerComposeInput); err != nil {
		return fmt.Errorf("failed to register docker_compose_input: %w", err)
	}
	
	fmt.Println("- Registering socket_input")
	if err := c.RegisterPlugin(socketInput); err != nil {
		return fmt.Errorf("failed to register socket_input: %w", err)
	}
	
	fmt.Println("- Registering log_parser")
	if err := c.RegisterPlugin(parser); err != nil {
		return fmt.Errorf("failed to register log_parser: %w", err)
	}
	
	fmt.Println("- Registering stdout_output")
	if err := c.RegisterPlugin(stdoutOutput); err != nil {
		return fmt.Errorf("failed to register stdout_output: %w", err)
	}
	
	// Configure pipeline
	if err := configurePipeline(c); err != nil {
		return fmt.Errorf("failed to configure pipeline: %w", err)
	}
	
	return nil
}