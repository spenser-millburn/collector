package core

import (
	"sync"
	"testing"
	"time"

	"github.com/sliink/collector/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestNewEvent(t *testing.T) {
	sourceID := "test_source"
	eventType := model.EventDataReceived
	data := "test_data"
	
	event := NewEvent(eventType, sourceID, data)
	
	assert.Equal(t, eventType, event.Type)
	assert.Equal(t, sourceID, event.SourceID)
	assert.Equal(t, data, event.Data)
	assert.NotZero(t, event.Timestamp)
	assert.True(t, time.Since(event.Timestamp) < time.Second)
}

func TestNewEventBus(t *testing.T) {
	eventBus := NewEventBus()
	
	assert.NotNil(t, eventBus)
	assert.NotNil(t, eventBus.subscribers)
	assert.Equal(t, "event_bus", eventBus.ID())
	assert.Equal(t, "Event Bus", eventBus.Name())
}

func TestEventBusLifecycle(t *testing.T) {
	eventBus := NewEventBus()
	
	t.Run("Initialize sets correct status", func(t *testing.T) {
		success := eventBus.Initialize()
		assert.True(t, success)
		assert.Equal(t, model.StatusInitialized, eventBus.GetStatus())
	})
	
	t.Run("Start sets correct status", func(t *testing.T) {
		success := eventBus.Start()
		assert.True(t, success)
		assert.Equal(t, model.StatusRunning, eventBus.GetStatus())
	})
	
	t.Run("Stop sets correct status", func(t *testing.T) {
		success := eventBus.Stop()
		assert.True(t, success)
		assert.Equal(t, model.StatusStopped, eventBus.GetStatus())
	})
}

func TestEventBusSubscribeAndPublish(t *testing.T) {
	eventBus := NewEventBus()
	eventBus.Initialize()
	eventBus.Start()
	
	eventType := model.EventDataReceived
	sourceID := "test_source"
	data := "test_data"
	
	var receivedEvent Event
	var wg sync.WaitGroup
	wg.Add(1)
	
	t.Run("Subscribe adds callback to correct eventType", func(t *testing.T) {
		eventBus.Subscribe(eventType, "test_listener", func(event Event) {
			receivedEvent = event
			wg.Done()
		})
		
		assert.Len(t, eventBus.subscribers[eventType], 1)
	})
	
	t.Run("Publish sends event to subscribers", func(t *testing.T) {
		event := NewEvent(eventType, sourceID, data)
		eventBus.Publish(event)
		
		// Wait for event processing (with timeout)
		waitDone := make(chan struct{})
		go func() {
			wg.Wait()
			close(waitDone)
		}()
		
		select {
		case <-waitDone:
			// Continue with assertions
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Timed out waiting for event callback")
		}
		
		assert.Equal(t, eventType, receivedEvent.Type)
		assert.Equal(t, sourceID, receivedEvent.SourceID)
		assert.Equal(t, data, receivedEvent.Data)
	})
	
	t.Run("Unsubscribe removes callback", func(t *testing.T) {
		eventBus.Unsubscribe(eventType, "test_listener")
		assert.Empty(t, eventBus.subscribers[eventType])
	})
	
	t.Run("Publish with no subscribers does nothing", func(t *testing.T) {
		// No assertions needed, just making sure it doesn't panic
		event := NewEvent(model.EventError, "source", "data")
		eventBus.Publish(event)
	})
	
	t.Run("Publish when stopped does nothing", func(t *testing.T) {
		eventBus.Stop()
		
		var called bool
		eventBus.subscribers[eventType] = map[string]EventCallback{
			"test": func(event Event) { called = true },
		}
		
		event := NewEvent(eventType, sourceID, data)
		eventBus.Publish(event)
		
		time.Sleep(50 * time.Millisecond) // Give time for any goroutines to complete
		assert.False(t, called)
	})
}

func TestMultipleSubscribers(t *testing.T) {
	eventBus := NewEventBus()
	eventBus.Initialize()
	eventBus.Start()
	
	eventType := model.EventDataReceived
	event := NewEvent(eventType, "source", "data")
	
	var wg sync.WaitGroup
	wg.Add(2)
	
	var called1, called2 bool
	
	eventBus.Subscribe(eventType, "listener1", func(e Event) {
		called1 = true
		wg.Done()
	})
	
	eventBus.Subscribe(eventType, "listener2", func(e Event) {
		called2 = true
		wg.Done()
	})
	
	eventBus.Publish(event)
	
	// Wait for event processing (with timeout)
	waitDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitDone)
	}()
	
	select {
	case <-waitDone:
		// Continue with assertions
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timed out waiting for event callbacks")
	}
	
	assert.True(t, called1)
	assert.True(t, called2)
}