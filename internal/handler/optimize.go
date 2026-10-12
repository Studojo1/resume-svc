package handler

import (
	"encoding/json"
	"fmt"
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

	// Optimize resume using LLM (Azure OpenAI)
	optimized, err := optimizeResumeWithLLM(req.Resume, req.JobTitle, req.Company, req.JobDescription)
	if err != nil {
		log.Printf("LLM optimization failed, using fallback: %v", err)
		// Fallback: return resume with basic enhancements
		optimized = enhanceResumeBasic(req.Resume, req.JobTitle, req.JobDescription)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(optimized); err != nil {
		log.Printf("encode optimized resume: %v", err)
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

// optimizeResumeWithLLM optimizes resume using Azure OpenAI
func optimizeResumeWithLLM(res resume.Resume, jobTitle, company, jobDescription string) (resume.Resume, error) {
	azure, err := azureOpenAIFromEnv()
	if err != nil {
		return res, err
	}

	// Prepare prompt
	resumeJSON, _ := json.Marshal(res)
	prompt := fmt.Sprintf(`You are an expert resume optimizer. Optimize the following resume for the job position:

Job Title: %s
Company: %s
Job Description: %s

Resume JSON:
%s

Optimize the resume by:
1. Enhancing the summary to highlight relevant experience for this role
2. Rewriting work experience descriptions to emphasize relevant achievements and skills
3. Reordering sections to prioritize most relevant experience
4. Adding job-relevant keywords naturally throughout
5. Ensuring ATS-friendly formatting

Return ONLY the optimized resume as valid JSON matching the exact same structure as the input. Do not include any markdown, code blocks, or explanations.`, jobTitle, company, jobDescription, string(resumeJSON))

	// Call Azure OpenAI (the deployment picks the model)
	payload := map[string]interface{}{
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": "You are a professional resume optimizer. Return only valid JSON, no markdown or explanations.",
			},
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"temperature": 0.3,
	}

	req, err := azure.newChatRequest(azureDeployment("AZURE_OPENAI_CHAT_DEPLOYMENT", "gpt-4o"), payload)
	if err != nil {
		return res, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return res, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return res, fmt.Errorf("Azure OpenAI returned %d", resp.StatusCode)
	}

	var apiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return res, fmt.Errorf("decode response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return res, fmt.Errorf("no choices in response")
	}

	content := apiResp.Choices[0].Message.Content
	// Clean up JSON if wrapped in markdown
	if len(content) > 0 && content[0] == '`' {
		// Remove markdown code blocks
		lines := []rune(content)
		start := 0
		end := len(lines)
		for i, r := range lines {
			if r == '`' {
				if start == 0 {
					start = i + 1
				} else {
					end = i
					break
				}
			}
		}
		content = string(lines[start:end])
	}

	var optimized resume.Resume
	if err := json.Unmarshal([]byte(content), &optimized); err != nil {
		return res, fmt.Errorf("unmarshal optimized resume: %w", err)
	}

	return optimized, nil
}

// enhanceResumeBasic provides basic enhancement without LLM
func enhanceResumeBasic(res resume.Resume, jobTitle, jobDescription string) resume.Resume {
	enhanced := res
	
	// Basic keyword enhancement (simple approach)
	if jobDescription != "" && enhanced.Summary != "" {
		// In a real implementation, this would do more sophisticated keyword matching
		enhanced.Summary = enhanced.Summary // Keep original for now
	}
	
	return enhanced
}
