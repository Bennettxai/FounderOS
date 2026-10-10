/**
 * Knowledge Base Entity Types
 * Core data structures for the Notion-like document system
 */

// ============================================================================
// Document Types
// ============================================================================

export type DocumentType = 'document' | 'database' | 'folder' | 'whiteboard';
export type DocumentIcon = string | { type: 'emoji' | 'icon' | 'image'; value: string };

export interface Document {
	id: string;
	workspace_id: string;
	parent_id: string | null;
	type: DocumentType;
	title: string;
	icon: DocumentIcon | null;
	cover: string | null;
	content: Block[];
	properties: Record<string, PropertyValue>;
	is_template: boolean;
	is_favorite: boolean;
	is_archived: boolean;
	is_public: boolean;
	share_id: string | null;
	created_at: string;
	updated_at: string;
	created_by: string;
	last_edited_by: string;
}

export interface DocumentMeta {
	id: string;
	parent_id: string | null;
	type: DocumentType;
	title: string;
	icon: DocumentIcon | null;
	is_favorite: boolean;
	is_archived: boolean;
	updated_at: string;
	children_count: number;
}

// ============================================================================
// Block Types (Content Blocks)
// ============================================================================

export type BlockType =
	| 'paragraph'
	| 'heading_1'
	| 'heading_2'
	| 'heading_3'
	| 'bulleted_list'
	| 'numbered_list'
	| 'to_do'
	| 'toggle'
	| 'code'
	| 'quote'
	| 'callout'
	| 'divider'
	| 'image'
	| 'video'
	| 'file'
	| 'bookmark'
	| 'embed'
	| 'table'
	| 'table_row'
	| 'column_list'
	| 'column'
	| 'synced_block'
	| 'template'
	| 'link_to_page'
	| 'equation'
	| 'database_view';

export interface Block {
	id: string;
	type: BlockType;
	content: RichText[];
	properties: BlockProperties;
	children: Block[];
	created_at: string;
	updated_at: string;
}

export type DividerStyle = 'solid' | 'dashed' | 'dotted' | 'thick' | 'double' | 'gradient';

export interface BlockProperties {
	checked?: boolean; // for to_do
	language?: string; // for code
	caption?: RichText[]; // for media blocks
	url?: string; // for bookmark/embed
	color?: string; // for callout
	icon?: string; // for callout
	level?: number; // for headings
	collapsible?: boolean; // for toggle
	collapsed?: boolean; // for toggle
	divider_style?: DividerStyle; // for divider
	expanded?: boolean; // for expandable blocks
	title?: string; // for titled blocks
	description?: string; // for blocks with descriptions
	tableData?: unknown[][]; // for table blocks
}

// ============================================================================
// Rich Text Types
// ============================================================================

export interface RichText {
	type: 'text' | 'mention' | 'equation';
	text?: {
		content: string;
		link: string | null;
	};
	mention?: {
		type: 'user' | 'page' | 'date' | 'database';
		id: string;
	};
	equation?: {
		expression: string;
	};
	annotations: TextAnnotations;
	plain_text: string;
	href: string | null;
}

export interface TextAnnotations {
	bold: boolean;
	italic: boolean;
	strikethrough: boolean;
	underline: boolean;
	code: boolean;
	color: TextColor;
}

export type TextColor =
	| 'default'
	| 'gray'
	| 'brown'
	| 'orange'
	| 'yellow'
	| 'green'
	| 'blue'
	| 'purple'
	| 'pink'
	| 'red'
	| 'gray_background'
	| 'brown_background'
	| 'orange_background'
	| 'yellow_background'
	| 'green_background'
	| 'blue_background'
	| 'purple_background'
	| 'pink_background'
	| 'red_background';

// ============================================================================
// Property Types (for Database Views)
// ============================================================================

export type PropertyType =
	| 'title'
	| 'text'
	| 'number'
	| 'select'
	| 'multi_select'
	| 'date'
	| 'person'
	| 'file'
	| 'checkbox'
	| 'url'
	| 'email'
	| 'phone'
	| 'formula'
	| 'relation'
	| 'rollup'
	| 'created_time'
	| 'created_by'
	| 'last_edited_time'
	| 'last_edited_by';

export interface PropertySchema {
	id: string;
	name: string;
	type: PropertyType;
	options?: SelectOption[];
	formula?: string;
	relation?: {
		database_id: string;
		synced_property_name?: string;
	};
}

export interface SelectOption {
	id: string;
	name: string;
	color: string;
}

export type PropertyValue =
	| { type: 'title'; value: RichText[] }
	| { type: 'text'; value: RichText[] }
	| { type: 'number'; value: number | null }
	| { type: 'select'; value: SelectOption | null }
	| { type: 'multi_select'; value: SelectOption[] }
	| { type: 'date'; value: DateValue | null }
	| { type: 'checkbox'; value: boolean }
	| { type: 'url'; value: string | null }
	| { type: 'email'; value: string | null }
	| { type: 'phone'; value: string | null };

export interface DateValue {
	start: string;
	end: string | null;
	time_zone: string | null;
}

// ============================================================================
// View Types (for Sidebar Navigation)
// ============================================================================

export type SidebarView =
	| 'all'
	| 'favorites'
	| 'recent'
	| 'shared'
	| 'graph'
	| 'knowledge-graph'
	| 'trash'
	| 'profiles'
	| 'profiles-person'
	| 'profiles-business'
	| 'profiles-project';

export type ContextType = 'person' | 'business' | 'project' | 'custom' | 'document';

export interface TreeNode {
	id: string;
	document: DocumentMeta;
	children: TreeNode[];
	isExpanded: boolean;
	isLoading: boolean;
	depth: number;
}

// ============================================================================
// Profile/Context Types (Legacy Support)
// ============================================================================

export interface ContextProfile {
	id: string;
	workspace_id: string;
	name: string;
	description: string | null;
	type: 'person' | 'company' | 'project' | 'concept' | 'custom';
	avatar_url: string | null;
	metadata: Record<string, unknown>;
	linked_documents: string[];
	created_at: string;
	updated_at: string;
}

// ============================================================================
// Search Types
// ============================================================================

export interface SearchResult {
	id: string;
	type: 'document' | 'block' | 'profile';
	title: string;
	preview: string;
	icon: DocumentIcon | null;
	path: string[];
	score: number;
	highlights: string[];
}

export interface SearchFilters {
	types?: DocumentType[];
	parent_id?: string;
	is_favorite?: boolean;
	created_after?: string;
	created_before?: string;
	updated_after?: string;
	updated_before?: string;
}
