package plugin

import (
	"context"
	"fmt"
	"sync"

	"github.com/mkaganm/probex/internal/models"
)

// Type identifies the kind of plugin.
type Type string

const (
	TypeGenerator Type = "generator"
	TypeReporter  Type = "reporter"
	TypeHook      Type = "hook"
)

// Metadata describes a plugin.
type Metadata struct {
	Name        string `json:"name" yaml:"name"`
	Version     string `json:"version" yaml:"version"`
	Description string `json:"description" yaml:"description"`
	Author      string `json:"author" yaml:"author"`
	Type        Type   `json:"type" yaml:"type"`
}

// GeneratorPlugin generates test cases for endpoints.
type GeneratorPlugin interface {
	Meta() Metadata
	Generate(ctx context.Context, endpoint models.Endpoint) ([]models.TestCase, error)
}

// ReporterPlugin produces output from test results.
type ReporterPlugin interface {
	Meta() Metadata
	Report(ctx context.Context, summary *models.RunSummary) ([]byte, error)
	FileExtension() string
}

// HookPlugin is called at various points during the test lifecycle.
type HookPlugin interface {
	Meta() Metadata
	BeforeScan(ctx context.Context, baseURL string) error
	AfterScan(ctx context.Context, profile *models.APIProfile) error
	BeforeRun(ctx context.Context, tests []models.TestCase) ([]models.TestCase, error)
	AfterRun(ctx context.Context, summary *models.RunSummary) error
}

// Registry manages registered plugins.
type Registry struct {
	mu         sync.RWMutex
	generators map[string]GeneratorPlugin
	reporters  map[string]ReporterPlugin
	hooks      map[string]HookPlugin
}

// NewRegistry creates a new plugin registry.
func NewRegistry() *Registry {
	return &Registry{
		generators: make(map[string]GeneratorPlugin),
		reporters:  make(map[string]ReporterPlugin),
		hooks:      make(map[string]HookPlugin),
	}
}

// RegisterGenerator registers a generator plugin.
func (r *Registry) RegisterGenerator(p GeneratorPlugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	name := p.Meta().Name
	if _, exists := r.generators[name]; exists {
		return fmt.Errorf("generator plugin %q already registered", name)
	}
	r.generators[name] = p
	return nil
}

// RegisterReporter registers a reporter plugin.
func (r *Registry) RegisterReporter(p ReporterPlugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	name := p.Meta().Name
	if _, exists := r.reporters[name]; exists {
		return fmt.Errorf("reporter plugin %q already registered", name)
	}
	r.reporters[name] = p
	return nil
}

// RegisterHook registers a hook plugin.
func (r *Registry) RegisterHook(p HookPlugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	name := p.Meta().Name
	if _, exists := r.hooks[name]; exists {
		return fmt.Errorf("hook plugin %q already registered", name)
	}
	r.hooks[name] = p
	return nil
}

// Generators returns all registered generator plugins.
func (r *Registry) Generators() []GeneratorPlugin {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]GeneratorPlugin, 0, len(r.generators))
	for _, p := range r.generators {
		result = append(result, p)
	}
	return result
}

// Reporters returns all registered reporter plugins.
func (r *Registry) Reporters() []ReporterPlugin {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]ReporterPlugin, 0, len(r.reporters))
	for _, p := range r.reporters {
		result = append(result, p)
	}
	return result
}

// Hooks returns all registered hook plugins.
func (r *Registry) Hooks() []HookPlugin {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]HookPlugin, 0, len(r.hooks))
	for _, p := range r.hooks {
		result = append(result, p)
	}
	return result
}

// Generator returns a generator by name.
func (r *Registry) Generator(name string) (GeneratorPlugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.generators[name]
	return p, ok
}

// Reporter returns a reporter by name.
func (r *Registry) Reporter(name string) (ReporterPlugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.reporters[name]
	return p, ok
}

// ListAll returns metadata for all registered plugins.
func (r *Registry) ListAll() []Metadata {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var all []Metadata
	for _, p := range r.generators {
		all = append(all, p.Meta())
	}
	for _, p := range r.reporters {
		all = append(all, p.Meta())
	}
	for _, p := range r.hooks {
		all = append(all, p.Meta())
	}
	return all
}

// RunBeforeScan calls BeforeScan on all hook plugins.
func (r *Registry) RunBeforeScan(ctx context.Context, baseURL string) error {
	r.mu.RLock()
	hooks := make([]HookPlugin, 0, len(r.hooks))
	for _, h := range r.hooks {
		hooks = append(hooks, h)
	}
	r.mu.RUnlock()

	for _, h := range hooks {
		if err := h.BeforeScan(ctx, baseURL); err != nil {
			return fmt.Errorf("hook %s BeforeScan: %w", h.Meta().Name, err)
		}
	}
	return nil
}

// RunAfterScan calls AfterScan on all hook plugins.
func (r *Registry) RunAfterScan(ctx context.Context, profile *models.APIProfile) error {
	r.mu.RLock()
	hooks := make([]HookPlugin, 0, len(r.hooks))
	for _, h := range r.hooks {
		hooks = append(hooks, h)
	}
	r.mu.RUnlock()

	for _, h := range hooks {
		if err := h.AfterScan(ctx, profile); err != nil {
			return fmt.Errorf("hook %s AfterScan: %w", h.Meta().Name, err)
		}
	}
	return nil
}

// RunBeforeRun calls BeforeRun on all hook plugins, passing tests through each.
func (r *Registry) RunBeforeRun(ctx context.Context, tests []models.TestCase) ([]models.TestCase, error) {
	r.mu.RLock()
	hooks := make([]HookPlugin, 0, len(r.hooks))
	for _, h := range r.hooks {
		hooks = append(hooks, h)
	}
	r.mu.RUnlock()

	current := tests
	for _, h := range hooks {
		var err error
		current, err = h.BeforeRun(ctx, current)
		if err != nil {
			return nil, fmt.Errorf("hook %s BeforeRun: %w", h.Meta().Name, err)
		}
	}
	return current, nil
}

// RunAfterRun calls AfterRun on all hook plugins.
func (r *Registry) RunAfterRun(ctx context.Context, summary *models.RunSummary) error {
	r.mu.RLock()
	hooks := make([]HookPlugin, 0, len(r.hooks))
	for _, h := range r.hooks {
		hooks = append(hooks, h)
	}
	r.mu.RUnlock()

	for _, h := range hooks {
		if err := h.AfterRun(ctx, summary); err != nil {
			return fmt.Errorf("hook %s AfterRun: %w", h.Meta().Name, err)
		}
	}
	return nil
}
