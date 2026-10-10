/**
 * onboardingTypes.ts
 * Shared types, constants, and static data for the conversational onboarding flow.
 */
import { getLegacyApiBaseUrl } from "$lib/config/runtime";

export type OnboardingPhase =
  | "loading"
  | "intro"
  | "conversation"
  | "integrations";

export type QuestionType =
  | "company_name" // chat input
  | "business_type" // chips
  | "team_size" // chips (skip if freelance)
  | "role" // chat input
  | "challenge" // chat input
  | "integrations"
  | "complete";

export interface Message {
  id: string;
  sender: "agent" | "user";
  content: string;
  timestamp: Date;
}

export interface ExtractedData {
  workspaceName?: string;
  businessType?: string;
  teamSize?: string;
  role?: string;
  challenge?: string;
  integrations?: string[];
}

export interface ChipOption {
  id: string;
  label: string;
  icon?: string;
}

export interface HistoryEntry {
  question: QuestionType;
  data: ExtractedData;
  agentMessage: string;
}

export interface IntegrationDef {
  id: string;
  name: string;
  icon: string;
}

// ─── Static data ─────────────────────────────────────────────────────────────

export const businessTypeOptions: ChipOption[] = [
  { id: "agency", label: "Agency" },
  { id: "startup", label: "Startup" },
  { id: "freelance", label: "Freelance" },
  { id: "ecommerce", label: "E-commerce" },
  { id: "consulting", label: "Consulting" },
  { id: "other", label: "Other" },
];

export const teamSizeOptions: ChipOption[] = [
  { id: "solo", label: "Just me" },
  { id: "2-5", label: "2-5" },
  { id: "6-15", label: "6-15" },
  { id: "16-50", label: "16-50" },
  { id: "50+", label: "50+" },
];

export const questions: Record<
  QuestionType,
  { message: string; inputType: "chat" | "chips"; chips?: ChipOption[] }
> = {
  company_name: {
    message: "What's your company called?",
    inputType: "chat",
  },
  business_type: {
    message: "What kind of work do you do?",
    inputType: "chips",
    chips: businessTypeOptions,
  },
  team_size: {
    message: "How big is your team?",
    inputType: "chips",
    chips: teamSizeOptions,
  },
  role: {
    message: "What's your role?",
    inputType: "chat",
  },
  challenge: {
    message: "What's the biggest challenge you're hoping to solve?",
    inputType: "chat",
  },
  integrations: {
    message: "Perfect! Let's connect your favorite tools.",
    inputType: "chat",
  },
  complete: {
    message: "Perfect! Let's connect your favorite tools.",
    inputType: "chat",
  },
};

