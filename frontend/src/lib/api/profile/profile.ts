import { request, getApiBaseUrl } from '../base';
import type { ProfileUpdateData, ProfileUpdateResponse, PhotoUploadResponse } from './types';

export async function updateProfile(data: ProfileUpdateData) {
  return request<ProfileUpdateResponse>('/profile', {
    method: 'PUT',
    body: data
  });
}

export async function uploadProfilePhoto(file: File): Promise<PhotoUploadResponse> {
  const formData = new FormData();
  formData.append('file', file);

  const response = await fetch(`${getApiBaseUrl()}/profile/photo`, {
    method: 'POST',
    credentials: 'include',
    body: formData
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Upload failed' }));
    throw new Error(error.error || 'Upload failed');
  }

  return response.json();
}

export async function deleteProfilePhoto() {
  return request<{ message: string }>('/profile/photo', { method: 'DELETE' });
}
