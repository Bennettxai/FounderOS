import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { get } from 'svelte/store';
import {
	deployedAppsStore,
	getCategoryIconName,
	getCategoryColors,
	type DeployedApp
} from './deployedAppsStore';

// Mock fetch globally
global.fetch = vi.fn();

// Mock windowStore
vi.mock('./windowStore', () => ({
	windowStore: {
		registerDeployedApp: vi.fn()
	}
}));

// Mock browser environment
vi.mock('$app/environment', () => ({
	browser: true
}));

describe('deployedAppsStore', () => {
	const mockApps: DeployedApp[] = [
		{
			id: 'app1',
			name: 'Finance App',
			url: 'http://localhost:3001',
			port: 3001,
			status: 'running',
			deployedAt: '2024-01-01T00:00:00Z',
			metadata: {
				name: 'Finance App',
				description: 'Manage your finances',
				category: 'finance',
				icon: 'DollarSign',
				keywords: ['finance', 'money']
			}
		},
		{
			id: 'app2',
			name: 'Chat App',
			url: 'http://localhost:3002',
			port: 3002,
			status: 'running',
			deployedAt: '2024-01-02T00:00:00Z',
			metadata: {
				name: 'Chat App',
				description: 'Team communication',
				category: 'communication',
				icon: 'MessageSquare',
				keywords: ['chat', 'messaging']
			}
		},
		{
			id: 'app3',
			name: 'Crashed App',
			url: 'http://localhost:3003',
			port: 3003,
			status: 'crashed'
		}
	];

	beforeEach(() => {
		vi.clearAllMocks();
		// Reset fetch mock
		(global.fetch as any).mockReset();
	});

	afterEach(() => {
		// Stop any ongoing discovery
		deployedAppsStore.stopDiscovery();
	});

	describe('Initial State', () => {
		it('should have empty apps array initially', () => {
			const state = get(deployedAppsStore);
			expect(state.apps).toEqual([]);
		});

		it('should not be loading initially', () => {
			const state = get(deployedAppsStore);
			expect(state.loading).toBe(false);
		});

		it('should have no error initially', () => {
			const state = get(deployedAppsStore);
			expect(state.error).toBeNull();
		});
	});

	describe('refresh()', () => {
		it('should fetch deployed apps successfully', async () => {
			(global.fetch as any).mockResolvedValueOnce({
				ok: true,
				json: async () => ({ apps: mockApps.slice(0, 2) })
			});

			await deployedAppsStore.refresh();

			const state = get(deployedAppsStore);
			expect(state.apps).toHaveLength(2);
			expect(state.apps[0].name).toBe('Finance App');
			expect(state.loading).toBe(false);
			expect(state.error).toBeNull();
		});

		it('should set loading state during fetch', async () => {
			(global.fetch as any).mockImplementation(
				() =>
					new Promise((resolve) =>
						setTimeout(
							() =>
								resolve({
									ok: true,
									json: async () => ({ apps: mockApps })
								}),
							100
						)
					)
			);

			const refreshPromise = deployedAppsStore.refresh();

			// Check loading state immediately
			const loadingState = get(deployedAppsStore);
			expect(loadingState.loading).toBe(true);

			await refreshPromise;

			const finalState = get(deployedAppsStore);
			expect(finalState.loading).toBe(false);
		});

		it('should handle fetch error', async () => {
			(global.fetch as any).mockResolvedValueOnce({
				ok: false,
				statusText: 'Internal Server Error'
			});

			await deployedAppsStore.refresh();

			const state = get(deployedAppsStore);
			expect(state.loading).toBe(false);
			expect(state.error).toContain('Failed to fetch deployed apps');
		});

		it('should handle network error', async () => {
			(global.fetch as any).mockRejectedValueOnce(new Error('Network error'));

			await deployedAppsStore.refresh();

			const state = get(deployedAppsStore);
			expect(state.loading).toBe(false);
			expect(state.error).toBe('Network error');
		});

		it.skip('should fetch both deployed and user-generated apps', async () => {
			const deployedApps = [mockApps[0]];
			const userApps = [
				{
					id: 'user-app1',
					module_instance_id: 'user-app1',
					app_name: 'User App',
					generated_at: '2024-01-03T00:00:00Z',
					custom_config: {
						description: 'User generated app',
						category: 'productivity',
						keywords: ['user', 'custom']
					},
					custom_icon: 'Calendar'
				}
			];

			// Mock both API calls
			(global.fetch as any)
				.mockResolvedValueOnce({
					ok: true,
					json: async () => ({ apps: deployedApps })
				})
				.mockResolvedValueOnce({
					ok: true,
					json: async () => ({ apps: userApps })
				});

				await deployedAppsStore.refresh();

			const state = get(deployedAppsStore);
			expect(state.apps).toHaveLength(2);
			expect(state.apps[1].name).toBe('User App');
		});

		it('should handle user apps fetch failure gracefully', async () => {
			(global.fetch as any)
				.mockResolvedValueOnce({
					ok: true,
					json: async () => ({ apps: [mockApps[0]] })
				})
				.mockResolvedValueOnce({
					ok: false,
					statusText: 'Not Found'
				});

				await deployedAppsStore.refresh();

			const state = get(deployedAppsStore);
			// Should still have deployed apps even if user apps fail
			expect(state.apps).toHaveLength(1);
			expect(state.apps[0].name).toBe('Finance App');
		});
	});

	describe('getApp()', () => {
		beforeEach(async () => {
			(global.fetch as any).mockResolvedValueOnce({
				ok: true,
				json: async () => ({ apps: mockApps })
			});
			await deployedAppsStore.refresh();
		});

		it('should return app by ID', () => {
			const app = deployedAppsStore.getApp('app1');
			expect(app).toBeTruthy();
			expect(app?.name).toBe('Finance App');
		});

		it('should return undefined for non-existent app', () => {
			const app = deployedAppsStore.getApp('non-existent');
			expect(app).toBeUndefined();
		});
	});

	describe('startDiscovery()', () => {
		it('should start polling for apps', async () => {
			vi.useFakeTimers();

			(global.fetch as any).mockResolvedValue({
				ok: true,
				json: async () => ({ apps: mockApps })
			});

			await deployedAppsStore.startDiscovery();

			// Initial fetch
			expect(global.fetch).toHaveBeenCalledTimes(1);

			// Advance time by 10 seconds and flush promises
			await vi.advanceTimersByTimeAsync(10000);

			// Should have polled again
			expect(global.fetch).toHaveBeenCalledTimes(2);

			// Clean up BEFORE restoring timers to prevent infinite loop
			deployedAppsStore.stopDiscovery();

			// Clear any pending timers
			vi.clearAllTimers();
			vi.useRealTimers();
		});

		it('should not start multiple discovery instances', async () => {
			vi.useFakeTimers();

			(global.fetch as any).mockResolvedValue({
				ok: true,
				json: async () => ({ apps: mockApps })
			});

			await deployedAppsStore.startDiscovery();
			await deployedAppsStore.startDiscovery(); // Try to start again

			// Should only fetch once initially
			expect(global.fetch).toHaveBeenCalledTimes(1);

			vi.useRealTimers();
		});

		it.skip('should include workspace ID in user apps fetch', async () => {
			(global.fetch as any).mockResolvedValue({
				ok: true,
				json: async () => ({ apps: [] })
			});

				await deployedAppsStore.startDiscovery();

			// Should call both endpoints
			expect(global.fetch).toHaveBeenCalledWith('/api/osa/deployments', expect.any(Object));
			expect(global.fetch).toHaveBeenCalledWith('/api/workspaces/workspace-123/apps', expect.any(Object));
		});
	});

	describe('stopDiscovery()', () => {
		it('should stop polling', async () => {
			vi.useFakeTimers();

			(global.fetch as any).mockResolvedValue({
				ok: true,
				json: async () => ({ apps: mockApps })
			});

			await deployedAppsStore.startDiscovery();
			deployedAppsStore.stopDiscovery();

			const initialCallCount = (global.fetch as any).mock.calls.length;

			// Advance time
			vi.advanceTimersByTime(20000);
			await vi.runAllTimersAsync();

			// Should not have made additional calls
			expect((global.fetch as any).mock.calls.length).toBe(initialCallCount);

			vi.useRealTimers();
		});
	});

	describe('Category Helpers', () => {
		describe('getCategoryIconName()', () => {
			it('should return correct icon for finance', () => {
				expect(getCategoryIconName('finance')).toBe('DollarSign');
			});

			it('should return correct icon for communication', () => {
				expect(getCategoryIconName('communication')).toBe('MessageSquare');
			});

			it('should return correct icon for productivity', () => {
				expect(getCategoryIconName('productivity')).toBe('Calendar');
			});

			it('should return default icon for unknown category', () => {
				expect(getCategoryIconName('unknown')).toBe('AppWindow');
			});

			it('should be case-insensitive', () => {
				expect(getCategoryIconName('FINANCE')).toBe('DollarSign');
				expect(getCategoryIconName('Finance')).toBe('DollarSign');
			});

			it('should handle null/undefined', () => {
				expect(getCategoryIconName(null as any)).toBe('AppWindow');
				expect(getCategoryIconName(undefined as any)).toBe('AppWindow');
			});
		});

		describe('getCategoryColors()', () => {
			it('should return correct colors for finance', () => {
				const colors = getCategoryColors('finance');
				expect(colors.fg).toBe('#10b981');
				expect(colors.text).toBe('text-green-400');
			});

			it('should return correct colors for communication', () => {
				const colors = getCategoryColors('communication');
				expect(colors.fg).toBe('#3b82f6');
				expect(colors.text).toBe('text-blue-400');
			});

			it('should return default colors for unknown category', () => {
				const colors = getCategoryColors('unknown');
				expect(colors.fg).toBe('#6b7280');
				expect(colors.text).toBe('text-gray-400');
			});

			it('should be case-insensitive', () => {
				const colors1 = getCategoryColors('FINANCE');
				const colors2 = getCategoryColors('finance');
				expect(colors1.fg).toBe(colors2.fg);
			});

			it('should have all color properties', () => {
				const colors = getCategoryColors('finance');
				expect(colors).toHaveProperty('fg');
				expect(colors).toHaveProperty('bg');
				expect(colors).toHaveProperty('text');
			});
		});
	});

	describe('Integration with windowStore', () => {
		it('should register running apps with windowStore', async () => {
			const { windowStore } = await import('./windowStore');

			(global.fetch as any).mockResolvedValueOnce({
				ok: true,
				json: async () => ({ apps: mockApps })
			});

			await deployedAppsStore.refresh();

			// Should register the 2 running apps (not the crashed one)
			expect(windowStore.registerDeployedApp).toHaveBeenCalledTimes(2);
		});

		it('should enhance apps with category-based icons', async () => {
			const { windowStore } = await import('./windowStore');

			const appWithoutIcon = {
				id: 'app-no-icon',
				name: 'App Without Icon',
				url: 'http://localhost:3004',
				port: 3004,
				status: 'running' as const,
				metadata: {
					name: 'App Without Icon',
					description: 'Test app',
					category: 'finance',
					icon: '', // No icon
					keywords: []
				}
			};

			(global.fetch as any).mockResolvedValueOnce({
				ok: true,
				json: async () => ({ apps: [appWithoutIcon] })
			});

			await deployedAppsStore.refresh();

			// Should call registerDeployedApp with enhanced app
			expect(windowStore.registerDeployedApp).toHaveBeenCalledWith(
				expect.objectContaining({
					metadata: expect.objectContaining({
						icon: 'DollarSign' // Should use category-based icon
					})
				})
			);
		});
	});

	describe('Edge Cases', () => {
		it('should handle empty apps response', async () => {
			(global.fetch as any).mockResolvedValueOnce({
				ok: true,
				json: async () => ({ apps: [] })
			});

			await deployedAppsStore.refresh();

			const state = get(deployedAppsStore);
			expect(state.apps).toEqual([]);
			expect(state.error).toBeNull();
		});

		it('should handle response without apps property', async () => {
			(global.fetch as any).mockResolvedValueOnce({
				ok: true,
				json: async () => ({}) // No apps property
			});

			await deployedAppsStore.refresh();

			const state = get(deployedAppsStore);
			expect(state.apps).toEqual([]);
		});

		it('should handle malformed JSON', async () => {
			(global.fetch as any).mockResolvedValueOnce({
				ok: true,
				json: async () => {
					throw new Error('Invalid JSON');
				}
			});

			await deployedAppsStore.refresh();

			const state = get(deployedAppsStore);
			expect(state.error).toBeTruthy();
		});

		it('should handle apps with missing metadata', async () => {
			const appWithoutMetadata: DeployedApp = {
				id: 'app-no-meta',
				name: 'Basic App',
				url: 'http://localhost:3005',
				port: 3005,
				status: 'running'
			};

			(global.fetch as any).mockResolvedValueOnce({
				ok: true,
				json: async () => ({ apps: [appWithoutMetadata] })
			});

			await deployedAppsStore.refresh();

			const state = get(deployedAppsStore);
			expect(state.apps).toHaveLength(1);
			expect(state.apps[0].name).toBe('Basic App');
			// Store keeps original apps without modifying metadata
			// Metadata is only enhanced when registering with windowStore
			expect(state.apps[0].metadata).toBeUndefined();
		});
	});

	describe('Browser Environment', () => {
		it('should handle empty responses gracefully', async () => {
			// Note: The browser check happens at module load time via $app/environment
			// which is already mocked to return { browser: true } at the top of this file.
			// This test verifies the store can handle empty responses gracefully
			(global.fetch as any).mockResolvedValueOnce({
				ok: true,
				json: async () => ({ apps: [] })
			});

			await deployedAppsStore.refresh();

			const state = get(deployedAppsStore);
			expect(state.apps).toEqual([]);
			expect(state.error).toBeNull();
			// Verify fetch was called since browser is mocked as true
			expect(global.fetch).toHaveBeenCalledTimes(1);
		});
	});
});
