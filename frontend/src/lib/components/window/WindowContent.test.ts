import { fireEvent, render, screen, waitFor } from '@testing-library/svelte';
import { get } from 'svelte/store';
import { beforeEach, describe, expect, it } from 'vitest';
import WindowContent from './WindowContent.svelte';
import { windowStore } from '$lib/stores/windowStore';

describe('WindowContent canvas note', () => {
	beforeEach(() => {
		localStorage.clear();
		windowStore.reset();
	});

	it('keeps locally typed note text visible while debounced save runs', async () => {
		windowStore.openWindow('canvas-note-test', {
			title: 'Note',
			data: { text: '' }
		});

		const noteWindow = get(windowStore).windows.find((win) => win.module === 'canvas-note-test');
		expect(noteWindow).toBeTruthy();

		render(WindowContent, {
			props: {
				windowId: noteWindow!.id,
				module: 'canvas-note-test',
				windowTitle: 'Note',
				deployedApps: [],
				workspaceApps: [],
				windowData: noteWindow!.data
			}
		});

		const textarea = screen.getByLabelText('Canvas note') as HTMLTextAreaElement;
		await fireEvent.input(textarea, { target: { value: 'Keep this note' } });

		expect(textarea.value).toBe('Keep this note');

		await waitFor(() => {
			const savedWindow = get(windowStore).windows.find((win) => win.id === noteWindow!.id);
			expect(savedWindow?.data?.text).toBe('Keep this note');
		});
	});
});
