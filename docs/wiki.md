# Project Wiki

This page is the central index for PROBEX documentation.

## Start Here

- [Getting Started](getting-started.md) — Installation, first scan, and first test run
- [Architecture](architecture.md) — Core components, REST API, and system flow
- [Configuration](configuration.md) — Full `probex.yaml` and environment settings

## Testing and Security

- [Security Testing](security.md) — OWASP API Top 10 coverage
- [Plugins](plugins.md) — Internal and external plugin integration
- [IaC Discovery](iac-discovery.md) — Endpoint discovery from Terraform/Kubernetes/Docker Compose
- [Collective Intelligence](collective.md) — Shared community patterns

## SDKs

- [JavaScript / TypeScript SDK](sdk-js.md)
- [Java SDK](sdk-java.md)
- [Kotlin SDK](sdk-kotlin.md)

## Common Workflows

### Discover and test an API

```bash
probex scan https://api.example.com
probex run
```

### Run security-focused checks

```bash
probex run --category security
```

### Generate a CI-friendly report

```bash
probex report --format junit --output results.xml
```
