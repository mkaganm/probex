package runner

import "sync"

// VarContext holds variables extracted from test responses for chaining.
// For example, a POST /users response body's "id" field can be used in
// subsequent GET /users/{id} requests.
type VarContext struct {
	mu   sync.RWMutex
	vars map[string]any
}

// NewVarContext creates a new variable context.
func NewVarContext() *VarContext {
	return &VarContext{vars: make(map[string]any)}
}

// Set stores a variable.
func (vc *VarContext) Set(key string, value any) {
	vc.mu.Lock()
	defer vc.mu.Unlock()
	vc.vars[key] = value
}

// Get retrieves a variable.
func (vc *VarContext) Get(key string) (any, bool) {
	vc.mu.RLock()
	defer vc.mu.RUnlock()
	v, ok := vc.vars[key]
	return v, ok
}
