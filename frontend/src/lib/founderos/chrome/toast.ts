/**
 * The OS-wide toast stack, ported from FounderOS v1 components/Toaster.tsx (the
 * store half; Toaster.svelte draws it, mounted once by the /os layout).
 *
 * Toasts are for results that live somewhere else (archived, delegated,
 * created, failed). Actions whose result is visible in place confirm on the
 * button instead. Rules: one line, verb first, name the object. ok/warn/busy
 * auto-dismiss at 2.6s; err stays; hover pauses; destructive actions pass
 * `undo`. Stack bottom-right, newest on top, max 4.
 *
 *   const id = toast.busy('Asking the Conductor');
 *   toast.update(id, 'ok', 'Sent to the Conductor');
 */
import { writable } from 'svelte/store';

export type ToastKind = 'ok' | 'warn' | 'err' | 'busy';
export type Toast = { id: number; kind: ToastKind; text: string; undo?: () => void; born: number; held: boolean; ttl: number };

export const TOAST_TTL = 2600;
const MAX = 4;

export const toasts = writable<Toast[]>([]);

let n = 0;
const ttlFor = (kind: ToastKind) => (kind === 'err' ? Infinity : TOAST_TTL);

function push(kind: ToastKind, text: string, undo?: () => void): number {
	const id = ++n;
	toasts.update((l) => [...l.slice(-(MAX - 1)), { id, kind, text, undo, born: Date.now(), held: false, ttl: ttlFor(kind) }]);
	return id;
}

export const toast = {
	ok: (text: string, undo?: () => void) => push('ok', text, undo),
	warn: (text: string) => push('warn', text),
	err: (text: string) => push('err', text),
	busy: (text: string) => push('busy', text),
	update: (id: number, kind: ToastKind, text: string) =>
		toasts.update((l) => l.map((t) => (t.id === id ? { ...t, kind, text, born: Date.now(), ttl: ttlFor(kind) } : t))),
	close: (id: number) => toasts.update((l) => l.filter((t) => t.id !== id))
};

/** Hover holds a toast; letting go restarts its clock. */
export function hold(id: number, held: boolean): void {
	toasts.update((l) => l.map((t) => (t.id === id ? { ...t, held, born: held ? t.born : Date.now() } : t)));
}

/** Drop what has run out. The empty-list guard keeps an idle app from churning. */
export function sweep(now = Date.now()): void {
	toasts.update((l) => (l.length ? l.filter((t) => t.held || t.ttl === Infinity || now - t.born < t.ttl) : l));
}
