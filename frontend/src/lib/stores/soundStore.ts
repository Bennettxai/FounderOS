// Sound Store - Audio management for desktop environment
import { writable, get } from 'svelte/store';
import { browser } from '$app/environment';

// Sound event types
export type SoundEvent =
	| 'windowOpen'
	| 'windowClose'
	| 'windowMinimize'
	| 'windowMaximize'
	| 'click'
	| 'error'
	| 'notification'
	| 'startup'
	| 'typing';

// Sound pack identifiers
export type SoundPackId = 'silent' | 'classic' | 'modern' | 'retro' | 'minimal' | 'bubbly' | 'mechanical' | 'nature' | 'scifi' | 'custom';

// Individual sound configuration
export interface SoundConfig {
	enabled: boolean;
	volume: number; // 0-1
	url?: string; // Base64 data URL or file path
}

// Sound pack definition
export interface SoundPack {
	id: SoundPackId;
	name: string;
	description: string;
	sounds: Partial<Record<SoundEvent, string>>; // event -> audio data URL
}

// Sound settings state
export interface SoundSettings {
	enabled: boolean;
	masterVolume: number; // 0-1
	currentPack: SoundPackId;
	perEventSettings: Partial<Record<SoundEvent, SoundConfig>>;
	customSounds: Partial<Record<SoundEvent, string>>; // Custom uploaded sounds
}

// Storage key
const STORAGE_KEY = 'businessos_sound_settings';

// Default sound settings
const defaultSettings: SoundSettings = {
	enabled: false, // Sounds off by default
	masterVolume: 0.5,
	currentPack: 'silent',
	perEventSettings: {},
	customSounds: {},
};

// Built-in sound packs (using Web Audio API generated tones)
// In a real implementation, these would be actual audio files
const generateTone = (frequency: number, duration: number, type: OscillatorType = 'sine'): string => {
	// This is a placeholder - actual implementation would generate audio
	// For now, we'll use this to indicate the sound should be generated
	return `tone:${frequency}:${duration}:${type}`;
};

// Built-in sound pack definitions
export const builtInPacks: SoundPack[] = [
	{
		id: 'silent',
		name: 'Silent',
		description: 'No sounds',
		sounds: {},
	},
	{
		id: 'classic',
		name: 'Classic',
		description: 'Subtle clicks and chimes',
		sounds: {
			windowOpen: generateTone(800, 100, 'sine'),
			windowClose: generateTone(400, 100, 'sine'),
			windowMinimize: generateTone(600, 80, 'sine'),
			windowMaximize: generateTone(1000, 80, 'sine'),
			click: generateTone(1200, 30, 'square'),
			error: generateTone(200, 300, 'sawtooth'),
			notification: generateTone(880, 150, 'sine'),
			startup: generateTone(440, 500, 'sine'),
		},
	},
	{
		id: 'modern',
		name: 'Modern',
		description: 'Soft whooshes and pops',
		sounds: {
			windowOpen: generateTone(600, 150, 'sine'),
			windowClose: generateTone(300, 150, 'sine'),
			windowMinimize: generateTone(500, 100, 'sine'),
			windowMaximize: generateTone(700, 100, 'sine'),
			click: generateTone(1000, 20, 'sine'),
			error: generateTone(150, 400, 'sine'),
			notification: generateTone(660, 200, 'sine'),
			startup: generateTone(523, 600, 'sine'),
		},
	},
	{
		id: 'retro',
		name: 'Retro',
		description: '8-bit style sounds',
		sounds: {
			windowOpen: generateTone(440, 100, 'square'),
			windowClose: generateTone(220, 100, 'square'),
			windowMinimize: generateTone(330, 80, 'square'),
			windowMaximize: generateTone(550, 80, 'square'),
			click: generateTone(880, 30, 'square'),
			error: generateTone(110, 300, 'square'),
			notification: generateTone(660, 150, 'square'),
			startup: generateTone(262, 400, 'square'),
		},
	},
	{
		id: 'minimal',
		name: 'Minimal',
		description: 'Barely-there subtle ticks',
		sounds: {
			windowOpen: generateTone(1200, 40, 'sine'),
			windowClose: generateTone(800, 40, 'sine'),
			windowMinimize: generateTone(1000, 30, 'sine'),
			windowMaximize: generateTone(1400, 30, 'sine'),
			click: generateTone(2000, 15, 'sine'),
			error: generateTone(300, 150, 'sine'),
			notification: generateTone(1500, 60, 'sine'),
			startup: generateTone(800, 200, 'sine'),
		},
	},
	{
		id: 'bubbly',
		name: 'Bubbly',
		description: 'Playful pop sounds',
		sounds: {
			windowOpen: generateTone(523, 80, 'sine'),
			windowClose: generateTone(392, 80, 'sine'),
			windowMinimize: generateTone(440, 60, 'sine'),
			windowMaximize: generateTone(587, 60, 'sine'),
			click: generateTone(784, 25, 'sine'),
			error: generateTone(196, 200, 'triangle'),
			notification: generateTone(698, 100, 'sine'),
			startup: generateTone(523, 300, 'sine'),
		},
	},
	{
		id: 'mechanical',
		name: 'Mechanical',
		description: 'Typewriter and machine sounds',
		sounds: {
			windowOpen: generateTone(200, 50, 'sawtooth'),
			windowClose: generateTone(150, 60, 'sawtooth'),
			windowMinimize: generateTone(180, 40, 'sawtooth'),
			windowMaximize: generateTone(220, 40, 'sawtooth'),
			click: generateTone(400, 20, 'sawtooth'),
			error: generateTone(80, 200, 'sawtooth'),
			notification: generateTone(300, 80, 'sawtooth'),
			startup: generateTone(100, 400, 'sawtooth'),
		},
	},
	{
		id: 'nature',
		name: 'Nature',
		description: 'Organic, soft tones',
		sounds: {
			windowOpen: generateTone(350, 200, 'sine'),
			windowClose: generateTone(250, 200, 'sine'),
			windowMinimize: generateTone(300, 150, 'sine'),
			windowMaximize: generateTone(400, 150, 'sine'),
			click: generateTone(600, 50, 'triangle'),
			error: generateTone(180, 350, 'sine'),
			notification: generateTone(500, 180, 'triangle'),
			startup: generateTone(280, 600, 'sine'),
		},
	},
	{
		id: 'scifi',
		name: 'Sci-Fi',
		description: 'Futuristic electronic sounds',
		sounds: {
			windowOpen: generateTone(900, 120, 'sawtooth'),
			windowClose: generateTone(450, 120, 'sawtooth'),
			windowMinimize: generateTone(700, 80, 'sawtooth'),
			windowMaximize: generateTone(1100, 80, 'sawtooth'),
			click: generateTone(1800, 25, 'square'),
			error: generateTone(120, 400, 'sawtooth'),
			notification: generateTone(1200, 150, 'sawtooth'),
			startup: generateTone(600, 500, 'sawtooth'),
		},
	},
];

