// Package handlers provides HTTP handlers for the BusinessOS API.
// Dashboard CRUD handlers are split across:
//   - dashboard_handlers.go      — DashboardCRUDHandler struct, constructor, routes, dashboard CRUD
//   - dashboard_layout.go        — UpdateDashboardLayout, ShareUserDashboard, GetSharedDashboard
//   - dashboard_widgets.go       — ListWidgetTypes, GetWidgetSchema, ListDashboardTemplates, CreateDashboardFromTemplate
//   - dashboard_crud_helpers.go  — generateShareToken, transform helpers, dashboardUuidToString
package handlers
