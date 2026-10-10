package handlers

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/database/sqlc"
)

// TeamMember response transformation
type TeamMemberResponse struct {
	ID         string   `json:"id"`
	UserID     string   `json:"user_id"`
	Name       string   `json:"name"`
	Email      string   `json:"email"`
	Role       string   `json:"role"`
	AvatarUrl  *string  `json:"avatar_url"`
	Status     string   `json:"status"`
	Capacity   int32    `json:"capacity"`
	ManagerID  *string  `json:"manager_id"`
	Skills     []string `json:"skills"`
	HourlyRate *float64 `json:"hourly_rate"`
	JoinedAt   string   `json:"joined_at"`
	CreatedAt  string   `json:"created_at"`
	UpdatedAt  string   `json:"updated_at"`
}

func TransformTeamMember(m sqlc.TeamMember) TeamMemberResponse {
	status := "available"
	if m.Status.Valid {
		status = strings.ToLower(string(m.Status.Memberstatus))
	}

	capacity := int32(100)
	if m.Capacity != nil {
		capacity = *m.Capacity
	}

	var skills []string
	if m.Skills != nil {
		json.Unmarshal(m.Skills, &skills)
	}
	if skills == nil {
		skills = []string{}
	}

	return TeamMemberResponse{
		ID:         pgtypeUUIDToStringRequired(m.ID),
		UserID:     m.UserID,
		Name:       m.Name,
		Email:      m.Email,
		Role:       m.Role,
		AvatarUrl:  m.AvatarUrl,
		Status:     status,
		Capacity:   capacity,
		ManagerID:  pgtypeUUIDToString(m.ManagerID),
		Skills:     skills,
		HourlyRate: pgtypeNumericToFloat(m.HourlyRate),
		JoinedAt:   m.JoinedAt.Time.Format(time.RFC3339),
		CreatedAt:  m.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:  m.UpdatedAt.Time.Format(time.RFC3339),
	}
}

func TransformTeamMembers(members []sqlc.TeamMember) []TeamMemberResponse {
	result := make([]TeamMemberResponse, len(members))
	for i, m := range members {
		result[i] = TransformTeamMember(m)
	}
	return result
}

// TeamMemberListResponse for list view with additional computed fields
type TeamMemberListResponse struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Email          string  `json:"email"`
	Role           string  `json:"role"`
	AvatarUrl      *string `json:"avatar_url"`
	Status         string  `json:"status"`
	Capacity       int32   `json:"capacity"`
	ManagerID      *string `json:"manager_id"`
	ActiveProjects int     `json:"active_projects"`
	OpenTasks      int     `json:"open_tasks"`
	JoinedAt       string  `json:"joined_at"`
}

func TransformTeamMemberListRow(m sqlc.ListTeamMembersRow) TeamMemberListResponse {
	status := "available"
	if m.Status.Valid {
		status = strings.ToLower(string(m.Status.Memberstatus))
	}

	capacity := int32(100)
	if m.Capacity != nil {
		capacity = *m.Capacity
	}

	return TeamMemberListResponse{
		ID:             pgtypeUUIDToStringRequired(m.ID),
		Name:           m.Name,
		Email:          m.Email,
		Role:           m.Role,
		AvatarUrl:      m.AvatarUrl,
		Status:         status,
		Capacity:       capacity,
		ManagerID:      pgtypeUUIDToString(m.ManagerID),
		ActiveProjects: int(m.ActiveProjectCount),
		OpenTasks:      int(m.ActiveTaskCount),
		JoinedAt:       m.JoinedAt.Time.Format(time.RFC3339),
	}
}

func TransformTeamMemberListRows(members []sqlc.ListTeamMembersRow) []TeamMemberListResponse {
	result := make([]TeamMemberListResponse, len(members))
	for i, m := range members {
		result[i] = TransformTeamMemberListRow(m)
	}
	return result
}

// Project response transformation
type ProjectResponse struct {
	ID              string                 `json:"id"`
	UserID          string                 `json:"user_id"`
	Name            string                 `json:"name"`
	Description     *string                `json:"description"`
	Status          string                 `json:"status"`
	Priority        string                 `json:"priority"`
	ClientName      *string                `json:"client_name"`
	ProjectType     string                 `json:"project_type"`
	ProjectMetadata map[string]interface{} `json:"project_metadata"`
	Notes           []ProjectNoteResponse  `json:"notes"`
	CreatedAt       string                 `json:"created_at"`
	UpdatedAt       string                 `json:"updated_at"`
}

