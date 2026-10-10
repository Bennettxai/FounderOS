/** Signal Theory types — matches backend signal package */

export type OsaMode = "BUILD" | "ASSIST" | "ANALYZE" | "EXECUTE" | "MAINTAIN";
export type SignalGenre = "DIRECT" | "INFORM" | "COMMIT" | "DECIDE" | "EXPRESS";

export interface SignalEnvelope {
  mode: OsaMode;
  genre: SignalGenre;
  weight: number;
  docType: string;
  confidence: number;
}

export interface SignalHealthResponse {
  status: "healthy" | "degraded" | "unknown";
  classification: {
    active: boolean;
    type: string;
    latency: string;
  };
  metrics: {
    action_completion: boolean;
    re_encoding: boolean;
    signal_bounce: boolean;
    genre_recognition: boolean;
    feedback_closure: boolean;
    time_to_decide: boolean;
  };
  feedback_loop: {
    homeostatic_loop: boolean;
    double_loop: boolean;
    algedonic_channel: boolean;
    interval: string;
  };
}

export interface GenreDistribution {
  genre: SignalGenre;
  count: number;
  percentage: number;
}

export interface ModeDistribution {
  mode: OsaMode;
  count: number;
  percentage: number;
}

/** Genre descriptions for UI display */
export const GENRE_LABELS: Record<SignalGenre, string> = {
  DIRECT: "Action",
  INFORM: "Question",
  COMMIT: "Commitment",
  DECIDE: "Decision",
  EXPRESS: "Feedback",
};

export const GENRE_DESCRIPTIONS: Record<SignalGenre, string> = {
  DIRECT: "User wants something created or done",
  INFORM: "User is asking a question or seeking information",
  COMMIT: "User is making a commitment or planning",
  DECIDE: "User needs help choosing between options",
  EXPRESS: "User is expressing a feeling or giving feedback",
};

export const MODE_LABELS: Record<OsaMode, string> = {
  BUILD: "Build",
  ASSIST: "Assist",
  ANALYZE: "Analyze",
  EXECUTE: "Execute",
  MAINTAIN: "Maintain",
};
