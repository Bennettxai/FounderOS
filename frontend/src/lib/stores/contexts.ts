import { writable } from "svelte/store";
import { api } from "$lib/api";
import type {
  Context,
  ContextListItem,
  CreateContextData,
  UpdateContextData,
  Block,
  ShareResponse,
} from "$lib/api/contexts/types";

interface ContextsState {
  contexts: ContextListItem[];
  currentContext: Context | null;
  loading: boolean;
}

interface ContextFilters {
  type?: string;
  includeArchived?: boolean;
  templatesOnly?: boolean;
  parentId?: string;
  search?: string;
}

function createContextsStore() {
  const { subscribe, update } = writable<ContextsState>({
    contexts: [],
    currentContext: null,
    loading: false,
  });

  return {
    subscribe,

    async loadContexts(filters?: ContextFilters | string) {
      update((s) => ({ ...s, loading: true }));
      try {
        // Support both old string format and new filters object
        const filterObj =
          typeof filters === "string" ? { type: filters } : filters;
        const contexts = await api.getContexts(filterObj);
        update((s) => ({ ...s, contexts, loading: false }));
      } catch (error) {
        console.error("Failed to load contexts:", error);
        update((s) => ({ ...s, loading: false }));
      }
    },

    async loadContext(id: string) {
      update((s) => ({ ...s, loading: true }));
      try {
        const context = await api.getContext(id);
        update((s) => ({ ...s, currentContext: context, loading: false }));
        return context;
      } catch (error) {
        console.error("Failed to load context:", error);
        update((s) => ({ ...s, loading: false }));
        throw error;
      }
    },

    async createContext(data: CreateContextData) {
      try {
        const context = await api.createContext(data);
        update((s) => ({
          ...s,
          contexts: [
            {
              id: context.id,
              name: context.name,
              type: context.type,
              icon: context.icon,
              cover_image: context.cover_image,
              parent_id: context.parent_id,
              is_template: context.is_template,
              is_archived: context.is_archived,
              word_count: context.word_count,
              property_schema: context.property_schema,
              properties: context.properties,
              client_id: context.client_id,
              updated_at: context.updated_at,
            },
            ...s.contexts,
          ],
        }));
        return context;
      } catch (error) {
        console.error("Failed to create context:", error);
        throw error;
      }
    },

    async updateContext(id: string, data: UpdateContextData) {
      try {
        const context = await api.updateContext(id, data);
        update((s) => ({
          ...s,
          contexts: s.contexts.map((c) =>
            c.id === id
              ? {
                  id: context.id,
                  name: context.name,
                  type: context.type,
                  icon: context.icon,
                  cover_image: context.cover_image,
                  parent_id: context.parent_id,
                  is_template: context.is_template,
                  is_archived: context.is_archived,
                  word_count: context.word_count,
                  property_schema: context.property_schema,
                  properties: context.properties,
                  client_id: context.client_id,
                  updated_at: context.updated_at,
                }
              : c,
          ),
          currentContext:
            s.currentContext?.id === id ? context : s.currentContext,
        }));
        return context;
      } catch (error) {
        console.error("Failed to update context:", error);
        throw error;
      }
    },

    async updateBlocks(id: string, blocks: Block[], wordCount?: number) {
      try {
        const context = await api.updateContextBlocks(id, {
          blocks,
          word_count: wordCount,
        });
        update((s) => ({
          ...s,
          currentContext:
            s.currentContext?.id === id ? context : s.currentContext,
        }));
        return context;
      } catch (error) {
        console.error("Failed to update blocks:", error);
        throw error;
      }
    },

    async enableSharing(id: string): Promise<ShareResponse> {
      try {
        const response = await api.enableContextSharing(id);
        update((s) => ({
          ...s,
          currentContext:
            s.currentContext?.id === id
              ? {
                  ...s.currentContext,
                  is_public: true,
                  share_id: response.share_id,
                }
              : s.currentContext,
        }));
        return response;
      } catch (error) {
        console.error("Failed to enable sharing:", error);
        throw error;
      }
    },

    async disableSharing(id: string) {
      try {
        await api.disableContextSharing(id);
        update((s) => ({
          ...s,
          currentContext:
            s.currentContext?.id === id
              ? { ...s.currentContext, is_public: false }
              : s.currentContext,
        }));
      } catch (error) {
        console.error("Failed to disable sharing:", error);
        throw error;
      }
    },

    async duplicateContext(id: string) {
      try {
        const context = await api.duplicateContext(id);
        update((s) => ({
          ...s,
          contexts: [
            {
              id: context.id,
              name: context.name,
              type: context.type,
              icon: context.icon,
              cover_image: context.cover_image,
              parent_id: context.parent_id,
              is_template: context.is_template,
              is_archived: context.is_archived,
              word_count: context.word_count,
              property_schema: context.property_schema,
              properties: context.properties,
              client_id: context.client_id,
              updated_at: context.updated_at,
            },
            ...s.contexts,
          ],
        }));
        return context;
      } catch (error) {
        console.error("Failed to duplicate context:", error);
        throw error;
      }
    },

    async archiveContext(id: string) {
      try {
        await api.archiveContext(id);
        update((s) => ({
          ...s,
          contexts: s.contexts.filter((c) => c.id !== id),
          currentContext:
            s.currentContext?.id === id
              ? { ...s.currentContext, is_archived: true }
              : s.currentContext,
        }));
      } catch (error) {
        console.error("Failed to archive context:", error);
        throw error;
      }
    },

    async unarchiveContext(id: string) {
      try {
        await api.unarchiveContext(id);
        update((s) => ({
          ...s,
          currentContext:
            s.currentContext?.id === id
              ? { ...s.currentContext, is_archived: false }
              : s.currentContext,
        }));
      } catch (error) {
        console.error("Failed to unarchive context:", error);
        throw error;
      }
    },

    async deleteContext(id: string) {
      try {
        await api.deleteContext(id);
        update((s) => ({
          ...s,
          contexts: s.contexts.filter((c) => c.id !== id),
          currentContext: s.currentContext?.id === id ? null : s.currentContext,
        }));
      } catch (error) {
        console.error("Failed to delete context:", error);
        throw error;
      }
    },

    clearCurrent() {
      update((s) => ({ ...s, currentContext: null }));
    },
  };
}

export const contexts = createContextsStore();
