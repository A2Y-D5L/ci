// file: cmd/example/main.go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/a2y-d5l/ci/wip/engine"
)

// loggingSink is a simple EventSink that prints a human-readable log of
// target lifecycle events. In a real system, this could be a TUI renderer,
// JSON logger, or test recorder.
type loggingSink struct{}

func (loggingSink) HandleEvent(ev engine.Event) {
	name := "<nil>"
	if ev.Target != nil {
		name = ev.Target.Name
	}

	switch ev.Type {
	case engine.EventTargetStarted:
		fmt.Printf("[event] %s STARTED at %s\n", name, ev.Time.Format(time.RFC3339Nano))
	case engine.EventTargetCompleted:
		status := "ok"
		if ev.Result != nil && ev.Result.Err != nil {
			status = "failed"
		}
		fmt.Printf("[event] %s COMPLETED (%s) at %s\n", name, status, ev.Time.Format(time.RFC3339Nano))
	case engine.EventTargetSkipped:
		fmt.Printf("[event] %s SKIPPED at %s\n", name, ev.Time.Format(time.RFC3339Nano))
	default:
		fmt.Printf("[event] %s UNKNOWN event at %s\n", name, ev.Time.Format(time.RFC3339Nano))
	}
}

func main() {
	ctx := context.Background()

	// Define a small CI graph:
	//
	//   lint   test
	//     \   /
	//    build
	//      |
	//   package
	//      |
	//    all
	targets := []engine.Target{
		{
			Name: "lint",
			Desc: "Run linters",
			Run: func(ctx context.Context) error {
				time.Sleep(300 * time.Millisecond)
				return nil
			},
		},
		{
			Name: "test",
			Desc: "Run unit tests",
			Run: func(ctx context.Context) error {
				time.Sleep(500 * time.Millisecond)
				return nil
			},
		},
		{
			Name: "build",
			Desc: "Build binaries",
			Deps: []string{"lint", "test"},
			Run: func(ctx context.Context) error {
				time.Sleep(400 * time.Millisecond)
				return nil
			},
		},
		{
			Name: "package",
			Desc: "Package artifacts",
			Deps: []string{"build"},
			Run: func(_ context.Context) error {
				time.Sleep(200 * time.Millisecond)
				return nil
			},
		},
		{
			Name: "all",
			Desc: "Top-level aggregate target",
			Deps: []string{"package"},
			Run: func(_ context.Context) error {
				time.Sleep(100 * time.Millisecond)
				return nil
			},
		},
	}

	// Compile the plan once.
	plan, err := engine.BuildPlan(targets...)
	if err != nil {
		panic(err)
	}

	fmt.Println("Plan targets:", plan.TargetNames())
	fmt.Println("Plan stages:")
	for _, s := range plan.Stages() {
		fmt.Println("  ", s)
	}

	// Create an executor from the plan with an attached logging sink.
	exec := engine.NewExecutor(
		plan,
		engine.WithMaxWorkers(4),
		engine.WithFailFast(),
		engine.WithEventSink(loggingSink{}),
	)

	// Start a goroutine to consume the event stream.
	go func() {
		for ev := range exec.Events() {
			// loggingSink already prints via WithEventSink.
			// In a real system you might fan these out to multiple sinks.
			_ = ev
		}
	}()

	// Start another goroutine to observe Result snapshots.
	go func() {
		for res := range exec.Results() {
			fmt.Printf("[results chan] %-8s skipped=%v err=%v\n", res.Name, res.Skipped, res.Err)
		}
	}()

	// Run only "all" (and its transitive deps).
	fmt.Println("\n--- Running target: all ---")
	summary, err := exec.Run(ctx, "all")
	if err != nil {
		fmt.Printf("Run completed with error: %v\n", err)
	}
	printSummary(summary)

	// Reuse the same plan with a different root and context.
	fmt.Println("\n--- Running target: build (only) ---")
	ctx2, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	exec2 := engine.NewExecutor(plan, engine.WithMaxWorkers(2))
	go func() {
		for range exec2.Events() {
			// ignore for this run
		}
	}()
	go func() {
		for range exec2.Results() {
			// ignore for this run
		}
	}()

	summary2, err := exec2.Run(ctx2, "build")
	if err != nil {
		fmt.Printf("Run completed with error: %v\n", err)
	}
	printSummary(summary2)
}

func printSummary(summary engine.RunSummary) {
	fmt.Print("Result:")
	if summary.Failed {
		fmt.Print(" FAILED")
	}
	fmt.Println()
	for name, res := range summary.Results {
		status := "ok"
		if res.Skipped {
			status = "skipped"
		} else if res.Err != nil {
			status = "failed"
		}
		fmt.Printf("  %-8s status=%-7s start=%s end=%s err=%v\n",
			name, status,
			res.StartedAt.Format(time.RFC3339Nano),
			res.CompletedAt.Format(time.RFC3339Nano),
			res.Err,
		)
	}
}