export const integrationDefs: IntegrationDef[] = [
  {
    id: "google",
    name: "Google Workspace",
    icon: '<svg viewBox="0 0 24 24"><path fill="#4285F4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"/><path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"/><path fill="#FBBC05" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"/><path fill="#EA4335" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"/></svg>',
  },
  {
    id: "microsoft",
    name: "Microsoft 365",
    icon: '<svg viewBox="0 0 24 24"><rect x="1" y="1" width="10" height="10" fill="#F25022"/><rect x="13" y="1" width="10" height="10" fill="#7FBA00"/><rect x="1" y="13" width="10" height="10" fill="#00A4EF"/><rect x="13" y="13" width="10" height="10" fill="#FFB900"/></svg>',
  },
  {
    id: "slack",
    name: "Slack",
    icon: '<svg viewBox="0 0 24 24"><path fill="#E01E5A" d="M5.042 15.165a2.528 2.528 0 0 1-2.52 2.523A2.528 2.528 0 0 1 0 15.165a2.527 2.527 0 0 1 2.522-2.52h2.52v2.52z"/><path fill="#E01E5A" d="M6.313 15.165a2.527 2.527 0 0 1 2.521-2.52 2.527 2.527 0 0 1 2.521 2.52v6.313A2.528 2.528 0 0 1 8.834 24a2.528 2.528 0 0 1-2.521-2.522v-6.313z"/><path fill="#36C5F0" d="M8.834 5.042a2.528 2.528 0 0 1-2.521-2.52A2.528 2.528 0 0 1 8.834 0a2.528 2.528 0 0 1 2.521 2.522v2.52H8.834z"/><path fill="#36C5F0" d="M8.834 6.313a2.528 2.528 0 0 1 2.521 2.521 2.528 2.528 0 0 1-2.521 2.521H2.522A2.528 2.528 0 0 1 0 8.834a2.528 2.528 0 0 1 2.522-2.521h6.312z"/><path fill="#2EB67D" d="M18.958 8.834a2.528 2.528 0 0 1 2.52-2.521A2.528 2.528 0 0 1 24 8.834a2.528 2.528 0 0 1-2.522 2.521h-2.52V8.834z"/><path fill="#2EB67D" d="M17.687 8.834a2.528 2.528 0 0 1-2.521 2.521 2.528 2.528 0 0 1-2.521-2.521V2.522A2.528 2.528 0 0 1 15.166 0a2.528 2.528 0 0 1 2.521 2.522v6.312z"/><path fill="#ECB22E" d="M15.166 18.958a2.528 2.528 0 0 1 2.521 2.52A2.528 2.528 0 0 1 15.166 24a2.528 2.528 0 0 1-2.521-2.522v-2.52h2.521z"/><path fill="#ECB22E" d="M15.166 17.687a2.528 2.528 0 0 1-2.521-2.521 2.528 2.528 0 0 1 2.521-2.521h6.312A2.528 2.528 0 0 1 24 15.166a2.528 2.528 0 0 1-2.522 2.521h-6.312z"/></svg>',
  },
  {
    id: "notion",
    name: "Notion",
    icon: '<svg viewBox="0 0 24 24"><path fill="currentColor" d="M4.459 4.208c.746.606 1.026.56 2.428.466l13.215-.793c.28 0 .047-.28-.046-.326L17.86 2.02c-.42-.327-.98-.7-2.053-.607L3.01 2.73c-.466.046-.56.28-.374.466zm.793 3.08v13.904c0 .747.373 1.027 1.213.98l14.523-.84c.84-.047.933-.56.933-1.167V6.354c0-.606-.233-.933-.746-.886l-15.177.887c-.56.046-.746.326-.746.933zm14.337.745c.093.42 0 .84-.42.888l-.7.14v10.264c-.608.327-1.168.514-1.635.514-.746 0-.932-.234-1.494-.933l-4.577-7.186v6.952L12.21 19s0 .84-1.168.84l-3.222.186c-.093-.186 0-.653.327-.746l.84-.233V9.854L7.822 9.76c-.094-.42.14-1.026.793-1.073l3.456-.233 4.764 7.279v-6.44l-1.215-.14c-.093-.514.28-.886.747-.933zM2.571 1.027l13.636-1c1.634-.14 2.053-.047 3.082.7l4.249 2.986c.7.513.933.653.933 1.213v16.378c0 1.026-.373 1.634-1.68 1.726l-15.458.934c-.98.047-1.447-.093-1.96-.747l-3.128-4.053c-.56-.747-.793-1.306-.793-1.96V2.667c0-.839.374-1.54 1.119-1.64z"/></svg>',
  },
  {
    id: "linear",
    name: "Linear",
    icon: '<svg viewBox="0 0 24 24"><path fill="#5E6AD2" d="M3.357 3.357a12.166 12.166 0 0 0-1.473 2.195l16.564 16.564a12.166 12.166 0 0 0 2.195-1.473L3.357 3.357zm-1.947 3.49A12.147 12.147 0 0 0 .69 9.75L14.25 23.31a12.147 12.147 0 0 0 2.903-.72L1.41 6.847zm-.655 4.18a12.244 12.244 0 0 0-.065 1.973l7.31 7.31c.656.03 1.317-.003 1.973-.065L.755 11.027zm.025 3.528 5.618 5.618a12.198 12.198 0 0 1-5.618-5.618zm19.863-3.198L14.027.755a12.244 12.244 0 0 0-1.973.065l9.218 9.218c.062-.656.095-1.317.065-1.973zm.047-1.555a12.147 12.147 0 0 0-.72-2.903L6.847 1.41a12.147 12.147 0 0 0-2.903.72L19.688 17.87zm1.06 3.485a12.166 12.166 0 0 0 1.473-2.195L5.959 2.528a12.166 12.166 0 0 0-2.195 1.473l17.286 17.286z"/></svg>',
  },
  {
    id: "hubspot",
    name: "HubSpot",
    icon: '<svg viewBox="0 0 24 24"><path fill="#FF7A59" d="M18.164 7.93V5.084a2.198 2.198 0 0 0 1.267-1.984v-.066A2.198 2.198 0 0 0 17.233.836h-.066a2.198 2.198 0 0 0-2.198 2.198v.066c0 .867.503 1.617 1.232 1.974v2.862a6.175 6.175 0 0 0-2.868 1.165l-7.62-5.937a2.636 2.636 0 1 0-1.168 1.49l7.322 5.705a6.205 6.205 0 0 0 .035 6.103l-2.22 2.22a2.092 2.092 0 1 0 1.131 1.131l2.234-2.234a6.2 6.2 0 1 0 5.087-9.649zm-1.031 9.544a3.343 3.343 0 1 1 0-6.687 3.343 3.343 0 0 1 0 6.687z"/></svg>',
  },
  {
    id: "airtable",
    name: "Airtable",
    icon: '<svg viewBox="0 0 24 24"><path fill="#FCB400" d="M11.992 2 2 6.462v11.078L11.992 22l9.992-4.462V6.462L11.992 2z"/><path fill="#18BFFF" d="M12 2.016 2.016 6.462v.082l9.976 4.362 9.992-4.362-.008-.082L12 2.016z"/><path fill="#F82B60" d="m2 6.544-.008 11.002L11.984 22V10.924L2 6.544z"/><path fill="#666" d="M22 6.544 12.016 22V10.924L22 6.544z" opacity=".25"/></svg>',
  },
  {
    id: "clickup",
    name: "ClickUp",
    icon: '<svg viewBox="0 0 24 24"><path fill="#7B68EE" d="m3.986 14.446 2.824-2.173a5.568 5.568 0 0 0 5.19 3.452 5.568 5.568 0 0 0 5.19-3.452l2.824 2.173A9.141 9.141 0 0 1 12 19.725a9.141 9.141 0 0 1-8.014-5.28z"/><path fill="#49CCF9" d="m4 9.247 2.778 2.135 5.222-4.28 5.222 4.28L20 9.247 12 3 4 9.247z"/></svg>',
  },
  {
    id: "fathom",
    name: "Fathom",
    icon: '<svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="10" fill="#7C3AED"/><path fill="white" d="M8 12a4 4 0 0 1 8 0v4H8v-4z"/></svg>',
  },
];

