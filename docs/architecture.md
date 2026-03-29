# Architecture

PROBEX is a polyglot monorepo with a Go core, Python AI brain, and SDKs in multiple languages.

## System Overview

```
                    ┌──────────────────────────────────┐
                    │         PROBEX CLI (Go)           │
                    │                                    │
                    │  ┌────────┐ ┌──────┐ ┌─────────┐ │
                    │  │Scanner │ │Runner│ │Generator│ │
                    │  └────────┘ └──────┘ └─────────┘ │
                    │  ┌────────┐ ┌──────┐ ┌─────────┐ │
                    │  │ Watch  │ │Guard │ │Dashboard│ │
                    │  └────────┘ └──────┘ └─────────┘ │
                    │  ┌────────┐ ┌──────┐ ┌─────────┐ │
                    │  │ Proxy  │ │Plugin│ │  Graph  │ │
                    │  └────────┘ └──────┘ └─────────┘ │
                    └───────┬──────────┬───────────────┘
                            │          │
               ┌────────────▼──┐  ┌────▼────────────┐
               │  REST API     │  │  Python Brain    │
               │  (port 9712)  │  │  (subprocess)    │
               └───────┬───────┘  └─────────────────┘
                       │
          ┌────────────┼────────────┬──────────────┐
          ▼            ▼            ▼              ▼
     ┌─────────┐ ┌─────────┐ ┌──────────┐ ┌──────────┐
     │  JS SDK │ │Java SDK │ │Kotlin SDK│ │ VS Code  │
     └─────────┘ └─────────┘ └──────────┘ └──────────┘
```

## Go Core (`core/`)

The Go binary is the primary interface. It handles all scanning, test generation, execution, and reporting.

### Package Structure

```
core/
├── main.go                    Entry point
├── cmd/                       CLI commands (Cobra)
│   ├── root.go               Root command, flags
│   ├── scan.go               probex scan
│   ├── run.go                probex run
│   ├── test_nl.go            probex test (NL-to-test)
│   ├── watch.go              probex watch
│   ├── guard.go              probex guard
│   ├── learn.go              probex learn
│   ├── report.go             probex report
│   ├── serve.go              probex serve
│   ├── proxy.go              probex proxy
│   ├── graph.go              probex graph
│   ├── discover.go           probex discover (IaC)
│   ├── collective.go         probex collective
│   └── config.go             probex config
│
├── internal/
│   ├── scanner/              Endpoint discovery
│   │   ├── scanner.go        7-phase scan orchestrator
│   │   ├── openapi.go        OpenAPI/Swagger parser
│   │   ├── crawler.go        Link-following crawler
│   │   ├── wordlist.go       Common path probing
│   │   ├── graphql.go        GraphQL introspection
│   │   ├── websocket.go      WebSocket detection
│   │   ├── grpc.go           gRPC reflection
│   │   └── iac/              IaC file scanning
│   │       └── iac.go        Terraform, K8s, Compose
│   │
│   ├── generator/            Test generation
│   │   ├── engine.go         Generator orchestrator
│   │   ├── happy_path.go     Status code, schema tests
│   │   ├── edge_case.go      Boundary, missing field tests
│   │   ├── security.go       OWASP API Top 10
│   │   ├── fuzzer.go         Mutation-based fuzzing
│   │   ├── relationship.go   CRUD cycle tests
│   │   ├── concurrency.go    Race condition tests
│   │   ├── graphql.go        GraphQL-specific tests
│   │   ├── websocket.go      WebSocket tests
│   │   └── grpc.go           gRPC tests
│   │
│   ├── runner/               Test execution
│   │   ├── executor.go       Concurrent goroutine runner
│   │   ├── assertion.go      Assertion engine
│   │   └── context.go        Variable chaining
│   │
│   ├── watch/                Continuous monitoring
│   │   ├── watcher.go        Poll loop
│   │   ├── anomaly.go        Z-score anomaly detection
│   │   ├── drift.go          Schema drift detection
│   │   └── alerter.go        Notification dispatch
│   │
│   ├── server/               REST API for SDKs
│   │   ├── server.go         HTTP server setup
│   │   └── handlers.go       API endpoint handlers
│   │
│   ├── dashboard/            Embedded web dashboard
│   │   └── dashboard.go      HTML/JS SPA served at /dashboard
│   │
│   ├── proxy/                Traffic capture
│   │   └── proxy.go          Reverse proxy with HAR export
│   │
│   ├── plugin/               Plugin system
│   │   ├── plugin.go         Registry + Go interfaces
│   │   └── grpc_plugin.go    HTTP JSON-RPC external plugins
│   │
│   ├── graph/                Endpoint visualization
│   │   └── graph.go          ASCII + DOT rendering
│   │
│   ├── collective/           Community patterns
│   │   └── collective.go     Anonymized pattern push/pull
│   │
│   ├── ai/                   Python brain bridge
│   │   ├── client.go         HTTP client for brain API
│   │   ├── bridge.go         Subprocess lifecycle
│   │   └── types.go          Shared data types
│   │
│   ├── models/               Data structures
│   │   ├── endpoint.go       Endpoint, Schema, AuthInfo
│   │   ├── profile.go        APIProfile
│   │   ├── testcase.go       TestCase, Assertion
│   │   ├── result.go         TestResult, RunSummary
│   │   └── config.go         Config, defaults
│   │
│   ├── storage/              Persistence
│   │   └── store.go          JSON file storage in .probex/
│   │
│   ├── report/               Report generation
│   │   ├── json.go           JSON output
│   │   ├── junit.go          JUnit XML
│   │   └── html.go           HTML with templates
│   │
│   ├── schema/               Schema inference
│   │   └── inferrer.go       JSON → Schema
│   │
│   ├── auth/                 Auth detection
│   │   └── detector.go       401/403 pattern analysis
│   │
│   └── ui/                   Terminal UI
│       ├── console.go        lipgloss styling
│       ├── progress.go       Progress indicators
│       └── table.go          Table rendering
│
└── test/                     Integration tests
```

