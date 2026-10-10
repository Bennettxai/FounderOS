import assert from "node:assert/strict";
import http from "node:http";
import type { AddressInfo } from "node:net";
import test from "node:test";

import { healthUrl } from "./health-url.ts";

// The Go backend binds 127.0.0.1 only. Electron's Node resolves "localhost" to
// ::1 first, so a localhost probe is refused and a healthy dev backend reads as
// down. The probe must dial the IPv4 loopback the backend actually listens on.
test("health probe reaches a backend bound to 127.0.0.1 only", async () => {
  const server = http.createServer((_req, res) => res.writeHead(200).end("ok"));
  await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", resolve));
  const { port } = server.address() as AddressInfo;
  try {
    const status = await new Promise<number>((resolve, reject) => {
      http.get(healthUrl(port), (res) => resolve(res.statusCode ?? 0)).on("error", reject);
    });
    assert.equal(status, 200);
    assert.equal(healthUrl(port), `http://127.0.0.1:${port}/health`);
  } finally {
    server.close();
  }
});
