import { writable } from 'svelte/store';
import { api, type Project, type CreateProjectData } from '$lib/api/projects';

interface ProjectsState {
	projects: Project[];
	currentProject: Project | null;
	loading: boolean;
}

function createProjectsStore() {
	const { subscribe, update } = writable<ProjectsState>({
		projects: [],
		currentProject: null,
		loading: false
	});

	return {
		subscribe,

		async loadProjects(status?: string) {
			update((s) => ({ ...s, loading: true }));
			try {
				const projects = await api.getProjects(status);
				update((s) => ({ ...s, projects, loading: false }));
			} catch (error) {
				console.error('Failed to load projects:', error);
				update((s) => ({ ...s, loading: false }));
			}
		},

		async loadProject(id: string) {
			update((s) => ({ ...s, loading: true }));
			try {
				const project = await api.getProject(id);
				update((s) => ({ ...s, currentProject: project, loading: false }));
			} catch (error) {
				console.error('Failed to load project:', error);
				update((s) => ({ ...s, loading: false }));
			}
		},

		async createProject(data: CreateProjectData) {
			try {
				const project = await api.createProject(data);
				update((s) => ({ ...s, projects: [project, ...s.projects] }));
				return project;
			} catch (error) {
				console.error('Failed to create project:', error);
				throw error;
			}
		},

		async updateProject(id: string, data: Partial<CreateProjectData>) {
			try {
				const project = await api.updateProject(id, data);
				update((s) => ({
					...s,
					projects: s.projects.map((p) => (p.id === id ? project : p)),
					currentProject: s.currentProject?.id === id ? project : s.currentProject
				}));
				return project;
			} catch (error) {
				console.error('Failed to update project:', error);
				throw error;
			}
		},

		async deleteProject(id: string) {
			try {
				await api.deleteProject(id);
				update((s) => ({
					...s,
					projects: s.projects.filter((p) => p.id !== id),
					currentProject: s.currentProject?.id === id ? null : s.currentProject
				}));
			} catch (error) {
				console.error('Failed to delete project:', error);
				throw error;
			}
		},

		async addNote(projectId: string, content: string) {
			try {
				const note = await api.addProjectNote(projectId, content);
				update((s) => {
					if (!s.currentProject) return s;
					return {
						...s,
						currentProject: {
							...s.currentProject,
							notes: [...s.currentProject.notes, note as typeof s.currentProject.notes[0]]
						}
					};
				});
				return note;
			} catch (error) {
				console.error('Failed to add note:', error);
				throw error;
			}
		},

		/**
		 * Seed the store with preloaded data (from +page.ts load function).
		 * Avoids a redundant fetch when data was already prefetched on hover.
		 */
		setProjects(projectList: Project[]) {
			update((s) => ({ ...s, projects: projectList, loading: false }));
		},

		clearCurrent() {
			update((s) => ({ ...s, currentProject: null }));
		}
	};
}

export const projects = createProjectsStore();
