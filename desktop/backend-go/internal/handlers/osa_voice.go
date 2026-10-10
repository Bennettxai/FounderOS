package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
)

// OSASpeakRequest represents a request for OSA to speak
type OSASpeakRequest struct {
	Text string `json:"text" binding:"required"`
}

// OSAVoiceHandler handles OSA text-to-speech endpoints.
type OSAVoiceHandler struct {
	elevenLabsService *services.ElevenLabsService
}

// NewOSAVoiceHandler creates a new OSAVoiceHandler.
func NewOSAVoiceHandler(elevenLabsService *services.ElevenLabsService) *OSAVoiceHandler {
	return &OSAVoiceHandler{elevenLabsService: elevenLabsService}
}

// maxTTSCharacters is the maximum allowed length for TTS text input.
const maxTTSCharacters = 5000

// HandleOSASpeak converts text to speech using ElevenLabs
// POST /api/osa/speak
func (h *OSAVoiceHandler) HandleOSASpeak(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		slog.Error("[OSA Voice] Unauthorized access attempt")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req OSASpeakRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Warn("[OSA Voice] Invalid request", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if len(req.Text) > maxTTSCharacters {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Text exceeds maximum length of %d characters", maxTTSCharacters),
		})
		return
	}

	slog.Info("[OSA Voice] Speech request",
		"user_id", user.ID,
		"text_length", len(req.Text))

	// Convert text to speech
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	audioData, err := h.elevenLabsService.TextToSpeech(ctx, req.Text)
	if err != nil {
		slog.Error("[OSA Voice] TTS failed",
			"user_id", user.ID,
			"error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Text-to-speech failed"})
		return
	}

	slog.Info("[OSA Voice] ✅ Speech generated successfully",
		"user_id", user.ID,
		"audio_size_bytes", len(audioData))

	// Return audio as MP3
	c.Header("Content-Type", "audio/mpeg")
	c.Header("Content-Length", fmt.Sprintf("%d", len(audioData)))
	c.Data(http.StatusOK, "audio/mpeg", audioData)
}

// HandleOSASpeakStream streams TTS audio in chunks (for longer text)
// POST /api/osa/speak/stream
func (h *OSAVoiceHandler) HandleOSASpeakStream(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		slog.Error("[OSA Voice] Unauthorized access attempt")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req OSASpeakRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Warn("[OSA Voice] Invalid request", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if len(req.Text) > maxTTSCharacters {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Text exceeds maximum length of %d characters", maxTTSCharacters),
		})
		return
	}

	slog.Info("[OSA Voice] Streaming speech request",
		"user_id", user.ID,
		"text_length", len(req.Text))

	// Set streaming headers
	c.Header("Content-Type", "audio/mpeg")
	c.Header("Transfer-Encoding", "chunked")
	c.Header("Cache-Control", "no-cache")

	// Stream audio chunks
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	audioChan, errChan := h.elevenLabsService.TextToSpeechStream(ctx, req.Text)

	// Write chunks to response
	w := c.Writer
	flusher, ok := w.(http.Flusher)
	if !ok {
		slog.Error("[OSA Voice] Streaming not supported")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Streaming not supported"})
		return
	}

	totalBytes := 0
	for {
		select {
		case chunk, ok := <-audioChan:
			if !ok {
				// Channel closed, streaming complete
				slog.Info("[OSA Voice] ✅ Streaming complete",
					"user_id", user.ID,
					"total_bytes", totalBytes)
				return
			}

			// Write chunk
			n, err := w.Write(chunk)
			if err != nil {
				slog.Error("[OSA Voice] Write error", "error", err)
				return
			}

			totalBytes += n
			flusher.Flush()

		case err := <-errChan:
			if err != nil {
				slog.Error("[OSA Voice] Streaming error",
					"user_id", user.ID,
					"error", err)
				return
			}

		case <-ctx.Done():
			slog.Warn("[OSA Voice] Streaming timeout", "user_id", user.ID)
			return
		}
	}
}
