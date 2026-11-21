'use client';

import { useEffect, useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import Image from 'next/image';
import { AudioLines, Film, Image as ImageIcon, Link2, Maximize2, Plus, Search, Shuffle, SlidersHorizontal } from 'lucide-react';
import { activeCampaign, contentCampaigns } from '@/lib/content-campaigns';
import { ELEMENT_MODELS } from '@/lib/higgsfield/catalog';
import { requestCompose } from '@/lib/content-compose';
import {
  assetUrl,
  type ContentToolKey,
  type MediaAsset,
  type MediaManifest,
  type ProductionTask,
  type TaskSpec,
} from '@/lib/content-schema';
import type { HiggsfieldAccount, IdentityAsset } from '@/lib/higgsfield/account';
import type { UploadAsset } from '@/lib/content-uploads';
import {
  DEFAULT_MEDIA_FILTER,
  filterMediaAssets,
  formatTimestamp,
  mediaFilterOptions,
  modelDisplayName,
  taskSpecOrFallback,
  TASK_TYPE_TOOL_KEY,
  upscaleSpec,
  type MediaFilterState,
} from '@/lib/content-feed';
import { VerbButton } from '@/components/content/ActionRow';
import { HoverVideo } from '@/components/content/MediaPreview';
import { MediaDetailOverlay } from '@/components/content/MediaDetail';
import { useMeltClose } from '@/lib/use-melt';
import { Layer } from '@/components/g/Overlays';

/** Filename inside public/content-media/uploads/ — mirrors lib/content-uploads.ts's
 * uploadUrl without importing that (server-only, node:fs) module into the client bundle. */
function uploadUrl(upload: UploadAsset): string {
  return `/content-media/uploads/${upload.file}`;
}

const CAMPAIGN_NAME_BY_ID = new Map(contentCampaigns().map((c) => [c.id, c.name]));

function EmptyNote({ children }: { children: React.ReactNode }) {
  return (
    <p className="rounded-lg-t border border-dashed border-os-border px-5 py-8 text-center text-[13px] text-os-muted">
      {children}
    </p>
  );
}

/**
 * The redesigned media library: real Identity assets first (Soul characters,
 * then Elements, then the honest "Train new avatar" card), Generated media
 * with a filter bar, then Uploads. Every card opens the same detail overlay
 * pattern as Production, with the same Upscale/Vary/Reuse settings verbs
 * dispatching through requestCompose. No seeded/sample media anywhere.
 */
export function MediaLibrary({
  manifest,
  tasks,
  account,
  uploads,
}: {
  manifest: MediaManifest;
  tasks: ProductionTask[];
  account: HiggsfieldAccount | null;
  uploads: UploadAsset[];
}) {
  return (
    <div className="flex flex-col gap-8">
      <IdentitySection account={account} />
      <GeneratedSection manifest={manifest} tasks={tasks} />
      <UploadsSection uploads={uploads} />
    </div>
  );
}

// ---------------------------------------------------------------------------
// A. Identity
// ---------------------------------------------------------------------------

function IdentitySection({ account }: { account: HiggsfieldAccount | null }) {
  const [trainOpen, setTrainOpen] = useState(false);
  const characters = account?.characters ?? [];
  const elements = account?.elements ?? [];

  return (
    <div>
      <div className="mb-3 flex flex-wrap items-center justify-between gap-3">
        <span className="glass-label">Identity — real Higgsfield Soul characters and Elements</span>
        {account && (
          <span className="font-mono text-[11px] tabular-nums text-os-dim">
            {characters.length + elements.length} assets
          </span>
        )}
      </div>
      {!account && (
        <p className="mb-3 text-[12.5px] text-os-muted">
          Higgsfield account not synced yet — identity assets appear here once data/higgsfield/account.json is
          populated.
        </p>
      )}
      <div className="flex flex-wrap gap-3">
        {characters.map((c) => (
          <IdentityCard
            key={c.id}
            asset={c}
            badge="Soul V2"
            compat="Works with Soul 2.0 + Soul Cinema"
          />
        ))}
        {elements.map((e) => (
          <IdentityCard
            key={e.id}
            asset={e}
            badge={`Element · ${e.category ?? 'character'}`}
            compat={`Works with ${ELEMENT_MODELS.size} models`}
          />
        ))}
        <TrainNewAvatarCard onOpen={() => setTrainOpen(true)} />
      </div>
      {trainOpen && <TrainAvatarModal onClose={() => setTrainOpen(false)} />}
    </div>
  );
}

function IdentityCard({ asset, badge, compat }: { asset: IdentityAsset; badge: string; compat: string }) {
  return (
    <div className="glass-panel flex w-40 flex-col gap-2 p-3">
      <div className="relative aspect-square overflow-hidden rounded-[12px] bg-black/30">
        <Image src={asset.portrait} alt={asset.name} fill sizes="160px" className="object-cover" />
      </div>
      <span className="truncate text-[12.5px] text-os-text">{asset.name}</span>
      <span className="glass-chip self-start px-2 py-0.5 font-mono text-[10px] text-os-muted">{badge}</span>
      <span className="text-[10.5px] leading-snug text-os-dim">{compat}</span>
    </div>
  );
}

function TrainNewAvatarCard({ onOpen }: { onOpen: () => void }) {
  return (
    <button
      type="button"
      onClick={onOpen}
      className="hoverable flex w-40 flex-col items-center justify-center gap-2 rounded-[20px] border border-dashed border-os-border p-3 text-center"
    >
      <div className="flex aspect-square w-full items-center justify-center rounded-[12px] border border-dashed border-os-border-strong bg-black/15">
        <Plus className="h-5 w-5 text-os-dim" strokeWidth={1.7} />
      </div>
      <span className="text-[12.5px] text-os-muted">Train new avatar</span>
    </button>
  );
}

function TrainAvatarModal({ onClose }: { onClose: () => void }) {
  const router = useRouter();
  const [name, setName] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Close melts the face (lib/use-melt); the library unmounts it once it has melted.
  const melt = useMeltClose(onClose);
  const close = melt.close;
  useEffect(() => {
    const prevOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') close();
    }
    window.addEventListener('keydown', onKey);
    return () => {
      document.body.style.overflow = prevOverflow;
      window.removeEventListener('keydown', onKey);
    };
  }, [close]);

  async function submit() {
    const trimmed = name.trim();
    if (!trimmed) {
      setError('Name the identity first.');
      return;
    }
    const campaignId = activeCampaign()?.id ?? contentCampaigns()[0]?.id;
    if (!campaignId) {
      setError('No campaign to file this under.');
      return;
    }
    setBusy(true);
    setError(null);
    try {
      const res = await fetch('/api/content/queue/spec', {
        method: 'POST',
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({
          toolKey: 'avatar',
          campaignId,
          title: `Train new Soul: ${trimmed}`,
          spec: { model: 'soul_2', prompt: `Train a new Soul identity named "${trimmed}".` },
        }),
      });
      if (res.ok) {
        close();
        router.refresh();
      } else {
        const body = await res.json().catch(() => ({}));
        setError(typeof body?.error === 'string' ? body.error : 'Could not queue the training task.');
      }
    } finally {
      setBusy(false);
    }
  }

  // Portalled to <body>: under a frosted ancestor the fixed layer would be trapped.
  return (
    <Layer>
      <div className="fixed inset-0 z-[100] flex items-center justify-center p-6" role="presentation" {...melt.layer}>
        {/* The scrim is the face's sibling (glass brief §3.2): it frosts and dims the page; the face frosts on its own and melts on close. */}
        <div className="gl-scrim" aria-hidden onClick={close} />
        <div
          ref={melt.faceRef}
          onAnimationEnd={melt.onAnimationEnd}
          className={`glass-panel glass-face--strong gl-materialise relative w-full max-w-md p-6${melt.melting ? ' is-melting' : ''}`}
          role="dialog"
          aria-modal="true"
          aria-label="Train new avatar"
        >
          <h3 className="text-[14px] text-os-text">Train a new avatar</h3>
          <p className="mt-2 text-[12.5px] leading-relaxed text-os-muted">
            Soul training runs on 5 to 20 photos of one person and takes about 10 minutes. Once trained, the identity
            works with Soul 2.0 and Soul Cinema like any character above. This queues the training run for the
            operator — there is no instant version.
          </p>
          <label className="mt-4 block">
            <span className="glass-label">Name</span>
            <input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g. identity 2"
              className="mt-1.5 w-full rounded-md-t border border-os-border-strong bg-transparent px-3 py-2 text-[13px] text-os-text outline-none focus:border-os-border"
            />
          </label>
          {error && <p className="mt-2 text-[12px] text-os-err">{error}</p>}
          <div className="mt-5 flex items-center justify-end gap-2">
            <button type="button" onClick={close} className="c-btn">
              Cancel
            </button>
            <button type="button" onClick={submit} disabled={busy} className="c-btn c-btn-primary">
              Queue training
            </button>
          </div>
        </div>
      </div>
    </Layer>
  );
}

