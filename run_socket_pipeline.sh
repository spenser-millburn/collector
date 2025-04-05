#!/bin/bash
# Run a socket-to-stdout pipeline

# Build the collector if needed
if [ ! -f build/collector ] || [ cmd/collector/main.go -nt build/collector ]; then
    echo "Building collector..."
    go build -o build/collector cmd/collector/main.go
    if [ $? -ne 0 ]; then
        echo "Build failed"
        exit 1
    fi
fi

# Start the collector with socket config
echo "Starting collector with socket input and stdout output..."
echo "Using socket_pipeline.json configuration"
./build/collector --config config/socket_pipeline.json --stdout &
COLLECTOR_PID=$!

# Wait for collector to start
sleep 2

# Verify collector is running
if ! kill -0 $COLLECTOR_PID 2>/dev/null; then
    echo "Collector failed to start"
    exit 1
fi

echo "Collector started with PID $COLLECTOR_PID"
echo "Socket server listening on localhost:8888"
echo

# Start the Python client
echo "Starting Python socket client..."
echo "Press Ctrl+C to stop"
echo

# Run the Python client (will show its own output)
python3 socket_client.py "$@"

# When the client is stopped, also stop the collector
echo "Stopping collector..."
kill $COLLECTOR_PID
wait $COLLECTOR_PID

echo "Done"