package handlers

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/utils"
)

// listWorkspaceTasksQuery mirrors the sqlc ListTasks query (same columns,
// filters, and ordering) but scopes by workspace_id instead of user_id so
// active workspace members see shared tasks.
const listWorkspaceTasksQuery = `
SELECT id, user_id, title, description, status, priority, due_date, start_date, completed_at, project_id, assignee_id, parent_task_id, custom_status_id, position, created_at, updated_at
FROM tasks
WHERE workspace_id = $1
  AND ($2::taskstatus IS NULL OR status = $2)
  AND ($3::taskpriority IS NULL OR priority = $3)
  AND ($4::uuid IS NULL OR project_id = $4)
ORDER BY
  CASE priority WHEN 'critical' THEN 1 WHEN 'high' THEN 2 WHEN 'medium' THEN 3 WHEN 'low' THEN 4 END,
  due_date ASC NULLS LAST,
  created_at DESC`

// taskWorkspaceScope resolves the X-Workspace-ID header and confirms active
// membership, same model as calendar events. Returns (workspaceID, true) when
// the caller is an active member, or (uuid.Nil, false) to fall through to the
// user-scoped path.
func (h *DashboardItemHandler) taskWorkspaceScope(c *gin.Context, userID string) (uuid.UUID, bool) {
	hdr := c.GetHeader("X-Workspace-ID")
	if hdr == "" {
		return uuid.Nil, false
	}
	wsID, err := uuid.Parse(hdr)
	if err != nil {
		return uuid.Nil, false
	}
	var member bool
	err = h.pool.QueryRow(c.Request.Context(),
		`SELECT EXISTS(SELECT 1 FROM workspace_members WHERE workspace_id=$1 AND user_id=$2 AND status='active')`,
		wsID, userID).Scan(&member)
	if err != nil || !member {
		return uuid.Nil, false
	}
	return wsID, true
}

// ListTasks returns all tasks for the current user
func (h *DashboardItemHandler) ListTasks(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	queries := sqlc.New(h.pool)

	// Parse optional filters
	statusQuery := c.Query("status")
	var status sqlc.NullTaskstatus
	if statusQuery != "" {
		status = sqlc.NullTaskstatus{Taskstatus: stringToTaskStatus(statusQuery), Valid: true}
	}

	priorityQuery := c.Query("priority")
	var priority sqlc.NullTaskpriority
	if priorityQuery != "" {
		priority = sqlc.NullTaskpriority{Taskpriority: stringToTaskPriority(priorityQuery), Valid: true}
	}

	var projectID pgtype.UUID
	if pid := c.Query("project_id"); pid != "" {
		if parsed, err := uuid.Parse(pid); err == nil {
			projectID = pgtype.UUID{Bytes: parsed, Valid: true}
		}
	}

	// Workspace branch: when an active workspace is resolved, list the
	// workspace's tasks so teammates see shared work. Falls through to the
	// user-scoped path otherwise.
	if wsID, scoped := h.taskWorkspaceScope(c, user.ID); scoped {
		rows, err := h.pool.Query(c.Request.Context(), listWorkspaceTasksQuery,
			wsID, status, priority, projectID)
		if err != nil {
			slog.Default().Error("Failed to list workspace tasks", "error", err, "user_id", user.ID, "workspace_id", wsID)
			utils.RespondInternalError(c, slog.Default(), "list tasks", err)
			return
		}
		defer rows.Close()

		wsTasks := []sqlc.Task{}
		for rows.Next() {
			var t sqlc.Task
			if err := rows.Scan(
				&t.ID,
				&t.UserID,
				&t.Title,
				&t.Description,
				&t.Status,
				&t.Priority,
				&t.DueDate,
				&t.StartDate,
				&t.CompletedAt,
				&t.ProjectID,
				&t.AssigneeID,
				&t.ParentTaskID,
				&t.CustomStatusID,
				&t.Position,
				&t.CreatedAt,
				&t.UpdatedAt,
			); err != nil {
				utils.RespondInternalError(c, slog.Default(), "scan workspace task", err)
				return
			}
			wsTasks = append(wsTasks, t)
		}
		if err := rows.Err(); err != nil {
			utils.RespondInternalError(c, slog.Default(), "iterate workspace tasks", err)
			return
		}

		c.JSON(http.StatusOK, TransformTasks(wsTasks))
		return
	}

	tasks, err := queries.ListTasks(c.Request.Context(), sqlc.ListTasksParams{
		UserID:    user.ID,
		Status:    status,
		Priority:  priority,
		ProjectID: projectID,
	})
	if err != nil {
		slog.Default().Error("Failed to list tasks", "error", err, "user_id", user.ID)
		utils.RespondInternalError(c, slog.Default(), "list tasks", err)
		return
	}

	c.JSON(http.StatusOK, TransformTasks(tasks))
}