type ProjectNoteResponse struct {
	ID        string `json:"id"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

func TransformProject(p sqlc.Project) ProjectResponse {
	status := "active"
	if p.Status.Valid {
		status = strings.ToLower(string(p.Status.Projectstatus))
	}

	priority := "medium"
	if p.Priority.Valid {
		priority = strings.ToLower(string(p.Priority.Projectpriority))
	}

	projectType := "other"
	if p.ProjectType != nil {
		projectType = *p.ProjectType
	}

	var metadata map[string]interface{}
	if p.ProjectMetadata != nil {
		json.Unmarshal(p.ProjectMetadata, &metadata)
	}

	return ProjectResponse{
		ID:              pgtypeUUIDToStringRequired(p.ID),
		UserID:          p.UserID,
		Name:            p.Name,
		Description:     p.Description,
		Status:          status,
		Priority:        priority,
		ClientName:      p.ClientName,
		ProjectType:     projectType,
		ProjectMetadata: metadata,
		Notes:           []ProjectNoteResponse{}, // Will be populated separately
		CreatedAt:       p.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:       p.UpdatedAt.Time.Format(time.RFC3339),
	}
}

func TransformProjects(projects []sqlc.Project) []ProjectResponse {
	result := make([]ProjectResponse, len(projects))
	for i, p := range projects {
		result[i] = TransformProject(p)
	}
	return result
}

func TransformProjectNote(n sqlc.ProjectNote) ProjectNoteResponse {
	return ProjectNoteResponse{
		ID:        pgtypeUUIDToStringRequired(n.ID),
		Content:   n.Content,
		CreatedAt: n.CreatedAt.Time.Format(time.RFC3339),
	}
}

// Context response transformation
type ContextResponse struct {
	ID                   string                   `json:"id"`
	UserID               string                   `json:"user_id"`
	Name                 string                   `json:"name"`
	Type                 string                   `json:"type"`
	Content              *string                  `json:"content"`
	StructuredData       map[string]interface{}   `json:"structured_data"`
	SystemPromptTemplate *string                  `json:"system_prompt_template"`
	Blocks               []map[string]interface{} `json:"blocks"`
	CoverImage           *string                  `json:"cover_image"`
	Icon                 *string                  `json:"icon"`
	ParentID             *string                  `json:"parent_id"`
	IsTemplate           bool                     `json:"is_template"`
	IsArchived           bool                     `json:"is_archived"`
	LastEditedAt         *string                  `json:"last_edited_at"`
	WordCount            int32                    `json:"word_count"`
	IsPublic             bool                     `json:"is_public"`
	ShareID              *string                  `json:"share_id"`
	PropertySchema       []map[string]interface{} `json:"property_schema"`
	Properties           map[string]interface{}   `json:"properties"`
	ClientID             *string                  `json:"client_id"`
	CreatedAt            string                   `json:"created_at"`
	UpdatedAt            string                   `json:"updated_at"`
}

func TransformContext(ctx sqlc.Context) ContextResponse {
	contextType := "document"
	if ctx.Type.Valid {
		contextType = strings.ToLower(string(ctx.Type.Contexttype))
	}

	isTemplate := false
	if ctx.IsTemplate != nil {
		isTemplate = *ctx.IsTemplate
	}

	isArchived := false
	if ctx.IsArchived != nil {
		isArchived = *ctx.IsArchived
	}

	wordCount := int32(0)
	if ctx.WordCount != nil {
		wordCount = *ctx.WordCount
	}

	isPublic := false
	if ctx.IsPublic != nil {
		isPublic = *ctx.IsPublic
	}

	var structuredData map[string]interface{}
	if ctx.StructuredData != nil {
		json.Unmarshal(ctx.StructuredData, &structuredData)
	}

	var blocks []map[string]interface{}
	if ctx.Blocks != nil {
		json.Unmarshal(ctx.Blocks, &blocks)
	}

	var propertySchema []map[string]interface{}
	if ctx.PropertySchema != nil {
		json.Unmarshal(ctx.PropertySchema, &propertySchema)
	}

	var properties map[string]interface{}
	if ctx.Properties != nil {
		json.Unmarshal(ctx.Properties, &properties)
	}

	return ContextResponse{
		ID:                   pgtypeUUIDToStringRequired(ctx.ID),
		UserID:               ctx.UserID,
		Name:                 ctx.Name,
		Type:                 contextType,
		Content:              ctx.Content,
		StructuredData:       structuredData,
		SystemPromptTemplate: ctx.SystemPromptTemplate,
		Blocks:               blocks,
		CoverImage:           ctx.CoverImage,
		Icon:                 ctx.Icon,
		ParentID:             pgtypeUUIDToString(ctx.ParentID),
		IsTemplate:           isTemplate,
		IsArchived:           isArchived,
		LastEditedAt:         pgtypeTimestampToString(ctx.LastEditedAt),
		WordCount:            wordCount,
		IsPublic:             isPublic,
		ShareID:              ctx.ShareID,
		PropertySchema:       propertySchema,
		Properties:           properties,
		ClientID:             pgtypeUUIDToString(ctx.ClientID),
		CreatedAt:            ctx.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:            ctx.UpdatedAt.Time.Format(time.RFC3339),
	}
}

func TransformContexts(contexts []sqlc.Context) []ContextResponse {
	result := make([]ContextResponse, len(contexts))
	for i, ctx := range contexts {
		result[i] = TransformContext(ctx)
	}
	return result
}

// Node response transformation
type NodeResponse struct {
	ID              string   `json:"id"`
	UserID          string   `json:"user_id"`
	ParentID        *string  `json:"parent_id"`
	ContextID       *string  `json:"context_id"`
	Name            string   `json:"name"`
	Type            string   `json:"type"`
	Health          string   `json:"health"`
	Purpose         *string  `json:"purpose"`
	CurrentStatus   *string  `json:"current_status"`
	ThisWeekFocus   []string `json:"this_week_focus"`
	DecisionQueue   []string `json:"decision_queue"`
	DelegationReady []string `json:"delegation_ready"`
	IsActive        bool     `json:"is_active"`
	IsArchived      bool     `json:"is_archived"`
	SortOrder       int32    `json:"sort_order"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
}

