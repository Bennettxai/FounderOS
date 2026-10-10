// Wire types for GET /api/founderos/pages/chats and /pages/chats/:id
// (internal/founderos/api/page_chats.go).
export type ConversationSummary = {
	agentId: string;
	agentName: string;
	lastMessage: string;
	lastAt: string;
	messageCount: number;
};

export type RosterAgent = { id: string; name: string; description: string };

export type AgentMessage = {
	id: string;
	agentId: string;
	role: 'user' | 'assistant' | 'tool';
	content: string;
	toolCalls: { name: string }[];
	createdAt: string;
};

export type ChatsView = {
	summaries: ConversationSummary[];
	roster: RosterAgent[];
	conductorModel: string | null;
	boardUrl: string | null;
	/** The model behind direct chats; not configured without AI_GATEWAY_API_KEY. */
	llm: { configured: boolean; detail: string };
};

/** POST /pages/chats/:id — configured:false is the honest no-key answer (200). */
export type ChatSendResult = {
	configured: boolean;
	error?: string;
	reply?: string;
	routedTo?: string;
	messages: AgentMessage[];
};
