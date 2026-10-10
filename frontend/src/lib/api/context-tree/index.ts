export * from './types';
export * from './context-tree';

import * as contextTreeApi from './context-tree';

export const api = {
  // Context Tree
  getContextTree: contextTreeApi.getContextTree,
  searchContextTree: contextTreeApi.searchContextTree,
  loadContextItem: contextTreeApi.loadContextItem,
  getContextStats: contextTreeApi.getContextStats,

  // Loading Rules
  getLoadingRules: contextTreeApi.getLoadingRules,

  // Context Sessions
  createContextSession: contextTreeApi.createContextSession,
  getContextSession: contextTreeApi.getContextSession,
  updateContextSession: contextTreeApi.updateContextSession,
  endContextSession: contextTreeApi.endContextSession,
};

export default api;
