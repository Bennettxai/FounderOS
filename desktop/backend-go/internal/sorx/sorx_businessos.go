package sorx

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// ============================================================================
// BusinessOS Context Gathering
// ============================================================================

func businessOSGatherContext(ctx context.Context, ac ActionContext) (interface{}, error) {
	sources, _ := ac.Params["sources"].([]interface{})

	pool, err := getPoolFromContext(ac)
	if err != nil {
		slog.Warn("gather_context: pool not available, returning empty context", "error", err)
		return map[string]interface{}{
			"sources":       []string{},
			"documents":     []interface{}{},
			"conversations": []interface{}{},
			"artifacts":     []interface{}{},
			"clients":       []interface{}{},
			"projects":      []interface{}{},
			"context_built": false,
			"error":         err.Error(),
		}, nil
	}

	sourceList := make([]string, 0)
	for _, s := range sources {
		if str, ok := s.(string); ok {
			sourceList = append(sourceList, str)
		}
	}

	userID := ac.Execution.UserID
	contextData := map[string]interface{}{
		"sources":       sourceList,
		"documents":     []interface{}{},
		"conversations": []interface{}{},
		"artifacts":     []interface{}{},
		"clients":       []interface{}{},
		"projects":      []interface{}{},
		"context_built": true,
	}

	// Gather documents if requested
	for _, source := range sourceList {
		switch source {
		case "documents":
			rows, err := pool.Query(ctx, `
				SELECT id, title, content, created_at
				FROM documents
				WHERE user_id = $1
				ORDER BY created_at DESC
				LIMIT 10
			`, userID)
			if err == nil {
				var docs []interface{}
				for rows.Next() {
					var id, title, content string
					var createdAt time.Time
					if err := rows.Scan(&id, &title, &content, &createdAt); err == nil {
						docs = append(docs, map[string]interface{}{
							"id":         id,
							"title":      title,
							"content":    content,
							"created_at": createdAt,
						})
					}
				}
				rows.Close()
				contextData["documents"] = docs
			}

		case "conversations":
			rows, err := pool.Query(ctx, `
				SELECT id, title, created_at
				FROM conversations
				WHERE user_id = $1
				ORDER BY created_at DESC
				LIMIT 10
			`, userID)
			if err == nil {
				var convs []interface{}
				for rows.Next() {
					var id, title string
					var createdAt time.Time
					if err := rows.Scan(&id, &title, &createdAt); err == nil {
						convs = append(convs, map[string]interface{}{
							"id":         id,
							"title":      title,
							"created_at": createdAt,
						})
					}
				}
				rows.Close()
				contextData["conversations"] = convs
			}

		case "clients":
			rows, err := pool.Query(ctx, `
				SELECT id, name, email, created_at
				FROM clients
				WHERE user_id = $1
				ORDER BY created_at DESC
				LIMIT 10
			`, userID)
			if err == nil {
				var clients []interface{}
				for rows.Next() {
					var id, name, email string
					var createdAt time.Time
					if err := rows.Scan(&id, &name, &email, &createdAt); err == nil {
						clients = append(clients, map[string]interface{}{
							"id":         id,
							"name":       name,
							"email":      email,
							"created_at": createdAt,
						})
					}
				}
				rows.Close()
				contextData["clients"] = clients
			}
		}
	}

	return contextData, nil
}

// ============================================================================
// BusinessOS Platform Actions
// ============================================================================

func businessOSCreateTasks(ctx context.Context, ac ActionContext) (interface{}, error) {
	from, _ := ac.Params["from"].(string)

	slog.Info("businessOSCreateTasks", "user_id", ac.Execution.UserID, "from", from)

	pool, err := getPoolFromContext(ac)
	if err != nil {
		return nil, err
	}

	// Get items from previous step results
	items := ac.Execution.StepResults[from]
	if items == nil {
		return map[string]interface{}{
			"created": 0,
			"source":  from,
			"error":   "source data not found",
		}, nil
	}

	created := 0
	if itemList, ok := items.([]interface{}); ok {
		for _, item := range itemList {
			if itemMap, ok := item.(map[string]interface{}); ok {
				title, _ := itemMap["title"].(string)
				if title == "" {
					continue
				}

				_, err := pool.Exec(ctx, `
					INSERT INTO tasks (user_id, title, status, created_at)
					VALUES ($1, $2, 'pending', NOW())
				`, ac.Execution.UserID, title)

				if err == nil {
					created++
				} else {
					slog.Warn("failed to create task", "title", title, "error", err)
				}
			}
		}
	}

	slog.Info("businessOSCreateTasks success", "created", created)
	return map[string]interface{}{
		"created": created,
		"source":  from,
		"items":   items,
	}, nil
}

