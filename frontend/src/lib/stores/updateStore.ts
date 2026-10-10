import { browser } from '$app/environment';
import { getElectronAPI, isElectron } from '$lib/utils/platform';
import { writable } from 'svelte/store';

export type UpdateState =
	| 'idle'
	| 'checking'
	| 'available'
	| 'downloading'
	| 'downloaded'
	| 'none'
	| 'unsupported'
	| 'error';

export interface UpdateProgress {
	percent: number;
	transferred?: number;
	total?: number;
	bytesPerSecond?: number;
}

export interface UpdateModel {
	isDesktop: boolean;
	isPackaged: boolean;
	channel: string;
	minimumSupportedVersion: string | null;
	supported: boolean;
	state: UpdateState;
	currentVersion: string | null;
	latestVersion: string | null;
	releaseDate: string | null;
	releaseNotes: unknown;
	progress: UpdateProgress | null;
	error: string;
	message: string;
	checkedAt: string | null;
	dismissedVersion: string | null;
}

const initialState: UpdateModel = {
	isDesktop: false,
	isPackaged: false,
	channel: 'web',
	minimumSupportedVersion: null,
	supported: true,
	state: 'idle',
	currentVersion: null,
	latestVersion: null,
	releaseDate: null,
	releaseNotes: null,
	progress: null,
	error: '',
	message: '',
	checkedAt: null,
	dismissedVersion: browser ? localStorage.getItem('businessos_update_dismissed_version') : null,
};

const { subscribe, update, set } = writable<UpdateModel>(initialState);

let initialized = false;
let unsubscribers: Array<() => void> = [];

function normalizeNotes(notes: unknown): unknown {
	if (Array.isArray(notes)) {
		return notes
			.map((item) => {
				if (typeof item === 'string') return item;
				if (item && typeof item === 'object' && 'note' in item) return String((item as { note: unknown }).note);
				return '';
			})
			.filter(Boolean)
			.join('\n\n');
	}
	return notes;
}

function applyCheckResponse(res: any) {
	update((state) => {
		if (!res?.ok) {
			return {
				...state,
				state: 'error',
				error: res?.error || 'Update check failed.',
				message: '',
				checkedAt: new Date().toISOString(),
			};
		}

		if (res.available) {
			return {
				...state,
				state: 'available',
				latestVersion: res.version ?? null,
				currentVersion: res.currentVersion ?? state.currentVersion,
				releaseDate: res.releaseDate ?? null,
				releaseNotes: normalizeNotes(res.releaseNotes ?? null),
				error: '',
				message: '',
				checkedAt: new Date().toISOString(),
			};
		}

		return {
			...state,
			state: 'none',
			latestVersion: null,
			currentVersion: res.currentVersion ?? state.currentVersion,
			releaseDate: null,
			releaseNotes: null,
			progress: null,
			error: '',
			message: res.message || 'BusinessOS is up to date.',
			checkedAt: new Date().toISOString(),
		};
	});
}

function getPayload(payload: unknown): Record<string, any> {
	return payload && typeof payload === 'object' ? (payload as Record<string, any>) : {};
}

