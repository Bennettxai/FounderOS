package handlers

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/utils"
)

// VoiceNotesHandler handles voice note storage and transcription
type VoiceNotesHandler struct {
	whisper *services.WhisperService
	pool    *pgxpool.Pool
	emb     *services.EmbeddingService
}

// NewVoiceNotesHandler creates a new voice notes handler
func NewVoiceNotesHandler(pool *pgxpool.Pool, embeddingService *services.EmbeddingService) *VoiceNotesHandler {
	return &VoiceNotesHandler{
		whisper: services.NewWhisperService(),
		pool:    pool,
		emb:     embeddingService,
	}
}

// VoiceNoteResponse represents a voice note
type VoiceNoteResponse struct {
	ID             string  `json:"id"`
	Filename       string  `json:"filename,omitempty"`
	Transcript     string  `json:"transcript"`
	Duration       float64 `json:"duration"`
	WordCount      int     `json:"word_count"`
	WordsPerMinute float64 `json:"words_per_minute"`
	Language       string  `json:"language,omitempty"`
	CreatedAt      string  `json:"created_at"`
	URL            string  `json:"url,omitempty"`
	ContextID      *string `json:"context_id,omitempty"`
	ProjectID      *string `json:"project_id,omitempty"`
}

// UploadVoiceNote handles voice note upload with transcription
func (h *VoiceNotesHandler) UploadVoiceNote(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	// Parse multipart form with 50MB max for audio
	if err := c.Request.ParseMultipartForm(50 << 20); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	file, header, err := c.Request.FormFile("audio")
	if err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}
	defer file.Close()

	// Get optional context_id
	contextID := c.PostForm("context_id")

	// Generate unique ID
	noteID := uuid.New().String()

	// Determine file extension from filename
	ext := filepath.Ext(header.Filename)
	if ext == "" {
		ext = ".webm"
	}

	// Create voice notes directory
	voiceDir := filepath.Join("uploads", "voice_notes", user.ID)
	if err := os.MkdirAll(voiceDir, 0755); err != nil {
		utils.RespondInternalError(c, slog.Default(), "create directory", err)
		return
	}

	// Save the audio file
	filename := noteID + ext
	filePath := filepath.Join(voiceDir, filename)

	out, err := os.Create(filePath)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "create file", err)
		return
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
		utils.RespondInternalError(c, slog.Default(), "save file", err)
		return
	}

	// Get optional project_id
	projectID := c.PostForm("project_id")

	// Transcribe if whisper is available
	var transcript string
	var duration float64
	var language string

	if h.whisper.IsAvailable() {
		result, err := h.whisper.TranscribeFile(c.Request.Context(), filePath)
		if err != nil {
			// Log but don't fail - save note without transcript
			slog.Warn("transcription failed", "error", err)
		} else {
			transcript = result.Text
			duration = result.Duration
			language = result.Language
		}
	}

	// Calculate stats
	wordCount := countWords(transcript)
	durationSeconds := int32(duration)
	var wordsPerMinute float64
	if durationSeconds > 0 {
		wordsPerMinute = float64(wordCount) / float64(durationSeconds) * 60.0
	}

	// Save to database if pool is available
	var dbNoteID string
	if h.pool != nil && transcript != "" {
		queries := sqlc.New(h.pool)

		var ctxUUID, projUUID pgtype.UUID
		if contextID != "" {
			if parsed, err := uuid.Parse(contextID); err == nil {
				ctxUUID = pgtype.UUID{Bytes: parsed, Valid: true}
			}
		}
		if projectID != "" {
			if parsed, err := uuid.Parse(projectID); err == nil {
				projUUID = pgtype.UUID{Bytes: parsed, Valid: true}
			}
		}

		wpmNumeric := pgtype.Numeric{}
		wpmNumeric.Scan(wordsPerMinute)

		voiceNote, err := queries.CreateVoiceNote(c.Request.Context(), sqlc.CreateVoiceNoteParams{
			UserID:          user.ID,
			Transcript:      transcript,
			DurationSeconds: durationSeconds,
			WordCount:       int32(wordCount),
			WordsPerMinute:  wpmNumeric,
			Language:        stringPtrOrNil(language),
			AudioFilePath:   stringPtrOrNil(filePath),
			ContextID:       ctxUUID,
			ProjectID:       projUUID,
		})
		if err == nil {
			dbNoteID = voiceNote.ID.String()

			// Generate and store embedding for semantic search (best-effort)
			if h.emb != nil {
				if emb, err := h.emb.GenerateEmbedding(c.Request.Context(), transcript); err == nil && len(emb) > 0 {
					vec := pgvector.NewVector(emb)
					_, _ = h.pool.Exec(c.Request.Context(), `UPDATE voice_notes SET embedding = $1 WHERE id = $2 AND user_id = $3`, vec, voiceNote.ID, user.ID)
				}
			}
		} else {
			slog.Warn("failed to save voice note to database", "error", err)
		}
	}

	// Use database ID if available, otherwise use file ID
	responseID := noteID
	if dbNoteID != "" {
		responseID = dbNoteID
	}

	// Build response
	response := VoiceNoteResponse{
		ID:             responseID,
		Filename:       filename,
		Transcript:     transcript,
		Duration:       duration,
		WordCount:      wordCount,
		WordsPerMinute: wordsPerMinute,
		Language:       language,
		CreatedAt:      time.Now().UTC().Format(time.RFC3339),
		URL:            fmt.Sprintf("/api/voice-notes/%s", responseID),
	}

	if contextID != "" {
		response.ContextID = &contextID
	}
	if projectID != "" {
		response.ProjectID = &projectID
	}

	c.JSON(http.StatusOK, response)
}

