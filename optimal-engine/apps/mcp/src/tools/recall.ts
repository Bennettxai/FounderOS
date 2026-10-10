import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { McpError, ErrorCode } from "@modelcontextprotocol/sdk/types.js";
import { z } from "zod";
import { engine, EngineError } from "../client.js";
import { config } from "../config.js";

const RECALL_ACTIONS = ["actions", "who", "when", "where", "owns"] as const;
type RecallAction = (typeof RECALL_ACTIONS)[number];

const ACTION_DESCRIPTIONS: Record<RecallAction, string> = {
  actions: "Past commitments and action items by topic.",
  who: "Ownership lookup — who is responsible for a topic or thing.",
  when: "Schedule lookup — when something happened or is planned.",
  where: "Location or file lookup — where something lives.",
  owns: "Open commitments by actor — what an actor currently owns.",
};

export function registerRecall(server: McpServer): void {
  server.tool(
    "recall",
    "Typed cued recall over a workspace. Use for ownership, commitments, timing, location/file lookup, and open actor responsibilities when a full synthesis answer is not needed.",
    {
      action: z
        .enum(RECALL_ACTIONS)
        .describe(
          [
            "Which recall verb to execute:",
            ...RECALL_ACTIONS.map(
              (a) => `  - "${a}": ${ACTION_DESCRIPTIONS[a]}`,
            ),
          ].join("\n"),
        ),
      workspace: z
        .string()
        .optional()
        .describe(
          `Workspace to recall from. Defaults to "${config.defaultWorkspace}".`,
        ),
      topic: z
        .string()
        .optional()
        .describe("Topic or subject to recall (used by 'actions' and 'who')."),
      event: z
        .string()
        .optional()
        .describe(
          "Event name or description to look up timing for (used by 'when').",
        ),
      thing: z
        .string()
        .optional()
        .describe(
          "Thing, file, or concept whose location to find (used by 'where').",
        ),
      actor: z
        .string()
        .optional()
        .describe(
          "Person or team whose open commitments to look up (used by 'owns').",
        ),
    },
    async ({ action, workspace, topic, event, thing, actor }) => {
      // Validate that the right param is present for the chosen action
      if (action === "when" && !event && !topic) {
        throw new McpError(
          ErrorCode.InvalidParams,
          `Recall action "when" requires "event" or "topic".`,
        );
      }
      if (action === "where" && !thing && !topic) {
        throw new McpError(
          ErrorCode.InvalidParams,
          `Recall action "where" requires "thing" or "topic".`,
        );
      }
      if (action === "owns" && !actor) {
        throw new McpError(
          ErrorCode.InvalidParams,
          `Recall action "owns" requires "actor".`,
        );
      }

      try {
        const result = await engine.recall(action, {
          workspace: workspace ?? config.defaultWorkspace,
          topic,
          event,
          thing,
          actor,
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
