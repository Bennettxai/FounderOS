/**
 * Vercel AI SDK (v6) tool wrappers for the Optimal Engine.
 *
 * Peer dependency: `ai` >= 6.0.0, `zod` >= 3.0.0
 *
 * @example
 * import { OptimalEngine } from "@optimal-engine/client"
 * import { optimalEngineTools } from "@optimal-engine/client/adapters/ai-sdk"
 * import { generateText } from "ai"
 *
 * const engine = new OptimalEngine({ baseUrl: "http://localhost:4200", workspace: "my-ws" })
 * const tools = optimalEngineTools(engine)
 *
 * const { text } = await generateText({
 *   model: yourModel,
 *   tools,
 *   prompt: "What did we decide about the auth flow last week?",
 * })
 */

import { tool } from "ai";
import { z } from "zod";
import type { OptimalEngine } from "../client.js";
import type { MemoryId } from "../types.js";

export interface OptimalEngineToolsOptions {
  /** Override the default workspace for all tool calls. */
  workspace?: string;
}

/**
 * Returns a record of Vercel AI SDK `tool()` definitions that give an LLM
 * access to the Optimal Engine's memory and retrieval capabilities.
 *
 * Pass the returned object directly to `generateText`, `streamText`, etc.
 */
export function optimalEngineTools(
  client: OptimalEngine,
  opts: OptimalEngineToolsOptions = {},
) {
  const ws = opts.workspace;

  return {
    /**
     * Ask the second brain a question. Curated wiki first, hybrid search
     * second. Returns ACL-scoped, audience-shaped, bandwidth-matched envelope
     * with hot citations.
     */
    askEngine: tool({
      description:
        "Ask the second brain a question. Curated wiki first, hybrid search second. Returns ACL-scoped, audience-shaped, bandwidth-matched envelope with hot citations.",
      inputSchema: z.object({
        query: z.string().describe("The question to ask the engine."),
        audience: z
          .string()
          .optional()
          .describe("Audience scope — e.g. 'public' or 'internal'."),
        format: z
          .string()
          .optional()
          .describe("Desired response format hint, e.g. 'markdown'."),
        bandwidth: z
          .enum(["l0", "l1", "full"])
          .optional()
          .describe(
            "Response depth: l0 = headline only, l1 = summary, full = complete.",
          ),
      }),
      execute: async (params, _options) => {
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

    /**
     * Search memory using hybrid semantic + keyword search.
     */
    searchMemory: tool({
      description:
        "Search long-term memory using hybrid semantic and keyword search. Returns ranked hits with content excerpts.",
      inputSchema: z.object({
        query: z.string().describe("Search query."),
        limit: z
          .number()
          .int()
          .positive()
          .optional()
          .describe("Maximum number of results to return."),
      }),
      execute: async (params, _options) => {
        return client.search(params.query, {
          ...(ws !== undefined ? { workspace: ws } : {}),
          ...(params.limit !== undefined ? { limit: params.limit } : {}),
        });
      },
    }),

    /**
     * Grep memory with structured filters.
     */
    grepMemory: tool({
      description:
        "Grep long-term memory with structured intent, scale, and modality filters. More precise than search when you know the shape of what you want.",
      inputSchema: z.object({
        query: z.string().describe("Pattern or phrase to grep for."),
        intent: z
          .string()
          .optional()
          .describe("Query intent hint, e.g. 'decision', 'action', 'fact'."),
        scale: z.string().optional().describe("Scale filter."),
        modality: z
          .string()
          .optional()
          .describe("Modality filter, e.g. 'text', 'code'."),
        limit: z
          .number()
          .int()
          .positive()
          .optional()
          .describe("Maximum number of results."),
      }),
      execute: async (params, _options) => {
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

    /**
     * Recover past actions, decisions, or commitments by topic.
     */
    recallActions: tool({
      description:
        "Recover past actions / decisions / commitments by topic and optionally actor + since.",
      inputSchema: z.object({
        topic: z
          .string()
          .describe("Topic or subject area to recall actions for."),
        actor: z
          .string()
          .optional()
          .describe("Filter by the person or system that took the action."),
        since: z
          .string()
          .optional()
          .describe(
            "ISO 8601 timestamp — only return actions after this point in time.",
          ),
      }),
      execute: async (params, _options) => {
        return client.recall.actions({
          topic: params.topic,
          ...(params.actor !== undefined ? { actor: params.actor } : {}),
          ...(params.since !== undefined ? { since: params.since } : {}),
          ...(ws !== undefined ? { workspace: ws } : {}),
        });
      },
    }),

    /**
     * Find who owns or is accountable for a topic.
     */
    recallWho: tool({
      description: "Find who owns / is accountable for a topic.",
      inputSchema: z.object({
        topic: z
          .string()
          .describe("Topic to look up ownership or accountability for."),
        role: z
          .string()
          .optional()
          .describe("Filter by role, e.g. 'owner', 'reviewer'."),
      }),
      execute: async (params, _options) => {
        return client.recall.who({
          topic: params.topic,
          ...(params.role !== undefined ? { role: params.role } : {}),
          ...(ws !== undefined ? { workspace: ws } : {}),
        });
      },
    }),

    /**
     * Add a memory to long-term storage.
     */
    addMemory: tool({
      description:
        "Add a fact, decision, or observation to long-term memory. Cited and integrity-gated — every memory ties to a source. Use when the user mentions information that should persist.",
      inputSchema: z.object({
        content: z
          .string()
          .describe("The content to store. Be explicit and self-contained."),
        audience: z
          .string()
          .optional()
          .describe("Audience scope — e.g. 'public' or 'internal'."),
        citationUri: z
          .string()
          .url()
          .optional()
          .describe("Source URI that this memory is derived from."),
        isStatic: z
          .boolean()
          .optional()
          .describe(
            "If true, treat as ground-truth static fact (not subject to decay).",
          ),
        metadata: z
          .record(z.string(), z.unknown())
          .optional()
          .describe("Arbitrary key-value metadata to attach."),
      }),
      execute: async (params, _options) => {
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

    /**
     * Soft-forget a memory by id.
     */
    forgetMemory: tool({
      description:
        "Soft-forget a memory by id. Audit trail preserved (reason recorded). Use when the user says to forget something.",
      inputSchema: z.object({
        id: z.string().describe("The memory id to forget."),
        reason: z
          .string()
          .optional()
          .describe("Why this memory should be forgotten."),
        forgetAfter: z
          .string()
          .optional()
          .describe(
            "ISO 8601 datetime — schedule forgetting for the future instead of immediately.",
          ),
      }),
      execute: async (params, _options) => {
        return client.memory.forget(params.id as MemoryId, {
          ...(params.reason !== undefined ? { reason: params.reason } : {}),
          ...(params.forgetAfter !== undefined
            ? { forgetAfter: params.forgetAfter }
            : {}),
        });
      },
    }),

    /**
     * Get the workspace profile.
     */
    getProfile: tool({
      description:
        "Get a 4-tier workspace profile: static (ground truth), dynamic (rolling), curated (wiki summary), activity (recent + top entities).",
      inputSchema: z.object({
        audience: z
          .string()
          .optional()
          .describe("Audience scope to filter profile by."),
        bandwidth: z
          .enum(["l0", "l1", "full"])
          .optional()
          .describe(
            "Profile depth: l0 = headline, l1 = summary, full = all tiers.",
          ),
        node: z
          .string()
          .optional()
          .describe("Anchor the profile around a specific graph node."),
      }),
      execute: async (params, _options) => {
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
  };
}

export type OptimalEngineTools = ReturnType<typeof optimalEngineTools>;
