import { render, waitFor } from '@testing-library/svelte';
import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('$lib/founderos/api', () => ({ founderosFetch: vi.fn(), FounderosApiError: class extends Error {} }));

import { founderosFetch } from '$lib/founderos/api';
import Page from './+page.svelte';

const fetchMock = vi.mocked(founderosFetch);

const empty = {
	workflows: [],
	jobs: [],
	agents: [],
	agentPresence: {},
	runsByOwner: {},
	toolIds: [],
	volume: {
		headline: 0, counts: { healthy: 0, overdue: 0, failing: 0, paused: 0, enabled: 0 }, chips: [], caption: '0 enabled · 0 runs recorded',
		meters: [{ label: 'Crons healthy (0/0)', frac: 0, display: 'none enabled', hue: 'var(--bn-warn)' }], foot: '0 workflows · 0 steps · 0 tools',
		series: [], runsInWindow: 0, failedInWindow: 0, rhythm: ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'].map((label) => ({ label, count: 0 })),
		load: { manualHours: 0, agentHours: 0, perWorkflow: [] },
		insight: { value: 0, headline: 'Every enabled task ran on its slot.', body: 'No scheduled tasks yet.', frac: 0 }
	}
};

beforeEach(() => fetchMock.mockReset());

describe('/os/workflows page', () => {
	it('loads the view once and renders honest empty states (never calls the drafting CLI)', async () => {
		fetchMock.mockResolvedValueOnce(empty);
		const { getByText } = render(Page);
		expect(getByText('loading workflows…')).toBeTruthy();
		await waitFor(() => expect(getByText('No scheduled tasks yet.')).toBeTruthy());
		expect(getByText(/No workflows mapped yet/)).toBeTruthy();
		expect(fetchMock).toHaveBeenCalledTimes(1);
		expect(fetchMock).toHaveBeenCalledWith('/pages/workflows/view');
	});

	it('says so when the backend cannot answer', async () => {
		fetchMock.mockRejectedValueOnce(new Error('founderos workspace missing: run founderos-bootstrap (HTTP 503)'));
		const { getByText } = render(Page);
		await waitFor(() => expect(getByText(/Could not load workflows: founderos workspace missing/)).toBeTruthy());
	});
});
