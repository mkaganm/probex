# Security Testing

PROBEX automatically generates security tests based on the OWASP API Security Top 10.

## Coverage

| # | OWASP Category | PROBEX Tests |
|---|----------------|--------------|
| API1 | Broken Object Level Authorization (BOLA) | Access other user's resources via ID manipulation |
| API2 | Broken Authentication | Missing/expired/weak tokens, brute force |
| API3 | Broken Object Property Level Authorization | Mass assignment via extra fields in request body |
| API4 | Unrestricted Resource Consumption | Large payloads, missing rate limits |
| API5 | Broken Function Level Authorization (BFLA) | Non-admin access to admin endpoints |
| API6 | Server Side Request Forgery (SSRF) | Internal URL payloads in request fields |
| API7 | Security Misconfiguration | CORS headers, verbose errors, stack traces |
| API8 | Injection | SQL injection, XSS, path traversal payloads |
| API9 | Improper Inventory Management | — (detected via scan phase) |
| API10 | Unsafe Consumption of APIs | — (requires manual configuration) |

## Protocol-Specific Security

### GraphQL
- **Introspection exposure** — Introspection should be disabled in production
- **Depth limit** — Deeply nested queries can cause DoS
- **Batch/alias attacks** — Multiple operations in a single request

### WebSocket
- **CSWSH** — Cross-Site WebSocket Hijacking via Origin header validation
- **Auth bypass** — WebSocket connections without proper authentication
- **Invalid upgrade** — Non-WebSocket requests to WS endpoints

### gRPC
- **Reflection exposure** — Server reflection should be disabled in production
- **Auth bypass** — Requests without credentials
- **Invalid content type** — Non-gRPC content types

## Running Security Tests Only

```bash
probex run --category security
```

## Severity Levels

| Severity | Description | CI/CD Action |
|----------|-------------|--------------|
| `critical` | Exploitable vulnerability with high impact | Block deployment |
| `high` | Significant security issue | Block deployment |
| `medium` | Security concern, should be addressed | Warning |
| `low` | Minor issue, best practice | Informational |

## CI/CD Integration

```bash
# Fail build on critical and high findings
probex guard --fail-on critical,high

# Generate JUnit XML for CI reporting
probex report --format junit --output security-results.xml
```

## AI-Enhanced Security Analysis

With the AI brain enabled, PROBEX can:
- Generate more sophisticated attack scenarios
- Provide contextual remediation advice
- Classify anomalies with higher accuracy

```bash
probex run --ai --category security
```
