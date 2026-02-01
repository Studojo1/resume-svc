package tex

import (
	"archive/zip"
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/studojo/jobs/internal/domain/resume"
)

type Generator struct {
	tempDir string
}

func NewGenerator() *Generator {
	return &Generator{
		tempDir: os.TempDir(),
	}
}

// GeneratePDF produces a PDF from resume data using LaTeX.
func (g *Generator) GeneratePDF(res *resume.Resume) ([]byte, error) {
	latex, err := g.generateResumeLaTeX(res)
	if err != nil {
		return nil, fmt.Errorf("generate LaTeX: %w", err)
	}

	pdf, err := g.compileLaTeX(latex, "resume")
	if err != nil {
		return nil, fmt.Errorf("compile LaTeX: %w", err)
	}

	return pdf, nil
}

// GeneratePackage generates Resume, Cover Letter, and CV as a zip file.
func (g *Generator) GeneratePackage(res *resume.Resume, jobTitle, company, jobDescription string) ([]byte, error) {
	// Generate Resume PDF
	resumeLaTeX, err := g.generateResumeLaTeX(res)
	if err != nil {
		return nil, fmt.Errorf("generate resume LaTeX: %w", err)
	}
	resumePDF, err := g.compileLaTeX(resumeLaTeX, "resume")
	if err != nil {
		return nil, fmt.Errorf("compile resume: %w", err)
	}

	// Generate Cover Letter PDF
	coverLetterLaTeX, err := g.generateCoverLetterLaTeX(res, jobTitle, company, jobDescription)
	if err != nil {
		return nil, fmt.Errorf("generate cover letter LaTeX: %w", err)
	}
	coverLetterPDF, err := g.compileLaTeX(coverLetterLaTeX, "cover-letter")
	if err != nil {
		return nil, fmt.Errorf("compile cover letter: %w", err)
	}

	// Generate CV PDF (same as resume for now)
	cvPDF := resumePDF

	// Create zip file
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	files := []struct {
		name string
		data []byte
	}{
		{"Resume.pdf", resumePDF},
		{"Cover_Letter.pdf", coverLetterPDF},
		{"CV.pdf", cvPDF},
	}

	for _, file := range files {
		f, err := zipWriter.Create(file.name)
		if err != nil {
			return nil, fmt.Errorf("create zip entry: %w", err)
		}
		if _, err := f.Write(file.data); err != nil {
			return nil, fmt.Errorf("write zip entry: %w", err)
		}
	}

	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("close zip: %w", err)
	}

	return buf.Bytes(), nil
}

func escapeLaTeX(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\textbackslash{}")
	s = strings.ReplaceAll(s, "&", "\\&")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "$", "\\$")
	s = strings.ReplaceAll(s, "#", "\\#")
	s = strings.ReplaceAll(s, "^", "\\textasciicircum{}")
	s = strings.ReplaceAll(s, "_", "\\_")
	s = strings.ReplaceAll(s, "{", "\\{")
	s = strings.ReplaceAll(s, "}", "\\}")
	s = strings.ReplaceAll(s, "~", "\\textasciitilde{}")
	return s
}

func loadTemplate(name string) (string, error) {
	// Try to load from templates directory relative to the binary
	paths := []string{
		filepath.Join("templates", name+".tex"),
		filepath.Join("/app/templates", name+".tex"),
		filepath.Join(".", "templates", name+".tex"),
	}

	for _, path := range paths {
		if data, err := os.ReadFile(path); err == nil {
			return string(data), nil
		}
	}

	return "", fmt.Errorf("template %s not found", name)
}

