package core

import (
	"sync"
	"time"

	"github.com/sliink/collector/internal/model"
)

// Event represents a system event with metadata
type Event struct {
	Type      model.EventType
	SourceID  string
	Data      interface{}
	Timestamp time.Time
}

// NewEvent creates a new event
func NewEvent(eventType model.EventType, sourceID string, data interface{}) Event {
	return Event{
		Type:      eventType,
		SourceID:  sourceID,
		Data:      data,
		Timestamp: time.Now(),
	}
}

// EventCallback is a function that is called when an event occurs
type EventCallback func(Event)

// EventBus handles event publication and subscription
type EventBus struct {
	subscribers map[model.EventType]map[string]EventCallback
	mutex       sync.RWMutex
	BaseComponent
}

// NewEventBus creates a new event bus
func NewEventBus() *EventBus {
	e := &EventBus{
		subscribers:    make(map[model.EventType]map[string]EventCallback),
		BaseComponent:  NewBaseComponent("event_bus", "Event Bus"),
	}
	return e
}

// Initialize prepares the event bus for operation
func (b *EventBus) Initialize() bool {
	b.SetStatus(model.StatusInitialized)
	return true
}

// Start begins event bus operation
func (b *EventBus) Start() bool {
	b.SetStatus(model.StatusRunning)
	return true
}

// Stop halts event bus operation
func (b *EventBus) Stop() bool {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	// Clear all subscribers
	b.subscribers = make(map[model.EventType]map[string]EventCallback)
	
	b.SetStatus(model.StatusStopped)
	return true
}

// Subscribe registers a callback for a specific event type
func (b *EventBus) Subscribe(eventType model.EventType, listenerID string, callback EventCallback) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if b.subscribers[eventType] == nil {
		b.subscribers[eventType] = make(map[string]EventCallback)
	}
	
	b.subscribers[eventType][listenerID] = callback
}

// Unsubscribe removes a subscriber from a specific event type
func (b *EventBus) Unsubscribe(eventType model.EventType, listenerID string) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if b.subscribers[eventType] != nil {
		delete(b.subscribers[eventType], listenerID)
	}
}

// Publish broadcasts an event to all subscribers
func (b *EventBus) Publish(event Event) {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	if b.GetStatus() != model.StatusRunning {
		return
	}

	if subscribers, exists := b.subscribers[event.Type]; exists {
		// Create a local copy of callbacks to avoid race conditions 
		var callbacks []EventCallback
		for _, callback := range subscribers {
			callbacks = append(callbacks, callback)
		}
		
		// Call the callbacks outside the lock
		for _, callback := range callbacks {
			callback(event) // Direct call instead of goroutine
		}
	}
}