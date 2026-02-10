package resume

// ContactInfo holds basic contact details.
type ContactInfo struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Location string `json:"location"`
	LinkedIn string `json:"linkedin"`
	Website  string `json:"website"`
}

// Resume is the input for PDF generation. No IDs or user context.
type Resume struct {
	Title       string      `json:"title"`
	Summary     string      `json:"summary"`
	ContactInfo ContactInfo `json:"contact_info"`
	TemplateID  string      `json:"template_id,omitempty"` // Template ID for rendering

	WorkExperiences []WorkExperience `json:"work_experiences,omitempty"`
	Educations      []Education      `json:"educations,omitempty"`
	Skills          []Skill          `json:"skills,omitempty"`
	Projects        []Project        `json:"projects,omitempty"`
	Certifications  []Certification  `json:"certifications,omitempty"`
}

type WorkExperience struct {
	Company     string  `json:"company"`
	Role        string  `json:"role"`
	StartDate   *string `json:"start_date"`
	EndDate     *string `json:"end_date"`
	IsCurrent   bool    `json:"is_current"`
	Description string  `json:"description"`
}

type Education struct {
	Institution  string  `json:"institution"`
	Degree       string  `json:"degree"`
	FieldOfStudy string  `json:"field_of_study"`
	StartDate    *string `json:"start_date"`
	EndDate      *string `json:"end_date"`
	IsCurrent    bool    `json:"is_current"`
	Description  string  `json:"description"`
}

type Skill struct {
	Category    string `json:"category"`
	Name        string `json:"name"`
	Proficiency string `json:"proficiency"`
}

type Project struct {
	Title       string  `json:"title"`
	URL         string  `json:"url"`
	StartDate   *string `json:"start_date"`
	EndDate     *string `json:"end_date"`
	Description string  `json:"description"`
}

type Certification struct {
	Name       string  `json:"name"`
	Issuer     string  `json:"issuer"`
	IssueDate  *string `json:"issue_date"`
	ExpiryDate *string `json:"expiry_date"`
	URL        string  `json:"url"`
}
