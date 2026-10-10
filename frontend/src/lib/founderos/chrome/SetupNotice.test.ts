import { render, screen } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';
import { FounderosApiError } from '$lib/founderos/api';
import { needsSetup } from './chrome';
import SetupNotice from './SetupNotice.svelte';

describe('needsSetup', () => {
	it('is the owner gate saying this account has no Founder OS yet', () => {
		expect(needsSetup(new FounderosApiError(403, 'x', { setup: true, error: 'run make demo' }))).toBe(true);
	});
	it('ignores every other failure', () => {
		expect(needsSetup(new FounderosApiError(403, 'x', { error: 'forbidden' }))).toBe(false);
		expect(needsSetup(new FounderosApiError(502, 'x', { setup: true }))).toBe(false);
		expect(needsSetup(new Error('network'))).toBe(false);
	});
});

describe('SetupNotice', () => {
	it('names the one command that sets Founder OS up for this account', () => {
		render(SetupNotice, { email: 'sam@vantage.example' });
		expect(screen.getByRole('heading').textContent).toMatch(/set up founder os/i);
		expect(screen.getByText('make demo OWNER=sam@vantage.example')).toBeTruthy();
	});
});