### Scan Pipeline

The scanner runs 7 phases sequentially. Each phase is independent — failures in one phase do not block others.

```
Phase 1: OpenAPI     — Look for /openapi.json, /swagger.json, etc.
Phase 2: Crawl       — Follow links from known endpoints
Phase 3: Wordlist    — Probe common API paths (/api/v1/users, /health, etc.)
Phase 4: GraphQL     — Detect and introspect GraphQL endpoints
Phase 5: WebSocket   — Detect WebSocket upgrade endpoints
Phase 6: gRPC        — Detect gRPC-Web and reflect services
Phase 7: Auth        — Detect authentication requirements
```

### Generator Pipeline

The generator engine runs 9 generators against each endpoint. Generators produce `TestCase` structs with HTTP requests and assertions.

### Execution Model

The runner uses a goroutine pool (configurable concurrency) to execute tests. Each test makes a real HTTP request and evaluates assertions against the response.

## Python Brain (`brain/`)

The AI brain is an optional component that provides:

- **Scenario generation** — AI-powered complex multi-step test scenarios
- **Security analysis** — Deeper security finding analysis
- **NL-to-test** — Natural language to test case conversion
- **Anomaly classification** — AI-assisted anomaly triage

It runs as a FastAPI server, started as a subprocess by the Go CLI when `--ai` is used. Supports Ollama (local, private) and Anthropic Claude (cloud) as AI providers.

## REST API

When `probex serve` is running, SDKs communicate via:

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/health` | Server health check (includes `ai_enabled` field) |
| POST | `/api/v1/scan` | Trigger a scan |
| GET | `/api/v1/profile` | Get current API profile |
| POST | `/api/v1/run` | Run tests (`use_ai: true` for AI-augmented runs) |
| GET | `/api/v1/results` | Get latest results |
| GET | `/api/v1/results/{id}` | Get specific run results |
| GET | `/api/v1/ai/health` | AI brain health check (returns 503 if AI not configured) |
| POST | `/api/v1/ai/scenarios` | Generate AI-powered test scenarios |
| POST | `/api/v1/ai/security` | AI security analysis |
| POST | `/api/v1/ai/nl-to-test` | Natural language to test conversion |
| POST | `/api/v1/ai/anomaly` | AI anomaly classification |
| GET | `/dashboard` | Web dashboard |
| GET | `/dashboard/api/summary` | Dashboard summary data |
| GET | `/dashboard/api/runs` | Dashboard run history |

The AI endpoints (`/api/v1/ai/*`) are proxy endpoints — the Go server forwards requests to the Python brain and returns the response. If the brain is not configured, these endpoints return `503 Service Unavailable`. Start the server with AI support using `probex serve --ai` (managed subprocess) or `probex serve --ai-url http://brain:9711` (external brain).

## Plugin System

PROBEX supports two plugin models:

1. **Go interface plugins** — Implement `GeneratorPlugin`, `ReporterPlugin`, or `HookPlugin` interfaces
2. **External HTTP plugins** — Any language, communicate via HTTP JSON-RPC at endpoints `/meta`, `/generate`, `/report`, `/hooks/*`

See [Plugins](plugins.md) for details.

## Storage

All data is stored locally in `.probex/` as JSON files:

```
.probex/
├── profile.json          Current API profile
├── results.json          Latest test results
└── runs/                 Historical run data
    ├── results_20240101_120000.json
    └── results_20240102_150000.json
```
