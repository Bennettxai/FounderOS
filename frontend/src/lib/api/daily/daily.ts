import { request } from '../base';
import type { DailyLog, CreateDailyLogData, UpdateDailyLogData } from './types';

export async function getDailyLogs(skip: number = 0, limit: number = 30) {
  return request<DailyLog[]>(`/daily/logs?skip=${skip}&limit=${limit}`);
}

export async function getTodayLog() {
  return request<DailyLog | null>('/daily/logs/today');
}

export async function getDailyLogByDate(date: string) {
  return request<DailyLog | null>(`/daily/logs/${date}`);
}

export async function saveDailyLog(data: CreateDailyLogData) {
  return request<DailyLog>('/daily/logs', { method: 'POST', body: data });
}

export async function updateDailyLog(id: string, data: UpdateDailyLogData) {
  return request<DailyLog>(`/daily/logs/${id}`, { method: 'PUT', body: data });
}

export async function deleteDailyLog(id: string) {
  return request(`/daily/logs/${id}`, { method: 'DELETE' });
}
