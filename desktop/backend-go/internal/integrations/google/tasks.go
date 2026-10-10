package google

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/api/option"
	"google.golang.org/api/tasks/v1"
)

// TaskList represents a Google Task list.
type TaskList struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	TaskListID string    `json:"task_list_id"`
	Title      string    `json:"title"`
	Kind       string    `json:"kind,omitempty"`
	Updated    time.Time `json:"updated,omitempty"`
	SyncedAt   time.Time `json:"synced_at"`
}

// GoogleTask represents a Google Task.
type GoogleTask struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	TaskID       string    `json:"task_id"`
	TaskListID   string    `json:"task_list_id"`
	Title        string    `json:"title"`
	Notes        string    `json:"notes,omitempty"`
	Status       string    `json:"status"` // needsAction, completed
	Due          time.Time `json:"due,omitempty"`
	Completed    time.Time `json:"completed,omitempty"`
	Deleted      bool      `json:"deleted"`
	Hidden       bool      `json:"hidden"`
	ParentTaskID string    `json:"parent_task_id,omitempty"`
	Position     string    `json:"position,omitempty"`
	Links        []TaskLink `json:"links,omitempty"`
	Updated      time.Time `json:"updated,omitempty"`
	SyncedAt     time.Time `json:"synced_at"`
}

// TaskLink represents a link associated with a task.
type TaskLink struct {
	Type        string `json:"type"` // email, etc.
	Description string `json:"description,omitempty"`
	Link        string `json:"link"`
}

// TasksService handles Google Tasks operations.
type TasksService struct {
	provider *Provider
}

// NewTasksService creates a new Tasks service.
func NewTasksService(provider *Provider) *TasksService {
	return &TasksService{provider: provider}
}

// GetTasksAPI returns a Google Tasks API service for a user.
func (s *TasksService) GetTasksAPI(ctx context.Context, userID string) (*tasks.Service, error) {
	tokenSource, err := s.provider.GetTokenSource(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get token source: %w", err)
	}

	srv, err := tasks.NewService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		return nil, fmt.Errorf("failed to create tasks service: %w", err)
	}

	return srv, nil
}

// SyncTaskLists syncs task lists from Google Tasks.
func (s *TasksService) SyncTaskLists(ctx context.Context, userID string) (*SyncTaskListsResult, error) {
	slog.Info("Task lists sync starting for user", "user_id", userID)

	srv, err := s.GetTasksAPI(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Tasks API: %w", err)
	}

	result := &SyncTaskListsResult{}
	pageToken := ""

	for {
		req := srv.Tasklists.List().MaxResults(100)
		if pageToken != "" {
			req.PageToken(pageToken)
		}

		resp, err := req.Do()
		if err != nil {
			return nil, fmt.Errorf("failed to list task lists: %w", err)
		}

		result.TotalLists += len(resp.Items)

		for _, list := range resp.Items {
			if err := s.saveTaskList(ctx, userID, list); err != nil {
				slog.Info("Failed to save task list", "id", list.Id, "error", err)
				result.FailedLists++
			} else {
				result.SyncedLists++
			}
		}

		pageToken = resp.NextPageToken
		if pageToken == "" {
			break
		}
	}

	slog.Info("Task lists sync complete for user : synced / lists", "user_id", userID, "synced", result.SyncedLists, "total", result.TotalLists)

	return result, nil
}

// SyncTaskListsResult represents the result of a task lists sync.
type SyncTaskListsResult struct {
	TotalLists  int `json:"total_lists"`
	SyncedLists int `json:"synced_lists"`
	FailedLists int `json:"failed_lists"`
}

// saveTaskList saves a Google Task list to the database.
func (s *TasksService) saveTaskList(ctx context.Context, userID string, list *tasks.TaskList) error {
	var updated *time.Time
	if list.Updated != "" {
		t, _ := time.Parse(time.RFC3339, list.Updated)
		updated = &t
	}

	_, err := s.provider.Pool().Exec(ctx, `
		INSERT INTO google_task_lists (
			user_id, task_list_id, title, kind, updated, synced_at
		) VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (user_id, task_list_id) DO UPDATE SET
			title = EXCLUDED.title,
			kind = EXCLUDED.kind,
			updated = EXCLUDED.updated,
			synced_at = NOW(),
			updated_at = NOW()
	`, userID, list.Id, list.Title, list.Kind, updated)

	return err
}

