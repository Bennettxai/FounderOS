package services

// --- API Backend Template ---
func apiBackendTemplate() *BuiltInTemplate {
	return &BuiltInTemplate{
		ID:          "api_backend",
		Name:        "API Backend",
		Description: "Go REST API with CRUD operations, authentication, and database integration",
		Category:    "operations",
		StackType:   "go",
		ConfigSchema: map[string]ConfigField{
			"app_name":      {Type: "string", Label: "Application Name", Default: "My API", Required: true},
			"module_name":   {Type: "string", Label: "Go Module Name", Default: "github.com/user/myapi", Required: true},
			"port":          {Type: "string", Label: "Server Port", Default: "8080", Required: false},
			"database_type": {Type: "select", Label: "Database", Default: "postgres", Options: []string{"postgres", "sqlite", "mysql"}},
			"auth_type":     {Type: "select", Label: "Authentication", Default: "jwt", Options: []string{"jwt", "session", "api_key"}},
		},
		FilesTemplate: map[string]string{
			"cmd/server/main.go": `package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"{{module_name}}/internal/handlers"
	"{{module_name}}/internal/middleware"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	slog.Info("starting {{app_name}}", "port", "{{port}}")

	mux := http.NewServeMux()

	// Register routes
	handlers.RegisterRoutes(mux)

	// Apply middleware
	handler := middleware.Chain(
		mux,
		middleware.Logger,
		middleware.CORS,
		middleware.Recovery,
	)

	srv := &http.Server{
		Addr:         ":{{port}}",
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	slog.Info("server started", "addr", srv.Addr)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}

	slog.Info("server exited properly")
}
`,
			"internal/handlers/routes.go": `package handlers

import (
	"net/http"
)

// RegisterRoutes registers all API routes
func RegisterRoutes(mux *http.ServeMux) {
	// Health check
	mux.HandleFunc("GET /api/health", HealthCheck)

	// CRUD resources
	mux.HandleFunc("GET /api/items", ListItems)
	mux.HandleFunc("POST /api/items", CreateItem)
	mux.HandleFunc("GET /api/items/{id}", GetItem)
	mux.HandleFunc("PUT /api/items/{id}", UpdateItem)
	mux.HandleFunc("DELETE /api/items/{id}", DeleteItem)
}
`,
			"internal/handlers/health.go": `package handlers

import (
	"encoding/json"
	"net/http"
)

// HealthCheck returns the API health status
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":  "healthy",
		"service": "{{app_name}}",
		"version": "1.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
`,
			"internal/handlers/items.go": `package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Item represents a generic resource
type Item struct {
	ID          string    ` + "`" + `json:"id"` + "`" + `
	Name        string    ` + "`" + `json:"name"` + "`" + `
	Description string    ` + "`" + `json:"description"` + "`" + `
	Status      string    ` + "`" + `json:"status"` + "`" + `
	CreatedAt   time.Time ` + "`" + `json:"created_at"` + "`" + `
	UpdatedAt   time.Time ` + "`" + `json:"updated_at"` + "`" + `
}

// In-memory store (replace with {{database_type}} in production)
var (
	items   = make(map[string]Item)
	itemsMu sync.RWMutex
)

// ListItems returns all items
func ListItems(w http.ResponseWriter, r *http.Request) {
	itemsMu.RLock()
	defer itemsMu.RUnlock()

	result := make([]Item, 0, len(items))
	for _, item := range items {
		result = append(result, item)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"items": result,
		"total": len(result),
	})
}

// CreateItem creates a new item
func CreateItem(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string ` + "`" + `json:"name"` + "`" + `
		Description string ` + "`" + `json:"description"` + "`" + `
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "name is required")
		return
	}

	item := Item{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		Status:      "active",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	itemsMu.Lock()
	items[item.ID] = item
	itemsMu.Unlock()

	slog.Info("item created", "id", item.ID, "name", item.Name)
	respondJSON(w, http.StatusCreated, item)
}

// GetItem returns a single item
func GetItem(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	itemsMu.RLock()
	item, exists := items[id]
	itemsMu.RUnlock()

	if !exists {
		respondError(w, http.StatusNotFound, "item not found")
		return
	}

	respondJSON(w, http.StatusOK, item)
}

// UpdateItem updates an existing item
func UpdateItem(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	itemsMu.RLock()
	item, exists := items[id]
	itemsMu.RUnlock()

	if !exists {
		respondError(w, http.StatusNotFound, "item not found")
		return
	}

	var req struct {
		Name        *string ` + "`" + `json:"name"` + "`" + `
		Description *string ` + "`" + `json:"description"` + "`" + `
		Status      *string ` + "`" + `json:"status"` + "`" + `
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name != nil {
		item.Name = *req.Name
	}
	if req.Description != nil {
		item.Description = *req.Description
	}
	if req.Status != nil {
		item.Status = *req.Status
	}
	item.UpdatedAt = time.Now()

	itemsMu.Lock()
	items[id] = item
	itemsMu.Unlock()

	slog.Info("item updated", "id", item.ID)
	respondJSON(w, http.StatusOK, item)
}

// DeleteItem deletes an item
func DeleteItem(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	itemsMu.Lock()
	_, exists := items[id]
	if exists {
		delete(items, id)
	}
	itemsMu.Unlock()

	if !exists {
		respondError(w, http.StatusNotFound, "item not found")
		return
	}

	slog.Info("item deleted", "id", id)
	w.WriteHeader(http.StatusNoContent)
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
`,
			"internal/middleware/middleware.go": `package middleware

import (
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

// Middleware is a function that wraps an http.Handler
type Middleware func(http.Handler) http.Handler

// Chain applies middlewares in order
func Chain(handler http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

// Logger logs HTTP requests
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"duration", time.Since(start).String(),
		)
	})
}

// CORS adds CORS headers with origin validation
func CORS(next http.Handler) http.Handler {
	allowed := os.Getenv("ALLOWED_ORIGINS")
	var origins []string
	if allowed != "" {
		for _, o := range strings.Split(allowed, ",") {
			origins = append(origins, strings.TrimSpace(o))
		}
	} else {
		// Default: localhost only (no wildcard)
		origins = []string{"http://localhost:5173", "http://localhost:3000"}

		// PRODUCTION GUARD: Warn if using default origins in production
		env := os.Getenv("ENVIRONMENT")
		if env == "production" || env == "prod" || os.Getenv("ENV") == "production" {
			slog.Error("SECURITY WARNING: Using default CORS origins in production. Set ALLOWED_ORIGINS environment variable to explicit domains.")
			// In strict mode, you could panic here:
			// panic("CORS MISCONFIGURATION: ALLOWED_ORIGINS must be set in production")
		}
	}

	originSet := make(map[string]bool, len(origins))
	for _, o := range origins {
		// Normalize: lowercase, trim trailing slashes
		normalized := strings.TrimRight(strings.ToLower(strings.TrimSpace(o)), "/")
		originSet[normalized] = true
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqOrigin := r.Header.Get("Origin")
		// Normalize incoming origin for case-insensitive comparison
		normalizedReq := strings.TrimRight(strings.ToLower(strings.TrimSpace(reqOrigin)), "/")
		if originSet[normalizedReq] {
			w.Header().Set("Access-Control-Allow-Origin", reqOrigin) // Reflect original
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Vary", "Origin")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Recovery recovers from panics
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("panic recovered", "error", err, "path", r.URL.Path)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
`,
			"go.mod": `module {{module_name}}

go 1.22

require (
	github.com/google/uuid v1.6.0
)
`,
			"Dockerfile": `FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/server .

EXPOSE {{port}}
CMD ["./server"]
`,
		},
	}
}
