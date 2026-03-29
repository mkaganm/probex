<div align="center">

<br>

<h1>
<code>probex</code>
</h1>

<p><strong>Zero-Test API Intelligence Engine</strong></p>

<p>
<a href="https://github.com/mkaganm/probex/actions/workflows/ci.yml"><img src="https://github.com/mkaganm/probex/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
<a href="https://go.dev/"><img src="https://img.shields.io/github/go-mod/go-version/mkaganm/probex?filename=core%2Fgo.mod&style=flat-square" alt="Go"></a>
<a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue?style=flat-square" alt="License"></a>
</p>

<p>Point at any API. Get tests. Automatically.</p>

<p>
<a href="docs/getting-started.md">Getting Started</a> &nbsp;&middot;&nbsp;
<a href="docs/architecture.md">Architecture</a> &nbsp;&middot;&nbsp;
<a href="docs/configuration.md">Configuration</a> &nbsp;&middot;&nbsp;
<a href="#sdk-integration">SDKs</a> &nbsp;&middot;&nbsp;
<a href="#contributing">Contributing</a>
</p>

<br>

</div>

```bash
$ probex scan https://api.example.com
  Discovered 47 endpoints (OpenAPI + crawl + wordlist)

$ probex run
  Generated 312 tests across 9 categories
  ✓ 298 passed  ✗ 8 failed  ⚠ 6 warnings
```

<br>

## Why?

Most API testing tools require you to write tests. PROBEX doesn't.

| | Traditional | PROBEX |
|:---|:---|:---|
| **Setup** | Hours writing test code | `probex scan` + `probex run` |
| **Coverage** | Only what you remember to test | OWASP Top 10, edge cases, fuzzing, concurrency |
| **Maintenance** | Tests break when API changes | Re-scan — tests regenerate |
| **AI** | Manual prompt engineering | Built-in scenario gen, security analysis, NL-to-test |
| **Protocols** | REST only | REST, GraphQL, WebSocket, gRPC |

<br>

## How it works

```
  SCAN             LEARN            GENERATE          RUN              WATCH
  ────             ─────            ────────          ───              ─────
  OpenAPI specs    Traffic replay   Happy path        Concurrent       Anomaly detection
  Link crawling    Schema inference Edge cases        Assertions       Schema drift
  Wordlist probe   Auth detection   OWASP security    HTML/XML/JSON    Regression alerts
  GraphQL intro    Pattern mining   Mutation fuzzing   CI/CD gating     Webhook/Slack
  WebSocket probe  Baseline build   Race conditions   Dashboard        Continuous
  gRPC reflection                   AI scenarios
```

> *The best test is the one never written.*

<br>

## Quick start

```bash
# Install from source
git clone https://github.com/mkaganm/probex.git
cd probex && make build

# Or use Docker
docker run probex/probex scan https://api.example.com
```

```bash
probex scan https://api.example.com          # Discover endpoints
probex run                                    # Generate & run tests
probex report --format html -o report.html   # HTML report
probex run --ai                               # AI-augmented testing
probex test "admins only on /admin/*"        # Natural language tests
```

<br>

## Architecture

```
┌──────────────────────────────────────────────────────┐
│                 PROBEX Server (:9712)                 │
│                                                      │
│   ┌───────────┐          ┌──────────────┐            │
│   │  Go CLI   │ ───────▶ │ Python Brain │            │
│   │  probex   │ ◀─────── │ FastAPI      │            │
│   └─────┬─────┘          └──────────────┘            │
│         │                       ▲                    │
│    /api/v1/*               /api/v1/ai/*              │
│         │                       │                    │
└─────────┼───────────────────────┼────────────────────┘
          │                       │
   ┌──────┼──────────┬────────────┼──────┐
   ▼      ▼          ▼            ▼      ▼
 ┌────┐ ┌──────┐ ┌────────┐ ┌────────────┐
 │ JS │ │ Java │ │ Kotlin │ │  VS Code   │
 │ SDK│ │ SDK  │ │  SDK   │ │ Extension  │
 └────┘ └──────┘ └────────┘ └────────────┘
```

| Component | Path | Stack |
|:----------|:-----|:------|
| **Core CLI** | `core/` | Go &mdash; Cobra CLI, 9 test generators, concurrent runner, REST server |
| **AI Brain** | `brain/` | Python &mdash; FastAPI, Ollama (local) or Claude (cloud) |
| **JS/TS SDK** | `sdk-js/` | TypeScript &mdash; npm client, Jest/Vitest plugins, GitHub Action |
| **Java SDK** | `sdk-java/` | Java 17 &mdash; SDK, JUnit 5 extension, Maven & Gradle plugins |
| **Kotlin SDK** | `sdk-kotlin/` | Kotlin &mdash; coroutine client, DSL, JUnit helper |

<br>

## Commands

| Command | What it does |
|:--------|:-------------|
| `probex scan <url>` | Discover endpoints via OpenAPI, crawl, wordlist, GraphQL, WebSocket, gRPC |
| `probex run [--ai]` | Generate and execute tests &mdash; `--ai` adds AI-powered scenarios |
| `probex test "<text>"` | Generate tests from natural language description |
| `probex watch` | Continuous monitoring &mdash; anomaly detection, schema drift, alerts |
| `probex guard` | CI/CD quality gate with severity-based exit codes |
| `probex learn --from-traffic *.har` | Learn API patterns from captured HTTP traffic |
| `probex report --format html` | Generate reports in JSON, JUnit XML, or HTML |
| `probex proxy <url>` | Reverse proxy that captures live traffic |
| `probex discover ./infra` | Find endpoints in Terraform, Kubernetes, Docker Compose files |
| `probex graph` | Visualize endpoint relationships (ASCII or Graphviz DOT) |
| `probex collective push\|pull` | Share and pull anonymous community test patterns |
| `probex config init` | Generate default `probex.yaml` |
| `probex serve [--ai]` | Start REST API server for SDK integration |

