/**
 * Onboarding Analysis Store
 * Manages SSE streaming of AI-powered user analysis during onboarding
 *
 * Features:
 * - Real-time progress streaming via SSE
 * - Automatic insight updates as analysis completes
 * - Graceful error handling and fallback
 * - Persists results for analyzing screens 1-3
 */

import { writable, derived } from "svelte/store";
import {
  startAnalysis,
  streamAnalysisProgress,
  getAnalysisProgress,
  type AnalysisStreamEvent,
  type AnalysisStatus,
} from "$lib/api/osa-onboarding";

export interface OnboardingAnalysisState {
  // Analysis metadata
  analysisId: string | null;
  status: AnalysisStatus | null;

  // AI-generated insights (3 phrases for 3 screens)
  insights: string[];

  // Additional analysis data
  interests: string[];
  toolsUsed: string[];
  summary: string;

  // State flags
  isStreaming: boolean;
  isLoading: boolean;
  error: string | null;

  // Timestamps
  startedAt: number | null;
  completedAt: number | null;
}

const initialState: OnboardingAnalysisState = {
  analysisId: null,
  status: null,
  insights: [],
  interests: [],
  toolsUsed: [],
  summary: "",
  isStreaming: false,
  isLoading: false,
  error: null,
  startedAt: null,
  completedAt: null,
};

