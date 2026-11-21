'use client';

import { useEffect, useRef, useState, type ChangeEvent } from 'react';
import { useRouter } from 'next/navigation';
import { Plus, Sparkles, X } from 'lucide-react';
import { ParamFields, type ParamValues } from '@/components/content/ParamFields';
import {
  HF_MODELS,
  MODEL_FAMILIES,
  ROLE_LABELS,
  findModel,
  identitySupport,
  modelsByFamily,
  type HfMediaRole,
  type HfMediaSlot,
  type HfModel,
} from '@/lib/higgsfield/catalog';
import {
  assetUrl,
  estimateCredits,
  findContentTool,
  type ContentTool,
  type MediaAsset,
  type SpecMedia,
  type TaskSpec,
} from '@/lib/content-schema';
import { onComposeRequest, type ComposeRequest } from '@/lib/content-compose';
import {
  buildDefaultSpec,
  buildQueueInput,
  applyModelChange as deriveSpecForModel,
  removeMedia,
  setIdentity,
  setMedia,
  setParam,
  specFromPartial,
  toggleSkill,
  toolForModel,
} from '@/lib/content-composer-spec';
import type { Campaign } from '@/lib/content-campaigns';
import type { HiggsfieldAccount, IdentityAsset } from '@/lib/higgsfield/account';
import type { UploadAsset } from '@/lib/content-uploads';
import { useMeltEnd, useMeltPhase } from '@/lib/use-melt';
import { Layer } from '@/components/g/Overlays';

const inputCls =
  'w-full rounded-[9px] border border-[rgb(var(--mist-rgb) / 0.14)] bg-[var(--surface-2)] px-3 py-2 text-[13px] text-os-text outline-none placeholder:text-os-dim';

function chipCls(active: boolean): string {
  return `rounded-[8px] border px-2.5 py-1 font-mono text-[11.5px] transition-colors ${
    active
      ? 'glass-chip gl-on text-os-text'
      : 'border-[rgb(var(--mist-rgb) / 0.14)] text-os-muted hover:text-os-text'
  }`;
}

type PickerTab = 'library' | 'uploads' | 'identity' | 'upload';

