import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@miosa/optimal-engine", () => ({
  createEngine: vi.fn((cfg) => ({ cfg })),
}));

vi.mock("$lib/api/workspace-admin", () => ({
  getEngineConfig: vi.fn(),
}));

import { getEngineConfig } from "$lib/api/workspace-admin";
import {
  getCachedEngineConfig,
  resetEngine,
  syncEngineConfig,
} from "./context";

describe("Optimal Engine context sync", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
    resetEngine();
    delete (globalThis as { electron?: unknown }).electron;
  });

  it("hydrates Electron's local engine cache for localhost workspace configs", async () => {
    const setConfig = vi.fn().mockResolvedValue({ ok: true });
    (globalThis as { electron?: unknown }).electron = {
      engine: { setConfig },
    };
    vi.mocked(getEngineConfig).mockResolvedValue({
      enabled: true,
      base_url: "http://localhost:4200",
      workspace: "agency-miosa",
      has_api_key: false,
    });

    await syncEngineConfig("workspace-123");

    expect(getCachedEngineConfig()).toEqual({
      enabled: true,
      base_url: "http://localhost:4200",
      api_key: "",
      workspace: "agency-miosa",
    });
    expect(setConfig).toHaveBeenCalledWith("workspace-123", {
      enabled: true,
      base_url: "http://localhost:4200",
      api_key: "",
      workspace: "agency-miosa",
    });
  });

  it("does not hydrate Electron for non-local engine URLs", async () => {
    const setConfig = vi.fn().mockResolvedValue({ ok: true });
    (globalThis as { electron?: unknown }).electron = {
      engine: { setConfig },
    };
    vi.mocked(getEngineConfig).mockResolvedValue({
      enabled: true,
      base_url: "https://engine.example.com",
      workspace: "external",
      has_api_key: true,
    });

    await syncEngineConfig("workspace-123");

    expect(setConfig).not.toHaveBeenCalled();
  });
});
