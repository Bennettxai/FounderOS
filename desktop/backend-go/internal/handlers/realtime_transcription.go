package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ============================================================================
// ⚠️ DEPRECATED: This endpoint is no longer used
// ============================================================================
// The voice transcription system now uses Deepgram API directly from the
// frontend via WebSocket connection. This provides:
// - Sub-300ms latency (vs 500-2000ms with local Whisper)
// - No FFmpeg dependency
// - No Whisper binary/model downloads
// - Better accuracy (~5% WER)
// - Zero backend processing
//
// See: /docs/DEEPGRAM_SETUP.md for the new implementation
// ============================================================================

// RealtimeTranscriptionRequest represents an audio chunk for transcription
type RealtimeTranscriptionRequest struct {
	SessionID string `form:"session_id"`
}

// RealtimeTranscriptionResponse represents the transcription result
type RealtimeTranscriptionResponse struct {
	Text     string `json:"text"`
	Language string `json:"language,omitempty"`
	IsFinal  bool   `json:"is_final"`
}

// HandleRealtimeTranscription - DEPRECATED: Returns error directing to Deepgram
// POST /api/transcribe/realtime
func HandleRealtimeTranscription(c *gin.Context) {
	slog.Warn("[Realtime Transcription] DEPRECATED endpoint called - use Deepgram WebSocket instead")

	// Return 410 Gone with migration information
	c.JSON(http.StatusGone, gin.H{
		"error":   "This endpoint is deprecated",
		"message": "Voice transcription now uses Deepgram API directly from the frontend",
		"details": "See /docs/DEEPGRAM_SETUP.md for setup instructions",
		"migration": map[string]string{
			"old_approach":        "Microphone → Backend → FFmpeg → Whisper",
			"new_approach":        "Microphone → Deepgram WebSocket (direct)",
			"latency_improvement": "500-2000ms → <300ms",
			"setup":               "Add VITE_DEEPGRAM_API_KEY to frontend .env",
			"documentation":       "/docs/DEEPGRAM_SETUP.md",
			"free_credits":        "$200 from Deepgram",
		},
	})
}
