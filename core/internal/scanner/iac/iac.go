// Package iac discovers API endpoints from Infrastructure-as-Code files.
//
// Supported formats:
//   - Terraform (HCL): API Gateway, Lambda URL, ALB, CloudFront
//   - Pulumi (YAML/JSON): API Gateway, Function URLs
//   - Kubernetes manifests: Ingress, Service
//   - Docker Compose: exposed ports and service names
package iac

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mkaganm/probex/internal/models"
)

// Scanner discovers API endpoints from IaC configuration files.
type Scanner struct {
	rootDir string
}

// New creates an IaC scanner rooted at the given directory.
func New(rootDir string) *Scanner {
	return &Scanner{rootDir: rootDir}
}

// Discovery holds the result of an IaC scan.
type Discovery struct {
	Endpoints []models.Endpoint `json:"endpoints"`
	Source    string            `json:"source"` // e.g. "terraform", "kubernetes"
	Files     []string          `json:"files"`
}

// Scan walks the directory tree looking for IaC files and extracts endpoints.
func (s *Scanner) Scan() (*Discovery, error) {
	var allEndpoints []models.Endpoint
	var allFiles []string
	var source string

	err := filepath.Walk(s.rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		name := info.Name()
		ext := filepath.Ext(name)

		switch {
		case ext == ".tf":
			eps, err := parseTerraform(path)
			if err == nil && len(eps) > 0 {
				allEndpoints = append(allEndpoints, eps...)
				allFiles = append(allFiles, path)
				source = "terraform"
			}

		case name == "docker-compose.yml" || name == "docker-compose.yaml" || name == "compose.yml" || name == "compose.yaml":
			eps, err := parseDockerCompose(path)
			if err == nil && len(eps) > 0 {
				allEndpoints = append(allEndpoints, eps...)
				allFiles = append(allFiles, path)
				if source == "" {
					source = "docker-compose"
				}
			}

		case name == "Pulumi.yaml" || name == "Pulumi.yml" || ext == ".yaml" || ext == ".yml":
			eps, err := parsePulumiOrK8s(path)
			if err == nil && len(eps) > 0 {
				allEndpoints = append(allEndpoints, eps...)
				allFiles = append(allFiles, path)
				if source == "" {
					source = "pulumi/k8s"
				}
			}
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking directory: %w", err)
	}

	return &Discovery{
		Endpoints: allEndpoints,
		Source:    source,
		Files:     allFiles,
	}, nil
}

// --- Terraform HCL parsing (regex-based, no HCL dependency) ---

var (
	tfRouteKeyRe = regexp.MustCompile(`route_key\s*=\s*"([^"]+)"`)
	tfPathPartRe = regexp.MustCompile(`path_part\s*=\s*"([^"]+)"`)
	tfMethodRe   = regexp.MustCompile(`http_method\s*=\s*"([^"]+)"`)
)

func parseTerraform(path string) ([]models.Endpoint, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	content := string(data)

	var endpoints []models.Endpoint

	// Find API Gateway v2 routes (e.g. "GET /users")
	routeKeys := tfRouteKeyRe.FindAllStringSubmatch(content, -1)
	for _, match := range routeKeys {
		key := match[1]
		if key == "$default" {
			continue
		}
		parts := strings.SplitN(key, " ", 2)
		method, p := "ANY", key
		if len(parts) == 2 {
			method = strings.ToUpper(parts[0])
			p = parts[1]
		}
		endpoints = append(endpoints, models.Endpoint{
			Method: method,
			Path:   p,
			Tags:   []string{"iac", "terraform", "api-gateway"},
			Source: models.SourceIaC,
		})
	}

	// Find API Gateway v1 resources
	pathParts := tfPathPartRe.FindAllStringSubmatch(content, -1)
	methods := tfMethodRe.FindAllStringSubmatch(content, -1)
	for i, pp := range pathParts {
		method := "ANY"
		if i < len(methods) {
			method = strings.ToUpper(methods[i][1])
		}
		endpoints = append(endpoints, models.Endpoint{
			Method: method,
			Path:   "/" + pp[1],
			Tags:   []string{"iac", "terraform", "api-gateway-v1"},
			Source: models.SourceIaC,
		})
	}

	// Lambda function URLs
	if strings.Contains(content, "aws_lambda_function_url") {
		endpoints = append(endpoints, models.Endpoint{
			Method: "ANY",
			Path:   "/",
			Tags:   []string{"iac", "terraform", "lambda-url"},
			Source: models.SourceIaC,
		})
	}

	return endpoints, nil
}

// --- Pulumi / Kubernetes YAML parsing ---

func parsePulumiOrK8s(path string) ([]models.Endpoint, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	content := string(data)

	var endpoints []models.Endpoint

	// K8s Ingress paths
	ingressPathRe := regexp.MustCompile(`path:\s*"?(/[^\s"]+)"?`)
	if strings.Contains(content, "kind: Ingress") || strings.Contains(content, "kind: IngressRoute") {
		matches := ingressPathRe.FindAllStringSubmatch(content, -1)
		for _, m := range matches {
			endpoints = append(endpoints, models.Endpoint{
				Method: "ANY",
				Path:   m[1],
				Tags:   []string{"iac", "kubernetes", "ingress"},
				Source: models.SourceIaC,
			})
		}
	}

	// K8s Service with NodePort/LoadBalancer
	if strings.Contains(content, "type: NodePort") || strings.Contains(content, "type: LoadBalancer") {
		portRe := regexp.MustCompile(`port:\s*(\d+)`)
		nameRe := regexp.MustCompile(`name:\s*(\S+)`)
		ports := portRe.FindAllStringSubmatch(content, -1)
		names := nameRe.FindAllStringSubmatch(content, 5)
		svcName := "service"
		if len(names) > 0 {
			svcName = names[0][1]
		}
		for _, p := range ports {
			endpoints = append(endpoints, models.Endpoint{
				Method:  "ANY",
				Path:    "/",
				BaseURL: fmt.Sprintf("http://%s:%s", svcName, p[1]),
				Tags:    []string{"iac", "kubernetes", "service"},
				Source:  models.SourceIaC,
			})
		}
	}

	// Pulumi apigateway routes
	if strings.Contains(content, "aws:apigateway") || strings.Contains(content, "aws:apigatewayv2") {
		routeRe := regexp.MustCompile(`routeKey:\s*"?([A-Z]+\s+/[^\s"]+)"?`)
		matches := routeRe.FindAllStringSubmatch(content, -1)
		for _, m := range matches {
			parts := strings.SplitN(m[1], " ", 2)
			method, p := "ANY", m[1]
			if len(parts) == 2 {
				method = parts[0]
				p = parts[1]
			}
			endpoints = append(endpoints, models.Endpoint{
				Method: method,
				Path:   p,
				Tags:   []string{"iac", "pulumi", "api-gateway"},
				Source: models.SourceIaC,
			})
		}
	}

	return endpoints, nil
}

// --- Docker Compose parsing (JSON / YAML lightweight) ---

func parseDockerCompose(path string) ([]models.Endpoint, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	content := string(data)

	var endpoints []models.Endpoint

	// Try JSON first.
	var compose map[string]any
	if err := json.Unmarshal(data, &compose); err == nil {
		return parseComposeMap(compose, path)
	}

	// Fallback: regex-based YAML parsing for ports.
	portRe := regexp.MustCompile(`ports:\s*\n((?:\s+-\s*"?\d+:\d+"?\n?)+)`)
	svcRe := regexp.MustCompile(`(\w[\w-]*):\s*\n`)
	portLineRe := regexp.MustCompile(`"?(\d+):(\d+)"?`)

	services := svcRe.FindAllStringSubmatch(content, -1)
	portBlocks := portRe.FindAllStringSubmatch(content, -1)

	for i, block := range portBlocks {
		svcName := "service"
		if i < len(services) {
			svcName = services[i][1]
		}
		portLines := portLineRe.FindAllStringSubmatch(block[1], -1)
		for _, pl := range portLines {
			hostPort := pl[1]
			endpoints = append(endpoints, models.Endpoint{
				Method:  "ANY",
				Path:    "/",
				BaseURL: fmt.Sprintf("http://localhost:%s", hostPort),
				Tags:    []string{"iac", "docker-compose", svcName},
				Source:  models.SourceIaC,
			})
		}
	}

	return endpoints, nil
}

func parseComposeMap(compose map[string]any, path string) ([]models.Endpoint, error) {
	var endpoints []models.Endpoint

	services, ok := compose["services"].(map[string]any)
	if !ok {
		return nil, nil
	}

	for name, svcRaw := range services {
		svc, ok := svcRaw.(map[string]any)
		if !ok {
			continue
		}
		ports, ok := svc["ports"].([]any)
		if !ok {
			continue
		}
		for _, portRaw := range ports {
			portStr := fmt.Sprintf("%v", portRaw)
			parts := strings.SplitN(portStr, ":", 2)
			hostPort := parts[0]
			if len(parts) == 2 {
				hostPort = parts[0]
			}
			endpoints = append(endpoints, models.Endpoint{
				Method:  "ANY",
				Path:    "/",
				BaseURL: fmt.Sprintf("http://localhost:%s", hostPort),
				Tags:    []string{"iac", "docker-compose", name},
				Source:  models.SourceIaC,
			})
		}
	}

	return endpoints, nil
}