func businessOSUpsertClients(ctx context.Context, ac ActionContext) (interface{}, error) {
	slog.Info("businessOSUpsertClients", "user_id", ac.Execution.UserID)

	pool, err := getPoolFromContext(ac)
	if err != nil {
		return nil, err
	}

	from, _ := ac.Params["from"].(string)
	items := ac.Execution.StepResults[from]

	upserted := 0
	if itemList, ok := items.([]interface{}); ok {
		for _, item := range itemList {
			if itemMap, ok := item.(map[string]interface{}); ok {
				name, _ := itemMap["name"].(string)
				email, _ := itemMap["email"].(string)

				if name != "" && email != "" {
					_, err := pool.Exec(ctx, `
						INSERT INTO clients (user_id, name, email, created_at)
						VALUES ($1, $2, $3, NOW())
						ON CONFLICT (user_id, email) DO UPDATE SET
							name = EXCLUDED.name,
							updated_at = NOW()
					`, ac.Execution.UserID, name, email)

					if err == nil {
						upserted++
					}
				}
			}
		}
	}

	slog.Info("businessOSUpsertClients success", "upserted", upserted)
	return map[string]interface{}{
		"upserted": upserted,
	}, nil
}

func businessOSCreateDailyLog(ctx context.Context, ac ActionContext) (interface{}, error) {
	slog.Info("businessOSCreateDailyLog", "user_id", ac.Execution.UserID)

	pool, err := getPoolFromContext(ac)
	if err != nil {
		return nil, err
	}

	content, _ := ac.Params["content"].(string)
	logType, _ := ac.Params["type"].(string)

	if logType == "" {
		logType = "general"
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO daily_logs (user_id, content, log_type, created_at)
		VALUES ($1, $2, $3, NOW())
	`, ac.Execution.UserID, content, logType)

	if err != nil {
		return nil, fmt.Errorf("failed to create daily log: %w", err)
	}

	return map[string]interface{}{
		"created": 1,
		"type":    logType,
	}, nil
}

func businessOSImportTasks(ctx context.Context, ac ActionContext) (interface{}, error) {
	// Get decision result if any
	decisionResult := ac.Execution.Context["decision_result"]

	slog.Info("businessOSImportTasks", "user_id", ac.Execution.UserID, "decision", decisionResult)

	pool, err := getPoolFromContext(ac)
	if err != nil {
		return nil, err
	}

	from, _ := ac.Params["from"].(string)
	items := ac.Execution.StepResults[from]

	imported := 0
	if itemList, ok := items.([]interface{}); ok {
		for _, item := range itemList {
			if itemMap, ok := item.(map[string]interface{}); ok {
				title, _ := itemMap["title"].(string)
				externalID, _ := itemMap["id"].(string)

				if title != "" {
					_, err := pool.Exec(ctx, `
						INSERT INTO tasks (user_id, title, external_id, status, created_at)
						VALUES ($1, $2, $3, 'pending', NOW())
						ON CONFLICT (user_id, external_id) DO UPDATE SET
							title = EXCLUDED.title,
							updated_at = NOW()
					`, ac.Execution.UserID, title, externalID)

					if err == nil {
						imported++
					}
				}
			}
		}
	}

	slog.Info("businessOSImportTasks success", "imported", imported)
	return map[string]interface{}{
		"imported": imported,
		"decision": decisionResult,
	}, nil
}

func businessOSCreateNodes(ctx context.Context, ac ActionContext) (interface{}, error) {
	nodeType, _ := ac.Params["type"].(string)
	source, _ := ac.Params["source"].(string)

	slog.Info("businessOSCreateNodes", "user_id", ac.Execution.UserID, "type", nodeType, "source", source)

	pool, err := getPoolFromContext(ac)
	if err != nil {
		return nil, err
	}

	items := ac.Execution.StepResults[source]
	created := 0

	if itemList, ok := items.([]interface{}); ok {
		for _, item := range itemList {
			if itemMap, ok := item.(map[string]interface{}); ok {
				title, _ := itemMap["title"].(string)
				content, _ := itemMap["content"].(string)

				if title != "" {
					_, err := pool.Exec(ctx, `
						INSERT INTO knowledge_nodes (user_id, type, title, content, created_at)
						VALUES ($1, $2, $3, $4, NOW())
					`, ac.Execution.UserID, nodeType, title, content)

					if err == nil {
						created++
					}
				}
			}
		}
	}

	slog.Info("businessOSCreateNodes success", "created", created)
	return map[string]interface{}{
		"created": created,
		"type":    nodeType,
		"source":  source,
	}, nil
}

func businessOSListPendingTasks(ctx context.Context, ac ActionContext) (interface{}, error) {
	slog.Info("businessOSListPendingTasks", "user_id", ac.Execution.UserID)

	pool, err := getPoolFromContext(ac)
	if err != nil {
		return nil, err
	}

	rows, err := pool.Query(ctx, `
		SELECT id, title, status, created_at
		FROM tasks
		WHERE user_id = $1 AND status = 'pending'
		ORDER BY created_at DESC
		LIMIT 50
	`, ac.Execution.UserID)

	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}
	defer rows.Close()

	tasks := make([]map[string]interface{}, 0)
	for rows.Next() {
		var id, title, status string
		var createdAt time.Time
		if err := rows.Scan(&id, &title, &status, &createdAt); err == nil {
			tasks = append(tasks, map[string]interface{}{
				"id":         id,
				"title":      title,
				"status":     status,
				"created_at": createdAt,
			})
		}
	}

	slog.Info("businessOSListPendingTasks success", "count", len(tasks))
	return map[string]interface{}{
		"tasks": tasks,
		"count": len(tasks),
	}, nil
}

func businessOSGetClientSummary(ctx context.Context, ac ActionContext) (interface{}, error) {
	clientID, _ := ac.Params["client_id"].(string)

	slog.Info("businessOSGetClientSummary", "user_id", ac.Execution.UserID, "client_id", clientID)

	pool, err := getPoolFromContext(ac)
	if err != nil {
		return nil, err
	}

	var name, email string
	var createdAt time.Time
	err = pool.QueryRow(ctx, `
		SELECT name, email, created_at
		FROM clients
		WHERE id = $1 AND user_id = $2
	`, clientID, ac.Execution.UserID).Scan(&name, &email, &createdAt)

	if err != nil {
		return nil, fmt.Errorf("client not found: %w", err)
	}

	// Get related data
	var projectCount, taskCount int
	pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM projects WHERE client_id = $1
	`, clientID).Scan(&projectCount)
	pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM tasks WHERE client_id = $1
	`, clientID).Scan(&taskCount)

	return map[string]interface{}{
		"client_id":     clientID,
		"name":          name,
		"email":         email,
		"created_at":    createdAt,
		"project_count": projectCount,
		"task_count":    taskCount,
		"summary":       fmt.Sprintf("%s (%s) - %d projects, %d tasks", name, email, projectCount, taskCount),
	}, nil
}

func businessOSGetPipelineSummary(ctx context.Context, ac ActionContext) (interface{}, error) {
	slog.Info("businessOSGetPipelineSummary", "user_id", ac.Execution.UserID)

	pool, err := getPoolFromContext(ac)
	if err != nil {
		return nil, err
	}

	rows, err := pool.Query(ctx, `
		SELECT stage, COUNT(*), COALESCE(SUM(value), 0)
		FROM deals
		WHERE user_id = $1
		GROUP BY stage
	`, ac.Execution.UserID)

	if err != nil {
		return nil, fmt.Errorf("failed to get pipeline summary: %w", err)
	}
	defer rows.Close()

	stages := make(map[string]interface{})
	var totalValue float64
	var totalDeals int

	for rows.Next() {
		var stage string
		var count int
		var value float64
		if err := rows.Scan(&stage, &count, &value); err == nil {
			stages[stage] = map[string]interface{}{
				"count": count,
				"value": value,
			}
			totalDeals += count
			totalValue += value
		}
	}

	return map[string]interface{}{
		"pipeline":    stages,
		"total_value": totalValue,
		"total_deals": totalDeals,
		"stages":      stages,
	}, nil
}

func businessOSGetMeetingContext(ctx context.Context, ac ActionContext) (interface{}, error) {
	meetingID, _ := ac.Params["meeting_id"].(string)

	slog.Info("businessOSGetMeetingContext", "user_id", ac.Execution.UserID, "meeting_id", meetingID)

	pool, err := getPoolFromContext(ac)
	if err != nil {
		return nil, err
	}

	// Get meeting details
	var title, description string
	var startTime time.Time
	err = pool.QueryRow(ctx, `
		SELECT title, description, start_time
		FROM calendar_events
		WHERE id = $1 AND user_id = $2
	`, meetingID, ac.Execution.UserID).Scan(&title, &description, &startTime)

	if err != nil {
		return nil, fmt.Errorf("meeting not found: %w", err)
	}

	// Get attendees
	attendeesRows, _ := pool.Query(ctx, `
		SELECT email, name
		FROM meeting_attendees
		WHERE meeting_id = $1
	`, meetingID)
	defer attendeesRows.Close()

	attendees := make([]map[string]interface{}, 0)
	for attendeesRows.Next() {
		var email, name string
		if err := attendeesRows.Scan(&email, &name); err == nil {
			attendees = append(attendees, map[string]interface{}{
				"email": email,
				"name":  name,
			})
		}
	}

	// Get previous meeting notes
	notesRows, _ := pool.Query(ctx, `
		SELECT content, created_at
		FROM meeting_notes
		WHERE meeting_id = $1
		ORDER BY created_at DESC
		LIMIT 5
	`, meetingID)
	defer notesRows.Close()

	notes := make([]map[string]interface{}, 0)
	for notesRows.Next() {
		var content string
		var createdAt time.Time
		if err := notesRows.Scan(&content, &createdAt); err == nil {
			notes = append(notes, map[string]interface{}{
				"content":    content,
				"created_at": createdAt,
			})
		}
	}

	return map[string]interface{}{
		"meeting_id":      meetingID,
		"title":           title,
		"description":     description,
		"start_time":      startTime,
		"attendees":       attendees,
		"previous_notes":  notes,
		"related_clients": []interface{}{}, // TODO: Link to clients
	}, nil
}
