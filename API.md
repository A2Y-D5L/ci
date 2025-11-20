# API Reference

Complete API reference for `github.com/a2y-d5l/ci`.

## Package `ci`

Core execution engine package.

### Functions

#### `RunTargets`

```go
func RunTargets(ctx context.Context, targets ...target.T) (map[string]Result, error)
```

Runs CI targets without event handling (backward compatible mode).

**Parameters:**
- `ctx` - Context for cancellation and timeouts
- `targets` - Variable number of targets to execute

**Returns:**
- `map[string]Result` - Results for each executed target (including dependencies)
- `error` - Aggregated errors from failed targets, or nil if all succeeded

**Example:**
```go
results, err := ci.RunTargets(context.Background(), target.All)
if err != nil {
    log.Fatal(err)
}
```

---

#### `RunTargetsWithHandler`

```go
func RunTargetsWithHandler(ctx context.Context, handler EventHandler, targets ...target.T) (map[string]Result, error)
```

Runs CI targets with an event handler for observability.

**Parameters:**
- `ctx` - Context for cancellation and timeouts
- `handler` - Event handler to receive execution events (can be nil)
- `targets` - Variable number of targets to execute

**Returns:**
- `map[string]Result` - Results for each executed target
- `error` - Aggregated errors from failed targets, or nil if all succeeded

**Example:**
```go
renderer := render.NewLogRenderer(os.Stdout)
results, err := ci.RunTargetsWithHandler(context.Background(), renderer, target.All)
```

### Types

#### `Result`

```go
type Result struct {
    Target      target.T
    Err         error
    StartedAt   time.Time
    CompletedAt time.Time
    Skipped     bool
}
```

Captures the outcome of a single target execution.

**Fields:**
- `Target` - The executed target
- `Err` - Error returned from the target function (nil if successful)
- `StartedAt` - When the target began running
- `CompletedAt` - When the target finished running
- `Skipped` - True if target didn't run due to upstream failure

---

#### `Event` (interface)

```go
type Event interface {
    Timestamp() time.Time
    TargetName() string
}
```

Base interface for all events emitted during pipeline execution.

**Methods:**
- `Timestamp()` - When the event occurred
- `TargetName()` - Name of the target (empty string for pipeline events)

---

#### `EventHandler` (interface)

```go
type EventHandler interface {
    HandleEvent(event Event)
}
```

Interface for receiving events from the engine.

**Methods:**
- `HandleEvent(event Event)` - Called for each event

**Example:**
```go
type MyHandler struct{}

func (h *MyHandler) HandleEvent(event ci.Event) {
    switch e := event.(type) {
    case ci.TargetStartedEvent:
        fmt.Printf("Started: %s\n", e.TargetName())
    case ci.TargetCompletedEvent:
        fmt.Printf("Completed: %s\n", e.TargetName())
    }
}
```

---

#### `EventHandlerFunc`

```go
type EventHandlerFunc func(Event)
```

Function adapter for EventHandler interface.

**Example:**
```go
handler := ci.EventHandlerFunc(func(event ci.Event) {
    fmt.Printf("Event: %T\n", event)
})
```

---

#### `PipelineStartedEvent`

```go
type PipelineStartedEvent struct {
    Time         time.Time
    TotalTargets int
}
```

Emitted when pipeline execution begins.

**Fields:**
- `Time` - When the pipeline started
- `TotalTargets` - Total number of targets that will execute

---

#### `TargetStartedEvent`

```go
type TargetStartedEvent struct {
    Time   time.Time
    Target target.T
}
```

Emitted when a target begins execution.

**Fields:**
- `Time` - When the target started
- `Target` - Reference to the target

---

#### `TargetOutputEvent`

```go
type TargetOutputEvent struct {
    Time   time.Time
    Target target.T
    Stream StreamType
    Line   string
}
```

Emitted when a target produces output (stdout or stderr).

**Fields:**
- `Time` - When the output was produced
- `Target` - Reference to the target
- `Stream` - StreamStdout or StreamStderr
- `Line` - The output line (without newline)

---

#### `TargetCompletedEvent`

```go
type TargetCompletedEvent struct {
    Time     time.Time
    Target   target.T
    Duration time.Duration
    Error    error
    Skipped  bool
}
```

Emitted when a target completes execution.

**Fields:**
- `Time` - When the target completed
- `Target` - Reference to the target
- `Duration` - How long the target ran
- `Error` - Error if failed (nil if successful)
- `Skipped` - True if target was skipped due to dependency failure

---

#### `PipelineCompletedEvent`

```go
type PipelineCompletedEvent struct {
    Time     time.Time
    Duration time.Duration
    Results  map[string]Result
    Error    error
}
```

Emitted when pipeline execution finishes.

**Fields:**
- `Time` - When the pipeline completed
- `Duration` - Total pipeline duration
- `Results` - Results for all targets
- `Error` - Aggregated error (nil if all succeeded)

---

