package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mkaganm/probex/internal/scanner/iac"
)

func TestIaCTerraformDiscovery(t *testing.T) {
	dir := t.TempDir()

	tf := `
resource "aws_apigatewayv2_route" "get_users" {
  api_id    = aws_apigatewayv2_api.main.id
  route_key = "GET /users"
  target    = "integrations/${aws_apigatewayv2_integration.lambda.id}"
}

resource "aws_apigatewayv2_route" "post_users" {
  api_id    = aws_apigatewayv2_api.main.id
  route_key = "POST /users"
  target    = "integrations/${aws_apigatewayv2_integration.lambda.id}"
}

resource "aws_apigatewayv2_route" "get_orders" {
  api_id    = aws_apigatewayv2_api.main.id
  route_key = "GET /orders/{id}"
  target    = "integrations/${aws_apigatewayv2_integration.lambda.id}"
}
`
	if err := os.WriteFile(filepath.Join(dir, "api.tf"), []byte(tf), 0644); err != nil {
		t.Fatal(err)
	}

	scanner := iac.New(dir)
	result, err := scanner.Scan()
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	if len(result.Endpoints) != 3 {
		t.Errorf("expected 3 endpoints, got %d", len(result.Endpoints))
	}

	// Verify first endpoint.
	found := false
	for _, ep := range result.Endpoints {
		if ep.Method == "GET" && ep.Path == "/users" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected GET /users endpoint")
	}
}

func TestIaCDockerComposeDiscovery(t *testing.T) {
	dir := t.TempDir()

	compose := `{
  "services": {
    "api": {
      "image": "myapi:latest",
      "ports": ["8080:8080", "9090:9090"]
    },
    "worker": {
      "image": "worker:latest"
    }
  }
}`
	if err := os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(compose), 0644); err != nil {
		t.Fatal(err)
	}

	scanner := iac.New(dir)
	result, err := scanner.Scan()
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	if len(result.Endpoints) != 2 {
		t.Errorf("expected 2 endpoints from docker-compose ports, got %d", len(result.Endpoints))
	}
}

func TestIaCKubernetesIngressDiscovery(t *testing.T) {
	dir := t.TempDir()

	ingress := `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: api-ingress
spec:
  rules:
    - host: api.example.com
      http:
        paths:
          - path: "/api/v1/users"
            pathType: Prefix
            backend:
              service:
                name: user-service
                port:
                  number: 8080
          - path: "/api/v1/orders"
            pathType: Prefix
            backend:
              service:
                name: order-service
                port:
                  number: 8081
`
	if err := os.WriteFile(filepath.Join(dir, "ingress.yaml"), []byte(ingress), 0644); err != nil {
		t.Fatal(err)
	}

	scanner := iac.New(dir)
	result, err := scanner.Scan()
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	if len(result.Endpoints) < 2 {
		t.Errorf("expected at least 2 ingress endpoints, got %d", len(result.Endpoints))
	}
}

func TestIaCEmptyDirectory(t *testing.T) {
	dir := t.TempDir()

	scanner := iac.New(dir)
	result, err := scanner.Scan()
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	if len(result.Endpoints) != 0 {
		t.Errorf("expected 0 endpoints, got %d", len(result.Endpoints))
	}
}
