package target

import "context"

// Preset targets for common CI/CD tasks.
// These are global variables by design to provide a simple, default workflow.
//
//nolint:gochecknoglobals // These are intentional preset targets for the public API
var (
	Lint = Cmd("Lint", "Run code linters", "golangci-lint", "run", "./...")

	Test = CmdWithDeps("Test", "Run unit tests", []T{Lint}, "go", "test", "./...")

	Build = CmdWithDeps("Build", "Build binaries/artifacts", []T{Test}, "go", "build", "./cmd/...")

	All = New("All", "Run full CI pipeline", func(_ context.Context) error {
		// All is an orchestration node; no additional work needed.
		return nil
	}, Build)
)
