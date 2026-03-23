package test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mkaganm/probex/internal/generator"
	"github.com/mkaganm/probex/internal/models"
	"github.com/mkaganm/probex/internal/scanner"
)

func TestGraphQLDetection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/graphql" && r.Method == "POST" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"data":{"__typename":"Query"}}`))
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()

	gs := scanner.NewGraphQLScanner(srv.URL)
	if !gs.DetectGraphQL(context.Background()) {
		t.Error("expected GraphQL endpoint to be detected")
	}
}

func TestGraphQLIntrospection(t *testing.T) {
	introspectionResponse := map[string]any{
		"data": map[string]any{
			"__schema": map[string]any{
				"queryType":    map[string]string{"name": "Query"},
				"mutationType": map[string]string{"name": "Mutation"},
				"types": []map[string]any{
					{
						"name": "Query",
						"kind": "OBJECT",
						"fields": []map[string]any{
							{
								"name":        "users",
								"description": "List all users",
								"type":        map[string]any{"kind": "LIST", "name": nil, "ofType": map[string]any{"kind": "OBJECT", "name": "User"}},
								"args":        []map[string]any{},
							},
							{
								"name":        "user",
								"description": "Get user by ID",
								"type":        map[string]any{"kind": "OBJECT", "name": "User"},
								"args": []map[string]any{
									{"name": "id", "type": map[string]any{"kind": "NON_NULL", "name": nil, "ofType": map[string]any{"kind": "SCALAR", "name": "ID"}}, "defaultValue": nil},
								},
							},
						},
					},
					{
						"name": "Mutation",
						"kind": "OBJECT",
						"fields": []map[string]any{
							{
								"name": "createUser",
								"type": map[string]any{"kind": "OBJECT", "name": "User"},
								"args": []map[string]any{
									{"name": "name", "type": map[string]any{"kind": "NON_NULL", "name": nil, "ofType": map[string]any{"kind": "SCALAR", "name": "String"}}, "defaultValue": nil},
									{"name": "email", "type": map[string]any{"kind": "SCALAR", "name": "String"}, "defaultValue": nil},
								},
							},
						},
					},
					{
						"name": "User",
						"kind": "OBJECT",
						"fields": []map[string]any{
							{"name": "id", "type": map[string]any{"kind": "SCALAR", "name": "ID"}, "args": []map[string]any{}},
							{"name": "name", "type": map[string]any{"kind": "SCALAR", "name": "String"}, "args": []map[string]any{}},
						},
					},
				},
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/graphql" && r.Method == "POST" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(introspectionResponse)
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()

	gs := scanner.NewGraphQLScanner(srv.URL)
	endpoints, err := gs.Discover(context.Background())
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	// Should find queries (users, user) + mutations (createUser) = 3.
	if len(endpoints) != 3 {
		t.Errorf("expected 3 endpoints, got %d", len(endpoints))
	}

	// Check that query endpoints have QUERY method.
	queryCount := 0
	mutationCount := 0
	for _, ep := range endpoints {
		if ep.Method == "QUERY" {
			queryCount++
		}
		if ep.Method == "MUTATION" {
			mutationCount++
		}
	}
	if queryCount != 2 {
		t.Errorf("expected 2 queries, got %d", queryCount)
	}
	if mutationCount != 1 {
		t.Errorf("expected 1 mutation, got %d", mutationCount)
	}
}

func TestGraphQLGeneratorOutput(t *testing.T) {
	ep := models.Endpoint{
		Method:  "QUERY",
		Path:    "/graphql",
		BaseURL: "http://localhost:8080",
		Tags:    []string{"graphql", "query"},
		Headers: map[string]string{
			"X-GraphQL-Operation": "users",
			"X-GraphQL-Type":      "query",
		},
		QueryParams: []models.Parameter{
			{Name: "limit", Type: "Int", Required: false},
		},
	}

	gen := generator.NewGraphQL()
	tests, err := gen.Generate(ep)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	if len(tests) < 5 {
		t.Errorf("expected at least 5 GraphQL tests, got %d", len(tests))
	}

	// Should have execution test.
	found := false
	for _, tc := range tests {
		if strings.Contains(tc.Name, "executes successfully") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'executes successfully' test")
	}

	// Should have introspection security test.
	found = false
	for _, tc := range tests {
		if strings.Contains(tc.Name, "introspection") {
			found = true
			if tc.Category != models.CategorySecurity {
				t.Error("introspection test should be security category")
			}
			break
		}
	}
	if !found {
		t.Error("expected introspection security test")
	}

	// All test requests should be POST to /graphql.
	for _, tc := range tests {
		if tc.Request.Method != "POST" {
			t.Errorf("test %q: expected POST method, got %s", tc.Name, tc.Request.Method)
		}
	}
}

func TestGraphQLGeneratorSkipsNonGraphQL(t *testing.T) {
	ep := models.Endpoint{
		Method:  "GET",
		Path:    "/api/users",
		BaseURL: "http://localhost:8080",
	}

	gen := generator.NewGraphQL()
	tests, err := gen.Generate(ep)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(tests) != 0 {
		t.Errorf("expected 0 tests for non-GraphQL endpoint, got %d", len(tests))
	}
}
