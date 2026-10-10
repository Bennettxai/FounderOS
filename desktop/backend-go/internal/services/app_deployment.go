package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AppDeploymentService handles local deployment of generated applications
type AppDeploymentService struct {
	pool            *pgxpool.Pool
	logger          *slog.Logger
	workspaceRoot   string
	deployedApps    map[uuid.UUID]*DeployedApp
	portAllocator   *PortAllocator
	mu              sync.RWMutex
}

// DeployedApp represents a running application instance
type DeployedApp struct {
	ID            uuid.UUID
	Name          string
	WorkflowID    string
	Port          int
	URL           string
	ProcessID     int
	Status        string
	DeployedAt    time.Time
	LastHealthy   time.Time
	AppType       string // node, python, static
	RootPath      string
	BuildOutput   string
	StartupOutput string
	Metadata      *AppMetadata // Extracted app metadata (category, icon, etc.)
	cmd           *exec.Cmd
}

// PortAllocator manages dynamic port assignment
type PortAllocator struct {
	nextPort int
	used     map[int]bool
	mu       sync.Mutex
}

// NewPortAllocator creates a new port allocator starting from port 9000
func NewPortAllocator(startPort int) *PortAllocator {
	return &PortAllocator{
		nextPort: startPort,
		used:     make(map[int]bool),
	}
}

// Allocate assigns the next available port
func (pa *PortAllocator) Allocate() int {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	for pa.used[pa.nextPort] {
		pa.nextPort++
	}

	port := pa.nextPort
	pa.used[port] = true
	pa.nextPort++

	return port
}

// Release marks a port as available
func (pa *PortAllocator) Release(port int) {
	pa.mu.Lock()
	defer pa.mu.Unlock()
	delete(pa.used, port)
}

// NewAppDeploymentService creates a new deployment service
func NewAppDeploymentService(pool *pgxpool.Pool, logger *slog.Logger, workspaceRoot string) *AppDeploymentService {
	if workspaceRoot == "" {
		workspaceRoot = "/tmp/businessos-apps"
	}

	// Create workspace root if it doesn't exist
	os.MkdirAll(workspaceRoot, 0755)

	return &AppDeploymentService{
		pool:          pool,
		logger:        logger,
		workspaceRoot: workspaceRoot,
		deployedApps:  make(map[uuid.UUID]*DeployedApp),
		portAllocator: NewPortAllocator(9000),
	}
}

// DeployApp extracts code bundle, installs deps, builds, and starts the app
func (s *AppDeploymentService) DeployApp(ctx context.Context, appID uuid.UUID) (*DeployedApp, error) {
	// Get app details from database
	var workflowID, name string
	var metadataJSON []byte

	query := `
		SELECT osa_workflow_id, name, metadata
		FROM osa_module_instances
		WHERE id = $1
	`

	err := s.pool.QueryRow(ctx, query, appID).Scan(&workflowID, &name, &metadataJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to get app: %w", err)
	}

	// Parse metadata to get code bundle
	var metadata map[string]interface{}
	if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	codeBundle, ok := metadata["code"].(string)
	if !ok || codeBundle == "" {
		return nil, fmt.Errorf("no code found in app metadata")
	}

	// Parse the multi-file bundle
	files := ParseFileBundle(codeBundle)
	if len(files) == 0 {
		return nil, fmt.Errorf("no files extracted from bundle")
	}

	// Create app directory
	appDir := filepath.Join(s.workspaceRoot, appID.String())
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create app directory: %w", err)
	}

	s.logger.Info("Deploying app", "app_id", appID, "name", name, "files", len(files))

	// Write all files to disk
	for _, file := range files {
		filePath := filepath.Join(appDir, file.Path)

		// Create parent directories
		fileDir := filepath.Dir(filePath)
		if err := os.MkdirAll(fileDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", fileDir, err)
		}

		// Write file
		if err := os.WriteFile(filePath, []byte(file.Content), 0644); err != nil {
			return nil, fmt.Errorf("failed to write file %s: %w", file.Path, err)
		}
	}

	// Detect app type and deploy accordingly
	appType := s.detectAppType(files)
	s.logger.Info("Detected app type", "type", appType)

	var deployedApp *DeployedApp
	switch appType {
	case "node":
		deployedApp, err = s.deployNodeApp(ctx, appID, name, workflowID, appDir)
	case "python":
		deployedApp, err = s.deployPythonApp(ctx, appID, name, workflowID, appDir)
	case "static":
		deployedApp, err = s.deployStaticApp(ctx, appID, name, workflowID, appDir)
	default:
		return nil, fmt.Errorf("unsupported app type: %s", appType)
	}

	if err != nil {
		return nil, err
	}

	// Extract app metadata (category, icon, description)
	// Convert files to bundleContent map for intelligent analysis
	bundleContent := make(map[string]string)
	for _, file := range files {
		bundleContent[file.Path] = file.Content
	}

	appMetadata, err := ExtractAppMetadata(appDir, bundleContent)
	if err != nil {
		s.logger.Warn("Failed to extract app metadata", "error", err)
	}
	// Note: ExtractAppMetadata now returns defaults on error, so appMetadata is always valid
	deployedApp.Metadata = appMetadata

	// Store deployed app
	s.mu.Lock()
	s.deployedApps[appID] = deployedApp
	s.mu.Unlock()

	// Update database
	updateQuery := `
		UPDATE osa_module_instances
		SET status = 'running',
		    deployed_at = NOW(),
		    deployment_url = $1,
		    deployment_port = $2
		WHERE id = $3
	`

	_, err = s.pool.Exec(ctx, updateQuery, deployedApp.URL, deployedApp.Port, appID)
	if err != nil {
		s.logger.Warn("Failed to update app status", "error", err)
	}

	return deployedApp, nil
}

