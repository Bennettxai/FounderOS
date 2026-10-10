// Knowledge Base Module - Main Exports
// Clean, modular Notion-like document system with Yjs CRDT support

// Types (Legacy document types)
export * from "./entities/types";

// Block System Types
export type {
  BlockFlavour,
  BlockSysProps,
  Block,
  BlockSnapshot,
  BlockCollection,
  BlockProps,
  BlockPropsMap,
  BlockIcon,
  TextDelta,
  TextAttributes,
  YTextRef,
  // Content block props
  PageBlockProps,
  ParagraphBlockProps,
  HeadingBlockProps,
  ListBlockProps,
  CodeBlockProps,
  QuoteBlockProps,
  CalloutBlockProps,
  DividerBlockProps,
  ImageBlockProps,
  BookmarkBlockProps,
  EmbedBlockProps,
  // Database block props
  DatabaseBlockProps,
  DatabaseRowBlockProps,
  ColumnSchema,
  ColumnType,
  ColumnTypeData,
  CellValue,
  DatabaseView,
  DatabaseViewType,
  FilterGroup,
  FilterCondition,
  FilterOperator,
  SortConfig,
  SelectOption,
  RollupAggregation,
  // Layout block props
  ColumnLayoutBlockProps,
  ColumnBlockProps,
  SyncedBlockBlockProps,
  LinkToPageBlockProps,
} from "./entities/block";

export {
  generateBlockId,
  isContainerFlavour,
  hasTextContent,
  getDefaultProps,
} from "./entities/block";

// Block Schemas (Zod validation)
export {
  blockSchema,
  blockFlavourSchema,
  blockPropsSchemaMap,
  validateBlock,
  validateBlockProps,
  // Content schemas
  pageBlockPropsSchema,
  paragraphBlockPropsSchema,
  headingBlockPropsSchema,
  listBlockPropsSchema,
  codeBlockPropsSchema,
  quoteBlockPropsSchema,
  calloutBlockPropsSchema,
  dividerBlockPropsSchema,
  imageBlockPropsSchema,
  bookmarkBlockPropsSchema,
  embedBlockPropsSchema,
  // Database schemas
  databaseBlockPropsSchema,
  databaseRowBlockPropsSchema,
  columnSchemaSchema,
  cellValueSchema,
  databaseViewSchema,
} from "./entities/schemas";

// Legacy Stores (Document-based)
export {
  documentsStore,
  activeDocumentStore,
  treeStore,
  sidebarStore,
  activeDocument,
  rootDocuments,
  favoriteDocuments,
  recentDocuments,
  documentMetas,
  documentTree,
  getChildren,
} from "./stores/documents";

// Yjs Block Stores (CRDT-backed)
export {
  createYjsDocStore,
  createBlockStore,
  createBlockDerived,
  createChildrenDerived,
  createRootBlockDerived,
  activeDocStore,
  type YjsDocStore,
  type BlockStore,
} from "./stores/yjs-block-store";

// Database Store (DataSource layer)
export {
  createDatabaseStore,
  createColumnDerived,
  createActiveViewDerived,
  createFilteredRowsDerived,
  createCellDerived,
  type DatabaseStore,
  type DatabaseState,
  type CellUpdate,
  type ColumnUpdate,
} from "./stores/database-store";

// Services
export {
  // Legacy document service
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
  // Share & Export
  enableSharing,
  disableSharing,
  exportDocumentAsMarkdown,
  exportDocumentAsJSON,
  // Page Adapter (Block <-> Page API bridge)
  pageToBlocks,
  blocksToPage,
  nodeTreeToDocumentTree,
  getPageIcon,
  getNodeIcon,
  mergePageMeta,
  // Knowledge Base Service
  knowledgeBase,
  initializeKnowledgeBase,
  getDocumentById,
  getDocumentPath,
  type KBDocument,
  type KnowledgeBaseState,
  type LoadedPage,
  // AI Integration Service (Creations <-> Knowledge Base)
  getCreations,
  getCreation,
  createCreation,
  updateCreation,
  deleteCreation,
  linkCreation,
  saveCreationToKB,
  createCreationFromMessage,
  getConversationCreations,
  getProjectCreations,
  getCreationIcon,
  detectCreationType,
  type Creation,
  type CreationListItem,
  type CreationType,
  type SaveToKBOptions,
  type CreateFromMessageOptions,
  // Deprecated aliases (for backwards compatibility)
  contextToBlocks,
  blocksToContext,
  getContextIcon,
  mergeContextMeta,
  knowledgeGraph,
  initializeKnowledgeGraph,
  type KnowledgeGraphDocument,
  type KnowledgeGraphState,
  type LoadedDocument,
  getArtifacts,
  getArtifact,
  createArtifact,
} from "./services";

// Sidebar Components
export {
  KBSidebar,
  SidebarHeader,
  SidebarSection,
  SidebarTreeItem,
} from "./views/sidebar";
export { default as QuickSearch } from "./views/sidebar/QuickSearch.svelte";

// Editor Components
export {
  DocumentEditor,
  EditorHeader,
  EditorToolbar,
  BlockRenderer,
} from "./views/editor";

// Database Components
export {
  Database,
  DatabaseTable,
  DatabaseViewTabs,
  TableHeader,
  TableCell,
  ColumnTypeIcon,
} from "./views/database";

// Graph View Components
export { GraphView, OptimalGraphView } from "./views/graph";

// Page View Components
export {
  EmptyPageView,
  FolderContentsView,
  GraphFrame,
  PageListLayout,
  ProfilesView,
  RecentDocumentsView,
  StatusView,
  type PagesNodeInfo,
  type RecentDocumentItem,
} from "./views/page";
