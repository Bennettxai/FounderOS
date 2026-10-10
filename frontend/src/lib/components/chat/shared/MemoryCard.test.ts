import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';
import { mockMemory } from '$lib/test-utils/mocks';
import MemoryCard from './MemoryCard.svelte';

describe('MemoryCard', () => {
	it('renders memory title and summary', () => {
		const memory = mockMemory({
			title: 'Test Memory Title',
			summary: 'This is a test summary'
		});

		render(MemoryCard, { props: { memory } });

		expect(screen.getByText('Test Memory Title')).toBeInTheDocument();
		expect(screen.getByText('This is a test summary')).toBeInTheDocument();
	});

	it('displays memory type badge', () => {
		const memory = mockMemory({ memory_type: 'fact' });

		render(MemoryCard, { props: { memory } });

		expect(screen.getByText('fact')).toBeInTheDocument();
	});

	it('shows importance score as percentage', () => {
		const memory = mockMemory({ importance_score: 0.85 });

		render(MemoryCard, { props: { memory } });

		expect(screen.getByText('85%')).toBeInTheDocument();
	});

	it('displays tags when present', () => {
		const memory = mockMemory({
			tags: ['javascript', 'testing', 'vitest']
		});

		render(MemoryCard, { props: { memory } });

		expect(screen.getByText('javascript')).toBeInTheDocument();
		expect(screen.getByText('testing')).toBeInTheDocument();
		expect(screen.getByText('vitest')).toBeInTheDocument();
	});

	it('shows +N indicator when more than 3 tags', () => {
		const memory = mockMemory({
			tags: ['tag1', 'tag2', 'tag3', 'tag4', 'tag5']
		});

		render(MemoryCard, { props: { memory } });

		expect(screen.getByText('+2')).toBeInTheDocument();
	});

	it('shows access count when greater than 0', () => {
		const memory = mockMemory({ access_count: 5 });

		render(MemoryCard, { props: { memory } });

		expect(screen.getByText('5')).toBeInTheDocument();
	});

	it('applies pinned styling when memory is pinned', () => {
		const memory = mockMemory({ is_pinned: true });

		const { container } = render(MemoryCard, { props: { memory } });
		const card = container.querySelector('.memory-card');

		expect(card).toHaveClass('pinned');
	});

	it('calls onClick when card is clicked', async () => {
		const memory = mockMemory();
		const onClick = vi.fn();

		const { container } = render(MemoryCard, { props: { memory, onClick } });
		const card = container.querySelector('.memory-card');

		if (card) {
			await fireEvent.click(card);
			expect(onClick).toHaveBeenCalledWith(memory);
		}
	});

	it('calls onClick when Enter key is pressed', async () => {
		const memory = mockMemory();
		const onClick = vi.fn();

		const { container } = render(MemoryCard, { props: { memory, onClick } });
		const card = container.querySelector('.memory-card');

		if (card) {
			await fireEvent.keyDown(card, { key: 'Enter' });
			expect(onClick).toHaveBeenCalledWith(memory);
		}
	});

	it('shows delete button when onDelete handler is provided', () => {
		const memory = mockMemory();
		const onDelete = vi.fn();

		render(MemoryCard, { props: { memory, onDelete } });

		const deleteButton = screen.getByLabelText('Delete memory');
		expect(deleteButton).toBeInTheDocument();
	});

	it('does not show delete button when onDelete handler is not provided', () => {
		const memory = mockMemory();

		render(MemoryCard, { props: { memory } });

		const deleteButton = screen.queryByLabelText('Delete memory');
		expect(deleteButton).not.toBeInTheDocument();
	});

	it('formats dates correctly for today', () => {
		const now = new Date();
		const memory = mockMemory({ created_at: now.toISOString() });

		render(MemoryCard, { props: { memory } });

		expect(screen.getByText('Today')).toBeInTheDocument();
	});

	it('formats dates correctly for yesterday', () => {
		const yesterday = new Date();
		yesterday.setDate(yesterday.getDate() - 1);
		const memory = mockMemory({ created_at: yesterday.toISOString() });

		render(MemoryCard, { props: { memory } });

		expect(screen.getByText('Yesterday')).toBeInTheDocument();
	});

	it('formats dates correctly for days ago', () => {
		const threeDaysAgo = new Date();
		threeDaysAgo.setDate(threeDaysAgo.getDate() - 3);
		const memory = mockMemory({ created_at: threeDaysAgo.toISOString() });

		render(MemoryCard, { props: { memory } });

		expect(screen.getByText('3 days ago')).toBeInTheDocument();
	});

	it('has correct accessibility attributes', () => {
		const memory = mockMemory();

		const { container } = render(MemoryCard, { props: { memory } });
		const card = container.querySelector('.memory-card');

		expect(card).toHaveAttribute('role', 'button');
		expect(card).toHaveAttribute('tabindex', '0');
	});

	it('renders pin button with correct aria-label when unpinned', () => {
		const memory = mockMemory({ is_pinned: false });

		render(MemoryCard, { props: { memory } });

		const pinButton = screen.getByLabelText('Pin memory');
		expect(pinButton).toBeInTheDocument();
	});

	it('renders pin button with correct aria-label when pinned', () => {
		const memory = mockMemory({ is_pinned: true });

		render(MemoryCard, { props: { memory } });

		const pinButton = screen.getByLabelText('Unpin memory');
		expect(pinButton).toBeInTheDocument();
	});

	it('stops event propagation when pin button is clicked', async () => {
		const memory = mockMemory();
		const onClick = vi.fn();

		render(MemoryCard, { props: { memory, onClick } });

		const pinButton = screen.getByLabelText('Pin memory');
		await fireEvent.click(pinButton);

		// onClick should not be called because event propagation was stopped
		expect(onClick).not.toHaveBeenCalled();
	});

	it('stops event propagation when delete button is clicked', async () => {
		const memory = mockMemory();
		const onClick = vi.fn();
		const onDelete = vi.fn();

		render(MemoryCard, { props: { memory, onClick, onDelete } });

		const deleteButton = screen.getByLabelText('Delete memory');
		await fireEvent.click(deleteButton);

		// onClick should not be called because event propagation was stopped
		expect(onClick).not.toHaveBeenCalled();
		expect(onDelete).toHaveBeenCalledWith(memory);
	});
});
