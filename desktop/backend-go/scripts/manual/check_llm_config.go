package main
import (
	"fmt"
	"os"
	"github.com/joho/godotenv"
)
func main() {
	godotenv.Load()
	
	fmt.Println("🔍 LLM Configuration:")
	fmt.Println("===================")
	fmt.Printf("AI_PROVIDER: %s\n", os.Getenv("AI_PROVIDER"))
	fmt.Printf("DEFAULT_MODEL: %s\n", os.Getenv("DEFAULT_MODEL"))
	fmt.Printf("OLLAMA_CLOUD_API_KEY: %s\n", os.Getenv("OLLAMA_CLOUD_API_KEY")[:20] + "...")
	fmt.Printf("ANTHROPIC_API_KEY: %s\n", os.Getenv("ANTHROPIC_API_KEY"))
	fmt.Printf("GROQ_API_KEY: %s\n", os.Getenv("GROQ_API_KEY"))
}
