package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/mkaganm/probex/internal/generator"
	"github.com/mkaganm/probex/internal/models"
	"github.com/mkaganm/probex/internal/runner"
	"github.com/mkaganm/probex/internal/scanner"
)

// maxRequestBodySize limits request body reads to 1MB.
const maxRequestBodySize = 1 * 1024 * 1024

func (s *Server) registerHandlers(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/health", s.handleHealth)
	mux.HandleFunc("GET /api/v1/profile", s.handleGetProfile)
	mux.HandleFunc("POST /api/v1/scan", s.handleScan)
	mux.HandleFunc("POST /api/v1/run", s.handleRun)
	mux.HandleFunc("GET /api/v1/results", s.handleGetResults)
	mux.HandleFunc("GET /api/v1/results/{id}", s.handleGetResultByID)
}

// writeJSON encodes v as JSON and writes it to the response.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"version": "1.0.0",
	})
}

func (s *Server) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	profile, err := s.store.LoadProfile()
	if err != nil {
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, "no profile found; run a scan first")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, profile)
}

// ScanRequest is the JSON body for POST /api/v1/scan.
type ScanRequest struct {
	BaseURL     string `json:"base_url"`
	MaxDepth    int    `json:"max_depth"`
	Concurrency int    `json:"concurrency"`
}

func (s *Server) handleScan(w http.ResponseWriter, r *http.Request) {
	var req ScanRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, maxRequestBodySize)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}
	if req.BaseURL == "" {
		writeError(w, http.StatusBadRequest, "base_url is required")
		return
	}

	// Apply defaults.
	if req.MaxDepth <= 0 {
		req.MaxDepth = 3
	}
	if req.Concurrency <= 0 {
		req.Concurrency = 10
	}

	opts := models.ScanOptions{
		MaxDepth:    req.MaxDepth,
		Concurrency: req.Concurrency,
		Timeout:     30 * time.Second,
		FollowLinks: true,
	}

	sc := scanner.New(req.BaseURL, opts)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	result, err := sc.Scan(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "scan failed: "+err.Error())
		return
	}

	// Build and persist the API profile.
	profile := &models.APIProfile{
		ID:        "profile_" + time.Now().Format("20060102_150405"),
		Name:      req.BaseURL,
		BaseURL:   result.BaseURL,
		Endpoints: result.Endpoints,
		Auth:      result.Auth,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		ScanConfig: models.ScanConfig{
			MaxDepth:    req.MaxDepth,
			Timeout:     opts.Timeout,
			Concurrency: req.Concurrency,
		},
	}

	if err := s.store.SaveProfile(profile); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save profile: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, profile)
}

// RunRequest is the JSON body for POST /api/v1/run.
type RunRequest struct {
	Categories  []string `json:"categories"`
	Concurrency int      `json:"concurrency"`
	Timeout     int      `json:"timeout"` // seconds
}

func (s *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	var req RunRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, maxRequestBodySize)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}

	// Load the current profile.
	profile, err := s.store.LoadProfile()
	if err != nil {
		if os.IsNotExist(err) {
			writeError(w, http.StatusPreconditionFailed, "no profile found; run a scan first")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load profile: "+err.Error())
		return
	}

	// Generate tests.
	eng := generator.New(profile)

	if len(req.Categories) > 0 {
		filter := make(map[models.TestCategory]bool)
		for _, c := range req.Categories {
			filter[models.TestCategory(c)] = true
		}
		eng.SetCategoryFilter(filter)
	}

	tests, err := eng.Generate()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "test generation failed: "+err.Error())
		return
	}

	if len(tests) == 0 {
		writeError(w, http.StatusUnprocessableEntity, "no tests generated for the current profile")
		return
	}

	// Apply defaults.
	concurrency := req.Concurrency
	if concurrency <= 0 {
		concurrency = 5
	}
	timeout := req.Timeout
	if timeout <= 0 {
		timeout = 30
	}

	runOpts := models.RunOptions{
		Concurrency: concurrency,
		Timeout:     time.Duration(timeout) * time.Second,
	}

	exec := runner.New(runOpts)
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()

	summary, err := exec.Execute(ctx, tests)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "test execution failed: "+err.Error())
		return
	}

	summary.ProfileID = profile.ID

	if err := s.store.SaveResults(summary); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save results: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, summary)
}

func (s *Server) handleGetResults(w http.ResponseWriter, r *http.Request) {
	summary, err := s.store.LoadResults()
	if err != nil {
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, "no results found; run tests first")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (s *Server) handleGetResultByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "result id is required")
		return
	}

	runs, err := s.store.ListRuns()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Match by timestamp portion of the filename (e.g. "20060102_150405")
	// or by the full filename.
	for _, run := range runs {
		nameNoExt := strings.TrimSuffix(run.Name, ".json")
		tsFromName := strings.TrimPrefix(nameNoExt, "results_")

		if tsFromName == id || run.Name == id || nameNoExt == id {
			data, err := os.ReadFile(run.Path)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			var summary models.RunSummary
			if err := json.Unmarshal(data, &summary); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, &summary)
			return
		}
	}

	writeError(w, http.StatusNotFound, "result not found for id: "+id)
}
