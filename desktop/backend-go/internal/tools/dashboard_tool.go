package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
)

// ConfigureDashboardTool handles all dashboard configuration operations
type ConfigureDashboardTool struct {
	pool   *pgxpool.Pool
	userID string
}

func (t *ConfigureDashboardTool) Name() string { return "configure_dashboard" }

func (t *ConfigureDashboardTool) Description() string {
	return `Configure user dashboards. Actions: create_dashboard, add_widget, remove_widget, reorder_widgets, update_widget_config, list_dashboards, get_dashboard, delete_dashboard, set_default, clone_dashboard.`
}

func (t *ConfigureDashboardTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"create_dashboard", "add_widget", "remove_widget", "reorder_widgets", "update_widget_config", "list_dashboards", "get_dashboard", "delete_dashboard", "set_default", "clone_dashboard"},
				"description": "The action to perform",
			},
			"dashboard_id": map[string]interface{}{
				"type":        "string",
				"description": "Dashboard UUID (required for most actions except create/list)",
			},
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Dashboard name (for create_dashboard)",
			},
			"widget_type": map[string]interface{}{
				"type":        "string",
				"description": "Widget type to add (for add_widget)",
			},
			"widget_id": map[string]interface{}{
				"type":        "string",
				"description": "Widget instance ID in layout (for remove_widget, update_widget_config)",
			},
			"config": map[string]interface{}{
				"type":        "object",
				"description": "Widget configuration (for add_widget, update_widget_config)",
			},
			"position": map[string]interface{}{
				"type":        "object",
				"description": "Widget position {x, y, w, h} (for add_widget)",
				"properties": map[string]interface{}{
					"x": map[string]interface{}{"type": "integer"},
					"y": map[string]interface{}{"type": "integer"},
					"w": map[string]interface{}{"type": "integer"},
					"h": map[string]interface{}{"type": "integer"},
				},
			},
			"order": map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": "Array of widget IDs in desired order (for reorder_widgets)",
			},
		},
		"required": []string{"action"},
	}
}

type DashboardInput struct {
	Action      string                 `json:"action"`
	DashboardID string                 `json:"dashboard_id"`
	Name        string                 `json:"name"`
	WidgetType  string                 `json:"widget_type"`
	WidgetID    string                 `json:"widget_id"`
	Config      map[string]interface{} `json:"config"`
	Position    *WidgetPosition        `json:"position"`
	Order       []string               `json:"order"`
}

type WidgetPosition struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

type LayoutWidget struct {
	ID     string                 `json:"id"`
	Type   string                 `json:"type"`
	X      int                    `json:"x"`
	Y      int                    `json:"y"`
	W      int                    `json:"w"`
	H      int                    `json:"h"`
	Config map[string]interface{} `json:"config"`
}

type DashboardLayout struct {
	Widgets []LayoutWidget `json:"widgets"`
}

// Helper to convert google/uuid to pgtype.UUID
func uuidToPgtype(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}


func (t *ConfigureDashboardTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var params DashboardInput
	if err := json.Unmarshal(input, &params); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}

	queries := sqlc.New(t.pool)

	switch params.Action {
	case "create_dashboard":
		return t.createDashboard(ctx, queries, params)
	case "add_widget":
		return t.addWidget(ctx, queries, params)
	case "remove_widget":
		return t.removeWidget(ctx, queries, params)
	case "reorder_widgets":
		return t.reorderWidgets(ctx, queries, params)
	case "update_widget_config":
		return t.updateWidgetConfig(ctx, queries, params)
	case "list_dashboards":
		return t.listDashboards(ctx, queries)
	case "get_dashboard":
		return t.getDashboard(ctx, queries, params)
	case "delete_dashboard":
		return t.deleteDashboard(ctx, queries, params)
	case "set_default":
		return t.setDefault(ctx, queries, params)
	case "clone_dashboard":
		return t.cloneDashboard(ctx, queries, params)
	default:
		return "", fmt.Errorf("unknown action: %s", params.Action)
	}
}

