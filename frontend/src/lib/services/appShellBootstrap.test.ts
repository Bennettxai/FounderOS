import { describe, expect, it, vi } from 'vitest';
import { initializeAppShellContext } from './appShellBootstrap';

describe('initializeAppShellContext', () => {
	it('restores the active workspace for embedded desktop modules', async () => {
		const initCSRF = vi.fn().mockResolvedValue(undefined);
		const loadSavedWorkspace = vi.fn().mockResolvedValue(undefined);
		const loadProjects = vi.fn().mockResolvedValue(undefined);

		const mode = await initializeAppShellContext(true, {
			initCSRF,
			loadSavedWorkspace,
			loadProjects
		});

		expect(mode).toBe('embedded');
		expect(initCSRF).toHaveBeenCalledOnce();
		expect(loadSavedWorkspace).toHaveBeenCalledOnce();
		expect(loadProjects).not.toHaveBeenCalled();
	});

	it('initializes the workspace and shell data for the full application', async () => {
		const initCSRF = vi.fn().mockResolvedValue(undefined);
		const loadSavedWorkspace = vi.fn().mockResolvedValue(undefined);
		const loadProjects = vi.fn().mockResolvedValue(undefined);

		const mode = await initializeAppShellContext(false, {
			initCSRF,
			loadSavedWorkspace,
			loadProjects
		});

		expect(mode).toBe('shell');
		expect(initCSRF).toHaveBeenCalledOnce();
		expect(loadSavedWorkspace).toHaveBeenCalledOnce();
		expect(loadProjects).toHaveBeenCalledOnce();
	});
});
