import { request } from "../base";
import type {
  ClientListResponse,
  ClientResponse,
  ClientDetailResponse,
  CreateClientData,
  UpdateClientData,
  ContactResponse,
  CreateContactData,
  UpdateContactData,
  InteractionResponse,
  CreateInteractionData,
  DealResponse,
  CreateDealData,
  UpdateDealData,
  ClientBoardResponse,
} from "./types";

export async function getClients(filters?: {
  status?: string;
  type?: string;
  search?: string;
  tags?: string[];
}) {
  const params = new URLSearchParams();
  if (filters?.status) params.set("status", filters.status);
  if (filters?.type) params.set("type", filters.type);
  if (filters?.search) params.set("search", filters.search);
  if (filters?.tags) filters.tags.forEach((tag) => params.append("tags", tag));
  const query = params.toString();
  const raw = await request<Record<string, unknown>>(
    `/clients${query ? `?${query}` : ""}`,
  );

  // Backend returns paginated format { data: [...], pagination: {...} }
  if (
    raw &&
    typeof raw === "object" &&
    "data" in raw &&
    Array.isArray(raw.data)
  ) {
    return raw.data as ClientListResponse[];
  }
  if (Array.isArray(raw)) {
    return raw as unknown as ClientListResponse[];
  }
  return [];
}

export async function getClient(id: string) {
  return request<ClientDetailResponse>(`/clients/${id}`);
}

// Client Board - aggregated view (client + projects + tasks + team + deals + interactions)
export async function getClientBoard(id: string) {
  return request<ClientBoardResponse>(`/clients/${id}/board`);
}

export async function createClient(data: CreateClientData) {
  return request<ClientResponse>("/clients", { method: "POST", body: data });
}

export async function updateClient(id: string, data: UpdateClientData) {
  return request<ClientResponse>(`/clients/${id}`, {
    method: "PUT",
    body: data,
  });
}

export async function updateClientStatus(id: string, status: string) {
  return request<ClientResponse>(`/clients/${id}/status`, {
    method: "PATCH",
    body: { status },
  });
}

export async function deleteClient(id: string) {
  return request(`/clients/${id}`, { method: "DELETE" });
}

// Contacts
export async function getClientContacts(clientId: string) {
  return request<ContactResponse[]>(`/clients/${clientId}/contacts`);
}

export async function createContact(clientId: string, data: CreateContactData) {
  return request<ContactResponse>(`/clients/${clientId}/contacts`, {
    method: "POST",
    body: data,
  });
}

export async function updateContact(
  clientId: string,
  contactId: string,
  data: UpdateContactData,
) {
  return request<ContactResponse>(
    `/clients/${clientId}/contacts/${contactId}`,
    { method: "PUT", body: data },
  );
}

export async function deleteContact(clientId: string, contactId: string) {
  return request(`/clients/${clientId}/contacts/${contactId}`, {
    method: "DELETE",
  });
}

// Interactions
export async function getClientInteractions(
  clientId: string,
  skip = 0,
  limit = 50,
) {
  return request<InteractionResponse[]>(
    `/clients/${clientId}/interactions?skip=${skip}&limit=${limit}`,
  );
}

export async function createInteraction(
  clientId: string,
  data: CreateInteractionData,
) {
  return request<InteractionResponse>(`/clients/${clientId}/interactions`, {
    method: "POST",
    body: data,
  });
}

// Client-specific Deals
export async function getClientDeals(clientId: string) {
  return request<DealResponse[]>(`/clients/${clientId}/deals`);
}

export async function createDeal(clientId: string, data: CreateDealData) {
  return request<DealResponse>(`/clients/${clientId}/deals`, {
    method: "POST",
    body: data,
  });
}

export async function updateDeal(
  clientId: string,
  dealId: string,
  data: UpdateDealData,
) {
  return request<DealResponse>(`/clients/${clientId}/deals/${dealId}`, {
    method: "PUT",
    body: data,
  });
}

// Note: Standalone pipeline deal operations (getAllDeals, updateDealStage) are in the deals module
