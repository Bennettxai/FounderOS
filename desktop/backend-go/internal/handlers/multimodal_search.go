package handlers

import (
	"encoding/base64"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rhl/businessos-backend/internal/services"
)

// MultiModalSearchHandler handles multi-modal search requests
type MultiModalSearchHandler struct {
	multiModalSearch *services.MultiModalSearchService
	imageEmbedding   *services.ImageEmbeddingService
}

// NewMultiModalSearchHandler creates a new handler
func NewMultiModalSearchHandler(
	multiModalSearch *services.MultiModalSearchService,
	imageEmbedding *services.ImageEmbeddingService,
) *MultiModalSearchHandler {
	return &MultiModalSearchHandler{
		multiModalSearch: multiModalSearch,
		imageEmbedding:   imageEmbedding,
	}
}

// UploadImageRequest represents an image upload request
type UploadImageRequest struct {
	Image       string                 `json:"image"` // Base64 encoded image
	Caption     string                 `json:"caption"`
	Description string                 `json:"description"`
	Tags        []string               `json:"tags"`
	ContextID   *uuid.UUID             `json:"context_id"`
	ProjectID   *uuid.UUID             `json:"project_id"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// SearchWithImageRequest represents a multi-modal search request
type SearchWithImageRequest struct {
	Image         string      `json:"image"` // Base64 encoded (optional)
	Query         string      `json:"query"` // Text query (optional)
	MaxResults    int         `json:"max_results"`
	IncludeText   bool        `json:"include_text"`
	IncludeImages bool        `json:"include_images"`
	ContextIDs    []uuid.UUID `json:"context_ids"`

	// Weights
	SemanticWeight float64 `json:"semantic_weight"`
	KeywordWeight  float64 `json:"keyword_weight"`
	ImageWeight    float64 `json:"image_weight"`

	// Re-ranking
	ReRankEnabled bool `json:"rerank_enabled"`
}

// SearchImagesByTextRequest for cross-modal search (text → images)
type SearchImagesByTextRequest struct {
	Query      string `json:"query"`
	MaxResults int    `json:"max_results"`
}

// UploadImage handles image upload and embedding generation
// POST /api/images/upload
func (h *MultiModalSearchHandler) UploadImage(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	var req UploadImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondBadRequestErr(c, "Invalid request body")
		return
	}

	// Decode base64 image
	imageData, err := base64.StdEncoding.DecodeString(req.Image)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid base64 image"})
		return
	}

	// Prepare metadata
	metadata := req.Metadata
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["caption"] = req.Caption
	metadata["description"] = req.Description
	metadata["tags"] = req.Tags
	if req.ContextID != nil {
		metadata["context_id"] = req.ContextID.String()
	}
	if req.ProjectID != nil {
		metadata["project_id"] = req.ProjectID.String()
	}

	// Store image with embedding
	result, err := h.imageEmbedding.StoreImageEmbedding(c.Request.Context(), userID, imageData, metadata)
	if err != nil {
		RespondInternalErr(c, "store image embedding", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         result.ID,
		"user_id":    result.UserID,
		"created_at": result.CreatedAt,
		"metadata":   result.Metadata,
		"message":    "Image uploaded and indexed successfully",
	})
}

// SearchWithImage performs multi-modal search (text + image)
// POST /api/search/multimodal
func (h *MultiModalSearchHandler) SearchWithImage(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	var req SearchWithImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondBadRequestErr(c, "Invalid request body")
		return
	}

	// Validate: must have either image or query
	if req.Image == "" && req.Query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Must provide either image or query"})
		return
	}

	// Decode image if provided
	var imageData []byte
	var err error
	if req.Image != "" {
		imageData, err = base64.StdEncoding.DecodeString(req.Image)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid base64 image"})
			return
		}
	}

	// Set defaults
	if req.MaxResults == 0 {
		req.MaxResults = 20
	}
	if req.SemanticWeight == 0 && req.KeywordWeight == 0 && req.ImageWeight == 0 {
		req.SemanticWeight = 0.4
		req.KeywordWeight = 0.3
		req.ImageWeight = 0.3
	}

	// Build search options
	opts := services.SearchOptions{
		SemanticWeight:    req.SemanticWeight,
		KeywordWeight:     req.KeywordWeight,
		ImageWeight:       req.ImageWeight,
		ReRankEnabled:     req.ReRankEnabled,
		MaxResults:        req.MaxResults,
		MinSimilarity:     0.3,
		IncludeText:       req.IncludeText || req.Query != "",
		IncludeImages:     req.IncludeImages || req.Image != "",
		ContextIDs:        req.ContextIDs,
		RecencyWeight:     0.3,
		QualityWeight:     0.3,
		InteractionWeight: 0.2,
	}

	// Perform search
	results, err := h.multiModalSearch.SearchWithImage(
		c.Request.Context(),
		imageData,
		req.Query,
		userID,
		opts,
	)
	if err != nil {
		RespondInternalErr(c, "multimodal search", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"count":   len(results),
		"query":   req.Query,
		"options": opts,
	})
}

// SearchImagesByText performs cross-modal search (text query → find images)
// POST /api/search/images-by-text
func (h *MultiModalSearchHandler) SearchImagesByText(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	var req SearchImagesByTextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondBadRequestErr(c, "Invalid request body")
		return
	}

	if req.Query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Query is required"})
		return
	}

	if req.MaxResults == 0 {
		req.MaxResults = 10
	}

	// Use multimodal search with only images
	opts := services.SearchOptions{
		SemanticWeight: 0,
		KeywordWeight:  0,
		ImageWeight:    1.0,
		MaxResults:     req.MaxResults,
		IncludeText:    false,
		IncludeImages:  true,
		MinSimilarity:  0.3,
	}

	results, err := h.multiModalSearch.SearchWithImage(
		c.Request.Context(),
		nil, // No image, only text
		req.Query,
		userID,
		opts,
	)
	if err != nil {
		RespondInternalErr(c, "search images by text", err)
		return
	}

	// Filter to only image results
	imageResults := make([]services.MultiModalSearchResult, 0)
	for _, r := range results {
		if r.Type == "image" {
			imageResults = append(imageResults, r)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"results": imageResults,
		"count":   len(imageResults),
		"query":   req.Query,
	})
}

// SearchSimilarImages finds images similar to a given image
// POST /api/search/similar-images
func (h *MultiModalSearchHandler) SearchSimilarImages(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	var req struct {
		Image      string `json:"image"` // Base64
		MaxResults int    `json:"max_results"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if req.Image == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Image is required"})
		return
	}

	// Decode image
	imageData, err := base64.StdEncoding.DecodeString(req.Image)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid base64 image"})
		return
	}

	if req.MaxResults == 0 {
		req.MaxResults = 10
	}

	// Search for similar images
	results, err := h.imageEmbedding.SearchSimilarImages(c.Request.Context(), imageData, userID, req.MaxResults)
	if err != nil {
		RespondInternalErr(c, "search similar images", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"count":   len(results),
	})
}

