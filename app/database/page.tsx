import { PageHeader } from '@/components/PageHeader';
import { SetupLink } from '@/components/SetupLink';
import { DatabaseExplorer } from '@/components/database/DatabaseExplorer';
import { scanBrain } from '@/lib/database/scan';
import { DatabaseG, DatabaseHeaderStats } from '@/components/database/DatabaseG';
import { GCard, GEmpty, GPage, GSectionHeader } from '@/components/g';
import { getAppearance } from '@/lib/appearance-server';
import { treeFor } from '@/lib/appearance';

export const dynamic = 'force-dynamic';

/**
 * Brain Files (route /database) shows the brain folder as an explorable block map: every file and
 * folder sized by space used, zoomable down to a single file's contents.
 * Read straight from disk on every visit (cached 60 s).
 */
export default function DatabasePage() {
  const db = scanBrain();
  // Two trees, one scan (glados-ui SPEC X5, S05 F05.20 to F05.24).
  if (treeFor(getAppearance(), '/database') === 'glados') {
    return (
      <GPage className="kg-page">
        <GSectionHeader
          size="page"
          title="Brain files"
          caption={db.configured ? <DatabaseHeaderStats total={db.tree.size} files={db.tree.files} root={db.root} /> : 'The brain folder, block by block'}
        />
        {db.configured ? (
          <DatabaseG initial={{ root: db.root, tree: db.tree, scannedAt: db.scannedAt, truncated: db.truncated }} />
        ) : (
          <GCard p={24}>
            <GEmpty
              kind="unconfigured"
              title="No brain folder found on this machine"
              body={`Looked in ${db.tried.join(' and ')}. Set BRAIN_DIR in .env.local to point at it.`}
              setupHref="/setup"
              setupHint="Set BRAIN_DIR to the brain folder's path"
            />
          </GCard>
        )}
      </GPage>
    );
  }
  return (
    <div>
      <PageHeader title="Brain files" eyebrow="The brain folder, block by block" />
      {db.configured ? (
        <DatabaseExplorer initial={{ root: db.root, tree: db.tree, scannedAt: db.scannedAt, truncated: db.truncated }} />
      ) : (
        <div className="rounded-lg-t border border-[var(--hairline)] bg-os-surface p-5">
          <div className="flex items-center gap-3">
            <span className="rounded-full border border-[var(--hairline)] px-2 py-[2px] text-[11px] text-os-muted">Not configured</span>
            <span className="text-[13px] text-os-text">No brain folder found on this machine.</span>
            <SetupLink href="/setup" hint="Set BRAIN_DIR to the brain folder's path" />
          </div>
          <p className="mt-2 text-[12.5px] text-os-muted">Looked in {db.tried.join(' and ')}. Set BRAIN_DIR in .env.local to point at it.</p>
        </div>
      )}
    </div>
  );
}