function ReferenceSlot({
  slot,
  spec,
  model,
  libraryAssets,
  uploads,
  identityOptions,
  onSetMedia,
  onRemoveMedia,
}: {
  slot: HfMediaSlot;
  spec: TaskSpec;
  model: HfModel;
  libraryAssets: MediaAsset[];
  uploads: UploadAsset[];
  identityOptions: IdentityAsset[];
  onSetMedia: (role: HfMediaRole, media: SpecMedia) => void;
  onRemoveMedia: (role: HfMediaRole, ref: string) => void;
}) {
  const [pickerOpen, setPickerOpen] = useState(false);
  const [tab, setTab] = useState<PickerTab>('library');
  const [uploading, setUploading] = useState(false);
  const [uploadError, setUploadError] = useState<string | null>(null);

  const selected = spec.medias.filter((m) => m.role === slot.role);
  const max = slot.max ?? 1;
  const full = selected.length >= max;
  const label = ROLE_LABELS[slot.role];
  const showIdentityTab = slot.type === 'image' && identityOptions.length > 0;

  const libraryOptions = libraryAssets.filter((a) => a.kind === slot.type);
  const uploadOptions = uploads.filter((u) => u.kind === slot.type);

  function pick(ref: string, mediaLabel?: string) {
    onSetMedia(slot.role, { role: slot.role, ref, label: mediaLabel });
    setPickerOpen(false);
  }

  async function handleFile(e: ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    setUploading(true);
    setUploadError(null);
    try {
      const form = new FormData();
      form.append('file', file);
      const res = await fetch('/api/content/upload', { method: 'POST', body: form });
      const body = await res.json().catch(() => ({}));
      if (!res.ok) throw new Error(typeof body?.error === 'string' ? body.error : 'Upload failed.');
      const asset = body.upload as UploadAsset;
      pick(`/content-media/uploads/${asset.file}`, asset.title);
    } catch (err) {
      setUploadError(err instanceof Error ? err.message : 'Upload failed.');
    } finally {
      setUploading(false);
      e.target.value = '';
    }
  }

  return (
    <div className="rounded-[12px] border border-[rgb(var(--mist-rgb) / 0.12)] p-3">
      <div className="flex items-center justify-between">
        <span className="text-[12px] text-os-muted">
          {label}
          {max > 1 ? ` (up to ${max})` : ''}
        </span>
        {!full && (
          <button
            type="button"
            onClick={() => setPickerOpen((v) => !v)}
            className="flex items-center gap-1 text-[11.5px] text-os-accent"
          >
            <Plus className="h-3 w-3" strokeWidth={2} /> Add
          </button>
        )}
      </div>

      {selected.length > 0 && (
        <div className="mt-2 flex flex-wrap gap-2">
          {selected.map((m) => (
            <div
              key={m.ref}
              className="relative h-14 w-14 overflow-hidden rounded-[8px] border border-[rgb(var(--mist-rgb) / 0.14)] bg-black/30"
            >
              {slot.type === 'image' ? (
                // eslint-disable-next-line @next/next/no-img-element
                <img src={m.ref} alt={m.label ?? label} className="h-full w-full object-cover" />
              ) : (
                <div className="grid h-full w-full place-items-center px-1 text-center text-[9px] text-os-dim">
                  {m.label ?? slot.type}
                </div>
              )}
              <button
                type="button"
                onClick={() => onRemoveMedia(slot.role, m.ref)}
                aria-label={`Remove ${label}`}
                className="absolute right-0.5 top-0.5 grid h-4 w-4 place-items-center rounded-full bg-black/60 text-os-text"
              >
                <X className="h-2.5 w-2.5" />
              </button>
            </div>
          ))}
        </div>
      )}

      {pickerOpen && (
        <div className="mt-2.5 rounded-[10px] border border-[rgb(var(--mist-rgb) / 0.12)] bg-black/25 p-2.5">
          <div className="flex flex-wrap gap-1.5 border-b border-[var(--hairline)] pb-2">
            {(['library', 'uploads', ...(showIdentityTab ? (['identity'] as const) : []), 'upload'] as PickerTab[]).map(
              (t) => (
                <button
                  key={t}
                  type="button"
                  onClick={() => setTab(t)}
                  className={`rounded-[6px] px-2 py-1 text-[11px] capitalize ${
                    tab === t ? 'gl-on text-os-text' : 'text-os-dim hover:text-os-text'
                  }`}
                >
                  {t}
                </button>
              ),
            )}
          </div>

          <div className="mt-2 max-h-40 overflow-y-auto">
            {tab === 'library' &&
              (libraryOptions.length === 0 ? (
                <p className="text-[11.5px] text-os-dim">No {slot.type} assets in the library yet.</p>
              ) : (
                <div className="grid grid-cols-4 gap-1.5">
                  {libraryOptions.map((a) => (
                    <button
                      key={a.id}
                      type="button"
                      onClick={() => pick(assetUrl(a))}
                      className="aspect-square overflow-hidden rounded-[6px] border border-[rgb(var(--mist-rgb) / 0.12)]"
                    >
                      {/* eslint-disable-next-line @next/next/no-img-element */}
                      <img src={assetUrl(a)} alt="" className="h-full w-full object-cover" />
                    </button>
                  ))}
                </div>
              ))}

            {tab === 'uploads' &&
              (uploadOptions.length === 0 ? (
                <p className="text-[11.5px] text-os-dim">No uploads of this kind yet.</p>
              ) : (
                <div className="grid grid-cols-4 gap-1.5">
                  {uploadOptions.map((u) => (
                    <button
                      key={u.id}
                      type="button"
                      onClick={() => pick(`/content-media/uploads/${u.file}`, u.title)}
                      title={u.title}
                      className="aspect-square overflow-hidden rounded-[6px] border border-[rgb(var(--mist-rgb) / 0.12)]"
                    >
                      {u.kind === 'image' ? (
                        // eslint-disable-next-line @next/next/no-img-element
                        <img src={`/content-media/uploads/${u.file}`} alt="" className="h-full w-full object-cover" />
                      ) : (
                        <div className="grid h-full w-full place-items-center text-[9px] text-os-dim">{u.kind}</div>
                      )}
                    </button>
                  ))}
                </div>
              ))}

            {tab === 'identity' && (
              <div className="flex flex-wrap gap-1.5">
                {identityOptions.map((a) => (
                  <button
                    key={a.id}
                    type="button"
                    onClick={() => pick(a.portrait, a.name)}
                    className="flex items-center gap-1.5 rounded-[8px] border border-[rgb(var(--mist-rgb) / 0.12)] px-2 py-1 text-[11px] text-os-muted"
                  >
                    {/* eslint-disable-next-line @next/next/no-img-element */}
                    <img src={a.portrait} alt="" className="h-4 w-4 rounded-full object-cover" />
                    {a.name}
                  </button>
                ))}
              </div>
            )}

            {tab === 'upload' && (
              <div>
                <input
                  type="file"
                  onChange={handleFile}
                  disabled={uploading}
                  className="text-[11.5px] text-os-muted file:mr-2 file:rounded-[6px] file:border-0 file:bg-[var(--surface-2)] file:px-2 file:py-1 file:text-os-text"
                />
                {uploading && <p className="mt-1 text-[11px] text-os-dim">Uploading…</p>}
                {uploadError && <p className="mt-1 text-[11px] text-os-err">{uploadError}</p>}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

export function Composer({
  campaigns,
  defaultCampaignId,
  account,
  libraryAssets,
  uploads,
}: {
  campaigns: Campaign[];
  defaultCampaignId: string | null;
  account: HiggsfieldAccount | null;
  libraryAssets: MediaAsset[];
  uploads: UploadAsset[];
}) {
  const router = useRouter();

  const [open, setOpen] = useState(false);
  const [tool, setTool] = useState<ContentTool | null>(null);
  const [model, setModel] = useState<HfModel | null>(null);
  const [spec, setSpec] = useState<TaskSpec | null>(null);
  const [title, setTitle] = useState('');
  const [campaignId, setCampaignId] = useState('');
  const [showAllModels, setShowAllModels] = useState(false);

  const [refining, setRefining] = useState(false);
  const [refineError, setRefineError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  function closeComposer() {
    setOpen(false);
  }

  useEffect(() => {
    return onComposeRequest((detail: ComposeRequest) => {
      let nextTool: ContentTool | null = detail.toolKey ? (findContentTool(detail.toolKey) ?? null) : null;
      let nextModel: HfModel | null = nextTool ? (findModel(nextTool.defaultModel) ?? null) : null;
      if (detail.spec?.model) nextModel = findModel(detail.spec.model) ?? nextModel;
      if (!nextModel) return; // nothing resolvable — an honest no-op, never a broken drawer
      if (!nextTool) nextTool = toolForModel(nextModel);

      const nextSpec = detail.spec ? specFromPartial(detail.spec, nextModel, nextTool) : buildDefaultSpec(nextModel, nextTool);

      setTool(nextTool);
      setModel(nextModel);
      setSpec(nextSpec);
      setTitle(detail.title ?? '');
      setCampaignId(detail.campaignId ?? defaultCampaignId ?? campaigns[0]?.id ?? '');
      setShowAllModels(false);
      setSubmitError(null);
      setRefineError(null);
      setOpen(true);
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [defaultCampaignId, campaigns]);

  useEffect(() => {
    if (!open) return;
    const prevOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') closeComposer();
    }
    window.addEventListener('keydown', onKey);
    return () => {
      document.body.style.overflow = prevOverflow;
      window.removeEventListener('keydown', onKey);
    };
  }, [open]);

  // Close melts the drawer (lib/use-melt): `open` goes false at once (Escape, scroll lock), and only
  // the face lingers for the melt.
  const faceRef = useRef<HTMLDivElement>(null);
  const melt = useMeltPhase(open);
  const onMeltEnd = useMeltEnd(faceRef, melt.melting, melt.end);
  if (!melt.mounted || !tool || !model || !spec) return null;

  function handleModelChange(id: string) {
    const next = findModel(id);
    if (!next || !spec) return;
    setModel(next);
    setSpec(deriveSpecForModel(spec, next));
  }

  const support = identitySupport(model.id);
  const identityList: IdentityAsset[] = account
    ? support === 'soul'
      ? account.characters
      : support === 'element'
        ? account.elements
        : []
    : [];

  const familyGroups: Array<[string, HfModel[]]> = showAllModels
    ? MODEL_FAMILIES.map((f): [string, HfModel[]] => [f, modelsByFamily(f)]).filter(([, list]) => list.length > 0)
    : [[model.family, modelsByFamily(model.family)]];

  async function refinePrompt() {
    if (!spec || !model) return;
    if (!spec.prompt.trim()) {
      setRefineError('Write a direction first.');
      return;
    }
    setRefining(true);
    setRefineError(null);
    try {
      const res = await fetch('/api/content/refine-prompt', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ prompt: spec.prompt, modelId: model.id, campaignId: campaignId || undefined }),
      });
      const body = await res.json().catch(() => ({}));
      if (!res.ok || !body.ok) {
        setRefineError(typeof body?.error === 'string' ? body.error : 'Refinement unavailable');
        return;
      }
      setSpec((s) => (s ? { ...s, prompt: body.prompt } : s));
    } catch {
      setRefineError('Refinement unavailable');
    } finally {
      setRefining(false);
    }
  }

  async function submit() {
    if (!tool || !model || !spec) return;
    if (!title.trim()) {
      setSubmitError('Give the production a title.');
      return;
    }
    if (!campaignId) {
      setSubmitError('Pick a campaign.');
      return;
    }
    setSubmitting(true);
    setSubmitError(null);
    try {
      const input = buildQueueInput(tool.key, campaignId, title.trim(), spec);
      const res = await fetch('/api/content/queue', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(input),
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(typeof body?.error === 'string' ? body.error : 'Could not queue that production.');
      }
      closeComposer();
      router.refresh();
    } catch (err) {
      setSubmitError(err instanceof Error ? err.message : 'Could not queue that production.');
    } finally {
      setSubmitting(false);
    }
  }

  const credits = estimateCredits({ model: model.id, batch: spec.batch });

  // Portalled to <body>: under a frosted ancestor the fixed drawer would be trapped.
  return (
    <Layer>
      <div className="fixed inset-0 z-[100]" role="presentation" data-melting={melt.melting ? '' : undefined}>
        {/* The scrim is the drawer's sibling (glass brief §3.2): it frosts and dims the page; the drawer frosts on its own. */}
        <div className="gl-scrim" aria-hidden onClick={closeComposer} />
        <div
          ref={faceRef}
          onAnimationEnd={onMeltEnd}
          className={`glass-panel glass-face--strong gl-materialise--drawer fixed inset-y-0 right-0 flex w-full max-w-[560px] flex-col overflow-hidden rounded-l-[24px] rounded-r-none${melt.melting ? ' is-melting' : ''}`}
          role="dialog"
          aria-modal="true"
          aria-label={`Compose ${tool.label}`}
        >
          <div className="flex shrink-0 items-center justify-between border-b border-[var(--hairline)] px-6 py-4">
            <div>
              <span className="glass-label">{tool.label}</span>
              <h2 className="mt-1 text-[16px] font-medium text-os-text">Compose production</h2>
            </div>
            <button
              type="button"
              onClick={closeComposer}
              aria-label="Close"
              className="grid h-8 w-8 place-items-center rounded-full border border-[rgb(var(--mist-rgb) / 0.14)] text-os-dim transition-colors hover:text-os-text"
            >
              <X className="h-4 w-4" />
            </button>
          </div>

          <div className="flex-1 overflow-y-auto px-6 py-5">
            {/* Creative zone */}
            <div className="flex flex-col gap-4">
              <label className="flex flex-col gap-1.5">
                <span className="text-[12px] text-os-muted">Title</span>
                <input value={title} onChange={(e) => setTitle(e.target.value)} placeholder="What this production is" className={inputCls} />
              </label>

              <label className="flex flex-col gap-1.5">
                <span className="text-[12px] text-os-muted">Campaign</span>
                <select value={campaignId} onChange={(e) => setCampaignId(e.target.value)} className={inputCls}>
                  {campaigns.map((c) => (
                    <option key={c.id} value={c.id}>
                      {c.name}
                    </option>
                  ))}
                </select>
              </label>

              <label className="flex flex-col gap-1.5">
                <div className="flex items-center justify-between">
                  <span className="text-[12px] text-os-muted">Creative direction</span>
                  <button
                    type="button"
                    onClick={refinePrompt}
                    disabled={refining}
                    className="flex items-center gap-1.5 text-[11.5px] text-os-accent disabled:opacity-50"
                  >
                    <Sparkles className="h-3 w-3" strokeWidth={2} />
                    {refining ? 'Refining…' : 'Refine with director'}
                  </button>
                </div>
                <textarea
                  value={spec.prompt}
                  onChange={(e) => setSpec((s) => (s ? { ...s, prompt: e.target.value } : s))}
                  rows={4}
                  placeholder="What goes into the model — composition, references, mood"
                  className={`${inputCls} resize-none leading-relaxed`}
                />
                {refineError && <p className="text-[12px] text-os-err">{refineError}</p>}
              </label>

              {model.medias.length > 0 && (
                <div className="flex flex-col gap-2.5">
                  <span className="glass-label">References</span>
                  {model.medias.map((slot) => (
                    <ReferenceSlot
                      key={slot.role}
                      slot={slot}
                      spec={spec}
                      model={model}
                      libraryAssets={libraryAssets}
                      uploads={uploads}
                      identityOptions={slot.type === 'image' ? identityList : []}
                      onSetMedia={(role, media) => setSpec((s) => (s ? setMedia(s, model, role, media) : s))}
                      onRemoveMedia={(role, ref) => setSpec((s) => (s ? removeMedia(s, role, ref) : s))}
                    />
                  ))}
                </div>
              )}

              <div>
                <span className="glass-label">Identity</span>
                {support === 'none' ? (
                  <p className="mt-1.5 text-[12px] text-os-dim">This model takes no identity reference.</p>
                ) : !account ? (
                  <p className="mt-1.5 text-[12px] text-os-dim">Higgsfield not synced — no identity assets available.</p>
                ) : identityList.length === 0 ? (
                  <p className="mt-1.5 text-[12px] text-os-dim">No {support === 'soul' ? 'Soul characters' : 'Elements'} synced yet.</p>
                ) : (
                  <div className="mt-1.5 flex flex-wrap gap-2">
                    {identityList.map((a) => {
                      const active = spec.identity?.id === a.id;
                      return (
                        <button
                          key={a.id}
                          type="button"
                          onClick={() =>
                            setSpec((s) =>
                              s ? setIdentity(s, model, active ? null : { kind: a.kind, id: a.id, name: a.name }) : s,
                            )
                          }
                          className={`glass-chip flex items-center gap-2 py-1 pl-1 pr-3 text-[12px] transition-colors ${
                            active ? 'border-[var(--accent-line)] text-os-text' : 'text-os-muted hover:text-os-text'
                          }`}
                        >
                          {/* eslint-disable-next-line @next/next/no-img-element */}
                          <img src={a.portrait} alt="" className="h-5 w-5 rounded-full object-cover" />
                          {a.name}
                        </button>
                      );
                    })}
                  </div>
                )}
              </div>

              {tool.skill && (
                <label className="flex items-center justify-between gap-3 border-t border-[var(--hairline)] py-2.5">
                  <span className="text-[12.5px] text-os-text">
                    Apply skill: <span className="font-mono text-os-muted">{tool.skill}</span>
                  </span>
                  <button
                    type="button"
                    role="switch"
                    aria-checked={!!spec.skill}
                    onClick={() => setSpec((s) => (s ? toggleSkill(s, tool.skill) : s))}
                    className={`relative h-5 w-9 shrink-0 rounded-full border transition-colors ${
                      spec.skill ? 'border-os-accent/50 bg-os-accent/25' : 'border-[rgb(var(--mist-rgb) / 0.18)] bg-black/25'
                    }`}
                  >
                    <span
                      className={`absolute top-[3px] h-3 w-3 rounded-full transition-all ${
                        spec.skill ? 'left-[19px] bg-os-accent' : 'left-[3px] bg-os-dim'
                      }`}
                    />
                  </button>
                </label>
              )}

              {/* Parameter rail */}
              <div className="mt-2 flex flex-col gap-4 border-t border-[var(--hairline)] pt-4">
                <label className="flex flex-col gap-1.5">
                  <span className="text-[12px] text-os-muted">Model</span>
                  <select value={model.id} onChange={(e) => handleModelChange(e.target.value)} className={inputCls}>
                    {familyGroups.map(([family, models]) => (
                      <optgroup key={family} label={family}>
                        {models.map((m) => (
                          <option key={m.id} value={m.id}>
                            {m.name} — {m.provider}
                            {m.unlim ? ' · unlim' : ''}
                          </option>
                        ))}
                      </optgroup>
                    ))}
                  </select>
                  <button
                    type="button"
                    onClick={() => setShowAllModels((v) => !v)}
                    className="self-start text-[11.5px] text-os-dim transition-colors hover:text-os-text"
                  >
                    {showAllModels ? `Only ${model.family} models` : 'All models'}
                  </button>
                </label>

                {model.aspectRatios.length > 0 && (
                  <div>
                    <span className="text-[12px] text-os-muted">Aspect ratio</span>
                    <div className="mt-1.5 flex flex-wrap gap-1.5">
                      {model.aspectRatios.map((ar) => (
                        <button
                          key={ar}
                          type="button"
                          onClick={() => setSpec((s) => (s ? { ...s, aspectRatio: ar } : s))}
                          className={chipCls(spec.aspectRatio === ar)}
                        >
                          {ar}
                        </button>
                      ))}
                    </div>
                  </div>
                )}

                {model.durations && model.durations.length > 0 && (
                  <div>
                    <span className="text-[12px] text-os-muted">Duration</span>
                    <div className="mt-1.5 flex flex-wrap gap-1.5">
                      {model.durations.map((d) => (
                        <button
                          key={d}
                          type="button"
                          onClick={() => setSpec((s) => (s ? { ...s, duration: d } : s))}
                          className={chipCls(spec.duration === d)}
                        >
                          {d}s
                        </button>
                      ))}
                    </div>
                  </div>
                )}

                {!model.durations && model.durationRange && (
                  <label className="flex flex-col gap-1.5">
                    <span className="text-[12px] text-os-muted">Duration (seconds)</span>
                    <input
                      type="number"
                      min={model.durationRange.min}
                      max={model.durationRange.max}
                      value={spec.duration ?? model.durationRange.min}
                      onChange={(e) => {
                        const range = model.durationRange!;
                        const v = Math.min(Math.max(Number(e.target.value), range.min), range.max);
                        setSpec((s) => (s ? { ...s, duration: v } : s));
                      }}
                      className={inputCls}
                    />
                  </label>
                )}

                <ParamFields
                  model={model}
                  values={spec.params as ParamValues}
                  onChange={(name, value) => setSpec((s) => (s ? setParam(s, name, value) : s))}
                />

                {(model.output === 'image' || model.output === 'video') && (
                  <div className="flex items-center justify-between">
                    <span className="text-[12px] text-os-muted">Batch</span>
                    <div className="flex items-center gap-2">
                      <button
                        type="button"
                        onClick={() => setSpec((s) => (s ? { ...s, batch: Math.max(1, s.batch - 1) } : s))}
                        className="c-btn h-7 w-7 justify-center p-0"
                        aria-label="Decrease batch"
                      >
                        −
                      </button>
                      <span className="w-4 text-center font-mono text-[13px] text-os-text">{spec.batch}</span>
                      <button
                        type="button"
                        onClick={() => setSpec((s) => (s ? { ...s, batch: Math.min(8, s.batch + 1) } : s))}
                        className="c-btn h-7 w-7 justify-center p-0"
                        aria-label="Increase batch"
                      >
                        +
                      </button>
                    </div>
                  </div>
                )}
              </div>
            </div>
          </div>

          <div className="flex shrink-0 items-center justify-between border-t border-[var(--hairline)] px-6 py-4">
            <span className="font-mono text-[11.5px] text-os-dim">
              ~{credits} credit{credits === 1 ? '' : 's'}
            </span>
            <div className="flex items-center gap-2.5">
              {submitError && <span className="text-[12px] text-os-err">{submitError}</span>}
              <button type="button" onClick={closeComposer} className="c-btn">
                Cancel
              </button>
              <button type="button" onClick={submit} disabled={submitting} className="c-btn c-btn-primary">
                {submitting ? 'Queuing…' : 'Queue production'}
              </button>
            </div>
          </div>
        </div>
      </div>
    </Layer>
  );
}
