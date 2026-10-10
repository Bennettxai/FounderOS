import { describe, it, expect, vi, beforeEach } from 'vitest';
import {
	request,
	raw,
	getApiBaseUrl,
	getCSRFToken,
	initCSRF,
	invalidateApiCache
} from './base';

// Mock fetch globally
global.fetch = vi.fn();

describe('API client (base API)', () => {
	beforeEach(() => {
		vi.clearAllMocks();
		(global.fetch as any).mockReset();
		invalidateApiCache();
		localStorage.clear();
		delete (globalThis as any).electron;
	});

	describe('GET requests via request()', () => {
		it('should make GET request with correct URL', async () => {
			(global.fetch as any).mockResolvedValueOnce({
				ok: true,
				json: async () => ({ data: 'test' })
			});

			await request('/test');

			const baseUrl = getApiBaseUrl();
			expect(global.fetch).toHaveBeenCalledWith(
				`${baseUrl}/test`,
				expect.objectContaining({
					method: 'GET'
				})
			);
		});

		it('should include credentials', async () => {
			(global.fetch as any).mockResolvedValueOnce({
				ok: true,
				json: async () => ({})
			});

			await request('/test');

			expect(global.fetch).toHaveBeenCalledWith(
				expect.any(String),
				expect.objectContaining({
					credentials: 'include'
				})
			);
		});

		it('should add query parameters', async () => {
			(global.fetch as any).mockResolvedValueOnce({
				ok: true,
				json: async () => ({})
			});

			await request('/test?foo=bar&page=1');

			expect(global.fetch).toHaveBeenCalledWith(
				expect.stringContaining('foo=bar'),
				expect.any(Object)
			);
		});

		it('should parse JSON response', async () => {
			const testData = { users: ['user1', 'user2'] };
			(global.fetch as any).mockResolvedValueOnce({
				ok: true,
				json: async () => testData
			});

			const result = await request<typeof testData>('/users');

			expect(result).toEqual(testData);
		});

		it('should throw on HTTP error', async () => {
			(global.fetch as any).mockResolvedValueOnce({
				ok: false,
				status: 404,
				json: async () => ({ detail: 'Not found' })
			});

			await expect(request('/not-found')).rejects.toThrow();
		});

		it('should surface backend error fields', async () => {
			(global.fetch as any).mockResolvedValueOnce({
				ok: false,
				status: 403,
				json: async () => ({ error: 'select a workspace first' })
			});

			await expect(request('/apps')).rejects.toThrow(
				'select a workspace first (HTTP 403)'
			);
		});

		it('should throw on network error', async () => {
			(global.fetch as any).mockRejectedValueOnce(new Error('Network error'));

			await expect(request('/test')).rejects.toThrow('Network error');
		});
	});

	describe('POST requests', () => {
		it('should make POST request with JSON body', async () => {
			const body = { name: 'Test User', email: 'test@example.com' };
			(global.fetch as any).mockResolvedValueOnce({
				ok: true,
				json: async () => ({ id: 1, ...body })
			});

			await request('/users', { method: 'POST', body });

			expect(global.fetch).toHaveBeenCalledWith(
				expect.any(String),
				expect.objectContaining({
					method: 'POST',
					body: JSON.stringify(body),
					headers: expect.objectContaining({
						'Content-Type': 'application/json'
					})
				})
			);
		});

		it('should handle empty body', async () => {
			(global.fetch as any).mockResolvedValueOnce({
				ok: true,
				json: async () => ({ success: true })
			});

			await request('/endpoint', { method: 'POST' });

			expect(global.fetch).toHaveBeenCalledWith(
				expect.any(String),
				expect.objectContaining({
					method: 'POST',
					body: undefined
				})
			);
		});

		it('should return response data', async () => {
			const responseData = { id: 1, created: true };
			(global.fetch as any).mockResolvedValueOnce({
				ok: true,
				json: async () => responseData
			});

			const result = await request('/create', { method: 'POST', body: { data: 'test' } });

			expect(result).toEqual(responseData);
		});

		it('should mirror workspace memory writes into Optimal Engine', async () => {
			const memory = vi.fn().mockResolvedValue({ ok: true });
			(globalThis as any).electron = { engine: { memory } };
			localStorage.setItem('businessos_current_workspace_id', 'workspace-1');
			(global.fetch as any).mockResolvedValueOnce({
				ok: true,
				json: async () => ({
					id: 'mem-1',
					title: 'Jordan follow-up',
					content: 'Ask Dominic about the Paykit structure.'
				})
			});

			await request('/workspaces/workspace-1/memories', {
				method: 'POST',
				body: {
					title: 'Jordan follow-up',
					content: 'Ask Dominic about the Paykit structure.'
				}
			});

			expect(memory).toHaveBeenCalledWith(
				'workspace-1',
				expect.objectContaining({
					content: expect.stringContaining('Ask Dominic about the Paykit structure.'),
					citation: 'businessos://workspace-memory/mem-1',
					metadata: expect.objectContaining({
						businessos_module: 'workspace-memory',
						businessos_id: 'mem-1'
					})
				})
			);
		});

		it('should mirror content writes into Optimal Engine', async () => {
			const memory = vi.fn().mockResolvedValue({ ok: true });
			(globalThis as any).electron = { engine: { memory } };
			localStorage.setItem('businessos_current_workspace_id', 'workspace-1');
			(global.fetch as any).mockResolvedValueOnce({
				ok: true,
				json: async () => ({
					id: 'content-1',
					title: 'Robert launch script',
					body: 'Record the Monday story using the den setup.'
				})
			});

			await request('/content', {
				method: 'POST',
				body: {
					title: 'Robert launch script',
					body: 'Record the Monday story using the den setup.'
				}
			});

			expect(memory).toHaveBeenCalledWith(
				'workspace-1',
				expect.objectContaining({
					content: expect.stringContaining('Record the Monday story using the den setup.'),
					citation: 'businessos://content/content-1'
				})
			);
		});

		it('should not mirror deletes into Optimal Engine', async () => {
			const memory = vi.fn().mockResolvedValue({ ok: true });
			(globalThis as any).electron = { engine: { memory } };
			localStorage.setItem('businessos_current_workspace_id', 'workspace-1');
			(global.fetch as any).mockResolvedValueOnce({
				ok: true,
				json: async () => ({ deleted: true })
			});

			await request('/workspaces/workspace-1/memories/mem-1', { method: 'DELETE' });

			expect(memory).not.toHaveBeenCalled();
		});
	});

	describe('CSRF initialization', () => {
		it('uses the resolved Electron API path and caches the response token', async () => {
			(window as any).electron = {};
			(global.fetch as any).mockResolvedValueOnce({
				ok: true,
				json: async () => ({ csrf_token: 'electron-csrf-token' })
			});

			await initCSRF();

			expect(global.fetch).toHaveBeenCalledWith(
				`${getApiBaseUrl()}/auth/csrf`,
				expect.objectContaining({ credentials: 'include' })
			);
			expect(getCSRFToken()).toBe('electron-csrf-token');
		});
	});

	describe('PUT requests', () => {
		it('should make PUT request', async () => {
			const body = { name: 'Updated' };
			(global.fetch as any).mockResolvedValueOnce({
				ok: true,
				json: async () => ({ id: 1, ...body })
			});

			await request('/users/1', { method: 'PUT', body });

			expect(global.fetch).toHaveBeenCalledWith(
				expect.any(String),
				expect.objectContaining({
					method: 'PUT',
					body: JSON.stringify(body)
				})
			);
		});
	});

	describe('PATCH requests', () => {
		it('should make PATCH request', async () => {
			const body = { email: 'new@example.com' };
			(global.fetch as any).mockResolvedValueOnce({
				ok: true,
				json: async () => ({ id: 1, ...body })
			});

			await request('/users/1', { method: 'PATCH', body });

			expect(global.fetch).toHaveBeenCalledWith(
				expect.any(String),
				expect.objectContaining({
					method: 'PATCH',
					body: JSON.stringify(body)
				})
			);
		});
	});

	describe('DELETE requests', () => {
		it('should make DELETE request', async () => {
			(global.fetch as any).mockResolvedValueOnce({
				ok: true,
				json: async () => ({ success: true })
			});

			await request('/users/1', { method: 'DELETE' });

			expect(global.fetch).toHaveBeenCalledWith(
				expect.any(String),
				expect.objectContaining({
					method: 'DELETE'
				})
			);
		});

		it('should make DELETE request with body', async () => {
			const body = { ids: [1, 2, 3] };
			(global.fetch as any).mockResolvedValueOnce({
				ok: true,
				json: async () => ({ deleted: 3 })
			});

			await request('/users', { method: 'DELETE', body });

			expect(global.fetch).toHaveBeenCalledWith(
				expect.any(String),
				expect.objectContaining({
					method: 'DELETE',
					body: JSON.stringify(body)
				})
			);
		});
	});

	describe('raw API', () => {
		it('should make raw GET request', async () => {
			const mockResponse = new Response(JSON.stringify({ data: 'test' }), {
				status: 200,
				headers: { 'Content-Type': 'application/json' }
			});
			(global.fetch as any).mockResolvedValueOnce(mockResponse);

			const response = await raw.get('/test');

			expect(response.status).toBe(200);
			const data = await response.json();
			expect(data).toEqual({ data: 'test' });
		});

		it('should make raw POST request', async () => {
			const mockResponse = new Response(JSON.stringify({ created: true }), {
				status: 201
			});
			(global.fetch as any).mockResolvedValueOnce(mockResponse);

			const response = await raw.post('/test', { name: 'Test' });

			expect(response.status).toBe(201);
		});
	});

	describe('Error handling', () => {
		it('should handle 401 unauthorized', async () => {
			(global.fetch as any).mockResolvedValueOnce({
				ok: false,
				status: 401,
				json: async () => ({ detail: 'Unauthorized' })
			});

			await expect(request('/protected')).rejects.toThrow();
		});

		it('should handle 500 server error', async () => {
			(global.fetch as any).mockResolvedValueOnce({
				ok: false,
				status: 500,
				json: async () => ({ detail: 'Internal server error' })
			});

			await expect(request('/test')).rejects.toThrow();
		});

		it('should handle malformed JSON response', async () => {
			(global.fetch as any).mockResolvedValueOnce({
				ok: false,
				status: 400,
				json: async () => {
					throw new Error('Invalid JSON');
				}
			});

			await expect(request('/test')).rejects.toThrow();
		});
	});

	describe('Edge cases', () => {
		it('should handle leading slash in endpoint', async () => {
			(global.fetch as any).mockResolvedValueOnce({
				ok: true,
				json: async () => ({})
			});

			await request('/test');

			expect(global.fetch).toHaveBeenCalled();
		});

		it('should handle endpoint without leading slash', async () => {
			(global.fetch as any).mockResolvedValueOnce({
				ok: true,
				json: async () => ({})
			});

			await request('test');

			expect(global.fetch).toHaveBeenCalled();
		});

		it('should handle complex nested paths', async () => {
			(global.fetch as any).mockResolvedValueOnce({
				ok: true,
				json: async () => ({})
			});

			await request('/users/123/posts/456');

			expect(global.fetch).toHaveBeenCalledWith(
				expect.stringContaining('/users/123/posts/456'),
				expect.any(Object)
			);
		});

		it('should handle query strings with special characters', async () => {
			(global.fetch as any).mockResolvedValueOnce({
				ok: true,
				json: async () => ({})
			});

			await request('/search?q=hello%20world&filter=a%26b');

			expect(global.fetch).toHaveBeenCalledWith(
				expect.stringContaining('search?q=hello%20world'),
				expect.any(Object)
			);
		});
	});

	describe('Timeout handling', () => {
		it('should timeout after specified duration', async () => {
			// Mock a fetch that respects the abort signal
			(global.fetch as any).mockImplementationOnce((_url: string, options: any) => {
				return new Promise((resolve, reject) => {
					const timeoutId = setTimeout(() => {
						resolve({
							ok: true,
							json: async () => ({ data: 'too late' })
						});
					}, 200); // Simulate slow response

					// Listen for abort signal
					if (options.signal) {
						options.signal.addEventListener('abort', () => {
							clearTimeout(timeoutId);
							const abortError = new Error('The operation was aborted');
							abortError.name = 'AbortError';
							reject(abortError);
						});
					}
				});
			});

			// Request with 100ms timeout should fail before the 200ms response
			await expect(request('/slow', { timeout: 100 })).rejects.toThrow('Request timeout after 100ms');
		}, 10000);

		it('should not timeout when request completes in time', async () => {
			(global.fetch as any).mockResolvedValueOnce({
				ok: true,
				json: async () => ({ data: 'success' })
			});

			const result = await request('/fast', { timeout: 1000 });

			expect(result).toEqual({ data: 'success' });
		});

		it('should work without timeout option', async () => {
			(global.fetch as any).mockResolvedValueOnce({
				ok: true,
				json: async () => ({ data: 'no timeout' })
			});

			const result = await request('/test');

			expect(result).toEqual({ data: 'no timeout' });
		});

		it('should ignore timeout if set to 0', async () => {
			(global.fetch as any).mockResolvedValueOnce({
				ok: true,
				json: async () => ({ data: 'no timeout' })
			});

			const result = await request('/test', { timeout: 0 });

			expect(result).toEqual({ data: 'no timeout' });
		});

		it('should ignore negative timeout values', async () => {
			(global.fetch as any).mockResolvedValueOnce({
				ok: true,
				json: async () => ({ data: 'no timeout' })
			});

			const result = await request('/test', { timeout: -100 });

			expect(result).toEqual({ data: 'no timeout' });
		});
	});
});
