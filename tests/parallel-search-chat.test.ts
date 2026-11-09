/**
 * Scripted model; real chat resolver, AI SDK wrappers and MCP HTTP transport.
 * These deterministic tests make no paid model or live search request.
 */
import { afterAll, afterEach, beforeEach, expect, test, vi } from 'vitest';
import { mkdtempSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import path from 'node:path';
import Database from 'better-sqlite3';
import { z } from 'zod';
import { openDb } from '@/lib/db';
import { chatWithAgent } from '@/lib/agents/chat';
import type { RuntimeAgent } from '@/lib/agents/runtime';

type Call = [string, Record<string, unknown>];
type Observed = { url: string; userAgent: string | null; authenticated: boolean; body: any };
const observed: Observed[] = [];
let requests: Observed[] = [];
let gatewayRequests: any[] = [];
let plan: Call[] = [];
let mode: 'ok' | 'quota' | 'malformed' | 'hang' | 'missing' | 'tool-error' = 'ok';
let pending = () => {};
const search: Call = ['webSearch', { objective: 'Research market news', search_queries: ['public market news'] }];
const page: Call = ['webFetch', { urls: ['https://example.com/report'] }];
const agent: RuntimeAgent = {
  id: 'research', name: 'Research', description: 'Research public information.', departmentId: 'tech',
  run: async () => ({ ok: true, summary: 'ready' }),
  chatTools: () => [{ name: 'localLookup', description: 'Read local data.', parameters: z.object({}), execute: async () => 'local context' }],
};
function script(calls: Call[]) { plan = calls; }
function modelResponse(body: any) {
  gatewayRequests.push(body);
  const lastUser = body.prompt.findLastIndex((message: any) => message.role === 'user');
  const results = body.prompt.slice(lastUser + 1).flatMap((message: any) => message.content).filter((part: any) => part.type === 'tool-result');
  const failure = results.find((part: any) => part.output.type.startsWith('error'));
  const next = !failure && plan[results.length];
  const content = next
    ? [{ type: 'tool-call', toolCallId: 'step-' + results.length, toolName: next[0], input: JSON.stringify(next[1]) }]
    : [{ type: 'text', text: failure ? 'Web research unavailable: ' + JSON.stringify(failure.output) : results.map((part: any) => typeof part.output.value === 'string' ? part.output.value : JSON.stringify(part.output.value)).join('\n') || 'No web tools used.' }];
  return Response.json({
    content,
    finishReason: { unified: next ? 'tool-calls' : 'stop', raw: next ? 'tool_calls' : 'stop' },
    usage: { inputTokens: { total: 1, noCache: 1, cacheRead: 0, cacheWrite: 0 }, outputTokens: { total: 1, text: 1, reasoning: 0 } },
    warnings: [],
  });
}
function calls() { return requests.filter((r) => r.body?.method === 'tools/call'); }
beforeEach(() => {
  vi.stubEnv('LLM_PROVIDER', 'gateway');
  vi.stubEnv('AI_GATEWAY_API_KEY', 'fixture-model-key');
  vi.stubEnv('GLADOS_ENV_LOCAL', path.join(tmpdir(), 'founderos-no-saved-web-config'));
  vi.stubEnv('WEB_SEARCH_PROVIDER', undefined);
  vi.stubEnv('PARALLEL_API_KEY', undefined);
  requests = []; gatewayRequests = []; mode = 'ok'; pending = () => {}; plan = [];
  vi.stubGlobal('fetch', vi.fn(async (input: string | URL | Request, init?: RequestInit) => {
    const req = new Request(input, init);
    req.signal.throwIfAborted();
    const body = req.method === 'POST' ? JSON.parse(await req.text()) : undefined;
    if (req.url === 'https://ai-gateway.vercel.sh/v3/ai/language-model') return modelResponse(body);
    if (req.url !== 'https://search.parallel.ai/mcp') throw new Error('Unexpected external URL: ' + req.url);
    const record = { url: req.url, userAgent: req.headers.get('user-agent'), authenticated: req.headers.has('authorization'), body };
    requests.push(record); observed.push(record);
    if (req.method !== 'POST') return new Response(null, { status: 405 });
    if (body.id === undefined) return new Response(null, { status: 202 });
    let result: unknown;
    if (body.method === 'initialize') result = { protocolVersion: body.params.protocolVersion, capabilities: { tools: {} }, serverInfo: { name: 'fixture', version: '1' } };
    else if (body.method === 'tools/list') result = { tools: (mode === 'missing' ? [] : ['web_search', 'web_fetch']).map((name) => ({ name, inputSchema: { type: 'object' } })) };
    else if (body.method === 'tools/call') {
      if (mode === 'quota') return new Response('anonymous fixture quota', { status: 429, headers: { 'Retry-After': '60' } });
      if (mode === 'hang') {
        pending();
        return new Promise<Response>((_, reject) => {
          if (req.signal.aborted) reject(req.signal.reason);
          else req.signal.addEventListener('abort', () => reject(req.signal.reason), { once: true });
        });
      }
      result = mode === 'tool-error' ? { isError: true, content: [{ type: 'text', text: 'service unavailable' }] } : mode === 'malformed' ? { content: 'invalid' } : { content: [{ type: 'text', text: body.params.name === 'web_search' ? 'News: https://example.com/report' : 'Report: revenue grew 5%.' }] };
    } else throw new Error('Unexpected MCP method: ' + body.method);
    return Response.json({ jsonrpc: '2.0', id: body.id, result });
  }));
});
afterEach(() => { vi.restoreAllMocks(); vi.unstubAllGlobals(); vi.unstubAllEnvs(); });
afterAll(() => {
  // Optional transport trace for diagnosing this caller outside Vitest.
  if (process.env.FOUNDEROS_REQUEST_LOG) writeFileSync(process.env.FOUNDEROS_REQUEST_LOG, JSON.stringify({ userAgent: 'FounderOS/1.0.0', callerPath: 'lib/connectors/parallel-search.ts', requests: observed }, null, 2) + '\n');
});

test('fresh no-key chat discovers, searches, fetches, consumes and persists output', async () => {
  const db = openDb(':memory:'); script([search, page, ['localLookup', {}]]);
  const res = await chatWithAgent(db, [agent], agent.id, 'Research news and read its source.');
  expect(res.reply).toContain('https://example.com/report'); expect(res.reply).toContain('revenue grew 5%'); expect(res.reply).toContain('local context');
  expect(res.messages.find((m) => m.role === 'tool')?.toolCalls).toHaveLength(3);
  expect(requests.some((r) => r.body?.method === 'initialize')).toBe(true);
  expect(requests.some((r) => r.body?.method === 'tools/list')).toBe(true);
  expect(calls().map((r) => r.body.params.name)).toEqual(['web_search', 'web_fetch']);
  for (const r of requests) { expect(r.url).toBe('https://search.parallel.ai/mcp'); expect(r.userAgent).toBe('FounderOS/1.0.0'); expect(r.authenticated).toBe(false); }
  const ids = calls().map((r) => r.body.params.arguments.session_id);
  expect(ids[0]).toMatch(/^[0-9a-f-]{36}$/); expect(ids[1]).toBe(ids[0]);
  script([search]); await chatWithAgent(db, [agent], agent.id, 'Follow up.');
  expect(calls().at(-1)?.body.params.arguments.session_id).toBe(ids[0]);
});
test('saved disable and blank settings prevent web calls while keeping local tools', async () => {
  const file = path.join(mkdtempSync(path.join(tmpdir(), 'founderos-web-env-')), '.env.local');
  vi.stubEnv('GLADOS_ENV_LOCAL', file); vi.stubEnv('WEB_SEARCH_PROVIDER', 'parallel');
  for (const value of ['none', '']) {
    writeFileSync(file, 'WEB_SEARCH_PROVIDER=' + value + '\n'); script([['localLookup', {}]]);
    expect((await chatWithAgent(openDb(':memory:'), [agent], agent.id, 'Local data.')).reply).toBe('local context');
    expect(Object.keys(Object.fromEntries(gatewayRequests.at(-1).tools.map((tool: any) => [tool.name, tool])))).toEqual(['localLookup']);
  }
  expect(requests).toEqual([]);
});
test('explicit provider and account key honor saved configuration, including blank keys', async () => {
  const file = path.join(mkdtempSync(path.join(tmpdir(), 'founderos-web-key-')), '.env.local');
  vi.stubEnv('GLADOS_ENV_LOCAL', file); vi.stubEnv('WEB_SEARCH_PROVIDER', 'none');
  writeFileSync(file, 'WEB_SEARCH_PROVIDER=parallel\nPARALLEL_API_KEY=fixture-account-key\n'); script([search]);
  await chatWithAgent(openDb(':memory:'), [agent], agent.id, 'Research.');
  expect(calls()).toHaveLength(1); expect(requests.every((r) => r.authenticated)).toBe(true);
  writeFileSync(file, 'WEB_SEARCH_PROVIDER=parallel\nPARALLEL_API_KEY=\n'); requests = [];
  await chatWithAgent(openDb(':memory:'), [agent], agent.id, 'Anonymous research.');
  expect(requests.every((r) => !r.authenticated)).toBe(true);
});
test('unknown explicit provider fails without fallback', async () => {
  vi.stubEnv('WEB_SEARCH_PROVIDER', 'unsupported');
  await expect(chatWithAgent(openDb(':memory:'), [agent], agent.id, 'Research.')).rejects.toThrow('WEB_SEARCH_PROVIDER must be parallel or none');
  expect(gatewayRequests).toEqual([]); expect(requests).toEqual([]);
});
test('thread IDs survive reopening a database and rotate when history is reset', async () => {
  const file = path.join(mkdtempSync(path.join(tmpdir(), 'founderos-web-thread-')), 'chat.db'); script([search]);
  await chatWithAgent(openDb(file), [agent], agent.id, 'Research.'); const first = calls()[0].body.params.arguments.session_id;
  await chatWithAgent(openDb(file), [agent], agent.id, 'Continue.'); expect(calls().at(-1)?.body.params.arguments.session_id).toBe(first);
  const raw = new Database(file); raw.prepare('DELETE FROM agent_messages WHERE agent_id = ?').run(agent.id); raw.close();
  await chatWithAgent(openDb(file), [agent], agent.id, 'New conversation.'); expect(calls().at(-1)?.body.params.arguments.session_id).not.toBe(first);
});
test.each([true, false])('interleaved conversations retain distinct IDs (shared database: %s)', async (shared) => {
  script([search, page]);
  const db = openDb(':memory:');
  const other = { ...agent, id: 'research-two' };
  await Promise.all([chatWithAgent(db, [agent], agent.id, 'First task.'), chatWithAgent(shared ? db : openDb(':memory:'), [other], other.id, 'Second task.')]);
  const ids = new Set(calls().map((r) => r.body.params.arguments.session_id)); expect(ids.size).toBe(2);
  for (const id of ids) expect(calls().filter((r) => r.body.params.arguments.session_id === id).map((r) => r.body.params.name)).toEqual(['web_search', 'web_fetch']);
});
test('quota stays an error and retry preserves the conversation ID', async () => {
  const db = openDb(':memory:'); script([search]); mode = 'quota';
  expect((await chatWithAgent(db, [agent], agent.id, 'Research.')).reply).toContain('unavailable');
  const id = calls()[0].body.params.arguments.session_id; mode = 'ok';
  expect((await chatWithAgent(db, [agent], agent.id, 'Try again.')).reply).toContain('https://example.com/report');
  expect(calls().at(-1)?.body.params.arguments.session_id).toBe(id);
});
test.each(['malformed', 'missing', 'tool-error'] as const)('%s server output stays an error', async (failure) => {
  script([search]); mode = failure;
  expect((await chatWithAgent(openDb(':memory:'), [agent], agent.id, 'Research.')).reply).toContain('unavailable');
});
test.each(['file:///etc/passwd', 'javascript:alert(1)', 'https://user:pass@example.com/'])('reject unsafe URL %s before any request', async (url) => {
  script([['webFetch', { urls: [url] }]]);
  expect((await chatWithAgent(openDb(':memory:'), [agent], agent.id, 'Read URL.')).reply).toContain('unavailable'); expect(requests).toEqual([]);
});
test('caller cancellation aborts an in-flight MCP request', async () => {
  const controller = new AbortController(); mode = 'hang'; script([search]);
  const started = new Promise<void>((resolve) => { pending = resolve; });
  const result = chatWithAgent(openDb(':memory:'), [agent], agent.id, 'Research.', { signal: controller.signal });
  await started; controller.abort(new Error('operator cancelled'));
  await expect(result).rejects.toThrow(/cancel|abort/i); expect(calls()).toHaveLength(1);
});
test('the twenty-second deadline aborts a stalled request', async () => {
  const realTimeout = AbortSignal.timeout.bind(AbortSignal);
  const timeout = vi.spyOn(AbortSignal, 'timeout').mockImplementation(() => realTimeout(20));
  mode = 'hang'; script([search]);
  expect((await chatWithAgent(openDb(':memory:'), [agent], agent.id, 'Research.')).reply).toContain('unavailable');
  expect(timeout).toHaveBeenCalledWith(20_000); expect(calls()).toHaveLength(1);
});
