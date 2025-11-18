package target

import "context"

type T interface {
	Name() string
	Description() string
	Dependencies() []T
	Run(ctx context.Context) error
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