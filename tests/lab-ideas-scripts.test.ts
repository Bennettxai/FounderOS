import { afterEach, beforeAll, describe, expect, test, vi } from 'vitest';
import Database from 'better-sqlite3';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { LAB_DDL } from '@/lib/db';
import { labStore, type LabStore } from '@/lib/lab/store';
import { seedLabFixtures } from '@/lib/lab/fixtures';
import { OPEN_CAP, SYNC_OPEN_CAP, openSyncIdeas, engineConcept, generateIdeas, groundIdeas, ideaContext, ideaPrompt, wizardPrefill, type RawIdea } from '@/lib/lab/ideas';
import { diffBodies, dryBody, generateScript, markPublished, planScript, regenerateSection, saveEdit, scriptPrompt, type ScriptReply } from '@/lib/lab/scripts';
import { bestLines } from '@/lib/lab/dna';
import { canonicalKey, parseGoogleTrends, parseHn, parseProductHunt, parseRedditAtom, parseRedditJson, syncFeeds, toTrendItems } from '@/lib/lab/feeds';
import { writeLabBrain } from '@/lib/lab/brain';
import { GENERATED_MARKER } from '@/lib/brain-docs';
import { runLabIntel } from '@/lib/lab/intel';
import { breakoutsFor, contentLabAgent, ideaQueue, labAgentName, openTrends, specimen, viralProfile } from '@/lib/agents/content-lab';
import { deterministicGate } from '@/lib/lab/gate';
import type { FetchLike } from '@/lib/lab/apify-client';

/**
 * Ideas (grounding), scripts (gate, regenerate, versions, diffs), feeds, G-Brain docs and
 * the Lab agent against the fictional fixture world. Model calls are fakes; dry paths
 * set LAB_DRY=1. Nothing here reaches a network or a paid API.
 */

const NOW = new Date('2026-10-10T15:00:00.000Z');
const fresh = (): LabStore => {
  const db = new Database(':memory:');
  db.exec(LAB_DDL);
  return labStore(db);
};
let world: LabStore;
const newWorld = () => {
  const s = fresh();
  seedLabFixtures(s, NOW);
  return s;
};
beforeAll(() => {
  world = newWorld();
});
afterEach(() => {
  delete process.env.LAB_DRY;
  vi.useRealTimers();
});
const SKILL = '# voice\n- No AI-tell words (unlock, leverage, seamless, etc.). No "however".\n';
const readSkill = (slug: string) => (slug === 'reel' ? '# reel rubric\nHook in the first second.' : slug === 'brody-voice-speaking' ? SKILL : null);

