// Workspace API types

export interface Workspace {
  id: string;
  name: string;
  slug: string;
  description: string | null;
  logo_url: string | null;
  plan_type: "free" | "starter" | "professional" | "enterprise";
  max_members: number;
  max_projects: number;
  max_storage_gb: number;
  settings: Record<string, unknown>;
  owner_id: string;
  created_at: string;
  updated_at: string;
}

export interface WorkspaceRole {
  id: string;
  workspace_id: string;
  name: string;
  display_name: string;
  description: string | null;
  color: string | null;
  icon: string | null;
  permissions: Record<string, Record<string, boolean | string>>;
  is_system: boolean;
  is_default: boolean;
  hierarchy_level: number;
  created_at: string;
  updated_at: string;
}

export interface WorkspaceMember {
  id: string;
  workspace_id: string;
  user_id: string;
  role: string;
  status: "active" | "invited" | "suspended" | "left";
  invited_by: string | null;
  invited_at: string | null;
  joined_at: string | null;
  custom_permissions: Record<string, Record<string, boolean | string>> | null;
  created_at: string;
  updated_at: string;
}

export interface UserWorkspaceProfile {
  id: string;
  workspace_id: string;
  user_id: string;
  display_name: string | null;
  title: string | null;
  department: string | null;
  avatar_url: string | null;
  work_email: string | null;
  phone: string | null;
  timezone: string | null;
  working_hours: Record<string, unknown> | null;
  notification_preferences: Record<string, unknown> | null;
  preferred_output_style: string | null;
  communication_preferences: Record<string, unknown> | null;
  expertise_areas: string[] | null;
  created_at: string;
  updated_at: string;
}

export interface CreateWorkspaceData {
  name: string;
  slug?: string;
  description?: string;
  plan_type?: "free" | "starter" | "professional" | "enterprise";
  organization_id?: string;
}

export interface UpdateWorkspaceData {
  name?: string;
  description?: string;
  logo_url?: string;
  settings?: Record<string, unknown>;
}

export interface UserRoleContext {
  user_id: string;
  workspace_id: string;
  role_name: string;
  role_display_name: string;
  hierarchy_level: number;
  permissions: Record<string, Record<string, boolean | string>>;
  title: string | null;
  department: string | null;
  expertise_areas: string[] | null;
}

export interface WorkspaceInvite {
  id: string;
  workspace_id: string;
  email: string;
  role: string;
  invited_by: string;
  status: "pending" | "accepted" | "expired" | "revoked";
  token: string;
  expires_at: string;
  created_at: string;
}

export interface CreateInviteData {
  email: string;
  role: string;
}

export interface UpdateMemberRoleData {
  role: string;
  custom_permissions?: Record<string, Record<string, boolean | string>>;
}

export interface CreateRoleData {
  name: string;
  display_name: string;
  description?: string;
  color?: string;
  icon?: string;
  permissions: Record<string, Record<string, boolean | string>>;
}

export interface UpdateRoleData {
  display_name?: string;
  description?: string;
  color?: string;
  icon?: string;
  permissions?: Record<string, Record<string, boolean | string>>;
}
