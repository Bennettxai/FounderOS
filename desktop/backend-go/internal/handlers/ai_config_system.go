package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/prompts"
	"github.com/rhl/businessos-backend/internal/utils"
)

// GetSystemInfo returns system hardware info and model recommendations
func (h *AIConfigHandler) GetSystemInfo(c *gin.Context) {
	info := SystemInfo{
		Platform: runtime.GOOS,
	}

	// Get actual memory info
	totalRAM, availRAM := getSystemMemory()
	info.TotalRAM = totalRAM
	info.AvailableRAM = availRAM

	// Check for GPU (basic check for macOS Metal support)
	if runtime.GOOS == "darwin" {
		info.HasGPU = true
		info.GPUName = "Apple Silicon / Metal"
	}

	// Model recommendations based on RAM
	if info.TotalRAM >= 32 {
		info.RecommendedModels = []RecommendedModel{
			{Name: "llama3.2:latest", Description: "Best balance of speed and quality", RAMRequired: "8GB", Speed: "Fast", Quality: "Excellent"},
			{Name: "qwen2.5:14b", Description: "Strong reasoning, larger context", RAMRequired: "12GB", Speed: "Medium", Quality: "Excellent"},
			{Name: "mistral:7b", Description: "Great for general tasks", RAMRequired: "6GB", Speed: "Fast", Quality: "Good"},
			{Name: "codellama:13b", Description: "Best for code tasks", RAMRequired: "10GB", Speed: "Medium", Quality: "Excellent"},
		}
	} else if info.TotalRAM >= 16 {
		info.RecommendedModels = []RecommendedModel{
			{Name: "llama3.2:3b", Description: "Fast and efficient", RAMRequired: "4GB", Speed: "Very Fast", Quality: "Good"},
			{Name: "llama3.2:latest", Description: "Best balance", RAMRequired: "8GB", Speed: "Fast", Quality: "Excellent"},
			{Name: "phi3:mini", Description: "Microsoft's efficient model", RAMRequired: "3GB", Speed: "Very Fast", Quality: "Good"},
			{Name: "mistral:7b", Description: "Great for general tasks", RAMRequired: "6GB", Speed: "Fast", Quality: "Good"},
		}
	} else {
		info.RecommendedModels = []RecommendedModel{
			{Name: "llama3.2:1b", Description: "Minimal resources", RAMRequired: "2GB", Speed: "Very Fast", Quality: "Basic"},
			{Name: "phi3:mini", Description: "Efficient small model", RAMRequired: "3GB", Speed: "Very Fast", Quality: "Good"},
			{Name: "tinyllama", Description: "Ultra lightweight", RAMRequired: "1GB", Speed: "Instant", Quality: "Basic"},
		}
	}

	c.JSON(http.StatusOK, info)
}

// GetAgentPrompts returns all available agent prompts
func (h *AIConfigHandler) GetAgentPrompts(c *gin.Context) {
	agents := []AgentInfo{
		{
			ID:          "default",
			Name:        "Business OS Assistant",
			Description: "General business operations assistant for comprehensive guidance",
			Prompt:      prompts.DefaultPrompt,
			Category:    "general",
		},
		{
			ID:          "document",
			Name:        "Document Creator",
			Description: "Creates polished, professional business documents with real content",
			Prompt:      prompts.DocumentCreatorPrompt,
			Category:    "specialist",
		},
		{
			ID:          "analyst",
			Name:        "Business Analyst",
			Description: "Analyzes data, identifies insights, and provides strategic recommendations",
			Prompt:      prompts.AnalystPrompt,
			Category:    "specialist",
		},
		{
			ID:          "planner",
			Name:        "Strategic Planner",
			Description: "Helps with planning, prioritization, and strategic thinking",
			Prompt:      prompts.PlannerPrompt,
			Category:    "specialist",
		},
		{
			ID:          "orchestrator",
			Name:        "Orchestrator",
			Description: "Main coordinator that routes requests to specialized agents",
			Prompt:      prompts.OrchestratorPrompt,
			Category:    "system",
		},
		{
			ID:          "daily_planning",
			Name:        "Daily Planning Assistant",
			Description: "Executive assistant for daily productivity and prioritization",
			Prompt:      prompts.DailyPlanningPrompt,
			Category:    "specialist",
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"agents": agents,
	})
}

// GetAgentPrompt returns a specific agent's prompt
func (h *AIConfigHandler) GetAgentPrompt(c *gin.Context) {
	agentID := c.Param("id")
	prompt := prompts.GetPrompt(agentID)

	if prompt == "" {
		utils.RespondNotFound(c, slog.Default(), "Agent")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":     agentID,
		"prompt": prompt,
	})
}

// formatSize formats bytes to human readable string
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return "< 1 KB"
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	units := []string{"KB", "MB", "GB", "TB"}
	value := float64(bytes) / float64(div)
	if exp < len(units) {
		return fmt.Sprintf("%.1f %s", value, units[exp])
	}
	return fmt.Sprintf("%.1f TB", value)
}

// getSystemMemory returns total and available RAM in GB
func getSystemMemory() (total int64, available int64) {
	switch runtime.GOOS {
	case "darwin":
		// Get total memory using sysctl on macOS
		out, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
		if err == nil {
			if bytes, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64); err == nil {
				total = bytes / (1024 * 1024 * 1024)
			}
		}

		// Get available memory using vm_stat
		out, err = exec.Command("vm_stat").Output()
		if err == nil {
			lines := strings.Split(string(out), "\n")
			var freePages, inactivePages int64
			pageSize := int64(4096) // Default page size

			for _, line := range lines {
				if strings.Contains(line, "page size") {
					parts := strings.Fields(line)
					for _, p := range parts {
						if val, err := strconv.ParseInt(p, 10, 64); err == nil && val > 0 {
							pageSize = val
							break
						}
					}
				}
				if strings.Contains(line, "Pages free:") {
					parts := strings.Fields(line)
					if len(parts) >= 3 {
						val := strings.TrimSuffix(parts[2], ".")
						freePages, _ = strconv.ParseInt(val, 10, 64)
					}
				}
				if strings.Contains(line, "Pages inactive:") {
					parts := strings.Fields(line)
					if len(parts) >= 3 {
						val := strings.TrimSuffix(parts[2], ".")
						inactivePages, _ = strconv.ParseInt(val, 10, 64)
					}
				}
			}
			available = (freePages + inactivePages) * pageSize / (1024 * 1024 * 1024)
		}

	case "linux":
		// Read /proc/meminfo on Linux
		content, err := os.ReadFile("/proc/meminfo")
		if err == nil {
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "MemTotal:") {
					parts := strings.Fields(line)
					if len(parts) >= 2 {
						if kb, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
							total = kb / (1024 * 1024)
						}
					}
				}
				if strings.HasPrefix(line, "MemAvailable:") {
					parts := strings.Fields(line)
					if len(parts) >= 2 {
						if kb, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
							available = kb / (1024 * 1024)
						}
					}
				}
			}
		}
	}

	// Defaults if detection failed
	if total == 0 {
		total = 16
	}
	if available == 0 {
		available = total / 2
	}

	return total, available
}
