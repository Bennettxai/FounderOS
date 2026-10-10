import { describe, expect, it } from 'vitest';
import { returnPath } from './returnTo';

describe('returnPath (where /login sends you after signing in)', () => {
	it('honours ?next= (the session gate) and ?redirect= (invites)', () => {
		expect(returnPath(new URLSearchParams('next=%2Fos%2Fclients'))).toBe('/os/clients');
		expect(returnPath(new URLSearchParams('redirect=/invite/abc'))).toBe('/invite/abc');
		expect(returnPath(new URLSearchParams('redirect=/a&next=/b'))).toBe('/a');
	});
	it('only same-site paths', () => {
		for (const bad of ['next=https://evil.example', 'next=//evil.example', 'next=javascript:alert(1)', 'next=', '']) expect(returnPath(new URLSearchParams(bad)), bad).toBeNull();
	});
});
