export * from './types';
export * from './ai';

import * as aiApi from './ai';

export const api = {
  // AI Providers & Models
  getAIProviders: aiApi.getAIProviders,
  updateAIProvider: aiApi.updateAIProvider,
  getAllModels: aiApi.getAllModels,
  getLocalModels: aiApi.getLocalModels,
  pullModel: aiApi.pullModel,
  warmupModel: aiApi.warmupModel,
  getAISystemInfo: aiApi.getAISystemInfo,
  saveAPIKey: aiApi.saveAPIKey,

  // Agent Prompts
  getAgentPrompts: aiApi.getAgentPrompts,
  getAgentPrompt: aiApi.getAgentPrompt,

  // MCP Tools
  getTools: aiApi.getTools,
  executeTool: aiApi.executeTool,

  // Custom Agents - CRUD
  getCustomAgents: aiApi.getCustomAgents,
  getCustomAgent: aiApi.getCustomAgent,
  createCustomAgent: aiApi.createCustomAgent,
  updateCustomAgent: aiApi.updateCustomAgent,
  deleteCustomAgent: aiApi.deleteCustomAgent,

  // Custom Agents - Filtering
  getAgentsByCategory: aiApi.getAgentsByCategory,

  // Agent Presets
  getAgentPresets: aiApi.getAgentPresets,
  getAgentPreset: aiApi.getAgentPreset,
  createFromPreset: aiApi.createFromPreset,

  // Agent Testing
  testAgent: aiApi.testAgent,
  testSandbox: aiApi.testSandbox,
};

export default api;
