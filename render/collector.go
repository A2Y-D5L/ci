package render

import (
	"sync"

	"github.com/a2y-d5l/ci"
)

// Collector captures all events for testing/inspection.
type Collector struct {
	mu     sync.Mutex
	Events []ci.Event
}

// NewCollector creates a new event collector.
func NewCollector() *Collector {
	return &Collector{
		Events: make([]ci.Event, 0),
	}
}

// HandleEvent captures the event for later inspection.
func (c *Collector) HandleEvent(event ci.Event) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Events = append(c.Events, event)
}

// GetEvents returns a copy of all captured events.
func (c *Collector) GetEvents() []ci.Event {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]ci.Event{}, c.Events...)
}

// TargetStartedCount returns how many times a target started.
func (c *Collector) TargetStartedCount(targetName string) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	count := 0
	for _, e := range c.Events {
		if started, ok := e.(ci.TargetStartedEvent); ok {
			if started.TargetName() == targetName {
				count++
			}
		}
	}
	return count
}

// TargetCompletedCount returns how many times a target completed.
func (c *Collector) TargetCompletedCount(targetName string) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	count := 0
	for _, e := range c.Events {
		if completed, ok := e.(ci.TargetCompletedEvent); ok {
			if completed.TargetName() == targetName {
				count++
			}
		}
	}
	return count
}

// TargetOutputCount returns how many output events a target produced.
func (c *Collector) TargetOutputCount(targetName string) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	count := 0
	for _, e := range c.Events {
		if output, ok := e.(ci.TargetOutputEvent); ok {
			if output.TargetName() == targetName {
				count++
			}
		}
	}
	return count
}
