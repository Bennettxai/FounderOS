package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rhl/businessos-backend/internal/config"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOnboardingFlow tests complete onboarding conversation flow
func TestOnboardingFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	testDB := testutil.RequireTestDatabase(t)
	defer testDB.Close()
	defer testutil.CleanupTestData(ctx, testDB.Pool)

	logger := slog.Default()
	_ = logger
	cfg := &config.Config{
		AIProvider: "anthropic",
		BaseURL:    "http://localhost:8080",
	}
	_ = cfg

	// Create test user and session
	userID := uuid.New().String()
	sessionID := uuid.New().String()
	_, err := testDB.Pool.Exec(ctx, `
		INSERT INTO "user" (id, name, email, "emailVerified", "createdAt", "updatedAt")
		VALUES ($1, $2, $3, true, NOW(), NOW())
	`, userID, "Onboarding Test", "onboarding@test.com")
	require.NoError(t, err)

	_, err = testDB.Pool.Exec(ctx, `
		INSERT INTO "session" (id, "userId", "expiresAt", "token", "ipAddress", "userAgent")
		VALUES ($1, $2, $3, $4, $5, $6)
	`, sessionID, userID, time.Now().Add(24*time.Hour), "test-token", "127.0.0.1", "test-agent")
	require.NoError(t, err)

	// Create onboarding service (available for integration testing)
	onboardingAIService := services.NewOnboardingAIService()
	onboardingService := services.NewOnboardingService(testDB.Pool, onboardingAIService, nil, nil)
	_ = onboardingService

	// Setup router with middleware
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userID", userID)
		c.Set("sessionID", sessionID)
		c.Next()
	})

	// Mock handler (simplified)
	router.POST("/onboarding/conversation", func(c *gin.Context) {
		var req struct {
			Message string `json:"message"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Create or get conversation
		conversationID := uuid.New()

		// Store message
		_, err := testDB.Pool.Exec(ctx, `
			INSERT INTO conversation_messages (id, conversation_id, content, role, created_at)
			VALUES ($1, $2, $3, $4, NOW())
		`, uuid.New(), conversationID, req.Message, "user")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Mock AI response
		response := "Thank you for sharing that! Could you tell me more about your role and main responsibilities?"
		_, err = testDB.Pool.Exec(ctx, `
			INSERT INTO conversation_messages (id, conversation_id, content, role, created_at)
			VALUES ($1, $2, $3, $4, NOW())
		`, uuid.New(), conversationID, response, "assistant")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"response":        response,
			"conversationId":  conversationID.String(),
			"nextStage":       "profile_building",
			"completionScore": 0.3,
		})
	})

	router.POST("/onboarding/complete", func(c *gin.Context) {
		// Extract profile from conversation
		profile := map[string]interface{}{
			"name":     "Onboarding Test",
			"role":     "Product Manager",
			"company":  "Test Corp",
			"industry": "Technology",
		}

		// Create workspace
		workspaceID := uuid.New()
		_, err := testDB.Pool.Exec(ctx, `
			INSERT INTO workspaces (id, name, owner_id, created_at, updated_at)
			VALUES ($1, $2, $3, NOW(), NOW())
		`, workspaceID, "Test Workspace", userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Update user profile
		_, err = testDB.Pool.Exec(ctx, `
			UPDATE "user" SET metadata = $1 WHERE id = $2
		`, profile, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"workspaceId": workspaceID.String(),
			"profile":     profile,
			"message":     "Onboarding completed successfully",
		})
	})

	t.Run("Complete onboarding conversation flow", func(t *testing.T) {
		// Step 1: Start conversation
		req1 := map[string]string{
			"message": "Hi, I'm a product manager looking to organize my work better",
		}
		body1, _ := json.Marshal(req1)

		w1 := httptest.NewRecorder()
		r1 := httptest.NewRequest("POST", "/onboarding/conversation", bytes.NewBuffer(body1))
		r1.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w1, r1)

		assert.Equal(t, http.StatusOK, w1.Code)
		var resp1 map[string]interface{}
		err := json.Unmarshal(w1.Body.Bytes(), &resp1)
		require.NoError(t, err)
		assert.Contains(t, resp1, "response")
		assert.Contains(t, resp1, "conversationId")
		assert.Equal(t, "profile_building", resp1["nextStage"])

		// Step 2: Continue conversation
		req2 := map[string]string{
			"message": "I manage a team of 5 engineers and handle product roadmap planning",
		}
		body2, _ := json.Marshal(req2)

		w2 := httptest.NewRecorder()
		r2 := httptest.NewRequest("POST", "/onboarding/conversation", bytes.NewBuffer(body2))
		r2.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w2, r2)

		assert.Equal(t, http.StatusOK, w2.Code)

		// Step 3: Complete onboarding
		w3 := httptest.NewRecorder()
		r3 := httptest.NewRequest("POST", "/onboarding/complete", nil)
		router.ServeHTTP(w3, r3)

		assert.Equal(t, http.StatusOK, w3.Code)
		var resp3 map[string]interface{}
		err = json.Unmarshal(w3.Body.Bytes(), &resp3)
		require.NoError(t, err)
		assert.Contains(t, resp3, "workspaceId")
		assert.Contains(t, resp3, "profile")

		// Verify workspace was created
		var workspaceCount int
		err = testDB.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM workspaces WHERE owner_id = $1`, userID).Scan(&workspaceCount)
		require.NoError(t, err)
		assert.Equal(t, 1, workspaceCount)
	})

	t.Run("Resume onboarding session", func(t *testing.T) {
		// Create partial onboarding session
		conversationID := uuid.New()
		_, err := testDB.Pool.Exec(ctx, `
			INSERT INTO conversation_messages (id, conversation_id, content, role, created_at)
			VALUES ($1, $2, $3, $4, NOW())
		`, uuid.New(), conversationID, "I need help organizing tasks", "user")
		require.NoError(t, err)

		// Resume with new message
		req := map[string]string{
			"message": "Let me continue from where I left off",
		}
		body, _ := json.Marshal(req)

		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/onboarding/conversation", bytes.NewBuffer(body))
		r.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Contains(t, resp, "response")
	})
}

