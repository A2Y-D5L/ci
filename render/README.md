# Render Package Quick Reference

The `render` package provides multiple rendering strategies for CI pipeline execution events.

## Quick Start

```go
import (
    "context"
    "os"
    "github.com/a2y-d5l/ci"
    "github.com/a2y-d5l/ci/render"
    "github.com/a2y-d5l/ci/target"
)

// Choose your renderer
renderer := render.NewLogRenderer(os.Stdout)

// Run with renderer
results, err := ci.RunTargetsWithHandler(context.Background(), renderer, target.All)
```

## Available Renderers

### LogRenderer - For CI/CD Environments

**Best for**: Jenkins, GitHub Actions, GitLab CI, CircleCI, etc.

```go
renderer := render.NewLogRenderer(os.Stdout)
```

**Features**:
- Timestamped output (millisecond precision)
- Namespaced by target name
- Stream indicators (stdout/stderr)
- Duration tracking
- Easily parsed by log aggregation tools

**Output Example**:
```
[2025-11-20 15:45:56.445] [PIPELINE] Starting with 4 targets
[2025-11-20 15:45:56.445] [Lint] Started
[2025-11-20 15:45:56.445] [Lint] completed (duration: 1.2s)
```

### SimpleRenderer - For Quick Feedback

**Best for**: Local development, scripts, minimal output

```go
renderer := render.NewSimpleRenderer(os.Stdout)
```

**Features**:
- Minimal output
- Unicode symbols (▶ ✓ ✗ ○)
- No timestamps
- Clean and concise

**Output Example**:
```
▶ Lint
✓ Lint
▶ Test
✓ Test
```

### TUIRenderer - For Rich Terminal UI

**Best for**: Interactive local development, monitoring

```go
renderer := render.NewTUIRenderer(os.Stdout)
```

**Features**:
- Box drawing characters
- Organized by target (non-interleaved output)
- Status indicators
- Pipeline summary
- Shows last 5 lines per target

**Output Example**:
```
╔═══════════════════════════════════════╗
║  CI Pipeline Starting                 ║
║  Total Targets: 4                     ║
╚═══════════════════════════════════════╝

┌─ Lint ─────────────────────────────────┐
│ ⚙ Running...                           │
│ golangci-lint run ./...                │
└────────────────────────────────────────┘
```

### Collector - For Testing & Analysis

**Best for**: Unit tests, programmatic access, metrics

```go
collector := render.NewCollector()
ci.RunTargetsWithHandler(ctx, collector, targets...)

// Access events
events := collector.GetEvents()
startCount := collector.TargetStartedCount("Lint")
outputCount := collector.TargetOutputCount("Test")
```

**Features**:
- Captures all events
- Thread-safe storage
- Helper methods for counting
- Perfect for testing

## Creating Custom Renderers

Implement the `ci.EventHandler` interface:

```go
type MyRenderer struct {
    // your state
}

func (r *MyRenderer) HandleEvent(event ci.Event) {
    switch e := event.(type) {
    case ci.PipelineStartedEvent:
        // Handle pipeline start
    case ci.TargetStartedEvent:
        // Handle target start
    case ci.TargetOutputEvent:
        // Handle target output (if using output capture)
    case ci.TargetCompletedEvent:
        // Handle target completion
    case ci.PipelineCompletedEvent:
        // Handle pipeline completion
    }
}
```

## Event Types

All events implement the `ci.Event` interface:

```go
type Event interface {
    Timestamp() time.Time
    TargetName() string  // Empty for pipeline events
}
```

### PipelineStartedEvent

Emitted when pipeline execution begins.

```go
type PipelineStartedEvent struct {
    Time         time.Time
    TotalTargets int
}
```

### TargetStartedEvent

Emitted when a target begins execution.

```go
type TargetStartedEvent struct {
    Time   time.Time
    Target target.T
}
```

### TargetOutputEvent

Emitted when a target produces output (Phase 3).

```go
type TargetOutputEvent struct {
    Time   time.Time
    Target target.T
    Stream StreamType  // StreamStdout or StreamStderr
    Line   string
}
```

### TargetCompletedEvent

Emitted when a target completes (success, failure, or skipped).

```go
type TargetCompletedEvent struct {
    Time     time.Time
    Target   target.T
    Duration time.Duration
    Error    error  // nil if successful
    Skipped  bool   // true if dependencies failed
}
```

### PipelineCompletedEvent

Emitted when pipeline execution finishes.

```go
type PipelineCompletedEvent struct {
    Time     time.Time
    Duration time.Duration
    Results  map[string]ci.Result
    Error    error  // Aggregated errors if any
}
```

## Thread Safety

All renderers are **thread-safe** and can be used concurrently:

```go
renderer := render.NewLogRenderer(os.Stdout)

// Safe to use from multiple goroutines
go ci.RunTargetsWithHandler(ctx1, renderer, targets1...)
go ci.RunTargetsWithHandler(ctx2, renderer, targets2...)
```

## Testing with Collector

```go
func TestMyPipeline(t *testing.T) {
    collector := render.NewCollector()
    
    _, err := ci.RunTargetsWithHandler(ctx, collector, myTarget)
    if err != nil {
        t.Fatal(err)
    }
    
    // Verify events
    if collector.TargetStartedCount("MyTarget") != 1 {
        t.Error("Expected target to start once")
    }
    
    events := collector.GetEvents()
    // Assert on event order, content, etc.
}
```

## Chaining Renderers

You can create a multi-renderer by composing handlers:

```go
type MultiRenderer struct {
    handlers []ci.EventHandler
}

func (m *MultiRenderer) HandleEvent(event ci.Event) {
    for _, h := range m.handlers {
        h.HandleEvent(event)
    }
}

// Use it
multi := &MultiRenderer{
    handlers: []ci.EventHandler{
        render.NewLogRenderer(logFile),
        render.NewTUIRenderer(os.Stdout),
        myCustomMetrics,
    },
}

ci.RunTargetsWithHandler(ctx, multi, targets...)
```

## Performance

All renderers have minimal overhead:

- Event handling is synchronous but fast
- Mutex locks are held only during writes
- No buffering or batching (real-time output)
- Memory usage proportional to number of targets

**Benchmark**: ~2-15 µs/op depending on DAG complexity (including execution)

## See Also

- [PHASE_2_COMPLETE.md](PHASE_2_COMPLETE.md) - Detailed Phase 2 documentation
- [examples/renderers/main.go](examples/renderers/main.go) - Example usage
- [render/render_test.go](render/render_test.go) - Test examples
