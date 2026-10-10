// Artifacts API Types

export type ArtifactType = 'proposal' | 'sop' | 'framework' | 'agenda' | 'report' | 'plan' | 'code' | 'document' | 'markdown' | 'other';

export interface ArtifactListItem {
  id: string;
  title: string;
  type: ArtifactType;
  summary: string | null;
  conversation_id: string | null;
  message_id: string | null;
  project_id: string | null;
  context_id: string | null;
  context_name: string | null;
  language: string | null;
  created_at: string;
  updated_at: string;
}

export interface Artifact extends ArtifactListItem {
  content: string;
  version: number;
}

export interface CreateArtifactData {
  title: string;
  content: string;
  type?: ArtifactType;
  summary?: string;
  conversation_id?: string;
  project_id?: string;
}

export interface UpdateArtifactData {
  title?: string;
  content?: string;
  summary?: string;
}

export interface ArtifactFilters {
  type?: string;
  conversationId?: string;
  projectId?: string;
  contextId?: string;
  unassignedOnly?: boolean;
}

export interface ArtifactVersion {
  id: string;
  artifact_id: string;
  version: number;
  content: string;
  created_at: string;
}
