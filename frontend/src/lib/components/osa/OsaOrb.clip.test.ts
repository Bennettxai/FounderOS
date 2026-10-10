import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";

// Chromium (and so Electron) lets a transformed <video> escape a
// border-radius + overflow:hidden clip, which shows the MP4's black frame as a
// square behind the orb. clip-path clips the composited layer itself.
describe("OsaOrb video clip", () => {
  it("clips the video wrapper with clip-path, not only border-radius", () => {
    const src = readFileSync(resolve(__dirname, "OsaOrb.svelte"), "utf8");
    const wrap = src.match(/\.orb-video-wrap \{([^}]*)\}/)?.[1] ?? "";
    expect(wrap).toMatch(/clip-path:\s*circle\(50%\)/);
  });
});
