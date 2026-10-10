package handlers

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/utils"
)

// ============================================================================
// CRM ACTIVITIES HANDLERS
// ============================================================================

// ListCRMActivities returns CRM activities for the current user
func (h *CRMHandler) ListCRMActivities(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	queries := sqlc.New(h.pool)

	// Parse query params
	activityType := c.Query("activity_type")
	isCompleted := c.Query("is_completed")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	var isCompletedBool *bool
	if isCompleted != "" {
		b := isCompleted == "true"
		isCompletedBool = &b
	}

	activities, err := queries.ListCRMActivities(c.Request.Context(), sqlc.ListCRMActivitiesParams{
		UserID:       user.ID,
		ActivityType: crmToNullActivityType(activityType),
		IsCompleted:  isCompletedBool,
		LimitVal:     int32(limit),
		OffsetVal:    int32(offset),
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "list activities", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"activities": transformCRMActivities(activities),
		"count":      len(activities),
	})
}

// ListDealActivities returns activities for a specific deal
func (h *CRMHandler) ListDealActivities(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	dealID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "deal_id")
		return
	}

	queries := sqlc.New(h.pool)
	activities, err := queries.ListDealActivities(c.Request.Context(), pgtype.UUID{Bytes: dealID, Valid: true})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "list activities", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"activities": transformCRMActivities(activities),
		"count":      len(activities),
	})
}

// CreateCRMActivityRequest represents the request to create a CRM activity
type CreateCRMActivityRequest struct {
	ActivityType    string   `json:"activity_type" binding:"required"`
	Subject         string   `json:"subject" binding:"required"`
	Description     *string  `json:"description"`
	Outcome         *string  `json:"outcome"`
	DealID          *string  `json:"deal_id"`
	CompanyID       *string  `json:"company_id"`
	ContactID       *string  `json:"contact_id"`
	Participants    []string `json:"participants"`
	ActivityDate    string   `json:"activity_date" binding:"required"`
	DurationMinutes *int32   `json:"duration_minutes"`
	// Call-specific
	CallDirection    *string `json:"call_direction"`
	CallDisposition  *string `json:"call_disposition"`
	CallRecordingURL *string `json:"call_recording_url"`
	// Email-specific
	EmailDirection *string `json:"email_direction"`
	EmailMessageID *string `json:"email_message_id"`
	// Meeting-specific
	MeetingLocation *string `json:"meeting_location"`
	MeetingURL      *string `json:"meeting_url"`
	// Completion
	OwnerID     *string `json:"owner_id"`
	IsCompleted bool    `json:"is_completed"`
}

// CreateCRMActivity creates a new CRM activity
func (h *CRMHandler) CreateCRMActivity(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	var req CreateCRMActivityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	queries := sqlc.New(h.pool)

	// Parse activity date
	activityDate, err := time.Parse(time.RFC3339, req.ActivityDate)
	if err != nil {
		activityDate, err = time.Parse("2006-01-02", req.ActivityDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid activity date format"})
			return
		}
	}

	// Convert participants to JSON
	participants := crmJSONBytes(req.Participants)

	activity, err := queries.CreateCRMActivity(c.Request.Context(), sqlc.CreateCRMActivityParams{
		UserID:           user.ID,
		ActivityType:     req.ActivityType,
		Subject:          req.Subject,
		Description:      req.Description,
		Outcome:          req.Outcome,
		DealID:           crmToNullUUID(crmPtrToString(req.DealID)),
		CompanyID:        crmToNullUUID(crmPtrToString(req.CompanyID)),
		ContactID:        crmToNullUUID(crmPtrToString(req.ContactID)),
		Participants:     participants,
		ActivityDate:     pgtype.Timestamptz{Time: activityDate, Valid: true},
		DurationMinutes:  req.DurationMinutes,
		CallDirection:    req.CallDirection,
		CallDisposition:  req.CallDisposition,
		CallRecordingUrl: req.CallRecordingURL,
		EmailDirection:   req.EmailDirection,
		EmailMessageID:   req.EmailMessageID,
		MeetingLocation:  req.MeetingLocation,
		MeetingUrl:       req.MeetingURL,
		OwnerID:          req.OwnerID,
		IsCompleted:      &req.IsCompleted,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "create activity", err)
		return
	}

	actBody := req.Subject
	if req.Description != nil {
		actBody += "\n\n" + *req.Description
	}
	actMeta := map[string]string{"crm_kind": "activity", "activity_type": req.ActivityType}
	if req.Outcome != nil {
		actMeta["outcome"] = *req.Outcome
	}
	h.enqueue(c.Request.Context(), services.Signal{
		Module:     services.ModuleCRM,
		ID:         "activity-" + uuidString(activity.ID.Bytes),
		AuthorID:   user.ID,
		Title:      req.Subject,
		Body:       actBody,
		Genre:      req.ActivityType,
		ModifiedAt: pgTimestampToTime(activity.UpdatedAt),
		Metadata:   actMeta,
	})

	c.JSON(http.StatusCreated, transformCRMActivity(activity))
}

// CompleteCRMActivity marks an activity as completed
func (h *CRMHandler) CompleteCRMActivity(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	activityID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "activity_id")
		return
	}

	var req struct {
		Outcome *string `json:"outcome"`
	}
	c.ShouldBindJSON(&req)

	queries := sqlc.New(h.pool)
	activity, err := queries.CompleteCRMActivity(c.Request.Context(), sqlc.CompleteCRMActivityParams{
		ID:          pgtype.UUID{Bytes: activityID, Valid: true},
		UserID:      user.ID,
		CompletedBy: &user.ID,
		Outcome:     req.Outcome,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "complete activity", nil)
		return
	}

	c.JSON(http.StatusOK, transformCRMActivity(activity))
}

// DeleteCRMActivity deletes a CRM activity
func (h *CRMHandler) DeleteCRMActivity(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	activityID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "activity_id")
		return
	}

	queries := sqlc.New(h.pool)
	err = queries.DeleteCRMActivity(c.Request.Context(), sqlc.DeleteCRMActivityParams{
		ID:     pgtype.UUID{Bytes: activityID, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "delete activity", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Activity deleted"})
}
