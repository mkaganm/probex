# Getting Started

This guide walks you through installing PROBEX and running your first automated API test suite.

## Prerequisites

- Go 1.23+ (for building from source)
- An API to test (local or remote)

Optional:
- Python 3.12+ (for AI brain features)
- Docker (for containerized usage)

## Installation

### Build from Source

```bash
git clone https://github.com/probex/probex.git
cd probex
make build
```

The binary is written to `./bin/probex`. Add it to your PATH:

```bash
export PATH="$PATH:$(pwd)/bin"
```

### Docker

```bash
docker pull probex/probex:latest
docker run probex/probex scan https://api.example.com
```

### Verify Installation

```bash
probex --version
# probex version 1.0.0
```

## First Scan

Point PROBEX at any HTTP API:

```bash
probex scan https://jsonplaceholder.typicode.com
```

PROBEX will:
1. Look for an OpenAPI/Swagger spec
2. Crawl discovered links
3. Probe common API paths from its wordlist
4. Detect GraphQL, WebSocket, and gRPC endpoints
5. Identify authentication mechanisms

The result is saved as an **API profile** in `.probex/profile.json`.

## Run Tests

```bash
probex run
```

PROBEX reads the saved profile and automatically generates test cases across multiple categories:
- Happy path validation
- Edge case testing
- Security checks (OWASP API Top 10)
- Fuzzing
- Concurrency testing

Results are displayed in the terminal and saved to `.probex/`.

## Filter by Category

Run only specific test categories:

```bash
probex run --category security
probex run --category security,edge_case,fuzz
```

## Generate Reports

```bash
# HTML report for humans
probex report --format html --output report.html

# JUnit XML for CI/CD
probex report --format junit --output results.xml

# JSON for programmatic consumption
probex report --format json --output results.json
```

## CI/CD Guard Mode

Use `guard` in CI pipelines to fail builds on findings:

```bash
probex scan $API_URL
probex guard --fail-on critical,high
# Exit code 1 if critical/high severity findings exist
```

## Configuration File

Initialize a configuration file for persistent settings:

```bash
probex config init
# Creates probex.yaml in the current directory
```

Edit `probex.yaml` to customize scan depth, concurrency, timeouts, AI settings, and more. See [Configuration](configuration.md) for the full reference.

## AI-Powered Testing

Enable AI features for smarter test generation:

```bash
# Install the Python brain
cd brain && pip install -e .

# Run with AI enabled
probex run --ai
```

Or use natural language to describe tests:

```bash
probex test "users with expired tokens should get 401 on all endpoints"
```

See [Architecture](architecture.md) for details on how the AI brain works.

## Next Steps

- [Architecture](architecture.md) — Understand how PROBEX works internally
- [Security Testing](security.md) — OWASP API Top 10 coverage details
- [Configuration](configuration.md) — Full configuration reference
- [SDK Integration](sdk-js.md) — Integrate PROBEX into your test suites
