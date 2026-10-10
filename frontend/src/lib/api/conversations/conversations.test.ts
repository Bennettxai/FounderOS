import { describe, it, expect, vi, beforeEach } from 'vitest';
import * as conversationsApi from './conversations';
import { mockConversation, mockMessage } from '$lib/test-utils/mocks';

// Mock the base request, getApiBaseUrl, and getCSRFToken functions
vi.mock('../base', async (importOriginal) => {
	const actual = await importOriginal<typeof import('../base')>();
	return {
		...actual,
		request: vi.fn(),
		getCSRFToken: vi.fn(() => 'test-csrf-token')
	};
});

import { request, getApiBaseUrl, getCSRFToken } from '../base';
const mockedRequest = vi.mocked(request);
const mockedGetApiBaseUrl = vi.mocked(getApiBaseUrl);
const mockedGetCSRFToken = vi.mocked(getCSRFToken);

describe('Conversations API', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	describe('getConversations', () => {
		it('fetches all conversations', async () => {
			const mockData = [mockConversation(), mockConversation()];
			// API now returns wrapped response with conversations and total
			mockedRequest.mockResolvedValueOnce({ conversations: mockData, total: 2 });

			const result = await conversationsApi.getConversations();

			expect(mockedRequest).toHaveBeenCalledWith('/chat/conversations');
			expect(result).toEqual({ conversations: mockData, total: 2 });
		});
	});

	describe('getConversation', () => {
		it('fetches a single conversation with messages', async () => {
			const conversation = mockConversation({ id: 'conv-123' });
			const messages = [mockMessage(), mockMessage()];

			mockedRequest.mockResolvedValueOnce({
				conversation,
				messages
			});

			const result = await conversationsApi.getConversation('conv-123');

			expect(mockedRequest).toHaveBeenCalledWith('/chat/conversations/conv-123');
			expect(result.id).toBe('conv-123');
			expect(result.messages).toEqual(messages);
			expect(result.message_count).toBe(2);
		});

		it('handles conversation with no messages', async () => {
			const conversation = mockConversation({ id: 'conv-123' });

			mockedRequest.mockResolvedValueOnce({
				conversation,
				messages: []
			});

			const result = await conversationsApi.getConversation('conv-123');

			expect(result.messages).toEqual([]);
			expect(result.message_count).toBe(0);
		});
	});

	describe('createConversation', () => {
		it('creates a new conversation with title', async () => {
			const mockData = mockConversation({ title: 'New Chat' });
			mockedRequest.mockResolvedValueOnce(mockData);

			const result = await conversationsApi.createConversation('New Chat');

			expect(mockedRequest).toHaveBeenCalledWith('/chat/conversations', {
				method: 'POST',
				body: { title: 'New Chat', context_id: undefined }
			});
			expect(result).toEqual(mockData);
		});

		it('creates a new conversation with context ID', async () => {
			const mockData = mockConversation({ context_id: 'ctx-123' });
			mockedRequest.mockResolvedValueOnce(mockData);

			const result = await conversationsApi.createConversation('New Chat', 'ctx-123');

			expect(mockedRequest).toHaveBeenCalledWith('/chat/conversations', {
				method: 'POST',
				body: { title: 'New Chat', context_id: 'ctx-123' }
			});
			expect(result).toEqual(mockData);
		});

		it('creates a conversation without title', async () => {
			const mockData = mockConversation();
			mockedRequest.mockResolvedValueOnce(mockData);

			await conversationsApi.createConversation();

			expect(mockedRequest).toHaveBeenCalledWith('/chat/conversations', {
				method: 'POST',
				body: { title: undefined, context_id: undefined }
			});
		});
	});

	describe('deleteConversation', () => {
		it('deletes a conversation by ID', async () => {
			mockedRequest.mockResolvedValueOnce(undefined);

			await conversationsApi.deleteConversation('conv-123');

			expect(mockedRequest).toHaveBeenCalledWith('/chat/conversations/conv-123', {
				method: 'DELETE'
			});
		});
	});

	describe('updateConversation', () => {
		it('updates conversation title', async () => {
			const mockData = mockConversation({ id: 'conv-123', title: 'Updated Title' });
			mockedRequest.mockResolvedValueOnce(mockData);

			const result = await conversationsApi.updateConversation('conv-123', {
				title: 'Updated Title'
			});

			expect(mockedRequest).toHaveBeenCalledWith('/chat/conversations/conv-123', {
				method: 'PUT',
				body: { title: 'Updated Title' }
			});
			expect(result.title).toBe('Updated Title');
		});

		it('updates conversation context', async () => {
			const mockData = mockConversation({ id: 'conv-123', context_id: 'ctx-456' });
			mockedRequest.mockResolvedValueOnce(mockData);

			const result = await conversationsApi.updateConversation('conv-123', {
				context_id: 'ctx-456'
			});

			expect(mockedRequest).toHaveBeenCalledWith('/chat/conversations/conv-123', {
				method: 'PUT',
				body: { context_id: 'ctx-456' }
			});
			expect(result.context_id).toBe('ctx-456');
		});

		it('can clear context by setting to null', async () => {
			const mockData = mockConversation({ id: 'conv-123', context_id: null });
			mockedRequest.mockResolvedValueOnce(mockData);

			await conversationsApi.updateConversation('conv-123', {
				context_id: null
			});

			expect(mockedRequest).toHaveBeenCalledWith('/chat/conversations/conv-123', {
				method: 'PUT',
				body: { context_id: null }
			});
		});
	});

	describe('getConversationsByContext', () => {
		it('fetches conversations for a specific context', async () => {
			const mockData = [mockConversation({ context_id: 'ctx-123' })];
			mockedRequest.mockResolvedValueOnce(mockData);

			const result = await conversationsApi.getConversationsByContext('ctx-123');

			expect(mockedRequest).toHaveBeenCalledWith('/chat/conversations?context_id=ctx-123');
			expect(result).toEqual(mockData);
		});

		it('encodes context ID in URL', async () => {
			const mockData = [mockConversation()];
			mockedRequest.mockResolvedValueOnce(mockData);

			await conversationsApi.getConversationsByContext('ctx with spaces');

			expect(mockedRequest).toHaveBeenCalledWith(
				expect.stringContaining('ctx%20with%20spaces')
			);
		});
	});

	describe('sendMessage', () => {
		beforeEach(() => {
			// Mock global fetch for sendMessage
			global.fetch = vi.fn();
		});

		it('sends a message and returns stream', async () => {
			const mockStream = new ReadableStream();
			const mockResponse = {
				ok: true,
				body: mockStream,
				headers: new Map([['X-Conversation-Id', 'conv-123']])
			};
			mockResponse.headers.get = vi.fn((key: string) => {
				if (key === 'X-Conversation-Id') return 'conv-123';
				return undefined;
			});

			vi.mocked(global.fetch).mockResolvedValueOnce(mockResponse as any);

			const result = await conversationsApi.sendMessage('Hello, AI!');

			// Verify fetch was called with correct URL and options
			expect(global.fetch).toHaveBeenCalledWith(
				expect.stringContaining('/chat/message'),
				expect.objectContaining({
					method: 'POST',
					headers: expect.objectContaining({
						'Content-Type': 'application/json',
						'X-CSRF-Token': 'test-csrf-token'
					}),
					credentials: 'include',
					body: JSON.stringify({
						message: 'Hello, AI!',
						conversation_id: undefined,
						context_id: undefined,
						model: undefined,
						structured_output: undefined,
						output_style: undefined
					})
				})
			);

			expect(result.stream).toBe(mockStream);
			expect(result.conversationId).toBe('conv-123');
		});

		it('sends message with conversation ID', async () => {
			const mockStream = new ReadableStream();
			const mockResponse = {
				ok: true,
				body: mockStream,
				headers: new Map()
			};
			mockResponse.headers.get = vi.fn(() => null);

			vi.mocked(global.fetch).mockResolvedValueOnce(mockResponse as any);

			await conversationsApi.sendMessage('Hello again', 'conv-456');

			expect(global.fetch).toHaveBeenCalledWith(
				expect.any(String),
				expect.objectContaining({
					body: expect.stringContaining('"conversation_id":"conv-456"')
				})
			);
		});

		it('sends message with model and options', async () => {
			const mockStream = new ReadableStream();
			const mockResponse = {
				ok: true,
				body: mockStream,
				headers: new Map()
			};
			mockResponse.headers.get = vi.fn(() => null);

			vi.mocked(global.fetch).mockResolvedValueOnce(mockResponse as any);

			await conversationsApi.sendMessage(
				'Test message',
				undefined,
				undefined,
				'gpt-4',
				{ structured_output: true, output_style: 'formal' }
			);

			expect(global.fetch).toHaveBeenCalledWith(
				expect.any(String),
				expect.objectContaining({
					body: expect.stringContaining('"model":"gpt-4"')
				})
			);
			expect(global.fetch).toHaveBeenCalledWith(
				expect.any(String),
				expect.objectContaining({
					body: expect.stringContaining('"structured_output":true')
				})
			);
		});

		it('throws error on failed request', async () => {
			const mockResponse = {
				ok: false,
				json: async () => ({ detail: 'Chat service unavailable' })
			};

			vi.mocked(global.fetch).mockResolvedValueOnce(mockResponse as any);

			await expect(
				conversationsApi.sendMessage('Test')
			).rejects.toThrow('Chat service unavailable');
		});

		it('handles error when response.json() fails', async () => {
			const mockResponse = {
				ok: false,
				json: async () => {
					throw new Error('Invalid JSON');
				}
			};

			vi.mocked(global.fetch).mockResolvedValueOnce(mockResponse as any);

			await expect(
				conversationsApi.sendMessage('Test')
			).rejects.toThrow('Chat failed');
		});
	});

	describe('searchConversations', () => {
		it('searches conversations by query', async () => {
			const mockResults = [
				{
					message_id: 'msg-1',
					conversation_id: 'conv-1',
					content: 'Found content',
					role: 'user',
					created_at: new Date().toISOString()
				}
			];
			mockedRequest.mockResolvedValueOnce(mockResults);

			const result = await conversationsApi.searchConversations('test query');

			expect(mockedRequest).toHaveBeenCalledWith(
				'/chat/search?q=test%20query'
			);
			expect(result).toEqual(mockResults);
		});

		it('encodes special characters in search query', async () => {
			mockedRequest.mockResolvedValueOnce([]);

			await conversationsApi.searchConversations('test & query?');

			expect(mockedRequest).toHaveBeenCalledWith(
				expect.stringContaining('test%20%26%20query%3F')
			);
		});
	});

	describe('error handling', () => {
		it('propagates errors from failed requests', async () => {
			const error = new Error('Network error');
			mockedRequest.mockRejectedValueOnce(error);

			await expect(conversationsApi.getConversations()).rejects.toThrow('Network error');
		});

		it('handles 404 for missing conversation', async () => {
			const error = new Error('Conversation not found');
			mockedRequest.mockRejectedValueOnce(error);

			await expect(conversationsApi.getConversation('nonexistent')).rejects.toThrow(
				'Conversation not found'
			);
		});
	});
});
