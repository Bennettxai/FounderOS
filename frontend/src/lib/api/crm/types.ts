// CRM Types

// ============================================================================
// Companies
// ============================================================================

export interface Company {
  id: string;
  user_id: string;
  name: string;
  legal_name?: string;
  industry?: string;
  company_size?: string;
  website?: string;
  email?: string;
  phone?: string;
  address_line1?: string;
  address_line2?: string;
  city?: string;
  state?: string;
  postal_code?: string;
  country?: string;
  annual_revenue?: number;
  currency?: string;
  linkedin_url?: string;
  twitter_handle?: string;
  owner_id?: string;
  lifecycle_stage?: string;
  lead_source?: string;
  health_score?: number;
  engagement_score?: number;
  logo_url?: string;
  custom_fields?: Record<string, unknown>;
  metadata?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface CreateCompanyData {
  name: string;
  legal_name?: string;
  industry?: string;
  company_size?: string;
  website?: string;
  email?: string;
  phone?: string;
  address_line1?: string;
  address_line2?: string;
  city?: string;
  state?: string;
  postal_code?: string;
  country?: string;
  annual_revenue?: number;
  currency?: string;
  tax_id?: string;
  linkedin_url?: string;
  twitter_handle?: string;
  owner_id?: string;
  lifecycle_stage?: string;
  lead_source?: string;
  logo_url?: string;
  custom_fields?: Record<string, unknown>;
  metadata?: Record<string, unknown>;
}

export interface UpdateCompanyData extends Partial<CreateCompanyData> {
  name: string;
}

// ============================================================================
// Pipelines
// ============================================================================

export interface Pipeline {
  id: string;
  user_id: string;
  name: string;
  description?: string;
  pipeline_type?: "sales" | "hiring" | "projects" | "custom";
  currency?: string;
  is_default?: boolean;
  is_active?: boolean;
  color?: string;
  icon?: string;
  created_at: string;
  updated_at: string;
}

export interface CreatePipelineData {
  name: string;
  description?: string;
  pipeline_type?: string;
  currency?: string;
  is_default?: boolean;
  color?: string;
  icon?: string;
}

export interface UpdatePipelineData {
  name: string;
  description?: string;
  currency?: string;
  color?: string;
  icon?: string;
  is_active?: boolean;
}

// ============================================================================
// Pipeline Stages
// ============================================================================

export interface PipelineStage {
  id: string;
  pipeline_id: string;
  name: string;
  description?: string;
  position: number;
  probability?: number;
  stage_type?: "open" | "won" | "lost";
  rotting_days?: number;
  color?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateStageData {
  name: string;
  description?: string;
  position: number;
  probability?: number;
  stage_type?: string;
  rotting_days?: number;
  color?: string;
}

export interface UpdateStageData {
  name: string;
  description?: string;
  probability?: number;
  rotting_days?: number;
  color?: string;
}

// ============================================================================
// Deals
// ============================================================================

export interface Deal {
  id: string;
  user_id: string;
  pipeline_id: string;
  pipeline_name?: string;
  stage_id: string;
  stage_name?: string;
  name: string;
  description?: string;
  amount?: number;
  currency?: string;
  probability?: number;
  expected_close_date?: string;
  actual_close_date?: string;
  owner_id?: string;
  company_id?: string;
  company_name?: string;
  primary_contact_id?: string;
  status?: "open" | "won" | "lost";
  lost_reason?: string;
  priority?: "low" | "medium" | "high" | "urgent";
  lead_source?: string;
  deal_score?: number;
  custom_fields?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface CreateDealData {
  pipeline_id: string;
  stage_id: string;
  name: string;
  description?: string;
  amount?: number;
  currency?: string;
  probability?: number;
  expected_close_date?: string;
  owner_id?: string;
  company_id?: string;
  primary_contact_id?: string;
  status?: string;
  priority?: string;
  lead_source?: string;
  custom_fields?: Record<string, unknown>;
}

export interface UpdateDealData {
  name: string;
  description?: string;
  amount?: number;
  probability?: number;
  expected_close_date?: string;
  owner_id?: string;
  company_id?: string;
  primary_contact_id?: string;
  priority?: string;
  custom_fields?: Record<string, unknown>;
}

export interface DealStats {
  total_deals: number;
  open_deals: number;
  won_deals: number;
  lost_deals: number;
  open_value: number;
  won_value: number;
  lost_value: number;
  // Extended stats returned by some backend versions
  total_value?: number;
  avg_deal_value?: number;
  avg_close_days?: number;
  win_rate?: number;
}

// ============================================================================
// Activities
// ============================================================================

export type ActivityType =
  | "call"
  | "email"
  | "meeting"
  | "demo"
  | "note"
  | "task"
  | "lunch"
  | "deadline"
  | "other";

export interface CRMActivity {
  id: string;
  user_id: string;
  activity_type: ActivityType;
  subject: string;
  description?: string;
  outcome?: string;
  deal_id?: string;
  company_id?: string;
  contact_id?: string;
  participants?: string[];
  activity_date: string;
  duration_minutes?: number;
  call_direction?: "inbound" | "outbound";
  call_disposition?: string;
  call_recording_url?: string;
  email_direction?: "inbound" | "outbound";
  email_message_id?: string;
  meeting_location?: string;
  meeting_url?: string;
  owner_id?: string;
  is_completed?: boolean;
  completed_by?: string;
  completed_at?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateActivityData {
  activity_type: string;
  subject: string;
  description?: string;
  outcome?: string;
  deal_id?: string;
  company_id?: string;
  contact_id?: string;
  participants?: string[];
  activity_date: string;
  duration_minutes?: number;
  call_direction?: string;
  call_disposition?: string;
  call_recording_url?: string;
  email_direction?: string;
  email_message_id?: string;
  meeting_location?: string;
  meeting_url?: string;
  owner_id?: string;
  is_completed?: boolean;
}

// ============================================================================
// Contact-Company Relations
// ============================================================================

export interface ContactCompanyRelation {
  id: string;
  contact_id: string;
  company_id: string;
  job_title?: string;
  department?: string;
  role_type?: string;
  is_primary?: boolean;
  start_date?: string;
  end_date?: string;
  created_at: string;
  updated_at: string;
}

export interface LinkContactData {
  contact_id: string;
  job_title?: string;
  department?: string;
  role_type?: string;
  is_primary?: boolean;
}

// ============================================================================
// CRM Contact (company-contact relation row returned by ListCompanyContacts)
// ============================================================================

export interface CRMContact {
  id: string;
  contact_id: string;
  company_id: string;
  job_title?: string;
  department?: string;
  role_type?: string;
  is_primary?: boolean;
  is_active?: boolean;
  started_at?: string;
  ended_at?: string;
  created_at: string;
  updated_at: string;
  contact_name: string;
  contact_email?: string;
}

// ============================================================================
// API Response Types
// ============================================================================

export interface CompaniesResponse {
  companies: Company[];
  count: number;
}

export interface PipelinesResponse {
  pipelines: Pipeline[];
  count: number;
}

export interface StagesResponse {
  stages: PipelineStage[];
  count: number;
}

export interface DealsResponse {
  deals: Deal[];
  count: number;
}

export interface ActivitiesResponse {
  activities: CRMActivity[];
  count: number;
}

export interface ContactsResponse {
  contacts: CRMContact[];
  count: number;
}
