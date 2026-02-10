package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/studojo/jobs/internal/domain/resume"
	"github.com/studojo/jobs/internal/template"
	"github.com/studojo/jobs/internal/tex"
)

// TemplatesHandler handles template listing and metadata.
type TemplatesHandler struct {
	Gen *tex.Generator
}

// NewTemplatesHandler returns a new templates handler.
func NewTemplatesHandler(gen *tex.Generator) *TemplatesHandler {
	return &TemplatesHandler{Gen: gen}
}

// ListTemplates handles GET /templates - List available templates.
func (h *TemplatesHandler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	templates, err := template.ListTemplates()
	if err != nil {
		log.Printf("list templates: %v", err)
		http.Error(w, "failed to list templates", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(templates); err != nil {
		log.Printf("encode templates: %v", err)
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

// GetTemplate handles GET /templates/:id - Get template metadata.
func (h *TemplatesHandler) GetTemplate(w http.ResponseWriter, r *http.Request, templateID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tmpl, err := template.GetTemplate(templateID)
	if err != nil {
		log.Printf("get template: %v", err)
		http.Error(w, "template not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(tmpl); err != nil {
		log.Printf("encode template: %v", err)
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

// GetTemplatePreview handles GET /templates/:id/preview
// DEPRECATED: Template previews are now static PNG images stored in blob storage.
// This endpoint is no longer used - previews are served directly from storage.
// Returns 410 Gone to indicate the endpoint is deprecated.
func (h *TemplatesHandler) GetTemplatePreview(w http.ResponseWriter, r *http.Request, templateID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Return 410 Gone - this endpoint is deprecated
	// Template previews are now static images served from blob storage
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusGone)
	json.NewEncoder(w).Encode(map[string]string{
		"error": "Template preview generation is deprecated. Use static preview images from blob storage.",
		"message": "Template previews are now pre-generated static PNG images stored in blob storage.",
	})
}

// generateSampleResume creates a sample resume with realistic data for template previews.
func generateSampleResume(templateID string) resume.Resume {
	startDate1 := "2020-01"
	endDate1 := "2023-12"
	startDate2 := "2018-06"
	endDate2 := "2019-12"
	eduStart := "2014-09"
	eduEnd := "2018-05"

	return resume.Resume{
		Title:   "Senior Software Engineer",
		Summary: "Experienced software engineer with 5+ years of expertise in full-stack development, cloud architecture, and team leadership. Passionate about building scalable systems and mentoring junior developers.",
		ContactInfo: resume.ContactInfo{
			Name:     "Alex Johnson",
			Email:    "alex.johnson@email.com",
			Phone:    "+1 (555) 123-4567",
			Location: "San Francisco, CA",
			LinkedIn: "linkedin.com/in/alexjohnson",
			Website:  "alexjohnson.dev",
		},
		TemplateID: templateID,
		WorkExperiences: []resume.WorkExperience{
			{
				Company:   "Tech Innovations Inc.",
				Role:      "Senior Software Engineer",
				StartDate: &startDate1,
				EndDate:   &endDate1,
				IsCurrent: false,
				Description: "Led development of microservices architecture serving 1M+ users. Architected and implemented RESTful APIs using Go and Python. Mentored team of 5 junior engineers. Reduced system latency by 40% through optimization.",
			},
			{
				Company:   "StartupXYZ",
				Role:      "Full Stack Developer",
				StartDate: &startDate2,
				EndDate:   &endDate2,
				IsCurrent: false,
				Description: "Built customer-facing web applications using React and Node.js. Implemented CI/CD pipelines reducing deployment time by 60%. Collaborated with design team to create intuitive user experiences.",
			},
		},
		Educations: []resume.Education{
			{
				Institution:  "University of California, Berkeley",
				Degree:       "Bachelor of Science",
				FieldOfStudy: "Computer Science",
				StartDate:    &eduStart,
				EndDate:      &eduEnd,
				IsCurrent:    false,
				Description:  "Graduated Magna Cum Laude. Relevant coursework: Data Structures, Algorithms, Database Systems, Software Engineering.",
			},
		},
		Skills: []resume.Skill{
			{Category: "Languages", Name: "Go", Proficiency: "Expert"},
			{Category: "Languages", Name: "Python", Proficiency: "Expert"},
			{Category: "Languages", Name: "JavaScript/TypeScript", Proficiency: "Expert"},
			{Category: "Frameworks", Name: "React", Proficiency: "Advanced"},
			{Category: "Frameworks", Name: "Node.js", Proficiency: "Advanced"},
			{Category: "Cloud", Name: "AWS", Proficiency: "Advanced"},
			{Category: "Cloud", Name: "Docker", Proficiency: "Advanced"},
			{Category: "Databases", Name: "PostgreSQL", Proficiency: "Advanced"},
			{Category: "Tools", Name: "Git", Proficiency: "Expert"},
		},
		Projects: []resume.Project{
			{
				Title:     "Distributed Task Queue System",
				URL:       "github.com/alex/taskqueue",
				StartDate: &startDate1,
				EndDate:   nil,
				Description: "Open-source distributed task queue built with Go. Supports Redis and RabbitMQ backends. Used by 500+ developers.",
			},
		},
		Certifications: []resume.Certification{
			{
				Name:      "AWS Certified Solutions Architect",
				Issuer:    "Amazon Web Services",
				IssueDate: &startDate1,
				ExpiryDate: nil,
				URL:       "aws.amazon.com/certification",
			},
		},
	}
}

