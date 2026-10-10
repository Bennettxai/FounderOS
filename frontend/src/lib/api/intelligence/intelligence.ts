// Intelligence API - Conversation Analysis & Memory Extraction
import { request } from '../base';
import type {
  ConversationAnalysis,
  ExtractedMemory,
  ExtractionResult,
  AnalyzeConversationInput,
  ExtractMemoriesInput,
  ExtractFromVoiceNoteInput
} from './types';

/**
 * Analyze a conversation for insights, topics, and action items
 */
export async function analyzeConversation(
  input: AnalyzeConversationInput
): Promise<ConversationAnalysis> {
  return request<ConversationAnalysis>('/intelligence/analyze', {
    method: 'POST',
    body: JSON.stringify(input)
  });
}

/**
 * Get analysis for a specific conversation
 */
export async function getConversationAnalysis(
  conversationId: string
): Promise<ConversationAnalysis> {
  return request<ConversationAnalysis>(`/intelligence/conversations/${conversationId}`);
}

/**
 * Search conversation analyses
 * Renamed from searchConversations to avoid conflict with conversations module
 */
export async function searchConversationAnalyses(
  query: string,
  limit: number = 20
): Promise<ConversationAnalysis[]> {
  const params = new URLSearchParams({
    q: query,
    limit: limit.toString()
  });
  return request<ConversationAnalysis[]>(`/intelligence/conversations/search?${params}`);
}

/**
 * Extract memories from a conversation
 */
export async function extractMemoriesFromConversation(
  input: ExtractMemoriesInput
): Promise<ExtractionResult> {
  return request<ExtractionResult>('/intelligence/extract/conversation', {
    method: 'POST',
    body: JSON.stringify(input)
  });
}

/**
 * Extract memories from a voice note transcript
 */
export async function extractMemoriesFromVoiceNote(
  input: ExtractFromVoiceNoteInput
): Promise<ExtractionResult> {
  return request<ExtractionResult>('/intelligence/extract/voice-note', {
    method: 'POST',
    body: JSON.stringify(input)
  });
}

/**
 * Get all extracted memories
 */
export async function getExtractedMemories(
  type?: string,
  limit: number = 50
): Promise<ExtractedMemory[]> {
  const params = new URLSearchParams();
  if (type) params.append('type', type);
  params.append('limit', limit.toString());

  const query = params.toString();
  return request<ExtractedMemory[]>(`/intelligence/memories${query ? `?${query}` : ''}`);
}
