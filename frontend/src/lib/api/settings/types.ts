// Settings API Types

export interface UserSettings {
  id: string;
  user_id: string;
  default_model: string | null;
  email_notifications: boolean;
  daily_summary: boolean;
  theme: string;
  sidebar_collapsed: boolean;
  share_analytics: boolean;
  custom_settings: Record<string, unknown> | null;
  created_at: string;
  updated_at: string;
}

export interface UserSettingsUpdate {
  default_model?: string | null;
  email_notifications?: boolean;
  daily_summary?: boolean;
  theme?: string;
  sidebar_collapsed?: boolean;
  share_analytics?: boolean;
  custom_settings?: Record<string, unknown>;
}

export interface AvailableModel {
  name: string;
  display_name: string;
  provider: string;
  description: string | null;
}

export interface SystemInfo {
  ollama_mode: string;
  active_provider?: string;
  available_models: AvailableModel[];
  default_model: string;
}
