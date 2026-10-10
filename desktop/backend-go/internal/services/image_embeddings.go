package services

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ImageEmbeddingService handles generating embeddings for images using CLIP models
type ImageEmbeddingService struct {
	pool              *pgxpool.Pool
	httpClient        *http.Client
	provider          string // "openai", "replicate", "local"
	apiKey            string
	modelName         string
	dimensions        int
	localBaseURL      string // For local CLIP server
	embeddingCache    *EmbeddingCacheAdapter // Cache for image embeddings
}

// ImageEmbeddingConfig configures the image embedding service
type ImageEmbeddingConfig struct {
	Provider     string // "openai", "replicate", "local"
	APIKey       string
	ModelName    string // e.g., "clip-vit-base-patch32"
	Dimensions   int    // Default: 512 for CLIP
	LocalBaseURL string // e.g., "http://localhost:8000"
	Timeout      time.Duration
}

// ImageEmbeddingResult represents an image with its embedding
type ImageEmbeddingResult struct {
	ID           uuid.UUID
	UserID       string
	ImageURL     string
	ImageData    []byte
	Embedding    []float32
	Caption      string
	Metadata     map[string]interface{}
	ContextID    *uuid.UUID
	ProjectID    *uuid.UUID
	CreatedAt    time.Time
}

