import { request } from '../base';
import type { ProjectMember, AddProjectMemberData, UpdateMemberRoleData, ProjectAccessInfo } from './types';

/**
 * Get all members of a project
 */
export async function listProjectMembers(projectId: string): Promise<ProjectMember[]> {
  return request<ProjectMember[]>(`/projects/${projectId}/members`);
}

/**
 * Add a member to a project
 */
export async function addProjectMember(
  projectId: string,
  data: AddProjectMemberData
): Promise<ProjectMember> {
  return request<ProjectMember>(`/projects/${projectId}/members`, {
    method: 'POST',
    body: data,
  });
}

/**
 * Update a project member's role
 */
export async function updateProjectMemberRole(
  projectId: string,
  memberId: string,
  data: UpdateMemberRoleData
): Promise<ProjectMember> {
  return request<ProjectMember>(`/projects/${projectId}/members/${memberId}/role`, {
    method: 'PUT',
    body: data,
  });
}

/**
 * Remove a member from a project
 */
export async function removeProjectMember(projectId: string, memberId: string): Promise<void> {
  return request(`/projects/${projectId}/members/${memberId}`, {
    method: 'DELETE',
  }) as unknown as void;
}

/**
 * Check if a user has access to a project and get their permissions
 */
export async function checkProjectAccess(
  projectId: string,
  userId: string
): Promise<ProjectAccessInfo> {
  return request<ProjectAccessInfo>(`/projects/${projectId}/access/${userId}`);
}
