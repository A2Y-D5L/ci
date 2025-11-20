package target

import (
	"context"
	"io"
)

type T interface {
	Name() string
	Description() string
	Dependencies() []T
	Run(ctx context.Context) error
}

// RunWithStreams is an optional interface that targets can implement
// to support output capture. When a target implements this interface,
// the engine will provide custom io.Writer instances for stdout and stderr,
// allowing output to be captured and emitted as TargetOutputEvent.
//
// If a target does not implement this interface, it falls back to Run()
// and output goes directly to os.Stdout/Stderr (not captured).
type RunWithStreams interface {
	T
	RunWithStreams(ctx context.Context, stdout, stderr io.Writer) error
}

// Target is the logical name of a CI target in the repo.
type Target struct {
	name string
	desc string
	fn   func(ctx context.Context) error
	deps []T
}

func New(name, desc string, run func(ctx context.Context) error, deps ...T) *Target {
	return &Target{
		name: name,
		desc: desc,
		fn:   run,
		deps: deps,
	}
}

func (t *Target) Name() string {
	return t.name
}

func (t *Target) Description() string {
	return t.desc
}

func (t *Target) Dependencies() []T {
	return t.deps
}

func (t *Target) Run(ctx context.Context) error {
	return t.fn(ctx)
}