package learn

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mkaganm/probex/internal/models"
)

// Result holds the complete output of the learning process.
type Result struct {
	Profile         *models.APIProfile `json:"profile"`
	TrafficAnalysis *TrafficAnalysis   `json:"traffic_analysis"`
	PatternReport   *PatternReport     `json:"pattern_report"`
	HARFilesRead    int                `json:"har_files_read"`
	EntriesAnalyzed int                `json:"entries_analyzed"`
}

// Learner orchestrates the learning process from HAR files.
type Learner struct{}

// NewLearner creates a new Learner.
func NewLearner() *Learner {
	return &Learner{}
}

// Learn runs the full learning pipeline on a HAR file or directory of HAR files.
// If existingProfile is non-nil, it will be enriched with the learned data.
// Otherwise, a new profile is created.
func (l *Learner) Learn(ctx context.Context, harPath string, existingProfile *models.APIProfile) (*Result, error) {
	// Collect HAR files.
	harFiles, err := collectHARFiles(harPath)
	if err != nil {
		return nil, fmt.Errorf("collect HAR files: %w", err)
	}
	if len(harFiles) == 0 {
		return nil, fmt.Errorf("no HAR files found at %s", harPath)
	}

	// Parse all HAR files and merge.
	merged, err := parseAndMerge(harFiles)
	if err != nil {
		return nil, fmt.Errorf("parse HAR files: %w", err)
	}

	// Check for context cancellation.
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Run traffic analysis.
	trafficAnalysis := AnalyzeTraffic(merged)

	// Build baselines.
	baseline := BuildBaseline(merged.Grouped)

	// Mine patterns.
	patternReport := MinePatterns(merged.Grouped)

	// Build or enrich the profile.
	profile := existingProfile
	if profile == nil {
		profile = &models.APIProfile{
			ID:        fmt.Sprintf("learned-%d", time.Now().Unix()),
			Name:      "Learned from traffic",
			CreatedAt: time.Now(),
		}
	}

	// Merge endpoints into profile.
	mergeEndpoints(profile, merged.Endpoints)

	// Set baseline.
	if profile.Baseline == nil {
		profile.Baseline = baseline
	} else {
		// Merge baselines.
		for key, eb := range baseline.Endpoints {
			profile.Baseline.Endpoints[key] = eb
		}
	}

	// Detect base URL from endpoints.
	if profile.BaseURL == "" && len(merged.Endpoints) > 0 {
		profile.BaseURL = merged.Endpoints[0].BaseURL
	}

	profile.UpdatedAt = time.Now()

	return &Result{
		Profile:         profile,
		TrafficAnalysis: trafficAnalysis,
		PatternReport:   patternReport,
		HARFilesRead:    len(harFiles),
		EntriesAnalyzed: len(merged.Ordered),
	}, nil
}

// collectHARFiles returns a list of HAR file paths from a path (file or directory).
func collectHARFiles(path string) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if !info.IsDir() {
		return []string{path}, nil
	}

	// Walk directory for .har files.
	var files []string
	err = filepath.Walk(path, func(p string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !fi.IsDir() && strings.HasSuffix(strings.ToLower(fi.Name()), ".har") {
			files = append(files, p)
		}
		return nil
	})
	return files, err
}

// parseAndMerge parses multiple HAR files and merges them into a single ParsedHAR.
func parseAndMerge(files []string) (*ParsedHAR, error) {
	if len(files) == 1 {
		return ParseHARFile(files[0])
	}

	merged := &ParsedHAR{
		Grouped: make(map[EndpointKey][]Entry),
	}

	seenEndpoints := make(map[string]bool)

	for _, file := range files {
		parsed, err := ParseHARFile(file)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", file, err)
		}

		// Merge grouped entries.
		for key, entries := range parsed.Grouped {
			merged.Grouped[key] = append(merged.Grouped[key], entries...)
		}

		// Merge ordered entries.
		merged.Ordered = append(merged.Ordered, parsed.Ordered...)

		// Merge endpoints (deduplicate by ID).
		for _, ep := range parsed.Endpoints {
			if !seenEndpoints[ep.ID] {
				seenEndpoints[ep.ID] = true
				merged.Endpoints = append(merged.Endpoints, ep)
			}
		}
	}

	return merged, nil
}

// mergeEndpoints adds new endpoints to the profile, deduplicating by method+path.
func mergeEndpoints(profile *models.APIProfile, newEndpoints []models.Endpoint) {
	existing := make(map[string]bool)
	for _, ep := range profile.Endpoints {
		key := ep.Method + " " + ep.Path
		existing[key] = true
	}

	for _, ep := range newEndpoints {
		key := ep.Method + " " + ep.Path
		if !existing[key] {
			existing[key] = true
			profile.Endpoints = append(profile.Endpoints, ep)
		}
	}
}
