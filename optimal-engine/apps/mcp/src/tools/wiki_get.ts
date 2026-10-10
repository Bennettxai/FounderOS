import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { McpError, ErrorCode } from "@modelcontextprotocol/sdk/types.js";
import { z } from "zod";
import { engine, EngineError } from "../client.js";
import { config } from "../config.js";

export function registerWikiGet(server: McpServer): void {
  server.tool(
    "wiki_get",
    "Fetch a curated wiki page by slug. Audience-aware (sales/legal/exec/engineering/default). Returns rendered body + sources cited.",
    {
      slug: z
        .string()
        .min(1)
        .describe(
          "Wiki page slug, e.g. 'product-overview' or 'onboarding/week-1'.",
        ),
      workspace: z
        .string()
        .optional()
        .describe(
          `Workspace the wiki page belongs to. Defaults to "${config.defaultWorkspace}".`,
        ),
      audience: z
        .string()
        .optional()
        .describe(
          "Audience lens that selects the appropriate page variant (engineering, sales, legal, exec).",
        ),
    },
    async ({ slug, workspace, audience }) => {
      try {
        const result = await engine.wikiGet(slug, {
          workspace: workspace ?? config.defaultWorkspace,
          audience,
        });
        return {
          content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
        };
      } catch (err) {
        if (err instanceof EngineError) {
          if (err.status === 404) {
            throw new McpError(
              ErrorCode.InvalidParams,
              `Wiki page not found: "${slug}"`,
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
