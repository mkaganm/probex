# IaC Discovery

PROBEX can discover API endpoints from Infrastructure-as-Code files without running a live scan.

## Supported Formats

| Format | Resources Detected |
|--------|--------------------|
| **Terraform** (`.tf`) | API Gateway v1/v2 routes, Lambda function URLs, ALB rules |
| **Kubernetes** (`.yaml/.yml`) | Ingress paths, NodePort/LoadBalancer services |
| **Pulumi** (`.yaml`) | API Gateway routes |
| **Docker Compose** (`.yml/.yaml`) | Exposed ports per service |

## Usage

```bash
# Scan current directory
probex discover .

# Scan a specific directory
probex discover ./infrastructure

# Scan without merging into profile
probex discover ./terraform --merge=false
```

## How It Works

PROBEX walks the directory tree, identifies IaC files by extension and filename, and extracts API endpoint information using pattern matching.

### Terraform

Detects `aws_apigatewayv2_route` resources and extracts `route_key` values:

```hcl
resource "aws_apigatewayv2_route" "get_users" {
  api_id    = aws_apigatewayv2_api.main.id
  route_key = "GET /users"
}
```

Result: `GET /users` endpoint with tags `[iac, terraform, api-gateway]`.

Also detects API Gateway v1 `path_part` / `http_method` combinations and `aws_lambda_function_url` resources.

### Kubernetes

Detects Ingress resources and extracts path rules:

```yaml
kind: Ingress
spec:
  rules:
    - http:
        paths:
          - path: "/api/v1/users"
```

Result: `ANY /api/v1/users` endpoint with tags `[iac, kubernetes, ingress]`.

Also detects NodePort/LoadBalancer Services and extracts ports.

### Docker Compose

Extracts exposed port mappings from service definitions:

```yaml
services:
  api:
    ports:
      - "8080:8080"
```

Result: `ANY /` at `http://localhost:8080` with tags `[iac, docker-compose, api]`.

## Merging

By default, discovered endpoints are merged into the existing API profile. Duplicate endpoints (same method + path) are skipped.

```bash
# First scan live API
probex scan https://api.example.com

# Then enrich with IaC-discovered endpoints
probex discover ./infrastructure
# "Added 5 new endpoints to profile (total: 23)"
```