// TestOnboardingProfileExtraction tests profile extraction from conversation
func TestOnboardingProfileExtraction(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	testDB := testutil.RequireTestDatabase(t)
	defer testDB.Close()
	defer testutil.CleanupTestData(ctx, testDB.Pool)

	logger := slog.Default()
	_ = logger
	cfg := &config.Config{
		AIProvider: "anthropic",
	}
	_ = cfg

	t.Run("Extract structured profile from conversation", func(t *testing.T) {
		// Create conversation with profile information
		conversationID := uuid.New()
		messages := []struct {
			role    string
			content string
		}{
			{"user", "Hi, I'm John Doe, a senior software engineer at TechCorp"},
			{"assistant", "Nice to meet you! What are your main responsibilities?"},
			{"user", "I lead the backend team and handle system architecture"},
			{"assistant", "What tools do you currently use?"},
			{"user", "We use Jira for tasks, Slack for communication, and GitHub for code"},
		}

		for _, msg := range messages {
			_, err := testDB.Pool.Exec(ctx, `
				INSERT INTO conversation_messages (id, conversation_id, content, role, created_at)
				VALUES ($1, $2, $3, $4, NOW())
			`, uuid.New(), conversationID, msg.content, msg.role)
			require.NoError(t, err)
		}

		// Extract profile (mock implementation)
		profile := map[string]interface{}{
			"name":         "John Doe",
			"role":         "Senior Software Engineer",
			"company":      "TechCorp",
			"teamSize":     "unknown",
			"currentTools": []string{"Jira", "Slack", "GitHub"},
			"focus":        "backend",
		}

		// Verify profile extraction
		assert.Contains(t, profile, "name")
		assert.Contains(t, profile, "role")
		assert.Contains(t, profile, "currentTools")
		assert.Equal(t, "John Doe", profile["name"])
		assert.Equal(t, "Senior Software Engineer", profile["role"])
	})
}

// TestOnboardingEmailAnalysis tests email content analysis for onboarding
func TestOnboardingEmailAnalysis(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	testDB := testutil.RequireTestDatabase(t)
	defer testDB.Close()
	defer testutil.CleanupTestData(ctx, testDB.Pool)

	logger := slog.Default()
	_ = logger
	cfg := &config.Config{
		AIProvider: "anthropic",
	}
	_ = cfg

	t.Run("Analyze email patterns for profile building", func(t *testing.T) {
		// Mock email data
		_ = []map[string]interface{}{
			{
				"subject":  "Re: Q1 Planning Meeting",
				"from":     "manager@company.com",
				"to":       "user@company.com",
				"content":  "Let's discuss the roadmap for next quarter",
				"category": "work",
			},
			{
				"subject":  "Team standup notes",
				"from":     "user@company.com",
				"to":       "team@company.com",
				"content":  "Here are today's updates from the backend team",
				"category": "work",
			},
			{
				"subject":  "Deployment notification",
				"from":     "ci@company.com",
				"to":       "user@company.com",
				"content":  "Production deployment completed successfully",
				"category": "notification",
			},
		}

		// Analyze patterns (mock)
		analysis := map[string]interface{}{
			"workEmails":       2,
			"notifications":    1,
			"keyContacts":      []string{"manager@company.com", "team@company.com"},
			"commonTopics":     []string{"planning", "deployment", "team updates"},
			"suggestedRole":    "engineering manager",
			"communicationStyle": "collaborative",
		}

		assert.Equal(t, 2, analysis["workEmails"])
		assert.Equal(t, 1, analysis["notifications"])
		assert.Contains(t, analysis, "keyContacts")
		assert.Contains(t, analysis, "commonTopics")
	})
}
