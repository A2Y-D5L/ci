# Contributing to github.com/a2y-d5l/ci

Thank you for your interest in contributing! This document provides guidelines for contributing to the project.

## Getting Started

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:

   ```bash
   git clone https://github.com/YOUR_USERNAME/ci.git
   cd ci
   ```

3. **Install dependencies** (none! standard library only)
4. **Run tests** to ensure everything works:

   ```bash
   go test -v -race ./...
   ```

## Development Workflow

### Making Changes

1. **Create a branch** for your changes:

   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes** following the guidelines below
3. **Write or update tests** to cover your changes
4. **Run the test suite**:

   ```bash
   # Run all tests with race detection
   go test -v -race ./...
   
   # Check coverage
   go test -cover ./...
   
   # Run benchmarks
   go test -bench=. -benchmem ./...
   ```

5. **Format your code**:

   ```bash
   go fmt ./...
   ```

6. **Commit your changes** with clear, descriptive messages:

   ```bash
   git commit -m "Add feature X to improve Y"
   ```

### Submitting Changes

1. **Push to your fork**:

   ```bash
   git push origin feature/your-feature-name
   ```

2. **Open a Pull Request** on GitHub
3. **Describe your changes** clearly in the PR description
4. **Wait for review** - maintainers will review and provide feedback

## Code Guidelines

### Architecture Principles

This project follows strict architectural principles:

- **Separation of Concerns**: Engine executes, consumers render
- **Zero Coupling**: Engine has zero knowledge of rendering
- **Event-Driven**: All state changes emitted as events
- **Thread Safety**: All public APIs must be thread-safe
- **No Dependencies**: Standard library only (except tests)

### Code Style

- Follow standard Go conventions (`go fmt`, `go vet`)
- Use meaningful variable and function names
- Add comments for exported types, functions, and complex logic
- Keep functions focused and reasonably sized
- Avoid global state and side effects

### Testing Requirements

All contributions must include tests:

- **Unit tests** for new functionality
- **Integration tests** for end-to-end scenarios
- **Race detection** must pass (`go test -race`)
- **Maintain or improve** code coverage (currently 85%+)
- **Benchmark tests** for performance-critical changes

Example test structure:

```go
func TestFeature(t *testing.T) {
    for _, test := range []struct {
        name     string
        input    Input
        expected Output
    }{
        {
            name:     "descriptive_test_case_name",
            input:    ...,
            expected: ...,
        },
    } {
        t.Run(test.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### Documentation Requirements

- **Update README.md** if adding user-facing features
- **Update API.md** for new public APIs
- **Update CHANGELOG.md** with your changes
- **Add examples** for significant new features
- **Use clear comments** for complex code

## Types of Contributions

### Bug Fixes

- Check existing issues to avoid duplicates
- Include a test that demonstrates the bug
- Fix the bug and verify the test passes
- Update CHANGELOG.md under "Fixed" section

### New Features

- Open an issue first to discuss the feature
- Ensure it aligns with project goals and architecture
- Include comprehensive tests
- Update documentation
- Add examples if applicable
- Update CHANGELOG.md under "Added" section

### Documentation Improvements

- Fix typos, improve clarity, add examples
- Ensure accuracy of technical details
- Update CHANGELOG.md under "Changed" or "Documentation" section

### Performance Improvements

- Include benchmark results showing improvement
- Ensure no regression in other benchmarks
- Run the full benchmark suite:

  ```bash
  go test -bench=. -benchmem -count=5 ./... | tee benchmark.txt
  ```

## Project Structure

```text
ci/
├── engine.go           - Core execution engine
├── events.go           - Event types and interfaces
├── engine_test.go      - Engine tests
├── events_test.go      - Event system tests
├── integration_test.go - End-to-end tests
├── engine_bench_test.go- Performance benchmarks
├── target/
│   ├── types.go        - Target interface
│   ├── targets.go      - Built-in targets
│   └── command.go      - Command target implementation
├── render/
│   ├── logger.go       - Log renderer
│   ├── simple.go       - Simple renderer
│   ├── tui.go          - TUI renderer
│   ├── collector.go    - Event collector
│   └── render_test.go  - Renderer tests
└── examples/
    ├── renderers/      - Renderer examples
    └── output-capture/ - Output capture example
```

## Event-Driven Architecture

When contributing to the engine or renderers:

- **Engine** emits events, never renders
- **Renderers** consume events, never execute
- Events are the only communication channel
- New event types should be rare and well-justified

## Testing Philosophy

- Tests should be clear and maintainable
- Use table-driven tests where appropriate
- Test both success and failure cases
- Verify edge cases and error conditions
- Use `-race` flag to detect race conditions
- Prefer integration tests for end-to-end validation

## Performance Expectations

- Engine overhead should be minimal (microseconds)
- Concurrent execution should scale with `GOMAXPROCS`
- Memory allocations should be reasonable
- No performance regressions without justification

## Questions or Problems?

- **Check existing documentation**: README.md, API.md, GETTING_STARTED.md
- **Look at examples**: `examples/` directory
- **Read the tests**: Test files show usage patterns
- **Open an issue**: For bugs, feature requests, or questions

## Code of Conduct

- Be respectful and constructive
- Focus on the code, not the person
- Welcome newcomers and help them learn
- Give credit where credit is due

## Release Process

(For maintainers)

1. Update CHANGELOG.md with new version
2. Update version references in documentation
3. Tag the release: `git tag v0.x.0`
4. Push tag: `git push origin v0.x.0`
5. GitHub Actions will create the release

## License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.

---

Thank you for contributing to make `github.com/a2y-d5l/ci` better! 🚀