func (g *Generator) generateResumeLaTeX(res *resume.Resume) (string, error) {
	tmpl, err := loadTemplate("resume")
	if err != nil {
		return "", err
	}

	// Build contact info (single line format)
	var contactParts []string
	if res.ContactInfo.Email != "" {
		contactParts = append(contactParts, fmt.Sprintf("\\href{mailto:%s}{%s}", res.ContactInfo.Email, escapeLaTeX(res.ContactInfo.Email)))
	}
	if res.ContactInfo.Website != "" {
		contactParts = append(contactParts, fmt.Sprintf("\\href{https://%s}{%s}", res.ContactInfo.Website, escapeLaTeX(res.ContactInfo.Website)))
	}
	if res.ContactInfo.LinkedIn != "" {
		contactParts = append(contactParts, fmt.Sprintf("\\href{https://%s}{%s}", res.ContactInfo.LinkedIn, escapeLaTeX(res.ContactInfo.LinkedIn)))
	}
	if res.ContactInfo.Phone != "" {
		contactParts = append(contactParts, escapeLaTeX(res.ContactInfo.Phone))
	}
	contactInfo := strings.Join(contactParts, " $|$ ")

	// Build summary
	summary := ""
	if res.Summary != "" {
		summary = fmt.Sprintf("\\section*{PROFESSIONAL SUMMARY}\n%s\n\\vspace{6pt}\n", escapeLaTeX(res.Summary))
	}

	// Build experience using \entry command
	experience := ""
	if len(res.WorkExperiences) > 0 {
		experience = "\\section{EXPERIENCE}\n"
		for _, exp := range res.WorkExperiences {
			company := escapeLaTeX(exp.Company)
			location := ""
			role := escapeLaTeX(exp.Role)
			dateRange := ""
			if exp.StartDate != nil && *exp.StartDate != "" {
				startDate := escapeLaTeX(*exp.StartDate)
				if exp.EndDate != nil && *exp.EndDate != "" {
					dateRange = fmt.Sprintf("%s -- %s", startDate, escapeLaTeX(*exp.EndDate))
				} else if exp.IsCurrent {
					dateRange = fmt.Sprintf("%s -- Present", startDate)
				} else {
					dateRange = startDate
				}
			}
			experience += fmt.Sprintf("\\entry{%s}{%s}{%s}{%s}\n", company, location, role, dateRange)
			if exp.Description != "" {
				experience += "\\begin{itemize}[leftmargin=1em, noitemsep, topsep=2pt]\n"
				for _, line := range strings.Split(exp.Description, "\n") {
					line = strings.TrimSpace(line)
					if line != "" {
						experience += fmt.Sprintf("\\item %s\n", escapeLaTeX(line))
					}
				}
				experience += "\\end{itemize}\n"
			}
			experience += "\\vspace{6pt}\n\n"
		}
	}

	// Build education using \entry command
	education := ""
	if len(res.Educations) > 0 {
		education = "\\section{EDUCATION}\n"
		for _, edu := range res.Educations {
			institution := escapeLaTeX(edu.Institution)
			location := ""
			degree := escapeLaTeX(edu.Degree)
			if edu.FieldOfStudy != "" {
				degree += ", " + escapeLaTeX(edu.FieldOfStudy)
			}
			dateRange := ""
			if edu.StartDate != nil && *edu.StartDate != "" {
				startDate := escapeLaTeX(*edu.StartDate)
				if edu.EndDate != nil && *edu.EndDate != "" {
					dateRange = fmt.Sprintf("%s -- %s", startDate, escapeLaTeX(*edu.EndDate))
				} else if edu.IsCurrent {
					dateRange = fmt.Sprintf("%s -- Present", startDate)
				} else {
					dateRange = startDate
				}
			}
			education += fmt.Sprintf("\\entry{%s}{%s}{%s}{%s}\n", institution, location, degree, dateRange)
			if edu.Description != "" {
				education += escapeLaTeX(edu.Description) + "\n"
			}
			education += "\\vspace{6pt}\n\n"
		}
	}

	// Build skills by category
	skills := ""
	if len(res.Skills) > 0 {
		skillGroups := make(map[string][]string)
		for _, skill := range res.Skills {
			category := skill.Category
			if category == "" {
				category = "Other"
			}
			if skill.Name != "" {
				skillGroups[category] = append(skillGroups[category], escapeLaTeX(skill.Name))
			}
		}
		if len(skillGroups) > 0 {
			skills = "\\section{TECHNICAL SKILLS}\n"
			// Map categories to standard labels
			categoryMap := map[string]string{
				"Languages": "Languages",
				"Programming Languages": "Languages",
				"Frameworks": "Frameworks",
				"DevOps": "DevOps",
				"Tools": "Tools",
				"Specializations": "Specializations",
			}
			var parts []string
			for category, names := range skillGroups {
				label := categoryMap[category]
				if label == "" {
					label = category
				}
				parts = append(parts, fmt.Sprintf("\\textbf{%s}: %s", escapeLaTeX(label), strings.Join(names, ", ")))
			}
			skills += strings.Join(parts, " \\\\\n") + "\n"
			skills += "\\vspace{6pt}\n"
		}
	}

	// Build projects using \project command
	projects := ""
	if len(res.Projects) > 0 {
		projects = "\\section{PROJECTS}\n"
		for _, proj := range res.Projects {
			title := escapeLaTeX(proj.Title)
			dateRange := ""
			if proj.StartDate != nil && *proj.StartDate != "" {
				startDate := escapeLaTeX(*proj.StartDate)
				if proj.EndDate != nil && *proj.EndDate != "" {
					dateRange = fmt.Sprintf("%s -- %s", startDate, escapeLaTeX(*proj.EndDate))
				} else {
					dateRange = startDate
				}
			}
			technologies := ""
			if proj.URL != "" {
				technologies = fmt.Sprintf("\\href{https://%s}{%s}", proj.URL, escapeLaTeX(proj.URL))
			}
			projects += fmt.Sprintf("\\project{%s}{%s}{%s}\n", title, dateRange, technologies)
			if proj.Description != "" {
				projects += "\\begin{itemize}[leftmargin=1em, noitemsep, topsep=2pt]\n"
				for _, line := range strings.Split(proj.Description, "\n") {
					line = strings.TrimSpace(line)
					if line != "" {
						projects += fmt.Sprintf("\\item %s\n", escapeLaTeX(line))
					}
				}
				projects += "\\end{itemize}\n"
			}
			projects += "\\vspace{6pt}\n\n"
		}
	}

	// Build certifications as achievements using \project command
	achievements := ""
	if len(res.Certifications) > 0 {
		achievements = "\\section{ACHIEVEMENTS}\n"
		for _, cert := range res.Certifications {
			title := escapeLaTeX(cert.Name)
			date := ""
			if cert.IssueDate != nil && *cert.IssueDate != "" {
				date = escapeLaTeX(*cert.IssueDate)
			}
			category := escapeLaTeX(cert.Issuer)
			if cert.URL != "" {
				category += fmt.Sprintf(" | \\href{https://%s}{%s}", cert.URL, escapeLaTeX(cert.URL))
			}
			achievements += fmt.Sprintf("\\project{%s}{%s}{%s}\n", title, date, category)
			achievements += "\\begin{itemize}[leftmargin=1em, noitemsep, topsep=2pt]\n"
			achievements += fmt.Sprintf("\\item %s\n", escapeLaTeX(cert.Name))
			achievements += "\\end{itemize}\n"
			achievements += "\\vspace{4pt}\n\n"
		}
	}

	// Replace placeholders
	name := escapeLaTeX(res.ContactInfo.Name)
	if name == "" {
		name = "Resume"
	}

	tmpl = strings.ReplaceAll(tmpl, "{{NAME}}", name)
	tmpl = strings.ReplaceAll(tmpl, "{{CONTACT_INFO}}", contactInfo)
	tmpl = strings.ReplaceAll(tmpl, "{{SUMMARY}}", summary)
	tmpl = strings.ReplaceAll(tmpl, "{{EDUCATION}}", education)
	tmpl = strings.ReplaceAll(tmpl, "{{EXPERIENCE}}", experience)
	tmpl = strings.ReplaceAll(tmpl, "{{PROJECTS}}", projects)
	tmpl = strings.ReplaceAll(tmpl, "{{SKILLS}}", skills)
	tmpl = strings.ReplaceAll(tmpl, "{{ACHIEVEMENTS}}", achievements)

	return tmpl, nil
}

