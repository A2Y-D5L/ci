# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2025-11-20

### Added

- **Event-Driven Architecture**: Complete separation of execution engine and rendering
  - `PipelineStartedEvent` - Emitted when pipeline execution begins
  - `TargetStartedEvent` - Emitted when a target begins execution
  - `TargetOutputEvent` - Emitted when a target produces output
  - `TargetCompletedEvent` - Emitted when a target completes
  - `PipelineCompletedEvent` - Emitted when pipeline execution finishes
- **Multiple Renderers** in `render/` package:
  - `LogRenderer` - Timestamped, namespaced logs for CI/CD environments
  - `SimpleRenderer` - Minimal symbolic output for quick feedback
  - `TUIRenderer` - Rich terminal UI for interactive development
  - `Collector` - Event collector for testing and programmatic access
- **Output Capture**: Automatic capture of stdout/stderr from command targets
  - `target.Cmd()` - Create command targets with output capture
  - `target.CmdWithDeps()` - Create command targets with dependencies
  - `RunWithStreams` interface for custom output capture
- **Core Engine Features**:
  - Automatic DAG resolution with cycle detection
  - Concurrent execution with configurable parallelism
  - Context-aware cancellation and timeout support
  - Deterministic error reporting (alphabetical order)
  - Thread-safe implementation with zero race conditions
- **Comprehensive Documentation**:
  - README.md - Complete user guide with examples
  - API.md - Full API reference
  - GETTING_STARTED.md - 5-minute quick start guide
  - render/README.md - Renderer package documentation
- **Examples**:
  - `examples/renderers/` - Demonstrates all renderer types
  - `examples/output-capture/` - Demonstrates output capture features
- **Test Suite**:
  - 67 comprehensive tests covering all features
  - 84.7% code coverage on core package
  - 96.9% code coverage on render package
  - Race detection enabled on all tests
  - Benchmark suite for performance validation

### Technical Details

- **Go Version**: Requires Go 1.25+ (uses `WaitGroup.Go()`)
- **Dependencies**: Zero external dependencies (standard library only)
- **Thread Safety**: All public APIs are thread-safe
- **Performance**: Microsecond-level execution overhead
- **Architecture**: Clean separation of concerns with event-driven design

### Initial Release Notes

This is the first public release of `github.com/a2y-d5l/ci`, a composable, event-driven CI pipeline execution engine for Go.

**Highlights**:

- ✨ Complete separation of execution engine and rendering
- 🎯 Event-driven architecture for maximum flexibility
- ⚡ Automatic parallel execution with dependency resolution
- 🔒 Thread-safe, production-ready implementation
- 📊 Multiple renderers for different use cases
- 🚀 Zero external dependencies
- ✅ 67 tests with 85%+ coverage

**Developer Experience**:

- < 1 minute to first success
- Progressive complexity (simple → advanced)
- Multiple working examples
- Comprehensive documentation

See [README.md](README.md) for complete documentation and [GETTING_STARTED.md](GETTING_STARTED.md) for a quick start guide.

[0.1.0]: https://github.com/a2y-d5l/ci/releases/tag/v0.1.0