// detectAppType determines the type of app from files
func (s *AppDeploymentService) detectAppType(files []BundledFile) string {
	hasPackageJSON := false
	hasRequirements := false
	hasIndexHTML := false

	for _, file := range files {
		fileName := filepath.Base(file.Path)
		switch fileName {
		case "package.json":
			hasPackageJSON = true
		case "requirements.txt":
			hasRequirements = true
		case "index.html":
			hasIndexHTML = true
		}
	}

	if hasPackageJSON {
		return "node"
	}
	if hasRequirements {
		return "python"
	}
	if hasIndexHTML {
		return "static"
	}

	return "unknown"
}

// deployNodeApp installs deps, builds, and starts a Node.js app
func (s *AppDeploymentService) deployNodeApp(ctx context.Context, appID uuid.UUID, name, workflowID, appDir string) (*DeployedApp, error) {
	s.logger.Info("Deploying Node.js app", "dir", appDir)

	// Check package.json for scripts and monorepo structure FIRST
	packageJSONPath := filepath.Join(appDir, "package.json")
	packageData, err := os.ReadFile(packageJSONPath)
	var startScript string = "start"
	var isMonorepo bool = false
	var buildOutput string

	if err == nil {
		var packageJSON map[string]interface{}
		if json.Unmarshal(packageData, &packageJSON) == nil {
			if scripts, ok := packageJSON["scripts"].(map[string]interface{}); ok {
				// Check for monorepo patterns
				if _, hasInstallAll := scripts["install-all"]; hasInstallAll {
					isMonorepo = true
					s.logger.Info("Detected monorepo with install-all script")

					// Run install-all for monorepos
					installAllCmd := exec.Command("npm", "run", "install-all")
					installAllCmd.Dir = appDir
					installAllOutput, err := installAllCmd.CombinedOutput()
					if err != nil {
						s.logger.Warn("install-all failed, continuing anyway", "error", err)
					} else {
						buildOutput = string(installAllOutput)
					}
				}

				// Determine which start script to use
				if _, hasDev := scripts["dev"]; hasDev {
					startScript = "dev"
					// If dev script uses concurrently, it's likely a monorepo
					if devScript, ok := scripts["dev"].(string); ok {
						if strings.Contains(devScript, "concurrently") || strings.Contains(devScript, "cd frontend") {
							isMonorepo = true
						}
					}
				} else if _, hasStart := scripts["start"]; hasStart {
					startScript = "start"
				}

				// Check for build script
				if _, hasBuild := scripts["build"]; hasBuild {
					s.logger.Info("Building app...")
					buildCmd := exec.Command("npm", "run", "build")
					buildCmd.Dir = appDir
					buildCmdOutput, err := buildCmd.CombinedOutput()
					if err != nil {
						return nil, fmt.Errorf("npm build failed: %w\n%s", err, buildCmdOutput)
					}
					buildOutput += "\n" + string(buildCmdOutput)
				}
			}
		}
	}

	// If no install-all but is monorepo, install deps manually
	if !isMonorepo {
		// Regular app - just install at root
		s.logger.Info("Installing dependencies...")
		installCmd := exec.Command("npm", "install")
		installCmd.Dir = appDir
		installOutput, err := installCmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("npm install failed: %w\n%s", err, installOutput)
		}
		buildOutput += string(installOutput)
	} else {
		// Monorepo - install in subdirectories if they exist
		s.logger.Info("Installing monorepo dependencies...")

		// Install at root first
		installCmd := exec.Command("npm", "install")
		installCmd.Dir = appDir
		rootOutput, err := installCmd.CombinedOutput()
		if err != nil {
			s.logger.Warn("Root npm install failed", "error", err)
		} else {
			buildOutput += string(rootOutput)
		}

		// Install in common subdirectories
		for _, subdir := range []string{"frontend", "backend", "server", "client"} {
			subdirPath := filepath.Join(appDir, subdir)
			if _, err := os.Stat(subdirPath); err == nil {
				// Check if package.json exists
				if _, err := os.Stat(filepath.Join(subdirPath, "package.json")); err == nil {
					s.logger.Info("Installing dependencies in subdirectory", "subdir", subdir)
					subdirCmd := exec.Command("npm", "install")
					subdirCmd.Dir = subdirPath
					subdirOutput, err := subdirCmd.CombinedOutput()
					if err != nil {
						s.logger.Warn("Subdirectory npm install failed", "subdir", subdir, "error", err)
					} else {
						buildOutput += "\n" + string(subdirOutput)
					}
				}
			}
		}
	}

	// Allocate port
	port := s.portAllocator.Allocate()

	// Start the app with the appropriate script
	s.logger.Info("Starting app...", "port", port, "script", startScript)
	startCmd := exec.Command("npm", "run", startScript)
	startCmd.Dir = appDir
	startCmd.Env = append(os.Environ(), fmt.Sprintf("PORT=%d", port))

	// Capture output
	var startupOutput strings.Builder
	startCmd.Stdout = &startupOutput
	startCmd.Stderr = &startupOutput

	if err := startCmd.Start(); err != nil {
		s.portAllocator.Release(port)
		return nil, fmt.Errorf("failed to start app: %w", err)
	}

	// Wait a bit for startup
	time.Sleep(2 * time.Second)

	deployedApp := &DeployedApp{
		ID:            appID,
		Name:          name,
		WorkflowID:    workflowID,
		Port:          port,
		URL:           fmt.Sprintf("http://localhost:%d", port),
		ProcessID:     startCmd.Process.Pid,
		Status:        "running",
		DeployedAt:    time.Now(),
		LastHealthy:   time.Now(),
		AppType:       "node",
		RootPath:      appDir,
		BuildOutput:   buildOutput,
		StartupOutput: startupOutput.String(),
		cmd:           startCmd,
	}

	// Monitor process in background
	go s.monitorProcess(deployedApp)

	return deployedApp, nil
}

