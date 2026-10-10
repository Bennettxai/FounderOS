
export type TeamMemberStatus = 'available' | 'busy' | 'overloaded' | 'ooo';

export interface TeamMemberActivityResponse {
  id: string;
  activity_type: string;
  description: string;
  created_at: string;
}

export interface TeamMemberResponse {
  id: string;
  name: string;
  email: string;
  role: string;
  avatar_url: string | null;
  status: TeamMemberStatus;
  capacity: number;
  manager_id: string | null;
  skills: string[] | null;
  hourly_rate: number | null;
  joined_at: string;
  created_at: string;
  updated_at: string;
}

export interface TeamMemberListResponse {
  id: string;
  name: string;
  email: string;
  role: string;
  avatar_url: string | null;
  status: TeamMemberStatus;
  capacity: number;
  manager_id: string | null;
  active_projects: number;
  open_tasks: number;
  joined_at: string;
}

export interface TeamMemberDetailResponse extends TeamMemberResponse {
  active_projects: number;
  open_tasks: number;
  activities: TeamMemberActivityResponse[];
}

export interface CreateTeamMemberData {
  name: string;
  email: string;
  role: string;
  avatar_url?: string;
  manager_id?: string;
  skills?: string[];
  hourly_rate?: number;
}

export interface UpdateTeamMemberData {
  name?: string;
  email?: string;
  role?: string;
  avatar_url?: string;
  status?: TeamMemberStatus;
  capacity?: number;
  manager_id?: string | null;
  skills?: string[];
  hourly_rate?: number;
}
