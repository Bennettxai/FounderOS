import { describe, expect, it } from "vitest";

import { getDesktopModuleIds, getModuleCatalog, getModuleGroups } from "./workspaceModules";
import { MODULE_INFO } from "$lib/stores/desktop3d/moduleRegistry";
import { getModuleWindow } from "$lib/components/window/moduleWindowRegistry";
import { NAV_ITEMS } from "$lib/founderos/nav";

// The FounderOS modules show in the Command sidebar as their own group, in the
// FounderOS nav's order (lib/founderos/nav.ts, ported from FounderOS v1).
const FOUNDEROS = NAV_ITEMS.map((n) => [n.label, n.href]);

describe("FounderOS modules in the Command sidebar", () => {
  it("lists every FounderOS module as its own group, second after Operate", () => {
    const groups = getModuleGroups({});
    expect(groups[1].label).toBe("FounderOS");
    expect(groups[1].items.map((m) => [m.label, m.href])).toEqual(FOUNDEROS);
  });

  it("puts every FounderOS module in the 3D OS, each with a tile and a window", () => {
    const desktop = getDesktopModuleIds({});
    for (const m of getModuleCatalog().filter((m) => m.group === "FounderOS")) {
      expect(desktop, m.id).toContain(m.id);
      expect(MODULE_INFO[m.id], `${m.id} tile`).toMatchObject({ title: m.id === "founderos-os" ? "FounderOS" : m.label });
      expect(getModuleWindow(m.id)?.url, `${m.id} window`).toBe(m.href);
    }
  });

  it("is workspace based: a workspace shows only the FounderOS modules switched on for it", () => {
    const desktop = getDesktopModuleIds({ enabled_builtin_modules: ["dashboard", "founderos-trading", "founderos-finances"] });
    expect(desktop.filter((id) => id.startsWith("founderos-"))).toEqual(["founderos-finances", "founderos-trading"]);
  });
});
