import assert from "node:assert/strict";
import test from "node:test";

import { mediaPermission, originOf } from "./media.ts";

const own = ["http://localhost:5273", "app://localhost"];

test("asks macOS for the camera when the app's own page wants video", () => {
  assert.deepEqual(
    mediaPermission({ permission: "media", origin: "http://localhost:5273", mediaTypes: ["video"], trustedOrigins: own }),
    { allow: true, ask: ["camera"] },
  );
});

test("asks for both when the page wants camera and microphone", () => {
  assert.deepEqual(
    mediaPermission({ permission: "media", origin: "app://localhost", mediaTypes: ["video", "audio"], trustedOrigins: own }),
    { allow: true, ask: ["camera", "microphone"] },
  );
});

test("never grants media to a page that is not the app", () => {
  assert.deepEqual(
    mediaPermission({ permission: "media", origin: "https://evil.example", mediaTypes: ["video"], trustedOrigins: own }),
    { allow: false, ask: [] },
  );
});

test("leaves other permissions to Electron's default", () => {
  assert.equal(
    mediaPermission({ permission: "notifications", origin: "http://localhost:5273", mediaTypes: [], trustedOrigins: own }),
    null,
  );
});

test("origins are scheme and host, including the packaged app:// scheme", () => {
  assert.equal(originOf("app://localhost/window?x=1"), "app://localhost");
  assert.equal(originOf("http://localhost:5273/window"), "http://localhost:5273");
  assert.equal(originOf("not a url"), "");
});
