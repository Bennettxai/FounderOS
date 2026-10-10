/**
 * Knowledge Base Services
 *
 * Exports all services for the Knowledge Base module.
 *
 * TAXONOMY:
 * - Page = Document in Knowledge Base
 * - Creation = AI-generated output (code, documents, etc.)
 * - Node = OS layer building block
 */

// Page Adapter (Block <-> Page API bridge)
export {
	pageToBlocks,
	blocksToPage,
	nodeTreeToDocumentTree,
	getPageIcon,
	getNodeIcon,
	mergePageMeta,
	// Deprecated aliases
	contextToBlocks,
	blocksToContext,
	getContextIcon,
	mergeContextMeta
} from './page-adapter';

// Knowledge Base Service (main service)
export {
	knowledgeBase,
	initializeKnowledgeBase,
	getDocumentById,
	getDocumentPath,
	type KBDocument,
	type KnowledgeBaseState,
	type LoadedPage,
	// Deprecated aliases
	knowledgeGraph,
	initializeKnowledgeGraph,
	type KnowledgeGraphDocument,
	type KnowledgeGraphState,
	type LoadedDocument
} from './knowledge-base.service';

// AI Integration Service (Creations <-> Knowledge Base)
export {
	// Creation operations
	getCreations,
	getCreation,
	createCreation,
	updateCreation,
	deleteCreation,
	linkCreation,
	// Knowledge Base integration
	saveCreationToKB,
	createCreationFromMessage,
	getConversationCreations,
	getProjectCreations,
	// Utilities
	getCreationIcon,
	detectCreationType,
	// Types
	type Creation,
	type CreationListItem,
	type CreationType,
	type SaveToKBOptions,
	type CreateFromMessageOptions,
	// Deprecated aliases
	getArtifacts,
	getArtifact,
	createArtifact
} from './ai-integration.service';

// Legacy document service (will be replaced)
export {
	fetchDocuments,
	fetchDocument,
	createDocument,
	updateDocument,
	deleteDocument,
	restoreDocument,
	moveDocument,
	duplicateDocument,
	toggleFavorite,
	createEmptyParagraph,
	createBlock,
	searchDocuments,
	openDocument,
	closeDocument,
	openAndFetchDocument,
	enableSharing,
	disableSharing,
	exportDocumentAsMarkdown,
	exportDocumentAsJSON
} from './documents.service';
