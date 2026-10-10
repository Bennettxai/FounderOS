package handlers

import (
	"github.com/rhl/businessos-backend/internal/cache"
	"github.com/rhl/businessos-backend/internal/services"
)

// SetWebPushService sets the Web Push service (optional)
func (h *Handlers) SetWebPushService(svc *services.WebPushService) {
	h.webPushService = svc
	// Also set on dispatcher if available
	if h.notificationService != nil && svc != nil {
		h.notificationService.Dispatcher().SetWebPushService(svc)
	}
}

// SetEmailService sets the Email service (optional)
func (h *Handlers) SetEmailService(svc *services.EmailService) {
	h.emailService = svc
	// Also set on dispatcher if available
	if h.notificationService != nil && svc != nil {
		h.notificationService.Dispatcher().SetEmailService(svc)
	}
}

// SetCommentService sets the Comment service (optional)
func (h *Handlers) SetCommentService(svc *services.CommentService) {
	h.commentService = svc
}

// SetVoiceServices sets the voice services (3D Desktop - Whisper + ElevenLabs)
func (h *Handlers) SetVoiceServices(whisper *services.WhisperService, elevenLabs *services.ElevenLabsService) {
	h.whisperService = whisper
	h.elevenLabsService = elevenLabs
}

// SetWorkspaceService sets the workspace service (Feature 1)
func (h *Handlers) SetWorkspaceService(workspaceService *services.WorkspaceService) {
	h.workspaceService = workspaceService
}

// SetWorkspaceVersionService sets the workspace version service (Feature 1 - Versioning)
func (h *Handlers) SetWorkspaceVersionService(workspaceVersionService *services.WorkspaceVersionService) {
	h.workspaceVersionService = workspaceVersionService
}

// SetRoleContextService sets the role context service (Feature 1 - Permissions)
func (h *Handlers) SetRoleContextService(roleContextService *services.RoleContextService) {
	h.roleContextService = roleContextService
}

// SetMemoryHierarchyService sets the memory hierarchy service (Q1 - Memory Hierarchy)
func (h *Handlers) SetMemoryHierarchyService(memoryHierarchyService *services.MemoryHierarchyService) {
	h.memoryHierarchyService = memoryHierarchyService
}

// SetInviteService sets the workspace invite service (Feature 1 - Email Invites)
func (h *Handlers) SetInviteService(inviteService *services.WorkspaceInviteService) {
	h.inviteService = inviteService
}

// SetAuditService sets the workspace audit service (Feature 1 - Audit Logging)
func (h *Handlers) SetAuditService(auditService *services.WorkspaceAuditService) {
	h.auditService = auditService
}

// SetProjectAccessService sets the project access service (Feature 1 - Project Access Control)
func (h *Handlers) SetProjectAccessService(projectAccessService *services.ProjectAccessService) {
	h.projectAccessService = projectAccessService
}

// SetQueryCache sets the query cache service (optional, for Redis-based caching)
func (h *Handlers) SetQueryCache(queryCache *cache.QueryCache) {
	h.queryCache = queryCache
}

// SetSandboxHandler sets the sandbox deployment handler
func (h *Handlers) SetSandboxHandler(sandboxHandler *SandboxHandler) {
	h.sandboxHandler = sandboxHandler
}

// SetSandboxEditHandler sets the sandbox module edit lifecycle handler
func (h *Handlers) SetSandboxEditHandler(sandboxEditHandler *SandboxEditHandler) {
	h.sandboxEditHandler = sandboxEditHandler
}
