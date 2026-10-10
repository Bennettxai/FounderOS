/**
 * The hover-lens layer (FounderOS v1 lib/hooks/useLens.ts). One document-level
 * pointermove listener: every [data-lens] element gets --bn-lx/--bn-ly
 * (magnetic pull, px) and --bn-rx/--bn-ry (tilt, deg) while the pointer is
 * over it, the nearest [data-spot] ancestor gets --bn-sx/--bn-sy for its
 * .bn-spotlight radial, and :root gets --bn-px/--bn-py (viewport coords) for
 * the page-wide .bn-spotlight.is-page layer. Controls (data-lens="c") pull
 * 4px, rows ("r") 2px. Disabled entirely under prefers-reduced-motion.
 *
 * The /os layout installs it on mount and removes it on leave, so BusinessOS
 * pages never carry the listener.
 */

export const LENS_PULL = { c: 4, r: 2 } as const;

type Box = { left: number; top: number; width: number; height: number };

/** The four lens variables for a pointer at (x, y) over a box. */
export function lensVars(r: Box, x: number, y: number, kind: string): { lx: string; ly: string; ry: string; rx: string } {
	const dx = Math.max(-1, Math.min(1, ((x - r.left) / r.width - 0.5) * 2));
	const dy = Math.max(-1, Math.min(1, ((y - r.top) / r.height - 0.5) * 2));
	const k = kind === 'c' ? LENS_PULL.c : LENS_PULL.r;
	return {
		lx: (dx * k).toFixed(1) + 'px',
		ly: (dy * k).toFixed(1) + 'px',
		ry: (dx * k * 0.75).toFixed(1) + 'deg',
		rx: (-dy * k * 0.75).toFixed(1) + 'deg'
	};
}

const VARS = ['--bn-lx', '--bn-ly', '--bn-rx', '--bn-ry'];

/** Start the lens on a document; returns the cleanup. */
export function installLens(doc: Document = document): () => void {
	const win = doc.defaultView;
	if (!win || (typeof win.matchMedia === 'function' && win.matchMedia('(prefers-reduced-motion: reduce)').matches)) {
		return () => {};
	}
	let lensEl: HTMLElement | null = null;
	let spotEl: HTMLElement | null = null;
	const root = doc.documentElement;
	const reset = (el: HTMLElement) => VARS.forEach((v) => el.style.removeProperty(v));
	const dim = (el: HTMLElement) => {
		el.style.setProperty('--bn-sx', '-999px');
		el.style.setProperty('--bn-sy', '-999px');
	};
	const dimPage = () => {
		root.style.setProperty('--bn-px', '-999px');
		root.style.setProperty('--bn-py', '-999px');
	};

	const onMove = (e: Event) => {
		const { clientX, clientY } = e as MouseEvent;
		root.style.setProperty('--bn-px', clientX + 'px');
		root.style.setProperty('--bn-py', clientY + 'px');
		const t = e.target as Element | null;
		const el = (t?.closest?.('[data-lens]') as HTMLElement | null) ?? null;
		if (el !== lensEl) {
			if (lensEl) reset(lensEl);
			lensEl = el;
		}
		if (el) {
			const v = lensVars(el.getBoundingClientRect(), clientX, clientY, el.dataset.lens ?? 'r');
			el.style.setProperty('--bn-lx', v.lx);
			el.style.setProperty('--bn-ly', v.ly);
			el.style.setProperty('--bn-ry', v.ry);
			el.style.setProperty('--bn-rx', v.rx);
		}
		const card = (t?.closest?.('[data-spot]') as HTMLElement | null) ?? null;
		if (card !== spotEl) {
			if (spotEl) dim(spotEl);
			spotEl = card;
		}
		if (card) {
			const r = card.getBoundingClientRect();
			card.style.setProperty('--bn-sx', clientX - r.left + 'px');
			card.style.setProperty('--bn-sy', clientY - r.top + 'px');
		}
	};
	const onLeave = () => {
		dimPage();
		if (lensEl) {
			reset(lensEl);
			lensEl = null;
		}
		if (spotEl) {
			dim(spotEl);
			spotEl = null;
		}
	};

	doc.addEventListener('pointermove', onMove, { passive: true });
	doc.addEventListener('pointerleave', onLeave);
	return () => {
		doc.removeEventListener('pointermove', onMove);
		doc.removeEventListener('pointerleave', onLeave);
		if (lensEl) reset(lensEl);
		dimPage();
	};
}
