package target

import "context"

var (
	Lint = Cmd("Lint", "Run code linters", "golangci-lint", "run", "./...")

	Test = CmdWithDeps("Test", "Run unit tests", []T{Lint}, "go", "test", "./...")

	Build = CmdWithDeps("Build", "Build binaries/artifacts", []T{Test}, "go", "build", "./cmd/...")

	All = New("All", "Run full CI pipeline", func(ctx context.Context) error {
		// All is an orchestration node; no additional work needed.
		_ = ctx
		return nil
	}, Build)
)
