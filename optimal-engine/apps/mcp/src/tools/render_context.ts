import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { z } from "zod";
import { engine, EngineError } from "../client.js";
import { config } from "../config.js";
import { McpError, ErrorCode } from "@modelcontextprotocol/sdk/types.js";

export function registerRenderContext(server: McpServer): void {
  server.tool(
    "render_context",
    "Get audience-specific context and prompt guidance for rendering a source document as HTML. Use this when generating tailored HTML artifacts (exec summaries, engineering docs, onboarding guides) from knowledge base content. Returns the assembled context, render spec constraints, and a prompt you can use to generate the final HTML.",
    {
      source: z
        .string()
        .min(1)
        .describe(
          "Wiki slug or search query for the source content to render.",
        ),
      spec: z
        .string()
        .min(1)
        .describe(
          'Render spec name — defines the target audience and output constraints. Built-in specs: "exec" (executive summary), "engineering" (technical deep dive), "onboarding" (learning guide). Use render_specs to list all available specs.',
        ),
      workspace: z
        .string()
        .optional()
        .describe(
          `Workspace context. Defaults to "${config.defaultWorkspace}".`,
        ),
    },
    async ({ source, spec, workspace }) => {
      try {
        const result = await engine.renderContext({
          source,
          spec,
          workspace: workspace ?? config.defaultWorkspace,
        });
        return {
          content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
        };
      } catch (err) {
        if (err instanceof EngineError) {
          if (err.status === 404) {
            throw new McpError(
              ErrorCode.InvalidParams,
              `Render spec "${spec}" not found. Use render_specs to list available specs.`,
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

  server.tool(
    "render_specs",
    "List all available render specs for a workspace. Each spec defines an audience-specific rendering target (e.g. exec summary, engineering doc, onboarding guide) with constraints and HTML output hints.",
    {
      workspace: z
        .string()
        .optional()
        .describe(
          `Workspace to list specs for. Defaults to "${config.defaultWorkspace}".`,
        ),
    },
    async ({ workspace }) => {
      try {
        const result = await engine.renderSpecs({
          workspace: workspace ?? config.defaultWorkspace,
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
