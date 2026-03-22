package graph

import (
	"fmt"
	"sort"
	"strings"

	"github.com/probex/probex/internal/models"
)

// Edge represents a relationship between two endpoints.
type Edge struct {
	From   string
	To     string
	Label  string
	Weight int
}

// Graph represents the endpoint relationship graph.
type Graph struct {
	nodes map[string]*models.Endpoint
	edges []Edge
}

// New creates a Graph from a profile.
func New(profile *models.APIProfile) *Graph {
	g := &Graph{
		nodes: make(map[string]*models.Endpoint),
	}
	for i := range profile.Endpoints {
		ep := &profile.Endpoints[i]
		key := ep.Method + " " + ep.Path
		g.nodes[key] = ep
	}
	return g
}

// AddEdge adds a relationship edge.
func (g *Graph) AddEdge(from, to, label string, weight int) {
	g.edges = append(g.edges, Edge{From: from, To: to, Label: label, Weight: weight})
}

// InferEdges infers relationships from endpoint structure.
func (g *Graph) InferEdges() {
	keys := g.sortedKeys()

	for _, k1 := range keys {
		ep1 := g.nodes[k1]
		for _, k2 := range keys {
			if k1 == k2 {
				continue
			}
			ep2 := g.nodes[k2]

			// POST /resource -> GET /resource (create->list)
			if ep1.Method == "POST" && ep2.Method == "GET" && ep1.Path == ep2.Path {
				g.AddEdge(k1, k2, "create→list", 3)
			}

			// POST /resource -> GET /resource/{id} (create->read)
			if ep1.Method == "POST" && ep2.Method == "GET" &&
				strings.HasPrefix(ep2.Path, ep1.Path+"/") &&
				isParamSegment(strings.TrimPrefix(ep2.Path, ep1.Path+"/")) {
				g.AddEdge(k1, k2, "create→read", 5)
			}

			// GET /resource/{id} -> PUT /resource/{id} (read->update)
			if ep1.Method == "GET" && ep2.Method == "PUT" && ep1.Path == ep2.Path &&
				strings.Contains(ep1.Path, "{") {
				g.AddEdge(k1, k2, "read→update", 3)
			}

			// PUT /resource/{id} -> DELETE /resource/{id} (update->delete)
			if ep1.Method == "PUT" && ep2.Method == "DELETE" && ep1.Path == ep2.Path {
				g.AddEdge(k1, k2, "update→delete", 2)
			}

			// GET /resource/{id} -> DELETE /resource/{id} (read->delete)
			if ep1.Method == "GET" && ep2.Method == "DELETE" && ep1.Path == ep2.Path &&
				strings.Contains(ep1.Path, "{") {
				g.AddEdge(k1, k2, "read→delete", 2)
			}

			// Nested resources: POST /resource/{id}/child -> GET /resource/{id}/child
			if ep1.Method == "POST" && ep2.Method == "GET" && ep1.Path == ep2.Path &&
				strings.Count(ep1.Path, "/") >= 3 {
				g.AddEdge(k1, k2, "create→list", 3)
			}
		}
	}
}

// RenderASCII produces an ASCII representation of the endpoint relationship graph.
func (g *Graph) RenderASCII() string {
	if len(g.nodes) == 0 {
		return "  (no endpoints)"
	}

	var sb strings.Builder
	sb.WriteString("╔══════════════════════════════════════════════════╗\n")
	sb.WriteString("║          PROBEX — Endpoint Relationship Graph   ║\n")
	sb.WriteString("╚══════════════════════════════════════════════════╝\n\n")

	// Group endpoints by resource (base path).
	resources := g.groupByResource()
	resourceKeys := sortedMapKeys(resources)

	for _, resource := range resourceKeys {
		endpoints := resources[resource]
		sb.WriteString(fmt.Sprintf("  ┌─ %s\n", resource))
		for i, ep := range endpoints {
			prefix := "  │  "
			if i == len(endpoints)-1 {
				prefix = "  │  "
			}
			sb.WriteString(fmt.Sprintf("%s├── %s %s\n", prefix, ep.Method, ep.Path))
		}
		sb.WriteString("  │\n")
	}

	// Print edges.
	if len(g.edges) > 0 {
		sb.WriteString("  Relationships:\n")
		sb.WriteString("  ─────────────\n")

		// Sort by weight desc.
		sorted := make([]Edge, len(g.edges))
		copy(sorted, g.edges)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Weight > sorted[j].Weight
		})

		for _, e := range sorted {
			sb.WriteString(fmt.Sprintf("  %s\n", renderEdge(e)))
		}
	}

	// Print stats.
	sb.WriteString(fmt.Sprintf("\n  Endpoints: %d  |  Relationships: %d  |  Resources: %d\n",
		len(g.nodes), len(g.edges), len(resources)))

	return sb.String()
}

