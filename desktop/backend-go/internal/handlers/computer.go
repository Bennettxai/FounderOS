package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/config"
	"github.com/rhl/businessos-backend/internal/integrations/miosa"
	"github.com/rhl/businessos-backend/internal/utils"
)

// ComputerHandler serves cloud computer status and lifecycle endpoints.
// Cloud operations must go through the MIOSA runtime API. Handlers avoid
// fabricated success responses so production callers can detect bad setup.
type ComputerHandler struct {
	cfg         *config.Config
	miosaClient *miosa.ComputeClient
}

// NewComputerHandler constructs a ComputerHandler.
func NewComputerHandler(cfg *config.Config, miosaClient *miosa.ComputeClient) *ComputerHandler {
	return &ComputerHandler{cfg: cfg, miosaClient: miosaClient}
}

// ── request types ─────────────────────────────────────────────────────────────

type createComputerRequest struct {
	Plan string `json:"plan" binding:"required"`
}

type upgradeComputerRequest struct {
	Plan string `json:"plan" binding:"required"`
}

type runtimeActionRequest struct {
	// name comes from the URL param; body is intentionally empty for now
}

// planSpecs returns the resource allocation for a plan, reusing PlanInfo.
func planSpecs(plan string) PlanInfo {
	for _, p := range availablePlans {
		if p.ID == plan {
			return p
		}
	}
	// fallback for unknown plans — treat as pro
	return availablePlans[0]
}

// planToMIOSASize maps a BusinessOS plan name to the MIOSA VM size string.
func planToMIOSASize(plan string) string {
	switch plan {
	case "growth":
		return "medium"
	case "business":
		return "large"
	default: // "pro" and anything else
		return "small"
	}
}

// mapMIOSAComputer converts a MIOSA API computer to the BusinessOS Computer type.
// mapMIOSAStatus normalizes MIOSA status strings to what the frontend expects.
func mapMIOSAStatus(status string) string {
	switch status {
	case "active", "running":
		return "running"
	case "provisioning", "creating":
		return "provisioning"
	case "stopped", "paused":
		return "stopped"
	case "hibernating":
		return "hibernating"
	default:
		return status
	}
}