// SyncTasks syncs tasks from a specific task list.
func (s *TasksService) SyncTasks(ctx context.Context, userID, taskListID string) (*SyncTasksResult, error) {
	slog.Info("Tasks sync starting for user , list", "user_id", userID, "list_id", taskListID)

	srv, err := s.GetTasksAPI(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Tasks API: %w", err)
	}

	result := &SyncTasksResult{}
	pageToken := ""

	for {
		req := srv.Tasks.List(taskListID).
			MaxResults(100).
			ShowCompleted(true).
			ShowHidden(true)

		if pageToken != "" {
			req.PageToken(pageToken)
		}

		resp, err := req.Do()
		if err != nil {
			return nil, fmt.Errorf("failed to list tasks: %w", err)
		}

		result.TotalTasks += len(resp.Items)

		for _, task := range resp.Items {
			if err := s.saveTask(ctx, userID, taskListID, task); err != nil {
				slog.Info("Failed to save task", "id", task.Id, "error", err)
				result.FailedTasks++
			} else {
				result.SyncedTasks++
			}
		}

		pageToken = resp.NextPageToken
		if pageToken == "" {
			break
		}
	}

	slog.Info("Tasks sync complete for user , list : synced / tasks", "user_id", userID, "list_id", taskListID, "synced", result.SyncedTasks, "total", result.TotalTasks)

	return result, nil
}

// SyncTasksResult represents the result of a tasks sync.
type SyncTasksResult struct {
	TotalTasks  int `json:"total_tasks"`
	SyncedTasks int `json:"synced_tasks"`
	FailedTasks int `json:"failed_tasks"`
}

// saveTask saves a Google Task to the database.
func (s *TasksService) saveTask(ctx context.Context, userID, taskListID string, task *tasks.Task) error {
	var due, completed, updated *time.Time
	if task.Due != "" {
		t, _ := time.Parse(time.RFC3339, task.Due)
		due = &t
	}
	if task.Completed != nil && *task.Completed != "" {
		t, _ := time.Parse(time.RFC3339, *task.Completed)
		completed = &t
	}
	if task.Updated != "" {
		t, _ := time.Parse(time.RFC3339, task.Updated)
		updated = &t
	}

	// Extract links
	links := make([]TaskLink, 0)
	for _, l := range task.Links {
		links = append(links, TaskLink{
			Type:        l.Type,
			Description: l.Description,
			Link:        l.Link,
		})
	}

	_, err := s.provider.Pool().Exec(ctx, `
		INSERT INTO google_tasks (
			user_id, task_id, task_list_id, title, notes, status,
			due, completed, deleted, hidden, parent_task_id, position,
			links, updated, synced_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NOW())
		ON CONFLICT (user_id, task_id) DO UPDATE SET
			task_list_id = EXCLUDED.task_list_id,
			title = EXCLUDED.title,
			notes = EXCLUDED.notes,
			status = EXCLUDED.status,
			due = EXCLUDED.due,
			completed = EXCLUDED.completed,
			deleted = EXCLUDED.deleted,
			hidden = EXCLUDED.hidden,
			parent_task_id = EXCLUDED.parent_task_id,
			position = EXCLUDED.position,
			links = EXCLUDED.links,
			updated = EXCLUDED.updated,
			synced_at = NOW(),
			updated_at = NOW()
	`, userID, task.Id, taskListID, task.Title, task.Notes, task.Status,
		due, completed, task.Deleted, task.Hidden, task.Parent, task.Position,
		links, updated)

	return err
}

// SyncAllTasks syncs tasks from all task lists.
func (s *TasksService) SyncAllTasks(ctx context.Context, userID string) (*SyncAllTasksResult, error) {
	// First sync task lists
	listResult, err := s.SyncTaskLists(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to sync task lists: %w", err)
	}

	// Get all task lists
	rows, err := s.provider.Pool().Query(ctx, `
		SELECT task_list_id FROM google_task_lists WHERE user_id = $1
	`, userID)
	if err != nil {
		return nil, err
	}

	var listIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		listIDs = append(listIDs, id)
	}
	rows.Close()

	// Sync tasks from each list
	result := &SyncAllTasksResult{
		Lists: listResult,
	}

	for _, listID := range listIDs {
		taskResult, err := s.SyncTasks(ctx, userID, listID)
		if err != nil {
			slog.Info("Failed to sync tasks for list", "id", listID, "error", err)
			continue
		}
		result.TotalTasks += taskResult.TotalTasks
		result.SyncedTasks += taskResult.SyncedTasks
		result.FailedTasks += taskResult.FailedTasks
	}

	return result, nil
}