describe('ideas are grounded (SPEC §4.11, AC4)', () => {
  test('the anti-invention check drops unknown clusters, unknown or own posts, and thin evidence; it keeps the rest', () => {
    const ctx = ideaContext(world, { now: NOW });
    const c = ctx.clusters[0];
    const [p1, p2] = c.posts;
    const own = world.posts.all().find((p) => p.isOwned)!.id;
    const base = { title: 'A real idea', clusterId: c.id, angle: 'the cost breakdown', tension: null, formatFamily: 'tutorial_walkthrough' };
    const raw: RawIdea[] = [
      { ...base, evidence: [{ postId: p1.postId, why: 'a' }, { postId: p2.postId, why: 'b' }] },
      { ...base, title: 'Invented cluster', clusterId: 'fx-c-nope', evidence: [{ postId: p1.postId, why: '' }, { postId: p2.postId, why: '' }] },
      { ...base, title: 'Invented post', evidence: [{ postId: p1.postId, why: '' }, { postId: 'fx-p-9999', why: '' }] },
      { ...base, title: 'Own post as evidence', evidence: [{ postId: p1.postId, why: '' }, { postId: own, why: '' }] },
      { ...base, title: 'One post twice', evidence: [{ postId: p1.postId, why: '' }, { postId: p1.postId, why: '' }] },
      { ...base, title: 'Unknown family', formatFamily: 'interpretive_dance', evidence: [{ postId: p1.postId, why: '' }, { postId: p2.postId, why: '' }] },
    ];
    const { kept, dropped } = groundIdeas(raw, ctx);
    expect(kept.map((k) => k.title)).toEqual(['A real idea', 'Unknown family']);
    expect(kept[1].formatFamily).toBe(ctx.families[0]);
    expect(dropped.map((d) => d.reason)).toEqual(['unknown cluster fx-c-nope', 'cites unknown post fx-p-9999', `cites unknown post ${own}`, 'fewer than two evidence posts']);
  });

  test('the prompt carries only real ids, timing, exhausted angles and the rule', () => {
    const ctx = ideaContext(world, { now: NOW });
    const prompt = ideaPrompt(ctx, world.ideas.all());
    for (const c of ctx.clusters) expect(prompt).toContain(`cluster=${c.id}`);
    expect(prompt).toMatch(/post=fx-p-\d{4}/);
    expect(prompt).toContain('An idea that cites any other id is thrown away');
    expect(prompt).toContain('(dismissed: Window closed)');
  });

  test('generation stores only grounded ideas, computes the numbers itself, and stops at 10 open', async () => {
    const s = newWorld();
    const ctx = ideaContext(s, { now: NOW });
    const ask = vi.fn(async () => ({
      ok: true as const,
      data: {
        ideas: ctx.clusters.slice(0, 5).flatMap((c, i) => [
          { title: `Grounded ${i}`, clusterId: c.id, angle: `fresh angle ${i}`, tension: { falseBelief: 'x', reframe: 'y' }, formatFamily: 'tutorial_walkthrough', evidence: c.posts.slice(0, 2).map((p) => ({ postId: p.postId, why: '' })) },
          { title: `Invented ${i}`, clusterId: c.id, angle: `made up ${i}`, tension: null, formatFamily: 'tutorial_walkthrough', evidence: [{ postId: c.posts[0].postId, why: '' }, { postId: 'post-that-does-not-exist', why: '' }] },
        ]),
      },
    }));
    const before = openSyncIdeas(s);
    const r = await generateIdeas(s, { now: NOW, ask });
    expect(ask).toHaveBeenCalledOnce();
    expect(r.by).toBe('claude');
    expect(r.dropped).toHaveLength(5);
    expect(r.added.length).toBe(Math.min(5, SYNC_OPEN_CAP - before));
    expect(s.ideas.all('open').length).toBeLessThanOrEqual(OPEN_CAP);
    const known = new Set(s.posts.all().filter((p) => !p.isOwned).map((p) => p.id));
    for (const i of s.ideas.all()) for (const e of i.evidence) expect(known.has(e.postId)).toBe(true);
    const added = r.added[0];
    expect(added.fit).toBeGreaterThan(0);
    expect(added.fit).toBeLessThanOrEqual(1);
    expect(added.hookRecs.length).toBeGreaterThan(0);
    expect(added.hookRecs.every((h) => h.n > 0)).toBe(true);
    expect(added.strand.length).toBeGreaterThan(0);
    expect(added.evidence.every((e) => e.why.length > 0)).toBe(true);
    // the LLM call is on the ledger at zero (local CLI), not dry
    expect(s.spend.all().filter((x) => x.provider === 'llm' && !x.dry && x.at === NOW.toISOString())).toEqual([expect.objectContaining({ actualUsd: 0 })]);
  });

  test('dry generation never calls the model and still grounds', async () => {
    process.env.LAB_DRY = '1';
    const s = newWorld();
    for (const i of s.ideas.all('open')) s.ideas.setStatus(i.id, 'dismissed', { reason: 'reset' });
    const ask = vi.fn();
    const r = await generateIdeas(s, { now: NOW, ask });
    expect(ask).not.toHaveBeenCalled();
    expect(r.by).toBe('dry');
    expect(r.added.length).toBeGreaterThan(0);
    expect(r.added.every((i) => i.evidence.length >= 2)).toBe(true);
  });

  test('hand-offs: an Engine concept inside its intake limits, a Wizard prefill that is never auto-sent', () => {
    const idea = world.ideas.get('fx-idea-1')!;
    const concept = engineConcept(idea, 'Agent memory');
    expect(concept.text.length).toBeGreaterThanOrEqual(3);
    expect(concept.text.length).toBeLessThanOrEqual(300);
    expect(concept.text).toContain('the cost nobody shows');
    const w = wizardPrefill(idea, 'Agent memory');
    expect(w).toMatchObject({ broadcast: false, source: 'content-lab' });
    expect(w.text).toMatch(/^Content Lab idea: Agent memory: the cost nobody shows\./);
  });
});

