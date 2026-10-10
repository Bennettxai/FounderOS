/**
 * Sandbox Edit API Client
 *
 * Routes to backend: /api/v1/sandbox/edit/*
 * Built against Pedro's actual handler: sandbox_edit.go (pedro/sprint-03b-cherrypick)
 *
 * Backend route reference:
 *   POST   /api/v1/sandbox/edit/fork             → Fork
 *   GET    /api/v1/sandbox/edit/:id              → Get
 *   PUT    /api/v1/sandbox/edit/:id/files/:name  → UpdateFile
 *   POST   /api/v1/sandbox/edit/:id/validate     → Validate
 *   GET    /api/v1/sandbox/edit/:id/preview      → Preview
 *   POST   /api/v1/sandbox/edit/:id/apply        → Apply
 *   POST   /api/v1/sandbox/edit/:id/reject       → Reject
 */
import { request } from '$lib/api/base';

// ============================================================================
// Types (matching Pedro's sandbox_edit.go handler responses)
// ============================================================================

export type SandboxEditState = 'pending' | 'validated' | 'applied' | 'rejected';

export interface DiffEntry {
	filename: string;
	lines_added: number;
	lines_removed: number;
	diff: string;
}

export interface SandboxEdit {
	id: string;
	tenant_id: string;
	user_id: string;
	module_id: string;
	module_name: string;
	state: SandboxEditState;
	files: Record<string, string>;
	orig_files: Record<string, string>;
	diff?: DiffEntry[];
	errors?: string[];
	created_at: string;
	updated_at: string;
}

/** All endpoints return { sandbox: SandboxEdit } except UpdateFile */
interface SandboxEditResponse {
	sandbox: SandboxEdit;
}

// ============================================================================
// API Functions
// ============================================================================

/**
 * Fork a module into a sandbox for editing
 * POST /api/v1/sandbox/edit/fork
 */
export async function forkModule(
	moduleId: string,
	moduleName: string
): Promise<SandboxEdit> {
	const data = await request<SandboxEditResponse>('/sandbox/edit/fork', {
		method: 'POST',
		body: { module_id: moduleId, module_name: moduleName }
	});
	return data.sandbox;
}

/**
 * Get sandbox edit state
 * GET /api/v1/sandbox/edit/:id
 */
export async function getSandboxEdit(sandboxId: string): Promise<SandboxEdit> {
	const data = await request<SandboxEditResponse>(`/sandbox/edit/${sandboxId}`);
	return data.sandbox;
}

/**
 * Update a file in the sandbox
 * PUT /api/v1/sandbox/edit/:id/files/:name
 */
export async function updateFile(
	sandboxId: string,
	filename: string,
	content: string
): Promise<void> {
	const encodedName = encodeURIComponent(filename);
	await request<{ ok: boolean }>(`/sandbox/edit/${sandboxId}/files/${encodedName}`, {
		method: 'PUT',
		body: { content }
	});
}

/**
 * Validate all files in the sandbox
 * POST /api/v1/sandbox/edit/:id/validate
 * Returns sandbox with state "validated" on success, or "pending" with errors[] on failure
 */
export async function validateSandbox(sandboxId: string): Promise<SandboxEdit> {
	const data = await request<SandboxEditResponse>(`/sandbox/edit/${sandboxId}/validate`, {
		method: 'POST'
	});
	return data.sandbox;
}

/**
 * Get diff preview of proposed changes
 * GET /api/v1/sandbox/edit/:id/preview
 * Returns sandbox with diff[] populated
 */
export async function getPreview(sandboxId: string): Promise<SandboxEdit> {
	const data = await request<SandboxEditResponse>(`/sandbox/edit/${sandboxId}/preview`);
	return data.sandbox;
}

/**
 * Apply validated changes
 * POST /api/v1/sandbox/edit/:id/apply
 * Requires state == "validated"
 */
export async function applyChanges(sandboxId: string): Promise<SandboxEdit> {
	const data = await request<SandboxEditResponse>(`/sandbox/edit/${sandboxId}/apply`, {
		method: 'POST'
	});
	return data.sandbox;
}

/**
 * Reject/discard sandbox changes (works from any state)
 * POST /api/v1/sandbox/edit/:id/reject
 */
export async function rejectChanges(sandboxId: string): Promise<SandboxEdit> {
	const data = await request<SandboxEditResponse>(`/sandbox/edit/${sandboxId}/reject`, {
		method: 'POST'
	});
	return data.sandbox;
}