// Audio context for playing sounds
let audioContext: AudioContext | null = null;

function getAudioContext(): AudioContext | null {
	if (!browser) return null;

	if (!audioContext) {
		try {
			audioContext = new (window.AudioContext || (window as unknown as { webkitAudioContext: typeof AudioContext }).webkitAudioContext)();
		} catch (e) {
			console.warn('Web Audio API not supported:', e);
			return null;
		}
	}

	return audioContext;
}

// Play a generated tone
function playTone(frequency: number, duration: number, type: OscillatorType, volume: number) {
	const ctx = getAudioContext();
	if (!ctx) return;

	// Resume audio context if suspended (required for autoplay policy)
	if (ctx.state === 'suspended') {
		ctx.resume();
	}

	const oscillator = ctx.createOscillator();
	const gainNode = ctx.createGain();

	oscillator.connect(gainNode);
	gainNode.connect(ctx.destination);

	oscillator.type = type;
	oscillator.frequency.setValueAtTime(frequency, ctx.currentTime);

	// Apply volume with envelope for smoother sound
	gainNode.gain.setValueAtTime(0, ctx.currentTime);
	gainNode.gain.linearRampToValueAtTime(volume * 0.3, ctx.currentTime + 0.01);
	gainNode.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + duration / 1000);

	oscillator.start(ctx.currentTime);
	oscillator.stop(ctx.currentTime + duration / 1000);
}

// Play a base64 audio file
async function playBase64Audio(dataUrl: string, volume: number): Promise<void> {
	const ctx = getAudioContext();
	if (!ctx) return;

	if (ctx.state === 'suspended') {
		await ctx.resume();
	}

	try {
		// Convert base64 to array buffer
		const base64 = dataUrl.split(',')[1];
		const binary = atob(base64);
		const bytes = new Uint8Array(binary.length);
		for (let i = 0; i < binary.length; i++) {
			bytes[i] = binary.charCodeAt(i);
		}

		const audioBuffer = await ctx.decodeAudioData(bytes.buffer);

		const source = ctx.createBufferSource();
		const gainNode = ctx.createGain();

		source.buffer = audioBuffer;
		source.connect(gainNode);
		gainNode.connect(ctx.destination);
		gainNode.gain.value = volume;

		source.start(0);
	} catch (e) {
		console.warn('Failed to play audio:', e);
	}
}

// Load settings from localStorage
function loadSettings(): SoundSettings {
	if (!browser) return defaultSettings;

	try {
		const saved = localStorage.getItem(STORAGE_KEY);
		if (saved) {
			const parsed = JSON.parse(saved);
			return {
				...defaultSettings,
				...parsed,
			};
		}
	} catch (e) {
		console.error('Failed to load sound settings:', e);
	}

	return defaultSettings;
}

