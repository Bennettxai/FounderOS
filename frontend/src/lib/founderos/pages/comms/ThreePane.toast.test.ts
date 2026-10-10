// Archive/snooze confirm through the OS-wide toaster, as FounderOS v1's
// CommsThreePane does (useToast().ok(`${verb} — ${sender}`, undo)), not a
// page-local toast.
import { fireEvent, render, screen } from '@testing-library/svelte';
import { get } from 'svelte/store';
import { beforeEach, describe, expect, it } from 'vitest';
import { toasts } from '$lib/founderos/chrome/toast';
import ThreePane from './ThreePane.svelte';
import type { Lane } from './model';

const lanes: Lane[] = [
	{
		id: 'inbox-1', name: 'Ops', source: 'email', state: 'connected', detail: 'ops inbox', unread: 1,
		items: [{ id: 'a', sender: 'Ada Example', preview: 'hello', ts: '2026-09-07T10:00:00.000Z', unread: 1 }]
	}
];

beforeEach(() => toasts.set([]));

describe('ThreePane confirmations', () => {
	it('archiving a message raises an OS toast with undo, and no page-local toast', async () => {
		const { container } = render(ThreePane, { lanes, slackCards: [], slackRoster: { state: 'connected' }, channels: [], nowMs: Date.parse('2026-09-07T12:00:00Z') });
		await fireEvent.click(screen.getByText('Ada Example'));
		await fireEvent.click(screen.getByRole('button', { name: /archive/i }));
		const [t] = get(toasts);
		expect(t).toMatchObject({ kind: 'ok', text: 'Archived — Ada Example' });
		expect(typeof t.undo).toBe('function');
		expect(container.querySelector('[data-part="toast"]')).toBeNull();
	});
});
