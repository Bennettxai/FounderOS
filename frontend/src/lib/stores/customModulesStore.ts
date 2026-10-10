import { writable } from 'svelte/store';
import {
  getModules,
  getModule,
  createModule,
  updateModule,
  deleteModule,
  installModule,
  uninstallModule,
  exportModule,
  importModule,
  shareModule,
  getModuleVersions
} from '$lib/api/modules';
import type {
  CustomModule,
  CreateModuleData,
  ModuleFilters,
  ShareModuleData,
  UpdateModuleData,
  ModuleVersion,
  ModuleInstallation
} from '$lib/types/modules';

interface ModulesState {
  modules: CustomModule[];
  currentModule: CustomModule | null;
  versions: ModuleVersion[];
  installations: ModuleInstallation[];
  loading: boolean;
  error: string | null;
  filters: ModuleFilters;
  total: number;
}

function createModulesStore() {
  const { subscribe, update } = writable<ModulesState>({
    modules: [],
    currentModule: null,
    versions: [],
    installations: [],
    loading: false,
    error: null,
    filters: {
      category: null,
      search: '',
      sort: 'popular',
      visibility: null
    },
    total: 0
  });

  return {
    subscribe,

    /**
     * Load modules with current filters
     */
    async loadModules() {
      update((s) => ({ ...s, loading: true, error: null }));
      try {
        let state: ModulesState;
        update((s) => {
          state = s;
          return s;
        });

        const params = {
          category: state!.filters.category || undefined,
          search: state!.filters.search || undefined,
          sort: state!.filters.sort,
          visibility: state!.filters.visibility || undefined
        };

        const response = await getModules(params);
        update((s) => ({
          ...s,
          modules: response.modules,
          total: response.total,
          loading: false
        }));
      } catch (error) {
        console.error('Failed to load modules:', error);
        update((s) => ({
          ...s,
          loading: false,
          error: error instanceof Error ? error.message : 'Failed to load modules'
        }));
      }
    },

    /**
     * Search modules (debounced externally)
     */
    async searchModules(query: string) {
      update((s) => ({
        ...s,
        filters: { ...s.filters, search: query }
      }));
    },

    /**
     * Load a single module by ID
     */
    async loadModule(id: string) {
      update((s) => ({ ...s, loading: true, error: null }));
      try {
        const module = await getModule(id);
        update((s) => ({ ...s, currentModule: module, loading: false }));
        return module;
      } catch (error) {
        console.error('Failed to load module:', error);
        update((s) => ({
          ...s,
          loading: false,
          error: error instanceof Error ? error.message : 'Failed to load module'
        }));
        return null;
      }
    },

    /**
     * Create a new module
     */
    async createModule(data: CreateModuleData): Promise<CustomModule | null> {
      update((s) => ({ ...s, loading: true, error: null }));
      try {
        const module = await createModule(data);
        update((s) => ({
          ...s,
          modules: [module, ...s.modules],
          currentModule: module,
          loading: false
        }));
        return module;
      } catch (error) {
        console.error('Failed to create module:', error);
        update((s) => ({
          ...s,
          loading: false,
          error: error instanceof Error ? error.message : 'Failed to create module'
        }));
        return null;
      }
    },

    /**
     * Update an existing module
     */
    async updateModule(id: string, data: UpdateModuleData): Promise<CustomModule | null> {
      update((s) => ({ ...s, loading: true, error: null }));
      try {
        const module = await updateModule(id, data);
        update((s) => ({
          ...s,
          modules: s.modules.map((m) => (m.id === id ? module : m)),
          currentModule: s.currentModule?.id === id ? module : s.currentModule,
          loading: false
        }));
        return module;
      } catch (error) {
        console.error('Failed to update module:', error);
        update((s) => ({
          ...s,
          loading: false,
          error: error instanceof Error ? error.message : 'Failed to update module'
        }));
        return null;
      }
    },

    /**
     * Delete a module
     */
    async deleteModule(id: string): Promise<boolean> {
      update((s) => ({ ...s, loading: true, error: null }));
      try {
        await deleteModule(id);
        update((s) => ({
          ...s,
          modules: s.modules.filter((m) => m.id !== id),
          currentModule: s.currentModule?.id === id ? null : s.currentModule,
          loading: false
        }));
        return true;
      } catch (error) {
        console.error('Failed to delete module:', error);
        update((s) => ({
          ...s,
          loading: false,
          error: error instanceof Error ? error.message : 'Failed to delete module'
        }));
        return false;
      }
    },

    /**
     * Load module versions
     */
    async loadVersions(moduleId: string) {
      try {
        const response = await getModuleVersions(moduleId);
        update((s) => ({ ...s, versions: response.versions }));
      } catch (error) {
        console.error('Failed to load versions:', error);
      }
    },

    /**
     * Install a module
     */
    async installModule(moduleId: string, config?: Record<string, unknown>): Promise<boolean> {
      update((s) => ({ ...s, loading: true, error: null }));
      try {
        await installModule(moduleId, config);
        update((s) => ({
          ...s,
          loading: false
        }));
        // Reload the module to get updated install count
        await this.loadModule(moduleId);
        return true;
      } catch (error) {
        console.error('Failed to install module:', error);
        update((s) => ({
          ...s,
          loading: false,
          error: error instanceof Error ? error.message : 'Failed to install module'
        }));
        return false;
      }
    },

    /**
     * Uninstall a module
     */
    async uninstallModule(moduleId: string): Promise<boolean> {
      update((s) => ({ ...s, loading: true, error: null }));
      try {
        await uninstallModule(moduleId);
        update((s) => ({
          ...s,
          loading: false
        }));
        // Reload the module to get updated install count
        await this.loadModule(moduleId);
        return true;
      } catch (error) {
        console.error('Failed to uninstall module:', error);
        update((s) => ({
          ...s,
          loading: false,
          error: error instanceof Error ? error.message : 'Failed to uninstall module'
        }));
        return false;
      }
    },

    /**
     * Share a module
     */
    async shareModule(moduleId: string, data: ShareModuleData): Promise<boolean> {
      update((s) => ({ ...s, loading: true, error: null }));
      try {
        await shareModule(moduleId, data);
        update((s) => ({
          ...s,
          loading: false
        }));
        return true;
      } catch (error) {
        console.error('Failed to share module:', error);
        update((s) => ({
          ...s,
          loading: false,
          error: error instanceof Error ? error.message : 'Failed to share module'
        }));
        return false;
      }
    },

    /**
     * Export a module
     */
    async exportModule(moduleId: string): Promise<Blob | null> {
      update((s) => ({ ...s, loading: true, error: null }));
      try {
        const blob = await exportModule(moduleId);
        update((s) => ({ ...s, loading: false }));
        return blob;
      } catch (error) {
        console.error('Failed to export module:', error);
        update((s) => ({
          ...s,
          loading: false,
          error: error instanceof Error ? error.message : 'Failed to export module'
        }));
        return null;
      }
    },

    /**
     * Import a module
     */
    async importModule(file: File): Promise<CustomModule | null> {
      update((s) => ({ ...s, loading: true, error: null }));
      try {
        const module = await importModule(file);
        update((s) => ({
          ...s,
          modules: [module, ...s.modules],
          loading: false
        }));
        return module;
      } catch (error) {
        console.error('Failed to import module:', error);
        update((s) => ({
          ...s,
          loading: false,
          error: error instanceof Error ? error.message : 'Failed to import module'
        }));
        return null;
      }
    },

    /**
     * Update filters and reload modules
     */
    setFilters(filters: Partial<ModuleFilters>) {
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
          search: '',
          sort: 'popular',
          visibility: null
        }
      }));
    },

    /**
     * Clear current module
     */
    clearCurrent() {
      update((s) => ({ ...s, currentModule: null, versions: [] }));
    },

    /**
     * Clear error
     */
    clearError() {
      update((s) => ({ ...s, error: null }));
    }
  };
}

export const customModulesStore = createModulesStore();
