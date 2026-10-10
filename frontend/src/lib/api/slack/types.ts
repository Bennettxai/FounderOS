// Slack API Types
// Mirrors the Go backend shapes:
//   internal/integrations/slack/channels.go  (Channel)
//   internal/integrations/slack/messages.go  (Message)
//   internal/integrations/slack/handler.go   (response wrappers)
// and the schema in internal/database/schema.sql (slack_channels, slack_messages).

export interface SlackConnectionStatus {
  connected: boolean;
  workspace_name?: string;
  workspace_id?: string;
  user_name?: string;
  user_id?: string;
  connected_at?: string;
}

export interface SlackAuthResponse {
  auth_url: string;
}

export interface SlackChannel {
  id: string;
  user_id: string;
  slack_id: string;
  name: string;
  is_private: boolean;
  is_dm: boolean;
  member_count: number;
  topic?: string;
  purpose?: string;
  unread_count: number;
  last_activity?: string;
  created_at: string;
  updated_at: string;
}

// Reactions are not yet surfaced by the backend; kept optional so the UI
// branch survives the day they are added without re-typing.
export interface SlackReaction {
  icon: string;
  count: number;
  users?: string[];
}

export interface SlackMessage {
  id: string;
  user_id: string;
  channel_id: string;
  slack_ts: string;
  sender_id: string;
  sender_name: string;
  content: string;
  thread_ts?: string;
  reply_count: number;
  attachments?: unknown[];
  is_edited: boolean;
  sent_at: string;
  created_at: string;
  updated_at: string;
  reactions?: SlackReaction[];
}

export interface SlackChannelsResponse {
  channels: SlackChannel[];
  count: number;
}

export interface SlackMessagesResponse {
  messages: SlackMessage[];
  count: number;
}

export interface SlackSyncChannelsResult {
  total_channels: number;
  synced_channels: number;
  failed_channels: number;
}

export interface SlackSyncMessagesResult {
  total_messages: number;
  synced_messages: number;
  failed_messages: number;
}

export interface SlackSendMessageResponse {
  success: boolean;
}

export interface SlackDisconnectResponse {
  success: boolean;
}

export interface GetSlackMessagesParams {
  limit?: number;
  offset?: number;
}
