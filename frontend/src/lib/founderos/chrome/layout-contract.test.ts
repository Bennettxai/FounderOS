// The /os shell's half of the Conductor dock (FounderOS v1 tests/conductor-mock-1h
// "the content column glides aside instead of being covered"): the layout mounts
// the dock and reads its published width as the content column's right margin.
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';
import { CONDUCTOR_W_VAR } from './conductor';

const layout = readFileSync(resolve(__dirname, '../../../routes/(founderos)/os/+layout.svelte'), 'utf8');

describe('/os layout: the Conductor dock pushes the page', () => {
	it('mounts the dock once, fed the current path', () => {
		expect(layout).toContain("import ConductorPanel from '$lib/founderos/chrome/ConductorPanel.svelte'");
		expect(layout.match(/<ConductorPanel\b/g)).toHaveLength(1);
		expect(layout).toMatch(/<ConductorPanel pathname=\{page\.url\.pathname\}/);
	});

	it('the content column takes the dock width as its right margin and glides 420ms', () => {
		expect(CONDUCTOR_W_VAR).toBe('--bn-conductor-w');
		expect(layout).toContain(`margin-right: var(${CONDUCTOR_W_VAR}, 0px)`);
		expect(layout).toMatch(/transition:\s*margin-right 420ms/);
	});

	it('glides on the house ease, as FounderOS v1 .os-shell does', () => {
		expect(layout).toMatch(/transition:\s*margin-right 420ms var\(--bn-ease\)/);
	});

	it('a drag tracks 1:1: the glide is off while the edge is held', () => {
		expect(layout).toMatch(/html\.bn-conductor-dragging\)[^{]*\{\s*transition: none/);
	});
});

describe('/os layout: the interaction layer and the page frame (FounderOS v1 app/layout.tsx)', () => {
	it('installs the hover lens on mount and removes it on leave, so BusinessOS never carries it', () => {
		expect(layout).toMatch(/import \{[^}]*installLens[^}]*\} from '\$lib\/founderos\/kit\/lens'/);
		expect(layout).toMatch(/const stopLens = installLens\(document\)/);
		expect(layout).toMatch(/stopLens\(\)/);
	});
	it('mounts the page-wide spotlight once', () => {
		expect(layout.match(/<PageSpotlight\b/g)).toHaveLength(1);
	});
	it('main pads like FounderOS v1: px-6 pt-5 pb-12, 16px gutters on a phone', () => {
		expect(layout).toMatch(/<main class="[^"]*\bpx-6\b[^"]*\bpb-12\b[^"]*\bpt-5\b[^"]*max-\[600px\]:px-4/);
	});
});

// FounderOS v1 app/template.tsx + app/loading.tsx + app/layout.tsx metadata.
describe('/os layout: transitions, loading, toasts, title (FounderOS v1 app shell)', () => {
	it('every view remounts on navigation into the slide-up .bn-view entrance (template.tsx)', () => {
		expect(layout).toMatch(/\{#key page\.url\.pathname\}\s*<div class="bn-view">/);
	});

	it('the page sits in a fluid page frame (layout.tsx data-part="page-frame")', () => {
		expect(layout).toMatch(/<div data-part="page-frame" class="w-full min-w-0">/);
	});

	it('a click answers at once: the page skeleton stands in while another view loads (loading.tsx)', () => {
		expect(layout).toContain("import PageSkeleton from '$lib/founderos/chrome/PageSkeleton.svelte'");
		expect(layout).toMatch(/import \{ navigating, page \} from '\$app\/state'/);
		expect(layout).toMatch(/\{:else if loadingOther\}\s*<PageSkeleton \/>/);
		expect(layout).toMatch(/\{#if setup\}\s*<SetupNotice /);
		expect(layout).toMatch(/const loadingOther = \$derived\(!!navigating\.to && navigating\.to\.url\.pathname !== page\.url\.pathname\)/);
	});

	it('mounts the toast stack once', () => {
		expect(layout).toContain("import Toaster from '$lib/founderos/chrome/Toaster.svelte'");
		expect(layout.match(/<Toaster\b/g)).toHaveLength(1);
	});

	it('the tab reads FOUNDER OS with the OS emblem as its icon', () => {
		expect(layout).toMatch(/<title>FOUNDER OS<\/title>/);
		expect(layout).toMatch(/<link rel="icon" type="image\/png" sizes="128x128" href=\{osIcon\}/);
		expect(layout).toContain("import osIcon from '$lib/founderos/chrome/os-icon.png'");
	});
});

describe('/os 404 (Next not-found inside the chrome)', () => {
	const read = (p: string) => readFileSync(resolve(__dirname, '../../../routes/(founderos)/os', p), 'utf8');
	it('an unknown /os path throws a 404 inside the /os layout', () => {
		expect(read('[...rest]/+page.ts')).toMatch(/error\(404, 'This page could not be found\.'\)/);
	});
	it('reads "404 | This page could not be found." with the hairline divider, and titles the tab', () => {
		const err = read('+error.svelte');
		expect(err).toMatch(/<title>\{page\.status\}: \{message\}<\/title>/);
		expect(err).toMatch(/<h1 class="[^"]*border-r[^"]*text-\[24px\] font-medium[^"]*">\{page\.status\}<\/h1>/);
		expect(err).toMatch(/<h2 class="[^"]*text-\[14px\][^"]*">\{message\}<\/h2>/);
	});
});