describe('scripts (SPEC §4.12, AC8)', () => {
  const reply = (plan: ReturnType<typeof planScript>, bad: boolean): ScriptReply => ({
    hooks: [{ form: 'results_first', trigger: 'curiosity', spoken: 'I gave my agent a memory and the bill doubled', onScreen: 'The bill doubled' }],
    sections: plan.sections.map((s) => ({ type: s.type, spoken: (bad && s.type === 'demo' ? 'This is seamless — ' : '') + Array.from({ length: s.budget }, () => 'word').join(' '), onScreen: '', visualNote: '' })),
    cta: 'Follow for part two.',
    caption: 'The real numbers.',
    titles: ['Agent memory, priced'],
  });

  test('the plan: budget = duration × your WPM ÷ 60, sections from the strand, the voice skill read and its banned words', () => {
    const plan = planScript(world, 'fx-idea-1', { now: NOW, durationS: 45, readSkill });
    expect(plan.wpm).toBe(world.priors.get('yours:wpm')!.value && (world.priors.get('yours:wpm')!.value as { value: number }).value);
    expect(plan.words).toBe(Math.round((45 * plan.wpm) / 60));
    expect(plan.sections.reduce((a, s) => a + s.budget, 0)).toBeGreaterThanOrEqual(plan.words - plan.sections.length);
    expect(plan.sections[plan.sections.length - 1].endPct).toBe(100);
    expect(plan.skills.map((s) => s.slug)).toEqual(['reel', 'brody-voice-speaking']);
    expect(plan.bannedWords).toEqual(expect.arrayContaining(['unlock', 'leverage', 'seamless', 'however']));
    const prompt = scriptPrompt(plan, 'Brand: a studio for operators.');
    for (const block of ['## CONTEXT', '## VOICE', '## EVIDENCE', '## STRUCTURE']) expect(prompt).toContain(block);
    expect(prompt).toContain('Brand: a studio for operators.');
    expect(prompt).toContain('### skill: reel');
    expect(prompt).toContain(`${plan.words} spoken words`);
    expect(prompt).toContain(`"${plan.evidence[0].hookText}"`);
  });

  test('a failing draft is regenerated once with its failures fed back; the passing one is kept as v1', async () => {
    const s = newWorld();
    const plan = planScript(s, 'fx-idea-1', { now: NOW, durationS: 45, readSkill });
    const prompts: string[] = [];
    const ask = vi.fn(async (p: string) => {
      prompts.push(p);
      return { ok: true as const, data: reply(plan, prompts.length === 1) };
    });
    const askCheck = vi.fn(async () => ({ ok: true as const, data: { voice: 4, tension: 4, evidence: 4, register: 4, blame: {} } }));
    const r = await generateScript(s, 'fx-idea-1', { now: NOW, durationS: 45, ask, askCheck, readSkill, context: '' });
    expect(r.attempts).toBe(2);
    expect(prompts[1]).toContain('## FIX THESE FROM THE LAST ATTEMPT');
    expect(prompts[1]).toMatch(/No dashes/);
    expect(r.version.gate.passed).toBe(true);
    expect(r.version.gate.scores).toEqual({ voice: 4, tension: 4, evidence: 4, register: 4 });
    expect(s.scripts.versions(r.scriptId)).toHaveLength(1);
  });

  test('a model score under 3 also regenerates; with two failures the better attempt stays and shows its failures', async () => {
    const s = newWorld();
    const plan = planScript(s, 'fx-idea-1', { now: NOW, durationS: 45, readSkill });
    const ask = vi.fn(async () => ({ ok: true as const, data: reply(plan, false) }));
    const askCheck = vi.fn(async () => ({ ok: true as const, data: { voice: 2, tension: 4, evidence: 4, register: 4, blame: { voice: 1 } } }));
    const r = await generateScript(s, 'fx-idea-1', { now: NOW, durationS: 45, ask, askCheck, readSkill, context: '' });
    expect(ask).toHaveBeenCalledTimes(2);
    expect(r.version.gate.passed).toBe(false);
    expect(r.version.gate.checks.find((c) => c.id === 'model-voice')).toMatchObject({ ok: false, section: 1 });
  });

  test('an unavailable model check leaves the gate unscored, never invented', async () => {
    const s = newWorld();
    const plan = planScript(s, 'fx-idea-1', { now: NOW, durationS: 45, readSkill });
    const r = await generateScript(s, 'fx-idea-1', { now: NOW, durationS: 45, readSkill, context: '', ask: async () => ({ ok: true, data: reply(plan, false) }), askCheck: async () => ({ ok: false, unavailable: true, error: 'no claude' }) });
    expect(r.version.gate.scores).toBeNull();
    expect(r.version.gate.passed).toBe(true);
  });

  test('dry: the draft passes the gate by construction; section regenerate and edits add versions; edits become voice samples', async () => {
    process.env.LAB_DRY = '1';
    const s = newWorld();
    const r = await generateScript(s, 'fx-idea-4', { now: NOW, readSkill });
    expect(r.version.gate.passed).toBe(true);
    expect(deterministicGate(dryBody(planScript(s, 'fx-idea-2', { now: NOW, readSkill }))).passed).toBe(true);
    const v2 = await regenerateSection(s, r.scriptId, 1, { now: NOW, readSkill });
    expect(v2.version.version).toBe(2);
    const edited = structuredClone(v2.version.body);
    edited.sections[1].spoken = `${edited.sections[1].spoken} I paid forty dollars for that lesson last month.`;
    const e = saveEdit(s, r.scriptId, edited, { now: NOW });
    expect(e).toMatchObject({ changed: true });
    expect(e.version.version).toBe(3);
    expect(e.version.editDiff).toContain('+I paid forty dollars for that lesson last month.');
    expect(e.version.gate.scores).toBeNull();
    expect(saveEdit(s, r.scriptId, edited).changed).toBe(false);
    expect(bestLines(s)).toContain('I paid forty dollars for that lesson last month.');
    expect(() => saveEdit(s, r.scriptId, { sections: 'no' })).toThrow(/malformed/);
  });

  test('diffs are per field and sentence-level', () => {
    const plan = planScript(world, 'fx-idea-1', { now: NOW, readSkill });
    const a = dryBody(plan);
    const b = structuredClone(a);
    b.cta = 'Follow for the build.';
    expect(diffBodies(a, b)).toBe(`@@ cta @@\n-${a.cta}\n+Follow for the build.`);
    expect(diffBodies(a, a)).toBe('');
  });

  test('mark published links your post, marks the idea used and starts outcomes', async () => {
    process.env.LAB_DRY = '1';
    const s = newWorld();
    const r = await generateScript(s, 'fx-idea-2', { now: NOW, readSkill });
    const own = s.posts.all().find((p) => p.isOwned)!;
    const other = s.posts.all().find((p) => !p.isOwned)!;
    expect(markPublished(s, r.scriptId, other.id)).toEqual({ ok: false, error: 'pick one of your own posts' });
    expect(markPublished(s, r.scriptId, own.id)).toEqual({ ok: true, noPrediction: true }); // 06 section 0: no frozen prediction, so Studio says it will not count
    expect(s.ideas.get('fx-idea-2')!.status).toBe('used');
    expect(s.outcomes.all().filter((o) => o.scriptId === r.scriptId)).toHaveLength(1);
  });
});

