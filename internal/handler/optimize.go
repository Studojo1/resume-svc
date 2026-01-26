package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/studojo/jobs/internal/domain/resume"
)

// OptimizeHandler handles POST /optimize-resume: resume JSON + job description in, optimized resume JSON out.
type OptimizeHandler struct{}

// NewOptimizeHandler returns a new optimize handler.
func NewOptimizeHandler() *OptimizeHandler {
	return &OptimizeHandler{}
}

// OptimizeRequest is the request body for optimization.
type OptimizeRequest struct {
	Resume        resume.Resume `json:"resume"`
	JobTitle      string         `json:"job_title"`
	Company       string         `json:"company"`
	JobDescription string        `json:"job_description"`
}

// OptimizeResume reads resume JSON and job description from request body, returns optimized resume JSON.
func (h *OptimizeHandler) OptimizeResume(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req OptimizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// TODO: Implement actual optimization logic using AI/LLM
	// For now, return the resume as-is (placeholder)
	// In production, this would:
	// 1. Analyze job description for keywords and requirements
	// 2. Optimize resume summary, skills, and work experience descriptions
	// 3. Reorder sections based on relevance
	// 4. Enhance descriptions with job-relevant keywords

	optimized := req.Resume

	// Placeholder: Add a note in summary if job title is provided
	if req.JobTitle != "" && optimized.Summary != "" {
		// In real implementation, use LLM to rewrite summary
		optimized.Summary = optimized.Summary + " (Optimized for " + req.JobTitle + " position)"
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(optimized); err != nil {
		log.Printf("encode optimized resume: %v", err)
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}
