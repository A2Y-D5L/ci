# Getting Started with github.com/a2y-d5l/ci

A 5-minute guide to get you running CI pipelines in Go.

## Installation

```bash
go get github.com/a2y-d5l/ci
```

**Requirements**: Go 1.25 or later

## Your First Pipeline (30 seconds)

Create `main.go`:

```go
package main

import (
    "context"
    "log"
    
    "github.com/a2y-d5l/ci"
    "github.com/a2y-d5l/ci/target"
)

func main() {
    results, err := ci.RunTargets(context.Background(), target.All)
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Success! Ran %d targets", len(results))
}
```

Run it:

```bash
go run main.go
```

**That's it!** You just ran a CI pipeline with linting, testing, and building.

## What Just Happened?

The `target.All` target includes:

1. **Lint** - Runs `golangci-lint run ./...`
2. **Test** - Runs `go test ./...` (depends on Lint)
3. **Build** - Runs `go build ./cmd/...` (depends on Test)
4. **All** - Orchestrates the full pipeline (depends on Build)

The engine automatically:
- ✅ Resolved dependencies (Lint → Test → Build)
- ✅ Ran targets in parallel where possible
- ✅ Handled errors and skipped dependent targets on failure
- ✅ Returned results for all targets

## Adding Logging (1 minute)

Want to see what's happening? Add a renderer:

```go
package main

import (
    "context"
    "log"
    "os"
    
    "github.com/a2y-d5l/ci"
    "github.com/a2y-d5l/ci/render"
    "github.com/a2y-d5l/ci/target"
)

func main() {
    // Create a log renderer
    renderer := render.NewLogRenderer(os.Stdout)
    
    // Run with the renderer
    results, err := ci.RunTargetsWithHandler(
        context.Background(), 
        renderer, 
        target.All,
    )
    
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Success! Ran %d targets", len(results))
}
```

Now you'll see timestamped, namespaced logs:

```
[2025-11-20 16:05:51.750] [PIPELINE] Starting with 4 targets
[2025-11-20 16:05:51.750] [Lint] Started
[2025-11-20 16:05:51.752] [Lint] [stdout] golangci-lint run ./...
[2025-11-20 16:05:51.752] [Lint] completed (duration: 1.98ms)
...
```

## Creating Your Own Target (2 minutes)

Let's add a custom deploy target:

```go
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
    // Create a custom target that runs after Build
    deploy := target.Cmd(
        "Deploy",
        "Deploy to production",
        "kubectl", "apply", "-f", "deployment.yaml",
    )
    // Note: Cmd creates a target with automatic output capture
    
    // Or create a target with dependencies
    deployWithDeps := target.CmdWithDeps(
        "Deploy",
        "Deploy to production",
        []target.T{target.Build},  // Depends on Build
        "kubectl", "apply", "-f", "deployment.yaml",
    )
    
    renderer := render.NewLogRenderer(os.Stdout)
    results, err := ci.RunTargetsWithHandler(
        context.Background(),
        renderer,
        deployWithDeps,
    )
    
    if err != nil {
        fmt.Fprintf(os.Stderr, "Pipeline failed: %v\n", err)
        os.Exit(1)
    }
    
    fmt.Printf("✅ Deployed successfully! (%d targets)\n", len(results))
}
```

## Custom Logic Target (3 minutes)

Need custom Go code instead of shell commands?

```go
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
    // Create a target with custom logic
    notify := target.New(
        "Notify",
        "Send notification",
        func(ctx context.Context) error {
            // Your custom Go code here
            fmt.Println("📧 Sending deployment notification...")
            
            // Simulate sending notification
            // sendSlackMessage("Deployment complete!")
            
            return nil
        },
        target.Build,  // Depends on Build completing
    )
    
    renderer := render.NewLogRenderer(os.Stdout)
    results, err := ci.RunTargetsWithHandler(
        context.Background(),
        renderer,
        notify,
    )
    
    if err != nil {
        fmt.Fprintf(os.Stderr, "Pipeline failed: %v\n", err)
        os.Exit(1)
    }
    
    fmt.Println("✅ Pipeline complete!")
}
```

## Choose Your Renderer (4 minutes)

Different renderers for different environments:

### LogRenderer - For CI/CD

Best for: GitHub Actions, Jenkins, GitLab CI, etc.

```go
renderer := render.NewLogRenderer(os.Stdout)
```

Output:
```
[2025-11-20 16:05:51.750] [PIPELINE] Starting with 3 targets
[2025-11-20 16:05:51.750] [Lint] Started
[2025-11-20 16:05:51.752] [Lint] completed (duration: 1.98ms)
```

### SimpleRenderer - For Quick Local Runs

Best for: Quick feedback, minimal noise