#### `StreamType`

```go
type StreamType int

const (
    StreamStdout StreamType = iota
    StreamStderr
)
```

Indicates output stream type for TargetOutputEvent.

**Constants:**
- `StreamStdout` - Standard output
- `StreamStderr` - Standard error

---

## Package `target`

Target definition and built-in targets.

### Functions

#### `New`

```go
func New(name, desc string, run func(ctx context.Context) error, deps ...T) *Target
```

Creates a new target with the given properties.

**Parameters:**
- `name` - Unique target name
- `desc` - Human-readable description
- `run` - Function to execute for this target
- `deps` - Optional dependencies (other targets)

**Returns:**
- `*Target` - The created target

**Example:**
```go
myTarget := target.New("MyTarget", "Does something", 
    func(ctx context.Context) error {
        fmt.Println("Executing...")
        return nil
    },
    target.Lint, // Dependency
)
```

---

#### `Cmd`

```go
func Cmd(name, desc string, cmdName string, args ...string) *CmdTarget
```

Creates a command target with automatic output capture support.

**Parameters:**
- `name` - Unique target name
- `desc` - Human-readable description
- `cmdName` - Command to execute (e.g., "go", "echo")
- `args` - Command arguments

**Returns:**
- `*CmdTarget` - The created command target

**Example:**
```go
lint := target.Cmd("Lint", "Run linters", "golangci-lint", "run", "./...")
```

---

#### `CmdWithDeps`

```go
func CmdWithDeps(name, desc string, deps []T, cmdName string, args ...string) *CmdTarget
```

Creates a command target with dependencies.

**Parameters:**
- `name` - Unique target name
- `desc` - Human-readable description
- `deps` - Slice of dependency targets
- `cmdName` - Command to execute
- `args` - Command arguments

**Returns:**
- `*CmdTarget` - The created command target

**Example:**
```go
test := target.CmdWithDeps("Test", "Run tests",
    []target.T{lint},  // Dependencies
    "go", "test", "./...")
```

### Types

#### `T` (interface)

```go
type T interface {
    Name() string
    Description() string
    Dependencies() []T
    Run(ctx context.Context) error
}
```

Interface that all targets must implement.

**Methods:**
- `Name()` - Returns unique target name
- `Description()` - Returns human-readable description
- `Dependencies()` - Returns slice of dependency targets
- `Run(ctx context.Context)` - Executes the target

---

#### `RunWithStreams` (interface)

```go
type RunWithStreams interface {
    T
    RunWithStreams(ctx context.Context, stdout, stderr io.Writer) error
}
```

Optional interface for targets that support output capture.

**Methods:**
- All methods from `T`
- `RunWithStreams(ctx, stdout, stderr)` - Executes target with custom streams

**Note:** When a target implements this interface and an EventHandler is provided, the engine will call `RunWithStreams` instead of `Run`, enabling output capture.

---

#### `Target`

```go
type Target struct {
    // Fields are private
}
```

Default target implementation.

**Created by:** `target.New()`

---

#### `CmdTarget`

```go
type CmdTarget struct {
    *Target
    // Additional private fields
}
```

Command target with automatic RunWithStreams implementation.

**Created by:** `target.Cmd()` or `target.CmdWithDeps()`

### Built-in Targets

#### `Lint`

```go
var Lint = Cmd("Lint", "Run code linters", "golangci-lint", "run", "./...")
```

Runs golangci-lint on the codebase.

---

#### `Test`

```go
var Test = CmdWithDeps("Test", "Run unit tests", []T{Lint}, "go", "test", "./...")
```

Runs Go tests. Depends on `Lint`.

---

#### `Build`

```go
var Build = CmdWithDeps("Build", "Build binaries", []T{Test}, "go", "build", "./cmd/...")
```

Builds Go binaries. Depends on `Test`.

---

#### `All`

```go
var All = New("All", "Run full CI pipeline", func(ctx context.Context) error {
    return nil
}, Build)
```

Orchestration target that runs the full pipeline. Depends on `Build`.

---

## Package `render`

Built-in event renderers.

### Functions

#### `NewLogRenderer`

```go
func NewLogRenderer(w io.Writer) *LogRenderer
```

Creates a renderer that outputs timestamped, namespaced logs.

**Best for:** CI/CD environments, log aggregation

**Output format:**
```
[2025-11-20 16:05:51.750] [PIPELINE] Starting with 3 targets
[2025-11-20 16:05:51.750] [Lint] Started
[2025-11-20 16:05:51.752] [Lint] [stdout] golangci-lint run ./...
[2025-11-20 16:05:51.752] [Lint] completed (duration: 1.98ms)
```

**Example:**
```go
renderer := render.NewLogRenderer(os.Stdout)
ci.RunTargetsWithHandler(ctx, renderer, target.All)
```

---

#### `NewSimpleRenderer`

```go
func NewSimpleRenderer(w io.Writer) *SimpleRenderer
```

