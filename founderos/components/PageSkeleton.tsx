/**
 * The instant-paint fallback every route shows on click, before its
 * `force-dynamic` server render (live DB/connector reads) finishes.
 *
 * This is the fix for "the button doesn't respond to my click": without a
 * `loading.tsx`, the App Router keeps the OLD page fully painted and does
 * nothing else until the new page's entire render completes, so a click
 * looks dead for however long the server takes. Once a `loading.tsx` exists,
 * Next swaps to it the instant navigation starts -- same render budget as
 * before, but the screen answers back immediately instead of freezing.
 *
 * It is also what makes prefetching possible at all on a fully dynamic page:
 * Next can only prefetch UP TO the nearest loading boundary, so a route with
 * none has nothing to preload. Every sidebar link sits in the viewport, so
 * Next prefetches this shell for all of them as soon as the shell exists --
 * the "preload while I'm using it" half of the ask, for free.
 *
 * No page-specific content on purpose, but the shape is the Brand Deals slab
 * every page now lands in (2026-09-24): title row, a 2fr/1fr hero with a
 * volume card's meter tracks, then a row of three. A generic shape avoids
 * duplicating each page's real layout just to guess at it.
 *
 * The cards are what they stand in for: glass faces (a SlabCard is one) at the
 * card radius, at rest. A face never fades (glass brief §4), so the pulse is on
 * the leaves inside them and on the bars that stand in for text.
 */
function Line({ className = '' }: { className?: string }) {
  return <div className={`animate-pulse rounded-full bg-os-border ${className}`} />;
}

/** A card in outline: the face, and a bar where its title goes. */
function Card({ className = '' }: { className?: string }) {
  return (
    <div className={`glass-face rounded-[var(--gl-r-card)] px-6 py-5 ${className}`}>
      <Line className="h-4 w-32" />
    </div>
  );
}

/** The volume card in outline: a big numeral, a caption, three meter tracks. */
function VolumeOutline() {
  return (
    <div className="glass-face flex h-72 flex-col rounded-[var(--gl-r-card)] px-6 py-5">
      <Line className="h-4 w-32" />
      <Line className="mt-5 h-10 w-40" />
      <div className="mt-6 flex flex-1 flex-col justify-around border-t border-os-border pt-4">
        {[0.7, 0.45, 0.85].map((w) => (
          <div key={w} data-skel-meter>
            <div className="mb-2 h-2.5 w-24 animate-pulse rounded-full bg-os-border" />
            <div className="h-[10px] w-full rounded-full" style={{ background: 'color-mix(in oklab, var(--text) 6%, transparent)' }}>
              <div className="h-full animate-pulse rounded-full bg-os-border" style={{ width: `${w * 100}%` }} />
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

export function PageSkeleton() {
  return (
    <div className="os-slab">
      <div className="mb-7 flex items-end justify-between gap-4">
        <div>
          <Line className="mb-3 h-2.5 w-32" />
          <Line className="h-11 w-64" />
          <Line className="mt-3 h-2.5 w-80 max-w-full" />
        </div>
        <Line className="h-9 w-32" />
      </div>
      <div className="grid grid-cols-[2fr_1fr] gap-6 max-[1200px]:grid-cols-1">
        <Card className="h-72" />
        <VolumeOutline />
      </div>
      <div className="mt-6 grid grid-cols-3 gap-6 max-[1200px]:grid-cols-1">
        <Card className="h-44" />
        <Card className="h-44" />
        <Card className="h-44" />
      </div>
      <Card className="mt-6 h-48" />
    </div>
  );
}
