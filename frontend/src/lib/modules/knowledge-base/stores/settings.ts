import { writable, derived } from 'svelte/store';

/**
 * KB Settings Store — persisted to localStorage
 * Controls: font size, page width, auto-save delay, trash retention
 */

export interface KBSettings {
	fontSize: 'small' | 'default' | 'large';
	pageWidth: 'narrow' | 'default' | 'wide' | 'full';
	autoSaveDelay: number;
	trashRetention: number;
}

const STORAGE_KEY = 'bos-kb-settings';

const defaults: KBSettings = {
	fontSize: 'default',
	pageWidth: 'default',
	autoSaveDelay: 1000,
	trashRetention: 30
};

function loadSettings(): KBSettings {
	try {
		const raw = localStorage.getItem(STORAGE_KEY);
		if (raw) {
			const parsed = JSON.parse(raw);
			return { ...defaults, ...parsed };
		}
	} catch {
		// ignore corrupted data
	}
	return { ...defaults };
}

function createSettingsStore() {
	const { subscribe, set, update } = writable<KBSettings>(loadSettings());

	// Persist on every change
	subscribe((value) => {
		try {
			localStorage.setItem(STORAGE_KEY, JSON.stringify(value));
		} catch {
			// ignore quota errors
		}
	});

	return {
		subscribe,
		set,
		update,
		setFontSize(size: KBSettings['fontSize']) {
			update((s) => ({ ...s, fontSize: size }));
		},
		setPageWidth(width: KBSettings['pageWidth']) {
			update((s) => ({ ...s, pageWidth: width }));
		},
		setAutoSaveDelay(delay: number) {
			update((s) => ({ ...s, autoSaveDelay: Math.max(500, Math.min(10000, delay)) }));
		},
		setTrashRetention(days: number) {
			update((s) => ({ ...s, trashRetention: Math.max(0, Math.min(365, days)) }));
		},
		reset() {
			set({ ...defaults });
		}
	};
}

export const kbSettings = createSettingsStore();

// Derived CSS values for consumption in components
export const fontSizePx = derived(kbSettings, ($s) => {
	const map = { small: '14px', default: '16px', large: '18px' } as const;
	return map[$s.fontSize];
});

export const pageMaxWidth = derived(kbSettings, ($s) => {
	const map = { narrow: '680px', default: '900px', wide: '1200px', full: '100%' } as const;
	return map[$s.pageWidth];
});