// ---------------------------------------------------------------------------
// B. Generated
// ---------------------------------------------------------------------------

function toolKeyForAsset(asset: MediaAsset, task: ProductionTask | null): ContentToolKey {
  if (task) return TASK_TYPE_TOOL_KEY[task.type];
  return asset.kind === 'video' ? 'video' : 'image';
}

function reuseSpecForAsset(asset: MediaAsset, task: ProductionTask | null): TaskSpec {
  if (task) return taskSpecOrFallback(task);
  return {
    model: asset.model,
    params: {},
    aspectRatio: null,
    duration: null,
    prompt: '',
    medias: [{ role: asset.kind === 'video' ? 'video_references' : 'image_references', ref: assetUrl(asset) }],
    identity: null,
    skill: null,
    batch: 1,
  };
}

function upscaleSpecForAsset(asset: MediaAsset, task: ProductionTask | null): TaskSpec {
  if (task) return upscaleSpec(task);
  const video = asset.kind === 'video';
  return {
    model: video ? 'topaz_video' : 'topaz_image_generative',
    params: {},
    aspectRatio: null,
    duration: null,
    prompt: '',
    medias: [{ role: video ? 'video_references' : 'image_references', ref: assetUrl(asset) }],
    identity: null,
    skill: null,
    batch: 1,
  };
}

