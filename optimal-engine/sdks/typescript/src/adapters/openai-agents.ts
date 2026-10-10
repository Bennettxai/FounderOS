/**
 * OpenAI Agents SDK adapter for the Optimal Engine.
 *
 * Peer dependency: `openai` >= 4.0.0, `zod` >= 3.0.0
 *
 * @example
 * import { OptimalEngine } from "@optimal-engine/client"
 * import { optimalEngineAgentTools } from "@optimal-engine/client/adapters/openai-agents"
 * import { Agent, run } from "openai/agents"
 *
 * const engine = new OptimalEngine({ baseUrl: "http://localhost:4200", workspace: "my-ws" })
 * const tools = optimalEngineAgentTools(engine)
 *
 * const agent = new Agent({ name: "MemoryAgent", tools })
 * const result = await run(agent, "What did we decide about the auth flow?")
 */

import { z } from "zod";
import type { MemoryId } from "../types.js";
import type { OptimalEngine } from "../client.js";

// ---------------------------------------------------------------------------
// Minimal structural type for an OpenAI Agents SDK tool so we don't require
// the full `openai` package at compile time in non-agent builds.
// The execute signature uses `never` for contravariant safety: a concrete
// tool implementation always accepts a narrower type than `unknown`.
// ---------------------------------------------------------------------------