Creates a renderer that outputs minimal symbolic output.

**Best for:** Quick local runs, minimal output needed

**Output format:**
```
▶ Lint
✓ Lint
▶ Test
✓ Test
```

**Symbols:**
- `▶` - Target started
- `✓` - Target succeeded
- `✗` - Target failed
- `○` - Target skipped

**Example:**
```go
renderer := render.NewSimpleRenderer(os.Stdout)
ci.RunTargetsWithHandler(ctx, renderer, target.All)
```

---

#### `NewTUIRenderer`

```go
func NewTUIRenderer(w io.Writer) *TUIRenderer
```

Creates a renderer with rich terminal UI.

**Best for:** Local development, interactive sessions

**Output features:**
- Pipeline start/completion banners
- Organized output per target (non-interleaved)
- Summary section showing all results
- Box drawing characters for visual organization

**Example:**
```go
renderer := render.NewTUIRenderer(os.Stdout)
ci.RunTargetsWithHandler(ctx, renderer, target.All)
```

---

#### `NewCollector`

```go
func NewCollector() *Collector
```

Creates an event collector for testing/inspection.

**Best for:** Testing, programmatic event analysis

**Example:**
```go
collector := render.NewCollector()
ci.RunTargetsWithHandler(ctx, collector, target.All)

events := collector.GetEvents()
fmt.Printf("Collected %d events\n", len(events))
```

### Types

#### `LogRenderer`

```go
type LogRenderer struct {
    // Fields are private
}
```

Timestamped, namespaced log renderer.

**Thread-safe:** Yes

---

#### `SimpleRenderer`

```go
type SimpleRenderer struct {
    // Fields are private
}
```

Minimal symbolic output renderer.

**Thread-safe:** Yes

---

#### `TUIRenderer`

```go
type TUIRenderer struct {
    // Fields are private
}
```

Rich terminal UI renderer.

**Thread-safe:** Yes

---

#### `Collector`

```go
type Collector struct {
    // Fields are private
}
```

Event collector for testing/inspection.

**Methods:**
- `GetEvents() []ci.Event` - Returns copy of all captured events
- `TargetStartedCount(name string) int` - Count started events for target
- `TargetCompletedCount(name string) int` - Count completed events for target
- `TargetOutputCount(name string) int` - Count output events for target

**Thread-safe:** Yes

**Example:**
```go
collector := render.NewCollector()
ci.RunTargetsWithHandler(ctx, collector, target.All)

if collector.TargetStartedCount("Lint") != 1 {
    t.Error("Lint should have started once")
}
```

---

## Common Patterns

### Pattern 1: Simple Execution

```go
results, err := ci.RunTargets(context.Background(), target.All)
if err != nil {
    log.Fatal(err)
}
```

### Pattern 2: With Logging

```go
renderer := render.NewLogRenderer(os.Stdout)
results, err := ci.RunTargetsWithHandler(context.Background(), renderer, target.All)
```

### Pattern 3: Custom Target

```go
myTarget := target.New("MyTarget", "Custom target",
    func(ctx context.Context) error {
        // Your logic here
        return nil
    },
    target.Lint, // Dependency
)
```

### Pattern 4: Command Target

```go
deploy := target.Cmd("Deploy", "Deploy to production",
    "kubectl", "apply", "-f", "deployment.yaml")
```

### Pattern 5: Custom Event Handler

```go
handler := ci.EventHandlerFunc(func(event ci.Event) {
    switch e := event.(type) {
    case ci.TargetCompletedEvent:
        if e.Error != nil {
            log.Printf("FAILED: %s: %v", e.TargetName(), e.Error)
        }
    }
})

ci.RunTargetsWithHandler(ctx, handler, target.All)
```

### Pattern 6: Collecting Metrics

```go
collector := render.NewCollector()
results, err := ci.RunTargetsWithHandler(ctx, collector, target.All)

// Analyze events
for _, event := range collector.GetEvents() {
    if output, ok := event.(ci.TargetOutputEvent); ok {
        // Process output
    }
}
```

---

## Thread Safety

All public APIs are thread-safe:

- ✅ `RunTargets` and `RunTargetsWithHandler` can be called concurrently
- ✅ All renderers are thread-safe
- ✅ Event handlers receive events from multiple goroutines
- ✅ Collector methods are safe to call during execution

## Error Handling

Errors are aggregated using `errors.Join()`:

```go
results, err := ci.RunTargets(ctx, target.All)
if err != nil {
    // err contains all target errors in alphabetical order
    // Individual target errors also available in results map
    for name, result := range results {
        if result.Err != nil {
            fmt.Printf("%s failed: %v\n", name, result.Err)
        }
    }
}
```

## Context Cancellation

All target execution respects context cancellation:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()

results, err := ci.RunTargets(ctx, target.All)
// Execution will stop if context is cancelled
```

---

**Last Updated:** November 20, 2025  
**Version:** 0.1.0  
**Go Version:** 1.25+
