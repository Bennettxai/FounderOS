package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// AddMessage adds a message to the conversation
func (s *OnboardingService) AddMessage(ctx context.Context, sessionID uuid.UUID, role, content string, confidenceScore *float64, questionType *string) (*ConversationMessage, error) {
	// Get next sequence number
	var seqNum int
	err := s.pool.QueryRow(ctx, `
		SELECT COALESCE(MAX(sequence_number), 0) + 1
		FROM onboarding_conversation_history
		WHERE session_id = $1
	`, sessionID).Scan(&seqNum)
	if err != nil {
		return nil, fmt.Errorf("get sequence number: %w", err)
	}

	var msg ConversationMessage
	err = s.pool.QueryRow(ctx, `
		INSERT INTO onboarding_conversation_history (session_id, role, content, confidence_score, question_type, sequence_number)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, session_id, role, content, confidence_score, extracted_fields, question_type, sequence_number, created_at
	`, sessionID, role, content, confidenceScore, questionType, seqNum).Scan(
		&msg.ID, &msg.SessionID, &msg.Role, &msg.Content, &msg.ConfidenceScore,
		&msg.ExtractedFields, &msg.QuestionType, &msg.SequenceNumber, &msg.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("add message: %w", err)
	}

	return &msg, nil
}

// GetConversationHistory retrieves conversation messages
func (s *OnboardingService) GetConversationHistory(ctx context.Context, sessionID uuid.UUID, limit int) ([]ConversationMessage, error) {
	query := `
		SELECT id, session_id, role, content, confidence_score, extracted_fields, question_type, sequence_number, created_at
		FROM onboarding_conversation_history
		WHERE session_id = $1
		ORDER BY sequence_number ASC
	`
	if limit > 0 {
		query = fmt.Sprintf(`
			SELECT * FROM (
				SELECT id, session_id, role, content, confidence_score, extracted_fields, question_type, sequence_number, created_at
				FROM onboarding_conversation_history
				WHERE session_id = $1
				ORDER BY sequence_number DESC
				LIMIT %d
			) sub ORDER BY sequence_number ASC
		`, limit)
	}

	rows, err := s.pool.Query(ctx, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get conversation history: %w", err)
	}
	defer rows.Close()

	var messages []ConversationMessage
	for rows.Next() {
		var msg ConversationMessage
		err := rows.Scan(
			&msg.ID, &msg.SessionID, &msg.Role, &msg.Content, &msg.ConfidenceScore,
			&msg.ExtractedFields, &msg.QuestionType, &msg.SequenceNumber, &msg.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// ProcessUserMessage processes a user message and returns AI response
func (s *OnboardingService) ProcessUserMessage(ctx context.Context, sessionID uuid.UUID, userID, content string) (*SendMessageResponse, error) {
	// Get session
	session, err := s.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Verify ownership
	if session.UserID != userID {
		return nil, fmt.Errorf("session does not belong to user")
	}

	// Check if session is still valid
	if session.Status != "in_progress" {
		return nil, fmt.Errorf("session is not in progress")
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("session has expired")
	}

	// Add user message
	_, err = s.AddMessage(ctx, sessionID, "user", content, nil, nil)
	if err != nil {
		return nil, err
	}

	// Process based on current step (chip selections vs chat)
	response, err := s.processStepResponse(ctx, session, content)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// processStepResponse handles the response based on current step
func (s *OnboardingService) processStepResponse(ctx context.Context, session *OnboardingSession, content string) (*SendMessageResponse, error) {
	response := &SendMessageResponse{}
	extractedData := ExtractedOnboardingData{}

	// Parse existing extracted data
	if session.ExtractedData != nil {
		dataBytes, _ := json.Marshal(session.ExtractedData)
		json.Unmarshal(dataBytes, &extractedData)
	}

	// Get conversation history for AI context
	history, err := s.GetConversationHistory(ctx, session.ID, 10)
	if err != nil {
		history = []ConversationMessage{} // Continue without history if error
	}

	// Convert to AI message format
	var aiHistory []OnboardingChatMessage
	for _, msg := range history {
		role := msg.Role
		if role == "agent" {
			role = "assistant"
		}
		aiHistory = append(aiHistory, OnboardingChatMessage{
			Role:    role,
			Content: msg.Content,
		})
	}

	// Call AI to process the message and get response
	aiResponse, aiErr := s.aiService.ProcessMessage(
		ctx,
		content,
		session.CurrentStep,
		structToMap(extractedData),
		aiHistory,
	)

	// Variables for step processing
	nextStep := session.CurrentStep
	agentMessage := ""
	var validationError *ValidationError

	// Process based on current step with validation
	switch session.CurrentStep {
	case "company_name":
		// Validate company name
		if err := s.validator.ValidateCompanyName(content); err != nil {
			validationError = err
			if aiErr == nil && aiResponse != nil && aiResponse.AgentMessage != "" {
				agentMessage = aiResponse.AgentMessage
			} else {
				agentMessage = fmt.Sprintf("Hmm, %s. Could you try a different name?", err.Message)
			}
		} else {
			extractedData.WorkspaceName = s.validator.SanitizeInput(content)
			nextStep = "business_type"
			if aiErr == nil && aiResponse != nil && aiResponse.AgentMessage != "" {
				agentMessage = aiResponse.AgentMessage
			} else {
				agentMessage = "Great name! What kind of work does " + extractedData.WorkspaceName + " do?"
			}
		}

	case "business_type":
		// Normalize for validation but store raw input
		normalized := s.validator.NormalizeBusinessType(content)
		if err := s.validator.ValidateBusinessType(normalized); err != nil {
			validationError = err
			if aiErr == nil && aiResponse != nil && aiResponse.AgentMessage != "" {
				agentMessage = aiResponse.AgentMessage
			} else {
				agentMessage = "I didn't quite catch that. Are you an agency, startup, freelancer, consultant, or something else?"
			}
		} else {
			extractedData.BusinessType = s.validator.SanitizeInput(content) // Store sanitized input
			extractedData.NormalizedBusinessType = normalized               // Keep normalized for logic
			if normalized == "freelance" {
				extractedData.TeamSize = "solo (freelancer)"
				nextStep = "role"
				if aiErr == nil && aiResponse != nil && aiResponse.AgentMessage != "" {
					agentMessage = aiResponse.AgentMessage
				} else {
					agentMessage = "Solo power! What's your role - founder, consultant, or something else?"
				}
			} else {
				nextStep = "team_size"
				if aiErr == nil && aiResponse != nil && aiResponse.AgentMessage != "" {
					agentMessage = aiResponse.AgentMessage
				} else {
					agentMessage = "Nice! How big is your team?"
				}
			}
		}

	case "team_size":
		// Normalize for validation but store raw input
		normalized := s.validator.NormalizeTeamSize(content)
		if err := s.validator.ValidateTeamSize(normalized); err != nil {
			validationError = err
			if aiErr == nil && aiResponse != nil && aiResponse.AgentMessage != "" {
				agentMessage = aiResponse.AgentMessage
			} else {
				agentMessage = "Could you tell me roughly how many people are on your team? Just you, 2-5, 6-10, 11-50, or more?"
			}
		} else {
			extractedData.TeamSize = s.validator.SanitizeInput(content) // Store sanitized input
			extractedData.NormalizedTeamSize = normalized               // Keep normalized for logic
			nextStep = "role"
			if aiErr == nil && aiResponse != nil && aiResponse.AgentMessage != "" {
				agentMessage = aiResponse.AgentMessage
			} else {
				agentMessage = "Got it! And what's your role in the team?"
			}
		}

	case "role":
		// Store sanitized input and normalize for internal use
		extractedData.Role = s.validator.SanitizeInput(content)           // Store sanitized input
		extractedData.NormalizedRole = s.validator.NormalizeRole(content) // Keep normalized for logic
		nextStep = "challenge"
		if aiErr == nil && aiResponse != nil && aiResponse.AgentMessage != "" {
			agentMessage = aiResponse.AgentMessage
		} else {
			agentMessage = "Awesome! What's the biggest challenge you're hoping to solve with Business OS?"
		}

	case "challenge":
		// Validate challenge
		if err := s.validator.ValidateChallenge(content); err != nil {
			validationError = err
			if aiErr == nil && aiResponse != nil && aiResponse.AgentMessage != "" {
				agentMessage = aiResponse.AgentMessage
			} else {
				agentMessage = "Could you tell me a bit more about the challenges you're facing? A sentence or two would help me understand."
			}
		} else {
			extractedData.Challenge = s.validator.SanitizeInput(content)
			nextStep = "integrations"
			response.RecommendedIntegrations = ComputeRecommendations(extractedData.Challenge, extractedData.BusinessType)
			if aiErr == nil && aiResponse != nil && aiResponse.AgentMessage != "" {
				agentMessage = aiResponse.AgentMessage
			} else {
				agentMessage = "I hear you! Based on what you've shared, I've got some tool recommendations. Let's connect your favorites!"
			}
			response.IsComplete = false
		}

	case "integrations":
		nextStep = "complete"
		response.IsComplete = true
		agentMessage = "Perfect! Your workspace is ready. Let's get started!"
	}

	// If validation failed, don't advance the step
	if validationError != nil {
		response.Message = ConversationMessage{
			Role:    "agent",
			Content: agentMessage,
		}
		response.NextStep = session.CurrentStep // Stay on current step
		response.ExtractedData = extractedData

		// Add agent message for the retry prompt
		_, _ = s.AddMessage(ctx, session.ID, "agent", agentMessage, nil, nil)

		return response, nil
	}

	// Update session
	stepsCompleted := append(session.StepsCompleted, session.CurrentStep)
	extractedDataMap := structToMap(extractedData)

	_, err = s.pool.Exec(ctx, `
		UPDATE onboarding_sessions
		SET current_step = $1, steps_completed = $2, extracted_data = $3, updated_at = NOW()
		WHERE id = $4
	`, nextStep, stepsCompleted, extractedDataMap, session.ID)
	if err != nil {
		return nil, fmt.Errorf("update session: %w", err)
	}

	// Add agent message
	if agentMessage != "" {
		msg, err := s.AddMessage(ctx, session.ID, "agent", agentMessage, nil, &nextStep)
		if err != nil {
			return nil, err
		}
		response.Message = *msg
	}

	response.NextStep = nextStep
	response.ExtractedData = extractedData

	return response, nil
}
