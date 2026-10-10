import { API_BASE, request } from '../base';
import type {
  AppTemplate,
  AppTemplateRecommendation,
  BuiltInTemplateInfo,
  GenerateFromTemplateRequest,
  GenerateFromTemplateResponse,
  ListTemplatesParams,
  ListTemplatesResponse
} from './types';

/**
 * Fetch all app templates with optional filters
 */
export async function getAppTemplates(
  params?: ListTemplatesParams
): Promise<ListTemplatesResponse> {
  const searchParams = new URLSearchParams();

  if (params?.category) searchParams.set('category', params.category);
  if (params?.business_type) searchParams.set('business_type', params.business_type);
  if (params?.team_size) searchParams.set('team_size', params.team_size);
  if (params?.search) searchParams.set('search', params.search);
  if (params?.sort) searchParams.set('sort', params.sort);
  if (params?.limit) searchParams.set('limit', params.limit.toString());
  if (params?.offset) searchParams.set('offset', params.offset.toString());

  const url = `${API_BASE}/app-templates${searchParams.toString() ? `?${searchParams}` : ''}`;
  return request<ListTemplatesResponse>(url, { method: 'GET' });
}

/**
 * Get a single app template by ID
 */
export async function getAppTemplate(id: string): Promise<AppTemplate> {
  return request<AppTemplate>(`${API_BASE}/app-templates/${id}`, { method: 'GET' });
}

/**
 * Get built-in template definitions with config schemas
 */
export async function getBuiltInTemplates(): Promise<{ templates: BuiltInTemplateInfo[] }> {
  return request<{ templates: BuiltInTemplateInfo[] }>(
    `${API_BASE}/app-templates/builtin`,
    { method: 'GET' }
  );
}

/**
 * Get personalized template recommendations for a workspace
 */
export async function getTemplateRecommendations(
  workspaceId: string
): Promise<AppTemplateRecommendation[]> {
  return request<AppTemplateRecommendation[]>(
    `${API_BASE}/workspaces/${workspaceId}/template-recommendations`,
    { method: 'GET' }
  );
}

/**
 * Generate an app from a template with configuration.
 * Returns queue_item_id for async SSE tracking (same pipeline as freeform generation).
 */
export async function generateAppFromTemplate(
  templateId: string,
  request_body: GenerateFromTemplateRequest
): Promise<GenerateFromTemplateResponse> {
  return request<GenerateFromTemplateResponse>(
    `${API_BASE}/app-templates/${templateId}/generate`,
    {
      method: 'POST',
      body: JSON.stringify(request_body)
    }
  );
}
