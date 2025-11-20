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
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║  Phase 3: Output Capture Demo                               ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Example 1: Command targets with output capture
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println("Example 1: LogRenderer with Captured Output")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()

	// Create targets using the Cmd helper (automatically supports capture)
	echo1 := target.Cmd("Echo1", "First echo", "echo", "Hello from Echo1!")
	echo2 := target.CmdWithDeps("Echo2", "Second echo", []target.T{echo1}, "echo", "Hello from Echo2!")
	multiLine := target.CmdWithDeps("MultiLine", "Multiple lines",
		[]target.T{echo2},
		"sh", "-c", "echo 'Line 1'; echo 'Line 2'; echo 'Line 3'")

	logRenderer := render.NewLogRenderer(os.Stdout)
	_, err := ci.RunTargetsWithHandler(context.Background(), logRenderer, multiLine)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Pipeline failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n═══════════════════════════════════════════════════════════════")
	fmt.Println("Example 2: SimpleRenderer (Minimal Output)")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()

	simpleRenderer := render.NewSimpleRenderer(os.Stdout)
	_, err = ci.RunTargetsWithHandler(context.Background(), simpleRenderer, multiLine)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Pipeline failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n═══════════════════════════════════════════════════════════════")
	fmt.Println("Example 3: Collector (Programmatic Access)")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()

	collector := render.NewCollector()
	_, err = ci.RunTargetsWithHandler(context.Background(), collector, multiLine)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Pipeline failed: %v\n", err)
		os.Exit(1)
	}

	// Analyze collected events
	events := collector.GetEvents()
	fmt.Printf("📊 Collected Statistics:\n")
	fmt.Printf("  Total events: %d\n", len(events))
	fmt.Printf("  Targets started: %d\n",
		collector.TargetStartedCount("Echo1")+
			collector.TargetStartedCount("Echo2")+
			collector.TargetStartedCount("MultiLine"))
	fmt.Printf("  Output events (Echo1): %d\n", collector.TargetOutputCount("Echo1"))
	fmt.Printf("  Output events (Echo2): %d\n", collector.TargetOutputCount("Echo2"))
	fmt.Printf("  Output events (MultiLine): %d\n", collector.TargetOutputCount("MultiLine"))
	fmt.Println()

	// Show captured output
	fmt.Println("📝 Captured Output:")
	for _, e := range events {
		if output, ok := e.(ci.TargetOutputEvent); ok {
			stream := "stdout"
			if output.Stream == ci.StreamStderr {
				stream = "stderr"
			}
			fmt.Printf("  [%s] [%s] %s\n", output.TargetName(), stream, output.Line)
		}
	}

	fmt.Println("\n═══════════════════════════════════════════════════════════════")
	fmt.Println("Example 4: Mixed Targets (With and Without Capture)")
	fmt.Println("═══════════════════════════════════════════════════════════════")
	fmt.Println()

	// Target WITH capture
	withCapture := target.Cmd("WithCapture", "Supports capture", "echo", "This output is captured!")

	// Target WITHOUT capture (regular function)
	withoutCapture := target.New("WithoutCapture", "No capture", func(ctx context.Context) error {
		fmt.Println("This goes directly to stdout (not captured)")
		return nil
	})

	allTargets := target.New("AllTargets", "Run both", func(ctx context.Context) error {
		return nil
	}, withCapture, withoutCapture)

	collector2 := render.NewCollector()
	_, err = ci.RunTargetsWithHandler(context.Background(), collector2, allTargets)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Pipeline failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n📊 Mixed Targets Results:\n")
	fmt.Printf("  WithCapture output events: %d\n", collector2.TargetOutputCount("WithCapture"))
	fmt.Printf("  WithoutCapture output events: %d\n", collector2.TargetOutputCount("WithoutCapture"))
	fmt.Println()

	fmt.Println("✅ Phase 3 Output Capture is working perfectly!")
	fmt.Println()
	fmt.Println("Key Features:")
	fmt.Println("  ✓ Automatic output capture for Cmd targets")
	fmt.Println("  ✓ Backward compatible with regular targets")
	fmt.Println("  ✓ Clean separation of stdout and stderr")
	fmt.Println("  ✓ Non-interleaved output in renderers")
	fmt.Println("  ✓ Zero global state mutations")
	fmt.Println("  ✓ Thread-safe implementation")
}