function GeneratedSection({ manifest, tasks }: { manifest: MediaManifest; tasks: ProductionTask[] }) {
  const [filter, setFilter] = useState<MediaFilterState>(DEFAULT_MEDIA_FILTER);
  const [active, setActive] = useState<MediaAsset | null>(null);

  const taskById = useMemo(() => new Map(tasks.map((t) => [t.id, t])), [tasks]);
  const { campaignIds, models } = useMemo(
    () => mediaFilterOptions(manifest.assets, taskById),
    [manifest.assets, taskById],
  );
  const filtered = useMemo(
    () => filterMediaAssets(manifest.assets, filter, taskById),
    [manifest.assets, filter, taskById],
  );

  function dispatchUpscale(asset: MediaAsset) {
    const task = taskById.get(asset.taskId) ?? null;
    requestCompose({
      toolKey: 'enhance',
      spec: upscaleSpecForAsset(asset, task),
      title: `Upscale — ${task?.title ?? asset.file}`,
      campaignId: task?.campaignId,
    });
  }

  function dispatchVary(asset: MediaAsset) {
    const task = taskById.get(asset.taskId) ?? null;
    requestCompose({
      toolKey: toolKeyForAsset(asset, task),
      spec: reuseSpecForAsset(asset, task),
      title: `${task?.title ?? asset.file} — variation`,
      campaignId: task?.campaignId,
    });
  }

  function dispatchReuse(asset: MediaAsset) {
    const task = taskById.get(asset.taskId) ?? null;
    requestCompose({
      toolKey: toolKeyForAsset(asset, task),
      spec: { ...reuseSpecForAsset(asset, task), prompt: '' },
      title: task?.title ?? asset.file,
      campaignId: task?.campaignId,
    });
  }

  if (manifest.assets.length === 0) {
    return (
      <div>
        <span className="glass-label">Generated</span>
        <div className="mt-3">
          <EmptyNote>No produced media yet — queued work delivers here</EmptyNote>
        </div>
      </div>
    );
  }

  const activeTask = active ? (taskById.get(active.taskId) ?? null) : null;

  return (
    <div>
      <div className="mb-3 flex flex-wrap items-center justify-between gap-3">
        <span className="glass-label">Generated — {manifest.assets.length} assets</span>
      </div>
      <FilterBar filter={filter} onChange={setFilter} campaignIds={campaignIds} models={models} />
      {filtered.length === 0 ? (
        <div className="mt-4">
          <EmptyNote>Nothing matches these filters</EmptyNote>
        </div>
      ) : (
        <div className="mt-4 grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          {filtered.map((a) => (
            <GeneratedCard
              key={a.id}
              asset={a}
              onOpen={() => setActive(a)}
              onUpscale={() => dispatchUpscale(a)}
              onVary={() => dispatchVary(a)}
              onReuse={() => dispatchReuse(a)}
            />
          ))}
        </div>
      )}
      {active && (
        <MediaDetailOverlay
          title={activeTask?.title ?? active.file}
          outputRef={assetUrl(active)}
          kind={active.kind}
          spec={activeTask?.spec}
          model={active.model}
          createdAt={activeTask?.createdAt ?? active.createdAt}
          deliveredAt={active.createdAt}
          campaignName={activeTask ? (CAMPAIGN_NAME_BY_ID.get(activeTask.campaignId) ?? activeTask.campaignId) : '—'}
          credits={active.credits}
          onClose={() => setActive(null)}
          actions={
            <>
              <VerbButton icon={Maximize2} label="Upscale" onClick={() => dispatchUpscale(active)} />
              <VerbButton icon={Shuffle} label="Vary" onClick={() => dispatchVary(active)} />
              <VerbButton icon={SlidersHorizontal} label="Reuse settings" onClick={() => dispatchReuse(active)} />
            </>
          }
        />
      )}
    </div>
  );
}

