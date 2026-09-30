import { allConnectorStatuses } from '@/lib/connectors';
import { readEnvLocal } from '@/lib/creds';
import { oauthReadiness } from '@/lib/oauth/store';
import { connectionCatalog, integrationsByCategory, type CatalogEntry } from '@/lib/integrations-catalog';
import { integrationsVolume } from '@/lib/integrations-volume';
import { Slab, SlabTitle, SlabCard, BigStat, Chip, MeterStack, InsightCard, PILL } from '@/components/slab';
import { DotMatrix } from '@/components/slab-charts';
import { ApiKeys } from '@/components/ApiKeys';
import { ConnectionCard } from '@/components/ConnectionCard';
import { IntegrationBrowser } from '@/components/IntegrationBrowser';

export const dynamic = 'force-dynamic';

const GRID = 'grid gap-3 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4';

/**
 * The Connections board. 2026-09-24: it wears the Brand Deals slab
 * (components/slab). Every number in the hero and second row comes from
 * lib/integrations-volume, fed with the same live connector checks and
 * catalog the tiles below render; the tiles, connect flows, category
 * accordion and key editor are unchanged.
 */
export default async function ConnectionsPage() {
  const statuses = await allConnectorStatuses();
  const env = readEnvLocal();
  const catalog = connectionCatalog(statuses, env);
  // Null for every tile whose provider has no usable authorization-code flow.
  const oauthFor = (slug: string) => oauthReadiness(slug, env);
  const detailByConnector = new Map(statuses.map((s) => [s.id, s.detail]));
  const guidanceFor = (entry: CatalogEntry) =>
    entry.connectorId ? detailByConnector.get(entry.connectorId) : undefined;

  const byId = new Map(catalog.map((c) => [c.slug, c]));
  const connected = catalog.filter((c) => c.connected);
  const popular = catalog.filter((c) => c.popular);
  const categories = [...integrationsByCategory().entries()];
  const v = integrationsVolume({ statuses, catalog });

  const card = (entry: CatalogEntry) => (
    <ConnectionCard key={entry.slug} entry={entry} guidance={guidanceFor(entry)} oauth={oauthFor(entry.slug)} />
  );

  return (
    <Slab>
      <SlabTitle
        eyebrow="connections"
        title="Connections"
        meta={`${catalog.length} tools · ${v.counts.total} connector checks · live status, never a stored key alone`}
        right={
          <>
            <Chip tone={v.counts.error ? 'err' : 'ok'}>
              {v.counts.connected} live{v.counts.error ? ` · ${v.counts.error} erroring` : ''}
            </Chip>
            <a href="#api-keys" className={PILL}>
              API keys
            </a>
          </>
        }
      />

      {/* Hero row, Brand Deals' shape: what is live + the volume card */}
      <div className="grid grid-cols-[2fr_1fr] gap-6 max-[1200px]:grid-cols-1">
        <SlabCard i={1} title="Your connected tools" sub={`${connected.length} of ${catalog.length}`}>
          <div className="px-6 pb-6 pt-3">
            <BigStat size={30} value={connected.length} caption="catalog tools whose connector answered on this load" />
            <div className="mt-5 border-t border-os-border pt-5">
              {connected.length > 0 ? (
                <div className={GRID}>{connected.map(card)}</div>
              ) : (
                <div className="text-[12.5px] text-os-dim">Nothing is connected yet. Pick a tool below and connect it.</div>
              )}
            </div>
          </div>
        </SlabCard>

        <SlabCard i={2} title="Connection Volume" className="flex flex-col">
          <div className="flex flex-1 flex-col px-6 pb-6 pt-3">
            <BigStat value={v.headline} chips={v.chips} caption={v.caption} />
            <MeterStack meters={v.meters} foot={v.foot} empty="no connector checks ran" />
          </div>
        </SlabCard>
      </div>

      {/* Second row: by category, connector health, THE gradient card */}
      <div className="mt-6 grid grid-cols-3 gap-6 max-[1200px]:grid-cols-1">
        <SlabCard i={3} title="By Category" sub="connected tools">
          <div className="flex flex-col gap-5 px-6 pb-6 pt-3">
            <BigStat
              size={30}
              display={v.topCategory ? v.topCategory.name : 'none'}
              caption={v.topCategory ? `most connected · ${v.topCategory.count} live` : 'no category has a live tool'}
            />
            <DotMatrix cols={v.byCategory} hue="var(--ramp-1)" />
          </div>
        </SlabCard>

        <SlabCard i={4} title="Connector Health" sub="this load">
          <div className="flex flex-col gap-5 px-6 pb-6 pt-3">
            <BigStat size={30} value={v.counts.total} caption="connector checks, live · unset · error" />
            <DotMatrix cols={v.health} hue="var(--ok)" />
          </div>
        </SlabCard>

        <InsightCard
          i={5}
          badge="Needs you"
          value={v.insight.value}
          headline={v.insight.headline}
          body={v.insight.body}
          frac={v.insight.frac}
        />
      </div>

      {/* Popular */}
      <SlabCard i={6} title="Popular" sub={`${popular.length}`} className="mt-6">
        <div className="px-6 pb-6 pt-4">
          <div className={GRID}>{popular.map(card)}</div>
        </div>
      </SlabCard>

      {/* Browse by category: filter pills over the collapsible categories */}
      <SlabCard i={7} title="Browse by category" sub={`${categories.length}`} className="mt-6">
        <div className="px-6 pb-6 pt-4">
          <IntegrationBrowser
            categories={categories.map(([category, tools]) => ({
              label: category,
              count: tools.length,
              connected: tools.filter((t) => byId.get(t.slug)?.connected).length,
              grid: <div className={GRID}>{tools.map((tool) => card(byId.get(tool.slug) as CatalogEntry))}</div>,
            }))}
          />
        </div>
      </SlabCard>

      <SlabCard i={8} className="mt-6">
        <div id="api-keys" className="scroll-mt-24 px-6 pb-6 pt-5">
          <ApiKeys />
        </div>
      </SlabCard>
    </Slab>
  );
}
