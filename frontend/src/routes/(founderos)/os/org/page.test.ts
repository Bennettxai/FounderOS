import { render, waitFor } from '@testing-library/svelte';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const url = { current: new URL('http://localhost/os/org?venture=vantage') };
vi.mock('$app/state', () => ({
	page: {
		get url() {
			return url.current;
		}
	}
}));
vi.mock('$lib/founderos/api', () => ({ founderosFetch: vi.fn(), FounderosApiError: class extends Error {} }));

import { founderosFetch } from '$lib/founderos/api';
import Page from './+page.svelte';

const fetchMock = vi.mocked(founderosFetch);

const emptyView = {
	departments: [],
	agents: [],
	agentNames: {},
	conductor: null,
	tree: { totalAgents: 0, activeAgents: 0 },
	crews: [],
	live: { connected: false, error: 'board unreachable', conductor: null, byDepartment: {}, extras: [] },
	ventures: [{ id: 'vantage', label: 'Vantage', kind: 'AI agency', color: '#00ffaa', detail: 'd', brainTag: 'vantage', focus: ['Ship'], areaAgents: {}, agentIds: [] }],
	lifeAreas: [],
	lastBroadcast: null
};

beforeEach(() => fetchMock.mockReset());

describe('/os/org page', () => {
	it('loads /pages/org and applies the ?venture lens', async () => {
		fetchMock.mockResolvedValueOnce(emptyView);
		const { getByText, queryByText } = render(Page);
		expect(getByText('Agent Hierarchy')).toBeTruthy();
		await waitFor(() => expect(getByText('Vantage — executive focus')).toBeTruthy());
		expect(fetchMock).toHaveBeenCalledWith('/pages/org');
		// prod's header carries no agent counter
		expect(queryByText(/agents · \d+ active/)).toBeNull();
	});

	it('says so when the view cannot load, instead of drawing an empty org', async () => {
		fetchMock.mockRejectedValueOnce(new Error('founderos workspace missing: run founderos-bootstrap (HTTP 503)'));
		const { getByText } = render(Page);
		await waitFor(() => expect(getByText(/run founderos-bootstrap/)).toBeTruthy());
	});
});
