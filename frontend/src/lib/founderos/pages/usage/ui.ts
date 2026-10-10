/** Local class strings for /usage on --bn-* tokens. The pills and chips are
    the kit's (FounderOS v1 slab.tsx PILL / chipClass). */
export { PILL, chipClass } from '$lib/founderos/kit';

/** The lane / request bar track. */
export const TRACK = 'background: color-mix(in oklab, var(--bn-text) 8%, transparent)';

export const DETAIL = 'bn-border grid gap-6 rounded-[12px] border bg-[var(--bn-bg)] p-5 lg:grid-cols-[1.4fr_1fr]';

/** A plan card (UsageBoard.tsx PlanCard / OllamaCard): 12px, a pressable lens row. */
export const planCardClass = (selected: boolean) =>
	`bn-pressable is-row w-full rounded-[12px] border bg-[var(--bn-bg)] p-5 text-left ${selected ? 'border-[color:var(--bn-accent)]' : 'bn-border'}`;

/** The card a plan nobody reported gets. */
export const EMPTY_CARD = 'bn-dim bn-border rounded-[12px] border border-dashed p-5 text-[12.5px]';

export const POLL_MS = 10_000;
