import { describe, expect, it } from "vitest";
import { initialDesktopIcons } from "./desktopPersistence";

describe("initialDesktopIcons", () => {
  it("puts FounderOS on the desktop, top left under Business OS", () => {
    const icon = initialDesktopIcons.find((i) => i.module === "founderos-os");
    expect(icon).toMatchObject({ id: "icon-founderos-os", label: "FounderOS", x: 0, y: 1 });
  });
});
