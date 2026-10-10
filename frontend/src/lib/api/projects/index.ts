// Entrypoint for the `projects` domain folder
export * from "./types";
export * from "./projects";
export * from "./members";
export * from "./templates";

import * as projectsApi from "./projects";
import * as membersApi from "./members";
import * as templatesApi from "./templates";

export const api = {
  getProjects: projectsApi.getProjects,
  getProject: projectsApi.getProject,
  createProject: projectsApi.createProject,
  updateProject: projectsApi.updateProject,
  deleteProject: projectsApi.deleteProject,
  addProjectNote: projectsApi.addProjectNote,
  uploadProjectFile: projectsApi.uploadProjectFile,
  addProjectFileLink: projectsApi.addProjectFileLink,
  removeProjectFileLink: projectsApi.removeProjectFileLink,
  // Project templates
  getProjectTemplates: templatesApi.getProjectTemplates,
  createProjectFromTemplate: templatesApi.createProjectFromTemplate,
  // Project members
  listProjectMembers: membersApi.listProjectMembers,
  addProjectMember: membersApi.addProjectMember,
  updateProjectMemberRole: membersApi.updateProjectMemberRole,
  removeProjectMember: membersApi.removeProjectMember,
  checkProjectAccess: membersApi.checkProjectAccess,
};

export default api;
