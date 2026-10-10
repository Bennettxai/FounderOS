package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()

	// Check workspaces
	fmt.Println("=== WORKSPACES ===")
	rows, err := db.QueryContext(ctx, "SELECT id, name, created_at FROM workspaces LIMIT 10")
	if err != nil {
		log.Printf("Error querying workspaces: %v", err)
	} else {
		defer rows.Close()
		count := 0
		for rows.Next() {
			var id, name string
			var createdAt time.Time
			if err := rows.Scan(&id, &name, &createdAt); err != nil {
				log.Printf("Error scanning workspace: %v", err)
				continue
			}
			fmt.Printf("ID: %s, Name: %s, Created: %s\n", id, name, createdAt.Format(time.RFC3339))
			count++
		}
		fmt.Printf("Total workspaces: %d\n\n", count)
	}

	// Check app_generation_queue schema
	fmt.Println("=== APP_GENERATION_QUEUE SCHEMA ===")
	schemaRows, err := db.QueryContext(ctx, `
		SELECT column_name, data_type, is_nullable, column_default
		FROM information_schema.columns
		WHERE table_name = 'app_generation_queue'
		ORDER BY ordinal_position
	`)
	if err != nil {
		log.Printf("Error querying schema: %v", err)
	} else {
		defer schemaRows.Close()
		for schemaRows.Next() {
			var colName, dataType, isNullable string
			var colDefault sql.NullString
			if err := schemaRows.Scan(&colName, &dataType, &isNullable, &colDefault); err != nil {
				log.Printf("Error scanning schema: %v", err)
				continue
			}
			defaultVal := "NULL"
			if colDefault.Valid {
				defaultVal = colDefault.String
			}
			fmt.Printf("Column: %s, Type: %s, Nullable: %s, Default: %s\n",
				colName, dataType, isNullable, defaultVal)
		}
		fmt.Println()
	}

	// Check app_generation_queue contents
	fmt.Println("=== APP_GENERATION_QUEUE CONTENTS ===")
	queueRows, err := db.QueryContext(ctx, `
		SELECT id, workspace_id, template_id, status, priority,
		       generation_context, error_message, retry_count, max_retries,
		       created_at, started_at, completed_at
		FROM app_generation_queue
		ORDER BY created_at DESC
	`)
	if err != nil {
		log.Printf("Error querying queue: %v", err)
	} else {
		defer queueRows.Close()
		count := 0
		for queueRows.Next() {
			var id, workspaceID string
			var templateID sql.NullString
			var status sql.NullString
			var priority, retryCount, maxRetries sql.NullInt64
			var genContext sql.NullString
			var errorMsg sql.NullString
			var createdAt sql.NullTime
			var startedAt, completedAt sql.NullTime

			if err := queueRows.Scan(&id, &workspaceID, &templateID, &status, &priority,
				&genContext, &errorMsg, &retryCount, &maxRetries,
				&createdAt, &startedAt, &completedAt); err != nil {
				log.Printf("Error scanning queue item: %v", err)
				continue
			}

			item := map[string]interface{}{
				"id":           id,
				"workspace_id": workspaceID,
			}

			if templateID.Valid {
				item["template_id"] = templateID.String
			}
			if status.Valid {
				item["status"] = status.String
			}
			if priority.Valid {
				item["priority"] = priority.Int64
			}
			if genContext.Valid {
				item["generation_context"] = genContext.String
			}
			if errorMsg.Valid {
				item["error_message"] = errorMsg.String
			}
			if retryCount.Valid {
				item["retry_count"] = retryCount.Int64
			}
			if maxRetries.Valid {
				item["max_retries"] = maxRetries.Int64
			}
			if createdAt.Valid {
				item["created_at"] = createdAt.Time.Format(time.RFC3339)
			}
			if startedAt.Valid {
				item["started_at"] = startedAt.Time.Format(time.RFC3339)
			}
			if completedAt.Valid {
				item["completed_at"] = completedAt.Time.Format(time.RFC3339)
			}

			jsonData, _ := json.MarshalIndent(item, "", "  ")
			fmt.Println(string(jsonData))
			fmt.Println("---")
			count++
		}
		fmt.Printf("Total queue items: %d\n", count)
	}
}
