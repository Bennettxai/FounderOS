//go:build ignore

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

// Benchmark system performance (database, queries, etc.)
// Usage: go run scripts/benchmark_system.go

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                  SYSTEM PERFORMANCE BENCHMARK                    ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	godotenv.Load()
	ctx := context.Background()

	dbURL := os.Getenv("DATABASE_URL")
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		fmt.Printf("❌ Database connection failed: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	results := make(map[string]time.Duration)

	// Benchmark 1: Simple SELECT
	fmt.Println("🔍 Benchmark 1: Simple SELECT query")
	start := time.Now()
	for i := 0; i < 100; i++ {
		var result int
		pool.QueryRow(ctx, "SELECT 1").Scan(&result)
	}
	results["Simple SELECT (100x)"] = time.Since(start)
	fmt.Printf("   ⏱️  %v (avg: %v per query)\n", results["Simple SELECT (100x)"],
		results["Simple SELECT (100x)"]/100)
	fmt.Println()

	// Benchmark 2: Table count query
	fmt.Println("🔍 Benchmark 2: COUNT query")
	start = time.Now()
	for i := 0; i < 50; i++ {
		var count int
		pool.QueryRow(ctx, "SELECT COUNT(*) FROM app_templates").Scan(&count)
	}
	results["COUNT query (50x)"] = time.Since(start)
	fmt.Printf("   ⏱️  %v (avg: %v per query)\n", results["COUNT query (50x)"],
		results["COUNT query (50x)"]/50)
	fmt.Println()

	// Benchmark 3: Complex JOIN query
	fmt.Println("🔍 Benchmark 3: Complex JOIN query")
	start = time.Now()
	for i := 0; i < 20; i++ {
		rows, _ := pool.Query(ctx, `
			SELECT t.template_name, COUNT(q.id) as queue_count
			FROM app_templates t
			LEFT JOIN app_generation_queue q ON q.template_id = t.id
			GROUP BY t.id, t.template_name
		`)
		rows.Close()
	}
	results["Complex JOIN (20x)"] = time.Since(start)
	fmt.Printf("   ⏱️  %v (avg: %v per query)\n", results["Complex JOIN (20x)"],
		results["Complex JOIN (20x)"]/20)
	fmt.Println()

	// Benchmark 4: INSERT operation
	fmt.Println("🔍 Benchmark 4: INSERT operations")
	start = time.Now()
	testIDs := make([]uuid.UUID, 10)
	for i := 0; i < 10; i++ {
		testIDs[i] = uuid.New()
		pool.Exec(ctx, `
			INSERT INTO "user" (id, email, password_hash, name)
			VALUES ($1, $2, $3, $4)
		`, testIDs[i].String(), fmt.Sprintf("bench%d@example.com", i), "hash", "Bench User")
	}
	results["INSERT (10x)"] = time.Since(start)
	fmt.Printf("   ⏱️  %v (avg: %v per insert)\n", results["INSERT (10x)"],
		results["INSERT (10x)"]/10)
	fmt.Println()

	// Cleanup benchmark data
	for _, id := range testIDs {
		pool.Exec(ctx, `DELETE FROM "user" WHERE id = $1`, id.String())
	}

	// Benchmark 5: Transaction
	fmt.Println("🔍 Benchmark 5: Transaction (BEGIN/COMMIT)")
	start = time.Now()
	for i := 0; i < 20; i++ {
		tx, _ := pool.Begin(ctx)
		var result int
		tx.QueryRow(ctx, "SELECT 1").Scan(&result)
		tx.Commit(ctx)
	}
	results["Transaction (20x)"] = time.Since(start)
	fmt.Printf("   ⏱️  %v (avg: %v per transaction)\n", results["Transaction (20x)"],
		results["Transaction (20x)"]/20)
	fmt.Println()

	// Summary
	fmt.Println("╔══════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    BENCHMARK RESULTS                             ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	total := time.Duration(0)
	for name, duration := range results {
		fmt.Printf("%-30s %v\n", name+":", duration)
		total += duration
	}
	fmt.Println()
	fmt.Printf("Total benchmark time: %v\n", total)
	fmt.Println()

	// Performance assessment
	avgSimple := results["Simple SELECT (100x)"] / 100
	if avgSimple < 5*time.Millisecond {
		fmt.Println("🟢 Performance: EXCELLENT (< 5ms per simple query)")
	} else if avgSimple < 10*time.Millisecond {
		fmt.Println("🟡 Performance: GOOD (5-10ms per simple query)")
	} else if avgSimple < 50*time.Millisecond {
		fmt.Println("🟠 Performance: ACCEPTABLE (10-50ms per simple query)")
	} else {
		fmt.Println("🔴 Performance: SLOW (> 50ms per simple query)")
		fmt.Println("   Consider checking database connection or server load")
	}

	fmt.Println()
	fmt.Println("💡 Tips:")
	fmt.Println("   • Lower is better")
	fmt.Println("   • Local DB should be < 5ms per query")
	fmt.Println("   • Supabase (remote) typically 20-50ms per query")
	fmt.Println("   • High latency may affect E2E testing speed")
}