```go
renderer := render.NewSimpleRenderer(os.Stdout)
```

Output:
```
▶ Lint
✓ Lint
▶ Test
✓ Test
```

### TUIRenderer - For Rich Local UI

Best for: Interactive development, watching progress

```go
renderer := render.NewTUIRenderer(os.Stdout)
```

Output:
```
╔═══════════════════════════════════════╗
║  CI Pipeline Starting                 ║
║  Total Targets: 3                     ║
╚═══════════════════════════════════════╝

┌─ Lint ────────────────────────────────┐
│ ⚙ Running...                          │
│ golangci-lint run ./...               │
└────────────────────────────────────────┘
```

## Next Steps

### Option 1: Explore Examples

```bash
# See all renderers in action
go run github.com/a2y-d5l/ci/examples/renderers@latest

# See output capture demo
go run github.com/a2y-d5l/ci/examples/output-capture@latest
```

### Option 2: Read Documentation

- [README.md](README.md) - Complete user guide
- [API.md](API.md) - Full API reference
- [PHASE_3_COMPLETE.md](PHASE_3_COMPLETE.md) - Output capture guide

### Option 3: Learn from Tests

The test suite has excellent examples of real-world usage:

- `engine_test.go` - Basic engine usage
- `events_test.go` - Event handling
- `integration_test.go` - End-to-end scenarios
- `render/render_test.go` - Renderer usage

## Common Patterns

### Pattern 1: Run Multiple Specific Targets

```go
results, err := ci.RunTargets(ctx, target.Lint, target.Test)
```

### Pattern 2: Custom Event Handler

```go
handler := ci.EventHandlerFunc(func(event ci.Event) {
    switch e := event.(type) {
    case ci.TargetCompletedEvent:
        if e.Error != nil {
            log.Printf("❌ %s failed: %v", e.TargetName(), e.Error)
        } else {
            log.Printf("✅ %s completed in %v", e.TargetName(), e.Duration)
        }
    }
})

ci.RunTargetsWithHandler(ctx, handler, target.All)
```

### Pattern 3: With Timeout

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()

results, err := ci.RunTargets(ctx, target.All)
```

### Pattern 4: Collect Metrics

```go
collector := render.NewCollector()
results, err := ci.RunTargetsWithHandler(ctx, collector, target.All)

// Analyze events
events := collector.GetEvents()
for _, event := range events {
    // Process events programmatically
}
```

## Tips & Tricks

### Tip 1: Use Cmd for Shell Commands

Instead of:
```go
target.New("Build", "Build app", func(ctx context.Context) error {
    cmd := exec.CommandContext(ctx, "go", "build", "./...")
    return cmd.Run()
})
```

Do this:
```go
target.Cmd("Build", "Build app", "go", "build", "./...")
```

Benefits:
- ✅ Shorter code
- ✅ Automatic output capture
- ✅ Works with all renderers

### Tip 2: Choose the Right Renderer

- **CI/CD**: `LogRenderer` - Structured logs for aggregation
- **Quick check**: `SimpleRenderer` - Minimal output
- **Development**: `TUIRenderer` - Rich visual feedback
- **Testing**: `Collector` - Programmatic access

### Tip 3: Handle Errors Gracefully

```go
results, err := ci.RunTargets(ctx, target.All)
if err != nil {
    // err contains all target errors
    // Individual errors also in results map
    for name, result := range results {
        if result.Err != nil {
            log.Printf("%s: %v", name, result.Err)
        }
    }
    os.Exit(1)
}
```

## What Makes This Special?

✨ **Event-Driven**: Everything is an observable event  
🎯 **Zero Coupling**: Engine and rendering are completely separate  
⚡ **Concurrent**: Automatic parallel execution  
🔒 **Thread-Safe**: All operations are safe for concurrent use  
📊 **Observable**: Full visibility into execution  
🎨 **Flexible**: Multiple renderers for different needs  
🚀 **Fast**: Minimal overhead, excellent performance  
✅ **Production-Ready**: Thoroughly tested, well documented

## Help & Support

- **Documentation**: See [README.md](README.md) and [API.md](API.md)
- **Examples**: Check `examples/` directory
- **Tests**: Read test files for usage patterns
- **Issues**: Open an issue on GitHub

## You're Ready!

You now know:
- ✅ How to run the built-in targets
- ✅ How to add logging/rendering
- ✅ How to create custom targets
- ✅ How to choose the right renderer
- ✅ Common patterns and best practices

**Start building your CI pipeline!** 🚀

---

**Next**: Read [README.md](README.md) for complete documentation  
**API Reference**: See [API.md](API.md) for detailed API docs  
**Examples**: Run `go run examples/renderers/main.go`
