package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/utils"
)

// OutputStylesHandler handles output style listing and user output preferences.
type OutputStylesHandler struct {
	pool *pgxpool.Pool
}

// NewOutputStylesHandler creates a new OutputStylesHandler.
func NewOutputStylesHandler(pool *pgxpool.Pool) *OutputStylesHandler {
	return &OutputStylesHandler{pool: pool}
}

// RegisterOutputStylesRoutes registers output style routes under the provided ai group.
func RegisterOutputStylesRoutes(aiGroup *gin.RouterGroup, h *OutputStylesHandler) {
	aiGroup.GET("/output-styles", h.ListOutputStyles)
	aiGroup.GET("/output-preferences", h.GetUserOutputPreference)
	aiGroup.PUT("/output-preferences", h.UpsertUserOutputPreference)
}

type OutputStyle struct {
	ID                uuid.UUID       `json:"id"`
	Name              string          `json:"name"`
	DisplayName       string          `json:"display_name"`
	Description       *string         `json:"description,omitempty"`
	Icon              *string         `json:"icon,omitempty"`
	StyleType         string          `json:"style_type"`
	UseHeaders        bool            `json:"use_headers"`
	UseBullets        bool            `json:"use_bullets"`
	UseNumberedLists  bool            `json:"use_numbered_lists"`
	UseParagraphs     bool            `json:"use_paragraphs"`
	UseCodeBlocks     bool            `json:"use_code_blocks"`
	UseTables         bool            `json:"use_tables"`
	UseBlockquotes    bool            `json:"use_blockquotes"`
	Verbosity         string          `json:"verbosity"`
	MaxParagraphs     *int32          `json:"max_paragraphs,omitempty"`
	MaxBulletsPerSect *int32          `json:"max_bullets_per_section,omitempty"`
	Tone              string          `json:"tone"`
	StyleInstructions string          `json:"style_instructions"`
	BlockMapping      json.RawMessage `json:"block_mapping"`
	IsSystem          bool            `json:"is_system"`
	IsActive          bool            `json:"is_active"`
	SortOrder         int32           `json:"sort_order"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

type UserOutputPreferenceResponse struct {
	UserID             string            `json:"user_id"`
	DefaultStyleID     *uuid.UUID        `json:"default_style_id,omitempty"`
	DefaultStyleName   *string           `json:"default_style_name,omitempty"`
	StyleOverrides     map[string]string `json:"style_overrides"`
	CustomInstructions *string           `json:"custom_instructions,omitempty"`
	CreatedAt          time.Time         `json:"created_at"`
	UpdatedAt          time.Time         `json:"updated_at"`
}

type UpsertUserOutputPreferenceRequest struct {
	DefaultStyleID     *string           `json:"default_style_id"`
	DefaultStyleName   *string           `json:"default_style_name"`
	StyleOverrides     map[string]string `json:"style_overrides"`
	CustomInstructions *string           `json:"custom_instructions"`
}

// ListOutputStyles returns active output styles. Intended for Settings UI.
func (h *OutputStylesHandler) ListOutputStyles(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	rows, err := h.pool.Query(ctx, `
		SELECT
			id, name, display_name, description, icon, style_type,
			use_headers, use_bullets, use_numbered_lists, use_paragraphs,
			use_code_blocks, use_tables, use_blockquotes,
			verbosity, max_paragraphs, max_bullets_per_section,
			tone, style_instructions, block_mapping,
			is_system, is_active, sort_order, created_at, updated_at
		FROM output_styles
		WHERE is_active = TRUE
		ORDER BY sort_order ASC
	`)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "list output styles", err)
		return
	}
	defer rows.Close()

	styles := make([]OutputStyle, 0)
	for rows.Next() {
		var style OutputStyle
		if err := rows.Scan(
			&style.ID,
			&style.Name,
			&style.DisplayName,
			&style.Description,
			&style.Icon,
			&style.StyleType,
			&style.UseHeaders,
			&style.UseBullets,
			&style.UseNumberedLists,
			&style.UseParagraphs,
			&style.UseCodeBlocks,
			&style.UseTables,
			&style.UseBlockquotes,
			&style.Verbosity,
			&style.MaxParagraphs,
			&style.MaxBulletsPerSect,
			&style.Tone,
			&style.StyleInstructions,
			&style.BlockMapping,
			&style.IsSystem,
			&style.IsActive,
			&style.SortOrder,
			&style.CreatedAt,
			&style.UpdatedAt,
		); err != nil {
			utils.RespondInternalError(c, slog.Default(), "scan output style", err)
			return
		}
		styles = append(styles, style)
	}

	c.JSON(http.StatusOK, gin.H{"styles": styles})
}

// GetUserOutputPreference returns the current user's output style preferences.
func (h *OutputStylesHandler) GetUserOutputPreference(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	var defaultStyleID *uuid.UUID
	var defaultStyleName *string
	var overridesBytes []byte
	var customInstructions *string
	var createdAt time.Time
	var updatedAt time.Time

	err := h.pool.QueryRow(ctx, `
		SELECT
			p.default_style_id,
			s.name AS default_style_name,
			COALESCE(p.style_overrides, '{}'::jsonb) AS style_overrides,
			p.custom_instructions,
			p.created_at,
			p.updated_at
		FROM user_output_preferences p
		LEFT JOIN output_styles s ON p.default_style_id = s.id
		WHERE p.user_id = $1
		LIMIT 1
	`, user.ID).Scan(&defaultStyleID, &defaultStyleName, &overridesBytes, &customInstructions, &createdAt, &updatedAt)
	if err != nil {
		// No preference set yet
		c.JSON(http.StatusOK, gin.H{"preference": nil})
		return
	}

	overrides := map[string]string{}
	if len(overridesBytes) > 0 {
		_ = json.Unmarshal(overridesBytes, &overrides)
	}

	pref := UserOutputPreferenceResponse{
		UserID:             user.ID,
		DefaultStyleID:     defaultStyleID,
		DefaultStyleName:   defaultStyleName,
		StyleOverrides:     overrides,
		CustomInstructions: customInstructions,
		CreatedAt:          createdAt,
		UpdatedAt:          updatedAt,
	}

	c.JSON(http.StatusOK, gin.H{"preference": pref})
}

// UpsertUserOutputPreference creates/updates the current user's output style preferences.
func (h *OutputStylesHandler) UpsertUserOutputPreference(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	var req UpsertUserOutputPreferenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	var defaultStyleID *uuid.UUID
	if req.DefaultStyleID != nil && *req.DefaultStyleID != "" {
		parsed, err := uuid.Parse(*req.DefaultStyleID)
		if err != nil {
			utils.RespondInvalidID(c, slog.Default(), "default_style_id")
			return
		}
		defaultStyleID = &parsed
	} else if req.DefaultStyleName != nil && *req.DefaultStyleName != "" {
		var resolved uuid.UUID
		err := h.pool.QueryRow(ctx, `SELECT id FROM output_styles WHERE name = $1 LIMIT 1`, *req.DefaultStyleName).Scan(&resolved)
		if err != nil {
			utils.RespondBadRequest(c, slog.Default(), "Unknown default_style_name")
			return
		}
		defaultStyleID = &resolved
	}

	overrides := req.StyleOverrides
	if overrides == nil {
		overrides = map[string]string{}
	}
	overridesBytes, err := json.Marshal(overrides)
	if err != nil {
		utils.RespondBadRequest(c, slog.Default(), "Invalid style_overrides")
		return
	}

	var returnedDefaultStyleID *uuid.UUID
	var returnedOverrides []byte
	var returnedCustomInstructions *string
	var createdAt time.Time
	var updatedAt time.Time

	err = h.pool.QueryRow(ctx, `
		INSERT INTO user_output_preferences (user_id, default_style_id, style_overrides, custom_instructions, updated_at)
		VALUES ($1, $2, $3::jsonb, $4, NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			default_style_id = EXCLUDED.default_style_id,
			style_overrides = EXCLUDED.style_overrides,
			custom_instructions = EXCLUDED.custom_instructions,
			updated_at = NOW()
		RETURNING default_style_id, style_overrides, custom_instructions, created_at, updated_at
	`, user.ID, defaultStyleID, overridesBytes, req.CustomInstructions).Scan(
		&returnedDefaultStyleID,
		&returnedOverrides,
		&returnedCustomInstructions,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "save output preference", err)
		return
	}

	var returnedDefaultStyleName *string
	if returnedDefaultStyleID != nil {
		_ = h.pool.QueryRow(ctx, `SELECT name FROM output_styles WHERE id = $1 LIMIT 1`, *returnedDefaultStyleID).Scan(&returnedDefaultStyleName)
	}

	decodedOverrides := map[string]string{}
	if len(returnedOverrides) > 0 {
		_ = json.Unmarshal(returnedOverrides, &decodedOverrides)
	}

	pref := UserOutputPreferenceResponse{
		UserID:             user.ID,
		DefaultStyleID:     returnedDefaultStyleID,
		DefaultStyleName:   returnedDefaultStyleName,
		StyleOverrides:     decodedOverrides,
		CustomInstructions: returnedCustomInstructions,
		CreatedAt:          createdAt,
		UpdatedAt:          updatedAt,
	}

	c.JSON(http.StatusOK, gin.H{"preference": pref})
}
