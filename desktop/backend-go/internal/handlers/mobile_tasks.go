package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
)

// ListTasks returns paginated tasks.
// Query params: limit (1-50), cursor, status, due (today|week|overdue), fields
func (h *MobileHandler) ListTasks(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		MobileRespondUnauthorized(c)
		return
	}

	// Parse pagination
	limit := 20
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = ClampInt(parsed, 1, 50)
		}
	}

	// Parse cursor
	var cursorID uuid.UUID
	var cursorTime time.Time
	if cursor := c.Query("cursor"); cursor != "" {
		var err error
		cursorID, cursorTime, err = DecodeCursor(cursor)
		if err != nil {
			MobileRespondInvalidCursor(c)
			return
		}
	}

	// Parse filters
	statusFilter := c.Query("status")
	dueFilter := c.Query("due")

	// Parse field selection
	fields := ParseFieldsParam(c.Query("fields"))

	// Build query params
	params := sqlc.ListTasksForMobileParams{
		UserID:     user.ID,
		LimitCount: int32(limit + 1), // +1 to detect has_more
	}

	// Set cursor params if provided
	if !cursorTime.IsZero() {
		params.CursorUpdatedAt = pgtype.Timestamp{Time: cursorTime, Valid: true}
		params.CursorID = pgtype.UUID{Bytes: cursorID, Valid: true}
	}

	// Set status filter
	if statusFilter != "" {
		params.Status = sqlc.NullTaskstatus{
			Taskstatus: sqlc.Taskstatus(statusFilter),
			Valid:      true,
		}
	}

	// Set due filter
	if dueFilter != "" {
		params.DueFilter = &dueFilter
	}

	// Execute query
	ctx := c.Request.Context()
	rows, err := h.queries.ListTasksForMobile(ctx, params)
	if err != nil {
		MobileRespondInternalError(c)
		return
	}

	// Check if there are more results
	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	// Transform to mobile response
	tasks := make([]interface{}, 0, len(rows))
	var lastTask sqlc.ListTasksForMobileRow

	for _, row := range rows {
		task := transformMobileTaskRow(row)

		if len(fields) > 0 {
			tasks = append(tasks, SelectFields(task, fields))
		} else {
			tasks = append(tasks, task)
		}

		lastTask = row
	}

	// Build cursor for next page
	nextCursor := ""
	if hasMore && len(rows) > 0 {
		nextCursor = EncodeCursor(lastTask.ID.Bytes, lastTask.UpdatedAt.Time)
	}

	c.JSON(http.StatusOK, MobileTaskListResponse{
		Tasks:   tasks,
		Cursor:  nextCursor,
		HasMore: hasMore,
	})
}

// transformMobileTaskRow converts a ListTasksForMobileRow to MobileTaskResponse.
func transformMobileTaskRow(row sqlc.ListTasksForMobileRow) MobileTaskResponse {
	resp := MobileTaskResponse{
		ID:        row.ID.Bytes,
		Title:     row.Title,
		Status:    string(row.Status.Taskstatus),
		Priority:  string(row.Priority.Taskpriority),
		UpdatedAt: row.UpdatedAt.Time.Unix(),
	}

	if row.DueDate.Valid {
		s := row.DueDate.Time.Format("2006-01-02")
		resp.DueDate = &s
	}

	if row.ProjectName != nil {
		resp.Project = row.ProjectName
	}

	if row.AssigneeName != nil {
		resp.Assignee = row.AssigneeName
	}

	return resp
}

