import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

// The desktop (/window) sits outside the (app) shell, which is what restores
// the saved workspace. Without restoring it here, $currentWorkspace stayed
// null on the desktop and the 3D OS showed every module in every workspace.
describe('/window restores the saved workspace', () => {
	it('loads it on mount when none is current', () => {
		const src = readFileSync(resolve(__dirname, '+page.svelte'), 'utf8');
		expect(src).toMatch(/import \{[^}]*\bloadSavedWorkspace\b[^}]*\} from '\$lib\/stores\/workspaces'/);
		expect(src).toMatch(/if \(!get\(currentWorkspace\)\) loadSavedWorkspace\(\)/);
	});
});