// Save settings to localStorage
function saveSettings(settings: SoundSettings) {
	if (!browser) return;

	try {
		localStorage.setItem(STORAGE_KEY, JSON.stringify(settings));
	} catch (e) {
		console.error('Failed to save sound settings:', e);
	}
}

// Create the sound store
function createSoundStore() {
	const { subscribe, set, update } = writable<SoundSettings>(loadSettings());

	return {
		subscribe,

		// Initialize (call on client side)
		initialize: () => {
			const settings = loadSettings();
			set(settings);
		},

		// Toggle sounds on/off
		setEnabled: (enabled: boolean) => {
			update(state => {
				const newState = { ...state, enabled };
				saveSettings(newState);
				return newState;
			});
		},

		// Set master volume
		setMasterVolume: (volume: number) => {
			update(state => {
				const newState = { ...state, masterVolume: Math.max(0, Math.min(1, volume)) };
				saveSettings(newState);
				return newState;
			});
		},

		// Set current sound pack
		setCurrentPack: (packId: SoundPackId) => {
			update(state => {
				const newState = { ...state, currentPack: packId };
				saveSettings(newState);
				return newState;
			});
		},

		// Set per-event settings
		setEventSettings: (event: SoundEvent, config: Partial<SoundConfig>) => {
			update(state => {
				const current = state.perEventSettings[event] || { enabled: true, volume: 1 };
				const newState = {
					...state,
					perEventSettings: {
						...state.perEventSettings,
						[event]: { ...current, ...config },
					},
				};
				saveSettings(newState);
				return newState;
			});
		},

		// Upload custom sound for an event
		setCustomSound: (event: SoundEvent, dataUrl: string | null) => {
			update(state => {
				const customSounds = { ...state.customSounds };
				if (dataUrl) {
					customSounds[event] = dataUrl;
				} else {
					delete customSounds[event];
				}
				const newState = { ...state, customSounds };
				saveSettings(newState);
				return newState;
			});
		},

		// Play a sound for an event
		playSound: (event: SoundEvent) => {
			const state = get({ subscribe });

			// Check if sounds are enabled
			if (!state.enabled) return;

			// Check per-event settings
			const eventConfig = state.perEventSettings[event];
			if (eventConfig && !eventConfig.enabled) return;

			// Calculate volume
			const eventVolume = eventConfig?.volume ?? 1;
			const finalVolume = state.masterVolume * eventVolume;

			// Get sound source
			let soundSource: string | undefined;

			// Check for custom sound first
			if (state.currentPack === 'custom' && state.customSounds[event]) {
				soundSource = state.customSounds[event];
			} else {
				// Get from current pack
				const pack = builtInPacks.find(p => p.id === state.currentPack);
				soundSource = pack?.sounds[event];
			}

			if (!soundSource) return;

			// Play the sound
			if (soundSource.startsWith('tone:')) {
				// Generated tone
				const [, freq, dur, type] = soundSource.split(':');
				playTone(
					parseFloat(freq),
					parseFloat(dur),
					type as OscillatorType,
					finalVolume
				);
			} else if (soundSource.startsWith('data:')) {
				// Base64 audio
				playBase64Audio(soundSource, finalVolume);
			}
		},

		// Preview a sound pack
		previewPack: (packId: SoundPackId, event: SoundEvent = 'notification') => {
			const state = get({ subscribe });
			const pack = builtInPacks.find(p => p.id === packId);
			const soundSource = pack?.sounds[event];

			if (!soundSource) return;

			if (soundSource.startsWith('tone:')) {
				const [, freq, dur, type] = soundSource.split(':');
				playTone(
					parseFloat(freq),
					parseFloat(dur),
					type as OscillatorType,
					state.masterVolume
				);
			}
		},

		// Preview a custom sound
		previewCustomSound: (dataUrl: string) => {
			const state = get({ subscribe });
			playBase64Audio(dataUrl, state.masterVolume);
		},

		// Get available sound packs
		getSoundPacks: () => builtInPacks,

		// Reset to defaults
		reset: () => {
			if (browser) {
				localStorage.removeItem(STORAGE_KEY);
			}
			set(defaultSettings);
		},
	};
}

export const soundStore = createSoundStore();

// Helper to convert audio file to base64
export async function audioFileToBase64(file: File): Promise<string> {
	return new Promise((resolve, reject) => {
		const reader = new FileReader();
		reader.onload = () => resolve(reader.result as string);
		reader.onerror = reject;
		reader.readAsDataURL(file);
	});
}

// Sound event labels for UI
export const soundEventLabels: Record<SoundEvent, string> = {
	windowOpen: 'Window Open',
	windowClose: 'Window Close',
	windowMinimize: 'Window Minimize',
	windowMaximize: 'Window Maximize',
	click: 'Click',
	error: 'Error',
	notification: 'Notification',
	startup: 'Startup',
	typing: 'Typing',
};