func mapMIOSAComputer(mc miosa.MIOSAComputer, plan string) Computer {
	specs := planSpecs(plan)
	now := time.Now().UTC()

	// Parse timestamps — zero value on parse failure is acceptable.
	createdAt, _ := time.Parse(time.RFC3339, mc.InsertedAt)
	if createdAt.IsZero() {
		createdAt, _ = time.Parse(time.RFC3339, mc.CreatedAt)
	}
	updatedAt, _ := time.Parse(time.RFC3339, mc.UpdatedAt)
	if createdAt.IsZero() {
		createdAt = now
	}
	if updatedAt.IsZero() {
		updatedAt = now
	}

	// Use real specs from MIOSA if available, otherwise fall back to plan specs
	ram := specs.RAMGB
	if mc.MemoryMB > 0 {
		ram = mc.MemoryMB / 1024
		if ram == 0 {
			ram = 1
		}
	}
	cpus := specs.CPUs
	if mc.VCPUs > 0 {
		cpus = mc.VCPUs
	}
	storageGB := specs.StorageGB
	if mc.DiskGB > 0 {
		storageGB = mc.DiskGB
	}

	domain := mc.DesktopURL
	if domain == "" && mc.Slug != "" {
		domain = "https://" + mc.Slug + ".sandbox.miosa.ai"
	}

	return Computer{
		ID:          mc.ID,
		Status:      mapMIOSAStatus(mc.Status),
		Plan:        plan,
		RAM:         ram,
		CPUs:        cpus,
		StorageGB:   storageGB,
		StorageUsed: 0,
		Domain:      domain,
		Runtimes:    []Runtime{},
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
}

// ── handlers ──────────────────────────────────────────────────────────────────

// GetComputer handles GET /api/computer.
func (h *ComputerHandler) GetComputer(c *gin.Context) {
	if !h.cfg.IsCloudDeployment() {
		// Try real MIOSA even in non-cloud-deployment mode — the key may still be set.
		if h.miosaClient != nil {
			computers, err := h.miosaClient.ListComputers(c.Request.Context())
			if err == nil && len(computers) > 0 {
				// Prefer the BusinessOS template, then active, then most recent
				best := computers[0]
				for _, mc := range computers {
					if mc.TemplateType == "business_os" {
						best = mc
						break
					}
					if (mc.Status == "active" || mc.Status == "running") && best.Status != "active" {
						best = mc
					}
				}
				comp := mapMIOSAComputer(best, "pro")
				c.JSON(http.StatusOK, gin.H{"computer": comp})
				return
			}
			if err != nil {
				slog.WarnContext(c.Request.Context(), "miosa: list computers failed", "error", err)
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"computer": nil,
			"message":  "No cloud computer. Running locally.",
		})
		return
	}

	if h.miosaClient != nil {
		computers, err := h.miosaClient.ListComputers(c.Request.Context())
		if err == nil && len(computers) > 0 {
			best := computers[0]
			for _, mc := range computers {
				if mc.TemplateType == "business_os" {
					best = mc
					break
				}
			}
			comp := mapMIOSAComputer(best, "pro")
			c.JSON(http.StatusOK, gin.H{"computer": comp})
			return
		}
		if err != nil {
			slog.WarnContext(c.Request.Context(), "miosa: list computers failed", "error", err)
			c.JSON(http.StatusBadGateway, gin.H{"error": "MIOSA compute API unavailable", "details": err.Error()})
			return
		}
	}

	c.JSON(http.StatusServiceUnavailable, gin.H{"error": "MIOSA compute API is not configured"})
}

// CreateComputer handles POST /api/computer.
func (h *ComputerHandler) CreateComputer(c *gin.Context) {
	var req createComputerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	slog.InfoContext(c.Request.Context(), "computer: create requested", "plan", req.Plan)

	if h.miosaClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "MIOSA compute API is not configured"})
		return
	}

	size := planToMIOSASize(req.Plan)
	mc, err := h.miosaClient.CreateComputer(c.Request.Context(), "BusinessOS", "business_os", size)
	if err != nil {
		slog.WarnContext(c.Request.Context(), "miosa: create computer failed",
			"error", err, "plan", req.Plan)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to create MIOSA computer", "details": err.Error()})
		return
	}
	comp := mapMIOSAComputer(*mc, req.Plan)

	c.JSON(http.StatusAccepted, gin.H{
		"computer": comp,
		"status":   "provisioning",
		"message":  "Your computer is being created. This takes about 60 seconds.",
	})
}

// UpgradeComputer handles PUT /api/computer/upgrade.
// Accepts {"plan": "growth"} and returns an updated computer.
func (h *ComputerHandler) UpgradeComputer(c *gin.Context) {
	var req upgradeComputerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	slog.InfoContext(c.Request.Context(), "computer: upgrade requested", "plan", req.Plan)
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "MIOSA compute plan upgrades are not wired in this runtime yet",
		"plan":  req.Plan,
	})
}

// DeleteComputer handles DELETE /api/computer.
func (h *ComputerHandler) DeleteComputer(c *gin.Context) {
	slog.InfoContext(c.Request.Context(), "computer: terminate requested")

	computerID := c.Query("id")
	if computerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "computer id required"})
		return
	}
	if h.miosaClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "MIOSA compute API is not configured"})
		return
	}
	if err := h.miosaClient.DeleteComputer(c.Request.Context(), computerID); err != nil {
		slog.WarnContext(c.Request.Context(), "miosa: delete computer failed",
			"error", err, "computer_id", computerID)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to delete MIOSA computer", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "terminated"})
}