// RenderDOT produces a DOT/Graphviz representation.
func (g *Graph) RenderDOT() string {
	var sb strings.Builder
	sb.WriteString("digraph probex {\n")
	sb.WriteString("  rankdir=LR;\n")
	sb.WriteString("  node [shape=box, style=rounded, fontname=\"monospace\"];\n")
	sb.WriteString("  edge [fontname=\"monospace\", fontsize=10];\n\n")

	// Nodes.
	for key, ep := range g.nodes {
		color := methodColor(ep.Method)
		label := fmt.Sprintf("%s %s", ep.Method, ep.Path)
		sb.WriteString(fmt.Sprintf("  %q [label=%q, color=%q];\n",
			key, label, color))
	}
	sb.WriteString("\n")

	// Edges.
	for _, e := range g.edges {
		sb.WriteString(fmt.Sprintf("  %q -> %q [label=%q, penwidth=%d];\n",
			e.From, e.To, e.Label, max(1, e.Weight/2)))
	}

	sb.WriteString("}\n")
	return sb.String()
}

func (g *Graph) groupByResource() map[string][]models.Endpoint {
	resources := make(map[string][]models.Endpoint)
	for _, ep := range g.nodes {
		resource := extractResource(ep.Path)
		resources[resource] = append(resources[resource], *ep)
	}
	// Sort endpoints within each resource by method order.
	methodOrder := map[string]int{
		"GET": 1, "POST": 2, "PUT": 3, "PATCH": 4, "DELETE": 5,
	}
	for k := range resources {
		sort.Slice(resources[k], func(i, j int) bool {
			oi := methodOrder[resources[k][i].Method]
			oj := methodOrder[resources[k][j].Method]
			if oi != oj {
				return oi < oj
			}
			return resources[k][i].Path < resources[k][j].Path
		})
	}
	return resources
}

func (g *Graph) sortedKeys() []string {
	keys := make([]string, 0, len(g.nodes))
	for k := range g.nodes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func extractResource(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 {
		return "/"
	}
	// Find the first non-param segment.
	for i, p := range parts {
		if strings.HasPrefix(p, "{") || strings.HasPrefix(p, ":") {
			if i > 0 {
				return "/" + strings.Join(parts[:i], "/")
			}
			return "/"
		}
	}
	// For paths like /users or /users/profile, use the first segment.
	if len(parts) <= 2 {
		return "/" + parts[0]
	}
	return "/" + strings.Join(parts[:2], "/")
}

func isParamSegment(s string) bool {
	return strings.HasPrefix(s, "{") || strings.HasPrefix(s, ":") || s == ":id" || s == "{id}"
}

func renderEdge(e Edge) string {
	stars := strings.Repeat("★", min(e.Weight, 5))
	return fmt.Sprintf("%s  ──[%s]──▸  %s  %s", e.From, e.Label, e.To, stars)
}

func methodColor(method string) string {
	switch method {
	case "GET":
		return "blue"
	case "POST":
		return "green"
	case "PUT", "PATCH":
		return "orange"
	case "DELETE":
		return "red"
	default:
		return "black"
	}
}

func sortedMapKeys(m map[string][]models.Endpoint) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
