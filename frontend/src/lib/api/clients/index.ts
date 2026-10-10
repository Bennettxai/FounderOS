export * from "./types";
export * from "./clients";

import * as clientsApi from "./clients";

export const api = {
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
};

export default api;
