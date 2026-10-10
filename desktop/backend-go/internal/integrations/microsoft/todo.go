package microsoft

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// ToDoList represents a Microsoft To Do task list.
type ToDoList struct {
	ID              string    `json:"id"`
	UserID          string    `json:"user_id"`
	ListID          string    `json:"list_id"`
	DisplayName     string    `json:"display_name"`
	IsOwner         bool      `json:"is_owner"`
	IsShared        bool      `json:"is_shared"`
	WellknownListName string  `json:"wellknown_list_name,omitempty"`
	SyncedAt        time.Time `json:"synced_at"`
}

// ToDoTask represents a Microsoft To Do task.
type ToDoTask struct {
	ID                string    `json:"id"`
	UserID            string    `json:"user_id"`
	TaskID            string    `json:"task_id"`
	ListID            string    `json:"list_id"`
	Title             string    `json:"title"`
	BodyContent       string    `json:"body_content,omitempty"`
	BodyContentType   string    `json:"body_content_type,omitempty"`
	Importance        string    `json:"importance,omitempty"`
	Status            string    `json:"status"`
	DueDateTime       time.Time `json:"due_datetime,omitempty"`
	DueTimeZone       string    `json:"due_timezone,omitempty"`
	StartDateTime     time.Time `json:"start_datetime,omitempty"`
	CompletedDateTime time.Time `json:"completed_datetime,omitempty"`
	IsReminderOn      bool      `json:"is_reminder_on"`
	ReminderDateTime  time.Time `json:"reminder_datetime,omitempty"`
	Categories        []string  `json:"categories,omitempty"`
	CreatedDateTime   time.Time `json:"created_datetime,omitempty"`
	LastModifiedDateTime time.Time `json:"last_modified_datetime,omitempty"`
	SyncedAt          time.Time `json:"synced_at"`
}

// ToDoService handles Microsoft To Do operations.
type ToDoService struct {
	provider *Provider
}

// NewToDoService creates a new To Do service.
func NewToDoService(provider *Provider) *ToDoService {
	return &ToDoService{provider: provider}
}

