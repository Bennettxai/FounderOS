// Intelligence API Types - Conversation Analysis & Memory Extraction

// Renamed to avoid conflict with memory module's MemoryType
export type ExtractedMemoryType = 'fact' | 'preference' | 'decision' | 'task' | 'entity' | 'relationship' | 'insight';
export type MemorySource = 'conversation' | 'voice_note' | 'document' | 'manual';
export type SentimentType = 'positive' | 'negative' | 'neutral' | 'mixed';

// Renamed to avoid conflict with conversations module's Message
export interface IntelligenceMessage {
  role: 'user' | 'assistant' | 'system';
  content: string;
  timestamp?: string;
}

export interface ConversationAnalysis {
  id: string;
  conversation_id: string;
  user_id: string;
  title?: string;
  summary: string;
  key_points: string[];
  topics: string[];
  sentiment: {
    overall: SentimentType;
    scores?: Record<string, number>;
  };
  entities: {
    people: string[];
    tools: string[];
    concepts: string[];
    projects: string[];
  };
  action_items: string[];
  questions: string[];
  decisions: string[];
  code_mentions: Array<{
    file?: string;
    language?: string;
    snippet?: string;
  }>;
  message_count: number;
  token_count: number;
  duration?: string;
  metadata: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface ExtractedMemory {
  id: string;
  user_id: string;
  type: ExtractedMemoryType;
  content: string;
  summary?: string;
  tags: string[];
  entities: string[];
  related_to: string[];
  importance: number;
  source: MemorySource;
  source_id?: string;
  metadata: Record<string, unknown>;
  extracted_at: string;
}

export interface ExtractionOptions {
  extract_facts?: boolean;
  extract_preferences?: boolean;
  extract_decisions?: boolean;
  extract_tasks?: boolean;
  extract_entities?: boolean;
  min_importance?: number;
  max_memories?: number;
}

export interface ExtractionResult {
  memories: ExtractedMemory[];
  stats: {
    total_extracted: number;
    by_type: Record<ExtractedMemoryType, number>;
    processing_time_ms: number;
  };
}

export interface AnalyzeConversationInput {
  conversation_id: string;
  messages: IntelligenceMessage[];
}

export interface ExtractMemoriesInput {
  messages: IntelligenceMessage[];
  options?: ExtractionOptions;
}

export interface ExtractFromVoiceNoteInput {
  transcript: string;
  options?: ExtractionOptions;
}
