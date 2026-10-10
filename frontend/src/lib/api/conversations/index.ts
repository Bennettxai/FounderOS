export * from './types';
export * from './conversations';

import * as conversationsApi from './conversations';

export const api = {
  getConversations: conversationsApi.getConversations,
  getConversation: conversationsApi.getConversation,
  createConversation: conversationsApi.createConversation,
  deleteConversation: conversationsApi.deleteConversation,
  updateConversation: conversationsApi.updateConversation,
  getConversationsByContext: conversationsApi.getConversationsByContext,
  sendMessage: conversationsApi.sendMessage,
  searchConversations: conversationsApi.searchConversations,
};

export default api;
