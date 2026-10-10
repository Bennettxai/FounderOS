package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// WhisperService handles audio transcription using local whisper.cpp
type WhisperService struct {
	modelPath   string
	whisperPath string
}

// TranscriptionResult contains the transcription output
type TranscriptionResult struct {
	Text     string  `json:"text"`
	Language string  `json:"language,omitempty"`
	Duration float64 `json:"duration,omitempty"`
}

// NewWhisperService creates a new whisper transcription service
func NewWhisperService() *WhisperService {
	// Try to find whisper binary in common locations
	whisperPath := findWhisperBinary()
	modelPath := findWhisperModel()

	// Log what we found
	slog.Info("whisper service init", "binary", whisperPath, "model", modelPath)

	return &WhisperService{
		whisperPath: whisperPath,
		modelPath:   modelPath,
	}
}

// findWhisperBinary looks for whisper binary in common locations
func findWhisperBinary() string {
	// Check common paths - whisper-cli is the homebrew binary name
	paths := []string{
		"/opt/homebrew/bin/whisper-cli",
		"/usr/local/bin/whisper-cli",
		"whisper-cli", // In PATH
		"/usr/local/bin/whisper",
		"/opt/homebrew/bin/whisper",
		"/usr/bin/whisper",
		"whisper", // In PATH
	}

	// Also check for whisper.cpp main binary
	whisperCppPaths := []string{
		"/usr/local/bin/whisper-cpp",
		"/opt/homebrew/bin/whisper-cpp",
		filepath.Join(os.Getenv("HOME"), ".local/bin/whisper"),
		filepath.Join(os.Getenv("HOME"), "whisper.cpp/main"),
	}
	paths = append(paths, whisperCppPaths...)

	for _, p := range paths {
		if _, err := exec.LookPath(p); err == nil {
			return p
		}
		// Check if file exists directly
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	return ""
}

// findWhisperModel looks for whisper model in common locations
func findWhisperModel() string {
	// Check common model locations
	homeDir := os.Getenv("HOME")
	modelPaths := []string{
		filepath.Join(homeDir, "Library/Application Support/com.osvoice.desktop/transcription-models/ggml-tiny.bin"),
		filepath.Join(homeDir, "Library/Application Support/com.osvoice.desktop/transcription-models/ggml-base.bin"),
		filepath.Join(homeDir, "Library/Application Support/com.osvoice.desktop/transcription-models/ggml-large-v3-turbo.bin"),
		filepath.Join(homeDir, ".cache/whisper/ggml-base.bin"),
		filepath.Join(homeDir, ".cache/whisper/ggml-small.bin"),
		filepath.Join(homeDir, ".cache/whisper/ggml-medium.bin"),
		filepath.Join(homeDir, "whisper.cpp/models/ggml-base.en.bin"),
		filepath.Join(homeDir, "whisper.cpp/models/ggml-base.bin"),
		"/usr/local/share/whisper/ggml-base.bin",
	}

	for _, p := range modelPaths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	return ""
}

// IsAvailable checks if whisper is installed and configured
func (w *WhisperService) IsAvailable() bool {
	return w.whisperPath != "" && w.modelPath != ""
}

// GetStatus returns the current whisper configuration status
func (w *WhisperService) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"available":    w.IsAvailable(),
		"whisper_path": w.whisperPath,
		"model_path":   w.modelPath,
	}
}

// Transcribe transcribes audio data to text
func (w *WhisperService) Transcribe(ctx context.Context, audioData io.Reader, format string) (*TranscriptionResult, error) {
	if !w.IsAvailable() {
		return nil, fmt.Errorf("whisper is not configured. Install whisper.cpp and download a model")
	}

	// Create temp file for audio
	tmpDir := os.TempDir()
	audioFile, err := os.CreateTemp(tmpDir, "whisper-audio-*."+format)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(audioFile.Name())

	// Write audio data to temp file
	if _, err := io.Copy(audioFile, audioData); err != nil {
		return nil, fmt.Errorf("failed to write audio data: %w", err)
	}
	audioFile.Close()

	// Convert to WAV if not already in a supported format
	// whisper-cli supports: flac, mp3, ogg, wav
	inputFile := audioFile.Name()
	supportedFormats := map[string]bool{"wav": true, "mp3": true, "ogg": true, "flac": true}

	if !supportedFormats[strings.ToLower(format)] {
		// Need to convert using ffmpeg
		wavFile := strings.TrimSuffix(audioFile.Name(), "."+format) + ".wav"
		defer os.Remove(wavFile)

		ffmpegArgs := []string{
			"-i", audioFile.Name(),
			"-ar", "16000", // 16kHz sample rate (optimal for whisper)
			"-ac", "1", // Mono
			"-y", // Overwrite
			wavFile,
		}

		ffmpegCmd := exec.CommandContext(ctx, "ffmpeg", ffmpegArgs...)
		var ffmpegErr bytes.Buffer
		ffmpegCmd.Stderr = &ffmpegErr

		if err := ffmpegCmd.Run(); err != nil {
			return nil, fmt.Errorf("failed to convert audio with ffmpeg: %w, stderr: %s", err, ffmpegErr.String())
		}

		inputFile = wavFile
	}

	// Output file for transcription
	outputFile := inputFile + ".txt"
	defer os.Remove(outputFile)

	// Build whisper command
	// whisper.cpp main: ./main -m models/ggml-base.bin -f audio.wav
	args := []string{
		"-m", w.modelPath,
		"-f", inputFile,
		"-otxt",                               // Output as text
		"-of", outputFile[:len(outputFile)-4], // Output file without extension
	}

	// Create command with timeout
	cmdCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, w.whisperPath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("whisper failed: %w, stderr: %s", err, stderr.String())
	}
	duration := time.Since(start).Seconds()

	// Read output file
	transcription, err := os.ReadFile(outputFile)
	if err != nil {
		// Try reading from stdout as fallback
		transcription = stdout.Bytes()
	}

	return &TranscriptionResult{
		Text:     strings.TrimSpace(string(transcription)),
		Duration: duration,
	}, nil
}

// TranscribeFile transcribes an audio file
func (w *WhisperService) TranscribeFile(ctx context.Context, filePath string) (*TranscriptionResult, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Determine format from extension
	ext := strings.TrimPrefix(filepath.Ext(filePath), ".")
	if ext == "" {
		ext = "wav"
	}

	return w.Transcribe(ctx, file, ext)
}
