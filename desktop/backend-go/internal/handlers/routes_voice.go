package handlers

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/middleware"
)

// registerVoiceRoutes wires up transcription and voice routes:
// /api/transcribe, /api/osa (speak endpoints), /api/voice-notes.
func (h *Handlers) registerVoiceRoutes(api *gin.RouterGroup, auth gin.HandlerFunc) {
	// Transcription routes - /api/transcribe
	transcriptionHandler := NewTranscriptionHandler(h.pool)
	transcribe := api.Group("/transcribe")
	transcribe.Use(auth, middleware.RequireAuth())
	{
		transcribe.POST("", transcriptionHandler.TranscribeAudio)
		transcribe.GET("/status", transcriptionHandler.GetTranscriptionStatus)
		transcribe.POST("/realtime", HandleRealtimeTranscription) // Real-time voice transcription for active listening (deprecated)
	}
	slog.Info("Transcription routes registered (including real-time)")

	// OSA Voice routes - /api/osa/speak*
	if h.elevenLabsService != nil {
		voiceH := NewOSAVoiceHandler(h.elevenLabsService)
		osaVoice := api.Group("/osa")
		osaVoice.Use(auth, middleware.RequireAuth())
		{
			osaVoice.POST("/speak", voiceH.HandleOSASpeak)
			osaVoice.POST("/speak/stream", voiceH.HandleOSASpeakStream)
		}
		slog.Info("OSA voice routes registered")
	} else {
		slog.Info("OSA voice routes skipped: elevenLabsService not initialized")
	}

	// Voice notes routes - /api/voice-notes
	voiceNotesHandler := NewVoiceNotesHandler(h.pool, h.embeddingService)
	voiceNotes := api.Group("/voice-notes")
	voiceNotes.Use(auth, middleware.RequireAuth())
	{
		voiceNotes.GET("", voiceNotesHandler.ListVoiceNotes)
		voiceNotes.POST("", voiceNotesHandler.UploadVoiceNote)
		voiceNotes.GET("/stats", voiceNotesHandler.GetVoiceNoteStats)
		voiceNotes.GET("/:id", voiceNotesHandler.GetVoiceNote)
		voiceNotes.DELETE("/:id", voiceNotesHandler.DeleteVoiceNote)
		voiceNotes.POST("/:id/retranscribe", voiceNotesHandler.RetranscribeVoiceNote)
	}
}
