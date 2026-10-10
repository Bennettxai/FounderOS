// App Profiles API - Codebase Analysis
import { request } from '../base';
import type {
  ApplicationProfile,
  ProfileListItem,
  TechStackInfo,
  ComponentInfo,
  EndpointInfo,
  ProjectStructure,
  ModuleInfo,
  AnalyzeCodebaseInput
} from './types';

/**
 * Analyze a codebase and create a profile
 */
export async function analyzeCodebase(
  input: AnalyzeCodebaseInput
): Promise<ApplicationProfile> {
  return request<ApplicationProfile>('/app-profiles', {
    method: 'POST',
    body: JSON.stringify(input)
  });
}

/**
 * List all application profiles
 */
export async function listProfiles(): Promise<ProfileListItem[]> {
  return request<ProfileListItem[]>('/app-profiles');
}

/**
 * Get a specific profile by name
 */
export async function getProfile(name: string): Promise<ApplicationProfile> {
  return request<ApplicationProfile>(`/app-profiles/${encodeURIComponent(name)}`);
}

/**
 * Refresh/re-analyze a profile
 * Renamed from refreshProfile to avoid conflict with learning module
 */
export async function refreshAppProfile(name: string): Promise<ApplicationProfile> {
  return request<ApplicationProfile>(`/app-profiles/${encodeURIComponent(name)}/refresh`, {
    method: 'POST'
  });
}

/**
 * Delete a profile
 */
export async function deleteProfile(name: string): Promise<void> {
  await request<void>(`/app-profiles/${encodeURIComponent(name)}`, {
    method: 'DELETE'
  });
}

/**
 * Get profile components
 */
export async function getProfileComponents(name: string): Promise<ComponentInfo[]> {
  return request<ComponentInfo[]>(`/app-profiles/${encodeURIComponent(name)}/components`);
}

/**
 * Get profile endpoints
 */
export async function getProfileEndpoints(name: string): Promise<EndpointInfo[]> {
  return request<EndpointInfo[]>(`/app-profiles/${encodeURIComponent(name)}/endpoints`);
}

/**
 * Get profile structure
 */
export async function getProfileStructure(name: string): Promise<ProjectStructure> {
  return request<ProjectStructure>(`/app-profiles/${encodeURIComponent(name)}/structure`);
}

/**
 * Get profile modules
 */
export async function getProfileModules(name: string): Promise<ModuleInfo[]> {
  return request<ModuleInfo[]>(`/app-profiles/${encodeURIComponent(name)}/modules`);
}

/**
 * Get profile tech stack
 */
export async function getProfileTechStack(name: string): Promise<TechStackInfo> {
  return request<TechStackInfo>(`/app-profiles/${encodeURIComponent(name)}/tech-stack`);
}
