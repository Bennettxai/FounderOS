import { PageHeader } from '@/components/PageHeader';
import { SetupLink } from '@/components/SetupLink';
import { ComputeConsole } from '@/components/compute/ComputeConsole';
import { computeSnapshot } from '@/lib/compute/miosa';
import { ComputeG } from '@/components/compute/ComputeG';
import { GCard, GDotNumber, GEmpty, GPage, GSectionHeader } from '@/components/g';
import { getAppearance } from '@/lib/appearance-server';
import { treeFor } from '@/lib/appearance';

export const dynamic = 'force-dynamic';

/**
 * Compute — the cloud computers the agents work on (MIOSA), with a live view
 * of each desktop. Read from MIOSA on every visit; nothing is stored here.
 */
export default async function ComputePage() {
  const snap = await computeSnapshot();
  // Two trees, one fetch (glados-ui SPEC X5, S05 F05.60 to F05.63).
  if (treeFor(getAppearance(), '/compute') === 'glados') {
    return (
      <GPage className="kg-page">
        <GSectionHeader
          size="page"
          title="Compute"
          caption="Cloud computers your agents work on"
          action={
            snap.configured && snap.ok && snap.credits != null ? (
              <span className="kg-credits" title={`MIOSA credits on this account${snap.creditsExpireAt ? `, expiring ${snap.creditsExpireAt.slice(0, 10)}` : ''}`}>
                <GDotNumber value={snap.credits} size={3} label={`${snap.credits} MIOSA credits`} />
                <span className="kg-cap">MIOSA credits</span>
              </span>
            ) : undefined
          }
        />
        {!snap.configured ? (
          <GCard p={24}>
            <GEmpty
              kind="unconfigured"
              title="No MIOSA key on this machine"
              body="Cloud computers and their live desktops show here once the key is set."
              setupHref="/setup"
              setupHint="Set MIOSA_API_KEY (create one with: miosa auth token create)"
            />
          </GCard>
        ) : !snap.ok ? (
          <GCard p={24}>
            <GEmpty kind="error" title="MIOSA did not answer" body={snap.error} />
          </GCard>
        ) : snap.computers.length === 0 ? (
          <GCard p={24}>
            <GEmpty
              title="No computers yet"
              body={
                <>
                  Create one with <code data-mono-ok="">miosa computers create</code> and it appears here.
                </>
              }
            />
          </GCard>
        ) : (
          <ComputeG computers={snap.computers} />
        )}
      </GPage>
    );
  }
  const right = snap.configured && snap.ok && snap.credits != null
    ? <span className="rounded-full border border-[var(--hairline)] px-2.5 py-[3px] text-[11.5px] text-os-muted"><b className="text-os-text tabular-nums">{snap.credits.toLocaleString()}</b> MIOSA credits</span>
    : undefined;
  return (
    <div>
      <PageHeader title="Compute" eyebrow="Cloud computers your agents work on" right={right} />
      {!snap.configured ? (
        <div className="rounded-lg-t border border-[var(--hairline)] bg-os-surface p-5">
          <div className="flex items-center gap-3">
            <span className="rounded-full border border-[var(--hairline)] px-2 py-[2px] text-[11px] text-os-muted">Not configured</span>
            <span className="text-[13px] text-os-text">No MIOSA key on this machine.</span>
            <SetupLink href="/setup" hint="Set MIOSA_API_KEY (create one with: miosa auth token create)" />
          </div>
        </div>
      ) : !snap.ok ? (
        <div className="rounded-lg-t border border-[var(--hairline)] bg-os-surface p-5 text-[13px]">
          <span className="text-os-err">MIOSA did not answer: {snap.error}</span>
        </div>
      ) : snap.computers.length === 0 ? (
        <div className="rounded-lg-t border border-[var(--hairline)] bg-os-surface p-5 text-[13px] text-os-muted">
          No computers yet. Create one with <code className="text-os-text">miosa computers create</code> and it appears here.
        </div>
      ) : (
        <ComputeConsole computers={snap.computers} />
      )}
    </div>
  );
}
