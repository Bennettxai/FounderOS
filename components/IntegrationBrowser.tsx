'use client';

import { useState, type ReactNode } from 'react';
import { chipClass } from '@/components/slab';
import { IntegrationCategory } from '@/components/IntegrationCategory';

/**
 * "Browse by category" in the Brand Deals look (2026-09-24): the list card's
 * filter pills over the existing collapsible categories. "All" is the old
 * accordion exactly (first category open); a category pill narrows the list
 * to that one, opened. The tile grids are rendered on the server (BrandLogo
 * must never enter the client bundle) and arrive here as ready nodes.
 */
export type BrowseCategory = { label: string; count: number; connected: number; grid: ReactNode };

export function IntegrationBrowser({ categories }: { categories: BrowseCategory[] }) {
  const [active, setActive] = useState<string>('all');
  const shown = active === 'all' ? categories : categories.filter((c) => c.label === active);

  return (
    <div>
      <div className="mb-4 flex flex-wrap gap-2">
        <button type="button" className={chipClass(active === 'all')} onClick={() => setActive('all')} aria-pressed={active === 'all'}>
          All · {categories.reduce((n, c) => n + c.count, 0)}
        </button>
        {categories.map((c) => (
          <button
            key={c.label}
            type="button"
            className={chipClass(active === c.label)}
            onClick={() => setActive(c.label)}
            aria-pressed={active === c.label}
            title={`${c.connected} of ${c.count} connected`}
          >
            {c.label} · {c.count}
          </button>
        ))}
      </div>
      <div className="flex flex-col gap-2.5">
        {shown.map((c, idx) => (
          <IntegrationCategory
            // remount on filter change so a picked category always opens
            key={`${active}:${c.label}`}
            label={c.label}
            count={c.count}
            defaultOpen={active !== 'all' || idx === 0}
          >
            {c.grid}
          </IntegrationCategory>
        ))}
      </div>
    </div>
  );
}
