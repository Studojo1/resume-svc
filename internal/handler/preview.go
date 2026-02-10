package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/studojo/jobs/internal/domain/resume"
	"github.com/studojo/jobs/internal/tex"
)

// PreviewHandler handles POST /preview: Quick PDF generation for preview.
type PreviewHandler struct {
	Gen *tex.Generator
}

// NewPreviewHandler returns a handler that uses the given PDF generator.
func NewPreviewHandler(gen *tex.Generator) *PreviewHandler {
	return &PreviewHandler{Gen: gen}
}

// PreviewRequest is the request body for preview generation.
type PreviewRequest struct {
	Resume    resume.Resume `json:"resume"`
	TemplateID string       `json:"template_id,omitempty"`
}

// PreviewResume reads resume JSON from the request body, generates a PDF quickly for preview.
func (h *PreviewHandler) PreviewResume(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Debug logging: Log received resume data
	log.Printf("[preview] Received resume - Name: %s, Summary length: %d, WorkExperiences: %d, Educations: %d, Skills: %d, Projects: %d, Certifications: %d, TemplateID: %s",
		req.Resume.ContactInfo.Name,
		len(req.Resume.Summary),
		len(req.Resume.WorkExperiences),
		len(req.Resume.Educations),
		len(req.Resume.Skills),
		len(req.Resume.Projects),
		len(req.Resume.Certifications),
		req.Resume.TemplateID)

	// Validation: Ensure resume has at least some content
	hasContent := req.Resume.ContactInfo.Name != "" || 
		req.Resume.ContactInfo.Email != "" || 
		req.Resume.Summary != "" || 
		len(req.Resume.WorkExperiences) > 0 || 
		len(req.Resume.Educations) > 0 ||
		len(req.Resume.Skills) > 0 ||
		len(req.Resume.Projects) > 0 ||
		len(req.Resume.Certifications) > 0
	
	if !hasContent {
		log.Printf("[preview] Error: Resume is completely empty - cannot generate PDF")
		http.Error(w, "Resume is empty. Please add at least a name, email, or some content before generating a preview.", http.StatusBadRequest)
		return
	}

	// Set template ID if provided
	if req.TemplateID != "" {
		req.Resume.TemplateID = req.TemplateID
	}
	
	// Generate PDF (optimized for speed - no cover letter, no zip)
	pdf, err := h.Gen.GeneratePDF(&req.Resume)
	if err != nil {
		log.Printf("generate preview PDF: %v", err)
		http.Error(w, "failed to generate preview PDF", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `inline; filename="resume-preview.pdf"`)
	_, _ = w.Write(pdf)
}

