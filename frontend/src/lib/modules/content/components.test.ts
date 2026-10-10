import { fireEvent, render, screen } from '@testing-library/svelte';
import { describe, expect, it, vi } from 'vitest';
import type { ContentItem } from '$lib/api/content';
import ContentEditorModal from './ContentEditorModal.svelte';
import ContentOverview from './ContentOverview.svelte';
import ContentPipeline from './ContentPipeline.svelte';
import { buildContentBoards, emptyContentForm } from './model';

const contentItem: ContentItem = {
	id: 'content-1',
	title: 'Workspace launch reel',
	content_type: 'reel',
	status: 'to_edit',
	hook: 'A workspace-specific opening',
	body: '',
	caption: '',
	cta: '',
	channel: 'Instagram',
	link: '',
	category: 'Launch campaign',
	theme: 'Business Diagnosis',
	client: 'Northstar',
	campaign: '',
	owner: 'Morgan Owner',
	editor: 'Alex Editor',
	priority: 'normal',
	due_date: '',
	publish_date: '2026-07-15',
	asset_link: '',
	review_link: '',
	revision_notes: '',
	notes: '',
	views: 0,
	reach: 0,
	likes: 0,
	comments: 0,
	saves: 0,
	shares: 0,
	reposts: 0,
	follows: 0,
	profile_activity: 0,
	accounts_engaged: 0,
	avg_watch_time_seconds: 0,
	retention_rate: 0,
	analytics_notes: '',
	created_by: null,
	created_at: '2026-07-10T00:00:00Z',
	updated_at: '2026-07-10T00:00:00Z'
};

describe('ContentOS components', () => {
	it('renders workspace-derived content instead of compiled profile defaults', () => {
		render(ContentOverview, {
			items: [contentItem],
			workspaceName: 'Northstar workspace',
			onNew: vi.fn(),
			onOpenPipeline: vi.fn(),
			onEdit: vi.fn()
		});

		expect(screen.getByRole('heading', { name: 'Northstar workspace' })).toBeInTheDocument();
		expect(screen.getAllByText('Workspace launch reel').length).toBeGreaterThan(0);
		expect(screen.getByText('Launch campaign')).toBeInTheDocument();
	});

	it('offers active workspace members for owner and editor assignment', () => {
		render(ContentEditorModal, {
			form: emptyContentForm(),
			editing: null,
			profiles: ['Northstar'],
			workstreams: ['Launch campaign'],
			memberOptions: ['Alex Editor', 'Morgan Owner'],
			saving: false,
			onClose: vi.fn(),
			onSave: vi.fn()
		});

		const owner = screen.getByLabelText('Owner') as HTMLSelectElement;
		const editor = screen.getByLabelText('Editor') as HTMLSelectElement;
		expect(Array.from(owner.options).map((option) => option.text)).toContain('Morgan Owner');
		expect(Array.from(editor.options).map((option) => option.text)).toContain('Alex Editor');
	});

	it('opens the content details editor from the card pencil button', async () => {
		const onEdit = vi.fn();
		render(ContentPipeline, {
			boards: buildContentBoards([contentItem]),
			onNew: vi.fn(),
			onEdit,
			onPost: vi.fn(),
			onRemove: vi.fn(),
			onMove: vi.fn()
		});

		await fireEvent.mouseUp(screen.getByRole('button', { name: 'Edit Workspace launch reel' }));

		expect(onEdit).toHaveBeenCalledWith(contentItem);
	});

	it('opens a new content form from pipeline create controls before drag selection can take over', async () => {
		const onNew = vi.fn();
		render(ContentPipeline, {
			boards: buildContentBoards([contentItem]),
			onNew,
			onEdit: vi.fn(),
			onPost: vi.fn(),
			onRemove: vi.fn(),
			onMove: vi.fn()
		});

		await fireEvent.pointerDown(screen.getByRole('button', { name: 'New content' }));

		expect(onNew).toHaveBeenCalledWith('idea', 'Northstar');
	});

	it('filters a workstream by content theme', async () => {
		const standardItem: ContentItem = {
			...contentItem,
			id: 'content-2',
			title: 'Responsibility before freedom',
			theme: 'The Standard'
		};

		render(ContentPipeline, {
			boards: buildContentBoards([contentItem, standardItem]),
			onNew: vi.fn(),
			onEdit: vi.fn(),
			onPost: vi.fn(),
			onRemove: vi.fn(),
			onMove: vi.fn()
		});

		expect(screen.getByText('Workspace launch reel')).toBeInTheDocument();
		expect(screen.getByText('Responsibility before freedom')).toBeInTheDocument();

		await fireEvent.click(screen.getAllByRole('tab', { name: /The Standard/ })[0]);

		expect(screen.queryByText('Workspace launch reel')).not.toBeInTheDocument();
		expect(screen.getByText('Responsibility before freedom')).toBeInTheDocument();
	});
});
