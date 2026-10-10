// Extracted API types for domain modules

export interface Project {
  id: string;
  name: string;
  description: string | null;
  status: 'active' | 'paused' | 'completed' | 'archived';
  priority: 'critical' | 'high' | 'medium' | 'low';
  client_name: string | null;
  project_type: string;
  project_metadata: Record<string, unknown> | null;
  due_date?: string | null;
  created_at: string;
  updated_at: string;
  notes: ProjectNote[];
}

export interface ProjectNote {
  id: string;
  content: string;
  created_at: string;
}

export interface ProjectFileLink {
  name: string;
  size: string;
  url?: string;
}

export interface CreateProjectData {
  name: string;
  description?: string;
  status?: 'active' | 'paused' | 'completed' | 'archived';
  priority?: 'critical' | 'high' | 'medium' | 'low';
  client_name?: string;
  project_type?: string;
  project_metadata?: Record<string, unknown>;
  due_date?: string | null;
}

// Project Member Types
export type ProjectRole = 'lead' | 'contributor' | 'reviewer' | 'viewer';

export type MemberStatus = 'active' | 'inactive' | 'removed';

export interface ProjectMember {
  id: string;
  project_id: string;
  user_id: string;
  workspace_id: string;
  role: ProjectRole;
  can_edit: boolean;
  can_delete: boolean;
  can_invite: boolean;
  assigned_by: string;
  assigned_at: string;
  removed_at: string | null;
  status: MemberStatus;
  created_at: string;
  updated_at: string;
  // User details (from join)
  user_name?: string;
  user_email?: string;
  user_avatar?: string;
}

export interface AddProjectMemberData {
  user_id: string;
  role: ProjectRole;
  workspace_id: string;
}

export interface UpdateMemberRoleData {
  role: ProjectRole;
}

export interface ProjectAccessInfo {
  has_access: boolean;
  role: ProjectRole | null;
  can_edit: boolean;
  can_delete: boolean;
  can_invite: boolean;
}

// Re-export other common types as needed in future
