import { request } from '../base';
import type { UserSettings, UserSettingsUpdate, SystemInfo } from './types';

export async function getSettings() {
  return request<UserSettings>('/settings');
}

export async function updateSettings(data: UserSettingsUpdate) {
  return request<UserSettings>('/settings', { method: 'PUT', body: data });
}

export async function getSystemInfo() {
  return request<SystemInfo>('/settings/system');
}