// ─── Recommendation logic ─────────────────────────────────────────────────────

export function computeRecommendedIntegrations(
  extractedData: ExtractedData,
): string[] {
  const challenge = extractedData.challenge?.toLowerCase() || "";
  const businessType = extractedData.businessType || "";

  if (
    challenge.includes("organiz") ||
    challenge.includes("chaos") ||
    challenge.includes("mess")
  ) {
    return ["notion", "google", "linear"];
  }
  if (
    challenge.includes("scale") ||
    challenge.includes("grow") ||
    challenge.includes("automat")
  ) {
    return ["linear", "slack", "airtable"];
  }
  if (
    challenge.includes("client") ||
    challenge.includes("customer") ||
    challenge.includes("crm")
  ) {
    return ["hubspot", "slack", "google"];
  }
  if (
    challenge.includes("team") ||
    challenge.includes("collaborat") ||
    challenge.includes("communic")
  ) {
    return ["slack", "notion", "linear"];
  }
  if (
    challenge.includes("time") ||
    challenge.includes("busy") ||
    challenge.includes("meeting")
  ) {
    return ["google", "fathom", "slack"];
  }

  // Default by business type
  if (businessType === "agency" || businessType === "consulting")
    return ["hubspot", "slack", "notion"];
  if (businessType === "startup") return ["linear", "slack", "notion"];
  if (businessType === "freelance") return ["google", "notion", "fathom"];
  return ["google", "slack", "notion"];
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

export function generateId(): string {
  return Math.random().toString(36).substring(2, 9);
}

export function getApiBase(): string {
  return getLegacyApiBaseUrl();
}
