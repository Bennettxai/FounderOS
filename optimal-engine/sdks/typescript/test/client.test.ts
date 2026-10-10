import { describe, it, expect, vi, beforeEach } from "vitest";
import { OptimalEngine } from "../src/client.js";
import { OptimalEngineError } from "../src/error.js";
import { asMemoryId, asWorkspaceId } from "../src/types.js";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function mockFetch(
  status: number,
  body: unknown,
  headers: Record<string, string> = {},
): void {
  vi.stubGlobal(
    "fetch",
    vi.fn().mockResolvedValue({
      ok: status >= 200 && status < 300,
      status,
      statusText: status === 200 ? "OK" : "Error",
      headers: new Headers(headers),
      json: () => Promise.resolve(body),
    }),
  );
}

function lastFetchCall(): { url: string; init: RequestInit } {
  const mock = vi.mocked(fetch);
  const call = mock.mock.calls[mock.mock.calls.length - 1];
  if (call === undefined) throw new Error("fetch was never called");
  return { url: call[0] as string, init: call[1] as RequestInit };
}

// ---------------------------------------------------------------------------
// OptimalEngine — construction
// ---------------------------------------------------------------------------

describe("OptimalEngine", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  describe("constructor", () => {
    it("should use default baseUrl when none is provided", () => {
      const client = new OptimalEngine();
      expect(client.http.baseUrl).toBe("http://localhost:4200");
    });

    it("should strip trailing slash from baseUrl", () => {
      const client = new OptimalEngine({ baseUrl: "http://example.com/" });
      expect(client.http.baseUrl).toBe("http://example.com");
    });

    it("should expose sub-clients", () => {
      const client = new OptimalEngine();
      expect(client.memory).toBeDefined();
      expect(client.workspaces).toBeDefined();
      expect(client.wiki).toBeDefined();
      expect(client.recall).toBeDefined();
      expect(client.subscriptions).toBeDefined();
      expect(client.surface).toBeDefined();
    });
  });

  // ---------------------------------------------------------------------------
  // ask
  // ---------------------------------------------------------------------------

  describe("ask", () => {
    it("should POST to /api/rag with query", async () => {
      mockFetch(200, { answer: "42", citations: [] });
      const client = new OptimalEngine({ baseUrl: "http://localhost:4200" });

      const result = await client.ask("What is the meaning of life?");

      const { url, init } = lastFetchCall();
      expect(url).toBe("http://localhost:4200/api/rag");
      expect(init.method).toBe("POST");
      const body = JSON.parse(init.body as string);
      expect(body.query).toBe("What is the meaning of life?");
      expect(result.answer).toBe("42");
    });

    it("should forward workspace, audience, format, bandwidth", async () => {
      mockFetch(200, { answer: "ok" });
      const client = new OptimalEngine();

      await client.ask("q", {
        workspace: "ws1",
        audience: "internal",
        format: "markdown",
        bandwidth: "l1",
      });

      const { init } = lastFetchCall();
      const body = JSON.parse(init.body as string);
      expect(body.workspace).toBe("ws1");
      expect(body.audience).toBe("internal");
      expect(body.format).toBe("markdown");
      expect(body.bandwidth).toBe("l1");
    });

    it("should apply default workspace when none is given per-call", async () => {
      mockFetch(200, { answer: "ok" });
      const client = new OptimalEngine({ workspace: "default-ws" });

      await client.ask("q");

      const { init } = lastFetchCall();
      const body = JSON.parse(init.body as string);
      expect(body.workspace).toBe("default-ws");
    });
  });

  // ---------------------------------------------------------------------------
  // search
  // ---------------------------------------------------------------------------

  describe("search", () => {
    it("should GET /api/search with q param", async () => {
      mockFetch(200, { results: [] });
      const client = new OptimalEngine();

      await client.search("typescript", { limit: 10 });

      const { url, init } = lastFetchCall();
      expect(url).toContain("/api/search");
      expect(url).toContain("q=typescript");
      expect(url).toContain("limit=10");
      expect(init.method).toBe("GET");
    });
  });

  // ---------------------------------------------------------------------------
  // grep
  // ---------------------------------------------------------------------------

  describe("grep", () => {
    it("should GET /api/grep with all provided params", async () => {
      mockFetch(200, { results: [] });
      const client = new OptimalEngine();

      await client.grep("pattern", {
        workspace: "ws2",
        intent: "decision",
        scale: "micro",
        modality: "text",
        limit: 5,
        literal: true,
      });

      const { url } = lastFetchCall();
      expect(url).toContain("q=pattern");
      expect(url).toContain("intent=decision");
      expect(url).toContain("literal=true");
    });
  });

  // ---------------------------------------------------------------------------
  // memory CRUD
  // ---------------------------------------------------------------------------

  describe("memory.create", () => {
    it("should POST to /api/memory", async () => {
      const mem = { id: "m1", content: "hello" };
      mockFetch(200, mem);
      const client = new OptimalEngine();

      const result = await client.memory.create({ content: "hello" });

      const { url, init } = lastFetchCall();
      expect(url).toContain("/api/memory");
      expect(init.method).toBe("POST");
      expect(result.content).toBe("hello");
    });

    it("should map camelCase to snake_case fields", async () => {
      mockFetch(200, { id: "m2", content: "fact" });
      const client = new OptimalEngine();

      await client.memory.create({
        content: "fact",
        isStatic: true,
        citationUri: "https://example.com",
      });

      const { init } = lastFetchCall();
      const body = JSON.parse(init.body as string);
      expect(body.is_static).toBe(true);
      expect(body.citation_uri).toBe("https://example.com");
    });
  });

  describe("memory.forget", () => {
    it("should POST to /api/memory/:id/forget", async () => {
      mockFetch(200, { id: "m1", forgotten_at: "2025-01-01" });
      const client = new OptimalEngine();

      await client.memory.forget(asMemoryId("m1"), { reason: "outdated" });

      const { url, init } = lastFetchCall();
      expect(url).toContain("/api/memory/m1/forget");
      expect(init.method).toBe("POST");
      const body = JSON.parse(init.body as string);
      expect(body.reason).toBe("outdated");
    });
  });

  describe("memory.delete", () => {
    it("should DELETE /api/memory/:id", async () => {
      mockFetch(204, undefined);
      const client = new OptimalEngine();

      await client.memory.delete(asMemoryId("m99"));

      const { url, init } = lastFetchCall();
      expect(url).toContain("/api/memory/m99");
      expect(init.method).toBe("DELETE");
    });
  });

  // ---------------------------------------------------------------------------
  // workspaces
  // ---------------------------------------------------------------------------

  describe("workspaces.create", () => {
    it("should POST to /api/workspaces", async () => {
      const ws = { id: "ws1", slug: "sample", name: "Example" };
      mockFetch(200, ws);
      const client = new OptimalEngine();

      const result = await client.workspaces.create({
        slug: "sample",
        name: "Example",
      });

      const { url, init } = lastFetchCall();
      expect(url).toContain("/api/workspaces");
      expect(init.method).toBe("POST");
      expect(result.slug).toBe("sample");
    });
  });

  describe("workspaces.updateConfig", () => {
    it("should PATCH /api/workspaces/:id/config", async () => {
      mockFetch(200, { theme: "dark" });
      const client = new OptimalEngine();

      await client.workspaces.updateConfig(asWorkspaceId("ws1"), {
        theme: "dark",
      });

      const { url, init } = lastFetchCall();
      expect(url).toContain("/api/workspaces/ws1/config");
      expect(init.method).toBe("PATCH");
    });
  });

  // ---------------------------------------------------------------------------
  // recall
  // ---------------------------------------------------------------------------

  describe("recall.actions", () => {
    it("should GET /api/recall/actions with topic param", async () => {
      mockFetch(200, { items: [] });
      const client = new OptimalEngine();

      await client.recall.actions({
        topic: "auth",
        actor: "alice",
        since: "2025-01-01",
      });

      const { url } = lastFetchCall();
      expect(url).toContain("/api/recall/actions");
      expect(url).toContain("topic=auth");
      expect(url).toContain("actor=alice");
      expect(url).toContain("since=2025-01-01");
    });
  });

  describe("recall.who", () => {
    it("should GET /api/recall/who", async () => {
      mockFetch(200, { items: [] });
      const client = new OptimalEngine();

      await client.recall.who({ topic: "deployment" });

      const { url } = lastFetchCall();
      expect(url).toContain("/api/recall/who");
      expect(url).toContain("topic=deployment");
    });
  });

  // ---------------------------------------------------------------------------
  // wiki
  // ---------------------------------------------------------------------------

  describe("wiki.list", () => {
    it("should GET /api/wiki", async () => {
      mockFetch(200, []);
      const client = new OptimalEngine({ workspace: "main" });

      await client.wiki.list();

      const { url } = lastFetchCall();
      expect(url).toContain("/api/wiki");
      expect(url).toContain("workspace=main");
    });
  });

  describe("wiki.contradictions", () => {
    it("should GET /api/wiki/contradictions", async () => {
      mockFetch(200, []);
      const client = new OptimalEngine();

      await client.wiki.contradictions({ workspace: "ws3" });

      const { url } = lastFetchCall();
      expect(url).toContain("/api/wiki/contradictions");
    });
  });

  // ---------------------------------------------------------------------------
  // status
  // ---------------------------------------------------------------------------

  describe("status", () => {
    it("should GET /api/status", async () => {
      mockFetch(200, { status: "ok", version: "0.1.0" });
      const client = new OptimalEngine();

      const result = await client.status();

      const { url } = lastFetchCall();
      expect(url).toContain("/api/status");
      expect(result.status).toBe("ok");
    });
  });

  // ---------------------------------------------------------------------------
  // OptimalEngineError
  // ---------------------------------------------------------------------------

  describe("OptimalEngineError", () => {
    it("should throw OptimalEngineError on non-2xx responses", async () => {
      vi.stubGlobal(
        "fetch",
        vi.fn().mockResolvedValue({
          ok: false,
          status: 404,
          statusText: "Not Found",
          json: () =>
            Promise.resolve({ code: "NOT_FOUND", message: "Memory not found" }),
        }),
      );

      const client = new OptimalEngine();

      await expect(
        client.memory.get(asMemoryId("missing")),
      ).rejects.toMatchObject({
        name: "OptimalEngineError",
        status: 404,
        code: "NOT_FOUND",
        message: "Memory not found",
      });
    });

    it("should fall back to HTTP status text when body has no message", async () => {
      vi.stubGlobal(
        "fetch",
        vi.fn().mockResolvedValue({
          ok: false,
          status: 500,
          statusText: "Internal Server Error",
          json: () => Promise.reject(new SyntaxError("invalid json")),
        }),
      );

      const client = new OptimalEngine();

      await expect(client.status()).rejects.toMatchObject({
        name: "OptimalEngineError",
        status: 500,
      });
    });

    it("should be instanceof Error", () => {
      const err = new OptimalEngineError(400, "BAD_REQUEST", "bad");
      expect(err).toBeInstanceOf(Error);
      expect(err).toBeInstanceOf(OptimalEngineError);
    });
  });
});
