// The slab's header pill shapes as class strings (FounderOS v1 slab.tsx PILL,
// PILL_ACCENT, chipClass), for links and buttons alike. Colors live in
// kit.css (.bn-pill, .bn-pill-accent, .bn-filter-chip).

/** A header action in the slab's pill shape. */
export const PILL = 'bn-pressable bn-pill inline-flex items-center gap-1.5 rounded-full border px-4 py-2 text-[13px]';

/** The accent pill Brand Deals uses for "Open in Notion". */
export const PILL_ACCENT = 'bn-pill-accent inline-flex items-center gap-2 rounded-full border px-4 py-2 text-[13px]';

/** Filter chips under a list head: the active one is solid accent. */
export const chipClass = (active: boolean): string =>
	`bn-pressable bn-filter-chip rounded-full px-4 py-1.5 text-[12.5px] ${active ? 'is-on font-semibold' : 'border'}`;
