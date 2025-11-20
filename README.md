# ci

> A composable, event-driven CI pipeline execution engine for Go.

[![Go Version](https://img.shields.io/badge/Go-1.25%2B-blue)](https://go.dev/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/a2y-d5l/ci)](https://goreportcard.com/report/github.com/a2y-d5l/ci)

## Features

- **Event-Driven Architecture**: All execution state changes emitted as structured events
- **Output Capture**: Capture and stream target stdout/stderr as events
- **Clean Separation**: Engine executes, consumers render - zero coupling
- **Multiple Renderers**: TUI for local development, structured logs for CI/CD
- **Dependency Management**: Automatic DAG resolution with cycle detection
- **Concurrent Execution**: Parallel target execution respecting dependencies
- **Context-Aware**: Proper cancellation and timeout support
- **Extensible**: Custom event handlers for rendering, logging, metrics, etc.
- **Production-Ready**: Thread-safe, thoroughly tested, high performance

## Installation

```bash
go get github.com/a2y-d5l/ci
```

## Quick Start

### Basic Usage (No Events)

```go
import (
    "context"
    "github.com/a2y-d5l/ci"
    "github.com/a2y-d5l/ci/target"
)

func main() {
    results, err := ci.RunTargets(context.Background(), target.All)
    if err != nil {
        log.Fatal(err)
    }
}
```

### With Event Handler

```go
import (
    "context"
    "fmt"
    "github.com/a2y-d5l/ci"
    "github.com/a2y-d5l/ci/target"
)

func main() {
    // Create a simple event handler
    handler := ci.EventHandlerFunc(func(event ci.Event) {
        switch e := event.(type) {
        case ci.TargetStartedEvent:
            fmt.Printf("▶ Started: %s\n", e.TargetName())
        case ci.TargetCompletedEvent:
            if e.Error != nil {
                fmt.Printf("✗ Failed: %s (%v)\n", e.TargetName(), e.Error)
            } else if e.Skipped {
                fmt.Printf("○ Skipped: %s\n", e.TargetName())
            } else {
                fmt.Printf("✓ Completed: %s (%v)\n", e.TargetName(), e.Duration)
            }
        }
    })

    results, err := ci.RunTargetsWithHandler(context.Background(), handler, target.All)
    if err != nil {
        log.Fatal(err)
    }
}
```

### With Built-in Renderers

The `render` package provides ready-to-use renderers for common scenarios:

```go
import (
    "os"
    "github.com/a2y-d5l/ci"
    "github.com/a2y-d5l/ci/render"
    "github.com/a2y-d5l/ci/target"
)

func main() {
    // LogRenderer - for CI/CD environments
    // Outputs timestamped, namespaced logs with captured stdout/stderr
    renderer := render.NewLogRenderer(os.Stdout)
    
    results, err := ci.RunTargetsWithHandler(context.Background(), renderer, target.All)
    if err != nil {
        log.Fatal(err)
    }
}
```

**Output example:**

```text
[2025-11-20 16:05:51.750] [PIPELINE] Starting with 3 targets
[2025-11-20 16:05:51.750] [Lint] Started
[2025-11-20 16:05:51.752] [Lint] [stdout] golangci-lint run ./...
[2025-11-20 16:05:51.752] [Lint] completed (duration: 1.98ms)
```

**Other renderers:**

- `render.NewSimpleRenderer(w)` - Minimal output with symbols (▶ ✓ ✗ ○)
- `render.NewTUIRenderer(w)` - Rich terminal UI with organized, non-interleaved output
- `render.NewCollector()` - Captures events for testing/analysis

## Creating Custom Targets

### Simple Function Target

```go
import (
    "context"
    "github.com/a2y-d5l/ci/target"
)

var MyTarget = target.New(
    "my-target",
    "Description of my target",
    func(ctx context.Context) error {
        // Your custom logic here
        return nil
    },
    // Optional dependencies
    target.Lint,
    target.Test,
)
```

### Command Target (With Output Capture)

For running shell commands with automatic output capture:

```go
import (
    "github.com/a2y-d5l/ci/target"
)

// Simple command
var Build = target.Cmd("Build", "Build the application", "go", "build", "./...")

// Command with dependencies
var Deploy = target.CmdWithDeps(
    "Deploy", 
    "Deploy to production",
    []target.T{Build, Test},  // Dependencies
    "kubectl", "apply", "-f", "deployment.yaml",
)
```

**Benefits of `Cmd()`**:

- ✅ Automatic stdout/stderr capture
- ✅ Output emitted as `TargetOutputEvent`
- ✅ Works seamlessly with all renderers
- ✅ One-line target creation

## Event System

The engine emits events at every state transition, enabling custom rendering, logging, metrics collection, and more.

### Event Types

- **PipelineStartedEvent**: Emitted when pipeline execution begins
- **TargetStartedEvent**: Emitted when a target begins execution
- **TargetOutputEvent**: Emitted when a target produces output (stdout/stderr)
- **TargetCompletedEvent**: Emitted when a target completes (success, failure, or skipped)
- **PipelineCompletedEvent**: Emitted when pipeline execution finishes

### Event Handler Interface

```go
type EventHandler interface {
    HandleEvent(event Event)
}
```

All events implement:

```go
type Event interface {
    Timestamp() time.Time
    TargetName() string
}
```

### Example: Custom Event Collector

```go
type Collector struct {
    mu     sync.Mutex
    Events []ci.Event
}

func (c *Collector) HandleEvent(event ci.Event) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.Events = append(c.Events, event)
}

// Use it
collector := &Collector{}
results, err := ci.RunTargetsWithHandler(ctx, collector, targets...)
```

## Architecture

```text
┌─────────────────────────────────────────┐
│         Consumer Layer                   │
│  ┌──────────┐      ┌──────────┐        │
│  │ Renderer │      │  Metrics │        │
│  └────┬─────┘      └────┬─────┘        │
│       │                 │               │
│       └────────┬────────┘               │
└────────────────┼──────────────────────────┘
                 │ Events
┌────────────────┼──────────────────────────┐
│                ▼                          │
│         Event Stream                      │
│  (TargetStarted, Completed, etc.)        │
└────────────────┬──────────────────────────┘
                 │
┌────────────────┼──────────────────────────┐
│                ▼                          │
│       Execution Engine                    │
│  - DAG Resolution                         │
│  - Concurrency Control                    │
│  - Error Handling                         │
└─────────────────────────────────────────┘
```

The engine is **completely decoupled** from rendering. All state changes are emitted as events, and consumers decide how to process them.

## Performance

Benchmarks on Apple M4 (arm64):

- **Simple DAG**: ~2,490 ns/op, 2,568 B/op, 22 allocs/op
- **Diamond DAG**: ~3,570 ns/op, 3,512 B/op, 36 allocs/op  
- **Complex DAG**: ~4,603 ns/op, 4,328 B/op, 47 allocs/op

The event system adds minimal overhead to execution.

## Requirements

- **Go 1.25+** (uses `WaitGroup.Go()` introduced in Go 1.25)
- **Zero external dependencies** (standard library only)

## Project Status

**v0.1.0** - Initial public release (November 20, 2025)

- ✅ Production-ready
- ✅ Thread-safe, race-condition free
- ✅ 67 comprehensive tests
- ✅ 85%+ code coverage
- ✅ Zero external dependencies
- ✅ Fully documented

See [CHANGELOG.md](CHANGELOG.md) for release history.

## Documentation

- **[README.md](README.md)** - This file (overview and examples)
- **[GETTING_STARTED.md](GETTING_STARTED.md)** - 5-minute quick start guide
- **[API.md](API.md)** - Complete API reference
- **[CONTRIBUTING.md](CONTRIBUTING.md)** - Contribution guidelines
- **[CHANGELOG.md](CHANGELOG.md)** - Version history

## Examples

Run the examples to see the renderers in action:

```bash
# See all renderers
go run github.com/a2y-d5l/ci/examples/renderers@latest

# See output capture features
go run github.com/a2y-d5l/ci/examples/output-capture@latest
```

Or explore the [examples/](examples/) directory.

## Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

Apache License 2.0 - See [LICENSE](LICENSE) for details.

## Acknowledgments

Built with a focus on developer experience and clean architecture. Special thanks to the Go team for the excellent standard library and the `WaitGroup.Go()` addition in Go 1.25.
