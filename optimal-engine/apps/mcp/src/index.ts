#!/usr/bin/env node
/**
 * Optimal Engine MCP server — stdio entry point.
 *
 * Reads engine URL from OPTIMAL_ENGINE_URL (default http://localhost:4200),
 * default workspace from OPTIMAL_WORKSPACE (default "default"), and an
 * optional API key from OPTIMAL_API_KEY (sent as a Bearer token).
 *
 * Connects via stdio so MCP-capable agent clients can spawn this binary
 * directly. Tool surface is registered in server.ts.
 */

import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import { createServer } from "./server.js";

async function main() {
  const server = createServer();
  const transport = new StdioServerTransport();
  await server.connect(transport);
  // Graceful shutdown on SIGINT / SIGTERM when an MCP host disconnects.
  for (const sig of ["SIGINT", "SIGTERM"] as const) {
    process.on(sig, () => {
      void transport.close();
      process.exit(0);
    });
  }
}

main().catch((err) => {
  // MCP servers communicate over stdout — error logs MUST go to stderr.
  console.error("[optimal-engine-mcp] fatal:", err);
  process.exit(1);
});