// GetTask returns the full detail view for a single task.
func (h *MobileHandler) GetTask(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		MobileRespondUnauthorized(c)
		return
	}

	taskID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		MobileRespondValidationError(c, "id", "valid UUID")
		return
	}

	ctx := c.Request.Context()

	row, err := h.queries.GetTaskForMobile(ctx, sqlc.GetTaskForMobileParams{
		ID:     pgtype.UUID{Bytes: taskID, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		MobileRespondNotFound(c, "Task")
		return
	}

	subtaskCount, _ := h.queries.CountSubtasksForTask(ctx, pgtype.UUID{Bytes: taskID, Valid: true})
	commentsCount, _ := h.queries.CountCommentsByEntity(ctx, sqlc.CountCommentsByEntityParams{
		EntityType: "task",
		EntityID:   pgtype.UUID{Bytes: taskID, Valid: true},
	})

	c.JSON(http.StatusOK, transformMobileTaskDetailRow(row, int(subtaskCount), int(commentsCount)))
}

// transformMobileTaskDetailRow converts GetTaskForMobileRow to MobileTaskDetailResponse.
func transformMobileTaskDetailRow(row sqlc.GetTaskForMobileRow, subtaskCount int, commentsCount int) MobileTaskDetailResponse {
	resp := MobileTaskDetailResponse{
		ID:            row.ID.Bytes,
		Title:         row.Title,
		Status:        string(row.Status.Taskstatus),
		Priority:      string(row.Priority.Taskpriority),
		Tags:          []string{},
		CommentsCount: commentsCount,
		SubtasksCount: subtaskCount,
		CreatedAt:     row.CreatedAt.Time.Unix(),
		UpdatedAt:     row.UpdatedAt.Time.Unix(),
	}

	if row.Description != nil {
		resp.Description = row.Description
	}

	if row.DueDate.Valid {
		s := row.DueDate.Time.Format("2006-01-02")
		resp.DueDate = &s
	}

	if row.StartDate.Valid {
		s := row.StartDate.Time.Format("2006-01-02")
		resp.StartDate = &s
	}

	if row.CompletedAt.Valid {
		s := row.CompletedAt.Time.Format(time.RFC3339)
		resp.CompletedAt = &s
	}

	if row.ProjectID.Valid && row.ProjectName != nil {
		resp.Project = &MobileProjectRefResponse{
			ID:   row.ProjectID.Bytes,
			Name: *row.ProjectName,
		}
	}

	if row.AssigneeUuid.Valid && row.AssigneeName != nil {
		resp.Assignee = &MobileAssigneeResponse{
			ID:   uuid.UUID(row.AssigneeUuid.Bytes).String(),
			Name: *row.AssigneeName,
		}
		if row.AssigneeAvatar != nil {
			resp.Assignee.AvatarURL = row.AssigneeAvatar
		}
	}

	return resp
}

// QuickCreateTask creates a task from a minimal mobile payload.
func (h *MobileHandler) QuickCreateTask(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		MobileRespondUnauthorized(c)
		return
	}

	var req MobileTaskQuickCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		MobileRespondValidationError(c, "body", "valid JSON with title field")
		return
	}

	params := sqlc.QuickCreateTaskParams{
		UserID: user.ID,
		Title:  req.Title,
	}

	if req.DueDate != nil {
		if t, err := time.Parse("2006-01-02", *req.DueDate); err == nil {
			params.DueDate = pgtype.Timestamp{Time: t, Valid: true}
		} else {
			MobileRespondValidationError(c, "due_date", "YYYY-MM-DD format")
			return
		}
	}

	if req.Priority != nil {
		params.Priority = sqlc.NullTaskpriority{
			Taskpriority: sqlc.Taskpriority(*req.Priority),
			Valid:        true,
		}
	}

	ctx := c.Request.Context()
	task, err := h.queries.QuickCreateTask(ctx, params)
	if err != nil {
		MobileRespondInternalError(c)
		return
	}

	response := MobileTaskResponse{
		ID:        task.ID.Bytes,
		Title:     task.Title,
		Status:    string(task.Status.Taskstatus),
		Priority:  string(task.Priority.Taskpriority),
		UpdatedAt: task.CreatedAt.Time.Unix(),
	}

	if task.DueDate.Valid {
		s := task.DueDate.Time.Format("2006-01-02")
		response.DueDate = &s
	}

	c.JSON(http.StatusCreated, response)
}

// UpdateTaskStatus sets an explicit status on a task.
func (h *MobileHandler) UpdateTaskStatus(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		MobileRespondUnauthorized(c)
		return
	}

	taskID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		MobileRespondValidationError(c, "id", "valid UUID")
		return
	}

	var req MobileTaskStatusUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		MobileRespondValidationError(c, "status", "one of: todo, in_progress, done, cancelled")
		return
	}

	ctx := c.Request.Context()
	result, err := h.queries.UpdateTaskStatusMobile(ctx, sqlc.UpdateTaskStatusMobileParams{
		ID:     pgtype.UUID{Bytes: taskID, Valid: true},
		Status: sqlc.Taskstatus(req.Status),
		UserID: user.ID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			MobileRespondNotFound(c, "Task")
			return
		}
		MobileRespondInternalError(c)
		return
	}

	response := MobileTaskStatusResponse{
		ID:        result.ID.Bytes,
		Status:    string(result.Status.Taskstatus),
		UpdatedAt: result.UpdatedAt.Time.Unix(),
	}

	if result.CompletedAt.Valid {
		s := result.CompletedAt.Time.Format(time.RFC3339)
		response.CompletedAt = &s
	}

	c.JSON(http.StatusOK, response)
}

// ToggleTask toggles between todo/done — designed for swipe gestures.
func (h *MobileHandler) ToggleTask(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		MobileRespondUnauthorized(c)
		return
	}

	taskID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		MobileRespondValidationError(c, "id", "valid UUID")
		return
	}

	ctx := c.Request.Context()
	result, err := h.queries.ToggleTaskStatusMobile(ctx, sqlc.ToggleTaskStatusMobileParams{
		ID:     pgtype.UUID{Bytes: taskID, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			MobileRespondNotFound(c, "Task")
			return
		}
		MobileRespondInternalError(c)
		return
	}

	response := MobileTaskStatusResponse{
		ID:        result.ID.Bytes,
		Status:    string(result.Status.Taskstatus),
		UpdatedAt: result.UpdatedAt.Time.Unix(),
	}

	if result.CompletedAt.Valid {
		s := result.CompletedAt.Time.Format(time.RFC3339)
		response.CompletedAt = &s
	}

	c.JSON(http.StatusOK, response)
}
