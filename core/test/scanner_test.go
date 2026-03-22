package test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/probex/probex/internal/models"
	"github.com/probex/probex/internal/scanner"
)

func TestNewScanner(t *testing.T) {
	s := scanner.New("https://api.example.com", models.ScanOptions{
		MaxDepth:    3,
		Concurrency: 10,
	})
	if s == nil {
		t.Fatal("expected non-nil scanner")
	}
}

func TestScanWithOpenAPISpec(t *testing.T) {
	spec := map[string]any{
		"openapi": "3.0.0",
		"info": map[string]string{
			"title":   "Test API",
			"version": "1.0",
		},
		"paths": map[string]any{
			"/users": map[string]any{
				"get": map[string]any{
					"summary": "List users",
					"responses": map[string]any{
						"200": map[string]any{
							"description": "OK",
							"content": map[string]any{
								"application/json": map[string]any{
									"schema": map[string]any{
										"type": "array",
										"items": map[string]any{
											"type": "object",
											"properties": map[string]any{
												"id":   map[string]string{"type": "integer"},
												"name": map[string]string{"type": "string"},
											},
										},
									},
								},
							},
						},
					},
				},
				"post": map[string]any{
					"summary": "Create user",
					"requestBody": map[string]any{
						"content": map[string]any{
							"application/json": map[string]any{
								"schema": map[string]any{
									"type": "object",
									"properties": map[string]any{
										"name":  map[string]string{"type": "string"},
										"email": map[string]string{"type": "string"},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/openapi.json":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(spec)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	s := scanner.New(server.URL, models.ScanOptions{
		Concurrency: 5,
		FollowLinks: true,
	})

	result, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Endpoints) < 2 {
		t.Errorf("expected at least 2 endpoints from OpenAPI spec, got %d", len(result.Endpoints))
	}

	foundGet := false
	foundPost := false
	for _, ep := range result.Endpoints {
		if ep.Method == "GET" && ep.Path == "/users" {
			foundGet = true
		}
		if ep.Method == "POST" && ep.Path == "/users" {
			foundPost = true
		}
	}

	if !foundGet {
		t.Error("expected GET /users endpoint")
	}
	if !foundPost {
		t.Error("expected POST /users endpoint")
	}
}

func TestScanWordlistDiscovery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		case "/users":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]map[string]any{
				{"id": 1, "name": "Alice"},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	s := scanner.New(server.URL, models.ScanOptions{Concurrency: 5})
	result, err := s.Scan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Endpoints) < 2 {
		t.Errorf("expected at least 2 endpoints, got %d", len(result.Endpoints))
	}
}