// vmMetricsResult holds parsed metrics from inside the VM.
type vmMetricsResult struct {
	CPUPercent   float64
	RAMUsedMB    int
	RAMTotalMB   int
	DiskUsedGB   int
	DiskTotalGB  int
	UptimeSec    int64
	ProcessCount int
}

// metricsCommand is a single shell one-liner that outputs all metrics on one line.
// Output format: CPU:<pct> MEM:<used_mb> <total_mb> DISK:<used_gb> <total_gb> UP:<seconds> PROC:<count>
const metricsCommand = `echo "CPU:$(top -bn1 2>/dev/null | grep -i 'cpu(s)\|%cpu' | head -1 | awk '{for(i=1;i<=NF;i++){if($i~/^[0-9]/ && $(i-1)~/us/){print $i;exit}; if(i==1 && $i~/^[0-9]/){print $i;exit}}}' | tr -d '%,') MEM:$(free -m 2>/dev/null | awk '/Mem:/{print $3,$2}') DISK:$(df -BG / 2>/dev/null | awk 'NR==2{gsub("G","");print $3,$2}') UP:$(awk '{print int($1)}' /proc/uptime 2>/dev/null) PROC:$(ps aux 2>/dev/null | wc -l)"`

// Compiled regexes for parsing metricsCommand output.
// MEM and DISK each have two space-separated numbers after the colon.
var (
	reMetricsCPU  = regexp.MustCompile(`CPU:([\d.]+)`)
	reMetricsMEM  = regexp.MustCompile(`MEM:(\d+)\s+(\d+)`)
	reMetricsDISK = regexp.MustCompile(`DISK:(\d+)\s+(\d+)`)
	reMetricsUP   = regexp.MustCompile(`UP:(\d+)`)
	reMetricsPROC = regexp.MustCompile(`PROC:(\d+)`)
)

// parseVMMetrics parses the one-liner output from metricsCommand.
// Expected format: CPU:1.5 MEM:512 2048 DISK:3 20 UP:3600 PROC:43
// Any field that fails to parse is left at its zero value.
func parseVMMetrics(output string) vmMetricsResult {
	var m vmMetricsResult
	output = strings.TrimSpace(output)
	if output == "" {
		return m
	}

	if s := reMetricsCPU.FindStringSubmatch(output); len(s) == 2 {
		m.CPUPercent, _ = strconv.ParseFloat(s[1], 64)
	}
	if s := reMetricsMEM.FindStringSubmatch(output); len(s) == 3 {
		m.RAMUsedMB, _ = strconv.Atoi(s[1])
		m.RAMTotalMB, _ = strconv.Atoi(s[2])
	}
	if s := reMetricsDISK.FindStringSubmatch(output); len(s) == 3 {
		m.DiskUsedGB, _ = strconv.Atoi(s[1])
		m.DiskTotalGB, _ = strconv.Atoi(s[2])
	}
	if s := reMetricsUP.FindStringSubmatch(output); len(s) == 2 {
		m.UptimeSec, _ = strconv.ParseInt(s[1], 10, 64)
	}
	if s := reMetricsPROC.FindStringSubmatch(output); len(s) == 2 {
		if v, _ := strconv.Atoi(s[1]); v > 1 {
			m.ProcessCount = v - 1 // ps aux header line
		}
	}
	return m
}