describe('trend feeds (all gated or dry; nothing leaves the process here)', () => {
  test('parsers read HN, Reddit JSON and Atom, Product Hunt and the Google Trends actor', () => {
    expect(parseHn({ hits: [{ objectID: '1', title: 'Agent memory layer', url: 'https://x.dev/a', points: 120, num_comments: 40, created_at: '2026-10-09T00:00:00Z' }, { objectID: '2' }] })).toEqual([expect.objectContaining({ feed: 'hn', externalId: '1', score: 120, comments: 40, source: 'hn' })]);
    expect(parseRedditJson({ data: { children: [{ data: { id: 'a', title: 'MCP servers', permalink: '/r/x/a', score: 50, num_comments: 9, created_utc: 1760000000 } }, { data: { id: 'b', title: 'Rules', stickied: true } }] } }, 'LocalLLaMA')).toEqual([expect.objectContaining({ feed: 'reddit', source: 'r/LocalLLaMA', url: 'https://www.reddit.com/r/x/a' })]);
    const atom = '<feed><entry><title>Voice cloning &amp; consent</title><link href="https://www.reddit.com/r/x/c"/><id>t3_c</id><updated>2026-10-09T00:00:00Z</updated></entry></feed>';
    expect(parseRedditAtom(atom, 'OpenAI')).toEqual([expect.objectContaining({ externalId: 'c', title: 'Voice cloning & consent' })]);
    expect(parseProductHunt({ data: { posts: { edges: [{ node: { id: '9', name: 'Memo', tagline: 'agent memory', url: 'https://ph/x', votesCount: 300, commentsCount: 12 } }] } } })[0]).toMatchObject({ title: 'Memo: agent memory', score: 300 });
    expect(parseGoogleTrends([{ searchTerm: 'claude', relatedQueries_rising: [{ query: 'claude memory', value: 450 }] }])[0]).toMatchObject({ feed: 'google_trends', title: 'claude memory', score: 450 });
  });

  test('the same story on two feeds carries both sources; headlines match clusters', () => {
    const items = [
      { feed: 'hn' as const, externalId: '1', url: 'https://blog.dev/memory?ref=hn', title: 'Agent memory costs', score: 1, comments: 0, source: 'hn', at: NOW.toISOString() },
      { feed: 'reddit' as const, externalId: 'z', url: 'https://blog.dev/memory', title: 'Agent memory costs', score: 1, comments: 0, source: 'r/LocalLLaMA', at: NOW.toISOString() },
    ];
    expect(canonicalKey(items[0])).toBe(canonicalKey(items[1]));
    const rows = toTrendItems(world, items, NOW.toISOString());
    expect(rows.every((r) => r.sourceCount === 2 && r.sources.join() === 'hn,r/LocalLLaMA')).toBe(true);
    expect(rows[0].clusterId).toBe('fx-c-memory');
  });

  test('dry: no fetch at all, fixture items, and a real install keeps them out of its tables', async () => {
    process.env.LAB_DRY = '1';
    const s = newWorld();
    const before = s.trendItems.all().length;
    const fetch = vi.fn() as unknown as FetchLike;
    const r = await syncFeeds(s, { now: NOW, fetch });
    expect(fetch).not.toHaveBeenCalled();
    expect(r.dry).toBe(true);
    expect(r.items.length).toBeGreaterThan(0);
    expect(r.persisted).toBe(false);
    expect(s.trendItems.all().length).toBe(before);
  });

  test('live free feeds go through the injected fetch, log 0-cost runs, and skip Product Hunt without a token', async () => {
    const s = newWorld();
    const calls: string[] = [];
    const fetch: FetchLike = async (url) => {
      calls.push(url);
      const body = url.includes('algolia') ? { hits: [{ objectID: 'h1', title: 'Prompt caching cut our bill', url: 'https://x.dev/c', points: 99, num_comments: 3, created_at: NOW.toISOString() }] } : { data: { children: [{ data: { id: 'r1', title: 'MCP servers worth installing', permalink: '/r/a/r1', score: 10, num_comments: 2, created_utc: NOW.getTime() / 1000 } }] } };
      return { ok: true, status: 200, json: async () => body, text: async () => '' };
    };
    const r = await syncFeeds(s, { now: NOW, fetch, terms: ['caching'], subreddits: ['LocalLLaMA'], apify: null, productHuntToken: null });
    expect(calls.map((u) => new URL(u).host)).toEqual(['hn.algolia.com', 'www.reddit.com']);
    expect(r.persisted).toBe(true);
    expect(r.skipped.map((x) => x.feed)).toEqual(['producthunt', 'google_trends']);
    expect(s.trendItems.all().find((i) => i.id === 'hn:h1')?.clusterId).toBe('fx-c-cache');
    expect(s.spend.all().filter((x) => !x.dry && x.at === NOW.toISOString())).toHaveLength(0);
    expect(s.runs.recent(5).filter((x) => x.actor?.startsWith('feed:')).every((x) => x.costUsd === 0)).toBe(true);
  });
});

