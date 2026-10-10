export * from './types';
export * from './memory';

import * as memoryApi from './memory';

export const api = {
  // Memory CRUD
  getMemories: memoryApi.getMemories,
  getMemory: memoryApi.getMemory,
  createMemory: memoryApi.createMemory,
  updateMemory: memoryApi.updateMemory,
  deleteMemory: memoryApi.deleteMemory,
  pinMemory: memoryApi.pinMemory,

  // Memory Search
  searchMemories: memoryApi.searchMemories,
  getRelevantMemories: memoryApi.getRelevantMemories,

  // Memory Scoped
  getProjectMemories: memoryApi.getProjectMemories,
  getNodeMemories: memoryApi.getNodeMemories,

  // Memory Stats
  getMemoryStats: memoryApi.getMemoryStats,

  // User Facts
  getUserFacts: memoryApi.getUserFacts,
  updateUserFact: memoryApi.updateUserFact,
  confirmUserFact: memoryApi.confirmUserFact,
  rejectUserFact: memoryApi.rejectUserFact,
  deleteUserFact: memoryApi.deleteUserFact,
};

export default api;
