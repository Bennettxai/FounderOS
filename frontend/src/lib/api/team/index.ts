export * from './types';
export * from './team';

import * as teamApi from './team';

export const api = {
  getTeamMembers: teamApi.getTeamMembers,
  getTeamMember: teamApi.getTeamMember,
  createTeamMember: teamApi.createTeamMember,
  updateTeamMember: teamApi.updateTeamMember,
  deleteTeamMember: teamApi.deleteTeamMember,
  updateTeamMemberStatus: teamApi.updateTeamMemberStatus,
  updateTeamMemberCapacity: teamApi.updateTeamMemberCapacity,
};

export default api;