export const updateStore = {
	subscribe,

	async init() {
		if (!browser || initialized) return;
		initialized = true;

		const desktop = isElectron();
		const api = getElectronAPI();
		if (!desktop || !api) {
			update((state) => ({
				...state,
				isDesktop: false,
				isPackaged: false,
				channel: 'web',
				message: 'Updates are installed through the BusinessOS desktop app.',
			}));
			return;
		}

		try {
			const [version, platform, updateInfo] = await Promise.all([
				api.getVersion().catch(() => null),
				api.getPlatform?.().catch(() => null),
				api.updates?.getInfo?.().catch(() => null),
			]);
			update((state) => ({
				...state,
				isDesktop: true,
				isPackaged: Boolean(updateInfo?.isPackaged ?? platform?.isPackaged),
				channel: updateInfo?.channel || 'stable',
				minimumSupportedVersion: updateInfo?.minimumSupportedVersion ?? null,
				supported: updateInfo?.supported ?? true,
				state: updateInfo?.supported === false ? 'unsupported' : state.state,
				message:
					updateInfo?.supported === false
						? `This build is below the minimum supported version ${updateInfo.minimumSupportedVersion}.`
						: state.message,
				currentVersion: updateInfo?.currentVersion || version || state.currentVersion,
			}));
		} catch {
			update((state) => ({ ...state, isDesktop: true, channel: 'stable' }));
		}

		unsubscribers = [
			api.on('update:checking', () => {
				update((state) => ({ ...state, state: 'checking', error: '', message: '' }));
			}),
			api.on('update:available', (payload: unknown) => {
				const info = getPayload(payload);
				update((state) => ({
					...state,
					state: 'available',
					latestVersion: info.version ?? state.latestVersion,
					currentVersion: info.currentVersion ?? state.currentVersion,
					releaseDate: info.releaseDate ?? null,
					releaseNotes: normalizeNotes(info.releaseNotes ?? null),
					progress: null,
					error: '',
					message: '',
				}));
			}),
			api.on('update:not-available', () => {
				update((state) => ({
					...state,
					state: 'none',
					latestVersion: null,
					progress: null,
					error: '',
					message: 'BusinessOS is up to date.',
					checkedAt: new Date().toISOString(),
				}));
			}),
			api.on('update:download-progress', (payload: unknown) => {
				const info = getPayload(payload);
				update((state) => ({
					...state,
					state: 'downloading',
					progress: {
						percent: Number(info.percent ?? 0),
						transferred: Number(info.transferred ?? 0),
						total: Number(info.total ?? 0),
						bytesPerSecond: Number(info.bytesPerSecond ?? 0),
					},
				}));
			}),
			api.on('update:downloaded', (payload: unknown) => {
				const info = getPayload(payload);
				update((state) => ({
					...state,
					state: 'downloaded',
					latestVersion: info.version ?? state.latestVersion,
					progress: { percent: 100 },
					error: '',
					message: '',
				}));
			}),
			api.on('update:error', (message: unknown) => {
				update((state) => ({
					...state,
					state: 'error',
					error: typeof message === 'string' ? message : 'Update failed.',
				}));
			}),
		];
	},

	destroy() {
		for (const unsubscribe of unsubscribers) unsubscribe();
		unsubscribers = [];
		initialized = false;
		set(initialState);
	},

	async check() {
		const api = getElectronAPI();
		if (!api?.updates?.check) {
			update((state) => ({
				...state,
				state: 'error',
				error: 'Update checks are only available in the desktop app.',
			}));
			return;
		}
		update((state) => ({ ...state, state: 'checking', error: '', message: '' }));
		try {
			const res = await api.updates.check();
			applyCheckResponse(res);
		} catch (error) {
			update((state) => ({
				...state,
				state: 'error',
				error: error instanceof Error ? error.message : 'Update check failed.',
			}));
		}
	},

	async download() {
		const api = getElectronAPI();
		if (!api?.updates?.download) return;
		update((state) => ({ ...state, state: 'downloading', error: '', progress: { percent: 0 } }));
		try {
			const ok = await api.updates.download();
			if (!ok) {
				update((state) => ({ ...state, state: 'error', error: 'Update download failed.' }));
			}
		} catch (error) {
			update((state) => ({
				...state,
				state: 'error',
				error: error instanceof Error ? error.message : 'Update download failed.',
			}));
		}
	},

	async install() {
		const api = getElectronAPI();
		if (!api?.updates?.install) return;
		await api.updates.install();
	},

	dismiss(version: string | null) {
		if (browser && version) {
			localStorage.setItem('businessos_update_dismissed_version', version);
		}
		update((state) => ({ ...state, dismissedVersion: version }));
	},

	clearDismissed() {
		if (browser) localStorage.removeItem('businessos_update_dismissed_version');
		update((state) => ({ ...state, dismissedVersion: null }));
	},
};
