# Changelog

All notable changes to PROBEX are documented in this file.

## [1.0.0] - 2026-03-22

### Added

**Core CLI**
- `probex scan` — 7-phase endpoint discovery (OpenAPI, crawl, wordlist, GraphQL, WebSocket, gRPC, auth detection)
- `probex run` — Auto-generated test execution with 9 generators (happy path, edge case, security, fuzz, relationship, concurrency, GraphQL, WebSocket, gRPC)
- `probex test` — Natural language to test case generation via AI
- `probex watch` — Continuous monitoring with anomaly detection and schema drift
- `probex guard` — CI/CD gate with severity-based exit codes
- `probex learn` — Learn API behavior from HAR traffic files
- `probex report` — Generate reports in JSON, JUnit XML, and HTML formats
- `probex proxy` — Reverse proxy for live traffic capture with HAR export
- `probex discover` — Discover endpoints from IaC files (Terraform, Kubernetes, Docker Compose, Pulumi)
- `probex graph` — Endpoint relationship visualization (ASCII and Graphviz DOT)
- `probex collective push/pull` — Anonymous community pattern sharing
- `probex config init/show` — Configuration management
- `probex serve` — REST API server for SDK integration

**Security Testing**
- OWASP API Security Top 10 coverage (BOLA, broken auth, mass assignment, SSRF, BFLA, injection, misconfiguration)
- GraphQL-specific: introspection exposure, depth limiting, batch/alias attacks
- WebSocket-specific: CSWSH, auth bypass, invalid upgrade
- gRPC-specific: reflection exposure, auth bypass, content-type validation

**AI Brain (Python)**
- AI-powered test scenario generation
- Security analysis with contextual remediation
- Natural language to test case conversion
- Anomaly classification
- Supports Ollama (local) and Anthropic Claude (cloud)

**SDKs**
- JavaScript/TypeScript SDK (`@probex/sdk`) with Jest and Vitest plugins
- GitHub Actions action (`probex/action@v1`)
- Java SDK (`io.probex:probex-sdk`) with JUnit 5 extension
- Maven plugin (`io.probex:probex-maven-plugin`)
- Gradle plugin (`io.probex.gradle`)
- Kotlin SDK (`io.probex:probex-sdk-kotlin`) with coroutine support and DSL

**Infrastructure**
- VS Code extension with endpoint/results sidebar and endpoint graph webview
- Embedded web dashboard at `/dashboard`
- Plugin system (Go interfaces + HTTP JSON-RPC for external plugins)
- Docker images (full with Python brain, core-only minimal)
- Docker Compose for server deployment
- GoReleaser for cross-platform binary releases
- GitHub Actions CI/CD pipeline
