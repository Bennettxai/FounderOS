//go:build ignore

package main
import ("context"; "fmt"; "os"; "github.com/jackc/pgx/v5/pgxpool"; "github.com/joho/godotenv")
func main() {
	godotenv.Load()
	pool, _ := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	defer pool.Close()
	tables := []string{"app_templates", "user_generated_apps", "app_generation_queue"}
	for _, t := range tables {
		var count int
		pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM information_schema.columns WHERE table_name = $1", t).Scan(&count)
		if count > 0 {
			var rows int
			pool.QueryRow(context.Background(), fmt.Sprintf("SELECT COUNT(*) FROM \"%s\"", t)).Scan(&rows)
			fmt.Printf("✅ %s - %d columns, %d rows\n", t, count, rows)
		} else {
			fmt.Printf("❌ %s - NOT FOUND\n", t)
		}
	}
}
