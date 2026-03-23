# PROBEX — Zero-Test API Intelligence Engine

PROBEX discovers, learns, and autonomously tests your APIs. No test code needed — point it at your API and let it work.

```bash
probex scan https://api.example.com
probex run
# Done. Tests generated and executed automatically.
```

## Documentation

| | |
|---|---|
| [Getting Started](docs/getting-started.md) | Installation, first scan, quick start guide |
| [Architecture](docs/architecture.md) | System design, package structure, scan/generator pipelines |
| [Configuration](docs/configuration.md) | Full `probex.yaml` reference, AI modes, environment variables |
| [Security Testing](docs/security.md) | OWASP API Top 10 coverage, protocol-specific security |
| [SDK — JavaScript/TypeScript](docs/sdk-js.md) | npm client, Jest/Vitest plugins, GitHub Actions |
| [SDK — Java](docs/sdk-java.md) | Maven/Gradle SDK, JUnit 5 extension, Maven/Gradle plugins |
| [SDK — Kotlin](docs/sdk-kotlin.md) | Coroutine client, DSL, JUnit helper |
| [Plugins](docs/plugins.md) | Go interface + HTTP JSON-RPC external plugin system |
| [IaC Discovery](docs/iac-discovery.md) | Terraform, Kubernetes, Docker Compose endpoint discovery |
| [Collective Intelligence](docs/collective.md) | Anonymous community pattern sharing |

## What It Does

1. **Scan** — Discovers endpoints via OpenAPI specs, crawling, wordlists, GraphQL introspection, WebSocket probing, and gRPC reflection
2. **Learn** — Analyzes traffic patterns, infers schemas, detects auth mechanisms
3. **Generate** — Produces tests automatically: happy path, edge cases, security (OWASP API Top 10), fuzzing, concurrency, and more
4. **Run** — Executes tests concurrently with detailed assertions
5. **Watch** — Continuously monitors for anomalies, schema drift, and regressions
6. **Guard** — CI/CD gate that fails builds on critical findings

## Installation

### From Source

```bash
git clone https://github.com/mkaganm/probex.git
cd probex
make build
# Binary at ./bin/probex
```

### Docker

```bash
docker run probex/probex scan https://api.example.com
```

## Quick Start

```bash
# 1. Scan your API
probex scan https://api.example.com

# 2. Run auto-generated tests
probex run

# 3. View results
probex report --format html --output report.html

# 4. Enable AI-powered test generation (requires Python brain)
probex run --ai

# 5. Generate tests from natural language
probex test "non-admin users should not access /admin endpoints"
```

## Architecture

```
┌─────────────┐     ┌──────────────┐
│   Go CLI    │────▶│ Python Brain │  (AI-powered analysis)
│   probex    │◀────│  FastAPI/gRPC │
└──────┬──────┘     └──────────────┘
       │
       │  REST API (localhost:9712)
       │
  ┌────┼────────────┬──────────────┐
  ▼    ▼            ▼              ▼
┌────┐┌──────┐┌──────────┐┌────────────┐
│ JS ││ Java ││  Kotlin  ││  VS Code   │
│ SDK││ SDK  ││   SDK    ││ Extension  │
└────┘└──────┘└──────────┘└────────────┘
```

- **Go Core** (`core/`) — CLI, scanner, generators, runner, server, dashboard
- **Python Brain** (`brain/`) — AI analysis via Ollama (local) or Claude (cloud)
- **JS/TS SDK** (`sdk-js/`) — npm package + GitHub Actions action
- **Java SDK** (`sdk-java/`) — Maven SDK + Maven/Gradle plugins
- **Kotlin SDK** (`sdk-kotlin/`) — Coroutine-based client with DSL
- **VS Code Extension** (`vscode-extension/`) — Sidebar views, commands, endpoint graph

## CLI Commands

| Command | Description |
|---------|-------------|
| `probex scan <url>` | Discover API endpoints |
| `probex run` | Generate and execute tests |
| `probex test "<description>"` | AI-powered natural language test generation |
| `probex watch` | Continuous monitoring for anomalies and drift |
| `probex guard` | CI/CD gate with severity-based exit codes |
| `probex learn --from-traffic file.har` | Learn from captured traffic |
| `probex report --format html` | Generate reports (JSON, JUnit XML, HTML) |
| `probex proxy <url>` | Reverse proxy for live traffic capture |
| `probex discover ./infra` | Discover endpoints from IaC files |
| `probex graph` | Visualize endpoint relationships |
| `probex collective push/pull` | Community pattern sharing |
| `probex config init` | Initialize configuration |
| `probex serve` | Start REST API server for SDK integration |

## Test Categories

PROBEX generates tests across 9 categories:

- **Happy Path** — Status codes, schema validation, response times
- **Edge Cases** — Empty bodies, missing fields, wrong types, boundary values
- **Security** — OWASP API Top 10 (BOLA, broken auth, mass assignment, SSRF, injection, etc.)
- **Fuzzing** — Mutation-based, special characters, type confusion
- **Relationships** — CRUD cycles, cascade behavior, referential integrity
- **Concurrency** — Race conditions, idempotency, double-submit
- **GraphQL** — Introspection, depth limiting, batch/alias attacks
- **WebSocket** — Handshake validation, CSWSH, auth checks
- **gRPC** — Reflection security, streaming, content-type validation

## SDK Integration

### JavaScript/TypeScript

```javascript
import { Probex } from '@probex/sdk';

const client = new Probex('http://localhost:9712');
const results = await client.run({ baseUrl: 'https://api.example.com' });
console.log(`${results.passed} passed, ${results.failed} failed`);
```

### Java

```java
var client = new ProbexClient("http://localhost:9712");
var result = client.run();
assertThat(result.getFailed()).isZero();
```

### Kotlin

```kotlin
val result = probex("http://localhost:9712") {
    scan("https://api.example.com")
    run()
}
assert(result.isSuccess)
```

## Configuration

```bash
probex config init  # Creates probex.yaml with defaults
```

```yaml
version: "1"
target:
  base_url: "https://api.example.com"
scan:
  max_depth: 3
  concurrency: 10
run:
  concurrency: 5
  timeout: 30s
guard:
  fail_on: [critical, high]
ai:
  mode: offline  # offline, local, cloud, hybrid
  local:
    provider: ollama
    model: qwen3:4b
report:
  format: json
  output: stdout
```

## CI/CD Integration

### GitHub Actions

```yaml
- uses: probex/action@v1
  with:
    target-url: https://staging-api.example.com
    fail-on: critical,high
```

### Generic CI

```bash
probex scan $API_URL
probex guard --fail-on critical,high --report-file results.xml
```

## License

MIT