func (t *ConfigureDashboardTool) createDashboard(ctx context.Context, queries *sqlc.Queries, params DashboardInput) (string, error) {
	if params.Name == "" {
		return "", fmt.Errorf("name is required for create_dashboard")
	}

	layout := DashboardLayout{Widgets: []LayoutWidget{}}
	layoutJSON, err := json.Marshal(layout)
	if err != nil {
		return "", fmt.Errorf("marshal layout: %w", err)
	}

	visibility := "private"
	createdVia := "agent"

	dashboard, err := queries.CreateDashboard(ctx, sqlc.CreateDashboardParams{
		UserID:      t.userID,
		WorkspaceID: pgtype.UUID{Valid: false},
		Name:        params.Name,
		Description: nil,
		Layout:      layoutJSON,
		Visibility:  &visibility,
		CreatedVia:  &createdVia,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create dashboard: %w", err)
	}

	return fmt.Sprintf("Created dashboard '%s' (ID: %s). Add widgets using add_widget action.", dashboard.Name, uuidToString(dashboard.ID)), nil
}

func (t *ConfigureDashboardTool) addWidget(ctx context.Context, queries *sqlc.Queries, params DashboardInput) (string, error) {
	if params.DashboardID == "" {
		return "", fmt.Errorf("dashboard_id is required")
	}
	if params.WidgetType == "" {
		return "", fmt.Errorf("widget_type is required")
	}

	dashboardUUID, err := uuid.Parse(params.DashboardID)
	if err != nil {
		return "", fmt.Errorf("invalid dashboard_id: %w", err)
	}

	// Verify widget type exists
	widgetType, err := queries.GetWidgetTypeByName(ctx, params.WidgetType)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", fmt.Errorf("invalid widget_type '%s'. Use list_dashboards to see available widget types", params.WidgetType)
		}
		return "", fmt.Errorf("failed to verify widget type: %w", err)
	}

	// Get dashboard
	dashboard, err := queries.GetDashboard(ctx, sqlc.GetDashboardParams{
		ID:     uuidToPgtype(dashboardUUID),
		UserID: t.userID,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", fmt.Errorf("dashboard not found or access denied")
		}
		return "", fmt.Errorf("failed to get dashboard: %w", err)
	}

	// Parse layout
	var layout DashboardLayout
	if err := json.Unmarshal(dashboard.Layout, &layout); err != nil {
		layout = DashboardLayout{Widgets: []LayoutWidget{}}
	}

	// Default position
	pos := params.Position
	if pos == nil {
		pos = &WidgetPosition{X: 0, Y: len(layout.Widgets) * 2, W: 4, H: 2}
	}

	// Default config from widget type
	config := params.Config
	if config == nil {
		config = make(map[string]interface{})
	}
	var defaultConfig map[string]interface{}
	if widgetType.DefaultConfig != nil {
		json.Unmarshal(widgetType.DefaultConfig, &defaultConfig)
		for k, v := range defaultConfig {
			if _, exists := config[k]; !exists {
				config[k] = v
			}
		}
	}

	// Create widget
	widget := LayoutWidget{
		ID:     uuid.New().String(),
		Type:   params.WidgetType,
		X:      pos.X,
		Y:      pos.Y,
		W:      pos.W,
		H:      pos.H,
		Config: config,
	}

	layout.Widgets = append(layout.Widgets, widget)
	layoutJSON, err := json.Marshal(layout)
	if err != nil {
		return "", fmt.Errorf("marshal layout: %w", err)
	}

	_, err = queries.UpdateDashboardLayout(ctx, sqlc.UpdateDashboardLayoutParams{
		ID:     uuidToPgtype(dashboardUUID),
		UserID: t.userID,
		Layout: layoutJSON,
	})
	if err != nil {
		return "", fmt.Errorf("failed to update layout: %w", err)
	}

	return fmt.Sprintf("Added %s widget (ID: %s) at position (%d,%d) with size %dx%d", params.WidgetType, widget.ID, pos.X, pos.Y, pos.W, pos.H), nil
}

