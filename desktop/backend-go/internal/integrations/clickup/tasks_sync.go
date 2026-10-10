package clickup

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
)

// SyncTasks fetches tasks from a ClickUp list and persists them to the database.
func (p *Provider) SyncTasks(ctx context.Context, userID, listID string) (int, error) {
	tasks, err := p.GetTasks(ctx, userID, listID)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch tasks: %w", err)
	}

	queries := sqlc.New(p.pool)
	synced := 0

	for _, task := range tasks {
		assigneesJSON, _ := json.Marshal(task.Assignees)
		creatorJSON, _ := json.Marshal(task.Creator)
		tagsJSON, _ := json.Marshal(task.Tags)

		parseDateMillis := func(dateStr string) pgtype.Timestamptz {
			var ts pgtype.Timestamptz
			if dateStr == "" {
				return ts
			}
			if millis, err := strconv.ParseInt(dateStr, 10, 64); err == nil {
				ts.Time = time.Unix(0, millis*1000000)
				ts.Valid = true
			}
			return ts
		}

		dueDate := parseDateMillis(task.DueDate)
		startDate := parseDateMillis(task.StartDate)
		dateCreated := parseDateMillis(task.DateCreated)
		dateUpdated := parseDateMillis(task.DateUpdated)
		dateClosed := parseDateMillis(task.DateClosed)

		strPtr := func(s string) *string {
			if s == "" {
				return nil
			}
			return &s
		}

		int64Ptr := func(i int64) *int64 {
			if i == 0 {
				return nil
			}
			return &i
		}

		_, err := queries.UpsertClickUpTask(ctx, sqlc.UpsertClickUpTaskParams{
			UserID:        userID,
			TaskID:        task.ID,
			CustomID:      strPtr(task.CustomID),
			ListID:        task.List.ID,
			FolderID:      strPtr(task.Folder.ID),
			SpaceID:       task.Space.ID,
			Name:          task.Name,
			Description:   strPtr(task.Description),
			Status:        strPtr(task.Status.Status),
			StatusColor:   strPtr(task.Status.Color),
			Priority:      strPtr(task.Priority.Priority),
			PriorityColor: strPtr(task.Priority.Color),
			DueDate:       dueDate,
			StartDate:     startDate,
			DateCreated:   dateCreated,
			DateUpdated:   dateUpdated,
			DateClosed:    dateClosed,
			TimeSpent:     int64Ptr(task.TimeSpent),
			TimeEstimate:  nil,
			ParentTaskID:  strPtr(task.Parent),
			Assignees:     assigneesJSON,
			Creator:       creatorJSON,
			Tags:          tagsJSON,
			Url:           strPtr(task.URL),
		})
		if err != nil {
			slog.Warn("Failed to upsert task", "task_id", task.ID, "error", err)
			continue
		}
		synced++
	}

	return synced, nil
}
