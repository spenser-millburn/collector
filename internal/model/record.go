package model

import (
	"time"
)

// Record represents a raw record of data
type Record struct {
	Source     string
	Timestamp  time.Time
	RawData    []byte
	Attributes map[string]interface{}
}