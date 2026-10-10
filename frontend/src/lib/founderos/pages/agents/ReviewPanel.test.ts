import { fireEvent, render, screen, waitFor } from '@testing-library/svelte';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import ReviewPanel from './ReviewPanel.svelte';
import { actionTextOf, jsonBodyOf } from './review';
import type { DeliverableItem } from './types';

const json = (body: unknown, status = 200) => new Response(JSON.stringify(body), { status, headers: { 'content-type': 'application/json' } });

const item = (over: Partial<DeliverableItem> = {}): DeliverableItem => ({
	id: 'ws/STAGED-reply-to-sam.md',
	name: 'STAGED-reply-to-sam.md',
	kind: 'file',
	url: null,
	meta: 'f7d1ec4a',
	modifiedAt: new Date().toISOString(),
	sizeBytes: 40,
	accessCode: '',
	title: 'Reply to Sam',
	summary: '',
	revision: 'file|ws/STAGED-reply-to-sam.md|40',
	...over
});

let fetchMock: ReturnType<typeof vi.fn>;
beforeEach(() => {
	fetchMock = vi.fn((url: string) =>
		Promise.resolve(
			url.includes('mode=view')
				? json({ id: 'x', name: 'x.md', kind: 'text', text: '# Reply\n\nThanks **Sam**.\n\n| a | b |\n|---|---|\n| 1 | 2 |', truncated: true })
				: json({})
		)
	);
	vi.stubGlobal('fetch', fetchMock);
});
afterEach(() => {
	vi.unstubAllGlobals();
	document.body.innerHTML = '';
});

describe('review helpers (TaskReviewPanel.tsx)', () => {
	it('says what approve and dismiss do, by the kind the file name asks for', () => {
		expect(actionTextOf('STAGED-reply.md')).toBe('The agent already wrote this. Approve to send it. Dismiss to bin it.');
		expect(actionTextOf('fos-510-decision-priced.md')).toBe('Only you can make this call. Approve to go ahead. Dismiss to drop it.');
		expect(actionTextOf('launch-draft-v2.md')).toBe('A draft is waiting on you. Approve to publish it. Dismiss to send it back.');
		expect(actionTextOf('STAGED-reply-RESOLVED.md')).toBe('Finished work. Nothing is being asked of you.');
		expect(actionTextOf('audit.md')).toBe('Reference output. Nothing is being asked of you.');
		expect(actionTextOf('memo-DELIVER-BEFORE-1430Z.md')).toBe('Someone asked you for something. Approve to act on it. Dismiss to decline.');
	});

	it('a .json deliverable shows the document inside its envelope', () => {
		expect(jsonBodyOf('post.json', JSON.stringify({ body: 'Hello **there**' }))).toBe('Hello **there**');
		expect(jsonBodyOf('post.json', '{"x":1}')).toBe('{"x":1}');
		expect(jsonBodyOf('post.md', '{"body":"y"}')).toBe('{"body":"y"}');
	});
});

describe('ReviewPanel (TaskReviewPanel.tsx)', () => {
	it('slides in over the page: Review label, the document title, name · meta, the action line', async () => {
		render(ReviewPanel, { item: item(), onDecide: vi.fn(), onClose: vi.fn() });
		const panel = await screen.findByRole('dialog');
		expect(panel.getAttribute('data-part')).toBe('review-overlay');
		expect(panel.parentElement).toBe(document.body);
		expect(screen.getByText('Review')).toBeTruthy();
		expect(screen.getByText('Reply to Sam')).toBeTruthy();
		expect(screen.getByText(/STAGED-reply-to-sam\.md · f7d1ec4a/)).toBeTruthy();
		expect(screen.getByText('The agent already wrote this. Approve to send it. Dismiss to bin it.')).toBeTruthy();
	});

	it('renders the file as a document, not a raw pre, and warns when truncated', async () => {
		render(ReviewPanel, { item: item(), onDecide: vi.fn(), onClose: vi.fn() });
		expect(await screen.findByRole('heading', { name: 'Reply' })).toBeTruthy();
		expect(screen.getByText('Sam').tagName).toBe('STRONG');
		expect(screen.getByRole('table')).toBeTruthy();
		expect(screen.getByText('Showing the first part only. Download for the whole file.')).toBeTruthy();
	});

	it('approve and dismiss decide; a decided item offers undo instead', async () => {
		const onDecide = vi.fn();
		const { unmount } = render(ReviewPanel, { item: item(), onDecide, onClose: vi.fn() });
		await fireEvent.click(await screen.findByRole('button', { name: /approve/ }));
		expect(onDecide).toHaveBeenLastCalledWith(expect.objectContaining({ id: 'ws/STAGED-reply-to-sam.md' }), 'approved');
		await fireEvent.click(screen.getByRole('button', { name: /^dismiss/ }));
		expect(onDecide).toHaveBeenLastCalledWith(expect.objectContaining({ id: 'ws/STAGED-reply-to-sam.md' }), 'dismissed');
		unmount();
		render(ReviewPanel, {
			item: item(),
			decision: { id: 'x', decision: 'dismissed', decidedAt: '', decidedRevision: '', note: '' },
			onDecide,
			onClose: vi.fn()
		});
		await fireEvent.click(await screen.findByRole('button', { name: /undo dismissed/ }));
		expect(onDecide).toHaveBeenLastCalledWith(expect.anything(), null);
	});

	it('the backdrop and Escape close it; the panel itself does not', async () => {
		const onClose = vi.fn();
		render(ReviewPanel, { item: item(), onDecide: vi.fn(), onClose });
		const overlay = await screen.findByRole('dialog');
		await fireEvent.click(screen.getByText('Reply to Sam'));
		expect(onClose).not.toHaveBeenCalled();
		await fireEvent.click(overlay);
		expect(onClose).toHaveBeenCalledTimes(1);
		await fireEvent.keyDown(window, { key: 'Escape' });
		expect(onClose).toHaveBeenCalledTimes(2);
	});

	it('a proposal is a live page: no fetch, an open page link and its gate code', async () => {
		render(ReviewPanel, {
			item: item({ id: 'p1', name: 'nick', kind: 'link', url: 'https://p.example/nick', title: 'Nick Letizia', accessCode: 'NL-42' }),
			onDecide: vi.fn(),
			onClose: vi.fn()
		});
		expect(await screen.findByText(/live proposal page rather than a file/)).toBeTruthy();
		expect(screen.getByText('NL-42')).toBeTruthy();
		expect(screen.getByRole('link', { name: /open page/ }).getAttribute('href')).toBe('https://p.example/nick');
		await waitFor(() => expect(fetchMock.mock.calls.some(([u]) => String(u).includes('mode=view'))).toBe(false));
	});

	it('a file it cannot read says so', async () => {
		fetchMock.mockImplementation(() => Promise.resolve(json({ error: 'file not found' }, 404)));
		render(ReviewPanel, { item: item(), onDecide: vi.fn(), onClose: vi.fn() });
		expect(await screen.findByText('Could not read this file. It may have been moved or the board host is unreachable.')).toBeTruthy();
	});
});
