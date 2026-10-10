export interface AppShellBootstrapDependencies {
	initCSRF: () => Promise<void>;
	loadSavedWorkspace: () => Promise<void>;
	loadProjects: () => Promise<void>;
}

export async function initializeAppShellContext(
	isEmbedMode: boolean,
	dependencies: AppShellBootstrapDependencies
): Promise<'embedded' | 'shell'> {
	await dependencies.initCSRF();

	if (isEmbedMode) {
		await dependencies.loadSavedWorkspace();
		return 'embedded';
	}

	await Promise.all([
		dependencies.loadSavedWorkspace(),
		dependencies.loadProjects()
	]);
	return 'shell';
}