// stringPtrOrNil returns a pointer to the string if non-empty, otherwise nil
func stringPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// GetVoiceNote serves a voice note audio file
func (h *VoiceNotesHandler) GetVoiceNote(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	noteID := c.Param("id")
	if noteID == "" {
		utils.RespondInvalidID(c, slog.Default(), "note ID")
		return
	}

	// Look for the file with any extension
	voiceDir := filepath.Join("uploads", "voice_notes", user.ID)
	entries, err := os.ReadDir(voiceDir)
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Voice note")
		return
	}

	var filePath string
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), noteID) {
			filePath = filepath.Join(voiceDir, entry.Name())
			break
		}
	}

	if filePath == "" {
		utils.RespondNotFound(c, slog.Default(), "Voice note")
		return
	}

	// Determine content type
	ext := filepath.Ext(filePath)
	contentType := "audio/webm"
	switch ext {
	case ".wav":
		contentType = "audio/wav"
	case ".mp3":
		contentType = "audio/mpeg"
	case ".ogg":
		contentType = "audio/ogg"
	case ".m4a":
		contentType = "audio/mp4"
	}

	c.Header("Content-Type", contentType)
	c.File(filePath)
}

// DeleteVoiceNote deletes a voice note
func (h *VoiceNotesHandler) DeleteVoiceNote(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	noteID := c.Param("id")
	if noteID == "" {
		utils.RespondInvalidID(c, slog.Default(), "note ID")
		return
	}

	// Look for the file with any extension
	voiceDir := filepath.Join("uploads", "voice_notes", user.ID)
	entries, err := os.ReadDir(voiceDir)
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Voice note")
		return
	}

	var filePath string
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), noteID) {
			filePath = filepath.Join(voiceDir, entry.Name())
			break
		}
	}

	if filePath == "" {
		utils.RespondNotFound(c, slog.Default(), "Voice note")
		return
	}

	if err := os.Remove(filePath); err != nil {
		utils.RespondInternalError(c, slog.Default(), "delete voice note", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Voice note deleted"})
}

// ListVoiceNotes lists all voice notes for the user from database
func (h *VoiceNotesHandler) ListVoiceNotes(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	// Optional filters
	contextID := c.Query("context_id")
	projectID := c.Query("project_id")

	// Use database if available
	if h.pool != nil {
		queries := sqlc.New(h.pool)

		var notes []sqlc.VoiceNote
		var err error

		if contextID != "" {
			if ctxUUID, parseErr := uuid.Parse(contextID); parseErr == nil {
				notes, err = queries.ListVoiceNotesByContext(c.Request.Context(), sqlc.ListVoiceNotesByContextParams{
					UserID:    user.ID,
					ContextID: pgtype.UUID{Bytes: ctxUUID, Valid: true},
				})
			}
		} else if projectID != "" {
			if projUUID, parseErr := uuid.Parse(projectID); parseErr == nil {
				notes, err = queries.ListVoiceNotesByProject(c.Request.Context(), sqlc.ListVoiceNotesByProjectParams{
					UserID:    user.ID,
					ProjectID: pgtype.UUID{Bytes: projUUID, Valid: true},
				})
			}
		} else {
			notes, err = queries.ListVoiceNotes(c.Request.Context(), sqlc.ListVoiceNotesParams{
				UserID: user.ID,
				Limit:  100,
				Offset: 0,
			})
		}

		if err != nil {
			utils.RespondInternalError(c, slog.Default(), "fetch voice notes", err)
			return
		}

		var response []VoiceNoteResponse
		for _, note := range notes {
			resp := VoiceNoteResponse{
				ID:             note.ID.String(),
				Transcript:     note.Transcript,
				Duration:       float64(note.DurationSeconds),
				WordCount:      int(note.WordCount),
				WordsPerMinute: pgtypeNumericToFloat64(note.WordsPerMinute),
				CreatedAt:      note.CreatedAt.Time.UTC().Format(time.RFC3339),
			}
			if note.Language != nil {
				resp.Language = *note.Language
			}
			if note.AudioFilePath != nil {
				resp.URL = fmt.Sprintf("/api/voice-notes/%s/audio", note.ID.String())
			}
			if note.ContextID.Valid {
				cid := note.ContextID.Bytes
				cidStr := uuid.UUID(cid).String()
				resp.ContextID = &cidStr
			}
			if note.ProjectID.Valid {
				pid := note.ProjectID.Bytes
				pidStr := uuid.UUID(pid).String()
				resp.ProjectID = &pidStr
			}
			response = append(response, resp)
		}

		c.JSON(http.StatusOK, response)
		return
	}

	// Fallback to file-based listing
	voiceDir := filepath.Join("uploads", "voice_notes", user.ID)
	entries, err := os.ReadDir(voiceDir)
	if err != nil {
		c.JSON(http.StatusOK, []VoiceNoteResponse{})
		return
	}

	var notes []VoiceNoteResponse
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		ext := filepath.Ext(name)
		noteID := strings.TrimSuffix(name, ext)

		info, err := entry.Info()
		if err != nil {
			continue
		}

		note := VoiceNoteResponse{
			ID:        noteID,
			Filename:  name,
			CreatedAt: info.ModTime().UTC().Format(time.RFC3339),
			URL:       fmt.Sprintf("/api/voice-notes/%s", noteID),
		}

		notes = append(notes, note)
	}

	c.JSON(http.StatusOK, notes)
}

