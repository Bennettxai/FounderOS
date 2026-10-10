package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"
)

// ElevenLabsService handles text-to-speech via ElevenLabs API
type ElevenLabsService struct {
	apiKey  string
	voiceID string
	model   string
	client  *http.Client
}

// NewElevenLabsService creates a new ElevenLabs service
func NewElevenLabsService() *ElevenLabsService {
	apiKey := os.Getenv("ELEVENLABS_API_KEY")
	voiceID := os.Getenv("ELEVENLABS_VOICE_ID")
	model := os.Getenv("ELEVENLABS_MODEL")

	if model == "" {
		model = "eleven_multilingual_v2" // Default model
	}

	if apiKey == "" {
		slog.Info("[ElevenLabs] API key not configured")
	}

	if voiceID == "" {
		slog.Info("[ElevenLabs] Voice ID not configured")
	}

	return &ElevenLabsService{
		apiKey:  apiKey,
		voiceID: voiceID,
		model:   model,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// TextToSpeechRequest represents the request payload for ElevenLabs TTS
type TextToSpeechRequest struct {
	Text          string                 `json:"text"`
	ModelID       string                 `json:"model_id"`
	VoiceSettings map[string]interface{} `json:"voice_settings,omitempty"`
}

// TextToSpeech converts text to speech audio
// Returns audio data as []byte (MP3 format)
func (s *ElevenLabsService) TextToSpeech(ctx context.Context, text string) ([]byte, error) {
	if s.apiKey == "" {
		return nil, fmt.Errorf("ElevenLabs API key not configured")
	}

	if s.voiceID == "" {
		return nil, fmt.Errorf("ElevenLabs voice ID not configured")
	}

	// Prepare request payload
	payload := TextToSpeechRequest{
		Text:    text,
		ModelID: s.model,
		VoiceSettings: map[string]interface{}{
			"stability":        0.5,
			"similarity_boost": 0.75,
			"style":            0.0,
			"use_speaker_boost": true,
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("https://api.elevenlabs.io/v1/text-to-speech/%s", s.voiceID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("xi-api-key", s.apiKey)

	slog.Info("[ElevenLabs] Requesting TTS", "text_length", len(text), "model", s.model)

	// Execute request
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		slog.Error("[ElevenLabs] API error", "status", resp.StatusCode, "body", string(body))
		return nil, fmt.Errorf("ElevenLabs API error: %d - %s", resp.StatusCode, string(body))
	}

	// Read audio data
	audioData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	slog.Info("[ElevenLabs] ✅ TTS successful", "audio_size_bytes", len(audioData))

	return audioData, nil
}

// TextToSpeechStream streams audio in chunks (for large text)
// Returns a channel that emits audio chunks
func (s *ElevenLabsService) TextToSpeechStream(ctx context.Context, text string) (<-chan []byte, <-chan error) {
	audioChan := make(chan []byte, 10)
	errChan := make(chan error, 1)

	go func() {
		defer close(audioChan)
		defer close(errChan)

		if s.apiKey == "" {
			errChan <- fmt.Errorf("ElevenLabs API key not configured")
			return
		}

		if s.voiceID == "" {
			errChan <- fmt.Errorf("ElevenLabs voice ID not configured")
			return
		}

		// Prepare request payload with streaming enabled
		payload := map[string]interface{}{
			"text":     text,
			"model_id": s.model,
			"voice_settings": map[string]interface{}{
				"stability":         0.5,
				"similarity_boost":  0.75,
				"style":             0.0,
				"use_speaker_boost": true,
			},
		}

		jsonData, err := json.Marshal(payload)
		if err != nil {
			errChan <- fmt.Errorf("failed to marshal request: %w", err)
			return
		}

		// Create HTTP request with streaming
		url := fmt.Sprintf("https://api.elevenlabs.io/v1/text-to-speech/%s/stream", s.voiceID)
		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
		if err != nil {
			errChan <- fmt.Errorf("failed to create request: %w", err)
			return
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("xi-api-key", s.apiKey)

		slog.Info("[ElevenLabs] Requesting streaming TTS", "text_length", len(text))

		// Execute request
		resp, err := s.client.Do(req)
		if err != nil {
			errChan <- fmt.Errorf("failed to execute request: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			slog.Error("[ElevenLabs] Stream API error", "status", resp.StatusCode, "body", string(body))
			errChan <- fmt.Errorf("ElevenLabs API error: %d", resp.StatusCode)
			return
		}

		// Stream audio in chunks
		buffer := make([]byte, 4096)
		for {
			n, err := resp.Body.Read(buffer)
			if n > 0 {
				chunk := make([]byte, n)
				copy(chunk, buffer[:n])
				audioChan <- chunk
			}

			if err == io.EOF {
				break
			}

			if err != nil {
				errChan <- fmt.Errorf("stream read error: %w", err)
				return
			}
		}

		slog.Info("[ElevenLabs] ✅ Streaming TTS complete")
	}()

	return audioChan, errChan
}

// IsConfigured checks if the service is properly configured
func (s *ElevenLabsService) IsConfigured() bool {
	return s.apiKey != "" && s.voiceID != ""
}
