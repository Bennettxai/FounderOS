import { writable, get } from "svelte/store";
import {
  api,
  type ClientListResponse,
  type ClientDetailResponse,
  type ClientStatus,
  type ClientType,
  type CreateClientData,
  type UpdateClientData,
  type ContactResponse,
  type CreateContactData,
  type UpdateContactData,
  type InteractionResponse,
  type CreateInteractionData,
  type DealResponse,
  type DealStage,
  type CreateDealData,
  type UpdateDealData,
} from "$lib/api/clients";
import { getAllDeals, updateDealStage } from "$lib/api/deals";

export type ViewMode = "table" | "cards" | "kanban";

interface ClientFilters {
  status: ClientStatus | null;
  type: ClientType | null;
  search: string;
  tags: string[];
}

interface ClientsState {
  clients: ClientListResponse[];
  currentClient: ClientDetailResponse | null;
  allDeals: DealResponse[];
  loading: boolean;
  error: string | null;
  filters: ClientFilters;
  viewMode: ViewMode;
}

function createClientsStore() {
  const store = writable<ClientsState>({
    clients: [],
    currentClient: null,
    allDeals: [],
    loading: false,
    error: null,
    filters: {
      status: null,
      type: null,
      search: "",
      tags: [],
    },
    viewMode: "table",
  });
  const { subscribe, update } = store;

  return {
    subscribe,

    // ============ Client Methods ============

    async loadClients() {
      update((s) => ({ ...s, loading: true, error: null }));
      try {
        const state = get(store);
        const clients = await api.getClients({
          status: state.filters.status || undefined,
          type: state.filters.type || undefined,
          search: state.filters.search || undefined,
          tags: state.filters.tags.length > 0 ? state.filters.tags : undefined,
        });
        update((s) => ({ ...s, clients, loading: false }));
      } catch (error) {
        console.error("Failed to load clients:", error);
        update((s) => ({
          ...s,
          loading: false,
          error:
            error instanceof Error ? error.message : "Failed to load clients",
        }));
      }
    },

    async loadClient(id: string) {
      update((s) => ({ ...s, loading: true, error: null }));
      try {
        const client = await api.getClient(id);
        update((s) => ({ ...s, currentClient: client, loading: false }));
        return client;
      } catch (error) {
        console.error("Failed to load client:", error);
        update((s) => ({
          ...s,
          loading: false,
          error:
            error instanceof Error ? error.message : "Failed to load client",
        }));
        return null;
      }
    },

    async createClient(data: CreateClientData) {
      try {
        const client = await api.createClient(data);
        // Convert to list response and add to list
        const listClient: ClientListResponse = {
          id: client.id,
          name: client.name,
          type: client.type,
          email: client.email,
          phone: client.phone,
          status: client.status,
          source: client.source,
          assigned_to: client.assigned_to,
          lifetime_value: client.lifetime_value,
          tags: client.tags,
          created_at: client.created_at,
          last_contacted_at: client.last_contacted_at,
          contacts_count: 0,
          interactions_count: 0,
          deals_count: 0,
          active_deals_value: 0,
        };
        update((s) => ({ ...s, clients: [listClient, ...s.clients] }));
        return client;
      } catch (error) {
        console.error("Failed to create client:", error);
        throw error;
      }
    },

    async updateClient(id: string, data: UpdateClientData) {
      try {
        const client = await api.updateClient(id, data);
        update((s) => ({
          ...s,
          clients: s.clients.map((c) =>
            c.id === id
              ? {
                  ...c,
                  name: client.name,
                  type: client.type,
                  email: client.email,
                  phone: client.phone,
                  status: client.status,
                  source: client.source,
                  assigned_to: client.assigned_to,
                  lifetime_value: client.lifetime_value,
                  tags: client.tags,
                  last_contacted_at: client.last_contacted_at,
                }
              : c,
          ),
          currentClient:
            s.currentClient?.id === id
              ? { ...s.currentClient, ...client }
              : s.currentClient,
        }));
        return client;
      } catch (error) {
        console.error("Failed to update client:", error);
        throw error;
      }
    },

    async updateClientStatus(id: string, status: ClientStatus) {
      try {
        const client = await api.updateClientStatus(id, status);
        update((s) => ({
          ...s,
          clients: s.clients.map((c) =>
            c.id === id ? { ...c, status: client.status } : c,
          ),
          currentClient:
            s.currentClient?.id === id
              ? { ...s.currentClient, status: client.status }
              : s.currentClient,
        }));
        return client;
      } catch (error) {
        console.error("Failed to update client status:", error);
        throw error;
      }
    },

    async deleteClient(id: string) {
      try {
        await api.deleteClient(id);
        update((s) => ({
          ...s,
          clients: s.clients.filter((c) => c.id !== id),
          currentClient: s.currentClient?.id === id ? null : s.currentClient,
        }));
      } catch (error) {
        console.error("Failed to delete client:", error);
        throw error;
      }
    },

    // ============ Contact Methods ============

    async createContact(clientId: string, data: CreateContactData) {
      try {
        const contact = await api.createContact(clientId, data);
        update((s) => ({
          ...s,
          currentClient:
            s.currentClient?.id === clientId
              ? {
                  ...s.currentClient,
                  contacts: [...s.currentClient.contacts, contact],
                }
              : s.currentClient,
          clients: s.clients.map((c) =>
            c.id === clientId
              ? { ...c, contacts_count: c.contacts_count + 1 }
              : c,
          ),
        }));
        return contact;
      } catch (error) {
        console.error("Failed to create contact:", error);
        throw error;
      }
    },

    async updateContact(
      clientId: string,
      contactId: string,
      data: UpdateContactData,
    ) {
      try {
        const contact = await api.updateContact(clientId, contactId, data);
        update((s) => ({
          ...s,
          currentClient:
            s.currentClient?.id === clientId
              ? {
                  ...s.currentClient,
                  contacts: s.currentClient.contacts.map((c) =>
                    c.id === contactId ? contact : c,
                  ),
                }
              : s.currentClient,
        }));
        return contact;
      } catch (error) {
        console.error("Failed to update contact:", error);
        throw error;
      }
    },

    async deleteContact(clientId: string, contactId: string) {
      try {
        await api.deleteContact(clientId, contactId);
        update((s) => ({
          ...s,
          currentClient:
            s.currentClient?.id === clientId
              ? {
                  ...s.currentClient,
                  contacts: s.currentClient.contacts.filter(
                    (c) => c.id !== contactId,
                  ),
                }
              : s.currentClient,
          clients: s.clients.map((c) =>
            c.id === clientId
              ? { ...c, contacts_count: Math.max(0, c.contacts_count - 1) }
              : c,
          ),
        }));
      } catch (error) {
        console.error("Failed to delete contact:", error);
        throw error;
      }
    },

    // ============ Interaction Methods ============

    async createInteraction(clientId: string, data: CreateInteractionData) {
      try {
        const interaction = await api.createInteraction(clientId, data);
        update((s) => ({
          ...s,
          currentClient:
            s.currentClient?.id === clientId
              ? {
                  ...s.currentClient,
                  interactions: [interaction, ...s.currentClient.interactions],
                  last_contacted_at: interaction.occurred_at,
                }
              : s.currentClient,
          clients: s.clients.map((c) =>
            c.id === clientId
              ? {
                  ...c,
                  interactions_count: c.interactions_count + 1,
                  last_contacted_at: interaction.occurred_at,
                }
              : c,
          ),
        }));
        return interaction;
      } catch (error) {
        console.error("Failed to create interaction:", error);
        throw error;
      }
    },

    // ============ Deal Methods ============

    async loadAllDeals(stage?: DealStage) {
      update((s) => ({ ...s, loading: true, error: null }));
      try {
        const deals = await getAllDeals(stage);
        update((s) => ({ ...s, allDeals: deals, loading: false }));
      } catch (error) {
        console.error("Failed to load deals:", error);
        update((s) => ({
          ...s,
          loading: false,
          error:
            error instanceof Error ? error.message : "Failed to load deals",
        }));
      }
    },

    async createDeal(clientId: string, data: CreateDealData) {
      try {
        const deal = await api.createDeal(clientId, data);
        update((s) => ({
          ...s,
          currentClient:
            s.currentClient?.id === clientId
              ? { ...s.currentClient, deals: [...s.currentClient.deals, deal] }
              : s.currentClient,
          clients: s.clients.map((c) =>
            c.id === clientId
              ? {
                  ...c,
                  deals_count: c.deals_count + 1,
                  active_deals_value:
                    deal.stage !== "closed_won" && deal.stage !== "closed_lost"
                      ? c.active_deals_value + deal.value
                      : c.active_deals_value,
                }
              : c,
          ),
          allDeals: [...s.allDeals, deal],
        }));
        return deal;
      } catch (error) {
        console.error("Failed to create deal:", error);
        throw error;
      }
    },

    async updateDeal(clientId: string, dealId: string, data: UpdateDealData) {
      try {
        const deal = await api.updateDeal(clientId, dealId, data);
        update((s) => ({
          ...s,
          currentClient:
            s.currentClient?.id === clientId
              ? {
                  ...s.currentClient,
                  deals: s.currentClient.deals.map((d) =>
                    d.id === dealId ? deal : d,
                  ),
                }
              : s.currentClient,
          allDeals: s.allDeals.map((d) => (d.id === dealId ? deal : d)),
        }));
        return deal;
      } catch (error) {
        console.error("Failed to update deal:", error);
        throw error;
      }
    },

    async updateDealStage(dealId: string, stage: DealStage) {
      try {
        const deal = await updateDealStage(dealId, stage);
        update((s) => ({
          ...s,
          currentClient: s.currentClient
            ? {
                ...s.currentClient,
                deals: s.currentClient.deals.map((d) =>
                  d.id === dealId ? deal : d,
                ),
              }
            : null,
          allDeals: s.allDeals.map((d) => (d.id === dealId ? deal : d)),
        }));
        return deal;
      } catch (error) {
        console.error("Failed to update deal stage:", error);
        throw error;
      }
    },

    // ============ Filter & View Methods ============

    setFilters(filters: Partial<ClientFilters>) {
      update((s) => ({
        ...s,
        filters: { ...s.filters, ...filters },
      }));
    },

    clearFilters() {
      update((s) => ({
        ...s,
        filters: {
          status: null,
          type: null,
          search: "",
          tags: [],
        },
      }));
    },

    setViewMode(mode: ViewMode) {
      update((s) => ({ ...s, viewMode: mode }));
    },

    clearCurrent() {
      update((s) => ({ ...s, currentClient: null }));
    },

    clearError() {
      update((s) => ({ ...s, error: null }));
    },
  };
}

