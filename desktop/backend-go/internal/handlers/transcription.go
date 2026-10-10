package handlers

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/utils"
)

// TranscriptionHandler handles audio transcription requests
type TranscriptionHandler struct {
	whisper *services.WhisperService
	pool    *pgxpool.Pool
}

// NewTranscriptionHandler creates a new transcription handler
func NewTranscriptionHandler(pool *pgxpool.Pool) *TranscriptionHandler {
	return &TranscriptionHandler{
		whisper: services.NewWhisperService(),
		pool:    pool,
	}
}

// maxAudioUploadSize is the maximum allowed audio upload size (25 MB).
// Enforced via http.MaxBytesReader to prevent OOM from oversized files.
const maxAudioUploadSize = 25 << 20 // 25 MB

// TranscribeAudio handles audio transcription via multipart form
func (t *TranscriptionHandler) TranscribeAudio(c *gin.Context) {
	// Check if whisper is available
	if !t.whisper.IsAvailable() {
		utils.RespondInternalError(c, slog.Default(), "transcription service", nil)
		return
	}

	// Enforce upload size limit before parsing the multipart body.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxAudioUploadSize)

	// Get audio file from multipart form
	file, header, err := c.Request.FormFile("audio")
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": "Audio file exceeds maximum allowed size of 25 MB",
			})
			return
		}
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}
	defer file.Close()

	// Read audio data
	audioData, err := io.ReadAll(file)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": "Audio file exceeds maximum allowed size of 25 MB",
			})
			return
		}
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	// Determine format from filename or content type
	format := "webm"
	if header != nil && header.Filename != "" {
		parts := strings.Split(header.Filename, ".")
		if len(parts) > 1 {
			format = parts[len(parts)-1]
		}
	}
	contentType := c.ContentType()
	if strings.Contains(contentType, "wav") {
		format = "wav"
	} else if strings.Contains(contentType, "mp3") {
		format = "mp3"
	} else if strings.Contains(contentType, "ogg") {
		format = "ogg"
	}

	// Transcribe
	result, err := t.whisper.Transcribe(c.Request.Context(), bytes.NewReader(audioData), format)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "transcribe audio", err)
		return
	}

	// Calculate stats
	wordCount := countWords(result.Text)
	durationSeconds := int32(result.Duration)
	var wordsPerMinute float64
	if durationSeconds > 0 {
		wordsPerMinute = float64(wordCount) / float64(durationSeconds) * 60.0
	}

	response := gin.H{
		"text":             result.Text,
		"duration":         result.Duration,
		"language":         result.Language,
		"word_count":       wordCount,
		"words_per_minute": wordsPerMinute,
	}

	// Check if user wants to save the voice note
	saveNote := c.Query("save") == "true"
	user := middleware.GetCurrentUser(c)

	if saveNote && user != nil && t.pool != nil && result.Text != "" {
		queries := sqlc.New(t.pool)

		// Parse optional context/project IDs
		var contextID, projectID, conversationID pgtype.UUID
		if cid := c.Query("context_id"); cid != "" {
			if parsed, err := uuid.Parse(cid); err == nil {
				contextID = pgtype.UUID{Bytes: parsed, Valid: true}
			}
		}
		if pid := c.Query("project_id"); pid != "" {
			if parsed, err := uuid.Parse(pid); err == nil {
				projectID = pgtype.UUID{Bytes: parsed, Valid: true}
			}
		}
		if convID := c.Query("conversation_id"); convID != "" {
			if parsed, err := uuid.Parse(convID); err == nil {
				conversationID = pgtype.UUID{Bytes: parsed, Valid: true}
			}
		}

		// Convert WPM to pgtype.Numeric
		wpmNumeric := pgtype.Numeric{}
		wpmNumeric.Scan(wordsPerMinute)

		// Create voice note
		voiceNote, err := queries.CreateVoiceNote(c.Request.Context(), sqlc.CreateVoiceNoteParams{
			UserID:          user.ID,
			Transcript:      result.Text,
			DurationSeconds: durationSeconds,
			WordCount:       int32(wordCount),
			WordsPerMinute:  wpmNumeric,
			Language:        utils.StringPtr(result.Language),
			ContextID:       contextID,
			ProjectID:       projectID,
			ConversationID:  conversationID,
		})
		if err == nil {
			response["voice_note_id"] = voiceNote.ID.String()
		}
	}

	c.JSON(http.StatusOK, response)
}

// countWords counts words in text (split by whitespace)
func countWords(text string) int {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0
	}
	return len(strings.Fields(text))
}

// GetTranscriptionStatus returns the current transcription service status
func (t *TranscriptionHandler) GetTranscriptionStatus(c *gin.Context) {
	status := t.whisper.GetStatus()
	c.JSON(http.StatusOK, status)
}
