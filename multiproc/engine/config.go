package engine

import (
	"context"
	"time"
)

// CommandFactory creates Command instances from ProcessSpecs.
// This abstraction enables dependency injection for testing.
type CommandFactory func(ctx context.Context, spec ProcessSpec) (Command, error)

// Engine is the core execution engine that runs processes and emits ProcessLine events.
// It is decoupled from any specific rendering logic, making it reusable for
// different output formats (terminal UI, JSON logs, progress bars, etc.).
type Engine struct {
	// Specs defines the processes to run.
	Specs []ProcessSpec

	// ShutdownTimeout is the maximum time to wait for graceful shutdown.
	ShutdownTimeout time.Duration

	// CommandFactory creates commands. If nil, uses DefaultCommandFactory.
	CommandFactory CommandFactory
}

// New creates a new Engine with the given specs and optional shutdown timeout.
func New(specs []ProcessSpec, shutdownTimeout time.Duration) *Engine {
	return &Engine{
		Specs:           specs,
		ShutdownTimeout: shutdownTimeout,
		CommandFactory:  nil, // Will use DefaultCommandFactory
	}
}

// WithCommandFactory returns a copy of the engine with a custom command factory.
// This is primarily useful for testing.
func (eng *Engine) WithCommandFactory(factory CommandFactory) *Engine {
	return &Engine{
		Specs:           eng.Specs,
		ShutdownTimeout: eng.ShutdownTimeout,
		CommandFactory:  factory,
	}
}
