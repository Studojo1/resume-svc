package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/studojo/jobs/internal/domain/resume"
	"github.com/studojo/jobs/internal/tex"
)

// MakeHandler handles POST /make-resume: JSON in, PDF out.
type MakeHandler struct {
	Gen *tex.Generator
}

// NewMakeHandler returns a handler that uses the given PDF generator.
func NewMakeHandler(gen *tex.Generator) *MakeHandler {
	return &MakeHandler{Gen: gen}
}

// MakeResume reads resume JSON from the request body, generates a PDF, and streams it back.
func (h *MakeHandler) MakeResume(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req resume.Resume
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	pdf, err := h.Gen.GeneratePDF(&req)
	if err != nil {
		log.Printf("generate PDF: %v", err)
		http.Error(w, "failed to generate PDF", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="resume.pdf"`)
	_, _ = w.Write(pdf)
}

// GeneratePackageRequest is the request body for package generation.
type GeneratePackageRequest struct {
	Resume        resume.Resume `json:"resume"`
	JobTitle      string         `json:"job_title,omitempty"`
	Company       string         `json:"company,omitempty"`
	JobDescription string        `json:"job_description,omitempty"`
}

// GeneratePackage reads resume JSON and job info, generates Resume/Cover Letter/CV zip.
func (h *MakeHandler) GeneratePackage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req GeneratePackageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	zipData, err := h.Gen.GeneratePackage(&req.Resume, req.JobTitle, req.Company, req.JobDescription)
	if err != nil {
		log.Printf("generate package: %v", err)
		http.Error(w, "failed to generate package: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="resume-package.zip"`)
	_, _ = w.Write(zipData)
}
