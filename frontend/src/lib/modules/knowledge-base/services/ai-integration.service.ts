/**
 * AI Integration Service
 *
 * Bridges AI Creations (artifacts) with the Knowledge Base.
 * Allows saving AI-generated content as Pages in the Knowledge Base.
 *
 * TAXONOMY:
 * - Creation = AI-generated output (code, documents, frameworks, etc.)
 *   - Stored in artifacts table in backend
 *   - Can be saved to Knowledge Base as a Page
 * - Page = Document in Knowledge Base
 * - Conversation = Chat session with Messages
 */

import type {
	Artifact as ArtifactType_,
	ArtifactListItem as ArtifactListItemType_,
	ArtifactType,
	CreateArtifactData
} from '$lib/api/artifacts/types';
import type { Message, Block as ConversationBlock } from '$lib/api/conversations/types';
import type { CreateContextData as CreatePageData, ContextType } from '$lib/api/contexts/types';
import * as artifactsApi from '$lib/api/artifacts';
import * as contextsApi from '$lib/api/contexts';
import type { Block as PageBlock } from '$lib/api/contexts/types';
import { generateBlockId } from '../entities/block';

// ============================================================================
// Types
// ============================================================================

/**
 * Creation = AI-generated content (renamed from Artifact for taxonomy)
 */
export interface Creation extends ArtifactType_ {}
export interface CreationListItem extends ArtifactListItemType_ {}
export type CreationType = ArtifactType;

/**
 * Options for saving a creation to the Knowledge Base
 */
export interface SaveToKBOptions {
	/** Title for the page (defaults to creation title) */
	title?: string;
	/** Parent page ID */
	parentId?: string;
	/** Link to a Node in the OS */
	nodeId?: string;
	/** Page type (uses ContextType from API) */
	type?: ContextType;
	/** Custom icon */
	icon?: string;
	/** Include creation metadata as properties */
	includeMetadata?: boolean;
}

/**
 * Options for creating a creation from a message
 */
export interface CreateFromMessageOptions {
	/** Title for the creation */
	title: string;
	/** Type of creation */
	type?: CreationType;
	/** Summary of the creation */
	summary?: string;
	/** Project to link to */
	projectId?: string;
}

// ============================================================================
// Creation (Artifact) Operations
// ============================================================================

/**
 * Get all creations
 */
export async function getCreations(filters?: {
	type?: string;
	conversationId?: string;
	projectId?: string;
	contextId?: string;
	unassignedOnly?: boolean;
}): Promise<CreationListItem[]> {
	return artifactsApi.getArtifacts(filters);
}

/**
 * Get a single creation by ID
 */
export async function getCreation(id: string): Promise<Creation> {
	return artifactsApi.getArtifact(id);
}

/**
 * Create a new creation
 */
export async function createCreation(data: CreateArtifactData): Promise<Creation> {
	return artifactsApi.createArtifact(data);
}

/**
 * Update a creation
 */
export async function updateCreation(
	id: string,
	data: { title?: string; content?: string; summary?: string }
): Promise<Creation> {
	return artifactsApi.updateArtifact(id, data);
}

/**
 * Delete a creation
 */
export async function deleteCreation(id: string): Promise<void> {
	await artifactsApi.deleteArtifact(id);
}

/**
 * Link a creation to a project or context
 */
export async function linkCreation(
	id: string,
	options: { projectId?: string; contextId?: string; syncToKB?: boolean }
): Promise<Creation> {
	return artifactsApi.linkArtifact(id, {
		project_id: options.projectId,
		context_id: options.contextId,
		sync_to_kb: options.syncToKB
	});
}

// ============================================================================
// Knowledge Base Integration
// ============================================================================

/**
 * Save a creation to the Knowledge Base as a Page
 */
export async function saveCreationToKB(
	creationId: string,
	options: SaveToKBOptions = {}
): Promise<string> {
	// Fetch the full creation
	const creation = await getCreation(creationId);

	// Convert creation content to page blocks
	const blocks = creationContentToBlocks(creation);

	// Build page data
	const pageData: CreatePageData = {
		name: options.title || creation.title,
		type: options.type || 'document',
		icon: options.icon || getCreationIcon(creation.type),
		parent_id: options.parentId,
		blocks
	};

	// Create the page
	const page = await contextsApi.createContext(pageData);

	// Link the creation to the page
	await linkCreation(creationId, { contextId: page.id });

	return page.id;
}