function FilterBar({
  filter,
  onChange,
  campaignIds,
  models,
}: {
  filter: MediaFilterState;
  onChange: (f: MediaFilterState) => void;
  campaignIds: string[];
  models: string[];
}) {
  return (
    <div className="flex flex-wrap items-center gap-2">
      <div className="relative">
        <Search
          className="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-os-dim"
          strokeWidth={1.7}
        />
        <input
          value={filter.query}
          onChange={(e) => onChange({ ...filter, query: e.target.value })}
          placeholder="Search title or prompt"
          className="w-56 rounded-md-t border border-os-border-strong bg-transparent py-1.5 pl-8 pr-3 text-[12.5px] text-os-text outline-none focus:border-os-border"
        />
      </div>
      <div className="flex items-center gap-1 rounded-md-t border border-os-border-strong p-0.5">
        {(['all', 'image', 'video'] as const).map((k) => (
          <button
            key={k}
            type="button"
            onClick={() => onChange({ ...filter, kind: k })}
            className={`rounded-[5px] px-2.5 py-1 text-[11px] capitalize transition-colors ${
              filter.kind === k ? 'bg-os-surface3 text-os-text' : 'text-os-dim hover:text-os-text'
            }`}
          >
            {k}
          </button>
        ))}
      </div>
      {campaignIds.length > 0 && (
        <select
          value={filter.campaignId}
          onChange={(e) => onChange({ ...filter, campaignId: e.target.value })}
          className="c-chip"
        >
          <option value="all">All campaigns</option>
          {campaignIds.map((id) => (
            <option key={id} value={id}>
              {CAMPAIGN_NAME_BY_ID.get(id) ?? id}
            </option>
          ))}
        </select>
      )}
      {models.length > 0 && (
        <select
          value={filter.model}
          onChange={(e) => onChange({ ...filter, model: e.target.value })}
          className="c-chip"
        >
          <option value="all">All models</option>
          {models.map((m) => (
            <option key={m} value={m}>
              {modelDisplayName(m)}
            </option>
          ))}
        </select>
      )}
    </div>
  );
}