func (t *ConfigureDashboardTool) removeWidget(ctx context.Context, queries *sqlc.Queries, params DashboardInput) (string, error) {
	if params.DashboardID == "" || params.WidgetID == "" {
		return "", fmt.Errorf("dashboard_id and widget_id are required")
	}

	dashboardUUID, _ := uuid.Parse(params.DashboardID)

	dashboard, err := queries.GetDashboard(ctx, sqlc.GetDashboardParams{
		ID:     uuidToPgtype(dashboardUUID),
		UserID: t.userID,
	})
	if err != nil {
		return "", fmt.Errorf("dashboard not found: %w", err)
	}

	var layout DashboardLayout
	json.Unmarshal(dashboard.Layout, &layout)

	// Find and remove widget
	found := false
	var removedType string
	newWidgets := make([]LayoutWidget, 0, len(layout.Widgets))
	for _, w := range layout.Widgets {
		if w.ID == params.WidgetID {
			found = true
			removedType = w.Type
		} else {
			newWidgets = append(newWidgets, w)
		}
	}

	if !found {
		return "", fmt.Errorf("widget %s not found in dashboard", params.WidgetID)
	}

	layout.Widgets = newWidgets
	layoutJSON, err := json.Marshal(layout)
	if err != nil {
		return "", fmt.Errorf("marshal layout: %w", err)
	}

	_, err = queries.UpdateDashboardLayout(ctx, sqlc.UpdateDashboardLayoutParams{
		ID:     uuidToPgtype(dashboardUUID),
		UserID: t.userID,
		Layout: layoutJSON,
	})
	if err != nil {
		return "", fmt.Errorf("failed to update layout: %w", err)
	}

	return fmt.Sprintf("Removed %s widget from dashboard", removedType), nil
}

func (t *ConfigureDashboardTool) reorderWidgets(ctx context.Context, queries *sqlc.Queries, params DashboardInput) (string, error) {
	if params.DashboardID == "" || len(params.Order) == 0 {
		return "", fmt.Errorf("dashboard_id and order array are required")
	}

	dashboardUUID, _ := uuid.Parse(params.DashboardID)

	dashboard, err := queries.GetDashboard(ctx, sqlc.GetDashboardParams{
		ID:     uuidToPgtype(dashboardUUID),
		UserID: t.userID,
	})
	if err != nil {
		return "", fmt.Errorf("dashboard not found: %w", err)
	}

	var layout DashboardLayout
	json.Unmarshal(dashboard.Layout, &layout)

	// Create lookup map
	widgetMap := make(map[string]LayoutWidget)
	for _, w := range layout.Widgets {
		widgetMap[w.ID] = w
	}

	// Reorder based on provided order
	newWidgets := make([]LayoutWidget, 0, len(layout.Widgets))
	for i, id := range params.Order {
		if w, ok := widgetMap[id]; ok {
			w.Y = i * 2 // Stack vertically
			newWidgets = append(newWidgets, w)
			delete(widgetMap, id)
		}
	}
	// Append any remaining widgets not in order
	for _, w := range widgetMap {
		newWidgets = append(newWidgets, w)
	}

	layout.Widgets = newWidgets
	layoutJSON, err := json.Marshal(layout)
	if err != nil {
		return "", fmt.Errorf("marshal layout: %w", err)
	}

	_, err = queries.UpdateDashboardLayout(ctx, sqlc.UpdateDashboardLayoutParams{
		ID:     uuidToPgtype(dashboardUUID),
		UserID: t.userID,
		Layout: layoutJSON,
	})

	return fmt.Sprintf("Reordered %d widgets", len(newWidgets)), nil
}