// NewImageEmbeddingService creates a new image embedding service
func NewImageEmbeddingService(pool *pgxpool.Pool, config ImageEmbeddingConfig) *ImageEmbeddingService {
	if config.Dimensions == 0 {
		config.Dimensions = 512 // CLIP default
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.ModelName == "" {
		config.ModelName = "clip-vit-base-patch32"
	}

	return &ImageEmbeddingService{
		pool:         pool,
		httpClient:   &http.Client{Timeout: config.Timeout},
		provider:     config.Provider,
		apiKey:       config.APIKey,
		modelName:    config.ModelName,
		dimensions:   config.Dimensions,
		localBaseURL: config.LocalBaseURL,
	}
}

// SetEmbeddingCache sets the embedding cache for image embeddings
func (s *ImageEmbeddingService) SetEmbeddingCache(cache *EmbeddingCacheAdapter) {
	s.embeddingCache = cache
}

// GenerateEmbedding generates an embedding for an image
func (s *ImageEmbeddingService) GenerateEmbedding(ctx context.Context, imageData []byte) ([]float32, error) {
	// Generate a cache key from the image data
	// We'll use the base64 representation as the content key
	cacheKey := base64.StdEncoding.EncodeToString(imageData)

	// Check cache first (if available)
	if s.embeddingCache != nil {
		if cached, found, err := s.embeddingCache.GetEmbedding(ctx, cacheKey, "image"); err == nil && found {
			return cached, nil
		}
	}

	// Generate embedding based on provider
	var embedding []float32
	var err error

	switch s.provider {
	case "openai":
		embedding, err = s.generateOpenAIEmbedding(ctx, imageData)
	case "replicate":
		embedding, err = s.generateReplicateEmbedding(ctx, imageData)
	case "local":
		embedding, err = s.generateLocalEmbedding(ctx, imageData)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", s.provider)
	}

	if err != nil {
		return nil, err
	}

	// Cache the result with 48h TTL for images
	if s.embeddingCache != nil && embedding != nil {
		_ = s.embeddingCache.SetEmbedding(ctx, cacheKey, embedding, "image", 48*time.Hour)
	}

	return embedding, nil
}

// generateOpenAIEmbedding uses OpenAI's CLIP model
func (s *ImageEmbeddingService) generateOpenAIEmbedding(ctx context.Context, imageData []byte) ([]float32, error) {
	// OpenAI doesn't have direct image embedding API yet, but we can use their vision models
	// For now, this is a placeholder for when they add it
	// Alternative: Use Azure OpenAI or wait for official support

	base64Image := base64.StdEncoding.EncodeToString(imageData)

	reqBody := map[string]interface{}{
		"model": "clip-vit-large-patch14", // Hypothetical model name
		"input": base64Image,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/embeddings", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Data) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}

	return result.Data[0].Embedding, nil
}

// generateReplicateEmbedding uses Replicate's CLIP models
func (s *ImageEmbeddingService) generateReplicateEmbedding(ctx context.Context, imageData []byte) ([]float32, error) {
	base64Image := base64.StdEncoding.EncodeToString(imageData)

	reqBody := map[string]interface{}{
		"version": "latest", // Or specific version hash
		"input": map[string]interface{}{
			"image": "data:image/jpeg;base64," + base64Image,
			"task":  "embedding",
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Replicate uses a different API pattern - create prediction first
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.replicate.com/v1/predictions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token "+s.apiKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var prediction struct {
		ID     string                 `json:"id"`
		Status string                 `json:"status"`
		Output map[string]interface{} `json:"output"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&prediction); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Poll for completion if needed
	if prediction.Status == "starting" || prediction.Status == "processing" {
		prediction, err = s.pollReplicatePrediction(ctx, prediction.ID)
		if err != nil {
			return nil, err
		}
	}

	// Extract embedding from output
	if embeddingRaw, ok := prediction.Output["embedding"]; ok {
		if embeddingSlice, ok := embeddingRaw.([]interface{}); ok {
			embedding := make([]float32, len(embeddingSlice))
			for i, v := range embeddingSlice {
				if f, ok := v.(float64); ok {
					embedding[i] = float32(f)
				}
			}
			return embedding, nil
		}
	}

	return nil, fmt.Errorf("failed to extract embedding from response")
}

// generateLocalEmbedding uses a local CLIP server
func (s *ImageEmbeddingService) generateLocalEmbedding(ctx context.Context, imageData []byte) ([]float32, error) {
	base64Image := base64.StdEncoding.EncodeToString(imageData)

	reqBody := map[string]interface{}{
		"image": base64Image,
		"model": s.modelName,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/embed/image", s.localBaseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("local server error (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Embedding []float32 `json:"embedding"`
		Model     string    `json:"model"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Embedding, nil
}

// pollReplicatePrediction polls Replicate API until prediction completes
func (s *ImageEmbeddingService) pollReplicatePrediction(ctx context.Context, predictionID string) (struct {
	ID     string                 `json:"id"`
	Status string                 `json:"status"`
	Output map[string]interface{} `json:"output"`
}, error) {
	var prediction struct {
		ID     string                 `json:"id"`
		Status string                 `json:"status"`
		Output map[string]interface{} `json:"output"`
	}

	url := fmt.Sprintf("https://api.replicate.com/v1/predictions/%s", predictionID)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	timeout := time.After(30 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return prediction, ctx.Err()
		case <-timeout:
			return prediction, fmt.Errorf("polling timeout")
		case <-ticker.C:
			req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
			if err != nil {
				return prediction, err
			}
			req.Header.Set("Authorization", "Token "+s.apiKey)

			resp, err := s.httpClient.Do(req)
			if err != nil {
				continue // Retry
			}

			if resp.StatusCode == http.StatusOK {
				if err := json.NewDecoder(resp.Body).Decode(&prediction); err != nil {
					resp.Body.Close()
					return prediction, err
				}
				resp.Body.Close()

				if prediction.Status == "succeeded" {
					return prediction, nil
				} else if prediction.Status == "failed" || prediction.Status == "canceled" {
					return prediction, fmt.Errorf("prediction failed: %s", prediction.Status)
				}
			}
			resp.Body.Close()
		}
	}
}

// StoreImageEmbedding stores an image and its embedding in the database
func (s *ImageEmbeddingService) StoreImageEmbedding(ctx context.Context, userID string, imageData []byte, metadata map[string]interface{}) (*ImageEmbeddingResult, error) {
	// Generate embedding
	embedding, err := s.GenerateEmbedding(ctx, imageData)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Store in database
	id := uuid.New()

	metadataJSON, _ := json.Marshal(metadata)

	query := `
		INSERT INTO image_embeddings (
			id, user_id, image_data, embedding, metadata, created_at
		) VALUES ($1, $2, $3, $4, $5, NOW())
		RETURNING id, user_id, image_data, embedding, metadata, created_at
	`

	var result ImageEmbeddingResult
	var embeddingPg []float64

	// Convert []float32 to []float64 for PostgreSQL
	embeddingPg = make([]float64, len(embedding))
	for i, v := range embedding {
		embeddingPg[i] = float64(v)
	}

	err = s.pool.QueryRow(ctx, query, id, userID, imageData, embeddingPg, metadataJSON).Scan(
		&result.ID,
		&result.UserID,
		&result.ImageData,
		&embeddingPg,
		&metadataJSON,
		&result.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to store image embedding: %w", err)
	}

	// Convert back to []float32
	result.Embedding = make([]float32, len(embeddingPg))
	for i, v := range embeddingPg {
		result.Embedding[i] = float32(v)
	}

	return &result, nil
}

// SearchSimilarImages finds images similar to the given image
func (s *ImageEmbeddingService) SearchSimilarImages(ctx context.Context, imageData []byte, userID string, maxResults int) ([]ImageEmbeddingResult, error) {
	// Generate embedding for query image
	queryEmbedding, err := s.GenerateEmbedding(ctx, imageData)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Convert to []float64 for PostgreSQL
	embeddingPg := make([]float64, len(queryEmbedding))
	for i, v := range queryEmbedding {
		embeddingPg[i] = float64(v)
	}

	// Search for similar images using cosine similarity
	query := `
		SELECT
			id, user_id, image_data, embedding, metadata, created_at,
			1 - (embedding <=> $1::vector) as similarity
		FROM image_embeddings
		WHERE user_id = $2
		ORDER BY embedding <=> $1::vector
		LIMIT $3
	`

	rows, err := s.pool.Query(ctx, query, embeddingPg, userID, maxResults)
	if err != nil {
		return nil, fmt.Errorf("failed to search images: %w", err)
	}
	defer rows.Close()

	var results []ImageEmbeddingResult
	for rows.Next() {
		var result ImageEmbeddingResult
		var embeddingPg []float64
		var metadataJSON []byte
		var similarity float64

		err := rows.Scan(
			&result.ID,
			&result.UserID,
			&result.ImageData,
			&embeddingPg,
			&metadataJSON,
			&result.CreatedAt,
			&similarity,
		)
		if err != nil {
			continue
		}

		// Convert embedding
		result.Embedding = make([]float32, len(embeddingPg))
		for i, v := range embeddingPg {
			result.Embedding[i] = float32(v)
		}

		// Parse metadata
		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &result.Metadata)
		}

		results = append(results, result)
	}

	return results, nil
}

// GetImageEmbedding retrieves an image embedding by ID
func (s *ImageEmbeddingService) GetImageEmbedding(ctx context.Context, id uuid.UUID) (*ImageEmbeddingResult, error) {
	query := `
		SELECT id, user_id, image_data, embedding, metadata, created_at
		FROM image_embeddings
		WHERE id = $1
	`

	var result ImageEmbeddingResult
	var embeddingPg []float64
	var metadataJSON []byte

	err := s.pool.QueryRow(ctx, query, id).Scan(
		&result.ID,
		&result.UserID,
		&result.ImageData,
		&embeddingPg,
		&metadataJSON,
		&result.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get image embedding: %w", err)
	}

	// Convert embedding
	result.Embedding = make([]float32, len(embeddingPg))
	for i, v := range embeddingPg {
		result.Embedding[i] = float32(v)
	}

	// Parse metadata
	if len(metadataJSON) > 0 {
		json.Unmarshal(metadataJSON, &result.Metadata)
	}

	return &result, nil
}

// DeleteImageEmbedding deletes an image embedding
func (s *ImageEmbeddingService) DeleteImageEmbedding(ctx context.Context, id uuid.UUID, userID string) error {
	query := `DELETE FROM image_embeddings WHERE id = $1 AND user_id = $2`

	result, err := s.pool.Exec(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete image embedding: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("image embedding not found")
	}

	return nil
}
