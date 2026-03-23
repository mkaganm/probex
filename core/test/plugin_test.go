package test

import (
	"context"
	"testing"

	"github.com/mkaganm/probex/internal/models"
	"github.com/mkaganm/probex/internal/plugin"
)

// mockGenerator is a test generator plugin.
type mockGenerator struct {
	name  string
	tests []models.TestCase
}

func (m *mockGenerator) Meta() plugin.Metadata {
	return plugin.Metadata{
		Name:        m.name,
		Version:     "1.0.0",
		Description: "Mock generator for testing",
		Type:        plugin.TypeGenerator,
	}
}

func (m *mockGenerator) Generate(ctx context.Context, endpoint models.Endpoint) ([]models.TestCase, error) {
	return m.tests, nil
}

// mockHook is a test hook plugin.
type mockHook struct {
	name       string
	scanCalled bool
	runCalled  bool
}

func (m *mockHook) Meta() plugin.Metadata {
	return plugin.Metadata{
		Name:        m.name,
		Version:     "1.0.0",
		Description: "Mock hook for testing",
		Type:        plugin.TypeHook,
	}
}

func (m *mockHook) BeforeScan(ctx context.Context, baseURL string) error {
	m.scanCalled = true
	return nil
}

func (m *mockHook) AfterScan(ctx context.Context, profile *models.APIProfile) error {
	return nil
}

func (m *mockHook) BeforeRun(ctx context.Context, tests []models.TestCase) ([]models.TestCase, error) {
	m.runCalled = true
	return tests, nil
}

func (m *mockHook) AfterRun(ctx context.Context, summary *models.RunSummary) error {
	return nil
}

func TestPluginRegistryGenerator(t *testing.T) {
	reg := plugin.NewRegistry()

	gen := &mockGenerator{
		name: "test-gen",
		tests: []models.TestCase{
			{Name: "custom test", Category: models.CategoryHappyPath},
		},
	}

	if err := reg.RegisterGenerator(gen); err != nil {
		t.Fatalf("register: %v", err)
	}

	// Duplicate registration should fail.
	if err := reg.RegisterGenerator(gen); err == nil {
		t.Error("expected error on duplicate registration")
	}

	// Should be retrievable.
	found, ok := reg.Generator("test-gen")
	if !ok {
		t.Fatal("generator not found")
	}
	if found.Meta().Name != "test-gen" {
		t.Errorf("expected name test-gen, got %s", found.Meta().Name)
	}

	// Generate should work.
	tests, err := found.Generate(context.Background(), models.Endpoint{})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(tests) != 1 {
		t.Errorf("expected 1 test, got %d", len(tests))
	}
}

func TestPluginRegistryHooks(t *testing.T) {
	reg := plugin.NewRegistry()

	hook := &mockHook{name: "test-hook"}
	if err := reg.RegisterHook(hook); err != nil {
		t.Fatalf("register hook: %v", err)
	}

	// Run hooks.
	if err := reg.RunBeforeScan(context.Background(), "http://example.com"); err != nil {
		t.Fatalf("RunBeforeScan: %v", err)
	}
	if !hook.scanCalled {
		t.Error("BeforeScan was not called")
	}

	tests := []models.TestCase{{Name: "test1"}}
	result, err := reg.RunBeforeRun(context.Background(), tests)
	if err != nil {
		t.Fatalf("RunBeforeRun: %v", err)
	}
	if !hook.runCalled {
		t.Error("BeforeRun was not called")
	}
	if len(result) != 1 {
		t.Errorf("expected 1 test passed through, got %d", len(result))
	}
}

func TestPluginRegistryListAll(t *testing.T) {
	reg := plugin.NewRegistry()

	reg.RegisterGenerator(&mockGenerator{name: "gen1"})
	reg.RegisterGenerator(&mockGenerator{name: "gen2"})
	reg.RegisterHook(&mockHook{name: "hook1"})

	all := reg.ListAll()
	if len(all) != 3 {
		t.Errorf("expected 3 plugins, got %d", len(all))
	}
}