func (t *ConfigureDashboardTool) updateWidgetConfig(ctx context.Context, queries *sqlc.Queries, params DashboardInput) (string, error) {
	if params.DashboardID == "" || params.WidgetID == "" || params.Config == nil {
		return "", fmt.Errorf("dashboard_id, widget_id, and config are required")
	}

	dashboardUUID, _ := uuid.Parse(params.DashboardID)

	dashboard, err := queries.GetDashboard(ctx, sqlc.GetDashboardParams{
		ID:     uuidToPgtype(dashboardUUID),
		UserID: t.userID,
	})
	if err != nil {
		return "", fmt.Errorf("dashboard not found: %w", err)
	}

	var layout DashboardLayout
	json.Unmarshal(dashboard.Layout, &layout)

	// Find and update widget config
	found := false
	for i, w := range layout.Widgets {
		if w.ID == params.WidgetID {
			found = true
			for k, v := range params.Config {
				layout.Widgets[i].Config[k] = v
			}
			break
		}
	}

	if !found {
		return "", fmt.Errorf("widget %s not found", params.WidgetID)
	}

	layoutJSON, err := json.Marshal(layout)
	if err != nil {
		return "", fmt.Errorf("marshal layout: %w", err)
	}

	_, err = queries.UpdateDashboardLayout(ctx, sqlc.UpdateDashboardLayoutParams{
		ID:     uuidToPgtype(dashboardUUID),
		UserID: t.userID,
		Layout: layoutJSON,
	})
	if err != nil {
		return "", fmt.Errorf("failed to update layout: %w", err)
	}

	return "Widget configuration updated", nil
}

func (t *ConfigureDashboardTool) listDashboards(ctx context.Context, queries *sqlc.Queries) (string, error) {
	dashboards, err := queries.ListUserDashboards(ctx, t.userID)
	if err != nil {
		return "", fmt.Errorf("failed to list dashboards: %w", err)
	}

	// Also get widget types
	widgetTypes, _ := queries.ListWidgetTypes(ctx)

	var sb strings.Builder
	sb.WriteString("## Your Dashboards\n\n")

	if len(dashboards) == 0 {
		sb.WriteString("No dashboards found. Use create_dashboard to create one.\n\n")
	} else {
		for _, d := range dashboards {
			defaultMark := ""
			if d.IsDefault != nil && *d.IsDefault {
				defaultMark = " ⭐"
			}
			var layout DashboardLayout
			json.Unmarshal(d.Layout, &layout)
			sb.WriteString(fmt.Sprintf("- **%s**%s (ID: %s) - %d widgets\n", d.Name, defaultMark, uuidToString(d.ID), len(layout.Widgets)))
		}
	}

	sb.WriteString("\n## Available Widget Types\n\n")
	for _, wt := range widgetTypes {
		desc := ""
		if wt.Description != nil {
			desc = *wt.Description
		}
		category := ""
		if wt.Category != nil {
			category = *wt.Category
		}
		sb.WriteString(fmt.Sprintf("- `%s` (%s): %s\n", wt.WidgetType, category, desc))
	}

	return sb.String(), nil
}

func (t *ConfigureDashboardTool) getDashboard(ctx context.Context, queries *sqlc.Queries, params DashboardInput) (string, error) {
	if params.DashboardID == "" {
		return "", fmt.Errorf("dashboard_id is required")
	}

	dashboardUUID, _ := uuid.Parse(params.DashboardID)

	dashboard, err := queries.GetDashboard(ctx, sqlc.GetDashboardParams{
		ID:     uuidToPgtype(dashboardUUID),
		UserID: t.userID,
	})
	if err != nil {
		return "", fmt.Errorf("dashboard not found: %w", err)
	}

	var layout DashboardLayout
	json.Unmarshal(dashboard.Layout, &layout)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## %s\n\n", dashboard.Name))
	sb.WriteString(fmt.Sprintf("- ID: %s\n", uuidToString(dashboard.ID)))
	isDefault := dashboard.IsDefault != nil && *dashboard.IsDefault
	sb.WriteString(fmt.Sprintf("- Default: %v\n", isDefault))
	visibility := ""
	if dashboard.Visibility != nil {
		visibility = *dashboard.Visibility
	}
	sb.WriteString(fmt.Sprintf("- Visibility: %s\n", visibility))
	sb.WriteString(fmt.Sprintf("\n### Widgets (%d)\n\n", len(layout.Widgets)))

	for _, w := range layout.Widgets {
		configStr, err := json.Marshal(w.Config)
		if err != nil {
			configStr = []byte("{}")  // Fallback to empty object
		}
		sb.WriteString(fmt.Sprintf("- **%s** (ID: %s)\n  Position: (%d,%d) Size: %dx%d\n  Config: %s\n\n", w.Type, w.ID, w.X, w.Y, w.W, w.H, string(configStr)))
	}

	return sb.String(), nil
}