func TransformNode(n sqlc.Node) NodeResponse {
	nodeType := strings.ToLower(string(n.Type))

	health := "not_started"
	if n.Health.Valid {
		health = strings.ToLower(string(n.Health.Nodehealth))
	}

	isActive := false
	if n.IsActive != nil {
		isActive = *n.IsActive
	}

	isArchived := false
	if n.IsArchived != nil {
		isArchived = *n.IsArchived
	}

	sortOrder := int32(0)
	if n.SortOrder != nil {
		sortOrder = *n.SortOrder
	}

	var thisWeekFocus []string
	if n.ThisWeekFocus != nil {
		json.Unmarshal(n.ThisWeekFocus, &thisWeekFocus)
	}
	if thisWeekFocus == nil {
		thisWeekFocus = []string{}
	}

	var decisionQueue []string
	if n.DecisionQueue != nil {
		json.Unmarshal(n.DecisionQueue, &decisionQueue)
	}
	if decisionQueue == nil {
		decisionQueue = []string{}
	}

	var delegationReady []string
	if n.DelegationReady != nil {
		json.Unmarshal(n.DelegationReady, &delegationReady)
	}
	if delegationReady == nil {
		delegationReady = []string{}
	}

	return NodeResponse{
		ID:              pgtypeUUIDToStringRequired(n.ID),
		UserID:          n.UserID,
		ParentID:        pgtypeUUIDToString(n.ParentID),
		ContextID:       pgtypeUUIDToString(n.ContextID),
		Name:            n.Name,
		Type:            nodeType,
		Health:          health,
		Purpose:         n.Purpose,
		CurrentStatus:   n.CurrentStatus,
		ThisWeekFocus:   thisWeekFocus,
		DecisionQueue:   decisionQueue,
		DelegationReady: delegationReady,
		IsActive:        isActive,
		IsArchived:      isArchived,
		SortOrder:       sortOrder,
		CreatedAt:       n.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:       n.UpdatedAt.Time.Format(time.RFC3339),
	}
}

func TransformNodes(nodes []sqlc.Node) []NodeResponse {
	result := make([]NodeResponse, len(nodes))
	for i, n := range nodes {
		result[i] = TransformNode(n)
	}
	return result
}