// deployPythonApp installs deps and starts a Python app
func (s *AppDeploymentService) deployPythonApp(ctx context.Context, appID uuid.UUID, name, workflowID, appDir string) (*DeployedApp, error) {
	s.logger.Info("Deploying Python app", "dir", appDir)

	// Install dependencies
	s.logger.Info("Installing dependencies...")
	installCmd := exec.Command("pip", "install", "-r", "requirements.txt")
	installCmd.Dir = appDir
	installOutput, err := installCmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("pip install failed: %w\n%s", err, installOutput)
	}

	// Allocate port
	port := s.portAllocator.Allocate()

	// Start the app (assuming main.py or app.py)
	mainFile := "main.py"
	if _, err := os.Stat(filepath.Join(appDir, "app.py")); err == nil {
		mainFile = "app.py"
	}

	s.logger.Info("Starting app...", "port", port, "main", mainFile)
	startCmd := exec.Command("python", mainFile)
	startCmd.Dir = appDir
	startCmd.Env = append(os.Environ(), fmt.Sprintf("PORT=%d", port))

	var startupOutput strings.Builder
	startCmd.Stdout = &startupOutput
	startCmd.Stderr = &startupOutput

	if err := startCmd.Start(); err != nil {
		s.portAllocator.Release(port)
		return nil, fmt.Errorf("failed to start app: %w", err)
	}

	time.Sleep(2 * time.Second)

	deployedApp := &DeployedApp{
		ID:            appID,
		Name:          name,
		WorkflowID:    workflowID,
		Port:          port,
		URL:           fmt.Sprintf("http://localhost:%d", port),
		ProcessID:     startCmd.Process.Pid,
		Status:        "running",
		DeployedAt:    time.Now(),
		LastHealthy:   time.Now(),
		AppType:       "python",
		RootPath:      appDir,
		BuildOutput:   string(installOutput),
		StartupOutput: startupOutput.String(),
		cmd:           startCmd,
	}

	go s.monitorProcess(deployedApp)

	return deployedApp, nil
}

