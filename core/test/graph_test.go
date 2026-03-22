package test

import (
	"strings"
	"testing"

	"github.com/probex/probex/internal/graph"
	"github.com/probex/probex/internal/models"
)

func makeTestProfile() *models.APIProfile {
	return &models.APIProfile{
		ID:      "test",
		BaseURL: "http://localhost:8080",
		Endpoints: []models.Endpoint{
			{Method: "GET", Path: "/users"},
			{Method: "POST", Path: "/users"},
			{Method: "GET", Path: "/users/{id}"},
			{Method: "PUT", Path: "/users/{id}"},
			{Method: "DELETE", Path: "/users/{id}"},
			{Method: "GET", Path: "/posts"},
			{Method: "POST", Path: "/posts"},
		},
	}
}

func TestGraphInferEdges(t *testing.T) {
	profile := makeTestProfile()
	g := graph.New(profile)
	g.InferEdges()

	ascii := g.RenderASCII()
	if !strings.Contains(ascii, "Endpoint Relationship Graph") {
		t.Error("expected graph header in ASCII output")
	}
	if !strings.Contains(ascii, "Relationships:") {
		t.Error("expected relationships section")
	}
	if !strings.Contains(ascii, "Endpoints: 7") {
		t.Errorf("expected 7 endpoints in stats, got:\n%s", ascii)
	}
}

func TestGraphRenderDOT(t *testing.T) {
	profile := makeTestProfile()
	g := graph.New(profile)
	g.InferEdges()

	dot := g.RenderDOT()
	if !strings.Contains(dot, "digraph probex") {
		t.Error("expected DOT header")
	}
	if !strings.Contains(dot, "->") {
		t.Error("expected edges in DOT output")
	}
	if !strings.Contains(dot, "GET /users") {
		t.Error("expected endpoint labels in DOT")
	}
}

func TestGraphEmptyProfile(t *testing.T) {
	profile := &models.APIProfile{}
	g := graph.New(profile)
	ascii := g.RenderASCII()
	if !strings.Contains(ascii, "no endpoints") {
		t.Error("expected empty message")
	}
}