func (g *Generator) generateCoverLetterLaTeX(res *resume.Resume, jobTitle, company, jobDescription string) (string, error) {
	tmpl, err := loadTemplate("cover-letter")
	if err != nil {
		return "", err
	}

	// Build contact info (formatted for new template)
	var contactLines []string
	if res.ContactInfo.Name != "" {
		contactLines = append(contactLines, escapeLaTeX(res.ContactInfo.Name))
	}
	address := ""
	if res.ContactInfo.Location != "" {
		address = escapeLaTeX(res.ContactInfo.Location)
	}
	if address != "" {
		contactLines = append(contactLines, address)
	}
	contactInfo := strings.Join(contactLines, "\\\\\n")
	if res.ContactInfo.Email != "" {
		contactInfo += fmt.Sprintf("\\\\\n\\href{mailto:%s}{%s}", res.ContactInfo.Email, escapeLaTeX(res.ContactInfo.Email))
	}
	if res.ContactInfo.Phone != "" {
		contactInfo += fmt.Sprintf("\\\\\n%s", escapeLaTeX(res.ContactInfo.Phone))
	}

	// Build company info
	companyInfo := ""
	if company != "" {
		companyInfo = fmt.Sprintf("Hiring Manager\\\\\n%s", escapeLaTeX(company))
	}

	// Build subject
	subject := ""
	if jobTitle != "" {
		subject = fmt.Sprintf("Re: %s Position", escapeLaTeX(jobTitle))
	}

	// Build date (current date)
	dateStr := "\\today"

	// Build body
	var firstRole, firstSkill string
	if len(res.WorkExperiences) > 0 && res.WorkExperiences[0].Role != "" {
		firstRole = escapeLaTeX(res.WorkExperiences[0].Role)
	}
	if len(res.Skills) > 0 && res.Skills[0].Name != "" {
		firstSkill = escapeLaTeX(res.Skills[0].Name)
	}

	body := fmt.Sprintf("I am writing to express my strong interest in the %s position at %s.", escapeLaTeX(jobTitle), escapeLaTeX(company))
	if firstRole != "" {
		body += fmt.Sprintf(" With my background in %s", firstRole)
		if firstSkill != "" {
			body += fmt.Sprintf(" and experience in %s", firstSkill)
		}
		body += ","
	}
	body += " I am confident that I would be a valuable addition to your team.\n\n"

	if res.Summary != "" {
		body += escapeLaTeX(res.Summary) + "\n\n"
	}

	if jobDescription != "" {
		body += fmt.Sprintf("Based on the job description, I am particularly excited about the opportunity to contribute to %s's mission. My experience aligns well with the requirements, and I am eager to bring my skills and passion to this role.\n\n", escapeLaTeX(company))
	}

	body += fmt.Sprintf("I would welcome the opportunity to discuss how my background, skills, and enthusiasm can contribute to %s's continued success. Thank you for considering my application.", escapeLaTeX(company))

	signature := escapeLaTeX(res.ContactInfo.Name)
	if signature == "" {
		signature = "Your Name"
	}
	signatureLine := fmt.Sprintf("Sincerely,\\\\[2cm]\\rule{7cm}{0.4pt}\\\\%s", signature)

	// Replace placeholders
	tmpl = strings.ReplaceAll(tmpl, "{{CONTACT_INFO}}", contactInfo)
	tmpl = strings.ReplaceAll(tmpl, "{{DATE}}", dateStr)
	tmpl = strings.ReplaceAll(tmpl, "{{COMPANY_INFO}}", companyInfo)
	tmpl = strings.ReplaceAll(tmpl, "{{SUBJECT}}", subject)
	tmpl = strings.ReplaceAll(tmpl, "{{BODY}}", body)
	tmpl = strings.ReplaceAll(tmpl, "{{SIGNATURE}}", signatureLine)

	return tmpl, nil
}

