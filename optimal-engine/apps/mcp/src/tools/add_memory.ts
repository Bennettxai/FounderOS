import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { McpError, ErrorCode } from "@modelcontextprotocol/sdk/types.js";
import { z } from "zod";
import { engine, EngineError } from "../client.js";
import { config } from "../config.js";

export function registerAddMemory(server: McpServer): void {
  server.tool(
    "add_memory",
    "Submit durable information to Optimal Engine through the governed memory intake path. The engine preserves source context and should create reviewable/persistent memory according to policy. Use for facts, decisions, observations, or source-backed notes that should survive the session.",
    {
      content: z
        .string()
        .min(1)
        .describe(
          "The content to submit. Be precise and include enough source context for later review.",
        ),
      workspace: z
        .string()
        .optional()
        .describe(
          `Workspace to store into. Defaults to "${config.defaultWorkspace}".`,
        ),
      is_static: z
        .boolean()
        .optional()
        .describe(
          "Mark true only for stable facts that should be treated as durable context.",
        ),
      audience: z
        .string()
        .optional()
        .describe(
          "Audience lens this memory is relevant to (engineering, sales, legal, exec, all).",
        ),
      citation_uri: z
        .string()
        .optional()
        .describe(
          "URI of the source document or conversation this memory originates from.",
        ),
      source_chunk_id: z
        .string()
        .optional()
        .describe(
          "ID of the specific source chunk this memory is derived from.",
        ),
    },
    async ({
      content,
      workspace,
      is_static,
      audience,
      citation_uri,
      source_chunk_id,
    }) => {
      try {
        const result = await engine.addMemory({
          content,
          workspace: workspace ?? config.defaultWorkspace,
          is_static,
          audience,
          citation_uri,
          source_chunk_id,
        });
        return {
          content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
        };
      } catch (err) {
        if (err instanceof EngineError) {
          if (err.status === 422) {
            throw new McpError(
              ErrorCode.InvalidParams,
              `Memory integrity check failed: ${err.message}`,
            );
          }
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
