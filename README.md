<div align="center">

# PROBEX

### Zero-Test API Intelligence Engine

[![CI](https://github.com/mkaganm/probex/actions/workflows/ci.yml/badge.svg)](https://github.com/mkaganm/probex/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/mkaganm/probex?filename=core%2Fgo.mod)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/mkaganm/probex?include_prereleases)](https://github.com/mkaganm/probex/releases)

**Discover, learn, and autonomously test your APIs. No test code needed.**

[Getting Started](docs/getting-started.md) &bull; [Architecture](docs/architecture.md) &bull; [Configuration](docs/configuration.md) &bull; [SDKs](#sdk-integration) &bull; [Contributing](#contributing)

</div>

---

```bash
probex scan https://api.example.com
probex run
# Done. 60+ tests generated and executed automatically.
```

## Why PROBEX?

| | Traditional API Testing | PROBEX |
|---|---|---|
| **Setup time** | Hours/days writing test code | Seconds — just point at your API |
| **Coverage** | What you think to test | OWASP Top 10, edge cases, fuzzing, concurrency — automatically |
| **Maintenance** | Tests break when API changes | Re-scan and tests adapt |
| **AI** | Manual prompt engineering | Built-in AI scenarios, security analysis, NL-to-test |
| **Protocols** | Usually REST only | REST, GraphQL, WebSocket, gRPC |

## How It Works

```
  Scan          Learn          Generate        Run           Watch
  ─────         ─────          ────────        ───           ─────
  OpenAPI       Traffic        Happy path      Concurrent    Anomaly
  Crawling      Schemas        Edge cases      Assertions    Schema drift
  Wordlists     Auth detect    Security        Reports       Regression
  GraphQL       Patterns       Fuzzing         CI/CD gate    Alerts
  WebSocket     Baselines      Concurrency     Dashboard     Webhooks
  gRPC                         AI scenarios
```

> **"En iyi test, hic yazilmayan testtir."** — The best test is the one never written.

## Quick Start

### Install

```bash
# From source
git clone https://github.com/mkaganm/probex.git
cd probex && make build
# Binary at ./bin/probex

# Docker
docker run probex/probex scan https://api.example.com
```

### First Run

```bash
# 1. Discover endpoints
probex scan https://api.example.com

# 2. Run auto-generated tests
probex run

# 3. Generate HTML report
probex report --format html --output report.html

# 4. AI-powered testing (requires Python brain)
probex run --ai

# 5. Natural language test generation
probex test "non-admin users should not access /admin endpoints"
```

## Architecture

```
┌──────────────────────────────────────────────────────────┐
│                    PROBEX Server (:9712)                  │
│                                                          │
│  ┌─────────────┐     ┌──────────────┐                   │
│  │   Go CLI    │────▶│ Python Brain │  AI-powered        │
│  │   probex    │◀────│  FastAPI     │  analysis           │
│  └──────┬──────┘     └──────────────┘                   │
│         │                   ▲                            │
│         │  Core API         │ AI proxy                   │
│         │  /api/v1/*        │ /api/v1/ai/*               │
│         │                   │                            │
└─────────┼───────────────────┼────────────────────────────┘
          │                   │
     ┌────┼────────────┬──────┼───────┐
     ▼    ▼            ▼      ▼       ▼
  ┌────┐┌──────┐┌──────────┐┌────────────┐
  │ JS ││ Java ││  Kotlin  ││  VS Code   │
  │ SDK││ SDK  ││   SDK    ││ Extension  │
  └────┘└──────┘└──────────┘└────────────┘
```

| Component | Directory | Description |
|-----------|-----------|-------------|
| Go Core | `core/` | CLI, scanner, 9 generators, runner, server, dashboard |
| Python Brain | `brain/` | AI analysis via Ollama (local) or Claude (cloud) |
| JS/TS SDK | `sdk-js/` | npm package, Jest/Vitest plugins, GitHub Actions |
| Java SDK | `sdk-java/` | Maven SDK, JUnit 5 extension, Maven/Gradle plugins |
| Kotlin SDK | `sdk-kotlin/` | Coroutine client with DSL |

## CLI Commands

| Command | Description |
|:--------|:------------|
| `probex scan <url>` | Discover API endpoints (OpenAPI, crawl, GraphQL, WebSocket, gRPC) |
| `probex run [--ai]` | Generate and execute tests; `--ai` adds AI-powered scenarios |
| `probex test "<description>"` | AI natural language test generation |
| `probex watch` | Continuous monitoring for anomalies and schema drift |
| `probex guard` | CI/CD gate with severity-based exit codes |
| `probex learn --from-traffic *.har` | Learn patterns from captured traffic |
| `probex report --format html` | Generate reports (JSON, JUnit XML, HTML) |
| `probex proxy <url>` | Reverse proxy for live traffic capture |
| `probex discover ./infra` | Discover endpoints from Terraform, K8s, Docker Compose |
| `probex graph` | Visualize endpoint relationships (ASCII, Graphviz DOT) |
| `probex collective push\|pull` | Anonymous community pattern sharing |
| `probex config init` | Initialize `probex.yaml` configuration |
| `probex serve [--ai]` | Start REST API server for SDK integration |

## Test Categories

PROBEX generates tests across **9 categories** automatically:

| Category | What It Tests |
|:---------|:-------------|
| **Happy Path** | Status codes, schema validation, response times |
| **Edge Cases** | Empty bodies, missing fields, wrong types, boundary values |
| **Security** | OWASP API Top 10 — BOLA, broken auth, mass assignment, SSRF, injection |
| **Fuzzing** | Mutation-based, special characters, type confusion |
| **Relationships** | CRUD cycles, cascade behavior, referential integrity |
| **Concurrency** | Race conditions, idempotency, double-submit |
| **GraphQL** | Introspection, depth limiting, batch/alias attacks |
| **WebSocket** | Handshake validation, CSWSH, auth checks |
| **gRPC** | Reflection security, streaming, content-type validation |

## AI Features

Start the server with AI support to unlock intelligent testing:

```bash
probex serve --ai                        # Managed brain subprocess (Ollama)
probex serve --ai-url http://brain:9711  # External brain instance
```

| Feature | Endpoint | Description |
|:--------|:---------|:------------|
| Scenario Generation | `POST /api/v1/ai/scenarios` | AI generates test scenarios from endpoint specs |
| Security Analysis | `POST /api/v1/ai/security` | Deep OWASP analysis with remediation advice |
| NL-to-Test | `POST /api/v1/ai/nl-to-test` | "Test that non-admins can't access /admin" |
| Anomaly Classification | `POST /api/v1/ai/anomaly` | AI classifies runtime anomalies |

Supports **Ollama** (local, private) and **Anthropic Claude** (cloud) as AI providers.

## SDK Integration

<table>
<tr>
<td><b>JavaScript/TypeScript</b></td>
<td><b>Java</b></td>
<td><b>Kotlin</b></td>
</tr>
<tr>
<td>

```javascript
import { ProbexClient } from '@probex/sdk';

const client = new ProbexClient();
const results = await client.run();
console.log(`${results.passed} passed`);

// AI scenarios
const ai = await client.aiScenarios({
  endpoints,
  max_scenarios: 10,
});
```

</td>
<td>

```java
var client = new ProbexClient();
var result = client.run();
assert result.isSuccess();

// AI scenarios
var scenarios = client.aiScenarios(
  new ScenarioRequest(endpoints, 10)
);
```

</td>
<td>

```kotlin
val client = ProbexClient()
val result = client.run()
assert(result.isSuccess)

// AI scenarios
val scenarios = client.aiScenarios(
  ScenarioRequest(endpoints, 10)
)
client.close()
```

</td>
</tr>
</table>

> See full docs: [JS/TS SDK](docs/sdk-js.md) | [Java SDK](docs/sdk-java.md) | [Kotlin SDK](docs/sdk-kotlin.md)

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

### Guard Exit Codes

| Code | Meaning |
|:-----|:--------|
| `0` | All tests passed |
| `1` | Failures at or above threshold severity |
| `2` | Scan or execution error |

## Configuration

```bash
probex config init  # Creates probex.yaml
```

<details>
<summary><b>Example probex.yaml</b></summary>

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
  mode: offline  # offline | local | cloud | hybrid
  local:
    provider: ollama
    model: qwen3:4b
report:
  format: json
  output: stdout
```

</details>

## Documentation

| Document | Description |
|:---------|:------------|
| [Getting Started](docs/getting-started.md) | Installation, first scan, quick start |
| [Architecture](docs/architecture.md) | System design, REST API, pipelines |
| [Configuration](docs/configuration.md) | `probex.yaml` reference, AI modes, env vars |
| [Security Testing](docs/security.md) | OWASP API Top 10 coverage |
| [JS/TS SDK](docs/sdk-js.md) | npm client, Jest/Vitest, GitHub Actions |
| [Java SDK](docs/sdk-java.md) | Maven/Gradle, JUnit 5, plugins |
| [Kotlin SDK](docs/sdk-kotlin.md) | Coroutine client, DSL, JUnit helper |
| [Plugins](docs/plugins.md) | Go interface + HTTP JSON-RPC plugins |
| [IaC Discovery](docs/iac-discovery.md) | Terraform, K8s, Docker Compose scanning |
| [Collective Intelligence](docs/collective.md) | Anonymous community patterns |

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch (`git checkout -b feat/my-feature`)
3. Commit your changes (`git commit -m 'feat: add my feature'`)
4. Push to the branch (`git push origin feat/my-feature`)
5. Open a Pull Request

### Development Setup

```bash
# Go core
cd core && go test ./...

# Python brain
cd brain && pip install -e ".[dev]" && pytest

# JS SDK
cd sdk-js && npm install && npx tsc --noEmit

# Java SDK
cd sdk-java && mvn test

# All at once
make test-all
```

## License

[MIT](LICENSE) &mdash; use it however you want.

---

<div align="center">

**[probex.dev](https://github.com/mkaganm/probex)** &bull; Built with Go, Python, and a lot of API curiosity.

</div>
