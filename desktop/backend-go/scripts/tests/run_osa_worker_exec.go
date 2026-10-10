//go:build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

type QueueItem struct {
	ID               uuid.UUID
	WorkspaceID      uuid.UUID
	TemplateID       uuid.UUID
	Status           string
	Priority         int32
	GenerationContext string
	ErrorMessage     *string
	RetryCount       int32
	CreatedAt        time.Time
	StartedAt        *time.Time
	CompletedAt      *time.Time
}

func main() {
	// Load .env
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Failed to create connection pool: %v", err)
	}
	defer pool.Close()

	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║        OSA QUEUE WORKER - ACTUAL TEST EXECUTION             ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Step 1: Setup test data
	fmt.Println("📋 Step 1: Setting up test data...")
	if err := setupTestTemplates(ctx, pool); err != nil {
		log.Fatalf("Failed to setup templates: %v", err)
	}
	fmt.Println("✅ Test templates ready")
	fmt.Println()

	// Step 2: Get workspace
	fmt.Println("📋 Step 2: Finding test workspace...")
	workspaceID, err := getFirstWorkspace(ctx, pool)
	if err != nil {
		log.Fatalf("Failed to get workspace: %v", err)
	}
	fmt.Printf("✅ Using workspace: %s\n", workspaceID)
	fmt.Println()

	// Step 3: Insert test queue items
	fmt.Println("📋 Step 3: Inserting test queue items...")
	queueIDs, err := insertTestQueueItems(ctx, pool, workspaceID)
	if err != nil {
		log.Fatalf("Failed to insert queue items: %v", err)
	}
	fmt.Printf("✅ Inserted %d queue items\n", len(queueIDs))
	for i, id := range queueIDs {
		fmt.Printf("   %d. %s\n", i+1, id)
	}
	fmt.Println()

	// Step 4: Monitor queue (this is where worker would pick up items)
	fmt.Println("📋 Step 4: Monitoring queue status...")
	fmt.Println("⏱️  Note: Worker needs to be running separately to process items")
	fmt.Println("   Run: ./bin/server-with-worker")
	fmt.Println()

	// Show initial status
	showQueueStatus(ctx, pool)
	fmt.Println()

	// Monitor for changes
	fmt.Println("🔄 Monitoring for 30 seconds (press Ctrl+C to stop early)...")
	fmt.Println()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	timeout := time.After(30 * time.Second)
	iteration := 0

	for {
		select {
		case <-ticker.C:
			iteration++
			fmt.Printf("--- Check #%d at %s ---\n", iteration, time.Now().Format("15:04:05"))
			showQueueStatus(ctx, pool)

			// Check for completed items
			completed, err := getCompletedItems(ctx, pool, queueIDs)
			if err != nil {
				log.Printf("Error checking completed items: %v", err)
			} else if len(completed) > 0 {
				fmt.Println("\n🎉 Completed items detected:")
				for _, item := range completed {
					duration := ""
					if item.StartedAt != nil && item.CompletedAt != nil {
						duration = fmt.Sprintf(" (%.2fs)", item.CompletedAt.Sub(*item.StartedAt).Seconds())
					}
					fmt.Printf("   • %s - %s%s\n", item.ID, item.Status, duration)
					if item.ErrorMessage != nil && *item.ErrorMessage != "" {
						fmt.Printf("     Error: %s\n", *item.ErrorMessage)
					}
				}
			}
			fmt.Println()

		case <-timeout:
			fmt.Println("⏱️  Monitoring period ended")
			goto done
		}
	}

done:
	fmt.Println("\n╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    FINAL STATUS                              ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	showDetailedStatus(ctx, pool, queueIDs)

	// Cleanup option
	fmt.Println("\n⚠️  Test data cleanup:")
	fmt.Println("   To clean up test data, run:")
	fmt.Println("   DELETE FROM app_generation_queue WHERE template_id IN")
	fmt.Println("     (SELECT id FROM app_templates WHERE template_name LIKE 'test_%');")
}

func setupTestTemplates(ctx context.Context, pool *pgxpool.Pool) error {
	query := `
	INSERT INTO app_templates (
		template_name, category, display_name, description, icon_type,
		target_business_types, target_challenges, target_team_sizes,
		priority_score, template_config, required_modules, optional_features,
		generation_prompt, scaffold_type
	) VALUES
	($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	ON CONFLICT (template_name) DO UPDATE SET
		description = EXCLUDED.description,
		updated_at = NOW()
	`

	templates := [][]interface{}{
		{
			"test_crm_basic", "crm", "Test CRM System",
			"A basic CRM system for testing OSA worker functionality",
			"users",
			[]string{"saas", "startup", "small_business"},
			[]string{"customer_management", "sales_tracking"},
			[]string{"solo", "small"},
			80,
			`{"features": ["contacts", "deals", "tasks"], "complexity": "medium"}`,
			[]string{"auth", "database", "api"},
			[]string{"email_integration", "calendar", "reports"},
			"Generate a CRM application with contact management, deal tracking, and task management.",
			"full-stack",
		},
		{
			"test_todo_app", "productivity", "Test Todo Application",
			"A simple todo app for testing worker",
			"check-square",
			[]string{"personal", "startup"},
			[]string{"task_management", "productivity"},
			[]string{"solo"},
			60,
			`{"features": ["tasks", "lists", "tags"], "complexity": "simple"}`,
			[]string{"auth", "database"},
			[]string{"reminders", "collaboration"},
			"Generate a todo list application with task creation and completion tracking.",
			"frontend-focused",
		},
	}

	for _, t := range templates {
		_, err := pool.Exec(ctx, query, t...)
		if err != nil {
			return fmt.Errorf("failed to insert template %s: %w", t[0], err)
		}
	}

	return nil
}