func (t *ConfigureDashboardTool) deleteDashboard(ctx context.Context, queries *sqlc.Queries, params DashboardInput) (string, error) {
	if params.DashboardID == "" {
		return "", fmt.Errorf("dashboard_id is required")
	}

	dashboardUUID, _ := uuid.Parse(params.DashboardID)

	// Get name first
	dashboard, err := queries.GetDashboard(ctx, sqlc.GetDashboardParams{
		ID:     uuidToPgtype(dashboardUUID),
		UserID: t.userID,
	})
	if err != nil {
		return "", fmt.Errorf("dashboard not found: %w", err)
	}

	err = queries.DeleteDashboard(ctx, sqlc.DeleteDashboardParams{
		ID:     uuidToPgtype(dashboardUUID),
		UserID: t.userID,
	})
	if err != nil {
		return "", fmt.Errorf("failed to delete: %w", err)
	}

	return fmt.Sprintf("Deleted dashboard '%s'", dashboard.Name), nil
}

func (t *ConfigureDashboardTool) setDefault(ctx context.Context, queries *sqlc.Queries, params DashboardInput) (string, error) {
	if params.DashboardID == "" {
		return "", fmt.Errorf("dashboard_id is required")
	}

	dashboardUUID, _ := uuid.Parse(params.DashboardID)

	// Verify dashboard exists
	dashboard, err := queries.GetDashboard(ctx, sqlc.GetDashboardParams{
		ID:     uuidToPgtype(dashboardUUID),
		UserID: t.userID,
	})
	if err != nil {
		return "", fmt.Errorf("dashboard not found: %w", err)
	}

	// Clear existing default
	queries.ClearDefaultDashboard(ctx, t.userID)

	// Set new default
	queries.SetDefaultDashboard(ctx, sqlc.SetDefaultDashboardParams{
		ID:     uuidToPgtype(dashboardUUID),
		UserID: t.userID,
	})

	return fmt.Sprintf("'%s' is now your default dashboard", dashboard.Name), nil
}

func (t *ConfigureDashboardTool) cloneDashboard(ctx context.Context, queries *sqlc.Queries, params DashboardInput) (string, error) {
	if params.DashboardID == "" {
		return "", fmt.Errorf("dashboard_id is required")
	}

	dashboardUUID, _ := uuid.Parse(params.DashboardID)

	newName := params.Name
	if newName == "" {
		newName = "Dashboard Copy"
	}

	newDashboard, err := queries.DuplicateDashboard(ctx, sqlc.DuplicateDashboardParams{
		ID:     uuidToPgtype(dashboardUUID),
		UserID: t.userID,
		Name:   newName,
	})
	if err != nil {
		return "", fmt.Errorf("failed to clone: %w", err)
	}

	return fmt.Sprintf("Created '%s' (ID: %s) as a copy", newDashboard.Name, uuidToString(newDashboard.ID)), nil
}

// Helper to convert pgtype.UUID to string
func uuidToString(id pgtype.UUID) string {
	if !id.Valid {
		return ""
	}
	u := uuid.UUID(id.Bytes)
	return u.String()
}
