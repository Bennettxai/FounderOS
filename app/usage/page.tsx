import UsageBoard from '@/components/UsageBoard';
import { Slab, SlabTitle, PILL } from '@/components/slab';

export const dynamic = 'force-dynamic';

/** The token-burn board in the Brand Deals look (2026-09-24). The title row
    is static; every number below it is the live board's, polled every 10s. */
export default function UsagePage() {
  return (
    <Slab>
      <SlabTitle
        eyebrow="token burn · one plan per provider"
        title="Usage"
        meta="Claude · ChatGPT / Codex · Ollama · local file parsing only · no paid calls · refreshes every 10s"
        right={
          <a href="/api/usage" target="_blank" rel="noreferrer" className={PILL}>
            Raw reading
          </a>
        }
      />
      <UsageBoard />
    </Slab>
  );
}