function GeneratedCard({
  asset,
  onOpen,
  onUpscale,
  onVary,
  onReuse,
}: {
  asset: MediaAsset;
  onOpen: () => void;
  onUpscale: () => void;
  onVary: () => void;
  onReuse: () => void;
}) {
  return (
    <div className="glass-panel hoverable flex flex-col overflow-hidden">
      <button type="button" onClick={onOpen} className="block w-full text-left">
        <div className="relative aspect-video bg-black/30">
          {asset.kind === 'video' ? (
            <HoverVideo src={assetUrl(asset)} className="h-full w-full object-cover" />
          ) : (
            <Image
              src={assetUrl(asset)}
              alt={asset.file}
              fill
              sizes="(max-width: 640px) 100vw, (max-width: 1024px) 50vw, 25vw"
              className="object-cover"
            />
          )}
        </div>
      </button>
      <div className="flex flex-1 flex-col gap-2.5 p-3.5">
        <button type="button" onClick={onOpen} className="text-left">
          <span className="block truncate font-mono text-[11px] text-os-muted">{asset.file}</span>
        </button>
        <div className="flex flex-wrap items-center gap-1.5">
          <span className="glass-chip px-2 py-0.5 font-mono text-[10.5px] text-os-muted">
            {modelDisplayName(asset.model)}
          </span>
          <span className="glass-chip px-2 py-0.5 font-mono text-[10.5px] tabular-nums text-os-muted">
            {asset.credits} cr
          </span>
        </div>
        <div className="mt-auto grid grid-cols-3 gap-1.5 pt-1">
          <VerbButton icon={Maximize2} label="Upscale" onClick={onUpscale} />
          <VerbButton icon={Shuffle} label="Vary" onClick={onVary} />
          <VerbButton icon={SlidersHorizontal} label="Reuse settings" onClick={onReuse} />
        </div>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// C. Uploads
// ---------------------------------------------------------------------------

const UPLOAD_ICON: Record<UploadAsset['kind'], typeof ImageIcon> = { image: ImageIcon, video: Film, audio: AudioLines };

function UploadsSection({ uploads }: { uploads: UploadAsset[] }) {
  if (uploads.length === 0) {
    return (
      <div>
        <span className="glass-label">Uploads</span>
        <div className="mt-3">
          <EmptyNote>No uploads yet</EmptyNote>
        </div>
      </div>
    );
  }
  return (
    <div>
      <div className="mb-3 flex items-center justify-between gap-3">
        <span className="glass-label">Uploads — {uploads.length} files</span>
      </div>
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
        {uploads.map((u) => (
          <UploadCard key={u.id} upload={u} />
        ))}
      </div>
    </div>
  );
}

function UploadCard({ upload }: { upload: UploadAsset }) {
  const Icon = UPLOAD_ICON[upload.kind];

  function useAsReference() {
    // Audio uploads feed a video model's audio_references slot; image and
    // video uploads open their own tool with the matching reference role.
    const role =
      upload.kind === 'video' ? 'video_references' : upload.kind === 'audio' ? 'audio_references' : 'image_references';
    requestCompose({
      toolKey: upload.kind === 'image' ? 'image' : 'video',
      spec: { medias: [{ role, ref: uploadUrl(upload) }] },
    });
  }

  return (
    <div className="glass-panel flex items-center gap-3 p-3.5">
      <Icon className="h-4 w-4 shrink-0 text-os-dim" strokeWidth={1.7} />
      <div className="min-w-0 flex-1">
        <span className="block truncate text-[12.5px] text-os-text">{upload.title}</span>
        <span className="mt-0.5 block font-mono text-[10.5px] text-os-dim">{formatTimestamp(upload.createdAt)}</span>
      </div>
      <VerbButton icon={Link2} label="Use as reference" onClick={useAsReference} />
    </div>
  );
}