// GetMetrics handles GET /api/computer/metrics.
// Fetches real resource usage from inside the VM via envd exec through the sandbox
// proxy. Falls back to MIOSA spec totals when VM exec is unreachable.
func (h *ComputerHandler) GetMetrics(c *gin.Context) {
	if h.miosaClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "MIOSA compute API is not configured"})
		return
	}

	// Step 1 — find the active computer.
	computers, err := h.miosaClient.ListComputers(c.Request.Context())
	if err != nil || len(computers) == 0 {
		if err != nil {
			slog.WarnContext(c.Request.Context(), "metrics: list computers failed", "error", err)
			c.JSON(http.StatusBadGateway, gin.H{"error": "MIOSA compute API unavailable", "details": err.Error()})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "No MIOSA computer found"})
		return
	}

	// Prefer an active computer; any computer as fallback.
	mc := computers[0]
	for _, comp := range computers {
		if comp.Status == "active" || comp.Status == "running" {
			mc = comp
			break
		}
	}

	// Fill in spec totals from MIOSA — these are always available even if exec fails.
	totalRAM := mc.MemoryMB
	if totalRAM == 0 {
		totalRAM = 4096
	}
	totalCPUs := mc.VCPUs
	if totalCPUs == 0 {
		totalCPUs = 2
	}
	totalDisk := mc.DiskGB
	if totalDisk == 0 {
		totalDisk = 10
	}

	// Step 2 — exec metrics command inside the VM via envd.
	// Only attempt this when the VM is actually running.
	if (mc.Status == "active" || mc.Status == "running") && mc.Slug != "" {
		output, execErr := h.miosaClient.ExecInVM(c.Request.Context(), mc.Slug, metricsCommand)
		if execErr != nil {
			slog.WarnContext(c.Request.Context(), "metrics: exec in vm failed, using spec-only data",
				"error", execErr, "slug", mc.Slug)
		} else {
			vm := parseVMMetrics(output)

			// Sanity-check parsed totals against MIOSA spec — prefer whichever is non-zero.
			ramTotal := vm.RAMTotalMB
			if ramTotal == 0 {
				ramTotal = totalRAM
			}
			diskTotal := vm.DiskTotalGB
			if diskTotal == 0 {
				diskTotal = totalDisk
			}

			slog.InfoContext(c.Request.Context(), "metrics: real VM data",
				"cpu_pct", fmt.Sprintf("%.1f", vm.CPUPercent),
				"ram_used_mb", vm.RAMUsedMB,
				"ram_total_mb", ramTotal,
				"disk_used_gb", vm.DiskUsedGB,
				"uptime_sec", vm.UptimeSec,
			)

			c.JSON(http.StatusOK, gin.H{
				"cpu_percent":      vm.CPUPercent,
				"ram_used_mb":      vm.RAMUsedMB,
				"ram_total_mb":     ramTotal,
				"cpu_cores":        totalCPUs,
				"storage_used_gb":  vm.DiskUsedGB,
				"storage_total_gb": diskTotal,
				"network_in_mb":    0,
				"network_out_mb":   0,
				"uptime_seconds":   vm.UptimeSec,
				"process_count":    vm.ProcessCount,
				"source":           "envd",
			})
			return
		}
	}

	// Step 3 — VM not running or exec failed: return MIOSA spec totals, zero usage.
	// Calculate uptime from creation time as best-effort.
	var uptimeSec int64
	if created, parseErr := time.Parse(time.RFC3339, mc.InsertedAt); parseErr == nil {
		uptimeSec = int64(time.Since(created).Seconds())
	}

	c.JSON(http.StatusOK, gin.H{
		"cpu_percent":      0,
		"ram_used_mb":      0,
		"ram_total_mb":     totalRAM,
		"cpu_cores":        totalCPUs,
		"storage_used_gb":  0,
		"storage_total_gb": totalDisk,
		"network_in_mb":    0,
		"network_out_mb":   0,
		"uptime_seconds":   uptimeSec,
		"source":           "miosa",
	})
}