// SyncAllTasksResult represents the result of syncing all tasks.
type SyncAllTasksResult struct {
	Lists       *SyncTaskListsResult `json:"lists"`
	TotalTasks  int                  `json:"total_tasks"`
	SyncedTasks int                  `json:"synced_tasks"`
	FailedTasks int                  `json:"failed_tasks"`
}

// GetTaskLists retrieves all task lists for a user.
func (s *TasksService) GetTaskLists(ctx context.Context, userID string) ([]*TaskList, error) {
	rows, err := s.provider.Pool().Query(ctx, `
		SELECT id, user_id, task_list_id, title, kind, updated, synced_at
		FROM google_task_lists
		WHERE user_id = $1
		ORDER BY title
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lists []*TaskList
	for rows.Next() {
		var l TaskList
		var kind pgtype.Text
		var updated pgtype.Timestamptz

		err := rows.Scan(&l.ID, &l.UserID, &l.TaskListID, &l.Title, &kind, &updated, &l.SyncedAt)
		if err != nil {
			return nil, err
		}

		l.Kind = kind.String
		if updated.Valid {
			l.Updated = updated.Time
		}

		lists = append(lists, &l)
	}

	return lists, nil
}

// GetTasks retrieves tasks for a user, optionally filtered by list.
func (s *TasksService) GetTasks(ctx context.Context, userID string, taskListID string, includeCompleted bool, limit, offset int) ([]*GoogleTask, error) {
	query := `
		SELECT id, user_id, task_id, task_list_id, title, notes, status,
			due, completed, deleted, hidden, parent_task_id, position, updated, synced_at
		FROM google_tasks
		WHERE user_id = $1
	`
	args := []interface{}{userID}
	argIndex := 2

	if taskListID != "" {
		query += fmt.Sprintf(" AND task_list_id = $%d", argIndex)
		args = append(args, taskListID)
		argIndex++
	}

	if !includeCompleted {
		query += " AND status != 'completed'"
	}

	query += " AND deleted = false"
	query += fmt.Sprintf(" ORDER BY position, due NULLS LAST LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, offset)

	rows, err := s.provider.Pool().Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasksList []*GoogleTask
	for rows.Next() {
		var t GoogleTask
		var notes, parentTaskID, position pgtype.Text
		var due, completed, updated pgtype.Timestamptz

		err := rows.Scan(
			&t.ID, &t.UserID, &t.TaskID, &t.TaskListID, &t.Title, &notes, &t.Status,
			&due, &completed, &t.Deleted, &t.Hidden, &parentTaskID, &position, &updated, &t.SyncedAt,
		)
		if err != nil {
			return nil, err
		}

		t.Notes = notes.String
		t.ParentTaskID = parentTaskID.String
		t.Position = position.String
		if due.Valid {
			t.Due = due.Time
		}
		if completed.Valid {
			t.Completed = completed.Time
		}
		if updated.Valid {
			t.Updated = updated.Time
		}

		tasksList = append(tasksList, &t)
	}

	return tasksList, nil
}

// GetTask retrieves a single task by ID.
func (s *TasksService) GetTask(ctx context.Context, userID, taskID string) (*GoogleTask, error) {
	var t GoogleTask
	var notes, parentTaskID, position pgtype.Text
	var due, completed, updated pgtype.Timestamptz

	err := s.provider.Pool().QueryRow(ctx, `
		SELECT id, user_id, task_id, task_list_id, title, notes, status,
			due, completed, deleted, hidden, parent_task_id, position, updated, synced_at
		FROM google_tasks
		WHERE user_id = $1 AND task_id = $2
	`, userID, taskID).Scan(
		&t.ID, &t.UserID, &t.TaskID, &t.TaskListID, &t.Title, &notes, &t.Status,
		&due, &completed, &t.Deleted, &t.Hidden, &parentTaskID, &position, &updated, &t.SyncedAt,
	)
	if err != nil {
		return nil, err
	}

	t.Notes = notes.String
	t.ParentTaskID = parentTaskID.String
	t.Position = position.String
	if due.Valid {
		t.Due = due.Time
	}
	if completed.Valid {
		t.Completed = completed.Time
	}
	if updated.Valid {
		t.Updated = updated.Time
	}

	return &t, nil
}

// GetDueTasks retrieves tasks that are due before a specific date.
func (s *TasksService) GetDueTasks(ctx context.Context, userID string, dueBy time.Time) ([]*GoogleTask, error) {
	rows, err := s.provider.Pool().Query(ctx, `
		SELECT id, user_id, task_id, task_list_id, title, notes, status,
			due, completed, deleted, hidden, parent_task_id, position, updated, synced_at
		FROM google_tasks
		WHERE user_id = $1 AND due IS NOT NULL AND due <= $2 AND status != 'completed' AND deleted = false
		ORDER BY due
	`, userID, dueBy)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasksList []*GoogleTask
	for rows.Next() {
		var t GoogleTask
		var notes, parentTaskID, position pgtype.Text
		var due, completed, updated pgtype.Timestamptz

		err := rows.Scan(
			&t.ID, &t.UserID, &t.TaskID, &t.TaskListID, &t.Title, &notes, &t.Status,
			&due, &completed, &t.Deleted, &t.Hidden, &parentTaskID, &position, &updated, &t.SyncedAt,
		)
		if err != nil {
			return nil, err
		}

		t.Notes = notes.String
		t.ParentTaskID = parentTaskID.String
		t.Position = position.String
		if due.Valid {
			t.Due = due.Time
		}
		if completed.Valid {
			t.Completed = completed.Time
		}
		if updated.Valid {
			t.Updated = updated.Time
		}

		tasksList = append(tasksList, &t)
	}

	return tasksList, nil
}

// CreateTask creates a new task in Google Tasks.
func (s *TasksService) CreateTask(ctx context.Context, userID, taskListID string, task *GoogleTask) (*tasks.Task, error) {
	srv, err := s.GetTasksAPI(ctx, userID)
	if err != nil {
		return nil, err
	}

	googleTask := &tasks.Task{
		Title: task.Title,
		Notes: task.Notes,
	}

	if !task.Due.IsZero() {
		googleTask.Due = task.Due.Format(time.RFC3339)
	}

	created, err := srv.Tasks.Insert(taskListID, googleTask).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	// Save to database
	if err := s.saveTask(ctx, userID, taskListID, created); err != nil {
		slog.Info("Failed to save created task to database", "error", err)
	}

	return created, nil
}

// CompleteTask marks a task as completed.
func (s *TasksService) CompleteTask(ctx context.Context, userID, taskListID, taskID string) error {
	srv, err := s.GetTasksAPI(ctx, userID)
	if err != nil {
		return err
	}

	// Get current task
	task, err := srv.Tasks.Get(taskListID, taskID).Do()
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Update status
	task.Status = "completed"
	completedTime := time.Now().Format(time.RFC3339)
	task.Completed = &completedTime

	updated, err := srv.Tasks.Update(taskListID, taskID, task).Do()
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	// Save to database
	return s.saveTask(ctx, userID, taskListID, updated)
}

// DeleteTask deletes a task from Google Tasks.
func (s *TasksService) DeleteTask(ctx context.Context, userID, taskListID, taskID string) error {
	srv, err := s.GetTasksAPI(ctx, userID)
	if err != nil {
		return err
	}

	if err := srv.Tasks.Delete(taskListID, taskID).Do(); err != nil {
		return fmt.Errorf("failed to delete task from Google: %w", err)
	}

	// Delete from database
	_, err = s.provider.Pool().Exec(ctx, `
		DELETE FROM google_tasks WHERE user_id = $1 AND task_id = $2
	`, userID, taskID)

	return err
}

// CreateTaskList creates a new task list in Google Tasks.
func (s *TasksService) CreateTaskList(ctx context.Context, userID, title string) (*tasks.TaskList, error) {
	srv, err := s.GetTasksAPI(ctx, userID)
	if err != nil {
		return nil, err
	}

	list := &tasks.TaskList{
		Title: title,
	}

	created, err := srv.Tasklists.Insert(list).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to create task list: %w", err)
	}

	// Save to database
	if err := s.saveTaskList(ctx, userID, created); err != nil {
		slog.Info("Failed to save created task list to database", "error", err)
	}

	return created, nil
}

// IsConnected checks if Google Tasks is connected for a user.
func (s *TasksService) IsConnected(ctx context.Context, userID string) bool {
	var scopes []string
	err := s.provider.Pool().QueryRow(ctx, `
		SELECT scopes FROM google_oauth_tokens WHERE user_id = $1
	`, userID).Scan(&scopes)
	if err != nil {
		return false
	}

	for _, scope := range scopes {
		if containsTasksScope(scope) {
			return true
		}
	}
	return false
}

func containsTasksScope(scope string) bool {
	tasksScopes := []string{
		"https://www.googleapis.com/auth/tasks",
		"tasks",
		"tasks.readonly",
	}
	for _, s := range tasksScopes {
		if scope == s || scope == "https://www.googleapis.com/auth/"+s {
			return true
		}
	}
	return false
}
