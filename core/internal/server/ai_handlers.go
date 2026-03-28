package server

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/mkaganm/probex/internal/ai"
)

// requireAI checks that the AI client is available and writes a 503 if not.
// Returns true when AI is ready.
func (s *Server) requireAI(w http.ResponseWriter) bool {
	if s.aiClient == nil {
		writeError(w, http.StatusServiceUnavailable,
			"AI brain not available; start the server with --ai or --ai-url")
		return false
	}
	return true
}

func (s *Server) handleAIHealth(w http.ResponseWriter, r *http.Request) {
	if !s.requireAI(w) {
		return
	}

	health, err := s.aiClient.Health(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, "AI brain unreachable: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, health)
}

func (s *Server) handleAIScenarios(w http.ResponseWriter, r *http.Request) {
	if !s.requireAI(w) {
		return
	}

	var req ai.ScenarioRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, maxRequestBodySize)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}

	if len(req.Endpoints) == 0 {
		writeError(w, http.StatusBadRequest, "endpoints is required and must not be empty")
		return
	}

	resp, err := s.aiClient.GenerateScenarios(r.Context(), &req)
	if err != nil {
		writeError(w, http.StatusBadGateway, "AI scenario generation failed: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAISecurity(w http.ResponseWriter, r *http.Request) {
	if !s.requireAI(w) {
		return
	}

	var req ai.SecurityAnalysisRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, maxRequestBodySize)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}

	if len(req.Endpoints) == 0 {
		writeError(w, http.StatusBadRequest, "endpoints is required and must not be empty")
		return
	}

	resp, err := s.aiClient.AnalyzeSecurity(r.Context(), &req)
	if err != nil {
		writeError(w, http.StatusBadGateway, "AI security analysis failed: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAINLToTest(w http.ResponseWriter, r *http.Request) {
	if !s.requireAI(w) {
		return
	}

	var req ai.NLTestRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, maxRequestBodySize)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}

	if req.Description == "" {
		writeError(w, http.StatusBadRequest, "description is required")
		return
	}

	resp, err := s.aiClient.NLToTest(r.Context(), &req)
	if err != nil {
		writeError(w, http.StatusBadGateway, "AI NL-to-test generation failed: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAIAnomaly(w http.ResponseWriter, r *http.Request) {
	if !s.requireAI(w) {
		return
	}

	var req ai.AnomalyClassifyRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, maxRequestBodySize)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}

	if req.EndpointID == "" {
		writeError(w, http.StatusBadRequest, "endpoint_id is required")
		return
	}

	resp, err := s.aiClient.ClassifyAnomaly(r.Context(), &req)
	if err != nil {
		writeError(w, http.StatusBadGateway, "AI anomaly classification failed: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