// GetRuntimes handles GET /api/computer/runtimes.
// Queries the MIOSA computer's selected_apps for runtime info.
func (h *ComputerHandler) GetRuntimes(c *gin.Context) {
	if h.miosaClient != nil {
		computers, err := h.miosaClient.ListComputers(c.Request.Context())
		if err == nil && len(computers) > 0 {
			mc := computers[0]
			var runtimes []Runtime
			for _, app := range mc.SelectedApps {
				status := "stopped"
				if mc.Status == "active" {
					status = "active"
				}
				runtimes = append(runtimes, Runtime{
					Name:     app,
					Status:   status,
					MemoryMB: 0,
					Uptime:   0,
				})
			}
			// Always include the base system
			if mc.Status == "active" {
				runtimes = append(runtimes, Runtime{
					Name:     "envd",
					Status:   "active",
					MemoryMB: 0,
					Uptime:   0,
				})
			}
			c.JSON(http.StatusOK, gin.H{
				"runtimes": runtimes,
				"count":    len(runtimes),
				"source":   "miosa",
			})
			return
		}
		if err != nil {
			slog.WarnContext(c.Request.Context(), "miosa: list runtimes failed", "error", err)
			c.JSON(http.StatusBadGateway, gin.H{"error": "MIOSA compute API unavailable", "details": err.Error()})
			return
		}
	}
	if h.miosaClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "MIOSA compute API is not configured"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"runtimes": []Runtime{},
		"count":    0,
		"source":   "miosa",
	})
}

// StartRuntime handles POST /api/computer/runtimes/:name/start.
func (h *ComputerHandler) StartRuntime(c *gin.Context) {
	name := c.Param("name")
	slog.InfoContext(c.Request.Context(), "computer: runtime start requested", "name", name)
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "Runtime start is not available until MIOSA exposes per-runtime lifecycle APIs",
		"name":  name,
	})
}

// GetTerminalSession handles GET /api/computer/terminal-session.
// Creates a PTY session on the MIOSA VM and returns the WebSocket URL + auth token.
// If the computer is stopped/paused, it auto-wakes it and waits for envd to be ready.
func (h *ComputerHandler) GetTerminalSession(c *gin.Context) {
	if h.miosaClient == nil {
		c.JSON(http.StatusOK, gin.H{
			"mode":    "local",
			"message": "No cloud computer — use local terminal.",
		})
		return
	}

	// Get the most recent computer (prefer active, then any)
	computers, err := h.miosaClient.ListComputers(c.Request.Context())
	if err != nil || len(computers) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"mode":    "local",
			"message": "No computer found.",
		})
		return
	}

	// Find the best computer: prefer active, then provisioning, then any
	comp := computers[0]
	for _, mc := range computers {
		if mc.Status == "active" || mc.Status == "running" {
			comp = mc
			break
		}
	}

	// Auto-wake if stopped/paused
	if comp.Status == "stopped" || comp.Status == "paused" {
		slog.InfoContext(c.Request.Context(), "miosa: auto-waking computer for terminal", "id", comp.ID, "status", comp.Status)

		if startErr := h.miosaClient.StartComputer(c.Request.Context(), comp.ID); startErr != nil {
			slog.WarnContext(c.Request.Context(), "miosa: auto-wake failed, trying to create terminal anyway", "error", startErr)
		}

		// Poll for the computer to become active (max 60s, check every 3s)
		maxAttempts := 20
		for i := 0; i < maxAttempts; i++ {
			time.Sleep(3 * time.Second)
			updated, pollErr := h.miosaClient.GetComputer(c.Request.Context(), comp.ID)
			if pollErr != nil {
				continue
			}
			if updated.Status == "active" || updated.Status == "running" {
				comp = *updated
				slog.InfoContext(c.Request.Context(), "miosa: computer is now active", "id", comp.ID, "attempt", i+1)
				break
			}
			slog.InfoContext(c.Request.Context(), "miosa: waiting for computer to wake", "id", comp.ID, "status", updated.Status, "attempt", i+1)
		}
	}

	if comp.Status != "active" && comp.Status != "running" {
		c.JSON(http.StatusOK, gin.H{
			"mode":    "waking",
			"message": "Computer is " + comp.Status + " — it may take up to 90 seconds to start.",
		})
		return
	}

	// Create terminal session on the VM — retry up to 3 times (envd may still be booting)
	var session *miosa.TerminalSession
	var termErr error
	for attempt := 0; attempt < 3; attempt++ {
		session, termErr = h.miosaClient.CreateTerminalSession(c.Request.Context(), comp.ID, 120, 30)
		if termErr == nil && session != nil && session.SessionID != "" {
			break
		}
		slog.WarnContext(c.Request.Context(), "miosa: terminal session attempt failed, retrying",
			"attempt", attempt+1, "error", termErr)
		time.Sleep(3 * time.Second)
	}

	if termErr != nil || session == nil || session.SessionID == "" {
		errMsg := "envd not responding"
		if termErr != nil {
			errMsg = termErr.Error()
		}
		slog.WarnContext(c.Request.Context(), "miosa: create terminal session failed after retries", "error", errMsg)
		c.JSON(http.StatusOK, gin.H{
			"mode":    "local",
			"message": "Cloud terminal not ready: " + errMsg,
		})
		return
	}

	slug := comp.Slug
	if slug == "" && len(comp.ID) >= 8 {
		slug = comp.ID[:8]
	}

	wsURL := "wss://" + slug + ".sandbox.miosa.ai/ws/terminal/" + comp.ID + "/" + session.SessionID
	if session.StreamAuth != "" {
		wsURL += "?auth=" + session.StreamAuth
	}

	c.JSON(http.StatusOK, gin.H{
		"mode":        "cloud",
		"ws_url":      wsURL,
		"session_id":  session.SessionID,
		"computer_id": comp.ID,
		"slug":        slug,
		"expires_at":  session.ExpiresAt,
	})
}

