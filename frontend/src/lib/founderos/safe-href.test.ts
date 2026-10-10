import { describe, expect, it } from 'vitest';
import { safeHref } from './safe-href';

describe('safeHref', () => {
	it('passes http and https links through', () => {
		expect(safeHref('https://example.com/x?y=1')).toBe('https://example.com/x?y=1');
		expect(safeHref('http://example.com')).toBe('http://example.com');
	});

	it('drops any other scheme, whatever its case or padding', () => {
		for (const bad of ['javascript:alert(1)', ' JavaScript:alert(1)', 'data:text/html,<b>x</b>', 'vbscript:x', 'file:///etc/passwd'])
			expect(safeHref(bad), bad).toBeNull();
	});

	it('drops relative, empty and malformed values', () => {
		for (const bad of ['', null, undefined, '/os/funnel', 'example.com', 'https://', '//evil.example']) expect(safeHref(bad), String(bad)).toBeNull();
	});
});