describe('G-Brain docs, the intel cycle and the Lab agent', () => {
  test('a viral profile per family and the weekly digest, marked generated; a hand-edited page is kept', () => {
    const root = fs.mkdtempSync(path.join(os.tmpdir(), 'lab-brain-'));
    const r = writeLabBrain(world, { now: NOW, root });
    expect(r.paths).toEqual(expect.arrayContaining(['lab/viral-profile-tutorial-walkthrough.md', 'lab/digest-2026-w41.md']));
    const doc = fs.readFileSync(path.join(root, 'lab/viral-profile-tutorial-walkthrough.md'), 'utf8');
    expect(doc).toContain(GENERATED_MARKER);
    // v2 (04 5.5): case-control odds ratios, with the method line
    expect(doc).toContain('Method: case-control, Mantel-Haenszel odds ratios, Benjamini-Hochberg q 0.10.');
    expect(doc).toMatch(/\| Hook length \| 11-15 words \| \d+(\.\d)?x \| \d+ \| \d+ \| /);
    expect(doc).not.toMatch(/[—–]/);
    fs.writeFileSync(path.join(root, 'lab/viral-profile-talking-head.md'), '# mine, edited by hand');
    expect(writeLabBrain(world, { now: NOW, root }).skipped).toBe(1);
    expect(fs.readFileSync(path.join(root, 'lab/viral-profile-talking-head.md'), 'utf8')).toBe('# mine, edited by hand');
    expect(r.digest.headline).toMatch(/trends? opened, .*ideas? ready/);
  });

  test('a dry intel cycle runs every step and makes zero paid calls (AC6)', async () => {
    process.env.LAB_DRY = '1';
    const s = newWorld();
    const root = fs.mkdtempSync(path.join(os.tmpdir(), 'lab-intel-'));
    const r = await runLabIntel(s, { now: NOW, brainRoot: root });
    expect(r.dry).toBe(true);
    expect(r.steps.map((x) => [x.step, x.ok])).toEqual([['clusters', true], ['distill', true], ['trends', true], ['feeds', true], ['ideas', true], ['priors', true], ['brain', true]]);
    expect(r.profiles!.length).toBeGreaterThan(0);
    expect(s.spend.all().filter((x) => !x.dry && x.at >= NOW.toISOString())).toHaveLength(0);
  });

  test('the display name comes from roster data, never from code', () => {
    expect(labAgentName([])).toBe('Content Lab');
    expect(labAgentName([{ id: 'content-lab', name: 'Lab', tools: [] }])).toBe('Lab');
    expect(labAgentName([{ id: 'persona', name: 'Persona Name', tools: ['notion', 'content-lab'] }])).toBe('Persona Name');
    const a = contentLabAgent();
    expect(a).toMatchObject({ id: 'content-lab', name: 'Content Lab', departmentId: 'dept-marketing-growth' });
    expect(a.chatTools!().map((t) => t.name)).toEqual(['breakoutsFor', 'viralProfile', 'openTrends', 'ideaQueue', 'specimen', 'draftScript']);
  });

  test('read-only tools answer from the view with n; draftScript only creates a draft', async () => {
    process.env.LAB_DRY = '1';
    // The agent tools read the real clock (their windows end today); pin it to the fixture's NOW so the
    // 30-day breakout window still covers the fixture posts. Only Date is faked: the tools' awaits keep real timers.
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(NOW);
    const s = newWorld();
    const w = async () => ({ store: s, configured: true, setupHref: '/integrations#apify' });
    expect(await breakoutsFor(w, 'tutorial_walkthrough', 30)).toMatch(/breakouts in AI tool tutorials, last 30 days[\s\S]*baseline n \d+/);
    expect(await viralProfile(w, 'tutorial_walkthrough')).toMatch(/n \d+, 90% /);
    expect(await openTrends(w)).toMatch(/momentum/);
    expect(await ideaQueue(w)).toContain('fx-idea-1');
    expect(await specimen(w, 'fx-p-0001')).toContain('fx-p-0001');
    const before = s.scripts.all().length;
    const tools = contentLabAgent(w).chatTools!();
    const out = String(await tools.find((t) => t.name === 'draftScript')!.execute({ ideaId: 'fx-idea-3' }));
    expect(out).toMatch(/^Draft script-\w+ written/);
    expect(s.scripts.all().length).toBe(before + 1);
    expect(s.scripts.all().filter((x) => x.status === 'published').length).toBe(1);
    const run = await contentLabAgent(w).run();
    expect(run.ok).toBe(true);
    expect(run.summary).toMatch(/^Dry run\. clusters:/);
  });
});

