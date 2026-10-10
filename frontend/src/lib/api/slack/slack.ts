// Slack API client
// Routes under /integrations/slack/* — see internal/integrations/slack/handler.go.
import { request } from "$lib/api/base";
import type {
  SlackAuthResponse,
  SlackChannelsResponse,
  SlackConnectionStatus,
  SlackDisconnectResponse,
  SlackMessagesResponse,
  SlackSendMessageResponse,
  SlackSyncChannelsResult,
  SlackSyncMessagesResult,
  GetSlackMessagesParams,
} from "./types";

const SLACK_BASE = "/integrations/slack";

export function getSlackConnectionStatus(): Promise<SlackConnectionStatus> {
  return request<SlackConnectionStatus>(`${SLACK_BASE}/status`);
}

export function initiateSlackAuth(): Promise<SlackAuthResponse> {
  return request<SlackAuthResponse>(`${SLACK_BASE}/auth`);
}

export function disconnectSlack(): Promise<SlackDisconnectResponse> {
  return request<SlackDisconnectResponse>(`${SLACK_BASE}/disconnect`, {
    method: "POST",
  });
}

export function getSlackChannels(): Promise<SlackChannelsResponse> {
  return request<SlackChannelsResponse>(`${SLACK_BASE}/channels`);
}

export function syncSlackChannels(): Promise<SlackSyncChannelsResult> {
  return request<SlackSyncChannelsResult>(`${SLACK_BASE}/channels/sync`, {
    method: "POST",
  });
}

export function getSlackMessages(
  channelId: string,
  params?: GetSlackMessagesParams,
): Promise<SlackMessagesResponse> {
  const search = new URLSearchParams();
  if (params?.limit !== undefined) search.set("limit", String(params.limit));
  if (params?.offset !== undefined) search.set("offset", String(params.offset));
  const qs = search.toString();
  return request<SlackMessagesResponse>(
    `${SLACK_BASE}/messages/${channelId}${qs ? `?${qs}` : ""}`,
  );
}

// Backend handler binds JSON key `text` (slack/handler.go SendMessage).
export function sendSlackMessage(
  channelId: string,
  text: string,
): Promise<SlackSendMessageResponse> {
  return request<SlackSendMessageResponse>(
    `${SLACK_BASE}/messages/${channelId}`,
    { method: "POST", body: { text } },
  );
}

export function syncSlackMessages(
  channelId: string,
  limit?: number,
): Promise<SlackSyncMessagesResult> {
  const search = new URLSearchParams();
  if (limit !== undefined) search.set("limit", String(limit));
  const qs = search.toString();
  return request<SlackSyncMessagesResult>(
    `${SLACK_BASE}/messages/${channelId}/sync${qs ? `?${qs}` : ""}`,
    { method: "POST" },
  );
}
