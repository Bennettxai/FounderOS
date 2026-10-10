import type { RichText } from '../../entities/types';
import { extractRichTextPlainText, richTextToHTML } from '../../utils/rich-text';

export interface ContenteditableSyncSnapshot {
	blockId: string;
	content: RichText[] | string | null;
}

export interface ContenteditableSyncState {
	lastBlockId: string;
	lastContent: string;
}

export interface ContenteditableSyncOptions {
	getSnapshot: () => ContenteditableSyncSnapshot;
	state: ContenteditableSyncState;
	getActiveElement?: () => Element | null;
	onElementChange?: (node: HTMLElement | null) => void;
}

export interface ContenteditableSyncAction {
	update: () => void;
	destroy: () => void;
}

export function createContenteditableSyncState(
	snapshot?: ContenteditableSyncSnapshot
): ContenteditableSyncState {
	if (!snapshot) {
		return { lastBlockId: '', lastContent: '' };
	}

	return {
		lastBlockId: snapshot.blockId,
		lastContent: getContenteditablePlainText(snapshot)
	};
}

export function getContenteditablePlainText(snapshot: ContenteditableSyncSnapshot): string {
	return extractRichTextPlainText(snapshot.content);
}

export function renderContenteditableHTML(
	node: HTMLElement,
	snapshot: ContenteditableSyncSnapshot
): void {
	node.innerHTML = richTextToHTML(snapshot.content);
}

export function shouldSyncContenteditableDOM(
	state: ContenteditableSyncState,
	snapshot: ContenteditableSyncSnapshot,
	isActive: boolean
): boolean {
	const currentContent = getContenteditablePlainText(snapshot);
	return snapshot.blockId !== state.lastBlockId || (currentContent !== state.lastContent && !isActive);
}

export function syncContenteditableDOM(
	node: HTMLElement,
	state: ContenteditableSyncState,
	snapshot: ContenteditableSyncSnapshot,
	activeElement: Element | null = typeof document === 'undefined' ? null : document.activeElement
): boolean {
	const currentContent = getContenteditablePlainText(snapshot);

	if (snapshot.blockId !== state.lastBlockId || (currentContent !== state.lastContent && activeElement !== node)) {
		renderContenteditableHTML(node, snapshot);
		state.lastBlockId = snapshot.blockId;
		state.lastContent = currentContent;
		return true;
	}

	return false;
}

export function createContenteditableSyncAction({
	getSnapshot,
	state,
	getActiveElement = () => document.activeElement,
	onElementChange
}: ContenteditableSyncOptions): (node: HTMLElement) => ContenteditableSyncAction {
	return (node: HTMLElement) => {
		onElementChange?.(node);
		renderContenteditableHTML(node, getSnapshot());

		return {
			update() {
				syncContenteditableDOM(node, state, getSnapshot(), getActiveElement());
			},
			destroy() {
				onElementChange?.(null);
			}
		};
	};
}
