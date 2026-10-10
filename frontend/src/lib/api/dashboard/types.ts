// Dashboard API Types

export type TaskPriority = "critical" | "high" | "medium" | "low";
export type TaskStatus = "todo" | "in_progress" | "done" | "cancelled";

export interface FocusItem {
  id: string;
  text: string;
  completed: boolean;
  focus_date: string;
  created_at: string;
}

export interface Task {
  id: string;
  title: string;
  description: string | null;
  status: TaskStatus;
  priority: TaskPriority;
  due_date: string | null;
  start_date?: string | null;
  completed_at: string | null;
  project_id: string | null;
  assignee_id: string | null;
  parent_task_id?: string | null;
  estimated_hours?: number | null;
  created_at: string;
  updated_at: string;
}

export interface CreateTaskData {
  title: string;
  description?: string;
  priority?: TaskPriority;
  due_date?: string;
  start_date?: string;
  project_id?: string;
  assignee_id?: string;
  parent_task_id?: string;
  estimated_hours?: number;
}

export interface UpdateTaskData {
  title?: string;
  description?: string;
  status?: TaskStatus;
  priority?: TaskPriority;
  due_date?: string;
  project_id?: string;
  assignee_id?: string;
  position?: number;
}

export interface DashboardTask {
  id: string;
  title: string;
  project_name: string | null;
  due_date: string | null;
  priority: TaskPriority;
  completed: boolean;
}

export interface DashboardProject {
  id: string;
  name: string;
  client_name: string | null;
  project_type: string;
  due_date: string | null;
  progress: number;
  health: "healthy" | "at_risk" | "critical";
  team_count: number;
}

export type ActivityType =
  | "task_completed"
  | "task_started"
  | "project_created"
  | "project_updated"
  | "conversation"
  | "team"
  | "artifact";

export interface DashboardActivity {
  id: string;
  type: ActivityType;
  description: string;
  actor_name: string | null;
  actor_avatar: string | null;
  target_id: string | null;
  target_type: string | null;
  created_at: string;
}

export interface DashboardSummary {
  focus_items: FocusItem[];
  tasks: DashboardTask[];
  projects: DashboardProject[];
  activities: DashboardActivity[];
  energy_level: number | null;
  // Rollup counts returned by the backend summary endpoint.
  task_count?: number;
  project_count?: number;
  client_count?: number;
  context_count?: number;
  artifact_count?: number;
}
