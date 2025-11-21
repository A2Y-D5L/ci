package main

import (
	"context"
	"fmt"
	"os"

	"github.com/a2y-d5l/ci"
	"github.com/a2y-d5l/ci/render"
	"github.com/a2y-d5l/ci/target"
)

func main() {
	// Define some example targets
	lint := target.New("Lint", "Run code linters", func(_ context.Context) error {
		fmt.Println("Running golangci-lint...")
		return nil
	})

	test := target.New("Test", "Run unit tests", func(_ context.Context) error {
		fmt.Println("Running go test...")
		fmt.Println("=== RUN   TestExample")
		fmt.Println("--- PASS: TestExample (0.00s)")
		fmt.Println("PASS")
		return nil
	}, lint)

	build := target.New("Build", "Build binaries", func(_ context.Context) error {
		fmt.Println("Building application...")
		fmt.Println("Build successful")
		return nil
	}, test)

	all := target.New("All", "Run full CI pipeline", func(_ context.Context) error {
		// All is an orchestration target
		return nil
	}, build)

	// Example 1: Using LogRenderer (for CI/CD environments)
	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Println("Example 1: LogRenderer (CI/CD Style)")
	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Println()

	logRenderer := render.NewLogRenderer(os.Stdout)
	_, err := ci.RunTargetsWithHandler(context.Background(), logRenderer, all)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Pipeline failed: %v\n", err)
	}

	fmt.Println("\n═══════════════════════════════════════════════════════")
	fmt.Println("Example 2: SimpleRenderer (Minimal Output)")
	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Println()

	simpleRenderer := render.NewSimpleRenderer(os.Stdout)
	_, err = ci.RunTargetsWithHandler(context.Background(), simpleRenderer, all)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Pipeline failed: %v\n", err)
	}

	fmt.Println("\n═══════════════════════════════════════════════════════")
	fmt.Println("Example 3: TUIRenderer (Rich Terminal UI)")
	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Println()

	tuiRenderer := render.NewTUIRenderer(os.Stdout)
	_, err = ci.RunTargetsWithHandler(context.Background(), tuiRenderer, all)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Pipeline failed: %v\n", err)
	}

	fmt.Println("\n═══════════════════════════════════════════════════════")
	fmt.Println("Example 4: Collector (For Programmatic Access)")
	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Println()

	collector := render.NewCollector()
	_, err = ci.RunTargetsWithHandler(context.Background(), collector, all)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Pipeline failed: %v\n", err)
	}

	events := collector.GetEvents()
	fmt.Printf("Collected %d events:\n", len(events))
	fmt.Printf("  - Pipeline started: %d\n", countEventType[ci.PipelineStartedEvent](events))
	fmt.Printf("  - Targets started: %d\n", countEventType[ci.TargetStartedEvent](events))
	fmt.Printf("  - Targets completed: %d\n", countEventType[ci.TargetCompletedEvent](events))
	fmt.Printf("  - Pipeline completed: %d\n", countEventType[ci.PipelineCompletedEvent](events))
}

// countEventType is a generic helper to count events of a specific type.
func countEventType[T any](events []ci.Event) int {
	count := 0
	for _, e := range events {
		if _, ok := e.(T); ok {
			count++
		}
	}
	return count
}
