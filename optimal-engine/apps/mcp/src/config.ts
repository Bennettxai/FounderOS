/**
 * Runtime configuration — reads environment variables.
 * All tools import from here rather than touching process.env directly.
 */

export const config = {
  /** Base URL of the Optimal Engine HTTP API. */
  engineUrl: (
    process.env["OPTIMAL_ENGINE_URL"] ?? "http://localhost:4200"
  ).replace(/\/$/, ""),

  /** Default workspace used when callers omit the workspace param. */
  defaultWorkspace: process.env["OPTIMAL_WORKSPACE"] ?? "default",

  /** Optional API key sent as Bearer token on every request. */
  apiKey: process.env["OPTIMAL_API_KEY"] ?? null,
} as const;
