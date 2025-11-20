package ci

import (
	"time"

	"github.com/a2y-d5l/ci/target"
)

// Event represents a state change during pipeline execution.
type Event interface {
	Timestamp() time.Time
	TargetName() string
}

// PipelineStartedEvent is emitted when pipeline execution begins.
type PipelineStartedEvent struct {
	Time         time.Time
	TotalTargets int
}

func (e PipelineStartedEvent) Timestamp() time.Time { return e.Time }
func (e PipelineStartedEvent) TargetName() string   { return "" }

// TargetStartedEvent is emitted when a target begins execution.
type TargetStartedEvent struct {
	Time   time.Time
	Target target.T
}

func (e TargetStartedEvent) Timestamp() time.Time { return e.Time }
func (e TargetStartedEvent) TargetName() string   { return e.Target.Name() }

// TargetOutputEvent is emitted when a target produces output.
type TargetOutputEvent struct {
	Time   time.Time
	Target target.T
	Stream StreamType
	Line   string
}

func (e TargetOutputEvent) Timestamp() time.Time { return e.Time }
func (e TargetOutputEvent) TargetName() string   { return e.Target.Name() }

// TargetCompletedEvent is emitted when a target completes execution.
type TargetCompletedEvent struct {
	Time     time.Time
	Target   target.T
	Duration time.Duration
	Error    error
	Skipped  bool
}

func (e TargetCompletedEvent) Timestamp() time.Time { return e.Time }
func (e TargetCompletedEvent) TargetName() string   { return e.Target.Name() }

// PipelineCompletedEvent is emitted when pipeline execution finishes.
type PipelineCompletedEvent struct {
	Time     time.Time
	Duration time.Duration
	Results  map[string]Result
	Error    error
}

func (e PipelineCompletedEvent) Timestamp() time.Time { return e.Time }
func (e PipelineCompletedEvent) TargetName() string   { return "" }

// StreamType indicates output stream.
type StreamType int

const (
	StreamStdout StreamType = iota
	StreamStderr
)

// EventHandler receives events from the engine.
type EventHandler interface {
	HandleEvent(event Event)
}

// EventHandlerFunc is a function adapter for EventHandler.
type EventHandlerFunc func(Event)

func (f EventHandlerFunc) HandleEvent(event Event) {
	f(event)
}
