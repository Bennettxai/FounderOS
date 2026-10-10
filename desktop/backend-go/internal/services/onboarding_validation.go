package services

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"unicode/utf8"
)

// ValidationError represents a field validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Value   string `json:"value,omitempty"`
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationErrors is a collection of validation errors
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}
	var msgs []string
	for _, err := range e {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

func (e ValidationErrors) HasErrors() bool {
	return len(e) > 0
}

// Valid values for onboarding fields
var (
	ValidBusinessTypes = map[string]bool{
		"agency":     true,
		"startup":    true,
		"freelance":  true,
		"consulting": true,
		"ecommerce":  true,
		"saas":       true,
		"other":      true,
	}

	ValidTeamSizes = map[string]bool{
		"solo":  true,
		"2-5":   true,
		"6-10":  true,
		"11-50": true,
		"51+":   true,
	}

	ValidRoles = map[string]bool{
		"founder":    true,
		"executive":  true,
		"manager":    true,
		"employee":   true,
		"contractor": true,
	}

	ValidIntegrations = map[string]bool{
		"google":   true,
		"slack":    true,
		"notion":   true,
		"linear":   true,
		"hubspot":  true,
		"airtable": true,
		"fathom":   true,
		"github":   true,
		"figma":    true,
		"asana":    true,
		"trello":   true,
		"jira":     true,
	}

	// Company name regex: letters, numbers, spaces, hyphens, ampersands, periods, apostrophes
	companyNameRegex = regexp.MustCompile(`^[a-zA-Z0-9\s\-&.']+$`)
)

// OnboardingValidator handles validation for onboarding data
type OnboardingValidator struct{}

// NewOnboardingValidator creates a new validator
func NewOnboardingValidator() *OnboardingValidator {
	return &OnboardingValidator{}
}

// ValidateCompanyName validates the company/workspace name
func (v *OnboardingValidator) ValidateCompanyName(name string) *ValidationError {
	name = strings.TrimSpace(name)

	if name == "" {
		return &ValidationError{
			Field:   "company_name",
			Message: "Company name is required",
		}
	}

	length := utf8.RuneCountInString(name)
	if length < 2 {
		return &ValidationError{
			Field:   "company_name",
			Message: "Company name must be at least 2 characters",
			Value:   html.EscapeString(name),
		}
	}

	if length > 100 {
		return &ValidationError{
			Field:   "company_name",
			Message: "Company name must be less than 100 characters",
			Value:   html.EscapeString(name),
		}
	}

	if !companyNameRegex.MatchString(name) {
		return &ValidationError{
			Field:   "company_name",
			Message: "Company name contains invalid characters. Only letters, numbers, spaces, hyphens, ampersands, periods, and apostrophes are allowed",
			Value:   html.EscapeString(name),
		}
	}

	return nil
}

// SanitizeInput escapes HTML special characters to prevent XSS
func (v *OnboardingValidator) SanitizeInput(input string) string {
	return html.EscapeString(strings.TrimSpace(input))
}

// ValidateBusinessType validates the business type
func (v *OnboardingValidator) ValidateBusinessType(businessType string) *ValidationError {
	businessType = strings.ToLower(strings.TrimSpace(businessType))

	if businessType == "" {
		return &ValidationError{
			Field:   "business_type",
			Message: "Business type is required",
		}
	}

	if !ValidBusinessTypes[businessType] {
		return &ValidationError{
			Field:   "business_type",
			Message: fmt.Sprintf("Invalid business type. Must be one of: %s", getValidKeys(ValidBusinessTypes)),
			Value:   businessType,
		}
	}

	return nil
}

// ValidateTeamSize validates the team size
func (v *OnboardingValidator) ValidateTeamSize(teamSize string) *ValidationError {
	teamSize = strings.ToLower(strings.TrimSpace(teamSize))

	if teamSize == "" {
		return &ValidationError{
			Field:   "team_size",
			Message: "Team size is required",
		}
	}

	if !ValidTeamSizes[teamSize] {
		return &ValidationError{
			Field:   "team_size",
			Message: fmt.Sprintf("Invalid team size. Must be one of: %s", getValidKeys(ValidTeamSizes)),
			Value:   teamSize,
		}
	}

	return nil
}

// ValidateRole validates the user role
func (v *OnboardingValidator) ValidateRole(role string) *ValidationError {
	role = strings.ToLower(strings.TrimSpace(role))

	if role == "" {
		return &ValidationError{
			Field:   "role",
			Message: "Role is required",
		}
	}

	if !ValidRoles[role] {
		return &ValidationError{
			Field:   "role",
			Message: fmt.Sprintf("Invalid role. Must be one of: %s", getValidKeys(ValidRoles)),
			Value:   role,
		}
	}

	return nil
}

// ValidateChallenge validates the main challenge description
func (v *OnboardingValidator) ValidateChallenge(challenge string) *ValidationError {
	challenge = strings.TrimSpace(challenge)

	if challenge == "" {
		return &ValidationError{
			Field:   "challenge",
			Message: "Challenge description is required",
		}
	}

	length := utf8.RuneCountInString(challenge)
	if length < 10 {
		return &ValidationError{
			Field:   "challenge",
			Message: "Please provide a bit more detail about your challenge (at least 10 characters)",
			Value:   html.EscapeString(challenge),
		}
	}

	if length > 500 {
		return &ValidationError{
			Field:   "challenge",
			Message: "Challenge description is too long (max 500 characters)",
			Value:   html.EscapeString(challenge[:100]) + "...",
		}
	}

	return nil
}

// ValidateIntegrations validates the selected integrations
func (v *OnboardingValidator) ValidateIntegrations(integrations []string) *ValidationError {
	if len(integrations) == 0 {
		// Integrations are optional
		return nil
	}

	var invalidIntegrations []string
	for _, integration := range integrations {
		integration = strings.ToLower(strings.TrimSpace(integration))
		if !ValidIntegrations[integration] {
			invalidIntegrations = append(invalidIntegrations, integration)
		}
	}

	if len(invalidIntegrations) > 0 {
		return &ValidationError{
			Field:   "integrations",
			Message: fmt.Sprintf("Invalid integrations: %s. Valid options: %s", strings.Join(invalidIntegrations, ", "), getValidKeys(ValidIntegrations)),
			Value:   strings.Join(invalidIntegrations, ", "),
		}
	}

	return nil
}

// ValidateExtractedData validates all extracted onboarding data
func (v *OnboardingValidator) ValidateExtractedData(data *ExtractedOnboardingData) ValidationErrors {
	var errors ValidationErrors

	if data.WorkspaceName != "" {
		if err := v.ValidateCompanyName(data.WorkspaceName); err != nil {
			errors = append(errors, *err)
		}
	}

	if data.BusinessType != "" {
		if err := v.ValidateBusinessType(data.BusinessType); err != nil {
			errors = append(errors, *err)
		}
	}

	if data.TeamSize != "" {
		if err := v.ValidateTeamSize(data.TeamSize); err != nil {
			errors = append(errors, *err)
		}
	}

	if data.Role != "" {
		if err := v.ValidateRole(data.Role); err != nil {
			errors = append(errors, *err)
		}
	}

	if data.Challenge != "" {
		if err := v.ValidateChallenge(data.Challenge); err != nil {
			errors = append(errors, *err)
		}
	}

	if len(data.Integrations) > 0 {
		if err := v.ValidateIntegrations(data.Integrations); err != nil {
			errors = append(errors, *err)
		}
	}

	return errors
}

// ValidateForCompletion validates that all required fields are present and valid for completing onboarding
func (v *OnboardingValidator) ValidateForCompletion(data *ExtractedOnboardingData) ValidationErrors {
	var errors ValidationErrors

	// Company name is required
	if data.WorkspaceName == "" {
		errors = append(errors, ValidationError{
			Field:   "workspace_name",
			Message: "Company name is required to complete onboarding",
		})
	} else if err := v.ValidateCompanyName(data.WorkspaceName); err != nil {
		errors = append(errors, *err)
	}

	// Business type is required
	if data.BusinessType == "" {
		errors = append(errors, ValidationError{
			Field:   "business_type",
			Message: "Business type is required to complete onboarding",
		})
	} else if err := v.ValidateBusinessType(data.BusinessType); err != nil {
		errors = append(errors, *err)
	}

	// Team size is required
	if data.TeamSize == "" {
		errors = append(errors, ValidationError{
			Field:   "team_size",
			Message: "Team size is required to complete onboarding",
		})
	} else if err := v.ValidateTeamSize(data.TeamSize); err != nil {
		errors = append(errors, *err)
	}

	// Role is optional but must be valid if provided
	if data.Role != "" {
		if err := v.ValidateRole(data.Role); err != nil {
			errors = append(errors, *err)
		}
	}

	// Challenge is optional but must be valid if provided
	if data.Challenge != "" {
		if err := v.ValidateChallenge(data.Challenge); err != nil {
			errors = append(errors, *err)
		}
	}

	// Integrations are optional but must be valid if provided
	if len(data.Integrations) > 0 {
		if err := v.ValidateIntegrations(data.Integrations); err != nil {
			errors = append(errors, *err)
		}
	}

	return errors
}

// NormalizeBusinessType normalizes various business type inputs to standard values
func (v *OnboardingValidator) NormalizeBusinessType(input string) string {
	input = strings.ToLower(strings.TrimSpace(input))

	// Map common variations to standard values
	mappings := map[string]string{
		"marketing agency":    "agency",
		"creative agency":     "agency",
		"digital agency":      "agency",
		"design agency":       "agency",
		"tech startup":        "startup",
		"software startup":    "startup",
		"start-up":            "startup",
		"solo":                "freelance",
		"independent":         "freelance",
		"consultant":          "consulting",
		"e-commerce":          "ecommerce",
		"online store":        "ecommerce",
		"software":            "saas",
		"software as service": "saas",
	}

	if normalized, ok := mappings[input]; ok {
		return normalized
	}

	// Check if it's already a valid type
	if ValidBusinessTypes[input] {
		return input
	}

	return "other"
}

// NormalizeTeamSize normalizes various team size inputs to standard values
func (v *OnboardingValidator) NormalizeTeamSize(input string) string {
	input = strings.ToLower(strings.TrimSpace(input))

	// Map common variations
	mappings := map[string]string{
		"just me":   "solo",
		"1":         "solo",
		"one":       "solo",
		"myself":    "solo",
		"2":         "2-5",
		"3":         "2-5",
		"4":         "2-5",
		"5":         "2-5",
		"two":       "2-5",
		"three":     "2-5",
		"four":      "2-5",
		"five":      "2-5",
		"small":     "2-5",
		"6":         "6-10",
		"7":         "6-10",
		"8":         "6-10",
		"9":         "6-10",
		"10":        "6-10",
		"medium":    "11-50",
		"large":     "51+",
		"enterprise": "51+",
		"50+":       "51+",
		"100+":      "51+",
	}

	if normalized, ok := mappings[input]; ok {
		return normalized
	}

	if ValidTeamSizes[input] {
		return input
	}

	return "solo" // Default
}

// NormalizeRole normalizes various role inputs to standard values
func (v *OnboardingValidator) NormalizeRole(input string) string {
	input = strings.ToLower(strings.TrimSpace(input))

	mappings := map[string]string{
		"ceo":             "executive",
		"cto":             "executive",
		"cfo":             "executive",
		"coo":             "executive",
		"c-level":         "executive",
		"owner":           "founder",
		"co-founder":      "founder",
		"cofounder":       "founder",
		"team lead":       "manager",
		"team leader":     "manager",
		"project manager": "manager",
		"pm":              "manager",
		"developer":       "employee",
		"engineer":        "employee",
		"designer":        "employee",
		"freelancer":      "contractor",
		"consultant":      "contractor",
	}

	if normalized, ok := mappings[input]; ok {
		return normalized
	}

	if ValidRoles[input] {
		return input
	}

	return "employee" // Default
}

// ValidateFallbackForm validates the fallback form data
func (v *OnboardingValidator) ValidateFallbackForm(data *FallbackFormData) ValidationErrors {
	var errors ValidationErrors

	// Validate required fields from Quick Info
	if data.WorkspaceName == "" {
		errors = append(errors, ValidationError{
			Field:   "workspace_name",
			Message: "Workspace name is required",
		})
	} else if err := v.ValidateCompanyName(data.WorkspaceName); err != nil {
		errors = append(errors, *err)
	}

	if data.BusinessType == "" {
		errors = append(errors, ValidationError{
			Field:   "business_type",
			Message: "Business type is required",
		})
	} else if err := v.ValidateBusinessType(data.BusinessType); err != nil {
		errors = append(errors, *err)
	}

	// Validate fallback form fields (only if user filled out form)
	// If they connected integrations, fallback form is optional
	hasIntegrations := len(data.Integrations) > 0

	if !hasIntegrations {
		// Fallback form is required if no integrations
		if len(data.ToolsUsed) == 0 {
			errors = append(errors, ValidationError{
				Field:   "tools_used",
				Message: "Please select at least one tool you use (or connect an integration)",
			})
		}

		if data.MainFocus == "" {
			errors = append(errors, ValidationError{
				Field:   "main_focus",
				Message: "Please select your main work focus (or connect an integration)",
			})
		}

		if data.Challenge == "" {
			errors = append(errors, ValidationError{
				Field:   "challenge",
				Message: "Please describe your biggest challenge (or connect an integration)",
			})
		} else if err := v.ValidateChallenge(data.Challenge); err != nil {
			errors = append(errors, *err)
		}

		if data.WorkStyle == "" {
			errors = append(errors, ValidationError{
				Field:   "work_style",
				Message: "Please select your work style (or connect an integration)",
			})
		}

		if len(data.WhatWouldHelp) == 0 {
			errors = append(errors, ValidationError{
				Field:   "what_would_help",
				Message: "Please select at least one thing that would help (or connect an integration)",
			})
		}

		if len(data.WhatWouldHelp) > 3 {
			errors = append(errors, ValidationError{
				Field:   "what_would_help",
				Message: "Please select up to 3 options",
				Value:   fmt.Sprintf("%d selected", len(data.WhatWouldHelp)),
			})
		}
	}

	// Validate integrations if provided
	if len(data.Integrations) > 0 {
		if err := v.ValidateIntegrations(data.Integrations); err != nil {
			errors = append(errors, *err)
		}
	}

	return errors
}

// Helper function to get valid keys as a formatted string
func getValidKeys(m map[string]bool) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return strings.Join(keys, ", ")
}