export interface AgentTool<TInput> {
  name: string;
  description: string;
  parameters: z.ZodType<TInput>;
  execute: (input: TInput) => Promise<unknown>;
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type AnyAgentTool = AgentTool<any>;

export interface OptimalEngineAgentToolsOptions {
  /** Override the default workspace for all tool calls. */
  workspace?: string;
}

/**
 * Returns an array of OpenAI Agents SDK–compatible tool definitions that give
 * an agent access to the Optimal Engine's memory and retrieval capabilities.
 *
 * Pass the returned array to the `tools` option of `new Agent({ tools })`.
 */
export function optimalEngineAgentTools(
  client: OptimalEngine,
  opts: OptimalEngineAgentToolsOptions = {},
): AnyAgentTool[] {
  const ws = opts.workspace;

  function make<TInput>(def: AgentTool<TInput>): AgentTool<TInput> {
    return def;
  }

  return [
    make({
      name: "ask_engine",
      description:
        "Ask the second brain a question. Curated wiki first, hybrid search second. Returns ACL-scoped, audience-shaped, bandwidth-matched envelope with hot citations.",
      parameters: z.object({
        query: z.string().describe("The question to ask the engine."),
        audience: z.string().optional().describe("Audience scope."),
        format: z.string().optional().describe("Response format hint."),
        bandwidth: z
          .enum(["l0", "l1", "full"])
          .optional()
          .describe(
            "Response depth: l0 = headline, l1 = summary, full = complete.",
          ),
      }),
      execute: async (params) => {
        return client.ask(params.query, {
          ...(ws !== undefined ? { workspace: ws } : {}),
          ...(params.audience !== undefined
            ? { audience: params.audience }
            : {}),
          ...(params.format !== undefined ? { format: params.format } : {}),
          ...(params.bandwidth !== undefined
            ? { bandwidth: params.bandwidth }
            : {}),
        });
      },
    }),

    make({
      name: "search_memory",
      description:
        "Search long-term memory using hybrid semantic and keyword search.",
      parameters: z.object({
        query: z.string().describe("Search query."),
        limit: z.number().int().positive().optional().describe("Max results."),
      }),
      execute: async (params) => {
        return client.search(params.query, {
          ...(ws !== undefined ? { workspace: ws } : {}),
          ...(params.limit !== undefined ? { limit: params.limit } : {}),
        });
      },
    }),

    make({
      name: "grep_memory",
      description:
        "Grep long-term memory with structured intent, scale, and modality filters.",
      parameters: z.object({
        query: z.string().describe("Pattern or phrase to grep for."),
        intent: z.string().optional().describe("Query intent hint."),
        scale: z.string().optional().describe("Scale filter."),
        modality: z.string().optional().describe("Modality filter."),
        limit: z.number().int().positive().optional().describe("Max results."),
      }),
      execute: async (params) => {
        return client.grep(params.query, {
          ...(ws !== undefined ? { workspace: ws } : {}),
          ...(params.intent !== undefined ? { intent: params.intent } : {}),
          ...(params.scale !== undefined ? { scale: params.scale } : {}),
          ...(params.modality !== undefined
            ? { modality: params.modality }
            : {}),
          ...(params.limit !== undefined ? { limit: params.limit } : {}),
        });
      },
    }),

    make({
      name: "recall_actions",
      description:
        "Recover past actions / decisions / commitments by topic and optionally actor + since.",
      parameters: z.object({
        topic: z.string().describe("Topic to recall actions for."),
        actor: z.string().optional().describe("Filter by actor."),
        since: z
          .string()
          .optional()
          .describe("ISO 8601 timestamp lower bound."),
      }),
      execute: async (params) => {
        return client.recall.actions({
          topic: params.topic,
          ...(params.actor !== undefined ? { actor: params.actor } : {}),
          ...(params.since !== undefined ? { since: params.since } : {}),
          ...(ws !== undefined ? { workspace: ws } : {}),
        });
      },
    }),

    make({
      name: "recall_who",
      description: "Find who owns / is accountable for a topic.",
      parameters: z.object({
        topic: z.string().describe("Topic to look up."),
        role: z.string().optional().describe("Filter by role."),
      }),
      execute: async (params) => {
        return client.recall.who({
          topic: params.topic,
          ...(params.role !== undefined ? { role: params.role } : {}),
          ...(ws !== undefined ? { workspace: ws } : {}),
        });
      },
    }),

    make({
      name: "add_memory",
      description:
        "Add a fact, decision, or observation to long-term memory. Cited and integrity-gated — every memory ties to a source. Use when the user mentions information that should persist.",
      parameters: z.object({
        content: z.string().describe("Content to store."),
        audience: z.string().optional().describe("Audience scope."),
        citationUri: z.string().url().optional().describe("Source URI."),
        isStatic: z
          .boolean()
          .optional()
          .describe("Treat as static ground-truth fact."),
        metadata: z
          .record(z.string(), z.unknown())
          .optional()
          .describe("Arbitrary metadata."),
      }),
      execute: async (params) => {
        return client.memory.create({
          content: params.content,
          ...(ws !== undefined ? { workspace: ws } : {}),
          ...(params.audience !== undefined
            ? { audience: params.audience }
            : {}),
          ...(params.citationUri !== undefined
            ? { citationUri: params.citationUri }
            : {}),
          ...(params.isStatic !== undefined
            ? { isStatic: params.isStatic }
            : {}),
          ...(params.metadata !== undefined
            ? { metadata: params.metadata }
            : {}),
        });
      },
    }),

    make({
      name: "forget_memory",
      description:
        "Soft-forget a memory by id. Audit trail preserved (reason recorded). Use when the user says to forget something.",
      parameters: z.object({
        id: z.string().describe("Memory id to forget."),
        reason: z.string().optional().describe("Reason for forgetting."),
        forgetAfter: z
          .string()
          .optional()
          .describe("ISO 8601 datetime to schedule forgetting."),
      }),
      execute: async (params) => {
        return client.memory.forget(params.id as MemoryId, {
          ...(params.reason !== undefined ? { reason: params.reason } : {}),
          ...(params.forgetAfter !== undefined
            ? { forgetAfter: params.forgetAfter }
            : {}),
        });
      },
    }),

    make({
      name: "get_profile",
      description:
        "Get a 4-tier workspace profile: static (ground truth), dynamic (rolling), curated (wiki summary), activity (recent + top entities).",
      parameters: z.object({
        audience: z.string().optional().describe("Audience scope."),
        bandwidth: z
          .enum(["l0", "l1", "full"])
          .optional()
          .describe("Profile depth."),
        node: z.string().optional().describe("Anchor graph node."),
      }),
      execute: async (params) => {
        return client.profile({
          ...(ws !== undefined ? { workspace: ws } : {}),
          ...(params.audience !== undefined
            ? { audience: params.audience }
            : {}),
          ...(params.bandwidth !== undefined
            ? { bandwidth: params.bandwidth }
            : {}),
          ...(params.node !== undefined ? { node: params.node } : {}),
        });
      },
    }),
  ];
}

export type OptimalEngineAgentTools = ReturnType<
  typeof optimalEngineAgentTools
>;
