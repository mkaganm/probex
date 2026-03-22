package scanner

// DefaultWordlist contains common API paths to probe during scanning.
var DefaultWordlist = []string{
	// Health & status
	"/health", "/healthz", "/ready", "/readyz",
	"/status", "/ping", "/version", "/info",

	// API roots
	"/api", "/api/v1", "/api/v2", "/api/v3",
	"/v1", "/v2", "/v3",

	// Common REST resources
	"/users", "/user", "/accounts", "/account",
	"/products", "/items", "/orders", "/order",
	"/posts", "/comments", "/categories", "/tags",
	"/auth", "/login", "/register", "/signup",
	"/search", "/config", "/settings",

	// Admin
	"/admin", "/admin/api", "/internal",
	"/metrics", "/stats", "/analytics",

	// CRUD patterns (appended to discovered resources)
	"/{id}", "/create", "/update", "/delete",
	"/list", "/all", "/count",
}
