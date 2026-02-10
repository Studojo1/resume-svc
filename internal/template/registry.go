package template

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Template represents a resume template with metadata
type Template struct {
	ID          string `json:"id"`
	Version     string `json:"version"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	LaTeXFile   string `json:"latex_file"`
}

var (
	registry     map[string]*Template
	registryOnce sync.Once
)

// LoadRegistry loads templates from templates.json
// TODO: Update to load templates from blob storage (localstack/Azure) via API
// Templates should be fetched from /api/resumes/templates endpoint or directly from blob storage
// Current implementation uses filesystem as fallback
func LoadRegistry() error {
	var err error
	registryOnce.Do(func() {
		registry = make(map[string]*Template)
		
		// TODO: Load templates from blob storage first
		// 1. Check environment (USE_LOCALSTACK)
		// 2. Load templates from blob storage: templates/{templateId}/metadata.json
		// 3. Fallback to filesystem if blob storage unavailable
		
		// Try multiple paths for templates.json (filesystem fallback)
		paths := []string{
			"templates/templates.json",
			"/app/templates/templates.json",
			"./templates/templates.json",
		}
		
		var data []byte
		for _, path := range paths {
			if data, err = os.ReadFile(path); err == nil {
				break
			}
		}
		
		if err != nil {
			// If file doesn't exist, create default registry
			registry = getDefaultRegistry()
			return
		}
		
		var templates []Template
		if err = json.Unmarshal(data, &templates); err != nil {
			registry = getDefaultRegistry()
			return
		}
		
		for i := range templates {
			t := &templates[i]
			registry[t.ID] = t
		}
	})
	
	return err
}

// GetTemplate returns a template by ID, or default if not found
func GetTemplate(id string) (*Template, error) {
	if err := LoadRegistry(); err != nil {
		// Continue with default registry
	}
	
	if id == "" {
		id = "modern"
	}
	
	t, ok := registry[id]
	if !ok {
		// Return default template
		return registry["modern"], nil
	}
	
	return t, nil
}

// ListTemplates returns all available templates
func ListTemplates() ([]*Template, error) {
	if err := LoadRegistry(); err != nil {
		// Continue with default registry
	}
	
	templates := make([]*Template, 0, len(registry))
	for _, t := range registry {
		templates = append(templates, t)
	}
	
	return templates, nil
}

// getDefaultRegistry returns the default template registry
func getDefaultRegistry() map[string]*Template {
	return map[string]*Template{
		"modern": {
			ID:          "modern",
			Version:     "1.0",
			Name:        "Modern",
			Description: "Clean, professional design with modern typography",
			Category:    "professional",
			LaTeXFile:   "modern/resume.tex",
		},
		"classic": {
			ID:          "classic",
			Version:     "1.0",
			Name:        "Classic",
			Description: "Traditional format with timeless elegance",
			Category:    "traditional",
			LaTeXFile:   "classic/resume.tex",
		},
		"minimal": {
			ID:          "minimal",
			Version:     "1.0",
			Name:        "Minimal",
			Description: "Clean, ATS-friendly format with maximum readability",
			Category:    "ats-friendly",
			LaTeXFile:   "minimal/resume.tex",
		},
		"executive": {
			ID:          "executive",
			Version:     "1.0",
			Name:        "Executive",
			Description: "Sophisticated design for senior roles",
			Category:    "executive",
			LaTeXFile:   "executive/resume.tex",
		},
		"creative": {
			ID:          "creative",
			Version:     "1.0",
			Name:        "Creative",
			Description: "Bold design for creative and design roles",
			Category:    "creative",
			LaTeXFile:   "creative/resume.tex",
		},
	}
}

// LoadTemplate loads a template file by template ID and file type (resume, cover-letter)
// TODO: Update to load LaTeX files from blob storage
// Should load from: templates/{templateId}/resume.tex in blob storage
// Current implementation uses filesystem as fallback
func LoadTemplate(templateID, fileType string) (string, error) {
	template, err := GetTemplate(templateID)
	if err != nil {
		return "", fmt.Errorf("get template: %w", err)
	}
	
	// TODO: Load LaTeX file from blob storage first
	// 1. Check if template.LaTeXFile is a blob storage URL
	// 2. If yes, download from blob storage
	// 3. Fallback to filesystem
	
	// Construct file path
	fileName := fileType + ".tex"
	basePath := filepath.Dir(template.LaTeXFile)
	filePath := filepath.Join(basePath, fileName)
	
	// Try multiple paths (filesystem fallback)
	paths := []string{
		filepath.Join("templates", filePath),
		filepath.Join("/app/templates", filePath),
		filepath.Join(".", "templates", filePath),
		filepath.Join("templates", templateID, fileName),
		filepath.Join("/app/templates", templateID, fileName),
		filepath.Join(".", "templates", templateID, fileName),
	}
	
	for _, path := range paths {
		if data, err := os.ReadFile(path); err == nil {
			return string(data), nil
		}
	}
	
	return "", fmt.Errorf("template file not found: %s/%s", templateID, fileName)
}

