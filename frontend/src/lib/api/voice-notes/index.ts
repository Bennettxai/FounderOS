export * from './types';
export * from './voice-notes';

import * as voiceNotesApi from './voice-notes';

export const api = {
  getVoiceNotes: voiceNotesApi.getVoiceNotes,
  uploadVoiceNote: voiceNotesApi.uploadVoiceNote,
  getVoiceNoteAudio: voiceNotesApi.getVoiceNoteAudio,
  deleteVoiceNote: voiceNotesApi.deleteVoiceNote,
  retranscribeVoiceNote: voiceNotesApi.retranscribeVoiceNote,
};

export default api;