// GetVoiceNoteStats returns aggregate stats for the user's voice notes
func (h *VoiceNotesHandler) GetVoiceNoteStats(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	if h.pool == nil {
		utils.RespondInternalError(c, slog.Default(), "database connection", nil)
		return
	}

	queries := sqlc.New(h.pool)
	stats, err := queries.GetVoiceNoteStats(c.Request.Context(), user.ID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "fetch stats", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total_notes":            stats.TotalNotes,
		"total_duration_seconds": stats.TotalDurationSeconds,
		"total_words":            stats.TotalWords,
		"avg_words_per_minute":   pgtypeNumericToFloat64(stats.AvgWordsPerMinute),
	})
}

// pgtypeNumericToFloat64 converts pgtype.Numeric to float64
func pgtypeNumericToFloat64(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	f, _ := n.Float64Value()
	return f.Float64
}

// RetranscribeVoiceNote re-transcribes an existing voice note
func (h *VoiceNotesHandler) RetranscribeVoiceNote(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	if !h.whisper.IsAvailable() {
		utils.RespondInternalError(c, slog.Default(), "transcription service", nil)
		return
	}

	noteID := c.Param("id")
	if noteID == "" {
		utils.RespondInvalidID(c, slog.Default(), "note ID")
		return
	}

	// Find the file
	voiceDir := filepath.Join("uploads", "voice_notes", user.ID)
	entries, err := os.ReadDir(voiceDir)
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Voice note")
		return
	}

	var filePath string
	var filename string
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), noteID) {
			filePath = filepath.Join(voiceDir, entry.Name())
			filename = entry.Name()
			break
		}
	}

	if filePath == "" {
		utils.RespondNotFound(c, slog.Default(), "Voice note")
		return
	}

	// Transcribe
	result, err := h.whisper.TranscribeFile(c.Request.Context(), filePath)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "transcribe file", err)
		return
	}

	c.JSON(http.StatusOK, VoiceNoteResponse{
		ID:         noteID,
		Filename:   filename,
		Transcript: result.Text,
		Duration:   result.Duration,
		URL:        fmt.Sprintf("/api/voice-notes/%s", noteID),
	})
}
