package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/probex/probex/internal/models"
)

const probexDir = ".probex"

// Store manages local storage of API profiles and test results.
type Store struct {
	dir string
}

// New creates a new Store rooted at the given directory.
// If dir is empty, it defaults to .probex/ in the current directory.
func New(dir string) (*Store, error) {
	if dir == "" {
		dir = probexDir
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Store{dir: dir}, nil
}

// SaveProfile writes an API profile to disk.
func (s *Store) SaveProfile(profile *models.APIProfile) error {
	path := filepath.Join(s.dir, "profile.json")
	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// LoadProfile reads an API profile from disk.
func (s *Store) LoadProfile() (*models.APIProfile, error) {
	path := filepath.Join(s.dir, "profile.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var profile models.APIProfile
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, err
	}
	return &profile, nil
}

// ProfileExists returns true if a profile file exists on disk.
func (s *Store) ProfileExists() bool {
	path := filepath.Join(s.dir, "profile.json")
	_, err := os.Stat(path)
	return err == nil
}

// SaveResults writes test results to disk with a timestamp-based filename.
// The file is named results_YYYYMMDD_HHMMSS.json and a copy is saved as results.json
// for convenience.
func (s *Store) SaveResults(summary *models.RunSummary) error {
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return err
	}

	// Write with timestamp-based filename.
	ts := time.Now().Format("20060102_150405")
	tsPath := filepath.Join(s.dir, fmt.Sprintf("results_%s.json", ts))
	if err := os.WriteFile(tsPath, data, 0o644); err != nil {
		return err
	}

	// Also write as latest results.json for quick access.
	latestPath := filepath.Join(s.dir, "results.json")
	return os.WriteFile(latestPath, data, 0o644)
}

// LoadResults reads the most recent test results from disk.
// It reads results.json (the latest symlink/copy).
func (s *Store) LoadResults() (*models.RunSummary, error) {
	path := filepath.Join(s.dir, "results.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var summary models.RunSummary
	if err := json.Unmarshal(data, &summary); err != nil {
		return nil, err
	}
	return &summary, nil
}

// RunFile represents a past result file on disk.
type RunFile struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Timestamp time.Time `json:"timestamp"`
}

// ListRuns lists all past result files sorted by timestamp (newest first).
func (s *Store) ListRuns() ([]RunFile, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}

	var runs []RunFile
	for _, entry := range entries {
		name := entry.Name()
		// Match results_YYYYMMDD_HHMMSS.json pattern.
		if !strings.HasPrefix(name, "results_") || !strings.HasSuffix(name, ".json") {
			continue
		}
		// Skip the plain results.json.
		if name == "results.json" {
			continue
		}

		// Parse timestamp from filename: results_20060102_150405.json
		tsStr := strings.TrimPrefix(name, "results_")
		tsStr = strings.TrimSuffix(tsStr, ".json")
		ts, err := time.Parse("20060102_150405", tsStr)
		if err != nil {
			// Not a valid timestamp filename; skip.
			continue
		}

		runs = append(runs, RunFile{
			Name:      name,
			Path:      filepath.Join(s.dir, name),
			Timestamp: ts,
		})
	}

	// Sort newest first.
	sort.Slice(runs, func(i, j int) bool {
		return runs[i].Timestamp.After(runs[j].Timestamp)
	})

	return runs, nil
}
