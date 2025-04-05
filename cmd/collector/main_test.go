package main

import (
	"testing"

	"github.com/sliink/collector/internal/core"
	"github.com/stretchr/testify/assert"
)

func TestRegisterPlugins(t *testing.T) {
	// This is a basic test to ensure the main package components exist
	
	t.Run("Core can be created", func(t *testing.T) {
		c := core.NewCore()
		assert.NotNil(t, c)
		
		// Initialize core
		success := c.Initialize()
		assert.True(t, success)
		
		// Stop core
		success = c.Stop()
		assert.True(t, success)
	})
}

// This is a minimal test suite for the main package
// For comprehensive testing, see the unit tests for each component in the internal/* packages