// SyncLists syncs task lists from Microsoft To Do.
func (s *ToDoService) SyncLists(ctx context.Context, userID string) (*SyncListsResult, error) {
	slog.Info("To Do lists sync starting for user", "user_id", userID)

	client, err := s.provider.GetHTTPClient(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get HTTP client: %w", err)
	}

	result := &SyncListsResult{}

	resp, err := client.Get(GraphAPIBase + "/me/todo/lists")
	if err != nil {
		return nil, fmt.Errorf("failed to get lists: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %s", resp.Status)
	}

	var listResp struct {
		Value []graphToDoList `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	result.TotalLists = len(listResp.Value)

	for _, list := range listResp.Value {
		if err := s.saveList(ctx, userID, &list); err != nil {
			slog.Info("Failed to save list", "id", list.ID, "error", err)
			result.FailedLists++
		} else {
			result.SyncedLists++
		}
	}

	slog.Info("To Do lists sync complete for user : synced / lists", "user_id", userID, "synced", result.SyncedLists, "total", result.TotalLists)

	return result, nil
}

// SyncListsResult represents the result of a lists sync.
type SyncListsResult struct {
	TotalLists  int `json:"total_lists"`
	SyncedLists int `json:"synced_lists"`
	FailedLists int `json:"failed_lists"`
}

type graphToDoList struct {
	ID                string `json:"id"`
	DisplayName       string `json:"displayName"`
	IsOwner           bool   `json:"isOwner"`
	IsShared          bool   `json:"isShared"`
	WellknownListName string `json:"wellknownListName"`
}

func (s *ToDoService) saveList(ctx context.Context, userID string, list *graphToDoList) error {
	_, err := s.provider.Pool().Exec(ctx, `
		INSERT INTO microsoft_todo_lists (
			user_id, list_id, display_name, is_owner, is_shared, wellknown_list_name, synced_at
		) VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (user_id, list_id) DO UPDATE SET
			display_name = EXCLUDED.display_name,
			is_owner = EXCLUDED.is_owner,
			is_shared = EXCLUDED.is_shared,
			wellknown_list_name = EXCLUDED.wellknown_list_name,
			synced_at = NOW(),
			updated_at = NOW()
	`, userID, list.ID, list.DisplayName, list.IsOwner, list.IsShared, list.WellknownListName)

	return err
}

// SyncTasks syncs tasks from a specific list.
func (s *ToDoService) SyncTasks(ctx context.Context, userID, listID string) (*SyncTasksResult, error) {
	slog.Info("To Do tasks sync starting for user , list", "user_id", userID, "list_id", listID)

	client, err := s.provider.GetHTTPClient(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get HTTP client: %w", err)
	}

	result := &SyncTasksResult{}

	apiURL := fmt.Sprintf("%s/me/todo/lists/%s/tasks", GraphAPIBase, listID)
	resp, err := client.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %s", resp.Status)
	}

	var taskResp struct {
		Value []graphToDoTask `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&taskResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	result.TotalTasks = len(taskResp.Value)

	for _, task := range taskResp.Value {
		if err := s.saveTask(ctx, userID, listID, &task); err != nil {
			slog.Info("Failed to save task", "id", task.ID, "error", err)
			result.FailedTasks++
		} else {
			result.SyncedTasks++
		}
	}

	slog.Info("To Do tasks sync complete for user , list : synced / tasks", "user_id", userID, "list_id", listID, "synced", result.SyncedTasks, "total", result.TotalTasks)

	return result, nil
}

// SyncTasksResult represents the result of a tasks sync.
type SyncTasksResult struct {
	TotalTasks  int `json:"total_tasks"`
	SyncedTasks int `json:"synced_tasks"`
	FailedTasks int `json:"failed_tasks"`
}

type graphToDoTask struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Body       *struct {
		Content     string `json:"content"`
		ContentType string `json:"contentType"`
	} `json:"body"`
	Importance string `json:"importance"`
	Status     string `json:"status"`
	DueDateTime *struct {
		DateTime string `json:"dateTime"`
		TimeZone string `json:"timeZone"`
	} `json:"dueDateTime"`
	StartDateTime *struct {
		DateTime string `json:"dateTime"`
		TimeZone string `json:"timeZone"`
	} `json:"startDateTime"`
	CompletedDateTime *struct {
		DateTime string `json:"dateTime"`
		TimeZone string `json:"timeZone"`
	} `json:"completedDateTime"`
	IsReminderOn     bool     `json:"isReminderOn"`
	ReminderDateTime *struct {
		DateTime string `json:"dateTime"`
		TimeZone string `json:"timeZone"`
	} `json:"reminderDateTime"`
	Categories           []string `json:"categories"`
	CreatedDateTime      string   `json:"createdDateTime"`
	LastModifiedDateTime string   `json:"lastModifiedDateTime"`
}

func (s *ToDoService) saveTask(ctx context.Context, userID, listID string, task *graphToDoTask) error {
	// Extract body
	var bodyContent, bodyContentType string
	if task.Body != nil {
		bodyContent = task.Body.Content
		bodyContentType = task.Body.ContentType
	}

	// Parse dates
	var dueDateTime, startDateTime, completedDateTime, reminderDateTime *time.Time
	var dueTimeZone string
	if task.DueDateTime != nil && task.DueDateTime.DateTime != "" {
		t, _ := time.Parse("2006-01-02T15:04:05.0000000", task.DueDateTime.DateTime)
		dueDateTime = &t
		dueTimeZone = task.DueDateTime.TimeZone
	}
	if task.StartDateTime != nil && task.StartDateTime.DateTime != "" {
		t, _ := time.Parse("2006-01-02T15:04:05.0000000", task.StartDateTime.DateTime)
		startDateTime = &t
	}
	if task.CompletedDateTime != nil && task.CompletedDateTime.DateTime != "" {
		t, _ := time.Parse("2006-01-02T15:04:05.0000000", task.CompletedDateTime.DateTime)
		completedDateTime = &t
	}
	if task.ReminderDateTime != nil && task.ReminderDateTime.DateTime != "" {
		t, _ := time.Parse("2006-01-02T15:04:05.0000000", task.ReminderDateTime.DateTime)
		reminderDateTime = &t
	}

	var createdDateTime, lastModifiedDateTime *time.Time
	if task.CreatedDateTime != "" {
		t, _ := time.Parse(time.RFC3339, task.CreatedDateTime)
		createdDateTime = &t
	}
	if task.LastModifiedDateTime != "" {
		t, _ := time.Parse(time.RFC3339, task.LastModifiedDateTime)
		lastModifiedDateTime = &t
	}

	_, err := s.provider.Pool().Exec(ctx, `
		INSERT INTO microsoft_todo_tasks (
			user_id, task_id, list_id, title, body_content, body_content_type,
			importance, status, due_datetime, due_timezone, start_datetime,
			completed_datetime, is_reminder_on, reminder_datetime, categories,
			created_datetime, last_modified_datetime, synced_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, NOW())
		ON CONFLICT (user_id, task_id) DO UPDATE SET
			list_id = EXCLUDED.list_id,
			title = EXCLUDED.title,
			body_content = EXCLUDED.body_content,
			body_content_type = EXCLUDED.body_content_type,
			importance = EXCLUDED.importance,
			status = EXCLUDED.status,
			due_datetime = EXCLUDED.due_datetime,
			due_timezone = EXCLUDED.due_timezone,
			start_datetime = EXCLUDED.start_datetime,
			completed_datetime = EXCLUDED.completed_datetime,
			is_reminder_on = EXCLUDED.is_reminder_on,
			reminder_datetime = EXCLUDED.reminder_datetime,
			categories = EXCLUDED.categories,
			last_modified_datetime = EXCLUDED.last_modified_datetime,
			synced_at = NOW(),
			updated_at = NOW()
	`, userID, task.ID, listID, task.Title, bodyContent, bodyContentType,
		task.Importance, task.Status, dueDateTime, dueTimeZone, startDateTime,
		completedDateTime, task.IsReminderOn, reminderDateTime, task.Categories,
		createdDateTime, lastModifiedDateTime)

	return err
}

