// Generic relationship-management field pack for the public BusinessOS edition.
// Workspaces can replace these options with their own ontology and module fields.

export type Option = { id: string; label: string };

export type PipelineStage =
  "lead" | "contacted" | "qualified" | "proposal" | "won" | "lost" | "nurture";

export const PIPELINE_STAGES: { id: PipelineStage; label: string }[] = [
  { id: "lead", label: "Lead" },
  { id: "contacted", label: "Contacted" },
  { id: "qualified", label: "Qualified" },
  { id: "proposal", label: "Proposal" },
  { id: "won", label: "Won" },
  { id: "lost", label: "Lost" },
  { id: "nurture", label: "Nurture" },
];

export const DEFAULT_STAGE: PipelineStage = "lead";
export const AGENCY_TYPES: Option[] = [];
export const PHYSICAL_OFFICE_STATUS: Option[] = [];
export const OUTREACH_STATUS: Option[] = [
  { id: "not_contacted", label: "Not contacted" },
  { id: "contacted", label: "Contacted" },
  { id: "replied", label: "Replied" },
  { id: "no_response", label: "No response" },
  { id: "follow_up_later", label: "Follow up later" },
];
export const MEETING_PREFERENCE: Option[] = [];
export const PAIN_CATEGORY: Option[] = [];
export const OFFER_FIT: Option[] = [];
export const FIT_SCORES: { id: number; label: string }[] = [
  { id: 1, label: "1" },
  { id: 2, label: "2" },
  { id: 3, label: "3" },
  { id: 4, label: "4" },
  { id: 5, label: "5" },
];
export const TOOL_STACK_FIELDS: { key: string; label: string }[] = [];

export const CF = {
  agencyType: "agency_type",
  pipelineStage: "pipeline_stage",
  physicalOfficeStatus: "physical_office_status",
  outreachStatus: "outreach_status",
  meetingPreference: "meeting_preference",
  painCategory: "pain_category",
  offerFit: "offer_fit",
  fitScore: "fit_score",
  googleMapsUrl: "google_maps_url",
  linkedinUrl: "linkedin_url",
  ownerName: "owner_name",
  operatorName: "operator_name",
  leadOwner: "lead_owner",
  nextStepDate: "next_step_date",
  toolStack: "tool_stack",
  whoTheyServe: "who_they_serve",
  whyTheyCare: "why_they_care",
  proofOfActivity: "proof_of_activity",
  likelyPain: "likely_pain",
  nextAction: "next_action",
} as const;

export function optionLabel(
  opts: Option[],
  id: string | undefined | null,
): string {
  if (!id) return "";
  return opts.find((option) => option.id === id)?.label ?? id;
}

export function stageLabel(id: string | undefined | null): string {
  if (!id) return "";
  return PIPELINE_STAGES.find((stage) => stage.id === id)?.label ?? id;
}