// GetDesktopStream handles GET /api/computer/desktop-stream.
// Returns the desktop iframe URL with a VNC stream auth token.
func (h *ComputerHandler) GetDesktopStream(c *gin.Context) {
	if h.miosaClient == nil {
		c.JSON(http.StatusOK, gin.H{"mode": "local", "message": "No cloud computer."})
		return
	}

	computers, err := h.miosaClient.ListComputers(c.Request.Context())
	if err != nil || len(computers) == 0 {
		c.JSON(http.StatusOK, gin.H{"mode": "local", "message": "No computer found."})
		return
	}

	// Find best computer
	comp := computers[0]
	for _, mc := range computers {
		if mc.TemplateType == "business_os" {
			comp = mc
			break
		}
	}

	if comp.Status != "active" && comp.Status != "running" {
		c.JSON(http.StatusOK, gin.H{"mode": "local", "message": "Computer is " + comp.Status})
		return
	}

	// Get stream token for VNC
	streamToken, err := h.miosaClient.GetStreamToken(c.Request.Context(), comp.ID)
	if err != nil {
		slog.WarnContext(c.Request.Context(), "miosa: get stream token failed", "error", err)
		c.JSON(http.StatusOK, gin.H{"mode": "local", "message": "Failed to get stream token."})
		return
	}

	slug := comp.Slug
	if slug == "" && len(comp.ID) >= 8 {
		slug = comp.ID[:8]
	}

	host := slug + ".sandbox.miosa.ai"
	relayPath := "vnc/websockify?auth=" + streamToken.Token
	desktopURL := "https://" + host + "/desktop/index.html?autoconnect=1&host=" + host + "&encrypt=1&path=" + relayPath + "&resize=remote&show_control_bar=0&view_only=0&reconnect=1&password=&shared=1"

	c.JSON(http.StatusOK, gin.H{
		"mode":        "cloud",
		"desktop_url": desktopURL,
		"slug":        slug,
		"computer_id": comp.ID,
		"expires_at":  streamToken.ExpiresAt,
	})
}

// StopRuntime handles POST /api/computer/runtimes/:name/stop.
func (h *ComputerHandler) StopRuntime(c *gin.Context) {
	name := c.Param("name")
	slog.InfoContext(c.Request.Context(), "computer: runtime stop requested", "name", name)
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "Runtime stop is not available until MIOSA exposes per-runtime lifecycle APIs",
		"name":  name,
	})
}
