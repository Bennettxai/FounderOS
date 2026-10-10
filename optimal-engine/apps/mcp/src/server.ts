/**
 * MCP server setup — registers all tools and returns the configured server.
 * Transport wiring lives in index.ts (stdio) so this module stays testable.
 */

import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { registerAsk } from "./tools/ask.js";
import { registerSearch } from "./tools/search.js";
import { registerGrep } from "./tools/grep.js";
import { registerProfile } from "./tools/profile.js";
import { registerAddMemory } from "./tools/add_memory.js";
import { registerForgetMemory } from "./tools/forget_memory.js";
import { registerRecall } from "./tools/recall.js";
import { registerWikiGet } from "./tools/wiki_get.js";
import { registerWorkspaces } from "./tools/workspaces.js";
import { registerRenderContext } from "./tools/render_context.js";

export function createServer(): McpServer {
  const server = new McpServer({
    name: "optimal-engine",
    version: "0.3.0",
  });

  registerAsk(server);
  registerSearch(server);
  registerGrep(server);
  registerProfile(server);
  registerAddMemory(server);
  registerForgetMemory(server);
  registerRecall(server);
  registerWikiGet(server);
  registerWorkspaces(server);
  registerRenderContext(server);

  return server;
}
