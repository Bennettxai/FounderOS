import { describe, it, expect, vi, beforeEach } from 'vitest';
import * as memoryApi from './memory';
import { mockMemory, mockMemoryListItem, mockFetchResponse, mockErrorResponse } from '$lib/test-utils/mocks';

// Mock the base request function
vi.mock('../base', () => ({
	request: vi.fn()
}));

import { request } from '../base';
const mockedRequest = vi.mocked(request);

describe('Memory API', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	describe('getMemories', () => {
		it('fetches memories without filters', async () => {
			const mockData = [mockMemoryListItem(), mockMemoryListItem()];
			mockedRequest.mockResolvedValueOnce({ memories: mockData });

			const result = await memoryApi.getMemories();

			expect(mockedRequest).toHaveBeenCalledWith('/memories');
			expect(result).toEqual(mockData);
		});

		it('fetches memories with filters', async () => {
			const mockData = [mockMemoryListItem({ memory_type: 'fact' })];
			mockedRequest.mockResolvedValueOnce({ memories: mockData });

			const filters = {
				memory_type: 'fact' as const,
				is_pinned: true,
				limit: 10
			};

			const result = await memoryApi.getMemories(filters);

			expect(mockedRequest).toHaveBeenCalledWith(expect.stringContaining('memory_type=fact'));
			expect(mockedRequest).toHaveBeenCalledWith(expect.stringContaining('is_pinned=true'));
			expect(mockedRequest).toHaveBeenCalledWith(expect.stringContaining('limit=10'));
			expect(result).toEqual(mockData);
		});

		it('handles tags in filters', async () => {
			const mockData = [mockMemoryListItem()];
			mockedRequest.mockResolvedValueOnce({ memories: mockData });

			const filters = {
				tags: ['javascript', 'testing']
			};

			await memoryApi.getMemories(filters);

			expect(mockedRequest).toHaveBeenCalledWith(expect.stringContaining('tags=javascript%2Ctesting'));
		});
	});

	describe('getMemory', () => {
		it('fetches a single memory by ID', async () => {
			const mockData = mockMemory({ id: 'test-123' });
			mockedRequest.mockResolvedValueOnce(mockData);

			const result = await memoryApi.getMemory('test-123');

			expect(mockedRequest).toHaveBeenCalledWith('/memories/test-123');
			expect(result).toEqual(mockData);
		});
	});

	describe('createMemory', () => {
		it('creates a new memory', async () => {
			const createData = {
				title: 'New Memory',
				content: 'Memory content',
				memory_type: 'fact' as const
			};
			const mockData = mockMemory(createData);
			mockedRequest.mockResolvedValueOnce(mockData);

			const result = await memoryApi.createMemory(createData);

			expect(mockedRequest).toHaveBeenCalledWith('/memories', {
				method: 'POST',
				body: createData
			});
			expect(result).toEqual(mockData);
		});

		it('includes optional fields in create request', async () => {
			const createData = {
				title: 'New Memory',
				content: 'Memory content',
				memory_type: 'preference' as const,
				importance_score: 0.8,
				tags: ['important', 'work'],
				project_id: 'project-123'
			};
			const mockData = mockMemory(createData);
			mockedRequest.mockResolvedValueOnce(mockData);

			await memoryApi.createMemory(createData);

			expect(mockedRequest).toHaveBeenCalledWith('/memories', {
				method: 'POST',
				body: createData
			});
		});
	});

	describe('updateMemory', () => {
		it('updates an existing memory', async () => {
			const updateData = {
				title: 'Updated Title',
				importance_score: 0.9
			};
			const mockData = mockMemory({ id: 'test-123', ...updateData });
			mockedRequest.mockResolvedValueOnce(mockData);

			const result = await memoryApi.updateMemory('test-123', updateData);

			expect(mockedRequest).toHaveBeenCalledWith('/memories/test-123', {
				method: 'PUT',
				body: updateData
			});
			expect(result).toEqual(mockData);
		});
	});

	describe('deleteMemory', () => {
		it('deletes a memory by ID', async () => {
			mockedRequest.mockResolvedValueOnce(undefined);

			await memoryApi.deleteMemory('test-123');

			expect(mockedRequest).toHaveBeenCalledWith('/memories/test-123', {
				method: 'DELETE'
			});
		});
	});

	describe('pinMemory', () => {
		it('pins a memory', async () => {
			const mockData = mockMemory({ id: 'test-123', is_pinned: true });
			mockedRequest.mockResolvedValueOnce(mockData);

			const result = await memoryApi.pinMemory('test-123', true);

			expect(mockedRequest).toHaveBeenCalledWith('/memories/test-123/pin', {
				method: 'POST',
				body: { is_pinned: true }
			});
			expect(result.is_pinned).toBe(true);
		});

		it('unpins a memory', async () => {
			const mockData = mockMemory({ id: 'test-123', is_pinned: false });
			mockedRequest.mockResolvedValueOnce(mockData);

			const result = await memoryApi.pinMemory('test-123', false);

			expect(mockedRequest).toHaveBeenCalledWith('/memories/test-123/pin', {
				method: 'POST',
				body: { is_pinned: false }
			});
			expect(result.is_pinned).toBe(false);
		});
	});

	describe('searchMemories', () => {
		it('searches memories by query', async () => {
			const mockResults = [mockMemoryListItem()];
			mockedRequest.mockResolvedValueOnce({ results: mockResults });

			const params = {
				query: 'test search',
				limit: 5
			};

			const result = await memoryApi.searchMemories(params);

			expect(mockedRequest).toHaveBeenCalledWith('/memories/search', {
				method: 'POST',
				body: params
			});
			expect(result).toEqual(mockResults);
		});

		it('includes optional search parameters', async () => {
			const mockResults = [mockMemoryListItem()];
			mockedRequest.mockResolvedValueOnce({ results: mockResults });

			const params = {
				query: 'javascript',
				memory_type: 'learning' as const,
				project_id: 'proj-123',
				min_score: 0.7
			};

			await memoryApi.searchMemories(params);

			expect(mockedRequest).toHaveBeenCalledWith('/memories/search', {
				method: 'POST',
				body: params
			});
		});
	});

	describe('getRelevantMemories', () => {
		it('fetches relevant memories for a query', async () => {
			const mockResults = [mockMemoryListItem()];
			mockedRequest.mockResolvedValueOnce({ results: mockResults });

			const params = {
				query: 'test context',
				conversation_id: 'conv-123'
			};

			const result = await memoryApi.getRelevantMemories(params);

			expect(mockedRequest).toHaveBeenCalledWith('/memories/relevant', {
				method: 'POST',
				body: params
			});
			expect(result).toEqual(mockResults);
		});
	});

	describe('getProjectMemories', () => {
		it('fetches memories for a project', async () => {
			const mockData = [mockMemoryListItem()];
			mockedRequest.mockResolvedValueOnce({ memories: mockData });

			const result = await memoryApi.getProjectMemories('proj-123');

			expect(mockedRequest).toHaveBeenCalledWith('/memories/project/proj-123');
			expect(result).toEqual(mockData);
		});

		it('respects limit parameter', async () => {
			const mockData = [mockMemoryListItem()];
			mockedRequest.mockResolvedValueOnce({ memories: mockData });

			await memoryApi.getProjectMemories('proj-123', 5);

			expect(mockedRequest).toHaveBeenCalledWith(expect.stringContaining('limit=5'));
		});
	});

	describe('getNodeMemories', () => {
		it('fetches memories for a node', async () => {
			const mockData = [mockMemoryListItem()];
			mockedRequest.mockResolvedValueOnce({ memories: mockData });

			const result = await memoryApi.getNodeMemories('node-123');

			expect(mockedRequest).toHaveBeenCalledWith('/memories/node/node-123');
			expect(result).toEqual(mockData);
		});
	});

	describe('getMemoryStats', () => {
		it('fetches memory statistics', async () => {
			const mockStats = {
				total_memories: 100,
				active_memories: 95,
				pinned_memories: 10,
				by_type: {
					fact: 30,
					preference: 20,
					decision: 15,
					event: 10,
					learning: 15,
					context: 8,
					relationship: 2
				},
				avg_importance: 0.65,
				total_access_count: 450
			};
			mockedRequest.mockResolvedValueOnce(mockStats);

			const result = await memoryApi.getMemoryStats();

			expect(mockedRequest).toHaveBeenCalledWith('/memories/stats');
			expect(result).toEqual(mockStats);
		});
	});

	describe('error handling', () => {
		it('propagates errors from failed requests', async () => {
			const error = new Error('Network error');
			mockedRequest.mockRejectedValueOnce(error);

			await expect(memoryApi.getMemories()).rejects.toThrow('Network error');
		});

		it('handles 404 errors for missing memory', async () => {
			const error = new Error('Memory not found');
			mockedRequest.mockRejectedValueOnce(error);

			await expect(memoryApi.getMemory('nonexistent')).rejects.toThrow('Memory not found');
		});
	});
});
