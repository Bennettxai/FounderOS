// Root API index: re-export domain modules and provide a backwards-compatible `api` and `apiClient` shim.
export * from "./base";
export * from "./projects";
export * from "./conversations";
// Note: contexts module also exports a Block type, which conflicts with conversations' Block
// We skip the wildcard export to avoid ambiguity. Import from './contexts' directly if needed.
// export * from './contexts';
// Export specific types from contexts that don't conflict
export type {
  Context,
  ContextListItem,
  PropertySchema,
  Block as ContextBlock,
} from "./contexts/types";
export * from "./clients";
export * from "./calendar";
export * from "./team";
export * from "./dashboard";
export * from "./nodes";
export * from "./deals";
export * from "./daily";
export * from "./settings";
export * from "./artifacts";
export * from "./integrations";
export * from "./voice-notes";
export * from "./usage";
export * from "./ai";
export * from "./profile";
export * from "./memory";
export * from "./context-tree";
export * from "./gmail";
// CRM module - skip wildcard export to avoid conflicts with clients/dashboard types
// Import CRM types directly: import { type Company, type Deal } from '$lib/api/crm';
// Export specific non-conflicting CRM types
export type {
  Company,
  CompaniesResponse,
  CreateCompanyData,
  UpdateCompanyData,
  Pipeline,
  PipelinesResponse,
  CreatePipelineData,
  UpdatePipelineData,
  PipelineStage,
  StagesResponse,
  CreateStageData,
  UpdateStageData,
  Deal,
  DealsResponse,
  DealStats,
  CRMActivity,
  ActivitiesResponse,
  ContactCompanyRelation,
  ContactsResponse,
  LinkContactData,
} from "./crm";
// Re-export CRM deal types with CRM prefix to avoid conflicts
export type {
  CreateDealData as CRMCreateDealData,
  UpdateDealData as CRMUpdateDealData,
  ActivityType as CRMActivityType,
  CreateActivityData as CRMCreateActivityData,
} from "./crm";
export * from "./osa";

// Pedro Tasks API modules
export * from "./learning";
export * from "./pedro-documents";
export * from "./intelligence";
export * from "./app-profiles";

import * as projectsApi from "./projects";
import * as conversationsApi from "./conversations";
import * as contextsApi from "./contexts";
import * as clientsApi from "./clients";
import * as calendarApi from "./calendar";
import * as teamApi from "./team";
import * as dashboardApi from "./dashboard";
import * as nodesApi from "./nodes";
import * as dealsApi from "./deals";
import * as dailyApi from "./daily";
import * as settingsApi from "./settings";
import * as artifactsApi from "./artifacts";
import * as integrationsApi from "./integrations";
import * as voiceNotesApi from "./voice-notes";
import * as usageApi from "./usage";
import * as aiApi from "./ai";
import * as profileApi from "./profile";
import * as memoryApi from "./memory";
import * as contextTreeApi from "./context-tree";
import * as crmApi from "./crm";
import * as osaApi from "./osa";
import * as base from "./base";

