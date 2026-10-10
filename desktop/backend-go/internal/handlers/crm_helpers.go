package handlers

import (
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
)

// ============================================================================
// HELPER FUNCTIONS (CRM-specific to avoid redeclaration)
// ============================================================================

func crmToNullString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// crmJSONBytes converts an arbitrary value to JSON bytes for a JSONB column.
// Returns nil (SQL NULL) for empty/nil inputs so pgx doesn't send the bytes
// typed as bytea — which Postgres can't implicitly cast to jsonb and rejects
// with "invalid input syntax for type json".
func crmJSONBytes(v interface{}) []byte {
	switch x := v.(type) {
	case nil:
		return nil
	case map[string]interface{}:
		if len(x) == 0 {
			return nil
		}
	case []interface{}:
		if len(x) == 0 {
			return nil
		}
	case []string:
		if len(x) == 0 {
			return nil
		}
	}
	b, err := json.Marshal(v)
	if err != nil || string(b) == "null" {
		return nil
	}
	return b
}

func crmToNullUUID(s string) pgtype.UUID {
	if s == "" {
		return pgtype.UUID{}
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: id, Valid: true}
}

func crmToNumeric(f *float64) pgtype.Numeric {
	if f == nil {
		return pgtype.Numeric{}
	}
	var n pgtype.Numeric
	n.Scan(*f)
	return n
}

func crmNumericToFloat(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	f, _ := n.Float64Value()
	return f.Float64
}

func crmPtrToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func crmToNullActivityType(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// ============================================================================
// TRANSFORM FUNCTIONS
// ============================================================================

func transformCompany(c sqlc.Company) gin.H {
	return gin.H{
		"id":               crmUuidToString(c.ID),
		"user_id":          c.UserID,
		"name":             c.Name,
		"legal_name":       c.LegalName,
		"industry":         c.Industry,
		"company_size":     c.CompanySize,
		"website":          c.Website,
		"email":            c.Email,
		"phone":            c.Phone,
		"address_line1":    c.AddressLine1,
		"address_line2":    c.AddressLine2,
		"city":             c.City,
		"state":            c.State,
		"postal_code":      c.PostalCode,
		"country":          c.Country,
		"annual_revenue":   crmNumericToFloat(c.AnnualRevenue),
		"currency":         c.Currency,
		"linkedin_url":     c.LinkedinUrl,
		"twitter_handle":   c.TwitterHandle,
		"owner_id":         c.OwnerID,
		"lifecycle_stage":  c.LifecycleStage,
		"lead_source":      c.LeadSource,
		"health_score":     c.HealthScore,
		"engagement_score": c.EngagementScore,
		"logo_url":         c.LogoUrl,
		"custom_fields":    crmJsonToMap(c.CustomFields),
		"metadata":         crmJsonToMap(c.Metadata),
		"created_at":       c.CreatedAt.Time,
		"updated_at":       c.UpdatedAt.Time,
	}
}

func transformCompanies(companies []sqlc.Company) []gin.H {
	result := make([]gin.H, len(companies))
	for i, c := range companies {
		result[i] = transformCompany(c)
	}
	return result
}

func transformPipeline(p sqlc.Pipeline) gin.H {
	return gin.H{
		"id":            crmUuidToString(p.ID),
		"user_id":       p.UserID,
		"name":          p.Name,
		"description":   p.Description,
		"pipeline_type": p.PipelineType,
		"currency":      p.Currency,
		"is_default":    p.IsDefault,
		"is_active":     p.IsActive,
		"color":         p.Color,
		"icon":          p.Icon,
		"created_at":    p.CreatedAt.Time,
		"updated_at":    p.UpdatedAt.Time,
	}
}

func transformPipelines(pipelines []sqlc.Pipeline) []gin.H {
	result := make([]gin.H, len(pipelines))
	for i, p := range pipelines {
		result[i] = transformPipeline(p)
	}
	return result
}

func transformPipelineStage(s sqlc.PipelineStage) gin.H {
	return gin.H{
		"id":           crmUuidToString(s.ID),
		"pipeline_id":  crmUuidToString(s.PipelineID),
		"name":         s.Name,
		"description":  s.Description,
		"position":     s.Position,
		"probability":  s.Probability,
		"stage_type":   s.StageType,
		"rotting_days": s.RottingDays,
		"color":        s.Color,
		"created_at":   s.CreatedAt.Time,
		"updated_at":   s.UpdatedAt.Time,
	}
}

func transformPipelineStages(stages []sqlc.PipelineStage) []gin.H {
	result := make([]gin.H, len(stages))
	for i, s := range stages {
		result[i] = transformPipelineStage(s)
	}
	return result
}

func transformCRMDeal(d sqlc.ListCRMDealsRow) gin.H {
	return gin.H{
		"id":                  crmUuidToString(d.ID),
		"user_id":             d.UserID,
		"pipeline_id":         crmUuidToString(d.PipelineID),
		"pipeline_name":       d.PipelineName,
		"stage_id":            crmUuidToString(d.StageID),
		"stage_name":          d.StageName,
		"name":                d.Name,
		"description":         d.Description,
		"amount":              crmNumericToFloat(d.Amount),
		"currency":            d.Currency,
		"probability":         d.Probability,
		"expected_close_date": crmDateToString(d.ExpectedCloseDate),
		"actual_close_date":   crmDateToString(d.ActualCloseDate),
		"owner_id":            d.OwnerID,
		"company_id":          crmUuidToString(d.CompanyID),
		"company_name":        d.CompanyName,
		"primary_contact_id":  crmUuidToString(d.PrimaryContactID),
		"status":              d.Status,
		"lost_reason":         d.LostReason,
		"priority":            d.Priority,
		"lead_source":         d.LeadSource,
		"deal_score":          d.DealScore,
		"custom_fields":       crmJsonToMap(d.CustomFields),
		"created_at":          d.CreatedAt.Time,
		"updated_at":          d.UpdatedAt.Time,
	}
}

func transformCRMDealRow(d sqlc.GetCRMDealRow) gin.H {
	return gin.H{
		"id":                  crmUuidToString(d.ID),
		"user_id":             d.UserID,
		"pipeline_id":         crmUuidToString(d.PipelineID),
		"pipeline_name":       d.PipelineName,
		"stage_id":            crmUuidToString(d.StageID),
		"stage_name":          d.StageName,
		"name":                d.Name,
		"description":         d.Description,
		"amount":              crmNumericToFloat(d.Amount),
		"currency":            d.Currency,
		"probability":         d.Probability,
		"expected_close_date": crmDateToString(d.ExpectedCloseDate),
		"actual_close_date":   crmDateToString(d.ActualCloseDate),
		"owner_id":            d.OwnerID,
		"company_id":          crmUuidToString(d.CompanyID),
		"company_name":        d.CompanyName,
		"primary_contact_id":  crmUuidToString(d.PrimaryContactID),
		"status":              d.Status,
		"lost_reason":         d.LostReason,
		"priority":            d.Priority,
		"lead_source":         d.LeadSource,
		"deal_score":          d.DealScore,
		"custom_fields":       crmJsonToMap(d.CustomFields),
		"created_at":          d.CreatedAt.Time,
		"updated_at":          d.UpdatedAt.Time,
	}
}

func transformCRMDeals(deals []sqlc.ListCRMDealsRow) []gin.H {
	result := make([]gin.H, len(deals))
	for i, d := range deals {
		result[i] = transformCRMDeal(d)
	}
	return result
}

func transformCRMDealBasic(d sqlc.Deal) gin.H {
	return gin.H{
		"id":                  crmUuidToString(d.ID),
		"user_id":             d.UserID,
		"pipeline_id":         crmUuidToString(d.PipelineID),
		"stage_id":            crmUuidToString(d.StageID),
		"name":                d.Name,
		"description":         d.Description,
		"amount":              crmNumericToFloat(d.Amount),
		"currency":            d.Currency,
		"probability":         d.Probability,
		"expected_close_date": crmDateToString(d.ExpectedCloseDate),
		"actual_close_date":   crmDateToString(d.ActualCloseDate),
		"owner_id":            d.OwnerID,
		"company_id":          crmUuidToString(d.CompanyID),
		"primary_contact_id":  crmUuidToString(d.PrimaryContactID),
		"status":              d.Status,
		"lost_reason":         d.LostReason,
		"priority":            d.Priority,
		"lead_source":         d.LeadSource,
		"deal_score":          d.DealScore,
		"custom_fields":       crmJsonToMap(d.CustomFields),
		"created_at":          d.CreatedAt.Time,
		"updated_at":          d.UpdatedAt.Time,
	}
}

func transformCRMActivity(a sqlc.CrmActivity) gin.H {
	return gin.H{
		"id":                 crmUuidToString(a.ID),
		"user_id":            a.UserID,
		"activity_type":      a.ActivityType,
		"subject":            a.Subject,
		"description":        a.Description,
		"outcome":            a.Outcome,
		"deal_id":            crmUuidToString(a.DealID),
		"company_id":         crmUuidToString(a.CompanyID),
		"contact_id":         crmUuidToString(a.ContactID),
		"participants":       crmJsonToSlice(a.Participants),
		"activity_date":      crmTimestampToString(a.ActivityDate),
		"duration_minutes":   a.DurationMinutes,
		"call_direction":     a.CallDirection,
		"call_disposition":   a.CallDisposition,
		"call_recording_url": a.CallRecordingUrl,
		"email_direction":    a.EmailDirection,
		"email_message_id":   a.EmailMessageID,
		"meeting_location":   a.MeetingLocation,
		"meeting_url":        a.MeetingUrl,
		"owner_id":           a.OwnerID,
		"is_completed":       a.IsCompleted,
		"completed_by":       a.CompletedBy,
		"completed_at":       crmTimestampToString(a.CompletedAt),
		"created_at":         a.CreatedAt.Time,
		"updated_at":         a.UpdatedAt.Time,
	}
}

func transformCRMActivities(activities []sqlc.CrmActivity) []gin.H {
	result := make([]gin.H, len(activities))
	for i, a := range activities {
		result[i] = transformCRMActivity(a)
	}
	return result
}

func crmUuidToString(u pgtype.UUID) *string {
	if !u.Valid {
		return nil
	}
	s := uuid.UUID(u.Bytes).String()
	return &s
}

func crmDateToString(d pgtype.Date) *string {
	if !d.Valid {
		return nil
	}
	s := d.Time.Format("2006-01-02")
	return &s
}

func crmTimestampToString(t pgtype.Timestamptz) *string {
	if !t.Valid {
		return nil
	}
	s := t.Time.Format(time.RFC3339)
	return &s
}

func crmJsonToMap(b []byte) map[string]interface{} {
	if len(b) == 0 {
		return nil
	}
	var m map[string]interface{}
	json.Unmarshal(b, &m)
	return m
}

func crmJsonToSlice(b []byte) []interface{} {
	if len(b) == 0 {
		return nil
	}
	var s []interface{}
	json.Unmarshal(b, &s)
	return s
}

// ============================================================================
// ROUTE REGISTRATION
// ============================================================================

// RegisterCRMRoutes registers all CRM routes (companies, pipelines, deals, activities)
// on the given router group.
func RegisterCRMRoutes(api *gin.RouterGroup, h *CRMHandler, auth gin.HandlerFunc) {
	// CRM routes - /api/crm (full CRM pipeline system)
	crm := api.Group("/crm")
	crm.Use(auth, middleware.RequireAuth())
	{
		// Companies
		crm.GET("/companies", h.ListCompanies)
		crm.POST("/companies", h.CreateCompany)
		crm.GET("/companies/search", h.SearchCompanies)
		crm.GET("/companies/:id", h.GetCompany)
		crm.PUT("/companies/:id", h.UpdateCompany)
		crm.DELETE("/companies/:id", h.DeleteCompany)
		// Company contacts (linking to clients)
		crm.GET("/companies/:id/contacts", h.ListCompanyContacts)
		crm.POST("/companies/:id/contacts", h.LinkContactToCompany)
		crm.DELETE("/companies/:id/contacts/:relationId", h.UnlinkContactFromCompany)

		// Pipelines
		crm.GET("/pipelines", h.ListPipelines)
		crm.POST("/pipelines", h.CreatePipeline)
		crm.GET("/pipelines/:id", h.GetPipeline)
		crm.PUT("/pipelines/:id", h.UpdatePipeline)
		crm.DELETE("/pipelines/:id", h.DeletePipeline)
		// Pipeline stages
		crm.GET("/pipelines/:id/stages", h.ListPipelineStages)
		crm.POST("/pipelines/:id/stages", h.CreatePipelineStage)
		crm.PUT("/pipelines/:id/stages/:stageId", h.UpdatePipelineStage)
		crm.DELETE("/pipelines/:id/stages/:stageId", h.DeletePipelineStage)
		crm.POST("/pipelines/:id/stages/reorder", h.ReorderPipelineStages)

		// Deals (CRM pipeline deals)
		crm.GET("/deals", h.ListCRMDeals)
		crm.POST("/deals", h.CreateCRMDeal)
		crm.GET("/deals/stats", h.GetCRMDealStats)
		crm.GET("/deals/:id", h.GetCRMDeal)
		crm.PUT("/deals/:id", h.UpdateCRMDeal)
		crm.PATCH("/deals/:id/stage", h.MoveCRMDealStage)
		crm.PATCH("/deals/:id/status", h.UpdateCRMDealStatus)
		crm.DELETE("/deals/:id", h.DeleteCRMDeal)
		// Deal activities
		crm.GET("/deals/:id/activities", h.ListDealActivities)

		// Activities
		crm.GET("/activities", h.ListCRMActivities)
		crm.POST("/activities", h.CreateCRMActivity)
		crm.POST("/activities/:id/complete", h.CompleteCRMActivity)
		crm.DELETE("/activities/:id", h.DeleteCRMActivity)
	}

}
