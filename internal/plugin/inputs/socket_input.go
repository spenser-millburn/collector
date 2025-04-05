package inputs

import (
	"net"
	"sync"
	"time"

	"github.com/sliink/collector/internal/model"
)

// SocketInput represents a socket input plugin for collecting data from network sockets
type SocketInput struct {
	id            string
	status        model.Status
	config        map[string]interface{}
	listener      net.Listener
	listeners     []net.Listener
	mu            sync.Mutex
	done          chan struct{}
	statusMu      sync.RWMutex
	recordsChan   chan model.Record
	pendingRecords []model.Record
	recordsMu     sync.Mutex
}

// NewSocketInput creates a new socket input instance
func NewSocketInput(id string) *SocketInput {
	return &SocketInput{
		id:             id,
		status:         model.StatusStopped,
		config:         make(map[string]interface{}),
		listeners:      make([]net.Listener, 0),
		done:           make(chan struct{}),
		recordsChan:    make(chan model.Record, 100),
		pendingRecords: make([]model.Record, 0),
	}
}

// ID returns the plugin ID
func (p *SocketInput) ID() string {
	return p.id
}

// Type returns the plugin type
func (p *SocketInput) Type() model.PluginType {
	return model.InputPluginType
}

// GetType returns the plugin type
func (p *SocketInput) GetType() model.PluginType {
	return model.InputPluginType
}

// Name returns the plugin's human-readable name
func (p *SocketInput) Name() string {
	return "Socket Input"
}

// Status returns the component status
func (p *SocketInput) GetStatus() model.Status {
	p.statusMu.RLock()
	defer p.statusMu.RUnlock()
	return p.status
}

// SetStatus sets the component status
func (p *SocketInput) SetStatus(status model.Status) {
	p.statusMu.Lock()
	defer p.statusMu.Unlock()
	p.status = status
}

// Validate validates the plugin configuration
func (p *SocketInput) Validate() bool {
	// Check if there's a protocol available
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Check if enabled is explicitly set to false
	if enabled, ok := p.config["enabled"].(bool); ok && !enabled {
		// Disabled plugins are valid
		return true
	}
	
	// Check for required fields
	protocol, ok := p.config["protocol"].(string)
	if !ok || protocol == "" {
		protocol = "tcp" // Default to TCP if not specified
	}
	
	address, ok := p.config["address"].(string)
	if !ok || address == "" {
		address = "localhost:8888" // Default address if not specified
	}
	
	// Validate protocol is either tcp or udp
	if protocol != "tcp" && protocol != "udp" {
		return false
	}
	
	return true
}

// Configure configures the plugin
func (p *SocketInput) Configure(config map[string]interface{}) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	p.config = config
	return true
}

// RegisterWithCore registers the plugin with a core instance
func (p *SocketInput) RegisterWithCore(core model.CoreAPI) bool {
	// Add registration logic if needed
	return true
}

// Initialize prepares the plugin for operation
func (p *SocketInput) Initialize() bool {
	p.SetStatus(model.StatusInitialized)
	return true
}

// Start begins plugin operation
func (p *SocketInput) Start() bool {
	p.done = make(chan struct{})
	p.recordsChan = make(chan model.Record, 100)
	
	// Get configuration parameters
	p.mu.Lock()
	
	protocol, _ := p.config["protocol"].(string)
	if protocol == "" {
		protocol = "tcp"
	}
	
	address, _ := p.config["address"].(string)
	if address == "" {
		address = "localhost:8888"
	}
	
	p.mu.Unlock()
	
	// Start a goroutine to collect records from the channel
	go func() {
		for {
			select {
			case <-p.done:
				return
			case record := <-p.recordsChan:
				p.recordsMu.Lock()
				p.pendingRecords = append(p.pendingRecords, record)
				p.recordsMu.Unlock()
			}
		}
	}()
	
	// Start the socket listener
	listener, err := net.Listen(protocol, address)
	if err != nil {
		return false
	}
	
	p.mu.Lock()
	p.listener = listener
	p.listeners = append(p.listeners, listener)
	p.mu.Unlock()
	
	// Set status to running
	p.SetStatus(model.StatusRunning)
	
	// Start handling connections in a goroutine
	go p.handleConnections()
	
	return true
}

// Stop halts plugin operation
func (p *SocketInput) Stop() bool {
	// Signal goroutines to stop
	close(p.done)
	
	// Close all listeners
	p.mu.Lock()
	for _, listener := range p.listeners {
		if listener != nil {
			listener.Close()
		}
	}
	p.listeners = make([]net.Listener, 0)
	p.mu.Unlock()
	
	p.SetStatus(model.StatusStopped)
	return true
}

// Collect collects data from the input sources
func (p *SocketInput) Collect() []*model.DataBatch {
	// This is called periodically to collect data
	
	// Check if input is enabled
	p.mu.Lock()
	enabled, ok := p.config["enabled"].(bool)
	p.mu.Unlock()
	
	if ok && !enabled {
		// Skip collection if explicitly disabled
		return nil
	}
	
	// Check if we're running
	if p.GetStatus() != model.StatusRunning {
		return nil
	}
	
	// Get any pending records
	p.recordsMu.Lock()
	records := make([]model.Record, len(p.pendingRecords))
	copy(records, p.pendingRecords)
	p.pendingRecords = p.pendingRecords[:0] // Clear the pending records
	p.recordsMu.Unlock()
	
	// If no records, return an empty batch
	if len(records) == 0 {
		return []*model.DataBatch{
			{
				SourceID:    p.id,
				BatchType:   model.LogTelemetryType,
				Timestamp:   time.Now(),
				Records:     []model.Record{},
				Attributes:  map[string]interface{}{},
			},
		}
	}
	
	// Return a batch with the collected records
	return []*model.DataBatch{
		{
			SourceID:    p.id,
			BatchType:   model.LogTelemetryType,
			Timestamp:   time.Now(),
			Records:     records,
			Attributes:  map[string]interface{}{},
		},
	}
}

// Accept connections and handle data
func (p *SocketInput) handleConnections() {
	for {
		select {
		case <-p.done:
			return
		default:
			p.mu.Lock()
			listener := p.listener
			p.mu.Unlock()
			
			if listener == nil {
				return
			}
			
			conn, err := listener.Accept()
			if err != nil {
				// Check if we're shutting down
				select {
				case <-p.done:
					return
				default:
					// If not shutting down, this is an error
					continue
				}
			}
			
			// Handle each connection in a goroutine
			go p.handleConnection(conn)
		}
	}
}

// Handle a single connection
func (p *SocketInput) handleConnection(conn net.Conn) {
	defer conn.Close()
	
	// Buffer for reading data
	buffer := make([]byte, 4096)
	
	for {
		select {
		case <-p.done:
			return
		default:
			// Set read deadline to periodically check for done signal
			conn.SetReadDeadline(time.Now().Add(1 * time.Second))
			
			n, err := conn.Read(buffer)
			if err != nil {
				// Check if it's a timeout and we should continue
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				
				// Any other error means we're done with this connection
				return
			}
			
			if n > 0 {
				// Process the received data
				data := buffer[:n]
				
				// Create a record with the received data
				record := model.Record{
					Source:     p.id,
					Timestamp:  time.Now(),
					RawData:    data,
					Attributes: map[string]interface{}{
						"protocol": p.config["protocol"],
						"address":  p.config["address"],
					},
				}
				
				// In a real implementation, we would pass this to a channel
				// Send the record to the records channel for processing
				select {
				case p.recordsChan <- record:
					// Record sent successfully
				default:
					// Channel is full, drop the record
				}
			}
		}
	}
}