<br>

## Test categories

PROBEX generates tests across **9 categories** without any configuration:

| Category | Coverage |
|:---------|:---------|
| **Happy Path** | Status codes, schema validation, response time thresholds |
| **Edge Cases** | Empty bodies, missing fields, wrong types, boundary values |
| **Security** | OWASP API Top 10 &mdash; BOLA, broken auth, mass assignment, SSRF, injection |
| **Fuzzing** | Mutation-based payloads, special characters, type confusion |
| **Relationships** | CRUD lifecycle, cascade deletes, referential integrity |
| **Concurrency** | Race conditions, idempotency, double-submit prevention |
| **GraphQL** | Introspection leaks, depth attacks, batch/alias abuse |
| **WebSocket** | Handshake validation, CSWSH, message auth |
| **gRPC** | Reflection exposure, streaming abuse, content-type enforcement |

<br>

## AI features

```bash
probex serve --ai                        # Start with managed Ollama brain
probex serve --ai-url http://brain:9711  # Connect to external brain
```

| Feature | Endpoint | Description |
|:--------|:---------|:------------|
| **Scenario Generation** | `POST /api/v1/ai/scenarios` | Generate test scenarios from endpoint specifications |
| **Security Analysis** | `POST /api/v1/ai/security` | OWASP-focused analysis with remediation advice |
| **NL-to-Test** | `POST /api/v1/ai/nl-to-test` | Convert plain English to executable test cases |
| **Anomaly Classification** | `POST /api/v1/ai/anomaly` | Classify runtime anomalies with severity scoring |

**Providers:** [Ollama](https://ollama.ai) (local, private, free) or [Anthropic Claude](https://anthropic.com) (cloud, higher quality).

<br>

## SDK integration

<table>
<tr>
<th>JavaScript / TypeScript</th>
<th>Java</th>
<th>Kotlin</th>
</tr>
<tr>
<td>

```javascript
import { ProbexClient } from '@probex/sdk';

const client = new ProbexClient();
const results = await client.run();

// AI-powered scenarios
const scenarios = await client.aiScenarios({
  endpoints,
  max_scenarios: 10,
});
```

</td>
<td>

```java
var client = new ProbexClient();
var result = client.run();

// AI-powered scenarios
var scenarios = client.aiScenarios(
  new ScenarioRequest(endpoints, 10)
);
```

</td>
<td>

```kotlin
val client = ProbexClient()
val result = client.run()

// AI-powered scenarios
val scenarios = client.aiScenarios(
  ScenarioRequest(endpoints, 10)
)
client.close()
```

</td>
</tr>
</table>

Docs: [JS/TS](docs/sdk-js.md) &nbsp;&middot;&nbsp; [Java](docs/sdk-java.md) &nbsp;&middot;&nbsp; [Kotlin](docs/sdk-kotlin.md)

<br>

## CI/CD

**GitHub Actions:**

```yaml
- uses: probex/action@v1
  with:
    target-url: https://staging-api.example.com
    fail-on: critical,high
```

**Any CI:**

```bash
probex scan $API_URL
probex guard --fail-on critical,high --report-file results.xml
# Exit 0 = pass, 1 = threshold exceeded, 2 = error
```

<br>

## Configuration

```bash
probex config init   # Creates probex.yaml with sensible defaults
```

<details>
<summary>Example <code>probex.yaml</code></summary>
<br>

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
  mode: offline   # offline | local | cloud | hybrid
  local:
    provider: ollama
    model: qwen3:4b
report:
  format: json
  output: stdout
```

</details>

<br>

## Documentation

| | |
|:--|:--|
| [Getting Started](docs/getting-started.md) | Installation, first scan, quick start |
| [Architecture](docs/architecture.md) | System design, REST API reference, pipelines |
| [Configuration](docs/configuration.md) | `probex.yaml` reference, AI modes, environment variables |
| [Security Testing](docs/security.md) | OWASP API Top 10 test coverage |
| [Plugins](docs/plugins.md) | Go interface and HTTP JSON-RPC external plugins |
| [IaC Discovery](docs/iac-discovery.md) | Terraform, Kubernetes, Docker Compose scanning |
| [Collective Intelligence](docs/collective.md) | Anonymous community pattern sharing |

<br>

## Contributing

1. Fork the repo
2. Create your branch &mdash; `git checkout -b feat/something`
3. Commit &mdash; `git commit -m 'feat: add something'`
4. Push &mdash; `git push origin feat/something`
5. Open a Pull Request

**Development:**

```bash
cd core && go test ./...                    # Go tests (220 tests)
cd brain && pip install -e ".[dev]" && pytest  # Python tests (42 tests)
cd sdk-js && npm install && npx tsc --noEmit   # JS typecheck
cd sdk-java && mvn test                        # Java tests
make test-all                                  # Everything
```

<br>

## License

[MIT](LICENSE)

<br>

<div align="center">
<sub>Built with Go, Python, and a stubborn belief that APIs should test themselves.</sub>
</div>
