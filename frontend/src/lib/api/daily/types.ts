// Daily Logs API Types

export interface DailyLog {
  id: string;
  date: string;
  content: string;
  energy_level: number | null;
  extracted_actions: Record<string, unknown> | null;
  extracted_patterns: Record<string, unknown> | null;
  created_at: string;
  updated_at: string;
}

export interface CreateDailyLogData {
  content: string;
  energy_level?: number;
  date?: string;
}

export interface UpdateDailyLogData {
  content?: string;
  energy_level?: number;
}