function createOnboardingAnalysisStore() {
  const { subscribe, set, update } =
    writable<OnboardingAnalysisState>(initialState);

  // Active stream reader (for cleanup)
  let streamReader: ReadableStreamDefaultReader<Uint8Array> | null = null;

  /**
   * Start analysis and begin streaming progress
   * Automatically opens SSE stream for real-time updates
   */
  async function start(userId: string, workspaceId: string, maxEmails = 50) {
    update((s) => ({
      ...s,
      isLoading: true,
      isStreaming: false,
      error: null,
      startedAt: Date.now(),
    }));

    try {
      // Step 1: Initiate analysis on backend
      const response = await startAnalysis(userId, workspaceId, maxEmails);

      update((s) => ({
        ...s,
        analysisId: response.analysis_id,
        status: response.status,
        isLoading: false,
        isStreaming: true,
      }));

      // Step 2: Start streaming progress
      await streamProgress(response.analysis_id);
    } catch (err) {
      const errorMessage =
        err instanceof Error ? err.message : "Analysis failed";
      console.error("Failed to start analysis:", err);

      update((s) => ({
        ...s,
        isLoading: false,
        isStreaming: false,
        error: errorMessage,
      }));
    }
  }

  /**
   * Stream analysis progress via SSE
   * Updates store in real-time as backend processes
   */
  async function streamProgress(analysisId: string) {
    try {
      const stream = await streamAnalysisProgress(analysisId);
      if (!stream) {
        throw new Error("No stream available");
      }

      streamReader = stream.getReader();
      const decoder = new TextDecoder();

      while (true) {
        const { done, value } = await streamReader.read();

        if (done) {
          if (import.meta.env.DEV) console.log("Analysis stream complete");
          update((s) => ({ ...s, isStreaming: false }));
          break;
        }

        // Decode and parse SSE event
        const chunk = decoder.decode(value, { stream: true });
        const lines = chunk.split("\n");

        for (const line of lines) {
          if (line.startsWith("data:")) {
            try {
              const eventData = JSON.parse(line.substring(5).trim());
              handleStreamEvent(eventData);
            } catch (parseErr) {
              console.warn("Failed to parse SSE event:", parseErr);
            }
          }
        }
      }
    } catch (err) {
      console.error("Stream error:", err);
      update((s) => ({
        ...s,
        isStreaming: false,
        error: err instanceof Error ? err.message : "Stream failed",
      }));

      // Fallback: Poll for final result
      pollForCompletion(analysisId);
    } finally {
      streamReader = null;
    }
  }

  /**
   * Handle incoming SSE event
   * Updates store state based on progress/done/error events
   */
  function handleStreamEvent(event: AnalysisStreamEvent) {
    if (event.type === "progress" && event.data) {
      update((s) => ({
        ...s,
        status: event.data?.status || s.status,
        insights: event.data?.insights || s.insights,
        interests: event.data?.interests || s.interests,
      }));
    } else if (event.type === "done" && event.data) {
      update((s) => ({
        ...s,
        status: "completed",
        insights: event.data?.insights || s.insights,
        interests: event.data?.interests || s.interests,
        isStreaming: false,
        completedAt: Date.now(),
      }));
    } else if (event.type === "error") {
      update((s) => ({
        ...s,
        status: "failed",
        error: event.content || "Analysis error",
        isStreaming: false,
        completedAt: Date.now(),
      }));
    }
  }

  /**
   * Fallback: Poll for completion if streaming fails
   * Checks every 2 seconds until analysis completes
   */
  async function pollForCompletion(analysisId: string) {
    const maxAttempts = 60; // 2 minutes max (60 * 2s)
    let attempts = 0;

    const pollInterval = setInterval(async () => {
      attempts++;

      if (attempts > maxAttempts) {
        clearInterval(pollInterval);
        update((s) => ({
          ...s,
          error: "Analysis timeout",
          status: "failed",
          completedAt: Date.now(),
        }));
        return;
      }

      try {
        const progress = await getAnalysisProgress(analysisId);

        update((s) => ({
          ...s,
          status: progress.status,
          insights: progress.insights || s.insights,
          interests: progress.interests || s.interests,
          toolsUsed: progress.tools_used || s.toolsUsed,
          summary: progress.summary || s.summary,
        }));

        if (progress.status === "completed" || progress.status === "failed") {
          clearInterval(pollInterval);
          update((s) => ({ ...s, completedAt: Date.now() }));
        }
      } catch (err) {
        console.error("Polling error:", err);
      }
    }, 2000);
  }

  /**
   * Cancel active stream and reset state
   */
  function cancel() {
    if (streamReader) {
      streamReader.cancel();
      streamReader = null;
    }

    update((s) => ({
      ...s,
      isStreaming: false,
      isLoading: false,
    }));
  }

  /**
   * Reset to initial state
   */
  function reset() {
    cancel();
    set(initialState);
  }

  /**
   * Poll for analysis by user_id (for OAuth flow)
   * This is used when analysis is triggered automatically in OAuth callback
   */
  async function pollByUserId(userId: string) {
    update((s) => ({
      ...s,
      isLoading: true,
      error: null,
      startedAt: Date.now(),
    }));

    const { getBackendUrl } = await import("$lib/api/base");
    const backendUrl = getBackendUrl();
    const maxAttempts = 60; // 2 minutes max
    let attempts = 0;

    const pollInterval = setInterval(async () => {
      attempts++;

      if (attempts > maxAttempts) {
        clearInterval(pollInterval);
        update((s) => ({
          ...s,
          isLoading: false,
          error: "Analysis timeout",
          status: "failed",
          completedAt: Date.now(),
        }));
        return;
      }

      try {
        const response = await fetch(
          `${backendUrl}/api/osa-onboarding/user-analysis/${userId}`,
        );
        if (!response.ok) throw new Error("Failed to fetch analysis");

        const data = await response.json();

        update((s) => ({
          ...s,
          status: data.status as AnalysisStatus,
          insights: data.insights || [],
          toolsUsed: data.tools || [],
          isLoading: data.status === "analyzing",
          completedAt:
            data.status === "completed" || data.status === "failed"
              ? Date.now()
              : null,
        }));

        if (data.status === "completed" || data.status === "failed") {
          clearInterval(pollInterval);
        }
      } catch (err) {
        console.error("Polling error:", err);
      }
    }, 2000); // Poll every 2 seconds
  }

  return {
    subscribe,
    start,
    cancel,
    reset,
    pollByUserId,
  };
}

export const onboardingAnalysis = createOnboardingAnalysisStore();

/**
 * Derived store: Get insights for each analyzing screen (1, 2, 3)
 */
export const analyzingInsights = derived(onboardingAnalysis, ($analysis) => ({
  message1: $analysis.insights[0] || "No-code builder energy",
  message2: $analysis.insights[1] || "Design tools are your playground",
  message3: $analysis.insights[2] || "AI-curious, testing new platforms",
  hasRealData: $analysis.insights.length >= 3,
}));

/**
 * Derived store: Analysis completion status
 */
export const analysisComplete = derived(
  onboardingAnalysis,
  ($analysis) => $analysis.status === "completed",
);

/**
 * Derived store: Analysis failed status
 */
export const analysisFailed = derived(
  onboardingAnalysis,
  ($analysis) => $analysis.status === "failed",
);

/**
 * Derived store: Analysis duration in milliseconds
 */
export const analysisDuration = derived(onboardingAnalysis, ($analysis) => {
  if (!$analysis.startedAt) return 0;
  const endTime = $analysis.completedAt || Date.now();
  return endTime - $analysis.startedAt;
});
