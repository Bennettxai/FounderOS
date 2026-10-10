import type { Memory, MemoryListItem } from '$lib/api/memory/types';
import type { Conversation, Message } from '$lib/api/conversations/types';

/**
 * Mock factory for Memory objects
 *
 * @param overrides - Partial Memory to override defaults
 * @returns Complete Memory object with sensible defaults
 */
export const mockMemory = (overrides?: Partial<Memory>): Memory => ({
	id: 'mock-memory-123',
	user_id: 'mock-user-456',
	title: 'Test Memory',
	summary: 'This is a test memory summary',
	content: 'This is the full content of the test memory',
	memory_type: 'fact',
	importance_score: 5,
	is_pinned: false,
	is_active: true,
	tags: ['test', 'mock'],
	metadata: {},
	source_type: null,
	source_id: null,
	project_id: null,
	node_id: null,
	expires_at: null,
	access_count: 0,
	last_accessed_at: null,
	created_at: new Date().toISOString(),
	updated_at: new Date().toISOString(),
	...overrides
});

/**
 * Mock factory for MemoryListItem objects
 */
export const mockMemoryListItem = (overrides?: Partial<MemoryListItem>): MemoryListItem => ({
	id: 'mock-memory-list-123',
	title: 'Test Memory Item',
	summary: 'Test summary',
	memory_type: 'fact',
	importance_score: 5,
	is_pinned: false,
	is_active: true,
	tags: ['test'],
	project_id: null,
	node_id: null,
	access_count: 0,
	last_accessed_at: null,
	created_at: new Date().toISOString(),
	updated_at: new Date().toISOString(),
	...overrides
});

/**
 * Mock factory for Message objects
 */
export const mockMessage = (overrides?: Partial<Message>): Message => ({
	id: 'mock-message-123',
	role: 'user',
	content: 'Test message content',
	created_at: new Date().toISOString(),
	message_metadata: {},
	...overrides
});

/**
 * Mock factory for Conversation objects
 */
export const mockConversation = (overrides?: Partial<Conversation>): Conversation => ({
	id: 'mock-conversation-123',
	title: 'Test Conversation',
	context_id: null,
	created_at: new Date().toISOString(),
	updated_at: new Date().toISOString(),
	messages: [
		mockMessage({ role: 'user', content: 'Hello' }),
		mockMessage({ role: 'assistant', content: 'Hi there!' })
	],
	message_count: 2,
	...overrides
});

/**
 * Create an array of mock memories for testing lists
 */
export const mockMemoryArray = (count: number = 3): Memory[] => {
	return Array.from({ length: count }, (_, i) =>
		mockMemory({
			id: `memory-${i}`,
			title: `Test Memory ${i + 1}`,
			content: `Content for memory ${i + 1}`,
			importance_score: i + 1
		})
	);
};

/**
 * Create an array of mock conversations for testing lists
 */
export const mockConversationArray = (count: number = 3): Conversation[] => {
	return Array.from({ length: count }, (_, i) =>
		mockConversation({
			id: `conv-${i}`,
			title: `Conversation ${i + 1}`,
			message_count: i + 1
		})
	);
};

/**
 * Mock fetch response for testing API calls
 */
export const mockFetchResponse = <T>(data: T, ok: boolean = true): Response => {
	return {
		ok,
		status: ok ? 200 : 400,
		json: async () => data,
		text: async () => JSON.stringify(data),
		headers: new Headers({ 'content-type': 'application/json' })
	} as Response;
};

/**
 * Mock error response for testing error handling
 */
export const mockErrorResponse = (message: string = 'Test error', status: number = 400): Response => {
	return {
		ok: false,
		status,
		statusText: message,
		json: async () => ({ error: message }),
		text: async () => JSON.stringify({ error: message }),
		headers: new Headers({ 'content-type': 'application/json' })
	} as Response;
};
