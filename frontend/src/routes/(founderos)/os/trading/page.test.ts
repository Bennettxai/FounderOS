import { render, waitFor } from '@testing-library/svelte';
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';
import { tradingFixture } from '$lib/founderos/pages/trading/fixture';

const fetchMock = vi.fn();
vi.mock('$lib/founderos/api', () => ({ founderosFetch: (...args: unknown[]) => fetchMock(...args) }));

import Page from './+page.svelte';

// braces: mockReset returns the mock, and a function returned from beforeEach is run as a cleanup hook
beforeEach(() => {
	fetchMock.mockReset();
});
afterEach(() => vi.useRealTimers());

describe('/os/trading', () => {
	test('reads the page endpoint and renders the board', async () => {
		fetchMock.mockResolvedValue(tradingFixture());
		const { getByText } = render(Page);
		expect(getByText(/loading the trading board/i)).toBeTruthy();
		await waitFor(() => expect(getByText('Agent · the sleeve')).toBeTruthy());
		expect(fetchMock).toHaveBeenCalledWith('/pages/trading');
	});

	test('an unreachable store is an error, never an empty board', async () => {
		fetchMock.mockImplementation(async () => {
			throw new Error('trading store unreadable: connection refused (HTTP 503)');
		});
		const { getByText, queryByText } = render(Page);
		await waitFor(() => expect(getByText(/connection refused/)).toBeTruthy());
		expect(queryByText('No account data yet')).toBeNull();
	});

	test('re-reads every 60s, and a failed refresh keeps the last good board and says so', async () => {
		vi.useFakeTimers();
		fetchMock.mockResolvedValueOnce(tradingFixture());
		const { getByText } = render(Page);
		await vi.waitFor(() => expect(getByText('Agent · the sleeve')).toBeTruthy());
		fetchMock.mockRejectedValueOnce(new Error('HTTP 503'));
		await vi.advanceTimersByTimeAsync(60_000);
		expect(fetchMock).toHaveBeenCalledTimes(2);
		await vi.waitFor(() => expect(getByText(/refresh failed/)).toBeTruthy());
		expect(getByText('Agent · the sleeve')).toBeTruthy();
		fetchMock.mockResolvedValueOnce(tradingFixture());
		await vi.advanceTimersByTimeAsync(60_000);
		expect(fetchMock).toHaveBeenCalledTimes(3);
	});
});