export const clients = createClientsStore();

// Status colors for UI
export const statusColors: Record<ClientStatus, string> = {
  lead: "bg-purple-50 text-purple-700",
  prospect: "bg-blue-50 text-blue-700",
  active: "bg-emerald-50 text-emerald-700",
  inactive: "bg-gray-100 text-gray-600",
  churned: "bg-red-50 text-red-700",
};

export const statusLabels: Record<ClientStatus, string> = {
  lead: "Lead",
  prospect: "Prospect",
  active: "Active",
  inactive: "Inactive",
  churned: "Churned",
};

export const dealStageColors: Record<DealStage, string> = {
  qualification: "bg-gray-100 text-gray-700",
  proposal: "bg-blue-50 text-blue-700",
  negotiation: "bg-amber-50 text-amber-700",
  closed_won: "bg-emerald-50 text-emerald-700",
  closed_lost: "bg-red-50 text-red-700",
};

export const dealStageLabels: Record<DealStage, string> = {
  qualification: "Qualification",
  proposal: "Proposal",
  negotiation: "Negotiation",
  closed_won: "Won",
  closed_lost: "Lost",
};

export const interactionTypeLabels: Record<string, string> = {
  call: "Call",
  email: "Email",
  meeting: "Meeting",
  note: "Note",
};

export const interactionTypeIcons: Record<string, string> = {
  call: "📞",
  email: "✉️",
  meeting: "👥",
  note: "📝",
};
