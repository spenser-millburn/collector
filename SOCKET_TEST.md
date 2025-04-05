# Socket Input Test

This document explains how to run the socket-to-stdout pipeline test. This test demonstrates how to use the socket input plugin to receive data from a network socket and process it through the collector pipeline.

## Overview

The test consists of:

1. A socket input plugin that listens on a TCP socket and receives data
2. A Python client that sends log messages to the socket
3. A processing pipeline that parses and processes the received data
4. A stdout output that displays the processed data

## Requirements

- Go 1.21 or higher
- Python 3.6 or higher

## Quick Start

To run the test with default settings:

```bash
./run_socket_pipeline.sh
```

This will:
- Build the collector if needed
- Start the collector using the dedicated socket pipeline configuration (`config/socket_pipeline.json`)
- Start the Python client to send test messages
- Clean up when you press Ctrl+C

The configuration file sets up:
- A socket input plugin listening on port 8888
- A log parser that extracts timestamps and message content
- A stdout output that displays the processed logs

## Custom Options

The Python client supports several command-line options:

```bash
./run_socket_pipeline.sh --interval 0.5 --format json "Custom log message"
```

Available options:

- `--host`: Server hostname (default: localhost)
- `--port`: Server port (default: 8888)
- `--interval`: Interval between messages in seconds (default: 1.0)
- `--format`: Message format (choices: text, json, default: text)
- `message`: Message text to send (default: "Test message from socket client")

## Manual Setup

If you prefer to run the components individually:

1. Build the collector:
   ```bash
   go build -o build/collector cmd/collector/main.go
   ```

2. Start the collector with stdout output:
   ```bash
   ./build/collector --stdout --color
   ```

3. In another terminal, run the Python client:
   ```bash
   python3 socket_client.py [options] [message]
   ```

## How It Works

1. The collector starts and initializes the socket input plugin, which begins listening on port 8888
2. The Python client connects to the socket and sends log messages
3. The socket input plugin receives these messages and adds them to the processing pipeline
4. The log parser processes the messages (extracts timestamp, level, etc.)
5. The stdout output plugin formats and displays the processed messages

## Verifying It Works

When the pipeline is functioning correctly, you should see:

1. **Connection confirmation**: The Python client will show a message like:
   ```
   Connected to localhost:8888
   ```

2. **Message sending indicators**: For each message sent, the client will print:
   ```
   Sent: 2023-08-15 14:30:45 - Test message - #1
   ```

3. **Collector output**: The collector will show the processed message in its output:
   ```
   [INFO] 2023-08-15 14:30:45 - Test message - #1
   ```

The number of messages should match between the client and collector. The messages in the collector output should be formatted according to the configured parser and output formatting.

## Troubleshooting

- If the connection is refused, ensure the collector is running and the port is not in use
- If no data appears in the output, check that the Python client is correctly sending data
- If the processed data looks incorrect, check the parser patterns in the configuration file
- To see raw data without processing, try using the JSON format option: `--format json`
- For debugging, you can directly run the collector and watch its output while sending data

### Debugging File Inputs

If you notice the collector is still processing files even with the socket configuration:

1. The configuration explicitly disables file processing by setting:
   ```json
   "file_input": {
     "enabled": false,
     "paths": []
   }
   ```

2. Check if the file input is being forced to run by command-line flags
   - The `--input-file` flag will override settings and enable file input
   - Run without this flag to ensure only socket input is active

3. For verbose debug output, run:
   ```bash
   ./build/collector --config config/socket_pipeline.json --log-level=debug
   ```

4. To confirm only socket input is active, check the API's plugin status:
   ```bash
   curl http://localhost:8080/plugins
   ```

5. If problems persist, refer to the collector's core implementation in internal/core/core.go

## Extending

To extend this test:

1. Modify the socket_client.py script to send different types of data
2. Update the parser patterns in cmd/collector/main.go to handle your data format
3. Add additional processors to the pipeline for more complex data transformations

## API Integration

The socket input plugin can also be controlled via the API:

1. Start the collector with the API enabled (default):
   ```bash
   ./build/collector
   ```

2. Get the status of the socket input plugin:
   ```bash
   curl http://localhost:8080/plugins/INPUT/socket_input
   ```

3. Create a new pipeline with the socket input:
   ```bash
   curl -X POST http://localhost:8080/pipelines -d '{"type": "logs", "processors": ["log_parser"]}'
   ```