/**
 * Convert creation content to page blocks
 */
function creationContentToBlocks(creation: Creation): PageBlock[] {
	const blocks: PageBlock[] = [];
	const content = creation.content;

	// Detect content type and parse accordingly
	if (creation.type === 'code' || creation.language) {
		// Code creation - wrap in code block
		blocks.push({
			id: generateBlockId(),
			type: 'code',
			content: content,
			properties: {
				language: creation.language || detectLanguage(content)
			}
		});
	} else if (creation.type === 'markdown' || isMarkdown(content)) {
		// Markdown content - parse into blocks
		blocks.push(...parseMarkdownToBlocks(content));
	} else {
		// Plain text - split into paragraphs
		const paragraphs = content.split(/\n\n+/);
		for (const para of paragraphs) {
			if (para.trim()) {
				blocks.push({
					id: generateBlockId(),
					type: 'paragraph',
					content: para.trim()
				});
			}
		}
	}

	// Add summary as a callout if present
	if (creation.summary) {
		blocks.unshift({
			id: generateBlockId(),
			type: 'callout',
			content: creation.summary,
			properties: {
				icon: '📝',
				color: 'blue'
			}
		});
	}

	return blocks;
}

/**
 * Parse markdown content into page blocks
 */
function parseMarkdownToBlocks(markdown: string): PageBlock[] {
	const blocks: PageBlock[] = [];
	const lines = markdown.split('\n');
	let currentBlock: PageBlock | null = null;
	let inCodeBlock = false;
	let codeLanguage = '';
	let codeContent: string[] = [];

	for (const line of lines) {
		// Code block handling
		if (line.startsWith('```')) {
			if (inCodeBlock) {
				// End code block
				blocks.push({
					id: generateBlockId(),
					type: 'code',
					content: codeContent.join('\n'),
					properties: { language: codeLanguage || 'plain' }
				});
				inCodeBlock = false;
				codeContent = [];
				codeLanguage = '';
			} else {
				// Start code block
				inCodeBlock = true;
				codeLanguage = line.slice(3).trim();
			}
			continue;
		}

		if (inCodeBlock) {
			codeContent.push(line);
			continue;
		}

		// Headings
		const headingMatch = line.match(/^(#{1,3})\s+(.+)$/);
		if (headingMatch) {
			blocks.push({
				id: generateBlockId(),
				type: 'heading',
				content: headingMatch[2],
				properties: { level: headingMatch[1].length }
			});
			continue;
		}

		// Bullet lists
		if (line.match(/^[-*]\s+/)) {
			blocks.push({
				id: generateBlockId(),
				type: 'bulleted_list',
				content: line.replace(/^[-*]\s+/, '')
			});
			continue;
		}

		// Numbered lists
		if (line.match(/^\d+\.\s+/)) {
			blocks.push({
				id: generateBlockId(),
				type: 'numbered_list',
				content: line.replace(/^\d+\.\s+/, '')
			});
			continue;
		}

		// Blockquotes
		if (line.startsWith('> ')) {
			blocks.push({
				id: generateBlockId(),
				type: 'quote',
				content: line.slice(2)
			});
			continue;
		}

		// Horizontal rule
		if (line.match(/^---+$/) || line.match(/^\*\*\*+$/)) {
			blocks.push({
				id: generateBlockId(),
				type: 'divider',
				content: ''
			});
			continue;
		}

		// Regular paragraph
		if (line.trim()) {
			blocks.push({
				id: generateBlockId(),
				type: 'paragraph',
				content: line
			});
		}
	}

	// Handle unclosed code block
	if (inCodeBlock && codeContent.length > 0) {
		blocks.push({
			id: generateBlockId(),
			type: 'code',
			content: codeContent.join('\n'),
			properties: { language: codeLanguage || 'plain' }
		});
	}

	return blocks;
}

/**
 * Extract a creation from a conversation message
 */
export async function createCreationFromMessage(
	message: Message,
	conversationId: string,
	options: CreateFromMessageOptions
): Promise<Creation> {
	// Extract content - prefer code blocks if present
	let content = message.content;
	let detectedType: CreationType = options.type || 'document';
	let language: string | undefined;

	// Check for code blocks in message
	if (message.blocks && message.blocks.length > 0) {
		const codeBlock = message.blocks.find((b) => b.type === 'code');
		if (codeBlock && codeBlock.content) {
			content = codeBlock.content;
			detectedType = 'code';
			language = codeBlock.language;
		}
	}

	// Create the artifact
	return createCreation({
		title: options.title,
		content,
		type: detectedType,
		summary: options.summary,
		conversation_id: conversationId,
		project_id: options.projectId
	});
}

/**
 * Get creations for a conversation
 */
export async function getConversationCreations(
	conversationId: string
): Promise<CreationListItem[]> {
	return getCreations({ conversationId });
}

/**
 * Get creations for a project
 */
export async function getProjectCreations(projectId: string): Promise<CreationListItem[]> {
	return getCreations({ projectId });
}

// ============================================================================
// Utilities
// ============================================================================

/**
 * Get icon for creation type
 */
export function getCreationIcon(type: CreationType): string {
	const icons: Record<CreationType, string> = {
		proposal: '📋',
		sop: '📖',
		framework: '🏗️',
		agenda: '📅',
		report: '📊',
		plan: '🗺️',
		code: '💻',
		document: '📄',
		markdown: '📝',
		other: '📎'
	};
	return icons[type] ?? '📄';
}

/**
 * Detect if content is markdown
 */
function isMarkdown(content: string): boolean {
	// Check for common markdown patterns
	return (
		/^#{1,6}\s/m.test(content) || // Headings
		/^[-*]\s/m.test(content) || // Lists
		/```/.test(content) || // Code blocks
		/^\d+\.\s/m.test(content) || // Numbered lists
		/^>/m.test(content) // Blockquotes
	);
}

/**
 * Detect programming language from content
 */
function detectLanguage(content: string): string {
	// Simple heuristics
	if (content.includes('func ') && content.includes('package ')) return 'go';
	if (content.includes('function ') || content.includes('const ') || content.includes('=>'))
		return 'typescript';
	if (content.includes('def ') && content.includes(':')) return 'python';
	if (content.includes('<?php')) return 'php';
	if (content.includes('<template>') || content.includes('<script')) return 'svelte';
	if (content.includes('import React') || content.includes('jsx')) return 'tsx';
	if (content.includes('SELECT ') || content.includes('FROM ')) return 'sql';
	if (content.startsWith('{') && content.endsWith('}')) return 'json';
	if (content.includes('<!DOCTYPE') || content.includes('<html')) return 'html';
	if (content.includes('@media') || content.includes('{') && content.includes(':'))
		return 'css';
	return 'plain';
}

/**
 * Get creation type from content analysis
 */
export function detectCreationType(content: string, title?: string): CreationType {
	const lowerContent = content.toLowerCase();
	const lowerTitle = title?.toLowerCase() || '';

	if (lowerTitle.includes('proposal') || lowerContent.includes('proposed solution'))
		return 'proposal';
	if (lowerTitle.includes('sop') || lowerContent.includes('standard operating'))
		return 'sop';
	if (lowerTitle.includes('framework') || lowerContent.includes('architecture'))
		return 'framework';
	if (lowerTitle.includes('agenda') || lowerContent.includes('meeting agenda'))
		return 'agenda';
	if (lowerTitle.includes('report') || lowerContent.includes('findings'))
		return 'report';
	if (lowerTitle.includes('plan') || lowerContent.includes('action items'))
		return 'plan';
	if (content.includes('```') || content.includes('function ') || content.includes('class '))
		return 'code';
	if (isMarkdown(content)) return 'markdown';

	return 'document';
}

// ============================================================================
// Deprecated aliases for backwards compatibility
// ============================================================================

/** @deprecated Use getCreations instead */
export const getArtifacts = getCreations;

/** @deprecated Use getCreation instead */
export const getArtifact = getCreation;

/** @deprecated Use createCreation instead */
export const createArtifact = createCreation;