// CreateTask creates a new task
func (h *DashboardItemHandler) CreateTask(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	var req struct {
		Title       string  `json:"title" binding:"required"`
		Description *string `json:"description"`
		Status      *string `json:"status"`
		Priority    *string `json:"priority"`
		DueDate     *string `json:"due_date"`
		ProjectID   *string `json:"project_id"`
		AssigneeID  *string `json:"assignee_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	queries := sqlc.New(h.pool)

	// Parse optional fields
	var status sqlc.NullTaskstatus
	if req.Status != nil {
		status = sqlc.NullTaskstatus{
			Taskstatus: stringToTaskStatus(*req.Status),
			Valid:      true,
		}
	}

	var priority sqlc.NullTaskpriority
	if req.Priority != nil {
		priority = sqlc.NullTaskpriority{
			Taskpriority: stringToTaskPriority(*req.Priority),
			Valid:        true,
		}
	}

	var dueDate pgtype.Timestamp
	if req.DueDate != nil {
		if t, err := time.Parse(time.RFC3339, *req.DueDate); err == nil {
			dueDate = pgtype.Timestamp{Time: t, Valid: true}
		}
	}

	var projectID, assigneeID pgtype.UUID
	if req.ProjectID != nil {
		if parsed, err := uuid.Parse(*req.ProjectID); err == nil {
			projectID = pgtype.UUID{Bytes: parsed, Valid: true}
		}
	}
	if req.AssigneeID != nil {
		if parsed, err := uuid.Parse(*req.AssigneeID); err == nil {
			assigneeID = pgtype.UUID{Bytes: parsed, Valid: true}
		}
	}

	task, err := queries.CreateTask(c.Request.Context(), sqlc.CreateTaskParams{
		UserID:      user.ID,
		Title:       req.Title,
		Description: req.Description,
		Status:      status,
		Priority:    priority,
		DueDate:     dueDate,
		ProjectID:   projectID,
		AssigneeID:  assigneeID,
	})
	if err != nil {
		slog.Default().Error("Failed to create task", "error", err, "user_id", user.ID, "title", req.Title)
		utils.RespondInternalError(c, slog.Default(), "create task", err)
		return
	}

	// Invalidate dashboard caches when task is created
	h.invalidateDashboardCache(c, user.ID)

	// Mirror into the OptimalEngine knowledge graph so tasks appear in
	// the Pages graph view alongside everything else.
	taskMeta := map[string]string{}
	if req.Status != nil {
		taskMeta["status"] = *req.Status
	}
	if req.Priority != nil {
		taskMeta["priority"] = *req.Priority
	}
	if req.ProjectID != nil {
		taskMeta["project_id"] = *req.ProjectID
	}
	if req.AssigneeID != nil {
		taskMeta["assignee_id"] = *req.AssigneeID
	}
	h.enqueue(c.Request.Context(), services.Signal{
		Module:     services.ModuleTasks,
		ID:         "task-" + uuidString(task.ID.Bytes),
		AuthorID:   user.ID,
		Title:      req.Title,
		Body:       optString(req.Description),
		Genre:      "task",
		ModifiedAt: pgPlainTimestampToTime(task.UpdatedAt),
		Metadata:   taskMeta,
	})

	// Trigger notification if task was assigned to someone else
	if h.notificationTriggers != nil && req.AssigneeID != nil && *req.AssigneeID != user.ID {
		taskID := uuid.UUID(task.ID.Bytes)
		var projID *uuid.UUID
		if task.ProjectID.Valid {
			id := uuid.UUID(task.ProjectID.Bytes)
			projID = &id
		}
		go h.notificationTriggers.OnTaskAssigned(c.Request.Context(), services.TaskAssignedInput{
			TaskID:       taskID,
			TaskTitle:    task.Title,
			AssigneeID:   *req.AssigneeID,
			AssignerID:   user.ID,
			AssignerName: user.Name,
			ProjectID:    projID,
		})
	}

	c.JSON(http.StatusCreated, TransformTask(task))
}

// UpdateTask updates a task
func (h *DashboardItemHandler) UpdateTask(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "task ID")
		return
	}

	var req struct {
		Title       *string `json:"title"`
		Description *string `json:"description"`
		Status      *string `json:"status"`
		Priority    *string `json:"priority"`
		DueDate     *string `json:"due_date"`
		ProjectID   *string `json:"project_id"`
		AssigneeID  *string `json:"assignee_id"`
		Position    *int32  `json:"position"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	queries := sqlc.New(h.pool)

	// Get existing task (for comparison and ownership verification)
	existingTask, err := queries.GetTask(c.Request.Context(), sqlc.GetTaskParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Task")
		return
	}

	// Parse optional fields
	var status sqlc.NullTaskstatus
	if req.Status != nil {
		status = sqlc.NullTaskstatus{
			Taskstatus: stringToTaskStatus(*req.Status),
			Valid:      true,
		}
	}

	var priority sqlc.NullTaskpriority
	if req.Priority != nil {
		priority = sqlc.NullTaskpriority{
			Taskpriority: stringToTaskPriority(*req.Priority),
			Valid:        true,
		}
	}

	var dueDate pgtype.Timestamp
	if req.DueDate != nil {
		if t, err := time.Parse(time.RFC3339, *req.DueDate); err == nil {
			dueDate = pgtype.Timestamp{Time: t, Valid: true}
		}
	}

	var projectID, assigneeID pgtype.UUID
	if req.ProjectID != nil {
		if parsed, err := uuid.Parse(*req.ProjectID); err == nil {
			projectID = pgtype.UUID{Bytes: parsed, Valid: true}
		}
	}
	if req.AssigneeID != nil {
		if parsed, err := uuid.Parse(*req.AssigneeID); err == nil {
			assigneeID = pgtype.UUID{Bytes: parsed, Valid: true}
		}
	}

	task, err := queries.UpdateTask(c.Request.Context(), sqlc.UpdateTaskParams{
		ID:          pgtype.UUID{Bytes: id, Valid: true},
		Title:       req.Title,
		Description: req.Description,
		Status:      status,
		Priority:    priority,
		DueDate:     dueDate,
		ProjectID:   projectID,
		AssigneeID:  assigneeID,
		Position:    req.Position,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "update task", err)
		return
	}

	// Invalidate dashboard caches when task is updated
	h.invalidateDashboardCache(c, user.ID)

	// Trigger notifications for changes
	if h.notificationTriggers != nil {
		taskID := uuid.UUID(task.ID.Bytes)
		var projID *uuid.UUID
		if task.ProjectID.Valid {
			pid := uuid.UUID(task.ProjectID.Bytes)
			projID = &pid
		}

		// Check if assignee changed (new assignment)
		oldAssigneeID := ""
		if existingTask.AssigneeID.Valid {
			oldAssigneeID = uuid.UUID(existingTask.AssigneeID.Bytes).String()
		}
		newAssigneeID := ""
		if task.AssigneeID.Valid {
			newAssigneeID = uuid.UUID(task.AssigneeID.Bytes).String()
		}

		if newAssigneeID != "" && newAssigneeID != oldAssigneeID && newAssigneeID != user.ID {
			go h.notificationTriggers.OnTaskAssigned(c.Request.Context(), services.TaskAssignedInput{
				TaskID:       taskID,
				TaskTitle:    task.Title,
				AssigneeID:   newAssigneeID,
				AssignerID:   user.ID,
				AssignerName: user.Name,
				ProjectID:    projID,
			})
		}

		// Check if status changed to completed
		if req.Status != nil {
			newStatus := strings.ToLower(*req.Status)
			oldStatus := strings.ToLower(string(existingTask.Status.Taskstatus))
			if newStatus == "completed" && oldStatus != "completed" {
				go h.notificationTriggers.OnTaskCompleted(c.Request.Context(), services.TaskCompletedInput{
					TaskID:        taskID,
					TaskTitle:     task.Title,
					CompletedByID: user.ID,
					CompletedBy:   user.Name,
					OwnerID:       task.UserID,
					ProjectID:     projID,
				})
			} else if newStatus != oldStatus && oldAssigneeID != "" && oldAssigneeID != user.ID {
				// Status changed, notify assignee
				go h.notificationTriggers.OnTaskStatusChanged(c.Request.Context(), services.TaskStatusChangedInput{
					TaskID:      taskID,
					TaskTitle:   task.Title,
					OldStatus:   oldStatus,
					NewStatus:   newStatus,
					ChangedByID: user.ID,
					ChangedBy:   user.Name,
					AssigneeID:  oldAssigneeID,
				})
			}
		}
	}

	c.JSON(http.StatusOK, TransformTask(task))
}

// ToggleTask toggles the completion status of a task
func (h *DashboardItemHandler) ToggleTask(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "task ID")
		return
	}

	queries := sqlc.New(h.pool)

	// Get existing task for ownership verification and to check old status
	existingTask, err := queries.GetTask(c.Request.Context(), sqlc.GetTaskParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Task")
		return
	}

	task, err := queries.ToggleTaskStatus(c.Request.Context(), pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "toggle task", err)
		return
	}

	// Invalidate dashboard caches when task is toggled
	h.invalidateDashboardCache(c, user.ID)

	// Trigger completion notification if task was just completed
	if h.notificationTriggers != nil {
		oldStatus := strings.ToLower(string(existingTask.Status.Taskstatus))
		newStatus := strings.ToLower(string(task.Status.Taskstatus))

		if newStatus == "completed" && oldStatus != "completed" {
			taskID := uuid.UUID(task.ID.Bytes)
			var projID *uuid.UUID
			if task.ProjectID.Valid {
				pid := uuid.UUID(task.ProjectID.Bytes)
				projID = &pid
			}
			go h.notificationTriggers.OnTaskCompleted(c.Request.Context(), services.TaskCompletedInput{
				TaskID:        taskID,
				TaskTitle:     task.Title,
				CompletedByID: user.ID,
				CompletedBy:   user.Name,
				OwnerID:       task.UserID,
				ProjectID:     projID,
			})
		}
	}

	c.JSON(http.StatusOK, TransformTask(task))
}

// DeleteTask deletes a task
func (h *DashboardItemHandler) DeleteTask(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "task ID")
		return
	}

	queries := sqlc.New(h.pool)
	err = queries.DeleteTask(c.Request.Context(), sqlc.DeleteTaskParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "delete task", err)
		return
	}

	// Invalidate dashboard caches when task is deleted
	h.invalidateDashboardCache(c, user.ID)

	c.JSON(http.StatusOK, gin.H{"message": "Task deleted"})
}
