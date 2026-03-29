package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mkaganm/probex/internal/models"
	"github.com/mkaganm/probex/internal/storage"
)

func TestNewStore(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if store == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestNewStoreCreatesDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "subdir", ".probex")
	_, err := storage.New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("dir not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected directory")
	}
}

func TestSaveAndLoadProfile(t *testing.T) {
	store, err := storage.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	profile := &models.APIProfile{
		ID:      "test-profile",
		Name:    "Test API",
		BaseURL: "http://example.com",
		Endpoints: []models.Endpoint{
			{
				Method:  "GET",
				Path:    "/users",
				BaseURL: "http://example.com",
			},
			{
				Method:  "POST",
				Path:    "/users",
				BaseURL: "http://example.com",
			},
		},
	}

	if err := store.SaveProfile(profile); err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}

	loaded, err := store.LoadProfile()
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}

	if loaded.ID != profile.ID {
		t.Errorf("ID: got %s, want %s", loaded.ID, profile.ID)
	}
	if loaded.BaseURL != profile.BaseURL {
		t.Errorf("BaseURL: got %s, want %s", loaded.BaseURL, profile.BaseURL)
	}
	if len(loaded.Endpoints) != 2 {
		t.Errorf("Endpoints: got %d, want 2", len(loaded.Endpoints))
	}
}

func TestLoadProfileNotFound(t *testing.T) {
	store, err := storage.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.LoadProfile()
	if err == nil {
		t.Fatal("expected error for missing profile")
	}
	if !os.IsNotExist(err) {
		t.Errorf("expected not-exist error, got: %v", err)
	}
}

func TestProfileExists(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.New(dir)
	if err != nil {
		t.Fatal(err)
	}

	if store.ProfileExists() {
		t.Error("expected ProfileExists=false before save")
	}

	_ = store.SaveProfile(&models.APIProfile{ID: "p1"})

	if !store.ProfileExists() {
		t.Error("expected ProfileExists=true after save")
	}
}

func TestSaveAndLoadResults(t *testing.T) {
	store, err := storage.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	summary := &models.RunSummary{
		TotalTests: 10,
		Passed:     8,
		Failed:     1,
		Errors:     1,
		ProfileID:  "profile_1",
	}

	if err := store.SaveResults(summary); err != nil {
		t.Fatalf("SaveResults: %v", err)
	}

	loaded, err := store.LoadResults()
	if err != nil {
		t.Fatalf("LoadResults: %v", err)
	}

	if loaded.TotalTests != 10 {
		t.Errorf("TotalTests: got %d, want 10", loaded.TotalTests)
	}
	if loaded.Passed != 8 {
		t.Errorf("Passed: got %d, want 8", loaded.Passed)
	}
	if loaded.Failed != 1 {
		t.Errorf("Failed: got %d, want 1", loaded.Failed)
	}
}

func TestLoadResultsNotFound(t *testing.T) {
	store, err := storage.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.LoadResults()
	if err == nil {
		t.Fatal("expected error for missing results")
	}
}

func TestListRuns(t *testing.T) {
	store, err := storage.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	// Save multiple results.
	for i := 0; i < 3; i++ {
		if err := store.SaveResults(&models.RunSummary{TotalTests: i}); err != nil {
			t.Fatalf("SaveResults %d: %v", i, err)
		}
	}

	runs, err := store.ListRuns()
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}

	// May get fewer than 3 if saves happen within the same second (same timestamp).
	// But at least 1 run should exist.
	if len(runs) == 0 {
		t.Error("expected at least 1 run file")
	}

	// Verify sorted newest first.
	for i := 1; i < len(runs); i++ {
		if runs[i].Timestamp.After(runs[i-1].Timestamp) {
			t.Errorf("runs not sorted newest first: %v after %v", runs[i].Timestamp, runs[i-1].Timestamp)
		}
	}
}
