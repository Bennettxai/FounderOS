import { beforeEach, describe, expect, it, vi } from "vitest";

import { getAppCatalog, getApps } from "./apps";
import { request } from "./base";

vi.mock("./base", () => ({
  request: vi.fn(),
}));

const mockedRequest = vi.mocked(request);

describe("apps API", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockedRequest.mockResolvedValue({ apps: [], count: 0 });
  });

  it("routes app catalog reads to the workspace selected by the caller", async () => {
    await getAppCatalog(undefined, "workspace-agency");

    expect(mockedRequest).toHaveBeenCalledWith(
      "/apps/catalog",
      expect.objectContaining({
        headers: { "X-Workspace-ID": "workspace-agency" },
        skipCache: true,
      }),
    );
  });

  it("routes installed app reads to the workspace selected by the caller", async () => {
    await getApps(undefined, false, "workspace-agency");

    expect(mockedRequest).toHaveBeenCalledWith(
      "/apps",
      expect.objectContaining({
        headers: { "X-Workspace-ID": "workspace-agency" },
        skipCache: true,
      }),
    );
  });
});
