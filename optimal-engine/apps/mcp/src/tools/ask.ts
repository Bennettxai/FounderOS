import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { z } from "zod";
import { engine, EngineError } from "../client.js";
import { config } from "../config.js";
import { McpError, ErrorCode } from "@modelcontextprotocol/sdk/types.js";

export function registerAsk(server: McpServer): void {
  server.tool(
    "ask",
    "Ask Optimal Engine a workspace-scoped question. Returns governed retrieval/context output with evidence, memory, and projection metadata when available. Use this when an agent needs authorized context before answering or acting.",
    {
      query: z.string().min(1).describe("The question to ask the engine."),
      workspace: z
        .string()
        .optional()
        .describe(
          `Workspace to query. Defaults to "${config.defaultWorkspace}".`,
        ),
      audience: z
        .string()
        .optional()
        .describe(
          "Audience lens that shapes the response (e.g. engineering, sales, legal, exec). Omit for default.",
        ),
    },
    async ({ query, workspace, audience }) => {
      try {
        const result = await engine.ask({
          query,
          workspace: workspace ?? config.defaultWorkspace,
          audience,
        });
        return {
          content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
        };
      } catch (err) {
        if (err instanceof EngineError) {
          throw new McpError(
            ErrorCode.InternalError,
            `Engine error ${err.status}: ${err.message}`,
          );
        }
        throw err;
      }
    },
  );
}
