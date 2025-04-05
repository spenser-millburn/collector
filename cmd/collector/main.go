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
	if !c.Start() {
		fmt.Println("Failed to start core system")
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
			// Configure each pipeline
			for pipelineType, pipelineConfig := range pipelines {
				config, ok := pipelineConfig.(map[string]interface{})
				if !ok {
					continue
				}
				
				// Get processors for this pipeline
				var processorIDs []string
				if processors, ok := config["processors"].([]interface{}); ok {
					for _, processorID := range processors {
						if id, ok := processorID.(string); ok {
							processorIDs = append(processorIDs, id)
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
				}
			}
			
			return nil
		}
	}
	
	// If no pipeline configuration was found, configure a simple log pipeline
	err := pipeline.CreatePipeline(model.LogTelemetryType, []string{"log_parser"})
	if err != nil {
		return fmt.Errorf("failed to create log pipeline: %w", err)
	}
	
	return nil
}

func registerPlugins(c *core.Core) error {
	// Create input plugins
	fileInput := inputs.NewFileInput("file_input")
	
	// Configure file input
	fileInputConfig := map[string]interface{}{
		"paths": []interface{}{},
	}
	
	if inputFile != "" {
		fileInputConfig["paths"] = []interface{}{inputFile}
	} else {
		fileInputConfig["paths"] = []interface{}{"/var/log/*.log"}
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
	}
	dockerComposeInput.Configure(dockerComposeConfig)
	
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
	
	stdoutOutput.Configure(stdoutOutputConfig)
	
	// Register plugins with core
	if err := c.RegisterPlugin(fileInput); err != nil {
		return err
	}
	
	if err := c.RegisterPlugin(dockerComposeInput); err != nil {
		return err
	}
	
	if err := c.RegisterPlugin(parser); err != nil {
		return err
	}
	
	if err := c.RegisterPlugin(stdoutOutput); err != nil {
		return err
	}
	
	// Configure pipeline
	if err := configurePipeline(c); err != nil {
		return err
	}
	
	return nil
}