// GetImage retrieves an image by ID
// GET /api/images/:id
func (h *MultiModalSearchHandler) GetImage(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid image ID"})
		return
	}

	image, err := h.imageEmbedding.GetImageEmbedding(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}

	// Check ownership
	if image.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         image.ID,
		"user_id":    image.UserID,
		"metadata":   image.Metadata,
		"created_at": image.CreatedAt,
	})
}

// GetImageData retrieves the actual image file
// GET /api/images/:id/data
func (h *MultiModalSearchHandler) GetImageData(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid image ID"})
		return
	}

	image, err := h.imageEmbedding.GetImageEmbedding(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}

	// Check ownership
	if image.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Determine content type from metadata or default
	contentType := "image/jpeg"
	if image.Metadata != nil {
		if mimeType, ok := image.Metadata["mime_type"].(string); ok {
			contentType = mimeType
		}
	}

	// Return image data
	c.Data(http.StatusOK, contentType, image.ImageData)
}

// DeleteImage deletes an image
// DELETE /api/images/:id
func (h *MultiModalSearchHandler) DeleteImage(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid image ID"})
		return
	}

	err = h.imageEmbedding.DeleteImageEmbedding(c.Request.Context(), id, userID)
	if err != nil {
		RespondInternalErr(c, "delete image", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Image deleted successfully"})
}

// UploadImageMultipart handles image upload via multipart form
// POST /api/images/upload-file
func (h *MultiModalSearchHandler) UploadImageMultipart(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	// Parse multipart form
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No image file provided"})
		return
	}
	defer file.Close()

	// Read file data
	imageData, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read image"})
		return
	}

	// Get optional metadata from form
	caption := c.PostForm("caption")
	description := c.PostForm("description")

	metadata := map[string]interface{}{
		"caption":     caption,
		"description": description,
		"filename":    header.Filename,
		"mime_type":   header.Header.Get("Content-Type"),
		"file_size":   header.Size,
	}

	// Store image with embedding
	result, err := h.imageEmbedding.StoreImageEmbedding(c.Request.Context(), userID, imageData, metadata)
	if err != nil {
		RespondInternalErr(c, "store uploaded image", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         result.ID,
		"user_id":    result.UserID,
		"filename":   header.Filename,
		"size":       header.Size,
		"created_at": result.CreatedAt,
		"message":    "Image uploaded and indexed successfully",
	})
}

// GetSupportedModalities returns what search modalities are supported
// GET /api/search/modalities
func (h *MultiModalSearchHandler) GetSupportedModalities(c *gin.Context) {
	modalities := h.multiModalSearch.GetSupportedModalities()

	c.JSON(http.StatusOK, gin.H{
		"modalities": modalities,
		"features": map[string]bool{
			"text_search":     true,
			"semantic_search": true,
			"keyword_search":  true,
			"image_search":    h.imageEmbedding != nil,
			"cross_modal":     h.imageEmbedding != nil,
			"hybrid_search":   true,
			"reranking":       true,
		},
	})
}

// RegisterMultiModalRoutes registers all multimodal search routes
func RegisterMultiModalRoutes(r *gin.RouterGroup, mmHandler *MultiModalSearchHandler) {
	// Image management
	r.POST("/images/upload", mmHandler.UploadImage)
	r.POST("/images/upload-file", mmHandler.UploadImageMultipart)
	r.GET("/images/:id", mmHandler.GetImage)
	r.GET("/images/:id/data", mmHandler.GetImageData)
	r.DELETE("/images/:id", mmHandler.DeleteImage)

	// Multi-modal search
	r.POST("/search/multimodal", mmHandler.SearchWithImage)
	r.POST("/search/images-by-text", mmHandler.SearchImagesByText)
	r.POST("/search/similar-images", mmHandler.SearchSimilarImages)
	r.GET("/search/modalities", mmHandler.GetSupportedModalities)
}