describe('applyEditPatch (the UI trees post hand edits as a patch)', () => {
  test('lays sections and the hook choice over the current body; unknown keys stay malformed', async () => {
    const { applyEditPatch } = await import('@/lib/lab/scripts');
    const cur = {
      durationS: 60, wpm: 160, hookChoice: 0, cta: 'Follow', caption: 'c', titles: ['t'],
      hooks: [{ form: 'result_first', trigger: 'curiosity', spoken: 'a', onScreen: '', words: 1 }, { form: 'result_first', trigger: 'curiosity', spoken: 'b', onScreen: '', words: 1 }],
      sections: [{ type: 'hook', startPct: 0, endPct: 50, spoken: 'one', onScreen: '', visualNote: '' }, { type: 'cta', startPct: 50, endPct: 100, spoken: 'two', onScreen: '', visualNote: '' }],
    } as never as Parameters<typeof applyEditPatch>[0];
    const out = applyEditPatch(cur, { hookChoice: 1, sections: [{ spoken: 'uno', onScreen: 'x' }, { spoken: 'two', onScreen: '' }] }) as typeof cur;
    expect(out.hookChoice).toBe(1);
    expect(out.sections[0]).toMatchObject({ type: 'hook', spoken: 'uno', onScreen: 'x' });
    expect(out.sections[1].spoken).toBe('two');
    expect(applyEditPatch(cur, { nonsense: true })).toEqual({ nonsense: true });
    expect(applyEditPatch(cur, { sections: [{ spoken: 'only one' }] })).toEqual({ sections: [{ spoken: 'only one' }] });
  });
});
