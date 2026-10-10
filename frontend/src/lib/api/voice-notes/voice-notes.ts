import { request, getApiBaseUrl } from '../base';
import type { VoiceNote } from './types';

export async function getVoiceNotes(contextId?: string) {
  const params = contextId ? `?context_id=${contextId}` : '';
  return request<VoiceNote[]>(`/voice-notes${params}`);
}

export async function uploadVoiceNote(audioBlob: Blob, contextId?: string): Promise<VoiceNote> {
  const formData = new FormData();
  formData.append('audio', audioBlob, 'recording.webm');
  if (contextId) {
    formData.append('context_id', contextId);
  }
  const response = await fetch(`${getApiBaseUrl()}/voice-notes`, {
    method: 'POST',
    credentials: 'include',
    body: formData
  });
  if (!response.ok) {
    const error = await response.json().catch(() => ({ detail: 'Upload failed' }));
    throw new Error(error.detail || 'Upload failed');
  }
  return response.json();
}

export async function getVoiceNoteAudio(noteId: string): Promise<Blob> {
  const response = await fetch(`${getApiBaseUrl()}/voice-notes/${noteId}`, {
    credentials: 'include'
  });
  if (!response.ok) {
    throw new Error('Failed to fetch audio');
  }
  return response.blob();
}

export async function deleteVoiceNote(noteId: string) {
  return request(`/voice-notes/${noteId}`, { method: 'DELETE' });
}

export async function retranscribeVoiceNote(noteId: string) {
  return request<VoiceNote>(`/voice-notes/${noteId}/retranscribe`, { method: 'POST' });
}
