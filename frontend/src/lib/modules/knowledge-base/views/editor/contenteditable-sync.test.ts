import { describe, expect, it } from 'vitest';
import { parseInlineMarkdownToRichText } from '../../utils/rich-text';
import {
	createContenteditableSyncAction,
	createContenteditableSyncState,
	getContenteditablePlainText,
	renderContenteditableHTML,
	shouldSyncContenteditableDOM,
	syncContenteditableDOM
} from './contenteditable-sync';

describe('contenteditable rich-text sync helpers', () => {
	it('creates tracked state from a block snapshot', () => {
		const snapshot = {
			blockId: 'block-1',
			content: parseInlineMarkdownToRichText('Hello **world**')
		};

		expect(createContenteditableSyncState(snapshot)).toEqual({
			lastBlockId: 'block-1',
			lastContent: 'Hello world'
		});
		expect(getContenteditablePlainText(snapshot)).toBe('Hello world');
	});

	it('renders rich text into editor html', () => {
		const node = document.createElement('div');

		renderContenteditableHTML(node, {
			blockId: 'block-1',
			content: parseInlineMarkdownToRichText('Hello **world**')
		});

		expect(node.innerHTML).toBe('Hello <strong>world</strong>');
	});

	it('syncs when the block id changes', () => {
		const node = document.createElement('div');
		const state = createContenteditableSyncState({
			blockId: 'block-1',
			content: parseInlineMarkdownToRichText('Old')
		});

		const synced = syncContenteditableDOM(
			node,
			state,
			{ blockId: 'block-2', content: parseInlineMarkdownToRichText('New') },
			node
		);

		expect(synced).toBe(true);
		expect(node.innerHTML).toBe('New');
		expect(state).toEqual({ lastBlockId: 'block-2', lastContent: 'New' });
	});

	it('syncs external content changes only while the node is inactive', () => {
		const node = document.createElement('div');
		const state = createContenteditableSyncState({
			blockId: 'block-1',
			content: parseInlineMarkdownToRichText('Original')
		});
		const snapshot = { blockId: 'block-1', content: parseInlineMarkdownToRichText('External') };

		expect(shouldSyncContenteditableDOM(state, snapshot, true)).toBe(false);
		expect(syncContenteditableDOM(node, state, snapshot, node)).toBe(false);
		expect(node.innerHTML).toBe('');
		expect(state).toEqual({ lastBlockId: 'block-1', lastContent: 'Original' });

		expect(shouldSyncContenteditableDOM(state, snapshot, false)).toBe(true);
		expect(syncContenteditableDOM(node, state, snapshot, null)).toBe(true);
		expect(node.innerHTML).toBe('External');
		expect(state).toEqual({ lastBlockId: 'block-1', lastContent: 'External' });
	});

	it('creates a Svelte action-compatible sync wrapper', () => {
		const node = document.createElement('div');
		const seenNodes: Array<HTMLElement | null> = [];
		let snapshot = {
			blockId: 'block-1',
			content: parseInlineMarkdownToRichText('Initial')
		};
		const state = createContenteditableSyncState(snapshot);
		const action = createContenteditableSyncAction({
			getSnapshot: () => snapshot,
			state,
			getActiveElement: () => null,
			onElementChange: (nextNode) => seenNodes.push(nextNode)
		})(node);

		expect(node.innerHTML).toBe('Initial');
		expect(seenNodes).toEqual([node]);

		snapshot = { blockId: 'block-1', content: parseInlineMarkdownToRichText('Updated') };
		action.update();

		expect(node.innerHTML).toBe('Updated');
		expect(state.lastContent).toBe('Updated');

		action.destroy();
		expect(seenNodes).toEqual([node, null]);
	});
});
