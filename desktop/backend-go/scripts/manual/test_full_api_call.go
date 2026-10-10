package main
import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"github.com/joho/godotenv"
	"github.com/jackc/pgx/v5/pgxpool"
)
func main() {
	godotenv.Load()
	pool, _ := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	defer pool.Close()
	
	// Get EXACT data as backend GetCustomAgent returns
	var agent struct {
		ID          string  `json:"id"`
		Name        string  `json:"name"`
		DisplayName string  `json:"display_name"`
		Description *string `json:"description"`
		SystemPrompt string `json:"system_prompt"`
		IsActive    *bool   `json:"is_active"`
		TimesUsed   *int32  `json:"times_used"`
	}
	
	err := pool.QueryRow(context.Background(), `
		SELECT id, name, display_name, description, system_prompt, is_active, times_used
		FROM custom_agents
		WHERE id = $1
	`, "32814bf2-5ac8-4926-b534-ad5f2168f8be").Scan(
		&agent.ID, &agent.Name, &agent.DisplayName, &agent.Description, 
		&agent.SystemPrompt, &agent.IsActive, &agent.TimesUsed,
	)
	
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	jsonData, _ := json.MarshalIndent(agent, "", "  ")
	fmt.Println("🌐 EXACT API Response for agent detail:")
	fmt.Println(string(jsonData))
	
	fmt.Printf("\n📊 Field Analysis:\n")
	fmt.Printf("   is_active pointer: %p\n", agent.IsActive)
	if agent.IsActive != nil {
		fmt.Printf("   is_active value: %v\n", *agent.IsActive)
		if *agent.IsActive {
			fmt.Println("   ✅ Should show: GREEN 'Active'")
		} else {
			fmt.Println("   ❌ Should show: GRAY 'Inactive'")
		}
	} else {
		fmt.Println("   ⚠️  is_active is NULL - will show as Inactive!")
	}
}