// deployStaticApp serves static files
func (s *AppDeploymentService) deployStaticApp(ctx context.Context, appID uuid.UUID, name, workflowID, appDir string) (*DeployedApp, error) {
	s.logger.Info("Deploying static app", "dir", appDir)

	// Allocate port
	port := s.portAllocator.Allocate()

	// Start a simple HTTP server
	s.logger.Info("Starting static server...", "port", port)
	startCmd := exec.Command("python", "-m", "http.server", fmt.Sprintf("%d", port))
	startCmd.Dir = appDir

	var startupOutput strings.Builder
	startCmd.Stdout = &startupOutput
	startCmd.Stderr = &startupOutput

	if err := startCmd.Start(); err != nil {
		s.portAllocator.Release(port)
		return nil, fmt.Errorf("failed to start static server: %w", err)
	}

	time.Sleep(1 * time.Second)

	deployedApp := &DeployedApp{
		ID:            appID,
		Name:          name,
		WorkflowID:    workflowID,
		Port:          port,
		URL:           fmt.Sprintf("http://localhost:%d", port),
		ProcessID:     startCmd.Process.Pid,
		Status:        "running",
		DeployedAt:    time.Now(),
		LastHealthy:   time.Now(),
		AppType:       "static",
		RootPath:      appDir,
		BuildOutput:   "",
		StartupOutput: startupOutput.String(),
		cmd:           startCmd,
	}

	go s.monitorProcess(deployedApp)

	return deployedApp, nil
}

// monitorProcess watches the app process and updates status
func (s *AppDeploymentService) monitorProcess(app *DeployedApp) {
	if app.cmd == nil {
		return
	}

	// Wait for process to finish
	err := app.cmd.Wait()

	s.mu.Lock()
	defer s.mu.Unlock()

	if err != nil {
		s.logger.Warn("App process exited with error",
			"app_id", app.ID,
			"name", app.Name,
			"error", err)
		app.Status = "crashed"
	} else {
		s.logger.Info("App process exited normally",
			"app_id", app.ID,
			"name", app.Name)
		app.Status = "stopped"
	}

	// Release port
	s.portAllocator.Release(app.Port)

	// Update database
	ctx := context.Background()
	updateQuery := `
		UPDATE osa_module_instances
		SET status = $1
		WHERE id = $2
	`
	s.pool.Exec(ctx, updateQuery, app.Status, app.ID)
}

// StopApp stops a running application
func (s *AppDeploymentService) StopApp(appID uuid.UUID) error {
	s.mu.Lock()
	app, exists := s.deployedApps[appID]
	s.mu.Unlock()

	if !exists {
		return fmt.Errorf("app not found")
	}

	if app.cmd != nil && app.cmd.Process != nil {
		s.logger.Info("Stopping app", "app_id", appID, "pid", app.ProcessID)
		if err := app.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill process: %w", err)
		}
	}

	// Clean up directory
	if err := os.RemoveAll(app.RootPath); err != nil {
		s.logger.Warn("Failed to remove app directory", "dir", app.RootPath, "error", err)
	}

	// Remove from map
	s.mu.Lock()
	delete(s.deployedApps, appID)
	s.mu.Unlock()

	return nil
}

// GetDeployedApp retrieves info about a deployed app
func (s *AppDeploymentService) GetDeployedApp(appID uuid.UUID) (*DeployedApp, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	app, exists := s.deployedApps[appID]
	return app, exists
}

// ListDeployedApps returns all running apps
func (s *AppDeploymentService) ListDeployedApps() []*DeployedApp {
	s.mu.RLock()
	defer s.mu.RUnlock()

	apps := make([]*DeployedApp, 0, len(s.deployedApps))
	for _, app := range s.deployedApps {
		apps = append(apps, app)
	}
	return apps
}
