import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { McpError, ErrorCode } from "@modelcontextprotocol/sdk/types.js";
import { z } from "zod";
import { engine, EngineError } from "../client.js";
import { config } from "../config.js";

export function registerGrep(server: McpServer): void {
  server.tool(
    "grep",
    "Workspace-scoped semantic + literal grep. Filterable by intent (decision/action/fact/...), chunk scale (document/section/paragraph/chunk), modality. Returns matches with full signal trace. Use when looking for specific kinds of memories (decisions, schedules, ownership).",
    {
      q: z.string().min(1).describe("Search pattern — semantic or literal."),
      workspace: z
        .string()
        .optional()
        .describe(
          `Workspace to grep. Defaults to "${config.defaultWorkspace}".`,
        ),
      intent: z
        .enum([
          "decision",
          "action",
          "fact",
          "question",
          "observation",
          "schedule",
          "ownership",
        ])
        .optional()
        .describe("Filter results to a specific memory intent type."),
      scale: z
        .enum(["document", "section", "paragraph", "chunk"])
        .optional()
        .describe("Chunk granularity level to match against."),
    },
    async ({ q, workspace, intent, scale }) => {
      try {
        const result = await engine.grep({
          q,
          workspace: workspace ?? config.defaultWorkspace,
          intent,
          scale,
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