func (g *Generator) generateCVLaTeX(res *resume.Resume) (string, error) {
	// CV is same as resume for now
	return g.generateResumeLaTeX(res)
}

func (g *Generator) compileLaTeX(latexContent, baseName string) ([]byte, error) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp(g.tempDir, "latex-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	texFile := filepath.Join(tmpDir, baseName+".tex")
	if err := os.WriteFile(texFile, []byte(latexContent), 0644); err != nil {
		return nil, fmt.Errorf("write tex file: %w", err)
	}

	// Run pdflatex twice for proper references
	cmd := exec.Command("pdflatex", "-interaction=nonstopmode", "-output-directory", tmpDir, baseName+".tex")
	cmd.Dir = tmpDir
	var stderr bytes.Buffer
	var stdout bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		log.Printf("pdflatex first pass warning: %s", stderr.String())
	}

	// Second pass for references
	cmd = exec.Command("pdflatex", "-interaction=nonstopmode", "-output-directory", tmpDir, baseName+".tex")
	cmd.Dir = tmpDir
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		// Check if PDF was created despite errors
		pdfFile := filepath.Join(tmpDir, baseName+".pdf")
		if _, statErr := os.Stat(pdfFile); statErr == nil {
			// PDF exists, read it even if there were warnings
			pdfData, readErr := os.ReadFile(pdfFile)
			if readErr == nil {
				return pdfData, nil
			}
		}
		return nil, fmt.Errorf("pdflatex failed: %w, stderr: %s, stdout: %s", err, stderr.String(), stdout.String())
	}

	pdfFile := filepath.Join(tmpDir, baseName+".pdf")
	pdfData, err := os.ReadFile(pdfFile)
	if err != nil {
		return nil, fmt.Errorf("read pdf: %w", err)
	}

	return pdfData, nil
}

