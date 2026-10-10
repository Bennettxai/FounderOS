import { writable } from 'svelte/store';
import {
  getAppTemplates,
  getAppTemplate,
  getBuiltInTemplates,
  getTemplateRecommendations,
  generateAppFromTemplate,
  type AppTemplate,
  type AppTemplateRecommendation,
  type BuiltInTemplateInfo,
  type GenerationResult,
  type ListTemplatesParams,
  type TemplateCategory,
  type BusinessType,
  type TeamSize
} from '$lib/api/templates';

interface TemplateFilters {
  category: TemplateCategory | null;
  business_type: BusinessType | null;
  team_size: TeamSize | null;
  search: string;
  sort: 'popular' | 'newest' | 'name';
}

interface TemplateState {
  templates: AppTemplate[];
  recommendations: AppTemplateRecommendation[];
  builtInTemplates: BuiltInTemplateInfo[];
  currentTemplate: AppTemplate | null;
  generationResult: GenerationResult | null;
  queueItemId: string | null;
  generating: boolean;
  loading: boolean;
  error: string | null;
  filters: TemplateFilters;
  total: number;
}

function createTemplateStore() {
  const { subscribe, update } = writable<TemplateState>({
    templates: [],
    recommendations: [],
    builtInTemplates: [],
    currentTemplate: null,
    generationResult: null,
    queueItemId: null,
    generating: false,
    loading: false,
    error: null,
    filters: {
      category: null,
      business_type: null,
      team_size: null,
      search: '',
      sort: 'popular'
    },
    total: 0
  });

  return {
    subscribe,

    /**
     * Load templates with current filters
     */
    async loadTemplates() {
      update((s) => ({ ...s, loading: true, error: null }));
      try {
        let state: TemplateState;
        update((s) => {
          state = s;
          return s;
        });

        const params: ListTemplatesParams = {
          category: state!.filters.category || undefined,
          business_type: state!.filters.business_type || undefined,
          team_size: state!.filters.team_size || undefined,
          search: state!.filters.search || undefined,
          sort: state!.filters.sort
        };

        const response = await getAppTemplates(params);
        update((s) => ({
          ...s,
          templates: response.templates,
          total: response.total,
          loading: false
        }));
      } catch (error) {
        console.error('Failed to load templates:', error);
        update((s) => ({
          ...s,
          loading: false,
          error: error instanceof Error ? error.message : 'Failed to load templates'
        }));
      }
    },

    /**
     * Load a single template by ID
     */
    async loadTemplate(id: string) {
      update((s) => ({ ...s, loading: true, error: null }));
      try {
        const template = await getAppTemplate(id);
        update((s) => ({ ...s, currentTemplate: template, loading: false }));
        return template;
      } catch (error) {
        console.error('Failed to load template:', error);
        update((s) => ({
          ...s,
          loading: false,
          error: error instanceof Error ? error.message : 'Failed to load template'
        }));
        return null;
      }
    },

    /**
     * Load personalized recommendations for workspace
     */
    async loadRecommendations(workspaceId: string) {
      update((s) => ({ ...s, loading: true, error: null }));
      try {
        const recommendations = await getTemplateRecommendations(workspaceId);
        update((s) => ({ ...s, recommendations, loading: false }));
      } catch (error) {
        console.error('Failed to load recommendations:', error);
        update((s) => ({
          ...s,
          loading: false,
          error: error instanceof Error ? error.message : 'Failed to load recommendations'
        }));
      }
    },

    /**
     * Load built-in template definitions
     */
    async loadBuiltInTemplates() {
      try {
        const response = await getBuiltInTemplates();
        update((s) => ({ ...s, builtInTemplates: response.templates }));
      } catch (error) {
        console.error('Failed to load built-in templates:', error);
      }
    },

    /**
     * Generate app from template.
     * Backend returns queue_item_id for async SSE tracking (same pipeline as freeform gen).
     * If response includes a full result (sync fallback), sets generationResult directly.
     */
    async generateApp(
      templateId: string,
      workspaceId: string,
      appName: string,
      config?: Record<string, string | number | boolean>
    ): Promise<GenerationResult | null> {
      update((s) => ({ ...s, generating: true, error: null, generationResult: null, queueItemId: null }));
      try {
        const response = await generateAppFromTemplate(templateId, {
          workspace_id: workspaceId,
          app_name: appName,
          config
        });

        // Async queue flow: backend returns queue_item_id for SSE tracking
        if (response.queue_item_id) {
          update((s) => ({
            ...s,
            queueItemId: response.queue_item_id!
            // generating stays true — page will show AgentProgressPanel
          }));
          return null;
        }

        // Sync fallback: full result returned immediately
        if (response.result) {
          update((s) => ({
            ...s,
            generating: false,
            generationResult: response.result!
          }));
          return response.result;
        }

        update((s) => ({ ...s, generating: false }));
        return null;
      } catch (error) {
        console.error('Failed to generate app from template:', error);
        update((s) => ({
          ...s,
          generating: false,
          queueItemId: null,
          error: error instanceof Error ? error.message : 'Failed to generate app'
        }));
        return null;
      }
    },

    /**
     * Clear generation result and queue tracking state
     */
    clearGenerationResult() {
      update((s) => ({ ...s, generationResult: null, queueItemId: null, generating: false }));
    },

    /**
     * Update filters and reload templates
     */
    setFilters(filters: Partial<TemplateFilters>) {
      update((s) => ({
        ...s,
        filters: { ...s.filters, ...filters }
      }));
    },

    /**
     * Clear all filters
     */
    clearFilters() {
      update((s) => ({
        ...s,
        filters: {
          category: null,
          business_type: null,
          team_size: null,
          search: '',
          sort: 'popular'
        }
      }));
    },

    /**
     * Clear current template
     */
    clearCurrent() {
      update((s) => ({ ...s, currentTemplate: null }));
    },

    /**
     * Clear error
     */
    clearError() {
      update((s) => ({ ...s, error: null }));
    }
  };
}

export const templateStore = createTemplateStore();

// Category labels and icons for UI
export const categoryLabels: Record<TemplateCategory, string> = {
  crm: 'CRM',
  project_management: 'Project Management',
  hr: 'Human Resources',
  finance: 'Finance',
  marketing: 'Marketing',
  operations: 'Operations',
  custom: 'Custom'
};

export const categoryColors: Record<TemplateCategory, string> = {
  crm: 'bg-blue-50 text-blue-700 border-blue-200',
  project_management: 'bg-purple-50 text-purple-700 border-purple-200',
  hr: 'bg-green-50 text-green-700 border-green-200',
  finance: 'bg-emerald-50 text-emerald-700 border-emerald-200',
  marketing: 'bg-pink-50 text-pink-700 border-pink-200',
  operations: 'bg-orange-50 text-orange-700 border-orange-200',
  custom: 'bg-gray-50 text-gray-700 border-gray-200'
};
