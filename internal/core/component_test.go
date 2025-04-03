package core

import (
	"testing"

	"github.com/sliink/collector/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestNewBaseComponent(t *testing.T) {
	testCases := []struct {
		name     string
		id       string
		expected BaseComponent
	}{
		{
			name: "Creates component with correct ID and name",
			id:   "test_id",
			expected: BaseComponent{
				id:     "test_id",
				name:   "Test Component",
				status: model.StatusUninitialized,
				config: make(map[string]interface{}),
			},
		},
		{
			name: "Creates component with empty ID",
			id:   "",
			expected: BaseComponent{
				id:     "",
				name:   "Empty ID Component",
				status: model.StatusUninitialized,
				config: make(map[string]interface{}),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			component := NewBaseComponent(tc.id, tc.expected.name)
			
			assert.Equal(t, tc.expected.id, component.id)
			assert.Equal(t, tc.expected.name, component.name)
			assert.Equal(t, tc.expected.status, component.status)
			assert.NotNil(t, component.config)
		})
	}
}

func TestBaseComponentMethods(t *testing.T) {
	component := NewBaseComponent("test_id", "Test Component")
	
	t.Run("ID returns correct identifier", func(t *testing.T) {
		assert.Equal(t, "test_id", component.ID())
	})
	
	t.Run("Name returns correct name", func(t *testing.T) {
		assert.Equal(t, "Test Component", component.Name())
	})
	
	t.Run("GetStatus returns current status", func(t *testing.T) {
		assert.Equal(t, model.StatusUninitialized, component.GetStatus())
	})
	
	t.Run("SetStatus updates status", func(t *testing.T) {
		component.SetStatus(model.StatusRunning)
		assert.Equal(t, model.StatusRunning, component.GetStatus())
	})

	t.Run("Configure with nil config returns false", func(t *testing.T) {
		result := component.Configure(nil)
		assert.False(t, result)
	})
	
	t.Run("Configure with valid config returns true", func(t *testing.T) {
		config := map[string]interface{}{
			"test_key": "test_value",
		}
		result := component.Configure(config)
		assert.True(t, result)
		assert.Equal(t, "test_value", component.config["test_key"])
	})
}