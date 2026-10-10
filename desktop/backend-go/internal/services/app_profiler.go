package services

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AppProfilerService analyzes and profiles application codebases
type AppProfilerService struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewAppProfilerService creates a new app profiler service
func NewAppProfilerService(pool *pgxpool.Pool) *AppProfilerService {
	return &AppProfilerService{
		pool:   pool,
		logger: slog.Default().With("service", "app_profiler"),
	}
}

// ProfileApplication analyzes and profiles an application codebase.
// It orchestrates data collection, analysis, and persistence. Each phase
// is delegated to focused helpers in the app_profiler_collection,
// app_profiler_analysis, and app_profiler_queries files.
func (s *AppProfilerService) ProfileApplication(ctx context.Context, userID, rootPath, name string, opts *ProfileOptions) (*ApplicationProfile, error) {
	if opts == nil {
		opts = DefaultProfileOptions()
	}

	// Verify path exists
	info, err := os.Stat(rootPath)
	if err != nil {
		return nil, fmt.Errorf("path does not exist: %s", rootPath)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", rootPath)
	}

	profile := &ApplicationProfile{
		ID:                uuid.New().String(),
		UserID:            userID,
		Name:              name,
		RootPath:          rootPath,
		Languages:         make([]LanguageInfo, 0),
		Frameworks:        make([]string, 0),
		Components:        make([]ComponentInfo, 0),
		Modules:           make([]ModuleInfo, 0),
		APIEndpoints:      make([]APIEndpointInfo, 0),
		IntegrationPoints: make([]IntegrationPoint, 0),
		Metadata:          make(map[string]interface{}),
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	s.logger.Info("starting application profiling", "path", rootPath, "name", name)

	// --- Data collection (app_profiler_collection.go) ---
	profile.StructureTree = s.buildDirectoryTree(rootPath, opts, 0)
	profile.Languages, profile.LinesOfCode, profile.FileCount = s.analyzeLanguages(rootPath, opts)
	profile.AppType, profile.TechStack = s.detectTechStack(rootPath, profile.Languages)
	profile.Frameworks = s.detectFrameworks(rootPath)
	profile.Conventions = s.detectConventions(rootPath, opts)

	// --- Analysis (app_profiler_analysis.go) ---
	if opts.AnalyzeComponents {
		profile.Components = s.analyzeComponents(rootPath, profile.TechStack, opts)
		profile.TotalComponents = len(profile.Components)
	}

	profile.Modules = s.analyzeModules(rootPath, profile.Languages, opts)
	profile.TotalModules = len(profile.Modules)

	if opts.AnalyzeEndpoints {
		profile.APIEndpoints = s.analyzeEndpoints(rootPath, profile.TechStack, opts)
		profile.TotalEndpoints = len(profile.APIEndpoints)
	}

	if opts.AnalyzeDatabase {
		profile.DatabaseSchema = s.analyzeDatabaseSchema(rootPath)
	}

	if opts.ExtractReadme {
		profile.ReadmeSummary = s.extractReadmeSummary(rootPath)
	}

	profile.IntegrationPoints = s.detectIntegrations(rootPath)
	profile.Description = s.generateDescription(profile)
	profile.LastAnalyzedAt = time.Now()

	// --- Persistence (app_profiler_queries.go) ---
	if err := s.saveProfile(ctx, profile); err != nil {
		s.logger.Warn("failed to save profile", "error", err)
	}

	return profile, nil
}

// containsAny reports whether any element of slice contains any of the given items
// (case-insensitive substring match).
func containsAny(slice []string, items ...string) bool {
	for _, s := range slice {
		for _, item := range items {
			if strings.Contains(strings.ToLower(s), strings.ToLower(item)) {
				return true
			}
		}
	}
	return false
}