// Consolidated api object exposing common domain functions for backward compatibility.
export const api = {
  // Projects
  getProjects: projectsApi.getProjects,
  getProject: projectsApi.getProject,
  createProject: projectsApi.createProject,
  updateProject: projectsApi.updateProject,
  deleteProject: projectsApi.deleteProject,
  addProjectNote: projectsApi.addProjectNote,
  uploadProjectFile: projectsApi.uploadProjectFile,
  addProjectFileLink: projectsApi.addProjectFileLink,
  removeProjectFileLink: projectsApi.removeProjectFileLink,

  // Conversations
  getConversations: conversationsApi.getConversations,
  getConversation: conversationsApi.getConversation,
  createConversation: conversationsApi.createConversation,
  deleteConversation: conversationsApi.deleteConversation,
  updateConversation: conversationsApi.updateConversation,
  getConversationsByContext: conversationsApi.getConversationsByContext,
  sendMessage: conversationsApi.sendMessage,
  searchConversations: conversationsApi.searchConversations,

  // Contexts
  getContexts: contextsApi.getContexts,
  getContext: contextsApi.getContext,
  createContext: contextsApi.createContext,
  updateContext: contextsApi.updateContext,
  updateContextBlocks: contextsApi.updateContextBlocks,
  enableContextSharing: contextsApi.enableContextSharing,
  disableContextSharing: contextsApi.disableContextSharing,
  getPublicContext: contextsApi.getPublicContext,
  duplicateContext: contextsApi.duplicateContext,
  archiveContext: contextsApi.archiveContext,
  unarchiveContext: contextsApi.unarchiveContext,
  deleteContext: contextsApi.deleteContext,
  aggregateContext: contextsApi.aggregateContext,

  // Clients
  getClients: clientsApi.getClients,
  getClient: clientsApi.getClient,
  getClientBoard: clientsApi.getClientBoard,
  createClient: clientsApi.createClient,
  updateClient: clientsApi.updateClient,
  updateClientStatus: clientsApi.updateClientStatus,
  deleteClient: clientsApi.deleteClient,
  getClientContacts: clientsApi.getClientContacts,
  createContact: clientsApi.createContact,
  updateContact: clientsApi.updateContact,
  deleteContact: clientsApi.deleteContact,
  getClientInteractions: clientsApi.getClientInteractions,
  createInteraction: clientsApi.createInteraction,
  getClientDeals: clientsApi.getClientDeals,
  createDeal: clientsApi.createDeal,
  updateDeal: clientsApi.updateDeal,

  // Calendar
  getCalendarEvents: calendarApi.getCalendarEvents,
  getCalendarEvent: calendarApi.getCalendarEvent,
  createCalendarEvent: calendarApi.createCalendarEvent,
  updateCalendarEvent: calendarApi.updateCalendarEvent,
  deleteCalendarEvent: calendarApi.deleteCalendarEvent,
  syncCalendar: calendarApi.syncCalendar,
  getTodayEvents: calendarApi.getTodayEvents,
  getUpcomingEvents: calendarApi.getUpcomingEvents,
  getGoogleConnectionStatus: integrationsApi.getGoogleConnectionStatus,

  // Team
  getTeamMembers: teamApi.getTeamMembers,
  getTeamMember: teamApi.getTeamMember,
  createTeamMember: teamApi.createTeamMember,
  updateTeamMember: teamApi.updateTeamMember,
  deleteTeamMember: teamApi.deleteTeamMember,
  updateTeamMemberStatus: teamApi.updateTeamMemberStatus,
  updateTeamMemberCapacity: teamApi.updateTeamMemberCapacity,

  // Dashboard
  getDashboardSummary: dashboardApi.getDashboardSummary,
  getFocusItems: dashboardApi.getFocusItems,
  createFocusItem: dashboardApi.createFocusItem,
  updateFocusItem: dashboardApi.updateFocusItem,
  deleteFocusItem: dashboardApi.deleteFocusItem,
  getTasks: dashboardApi.getTasks,
  createTask: dashboardApi.createTask,
  updateTask: dashboardApi.updateTask,
  toggleTask: dashboardApi.toggleTask,
  deleteTask: dashboardApi.deleteTask,

  // Nodes
  getNodes: nodesApi.getNodes,
  getNodeTree: nodesApi.getNodeTree,
  getActiveNode: nodesApi.getActiveNode,
  getNode: nodesApi.getNode,
  createNode: nodesApi.createNode,
  updateNode: nodesApi.updateNode,
  activateNode: nodesApi.activateNode,
  deactivateNode: nodesApi.deactivateNode,
  deleteNode: nodesApi.deleteNode,
  getNodeChildren: nodesApi.getNodeChildren,
  reorderNode: nodesApi.reorderNode,

  // Deals
  getAllDeals: dealsApi.getAllDeals,
  updateDealStage: dealsApi.updateDealStage,

  // Daily Logs
  getDailyLogs: dailyApi.getDailyLogs,
  getTodayLog: dailyApi.getTodayLog,
  getDailyLogByDate: dailyApi.getDailyLogByDate,
  saveDailyLog: dailyApi.saveDailyLog,
  updateDailyLog: dailyApi.updateDailyLog,
  deleteDailyLog: dailyApi.deleteDailyLog,

  // Settings
  getSettings: settingsApi.getSettings,
  updateSettings: settingsApi.updateSettings,
  getSystemInfo: settingsApi.getSystemInfo,

  // Artifacts
  getArtifacts: artifactsApi.getArtifacts,
  getArtifact: artifactsApi.getArtifact,
  createArtifact: artifactsApi.createArtifact,
  updateArtifact: artifactsApi.updateArtifact,
  deleteArtifact: artifactsApi.deleteArtifact,
  linkArtifact: artifactsApi.linkArtifact,
  getArtifactVersions: artifactsApi.getArtifactVersions,
  restoreArtifactVersion: artifactsApi.restoreArtifactVersion,

  // Integrations - Google
  initiateGoogleAuth: integrationsApi.initiateGoogleAuth,
  disconnectGoogle: integrationsApi.disconnectGoogle,
  // Integrations - Slack
  initiateSlackAuth: integrationsApi.initiateSlackAuth,
  getSlackConnectionStatus: integrationsApi.getSlackConnectionStatus,
  disconnectSlack: integrationsApi.disconnectSlack,
  getSlackChannels: integrationsApi.getSlackChannels,
  getSlackNotifications: integrationsApi.getSlackNotifications,
  getSlackMessages: integrationsApi.getSlackMessages,
  sendSlackMessage: integrationsApi.sendSlackMessage,
  syncSlackChannels: integrationsApi.syncSlackChannels,
  syncSlackMessages: integrationsApi.syncSlackMessages,
  // Integrations - Notion
  initiateNotionAuth: integrationsApi.initiateNotionAuth,
  getNotionConnectionStatus: integrationsApi.getNotionConnectionStatus,
  disconnectNotion: integrationsApi.disconnectNotion,
  getNotionDatabases: integrationsApi.getNotionDatabases,
  getNotionPages: integrationsApi.getNotionPages,
  syncNotionDatabase: integrationsApi.syncNotionDatabase,

  // Voice Notes
  getVoiceNotes: voiceNotesApi.getVoiceNotes,
  uploadVoiceNote: voiceNotesApi.uploadVoiceNote,
  getVoiceNoteAudio: voiceNotesApi.getVoiceNoteAudio,
  deleteVoiceNote: voiceNotesApi.deleteVoiceNote,
  retranscribeVoiceNote: voiceNotesApi.retranscribeVoiceNote,

  // Usage Analytics
  getPlanUsage: usageApi.getPlanUsage,
  getPlanLimits: usageApi.getPlanLimits,
  getUsageSummary: usageApi.getUsageSummary,
  getUsageByProvider: usageApi.getUsageByProvider,
  getUsageByModel: usageApi.getUsageByModel,
  getUsageByAgent: usageApi.getUsageByAgent,
  getUsageTrend: usageApi.getUsageTrend,
  getMCPUsage: usageApi.getMCPUsage,

  // AI Configuration
  getAIProviders: aiApi.getAIProviders,
  updateAIProvider: aiApi.updateAIProvider,
  getAllModels: aiApi.getAllModels,
  getLocalModels: aiApi.getLocalModels,
  pullModel: aiApi.pullModel,
  warmupModel: aiApi.warmupModel,
  getAISystemInfo: aiApi.getAISystemInfo,
  saveAPIKey: aiApi.saveAPIKey,
  getAgentPrompts: aiApi.getAgentPrompts,
  getAgentPrompt: aiApi.getAgentPrompt,
  getTools: aiApi.getTools,
  executeTool: aiApi.executeTool,

  // OSA-5 Orchestration
  checkOSAHealth: osaApi.checkOSAHealth,
  generateApp: osaApi.generateApp,
  getQueueItemStatus: osaApi.getQueueItemStatus,
  getWorkspaces: osaApi.getWorkspaces,
  streamAppGeneration: osaApi.streamAppGeneration,
  parseGenerationEvent: osaApi.parseGenerationEvent,
  cancelGeneration: osaApi.cancelGeneration,

  // Profile
  updateProfile: profileApi.updateProfile,
  uploadProfilePhoto: profileApi.uploadProfilePhoto,
  deleteProfilePhoto: profileApi.deleteProfilePhoto,

  // Memory (Episodic Memory System)
  getMemories: memoryApi.getMemories,
  getMemory: memoryApi.getMemory,
  createMemory: memoryApi.createMemory,
  updateMemory: memoryApi.updateMemory,
  deleteMemory: memoryApi.deleteMemory,
  pinMemory: memoryApi.pinMemory,
  searchMemories: memoryApi.searchMemories,
  getRelevantMemories: memoryApi.getRelevantMemories,
  getProjectMemories: memoryApi.getProjectMemories,
  getNodeMemories: memoryApi.getNodeMemories,
  getMemoryStats: memoryApi.getMemoryStats,
  getUserFacts: memoryApi.getUserFacts,
  updateUserFact: memoryApi.updateUserFact,
  confirmUserFact: memoryApi.confirmUserFact,
  rejectUserFact: memoryApi.rejectUserFact,
  deleteUserFact: memoryApi.deleteUserFact,

  // Context Tree (Hierarchical Context Management)
  getContextTree: contextTreeApi.getContextTree,
  searchContextTree: contextTreeApi.searchContextTree,
  loadContextItem: contextTreeApi.loadContextItem,
  getContextStats: contextTreeApi.getContextStats,
  getLoadingRules: contextTreeApi.getLoadingRules,
  createContextSession: contextTreeApi.createContextSession,
  getContextSession: contextTreeApi.getContextSession,
  updateContextSession: contextTreeApi.updateContextSession,
  endContextSession: contextTreeApi.endContextSession,

  // CRM - Companies
  getCompanies: crmApi.getCompanies,
  getCompany: crmApi.getCompany,
  createCompany: crmApi.createCompany,
  updateCompany: crmApi.updateCompany,
  deleteCompany: crmApi.deleteCompany,
  searchCompanies: crmApi.searchCompanies,
  getCompanyContacts: crmApi.getCompanyContacts,
  linkContactToCompany: crmApi.linkContactToCompany,
  unlinkContactFromCompany: crmApi.unlinkContactFromCompany,

  // CRM - Pipelines
  getPipelines: crmApi.getPipelines,
  getPipeline: crmApi.getPipeline,
  createPipeline: crmApi.createPipeline,
  updatePipeline: crmApi.updatePipeline,
  deletePipeline: crmApi.deletePipeline,

  // CRM - Pipeline Stages
  getPipelineStages: crmApi.getPipelineStages,
  createPipelineStage: crmApi.createPipelineStage,
  updatePipelineStage: crmApi.updatePipelineStage,
  deletePipelineStage: crmApi.deletePipelineStage,
  reorderPipelineStages: crmApi.reorderPipelineStages,

  // CRM - Deals
  getCRMDeals: crmApi.getDeals,
  getCRMDeal: crmApi.getDeal,
  createCRMDeal: crmApi.createDeal,
  updateCRMDeal: crmApi.updateDeal,
  deleteCRMDeal: crmApi.deleteDeal,
  moveDealToStage: crmApi.moveDealToStage,
  updateDealStatus: crmApi.updateDealStatus,
  getDealStats: crmApi.getDealStats,

  // CRM - Activities
  getCRMActivities: crmApi.getActivities,
  getDealActivities: crmApi.getDealActivities,
  createCRMActivity: crmApi.createActivity,
  completeCRMActivity: crmApi.completeActivity,
  deleteCRMActivity: crmApi.deleteActivity,

  // Raw helper to inspect base URL if needed
  apiBase: base.getApiBaseUrl,
};

export const apiClient = base.raw;

export default api;
