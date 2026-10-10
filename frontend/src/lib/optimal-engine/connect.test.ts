import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  detectLocalEngine,
  ensureEngineWorkspace,
  isSelectableEngineWorkspace,
} from "./connect";

describe("Optimal Engine workspace provisioning", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    delete (globalThis as { electron?: unknown }).electron;
  });

  it("hides Engine routing and benchmark scopes from BusinessOS", () => {
    for (const slug of ["default", "inbox", "team", "knowledge-intake", "default-businessos", "benchmark-locomo"]) {
      expect(isSelectableEngineWorkspace(slug)).toBe(false);
    }
    expect(isSelectableEngineWorkspace("businessos")).toBe(true);
    expect(isSelectableEngineWorkspace("agency-miosa")).toBe(true);
  });

  it("returns an existing same-slug workspace without creating a duplicate", async () => {
    const createWorkspace = vi.fn();
    (globalThis as { electron?: unknown }).electron = {
      engine: {
        workspaces: vi.fn().mockResolvedValue({
          ok: true,
          workspaces: [{ id: "ws-1", slug: "miosa", name: "MIOSA" }],
          message: "ok",
        }),
        createWorkspace,
      },
    };

    const result = await ensureEngineWorkspace(
      "http://localhost:4200",
      undefined,
      { slug: "MIOSA", name: "MIOSA" },
    );

    expect(result.id).toBe("ws-1");
    expect(createWorkspace).not.toHaveBeenCalled();
  });

  it("creates the missing workspace through the desktop bridge", async () => {
    const createWorkspace = vi.fn().mockResolvedValue({
      ok: true,
      workspace: { id: "ws-2", slug: "agency-miosa", name: "Agency MIOSA" },
      message: "created",
    });
    (globalThis as { electron?: unknown }).electron = {
      engine: {
        workspaces: vi.fn().mockResolvedValue({
          ok: true,
          workspaces: [],
          message: "ok",
        }),
        createWorkspace,
      },
    };

    const result = await ensureEngineWorkspace(
      "http://localhost:4200",
      undefined,
      { slug: "Agency-MIOSA", name: "Agency MIOSA" },
    );

    expect(result.slug).toBe("agency-miosa");
    expect(createWorkspace).toHaveBeenCalledWith(
      "http://localhost:4200",
      undefined,
      { slug: "agency-miosa", name: "Agency MIOSA" },
    );
  });

  it("uses the actual bundled engine URL when it selected a free port", async () => {
    (globalThis as { electron?: unknown }).electron = {
      engine: {
        status: vi.fn().mockResolvedValue({
          running: true,
          url: "http://127.0.0.1:4217",
          port: 4217,
          dataDir: "/tmp/engine",
        }),
      },
    };

    await expect(detectLocalEngine()).resolves.toBe("http://127.0.0.1:4217");
  });
});
