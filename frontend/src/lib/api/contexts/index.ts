export * from './types';
export * from './contexts';

import * as contextsApi from './contexts';

export const api = {
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
};

export default api;