func getFirstWorkspace(ctx context.Context, pool *pgxpool.Pool) (uuid.UUID, error) {
	var id uuid.UUID
	err := pool.QueryRow(ctx, "SELECT id FROM workspaces LIMIT 1").Scan(&id)
	return id, err
}

func insertTestQueueItems(ctx context.Context, pool *pgxpool.Pool, workspaceID uuid.UUID) ([]uuid.UUID, error) {
	query := `
	WITH test_template AS (
		SELECT id FROM app_templates WHERE template_name = $1
	)
	INSERT INTO app_generation_queue (
		workspace_id, template_id, status, priority, generation_context
	)
	SELECT $2, tt.id, 'pending', $3, $4::jsonb
	FROM test_template tt
	RETURNING id
	`

	items := []struct {
		templateName string
		priority     int32
		context      string
	}{
		{
			"test_crm_basic",
			8,
			`{"app_name": "Test CRM from Worker", "description": "Testing OSA queue worker functionality"}`,
		},
		{
			"test_todo_app",
			5,
			`{"app_name": "My Todo List", "description": "Personal task manager"}`,
		},
	}

	var ids []uuid.UUID
	for _, item := range items {
		var id uuid.UUID
		err := pool.QueryRow(ctx, query,
			item.templateName,
			workspaceID,
			item.priority,
			item.context,
		).Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("failed to insert queue item for %s: %w", item.templateName, err)
		}
		ids = append(ids, id)
	}

	return ids, nil
}

func showQueueStatus(ctx context.Context, pool *pgxpool.Pool) {
	var pending, processing, completed, failed int
	err := pool.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE status = 'pending') as pending,
			COUNT(*) FILTER (WHERE status = 'processing') as processing,
			COUNT(*) FILTER (WHERE status = 'completed') as completed,
			COUNT(*) FILTER (WHERE status = 'failed') as failed
		FROM app_generation_queue
	`).Scan(&pending, &processing, &completed, &failed)

	if err != nil {
		log.Printf("Error querying status: %v", err)
		return
	}

	fmt.Printf("   📊 Queue: pending=%d, processing=%d, completed=%d, failed=%d\n",
		pending, processing, completed, failed)
}

func getCompletedItems(ctx context.Context, pool *pgxpool.Pool, queueIDs []uuid.UUID) ([]QueueItem, error) {
	if len(queueIDs) == 0 {
		return nil, nil
	}

	query := `
		SELECT id, workspace_id, template_id, status, priority,
		       generation_context::text, error_message, retry_count,
		       created_at, started_at, completed_at
		FROM app_generation_queue
		WHERE id = ANY($1) AND status IN ('completed', 'failed')
	`

	rows, err := pool.Query(ctx, query, queueIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []QueueItem
	for rows.Next() {
		var item QueueItem
		var startedAt, completedAt pgtype.Timestamptz
		var errorMsg pgtype.Text

		err := rows.Scan(
			&item.ID, &item.WorkspaceID, &item.TemplateID,
			&item.Status, &item.Priority, &item.GenerationContext,
			&errorMsg, &item.RetryCount, &item.CreatedAt,
			&startedAt, &completedAt,
		)
		if err != nil {
			return nil, err
		}

		if startedAt.Valid {
			t := startedAt.Time
			item.StartedAt = &t
		}
		if completedAt.Valid {
			t := completedAt.Time
			item.CompletedAt = &t
		}
		if errorMsg.Valid {
			item.ErrorMessage = &errorMsg.String
		}

		items = append(items, item)
	}

	return items, rows.Err()
}

func showDetailedStatus(ctx context.Context, pool *pgxpool.Pool, queueIDs []uuid.UUID) {
	query := `
		SELECT q.id, q.status, q.priority, q.retry_count,
		       t.template_name, t.display_name,
		       q.created_at, q.started_at, q.completed_at,
		       q.error_message
		FROM app_generation_queue q
		LEFT JOIN app_templates t ON q.template_id = t.id
		WHERE q.id = ANY($1)
		ORDER BY q.priority DESC, q.created_at ASC
	`

	rows, err := pool.Query(ctx, query, queueIDs)
	if err != nil {
		log.Printf("Error querying detailed status: %v", err)
		return
	}
	defer rows.Close()

	fmt.Println()
	for rows.Next() {
		var id uuid.UUID
		var status string
		var priority, retryCount int32
		var templateName, displayName string
		var createdAt time.Time
		var startedAt, completedAt pgtype.Timestamptz
		var errorMsg pgtype.Text

		err := rows.Scan(&id, &status, &priority, &retryCount,
			&templateName, &displayName,
			&createdAt, &startedAt, &completedAt, &errorMsg)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}

		fmt.Printf("📦 Queue Item: %s\n", id)
		fmt.Printf("   Template: %s (%s)\n", displayName, templateName)
		fmt.Printf("   Status: %s | Priority: %d | Retries: %d\n", status, priority, retryCount)
		fmt.Printf("   Created: %s\n", createdAt.Format("2006-01-02 15:04:05"))

		if startedAt.Valid {
			fmt.Printf("   Started: %s\n", startedAt.Time.Format("2006-01-02 15:04:05"))
		}
		if completedAt.Valid {
			fmt.Printf("   Completed: %s\n", completedAt.Time.Format("2006-01-02 15:04:05"))
			if startedAt.Valid {
				duration := completedAt.Time.Sub(startedAt.Time)
				fmt.Printf("   Duration: %.2f seconds\n", duration.Seconds())
			}
		}
		if errorMsg.Valid && errorMsg.String != "" {
			fmt.Printf("   ⚠️  Error: %s\n", errorMsg.String)
		}
		fmt.Println()
	}
}
