# Observability Collector

A modular, extensible system designed to collect, process, and export telemetry data (logs, metrics, and traces) from various sources. This is a Go implementation of the observability collector architecture described in the [architecture documentation](GO.md).

## Features

- Component-based design with loosely coupled components
- Plugin architecture for extending functionality
- Pipeline processing for flexible data transformations
- Event-driven communication between components
- Buffering and backpressure mechanisms for handling varying data volumes

## Getting Started

### Prerequisites

- Go 1.21 or later

### Building

```bash
# Clone the repository
git clone https://github.com/sliink/collector.git
cd collector

# Build the collector
go build -o collector ./cmd/collector
```

### Running

```bash
# Run with default configuration
./collector

# Run with a specific configuration file
./collector --config config/default.json

# Process a specific input file and output to stdout
./collector --input-file /path/to/logfile.log --stdout

# Use Docker Compose configuration for collecting logs from Docker services
./collector --config config/docker-compose.json

# Output in JSON format
./collector --stdout --json

# Colorize stdout output
./collector --stdout --color

# Process files once and exit
./collector --one-shot
```

## Architecture

The collector uses a modular architecture with several key components:

- **Core System**: Coordinates all components and manages lifecycle
- **Plugin System**: Input, processor, and output plugins for extensibility
- **Data Pipeline**: Manages the flow of data between components
- **Event System**: Enables loosely coupled communication
- **Buffer Management**: Handles data buffering and backpressure

See the [architecture documentation](GO.md) for a detailed explanation of the system.

## Extending the Collector

### Creating a Custom Input Plugin

To create a new input plugin, implement the `InputPlugin` interface:

```go
// MyCustomInput is an example of a custom input plugin
type MyCustomInput struct {
    plugin.BasePlugin
    // Add your custom fields here
}

func NewMyCustomInput(id string) *MyCustomInput {
    return &MyCustomInput{
        BasePlugin: plugin.NewBasePlugin(id, "My Custom Input", model.InputPluginType),
    }
}

// Collect gathers data from the source
func (m *MyCustomInput) Collect() []*model.DataBatch {
    // Implement your custom collection logic
    batch := model.NewDataBatch(model.LogTelemetryType)
    // Add data points to the batch
    return []*model.DataBatch{batch}
}

// Initialize prepares the plugin for operation
func (m *MyCustomInput) Initialize() bool {
    // Initialize your plugin
    m.SetStatus(model.StatusInitialized)
    return true
}

// Other required method implementations...
```

### Registering Your Custom Plugin

Register your custom plugin with the plugin factory and core system:

```go
factory := plugin.NewPluginFactory()
factory.RegisterInputPlugin("my_custom", func(id string) plugin.InputPlugin {
    return NewMyCustomInput(id)
})

// Create the plugin
myPlugin, _ := factory.CreatePlugin(model.InputPluginType, "my_custom", "my_input")

// Register with core
core.RegisterPlugin(myPlugin)
```

## Configuration

The collector uses a JSON configuration file with the following structure:

```json
{
  "system": {
    "id": "observability-collector",
    "version": "1.0.0",
    "log_level": "INFO"
  },
  "plugins": {
    "inputs": [...],
    "processors": [...],
    "outputs": [...]
  },
  "pipelines": {
    "logs": {
      "inputs": ["file_input"],
      "processors": ["log_parser"],
      "outputs": ["stdout_output"]
    }
  }
}
```

### Docker Compose Input Plugin

The Docker Compose input plugin collects logs from Docker Compose services:

```json
{
  "id": "docker_compose_input",
  "type": "docker_compose",
  "config": {
    "project_name": "myproject",
    "services": [
      "app",
      "database",
      "cache"
    ],
    "follow": true,
    "tail": "100",
    "timestamps": true,
    "compose_files": [
      "./docker-compose.yml",
      "./docker-compose.override.yml"
    ],
    "refresh_interval": "1m"
  }
}
```

Configuration options:

- `project_name`: Docker Compose project name
- `services`: List of services to collect logs from (empty for all services)
- `follow`: Whether to follow logs continuously (default: true)
- `tail`: Number of lines to show from the end of logs (default: "100")
- `timestamps`: Whether to include timestamps (default: true)
- `compose_files`: Specific compose files to use
- `refresh_interval`: How often to refresh container mappings (default: "1m")
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.