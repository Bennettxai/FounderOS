import { writable } from 'svelte/store';
import {
	api,
	type TeamMemberListResponse,
	type TeamMemberDetailResponse,
	type CreateTeamMemberData,
	type UpdateTeamMemberData
} from '$lib/api/team';

interface TeamState {
	members: TeamMemberListResponse[];
	currentMember: TeamMemberDetailResponse | null;
	loading: boolean;
	error: string | null;
}

function createTeamStore() {
	const { subscribe, update } = writable<TeamState>({
		members: [],
		currentMember: null,
		loading: false,
		error: null
	});

	return {
		subscribe,

		async loadMembers(options?: { status?: string; page?: number; pageSize?: number }) {
			update((s) => ({ ...s, loading: true, error: null }));
			try {
				const members = await api.getTeamMembers(options?.status);
				update((s) => ({ ...s, members, loading: false }));
			} catch (error) {
				console.error('Failed to load team members:', error);
				update((s) => ({
					...s,
					loading: false,
					error: error instanceof Error ? error.message : 'Failed to load team members'
				}));
			}
		},

		async loadMember(id: string) {
			update((s) => ({ ...s, loading: true, error: null }));
			try {
				const member = await api.getTeamMember(id);
				update((s) => ({ ...s, currentMember: member, loading: false }));
				return member;
			} catch (error) {
				console.error('Failed to load team member:', error);
				update((s) => ({
					...s,
					loading: false,
					error: error instanceof Error ? error.message : 'Failed to load team member'
				}));
				return null;
			}
		},

		async createMember(data: CreateTeamMemberData) {
			try {
				const member = await api.createTeamMember(data);
				// Convert response to list format
				const listMember: TeamMemberListResponse = {
					id: member.id,
					name: member.name,
					email: member.email,
					role: member.role,
					avatar_url: member.avatar_url,
					status: member.status,
					capacity: member.capacity,
					manager_id: member.manager_id,
					active_projects: 0,
					open_tasks: 0,
					joined_at: member.joined_at
				};
				update((s) => ({ ...s, members: [...s.members, listMember] }));
				return member;
			} catch (error) {
				console.error('Failed to create team member:', error);
				throw error;
			}
		},

		async updateMember(id: string, data: UpdateTeamMemberData) {
			try {
				const member = await api.updateTeamMember(id, data);
				update((s) => ({
					...s,
					members: s.members.map((m) =>
						m.id === id
							? {
									...m,
									name: member.name,
									email: member.email,
									role: member.role,
									avatar_url: member.avatar_url,
									status: member.status,
									capacity: member.capacity,
									manager_id: member.manager_id
								}
							: m
					),
					currentMember:
						s.currentMember?.id === id
							? { ...s.currentMember, ...member }
							: s.currentMember
				}));
				return member;
			} catch (error) {
				console.error('Failed to update team member:', error);
				throw error;
			}
		},

		async deleteMember(id: string) {
			try {
				await api.deleteTeamMember(id);
				update((s) => ({
					...s,
					members: s.members.filter((m) => m.id !== id),
					currentMember: s.currentMember?.id === id ? null : s.currentMember
				}));
			} catch (error) {
				console.error('Failed to delete team member:', error);
				throw error;
			}
		},

		async updateStatus(id: string, status: string) {
			try {
				const member = await api.updateTeamMemberStatus(id, status);
				update((s) => ({
					...s,
					members: s.members.map((m) =>
						m.id === id ? { ...m, status: member.status } : m
					),
					currentMember:
						s.currentMember?.id === id
							? { ...s.currentMember, status: member.status }
							: s.currentMember
				}));
				return member;
			} catch (error) {
				console.error('Failed to update member status:', error);
				throw error;
			}
		},

		async updateCapacity(id: string, capacity: number) {
			try {
				const member = await api.updateTeamMemberCapacity(id, capacity);
				update((s) => ({
					...s,
					members: s.members.map((m) =>
						m.id === id
							? { ...m, capacity: member.capacity, status: member.status }
							: m
					),
					currentMember:
						s.currentMember?.id === id
							? { ...s.currentMember, capacity: member.capacity, status: member.status }
							: s.currentMember
				}));
				return member;
			} catch (error) {
				console.error('Failed to update member capacity:', error);
				throw error;
			}
		},

		clearCurrent() {
			update((s) => ({ ...s, currentMember: null }));
		},

		clearError() {
			update((s) => ({ ...s, error: null }));
		}
	};
}

export const team = createTeamStore();
