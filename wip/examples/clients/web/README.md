# CI Pipeline Web Dashboard

A real-time, visually-rich web-based UI for rendering events, results, and summaries of CI engine executions.

## Features

- **Real-time Pipeline Visualization**: See targets organized by execution stages in a visual DAG representation
- **Live Event Log**: Watch events stream in real-time as targets start, complete, or fail
- **Interactive Controls**: Start, cancel, and clear pipeline runs with dedicated buttons
- **Failure Simulation**: Test failure handling with the "Run with Failure" button
- **Comprehensive Summary**: View detailed run statistics and per-target results after completion
- **Modern Dark Theme**: GitHub-inspired dark UI with smooth animations
- **Server-Sent Events (SSE)**: Efficient real-time communication without WebSocket complexity

## Architecture

```text
┌─────────────────────────────────────────────────────────────┐
│                     Web Browser                              │
│  ┌─────────────────────────────────────────────────────┐    │
│  │              JavaScript (app.js)                     │    │
│  │  • CIDashboard class manages all state              │    │
│  │  • EventSource for SSE connection                    │    │
│  │  • Dynamic DOM rendering for events/pipeline        │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
                              │
                              │ SSE (Server-Sent Events)
                              │ HTTP API
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                     Go Server (main.go)                      │
│  ┌─────────────────────────────────────────────────────┐    │
│  │              sseEventSink                            │    │
│  │  • Implements engine.EventHandler                    │    │
│  │  • Broadcasts events to all SSE clients              │    │
│  │  • Manages client connections                        │    │
│  └─────────────────────────────────────────────────────┘    │
│  ┌─────────────────────────────────────────────────────┐    │
│  │              HTTP Handlers                           │    │
│  │  • /api/events - SSE endpoint                        │    │
│  │  • /api/run    - Start pipeline execution            │    │
│  │  • /api/cancel - Cancel running pipeline             │    │
│  │  • /api/status - Check current run status            │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
                              │
                              │ engine.EventHandler
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                     CI Engine                                │
│  ┌─────────────────────────────────────────────────────┐    │
│  │              engine.Executor                         │    │
│  │  • Executes Plan with worker pool                   │    │
│  │  • Emits events via EventSink                        │    │
│  │  • Returns RunSummary                                │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

## Running the Server

```bash
# From the web directory
cd wip/examples/clients/web
go run .

# Or build and run
go build -o ci-dashboard
./ci-dashboard
```

The server starts on `http://localhost:8080` by default.

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/` | GET | Serves the main dashboard HTML |
| `/static/*` | GET | Serves static assets (CSS, JS) |
| `/api/events` | GET | SSE endpoint for real-time events |
| `/api/run` | POST | Start a new pipeline run |
| `/api/run?fail=true` | POST | Start a run with simulated failure |
| `/api/cancel` | POST | Cancel the currently running pipeline |
| `/api/status` | GET | Check if a pipeline is running |

## Event Types

The SSE stream sends JSON events:

### Plan Event

Sent at the start of a run with pipeline metadata:

```json
{
  "type": "plan",
  "targets": [{"name": "build", "desc": "...", "deps": ["lint"]}],
  "stages": [{"index": 0, "targets": ["checkout"]}]
}
```

### Target Events

```json
{"type": "started", "time": "...", "targetName": "build"}
{"type": "completed", "time": "...", "targetName": "build", "result": {...}}
{"type": "skipped", "time": "...", "targetName": "deploy"}
```

### Summary Event

Sent when the run completes:

```json
{
  "type": "summary",
  "failed": false,
  "results": {"build": {"durationMs": 450, ...}}
}
```

## Demo Pipeline

The included demo pipeline simulates a typical CI/CD workflow:

```text
Stage 0: checkout
Stage 1: install-deps
Stage 2: lint, format-check, unit-tests
Stage 3: build, integration-tests
Stage 4: security-scan, package
Stage 5: docker-build
Stage 6: push-registry
Stage 7: deploy-staging
Stage 8: smoke-tests
Stage 9: all
```

## Screenshot

The dashboard features:

- Dark theme with GitHub-inspired styling
- Real-time connection status indicator
- Pipeline graph with stages and targets
- Animated status indicators for running targets
- Scrollable event log with timestamps
- Comprehensive run summary with statistics

## Dependencies

- Go 1.21+ (for `embed` directive and generics)
- No external Go dependencies (stdlib only)
- No build tools required for frontend (vanilla JS/CSS)
