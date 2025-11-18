package target

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

var (
	Lint = New("Lint", "Run code linters", func(ctx context.Context) error {
		cmd := exec.CommandContext(ctx, "golangci-lint", "run", "./...")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("golangci-lint: %w", err)
		}
		return nil
	})

	Test = New("Test", "Run unit tests", func(ctx context.Context) error {
		cmd := exec.CommandContext(ctx, "go", "test", "./...")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("go test: %w", err)
		}
		return nil
	}, Lint)

	Build = New("Build", "Build binaries/artifacts", func(ctx context.Context) error {
		cmd := exec.CommandContext(ctx, "go", "build", "./cmd/...")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("go build: %w", err)
		}
		return nil
	}, Test)

	All = New("All", "Run full CI pipeline", func(ctx context.Context) error {
		// TargetAll is an orchestration node; no additional work needed.
		_ = ctx
		return nil
	}, Build)
)
