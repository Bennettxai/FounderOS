import { describe, expect, it } from "vitest";

import {
  DEFAULT_MODULE_PROFILE,
  getDesktopModuleIds,
  getEnabledModuleIds,
  getModuleGroups,
  getWorkspaceSidebarGroups,
  resolveWorkspaceModuleProfile,
} from "./workspaceModules";

describe("public workspace module foundation", () => {
  it("uses the primitive profile for a new workspace", () => {
    expect(resolveWorkspaceModuleProfile({})).toBe(DEFAULT_MODULE_PROFILE);
    expect(getEnabledModuleIds({})).toEqual(
      expect.arrayContaining([
        "dashboard",
        "agents",
        "knowledge",
        "calendar",
        "communications",
        "relationships",
        "projects",
        "tasks",
      ]),
    );
  });

  it("ignores private or unknown profile names", () => {
    expect(
      resolveWorkspaceModuleProfile({ module_profile: "private-profile" }),
    ).toBe(DEFAULT_MODULE_PROFILE);
  });

  it("honors an explicit workspace module list", () => {
    expect(
      getEnabledModuleIds({
        enabled_builtin_modules: ["dashboard", "projects", "content"],
      }),
    ).toEqual(["dashboard", "projects", "my-content"]);
  });

  it("drops unknown module ids and empty navigation groups", () => {
    const groups = getModuleGroups({
      enabled_builtin_modules: ["dashboard", "not-a-module"],
    });

    expect(groups).toHaveLength(1);
    expect(groups[0].label).toBe("Operate");
    expect(groups[0].items.map((item) => item.id)).toEqual(["dashboard"]);
  });

  it("projects foundational modules into the desktop", () => {
    const modules = getDesktopModuleIds({});

    expect(modules).toEqual(
      expect.arrayContaining([
        "dashboard",
        "agents",
        "knowledge",
        "calendar",
        "communication",
        "relationships",
        "projects",
        "tasks",
      ]),
    );
  });

  it("builds navigation from built-in and empty custom-module slots", () => {
    const groups = getWorkspaceSidebarGroups(
      {
        enabled_builtin_modules: ["dashboard", "agents", "apps"],
        sidebar_layout: [
          { label: "Command", builtin: ["dashboard", "agents"] },
          { label: "Custom", custom: ["example-module"] },
          { label: "Systems", builtin: ["apps"] },
        ],
      },
      [
        {
          id: "module-example",
          key: "example-module",
          name: "Example Module",
          icon: "box",
          sidebar_group: "Custom",
          sidebar_order: 10,
          share_scope: "workspace",
        },
      ],
    );

    expect(groups.map((group) => group.label)).toEqual([
      "Command",
      "Custom",
      "Systems",
    ]);
    expect(groups[1].items[0].href).toBe("/modules/module-example");
  });
});
