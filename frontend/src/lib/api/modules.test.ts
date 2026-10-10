import { beforeEach, describe, expect, it, vi } from "vitest";

const { request } = vi.hoisted(() => ({ request: vi.fn() }));

vi.mock("./base", () => ({ request }));

import { getWorkspaceModules } from "./modules";

describe("getWorkspaceModules", () => {
  beforeEach(() => {
    request.mockReset();
  });

  it("requests modules for the workspace passed by the caller", async () => {
    request.mockResolvedValue({
      data: [
        {
          id: "module-1",
          slug: "command-intelligence",
          name: "Command Intelligence",
          icon: "command",
          config: { sidebar_group: "Phase 1", sidebar_order: 1 },
          share_scope: "workspace",
        },
      ],
      pagination: { total_items: 1 },
    });

    const modules = await getWorkspaceModules("workspace-terrawatt");

    expect(request).toHaveBeenCalledWith(
      "/modules?workspace_id=workspace-terrawatt",
      { skipAuthRedirect: true },
    );
    expect(modules).toEqual([
      expect.objectContaining({
        id: "module-1",
        key: "command-intelligence",
        sidebar_group: "Phase 1",
        sidebar_order: 1,
      }),
    ]);
  });
});
