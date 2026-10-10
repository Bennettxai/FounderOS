import { request } from '../base';
import type {
  DashboardSummary,
  FocusItem,
  Task,
  CreateTaskData,
  UpdateTaskData
} from './types';


export async function getDashboardSummary() {
  return request<DashboardSummary>('/dashboard/summary');
}

export async function getFocusItems() {
  return request<FocusItem[]>('/dashboard/focus');
}

export async function createFocusItem(text: string) {
  return request<FocusItem>('/dashboard/focus', { method: 'POST', body: { text } });
}

export async function updateFocusItem(id: string, data: { text?: string; completed?: boolean }) {
  return request<FocusItem>(`/dashboard/focus/${id}`, { method: 'PUT', body: data });
}

export async function deleteFocusItem(id: string) {
  return request(`/dashboard/focus/${id}`, { method: 'DELETE' });
}

// ============ Tasks ============

export async function getTasks(filters?: { status?: string; priority?: string; projectId?: string }) {
  const params = new URLSearchParams();
  if (filters?.status) params.set('status_filter', filters.status);
  if (filters?.priority) params.set('priority_filter', filters.priority);
  if (filters?.projectId) params.set('project_id', filters.projectId);
  const query = params.toString();
  return request<Task[]>(`/dashboard/tasks${query ? `?${query}` : ''}`);
}

export async function createTask(data: CreateTaskData) {
  return request<Task>('/dashboard/tasks', { method: 'POST', body: data });
}

export async function updateTask(id: string, data: UpdateTaskData) {
  return request<Task>(`/dashboard/tasks/${id}`, { method: 'PUT', body: data });
}

export async function toggleTask(id: string) {
  return request<Task>(`/dashboard/tasks/${id}/toggle`, { method: 'POST' });
}

export async function deleteTask(id: string) {
  return request(`/dashboard/tasks/${id}`, { method: 'DELETE' });
}
