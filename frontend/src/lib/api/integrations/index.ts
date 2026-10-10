export * from './types';
export * from './integrations';

import * as integrationsApi from './integrations';

export const api = {
  // Generic
  getAllIntegrationsStatus: integrationsApi.getAllIntegrationsStatus,
  getIntegrationStatus: integrationsApi.getIntegrationStatus,
  initiateAuth: integrationsApi.initiateAuth,
  disconnectIntegration: integrationsApi.disconnectIntegration,
  syncIntegration: integrationsApi.syncIntegration,
  // Google
  initiateGoogleAuth: integrationsApi.initiateGoogleAuth,
  getGoogleConnectionStatus: integrationsApi.getGoogleConnectionStatus,
  disconnectGoogle: integrationsApi.disconnectGoogle,
  // Slack
  initiateSlackAuth: integrationsApi.initiateSlackAuth,
  getSlackConnectionStatus: integrationsApi.getSlackConnectionStatus,
  disconnectSlack: integrationsApi.disconnectSlack,
  getSlackChannels: integrationsApi.getSlackChannels,
  getSlackNotifications: integrationsApi.getSlackNotifications,
  getSlackMessages: integrationsApi.getSlackMessages,
  sendSlackMessage: integrationsApi.sendSlackMessage,
  syncSlackChannels: integrationsApi.syncSlackChannels,
  syncSlackMessages: integrationsApi.syncSlackMessages,
  // Notion
  initiateNotionAuth: integrationsApi.initiateNotionAuth,
  getNotionConnectionStatus: integrationsApi.getNotionConnectionStatus,
  disconnectNotion: integrationsApi.disconnectNotion,
  getNotionDatabases: integrationsApi.getNotionDatabases,
  getNotionPages: integrationsApi.getNotionPages,
  syncNotionDatabase: integrationsApi.syncNotionDatabase,
  // HubSpot
  initiateHubSpotAuth: integrationsApi.initiateHubSpotAuth,
  getHubSpotConnectionStatus: integrationsApi.getHubSpotConnectionStatus,
  disconnectHubSpot: integrationsApi.disconnectHubSpot,
  syncHubSpot: integrationsApi.syncHubSpot,
  // GoHighLevel
  initiateGoHighLevelAuth: integrationsApi.initiateGoHighLevelAuth,
  getGoHighLevelConnectionStatus: integrationsApi.getGoHighLevelConnectionStatus,
  disconnectGoHighLevel: integrationsApi.disconnectGoHighLevel,
  syncGoHighLevel: integrationsApi.syncGoHighLevel,
  // Linear
  initiateLinearAuth: integrationsApi.initiateLinearAuth,
  getLinearConnectionStatus: integrationsApi.getLinearConnectionStatus,
  disconnectLinear: integrationsApi.disconnectLinear,
  syncLinear: integrationsApi.syncLinear,
  // Asana
  initiateAsanaAuth: integrationsApi.initiateAsanaAuth,
  getAsanaConnectionStatus: integrationsApi.getAsanaConnectionStatus,
  disconnectAsana: integrationsApi.disconnectAsana,
  syncAsana: integrationsApi.syncAsana,
  // File Import
  importFile: integrationsApi.importFile,
  getImportProgress: integrationsApi.getImportProgress,
  // MCP Connectors
  getMCPConnectors: integrationsApi.getMCPConnectors,
  createMCPConnector: integrationsApi.createMCPConnector,
  updateMCPConnector: integrationsApi.updateMCPConnector,
  deleteMCPConnector: integrationsApi.deleteMCPConnector,
  testMCPConnector: integrationsApi.testMCPConnector,
  // Sorx Integration Module
  getProviders: integrationsApi.getProviders,
  getProvider: integrationsApi.getProvider,
  getConnectedIntegrations: integrationsApi.getConnectedIntegrations,
  getUserIntegration: integrationsApi.getUserIntegration,
  updateIntegrationSettings: integrationsApi.updateIntegrationSettings,
  disconnectUserIntegration: integrationsApi.disconnectUserIntegration,
  triggerIntegrationSync: integrationsApi.triggerIntegrationSync,
  getModuleIntegrations: integrationsApi.getModuleIntegrations,
  // AI Model Preferences
  getAIModelPreferences: integrationsApi.getAIModelPreferences,
  updateAIModelPreferences: integrationsApi.updateAIModelPreferences,
  // Sorx Decisions
  getPendingDecisions: integrationsApi.getPendingDecisions,
  getPendingDecision: integrationsApi.getPendingDecision,
  respondToDecision: integrationsApi.respondToDecision,
  // Sorx Skill Execution
  triggerSkill: integrationsApi.triggerSkill,
  getSkillExecution: integrationsApi.getSkillExecution,
};

export default api;