// SyncAllTasks syncs all tasks from all lists.
func (s *ToDoService) SyncAllTasks(ctx context.Context, userID string) (*SyncAllTasksResult, error) {
	// First sync lists
	listResult, err := s.SyncLists(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to sync lists: %w", err)
	}

	// Get all lists
	rows, err := s.provider.Pool().Query(ctx, `
		SELECT list_id FROM microsoft_todo_lists WHERE user_id = $1
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
	Lists       *SyncListsResult `json:"lists"`
	TotalTasks  int              `json:"total_tasks"`
	SyncedTasks int              `json:"synced_tasks"`
	FailedTasks int              `json:"failed_tasks"`
}

// GetLists retrieves all task lists for a user.
func (s *ToDoService) GetLists(ctx context.Context, userID string) ([]*ToDoList, error) {
	rows, err := s.provider.Pool().Query(ctx, `
		SELECT id, user_id, list_id, display_name, is_owner, is_shared, wellknown_list_name, synced_at
		FROM microsoft_todo_lists
		WHERE user_id = $1
		ORDER BY display_name
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lists []*ToDoList
	for rows.Next() {
		var l ToDoList
		var wellknownName *string

		err := rows.Scan(&l.ID, &l.UserID, &l.ListID, &l.DisplayName, &l.IsOwner, &l.IsShared, &wellknownName, &l.SyncedAt)
		if err != nil {
			return nil, err
		}

		if wellknownName != nil {
			l.WellknownListName = *wellknownName
		}

		lists = append(lists, &l)
	}

	return lists, nil
}

// GetTasks retrieves tasks for a user.
func (s *ToDoService) GetTasks(ctx context.Context, userID, listID string, includeCompleted bool, limit, offset int) ([]*ToDoTask, error) {
	query := `
		SELECT id, user_id, task_id, list_id, title, body_content, body_content_type,
			importance, status, due_datetime, due_timezone, start_datetime,
			completed_datetime, is_reminder_on, reminder_datetime, categories,
			created_datetime, last_modified_datetime, synced_at
		FROM microsoft_todo_tasks
		WHERE user_id = $1
	`
	args := []interface{}{userID}
	argIndex := 2

	if listID != "" {
		query += fmt.Sprintf(" AND list_id = $%d", argIndex)
		args = append(args, listID)
		argIndex++
	}

	if !includeCompleted {
		query += " AND status != 'completed'"
	}

	query += fmt.Sprintf(" ORDER BY due_datetime NULLS LAST LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, offset)

	rows, err := s.provider.Pool().Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*ToDoTask
	for rows.Next() {
		var t ToDoTask
		var bodyContent, bodyContentType, importance, dueTZ *string
		var dueDateTime, startDateTime, completedDateTime, reminderDateTime *time.Time
		var createdDateTime, lastModifiedDateTime *time.Time
		var categories []string

		err := rows.Scan(
			&t.ID, &t.UserID, &t.TaskID, &t.ListID, &t.Title, &bodyContent, &bodyContentType,
			&importance, &t.Status, &dueDateTime, &dueTZ, &startDateTime,
			&completedDateTime, &t.IsReminderOn, &reminderDateTime, &categories,
			&createdDateTime, &lastModifiedDateTime, &t.SyncedAt,
		)
		if err != nil {
			return nil, err
		}

		if bodyContent != nil {
			t.BodyContent = *bodyContent
		}
		if bodyContentType != nil {
			t.BodyContentType = *bodyContentType
		}
		if importance != nil {
			t.Importance = *importance
		}
		if dueTZ != nil {
			t.DueTimeZone = *dueTZ
		}
		if dueDateTime != nil {
			t.DueDateTime = *dueDateTime
		}
		if startDateTime != nil {
			t.StartDateTime = *startDateTime
		}
		if completedDateTime != nil {
			t.CompletedDateTime = *completedDateTime
		}
		if reminderDateTime != nil {
			t.ReminderDateTime = *reminderDateTime
		}
		t.Categories = categories
		if createdDateTime != nil {
			t.CreatedDateTime = *createdDateTime
		}
		if lastModifiedDateTime != nil {
			t.LastModifiedDateTime = *lastModifiedDateTime
		}

		tasks = append(tasks, &t)
	}

	return tasks, nil
}

// CreateTask creates a new task in Microsoft To Do.
func (s *ToDoService) CreateTask(ctx context.Context, userID, listID string, task *ToDoTask) (*ToDoTask, error) {
	client, err := s.provider.GetHTTPClient(ctx, userID)
	if err != nil {
		return nil, err
	}

	taskData := map[string]interface{}{
		"title": task.Title,
	}

	if task.BodyContent != "" {
		taskData["body"] = map[string]string{
			"content":     task.BodyContent,
			"contentType": "text",
		}
	}

	if !task.DueDateTime.IsZero() {
		taskData["dueDateTime"] = map[string]string{
			"dateTime": task.DueDateTime.Format("2006-01-02T15:04:05"),
			"timeZone": task.DueTimeZone,
		}
	}

	if task.Importance != "" {
		taskData["importance"] = task.Importance
	}

	jsonBody, _ := json.Marshal(taskData)

	apiURL := fmt.Sprintf("%s/me/todo/lists/%s/tasks", GraphAPIBase, listID)
	resp, err := client.Post(apiURL, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("failed to create task: %s", resp.Status)
	}

	var created graphToDoTask
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Save to database
	if err := s.saveTask(ctx, userID, listID, &created); err != nil {
		slog.Info("Failed to save created task to database", "error", err)
	}

	task.TaskID = created.ID
	return task, nil
}

// CompleteTask marks a task as completed.
func (s *ToDoService) CompleteTask(ctx context.Context, userID, listID, taskID string) error {
	client, err := s.provider.GetHTTPClient(ctx, userID)
	if err != nil {
		return err
	}

	taskData := map[string]interface{}{
		"status": "completed",
	}

	jsonBody, _ := json.Marshal(taskData)

	apiURL := fmt.Sprintf("%s/me/todo/lists/%s/tasks/%s", GraphAPIBase, listID, taskID)
	req, _ := http.NewRequest("PATCH", apiURL, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to complete task: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to complete task: %s", resp.Status)
	}

	// Update in database
	_, err = s.provider.Pool().Exec(ctx, `
		UPDATE microsoft_todo_tasks SET status = 'completed', completed_datetime = NOW(), updated_at = NOW()
		WHERE user_id = $1 AND task_id = $2
	`, userID, taskID)

	return err
}

// IsConnected checks if Microsoft To Do is connected for a user.
func (s *ToDoService) IsConnected(ctx context.Context, userID string) bool {
	var scopes []string
	err := s.provider.Pool().QueryRow(ctx, `
		SELECT scopes FROM microsoft_oauth_tokens WHERE user_id = $1
	`, userID).Scan(&scopes)
	if err != nil {
		return false
	}

	for _, scope := range scopes {
		if scope == "Tasks.Read" || scope == "Tasks.ReadWrite" {
			return true
		}
	}
	return false
}
