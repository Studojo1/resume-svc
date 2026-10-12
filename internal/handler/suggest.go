package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/studojo/jobs/internal/domain/resume"
)

// SuggestHandler handles POST /suggest: Quick section-level suggestions.
type SuggestHandler struct{}

// NewSuggestHandler returns a new suggest handler.
func NewSuggestHandler() *SuggestHandler {
	return &SuggestHandler{}
}

// SuggestRequest is the request body for suggestions.
type SuggestRequest struct {
	Resume        resume.Resume `json:"resume"`
	JobTitle      string        `json:"job_title,omitempty"`
	JobDescription string       `json:"job_description,omitempty"`
}

// Suggestion represents a single suggestion for a section.
type Suggestion struct {
	Section     string `json:"section"`
	Type        string `json:"type"` // "keyword", "completeness", "formatting"
	Message     string `json:"message"`
	Severity    string `json:"severity"` // "info", "warning", "error"
	Field       string `json:"field,omitempty"`
	SuggestedValue string `json:"suggested_value,omitempty"`
}

// Suggest provides AI-powered section-level suggestions using Azure OpenAI.
func (h *SuggestHandler) Suggest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SuggestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	suggestions := []Suggestion{}

	// Try to get AI suggestions from Azure OpenAI
	aiSuggestions, err := getAISuggestions(req.Resume, req.JobTitle, req.JobDescription)
	if err != nil {
		log.Printf("Failed to get AI suggestions: %v, falling back to rule-based suggestions", err)
	} else {
		suggestions = append(suggestions, aiSuggestions...)
	}

	// Check summary completeness
	if req.Resume.Summary == "" || len(strings.TrimSpace(req.Resume.Summary)) < 50 {
		suggestions = append(suggestions, Suggestion{
			Section:  "summary",
			Type:     "completeness",
			Message:  "Summary is too short. Aim for 2-3 sentences highlighting your key strengths.",
			Severity: "warning",
			Field:    "summary",
		})
	}

	// Check contact info
	if req.Resume.ContactInfo.Email == "" {
		suggestions = append(suggestions, Suggestion{
			Section:  "contact_info",
			Type:     "completeness",
			Message:  "Email address is required",
			Severity: "error",
			Field:    "email",
		})
	}
	if req.Resume.ContactInfo.Name == "" {
		suggestions = append(suggestions, Suggestion{
			Section:  "contact_info",
			Type:     "completeness",
			Message:  "Name is required",
			Severity: "error",
			Field:    "name",
		})
	}

	// Check work experience
	if len(req.Resume.WorkExperiences) == 0 {
		suggestions = append(suggestions, Suggestion{
			Section:  "work_experiences",
			Type:     "completeness",
			Message:  "Add at least one work experience to strengthen your resume",
			Severity: "warning",
		})
	} else {
		for i, exp := range req.Resume.WorkExperiences {
			if exp.Description == "" || len(strings.TrimSpace(exp.Description)) < 20 {
				suggestions = append(suggestions, Suggestion{
					Section:  "work_experiences",
					Type:     "completeness",
					Message:  "Work experience description is too brief. Add specific achievements and responsibilities.",
					Severity: "warning",
					Field:    "work_experiences",
				})
			}
			if exp.Role == "" {
				suggestions = append(suggestions, Suggestion{
					Section:  "work_experiences",
					Type:     "completeness",
					Message:  "Job title is required",
					Severity: "error",
					Field:    "work_experiences",
				})
			}
			_ = i // Avoid unused variable
		}
	}

	// Check education
	if len(req.Resume.Educations) == 0 {
		suggestions = append(suggestions, Suggestion{
			Section:  "educations",
			Type:     "completeness",
			Message:  "Add your education background",
			Severity: "warning",
		})
	}

	// Check skills
	if len(req.Resume.Skills) == 0 {
		suggestions = append(suggestions, Suggestion{
			Section:  "skills",
			Type:     "completeness",
			Message:  "Add relevant skills to showcase your expertise",
			Severity: "warning",
		})
	}

	// Extract keywords from job description if provided
	if req.JobDescription != "" {
		keywords := extractKeywords(req.JobDescription)
		missingKeywords := findMissingKeywords(req.Resume, keywords)
		
		if len(missingKeywords) > 0 {
			suggestions = append(suggestions, Suggestion{
				Section:  "keywords",
				Type:     "keyword",
				Message:  "Consider adding these keywords from the job description: " + strings.Join(missingKeywords[:min(5, len(missingKeywords))], ", "),
				Severity: "info",
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"suggestions": suggestions,
	}); err != nil {
		log.Printf("encode suggestions: %v", err)
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

// extractKeywords extracts common keywords from job description (simple approach)
func extractKeywords(jobDesc string) []string {
	// Common technical and professional keywords
	commonKeywords := []string{
		"python", "javascript", "java", "react", "node", "sql", "aws", "docker",
		"kubernetes", "agile", "scrum", "leadership", "communication", "analytical",
		"problem-solving", "team", "collaboration", "project management",
	}
	
	jobDescLower := strings.ToLower(jobDesc)
	found := []string{}
	
	for _, keyword := range commonKeywords {
		if strings.Contains(jobDescLower, keyword) {
			found = append(found, keyword)
		}
	}
	
	return found
}

// findMissingKeywords finds keywords from job description not present in resume
func findMissingKeywords(res resume.Resume, keywords []string) []string {
	resumeText := strings.ToLower(res.Summary)
	for _, exp := range res.WorkExperiences {
		resumeText += " " + strings.ToLower(exp.Description)
	}
	for _, skill := range res.Skills {
		resumeText += " " + strings.ToLower(skill.Name)
	}
	
	missing := []string{}
	for _, keyword := range keywords {
		if !strings.Contains(resumeText, keyword) {
			missing = append(missing, keyword)
		}
	}
	
	return missing
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// getAISuggestions uses Azure OpenAI to generate intelligent suggestions with suggested values
func getAISuggestions(res resume.Resume, jobTitle, jobDescription string) ([]Suggestion, error) {
	azure, err := azureOpenAIFromEnv()
	if err != nil {
		return nil, err
	}

	// Convert resume to JSON for the prompt
	resumeJSON, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal resume: %w", err)
	}

	// Build prompt
	prompt := fmt.Sprintf(`You are an expert resume reviewer. Analyze the following resume and provide specific, actionable suggestions for improvement.

Resume JSON:
%s

Job Title: %s
Job Description: %s

Provide suggestions in the following JSON format:
[
  {
    "section": "summary|contact_info|work_experiences|educations|skills|projects|certifications",
    "type": "keyword|completeness|formatting|rewrite",
    "message": "Brief explanation of the suggestion",
    "severity": "info|warning|error",
    "field": "specific field name if applicable",
    "suggested_value": "actual suggested text to replace/add (required for rewrite and keyword suggestions)"
  }
]

Focus on:
1. Missing keywords from the job description
2. Incomplete sections that need more detail
3. Formatting improvements
4. Rewriting sections for better impact and ATS compatibility

Return ONLY valid JSON array, no markdown or explanations.`, string(resumeJSON), jobTitle, jobDescription)

	// Call Azure OpenAI (the deployment picks the model)
	payload := map[string]interface{}{
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": "You are a professional resume optimizer. Return only valid JSON arrays, no markdown or explanations.",
			},
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"temperature": 0.3,
	}

	httpReq, err := azure.newChatRequest(azureDeployment("AZURE_OPENAI_MINI_DEPLOYMENT", "gpt-4o-mini"), payload)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Azure OpenAI returned %d", resp.StatusCode)
	}

	var apiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	content := apiResp.Choices[0].Message.Content
	// Clean up the content (remove markdown code blocks if present)
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```json") {
		content = strings.TrimPrefix(content, "```json")
		content = strings.TrimPrefix(content, "```")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	} else if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	}

	var aiSuggestions []Suggestion
	if err := json.Unmarshal([]byte(content), &aiSuggestions); err != nil {
		return nil, fmt.Errorf("unmarshal suggestions: %w", err)
	}

	return aiSuggestions, nil
}

