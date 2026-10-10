import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { fireEvent, render, screen } from '@testing-library/svelte';
import { beforeEach, describe, expect, it } from 'vitest';
import { COHORT_CTA, COHORT_SEEN, COHORT_STORAGE_KEY, COHORT_URL, shouldShowCohortModal } from './cohort';
import CohortBanner from './chrome/CohortBanner.svelte';
import CohortModal from './chrome/CohortModal.svelte';

// The demo's growth loop, ported from FounderOS v1 (lib/cohort.ts,
// tests/cohort.test.ts): a one-time welcome pop-up on Home the first time
// someone runs FounderOS, plus a permanent footer CTA under every view.

describe('cohort invite constants', () => {
	it('points at founderos.sh over https', () => {
		expect(COHORT_URL).toBe('https://founderos.sh');
	});
	it('carries the exact footer CTA line', () => {
		expect(COHORT_CTA).toBe(
			'Want help setting this up? Go to the FounderOS cohort to learn how to build the entire thing end-to-end and get it into production.'
		);
	});
	it('namespaces its storage key', () => {
		expect(COHORT_STORAGE_KEY).toMatch(/^founderos-/);
	});
});

describe('shouldShowCohortModal', () => {
	it('fires on Home (/os) for a fresh install', () => {
		expect(shouldShowCohortModal({ pathname: '/os', stored: null })).toBe(true);
		expect(shouldShowCohortModal({ pathname: '/os/', stored: null })).toBe(true);
	});
	it('stays down once dismissed', () => {
		expect(shouldShowCohortModal({ pathname: '/os', stored: COHORT_SEEN })).toBe(false);
	});
	it('never interrupts a deeper view', () => {
		expect(shouldShowCohortModal({ pathname: '/os/agents', stored: null })).toBe(false);
		expect(shouldShowCohortModal({ pathname: '/os/brain', stored: null })).toBe(false);
	});
});

describe('CohortBanner', () => {
	it('is a footer link to founderos.sh carrying the CTA', () => {
		const { container } = render(CohortBanner);
		const footer = container.querySelector('footer')!;
		expect(footer.textContent).toContain(COHORT_CTA);
		const a = footer.querySelector('a')!;
		expect(a.getAttribute('href')).toBe(COHORT_URL);
		expect(a.getAttribute('rel')).toBe('noreferrer');
		expect(a.textContent).toContain('founderos.sh');
	});
});

describe('CohortModal', () => {
	beforeEach(() => localStorage.clear());

	it('greets a fresh visitor on Home and remembers the dismissal', async () => {
		render(CohortModal, { pathname: '/os' });
		expect(screen.getByRole('dialog')).toBeTruthy();
		expect(screen.getByText('Build this for real')).toBeTruthy();
		await fireEvent.click(screen.getByRole('button', { name: 'Keep exploring' }));
		expect(screen.queryByRole('dialog')).toBeNull();
		expect(localStorage.getItem(COHORT_STORAGE_KEY)).toBe(COHORT_SEEN);
	});

	it('closes on Escape', async () => {
		render(CohortModal, { pathname: '/os' });
		await fireEvent.keyDown(window, { key: 'Escape' });
		expect(screen.queryByRole('dialog')).toBeNull();
	});

	it('stays hidden once seen, and off Home', () => {
		localStorage.setItem(COHORT_STORAGE_KEY, COHORT_SEEN);
		render(CohortModal, { pathname: '/os' });
		expect(screen.queryByRole('dialog')).toBeNull();
		localStorage.clear();
		render(CohortModal, { pathname: '/os/comms' });
		expect(screen.queryByRole('dialog')).toBeNull();
	});
});

describe('wiring', () => {
	it('the /os layout renders the banner after the page and mounts the modal', () => {
		const layout = readFileSync(resolve(__dirname, '../../routes/(founderos)/os/+layout.svelte'), 'utf8');
		expect(layout).toContain('<CohortModal');
		expect(layout.indexOf('{@render children()}')).toBeLessThan(layout.indexOf('<CohortBanner'));
	});
});
