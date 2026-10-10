export interface VoiceNote {
  id: string;
  filename: string;
  transcript: string;
  duration: number;
  created_at: string;
  url: string;
  context_id?: string;
}
