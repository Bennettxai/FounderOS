// FounderOS v1 components/ConductorComposer.tsx: every control works, none is decoration.
import { fireEvent, render, screen, waitFor } from '@testing-library/svelte';
import { describe, expect, it, vi } from 'vitest';
import ConductorComposer from './ConductorComposer.svelte';

const box = () => screen.getByRole('textbox', { name: /message the conductor/i }) as HTMLTextAreaElement;

describe('ConductorComposer', () => {
	it('Enter sends the trimmed text and empties the box; blank sends nothing', async () => {
		const onSend = vi.fn();
		render(ConductorComposer, { onSend });
		await fireEvent.keyDown(box(), { key: 'Enter' });
		expect(onSend).not.toHaveBeenCalled();
		await fireEvent.input(box(), { target: { value: '  hi there  ' } });
		await fireEvent.keyDown(box(), { key: 'Enter' });
		expect(onSend).toHaveBeenCalledWith('hi there', 'hi there');
		expect(box().value).toBe('');
	});

	it('the send button is off while disabled or empty', async () => {
		render(ConductorComposer, { onSend: vi.fn(), disabled: true });
		await fireEvent.input(box(), { target: { value: 'x' } });
		expect((screen.getByRole('button', { name: 'Send' }) as HTMLButtonElement).disabled).toBe(true);
	});

	it('an attached text file rides into the message, honestly labeled; the bubble shows only the text', async () => {
		const onSend = vi.fn();
		const { container } = render(ConductorComposer, { onSend });
		const file = new File(['line one\nline two'], 'notes.md', { type: 'text/markdown' });
		// jsdom's File has no text(); every browser the dock runs in does
		if (typeof file.text !== 'function') Object.defineProperty(file, 'text', { value: async () => 'line one\nline two' });
		const picker = container.querySelector('input[type="file"]') as HTMLInputElement;
		Object.defineProperty(picker, 'files', { value: [file] });
		await fireEvent.change(picker);
		await waitFor(() => expect(screen.getByText('notes.md')).toBeTruthy());
		await fireEvent.input(box(), { target: { value: 'summarize' } });
		await fireEvent.keyDown(box(), { key: 'Enter' });
		expect(onSend).toHaveBeenCalledWith('summarize\n\n[Attached: notes.md]\nline one\nline two', 'summarize');
		expect(screen.queryByText('notes.md')).toBeNull();
	});

	it('shows the live model chip only when the seat is known', () => {
		const { unmount } = render(ConductorComposer, { onSend: vi.fn(), model: 'claude-haiku-4-5' });
		expect(screen.getByText('claude-haiku-4-5')).toBeTruthy();
		unmount();
		render(ConductorComposer, { onSend: vi.fn(), model: null });
		expect(screen.queryByTitle(/model holding the Conductor seat/)).toBeNull();
	});

	it('the mic says so when the browser has no speech recognition', async () => {
		const onError = vi.fn();
		render(ConductorComposer, { onSend: vi.fn(), onError });
		await fireEvent.click(screen.getByRole('button', { name: /dictate/i }));
		expect(onError).toHaveBeenCalledWith('Voice input is not available in this browser.');
	});
});
