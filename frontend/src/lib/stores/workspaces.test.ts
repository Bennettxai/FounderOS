import { get } from "svelte/store";
import { beforeEach, describe, expect, it, vi } from "vitest";

import {
  type Workspace,
  getWorkspace,
  getWorkspaceMembers,
  getWorkspaceProfile,
  getWorkspaceRoles,
  getWorkspaces,
  getUserRoleContext,
} from "$lib/api/workspaces";
import { fetchWorkspaces as fetchKnowledgeWorkspaces } from "$lib/kb/client";
import {
  clearWorkspaceState,
  currentUserRoleContext,
  currentWorkspace,
  initializeWorkspaces,
  loadSavedWorkspace,
  switchWorkspace,
  workspaces,
} from "./workspaces";

vi.mock("$lib/api/workspaces", () => ({
  getWorkspaces: vi.fn(),
  getWorkspace: vi.fn(),
  getWorkspaceRoles: vi.fn(),
  getWorkspaceMembers: vi.fn(),
  getWorkspaceProfile: vi.fn(),
  getUserRoleContext: vi.fn(),
}));

vi.mock("$lib/kb/client", () => ({
  fetchWorkspaces: vi.fn(),
}));

const mockedGetWorkspaces = vi.mocked(getWorkspaces);
const mockedGetWorkspace = vi.mocked(getWorkspace);
const mockedGetWorkspaceRoles = vi.mocked(getWorkspaceRoles);
const mockedGetWorkspaceMembers = vi.mocked(getWorkspaceMembers);
const mockedGetWorkspaceProfile = vi.mocked(getWorkspaceProfile);
const mockedGetUserRoleContext = vi.mocked(getUserRoleContext);
const mockedFetchKnowledgeWorkspaces = vi.mocked(fetchKnowledgeWorkspaces);

describe("workspace store", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
    clearWorkspaceState();
    vi.spyOn(console, "warn").mockImplementation(() => {});

    mockedGetWorkspaceRoles.mockResolvedValue([]);
    mockedGetWorkspaceMembers.mockResolvedValue([]);
    mockedGetWorkspaceProfile.mockResolvedValue(null as never);
    mockedGetUserRoleContext.mockResolvedValue({
      user_id: "user-1",
      workspace_id: "workspace-1",
      role_name: "owner",
      role_display_name: "Owner",
      hierarchy_level: 0,
      permissions: {},
      title: null,
      department: null,
      expertise_areas: null,
    });
  });

  it("falls back to knowledge workspaces when the workspace API is unavailable", async () => {
    mockedGetWorkspaces.mockRejectedValue(new Error("cloud workspaces unavailable"));
    mockedFetchKnowledgeWorkspaces.mockResolvedValue([
      { slug: "agency-miosa", name: "Agency Miosa" },
    ]);

    await initializeWorkspaces();

    expect(get(workspaces)).toEqual([
      expect.objectContaining({
        id: "kb:agency-miosa",
        name: "Agency Miosa",
        slug: "agency-miosa",
      }),
    ]);
    expect(get(currentWorkspace)).toEqual(
      expect.objectContaining({
        id: "kb:agency-miosa",
        name: "Agency Miosa",
      }),
    );
    expect(get(currentUserRoleContext)).toEqual(
      expect.objectContaining({
        workspace_id: "kb:agency-miosa",
        role_display_name: "Owner",
      }),
    );
    expect(mockedGetWorkspace).not.toHaveBeenCalled();
  });

  it("loads the workspace list before applying a saved knowledge workspace selection", async () => {
    localStorage.setItem("businessos_current_workspace_id", "kb:personal-brand");
    mockedGetWorkspaces.mockRejectedValue(new Error("cloud workspaces unavailable"));
    mockedFetchKnowledgeWorkspaces.mockResolvedValue([
      { slug: "agency-miosa", name: "Agency Miosa" },
      { slug: "personal-brand", name: "Personal Brand" },
    ]);

    await loadSavedWorkspace();

    expect(get(workspaces)).toHaveLength(2);
    expect(get(currentWorkspace)).toEqual(
      expect.objectContaining({
        id: "kb:personal-brand",
        name: "Personal Brand",
      }),
    );
    expect(localStorage.getItem("businessos_current_workspace_id")).toBe(
      "kb:personal-brand",
    );
    expect(mockedGetWorkspace).not.toHaveBeenCalled();
  });

  it("restores the exact saved workspace without preferring a tenant-specific slug", async () => {
    localStorage.setItem("businessos_current_workspace_id", "workspace-personal");
    const personal: Workspace = {
      id: "workspace-personal",
      name: "My Workspace",
      slug: "my-workspace",
      description: null,
      logo_url: null,
      plan_type: "free",
      max_members: 5,
      max_projects: 10,
      max_storage_gb: 1,
      settings: {},
      owner_id: "user-1",
      created_at: "2026-06-24T00:00:00.000Z",
      updated_at: "2026-06-24T00:00:00.000Z",
    };
    const agency = { ...personal, id: "workspace-agency", name: "Agency Miosa", slug: "agency-miosa" };
    mockedGetWorkspaces.mockResolvedValue([agency, personal]);
    mockedFetchKnowledgeWorkspaces.mockResolvedValue([]);
    mockedGetWorkspace.mockResolvedValue(personal);

    await initializeWorkspaces();

    expect(mockedGetWorkspace).toHaveBeenCalledWith("workspace-personal");
    expect(get(currentWorkspace)?.id).toBe("workspace-personal");
    expect(localStorage.getItem("businessos_current_workspace_id")).toBe("workspace-personal");
  });

  it("keeps the saved workspace when a switch fails transiently (429 from many desktop tiles at once)", async () => {
    localStorage.setItem("businessos_current_workspace_id", "workspace-personal");
    const personal: Workspace = {
      id: "workspace-personal", name: "Personal", slug: "personal", description: null, logo_url: null,
      plan_type: "free", max_members: 5, max_projects: 10, max_storage_gb: 1, settings: {},
      owner_id: "user-1", created_at: "2026-06-24T00:00:00.000Z", updated_at: "2026-06-24T00:00:00.000Z",
    };
    mockedGetWorkspaces.mockResolvedValue([personal]);
    mockedFetchKnowledgeWorkspaces.mockResolvedValue([]);
    mockedGetWorkspace.mockRejectedValue(new Error("Too many requests. Please slow down. (HTTP 429)"));

    await initializeWorkspaces();

    expect(localStorage.getItem("businessos_current_workspace_id")).toBe("workspace-personal");
  });

  it("retries a transient failure and lands on the saved workspace, not a fallback", async () => {
    localStorage.setItem("businessos_current_workspace_id", "workspace-personal");
    const personal: Workspace = {
      id: "workspace-personal", name: "Personal", slug: "personal", description: null, logo_url: null,
      plan_type: "free", max_members: 5, max_projects: 10, max_storage_gb: 1, settings: {},
      owner_id: "user-1", created_at: "2026-06-24T00:00:00.000Z", updated_at: "2026-06-24T00:00:00.000Z",
    };
    const other = { ...personal, id: "workspace-other", name: "Other", slug: "other" };
    mockedGetWorkspaces.mockResolvedValue([other, personal]);
    mockedFetchKnowledgeWorkspaces.mockResolvedValue([]);
    mockedGetWorkspace
      .mockRejectedValueOnce(new Error("Too many requests. Please slow down. (HTTP 429)"))
      .mockRejectedValueOnce(new Error("Too many requests. Please slow down. (HTTP 429)"))
      .mockRejectedValueOnce(new Error("Too many requests. Please slow down. (HTTP 429)"))
      .mockResolvedValue(personal);

    await loadSavedWorkspace();

    expect(get(currentWorkspace)?.id).toBe("workspace-personal");
    expect(localStorage.getItem("businessos_current_workspace_id")).toBe("workspace-personal");
  });

  it("still forgets a saved workspace the account can no longer open (403/404)", async () => {
    localStorage.setItem("businessos_current_workspace_id", "workspace-gone");
    const gone: Workspace = {
      id: "workspace-gone", name: "Gone", slug: "gone", description: null, logo_url: null,
      plan_type: "free", max_members: 5, max_projects: 10, max_storage_gb: 1, settings: {},
      owner_id: "user-1", created_at: "2026-06-24T00:00:00.000Z", updated_at: "2026-06-24T00:00:00.000Z",
    };
    mockedGetWorkspaces.mockResolvedValue([gone]);
    mockedFetchKnowledgeWorkspaces.mockResolvedValue([]);
    mockedGetWorkspace.mockRejectedValue(new Error("Workspace not found (HTTP 404)"));

    await initializeWorkspaces();

    expect(localStorage.getItem("businessos_current_workspace_id")).toBeNull();
  });

  it("uses API order when no prior workspace selection exists", async () => {
    const first: Workspace = {
      id: "workspace-first",
      name: "Customer Workspace",
      slug: "customer-workspace",
      description: null,
      logo_url: null,
      plan_type: "free",
      max_members: 5,
      max_projects: 10,
      max_storage_gb: 1,
      settings: {},
      owner_id: "user-1",
      created_at: "2026-06-24T00:00:00.000Z",
      updated_at: "2026-06-24T00:00:00.000Z",
    };
    const agency = { ...first, id: "workspace-agency", name: "Agency Miosa", slug: "agency-miosa" };
    mockedGetWorkspaces.mockResolvedValue([first, agency]);
    mockedFetchKnowledgeWorkspaces.mockResolvedValue([]);
    mockedGetWorkspace.mockResolvedValue(first);

    await initializeWorkspaces();

    expect(mockedGetWorkspace).toHaveBeenCalledWith("workspace-first");
    expect(get(currentWorkspace)?.id).toBe("workspace-first");
  });

  it("opens the FounderOS HQ workspace first when nothing is saved", async () => {
    const mine: Workspace = {
      id: "workspace-mine",
      name: "My Workspace",
      slug: "workspace-eb3ac8280a61e17b",
      description: null,
      logo_url: null,
      plan_type: "free",
      max_members: 5,
      max_projects: 10,
      max_storage_gb: 1,
      settings: {},
      owner_id: "user-1",
      created_at: "2026-06-24T00:00:00.000Z",
      updated_at: "2026-06-24T00:00:00.000Z",
    };
    const hq = { ...mine, id: "workspace-hq", name: "HQ", slug: "founderos" };
    mockedGetWorkspaces.mockResolvedValue([mine, hq]);
    mockedFetchKnowledgeWorkspaces.mockResolvedValue([]);
    mockedGetWorkspace.mockResolvedValue(hq);

    await initializeWorkspaces();

    expect(mockedGetWorkspace).toHaveBeenCalledWith("workspace-hq");
    expect(get(currentWorkspace)?.id).toBe("workspace-hq");
  });

  it("persists the selected workspace before publishing it to subscribers", async () => {
    const workspace: Workspace = {
      id: "workspace-agency",
      name: "Agency MIOSA",
      slug: "agency-miosa",
      description: null,
      logo_url: null,
      plan_type: "free",
      max_members: 5,
      max_projects: 10,
      max_storage_gb: 1,
      settings: {},
      owner_id: "user-1",
      created_at: "2026-07-25T00:00:00.000Z",
      updated_at: "2026-07-25T00:00:00.000Z",
    };
    workspaces.set([workspace]);
    mockedGetWorkspace.mockResolvedValue(workspace);

    const observedWorkspaceIds: Array<string | null> = [];
    const unsubscribe = currentWorkspace.subscribe((selected) => {
      if (!selected) return;
      observedWorkspaceIds.push(
        localStorage.getItem("businessos_current_workspace_id"),
      );
    });

    await switchWorkspace(workspace.id);
    unsubscribe();

    expect(observedWorkspaceIds).toEqual(["workspace-agency"]);
  });

  it("clears a stale saved workspace and falls back to an accessible workspace", async () => {
    localStorage.setItem("businessos_current_workspace_id", "workspace-stale");
    mockedGetWorkspaces.mockResolvedValue([
      {
        id: "workspace-1",
        name: "MIOSA",
        slug: "miosa",
        description: null,
        logo_url: null,
        plan_type: "free",
        max_members: 5,
        max_projects: 10,
        max_storage_gb: 1,
        settings: {},
        owner_id: "user-1",
        created_at: "2026-06-24T00:00:00.000Z",
        updated_at: "2026-06-24T00:00:00.000Z",
      },
    ]);
    mockedFetchKnowledgeWorkspaces.mockResolvedValue([]);
    mockedGetWorkspace.mockResolvedValue({
      id: "workspace-1",
      name: "MIOSA",
      slug: "miosa",
      description: null,
      logo_url: null,
      plan_type: "free",
      max_members: 5,
      max_projects: 10,
      max_storage_gb: 1,
      settings: {},
      owner_id: "user-1",
      created_at: "2026-06-24T00:00:00.000Z",
      updated_at: "2026-06-24T00:00:00.000Z",
    });

    await loadSavedWorkspace();

    expect(get(currentWorkspace)).toEqual(
      expect.objectContaining({
        id: "workspace-1",
        name: "MIOSA",
      }),
    );
    expect(localStorage.getItem("businessos_current_workspace_id")).toBe(
      "workspace-1",
    );
  });

  it("uses database workspaces without mixing in engine knowledge scopes", async () => {
    mockedGetWorkspaces.mockResolvedValue([
      {
        id: "workspace-db-1",
        name: "BusinessOS",
        slug: "businessos",
        description: null,
        logo_url: null,
        plan_type: "free",
        max_members: 5,
        max_projects: 10,
        max_storage_gb: 1,
        settings: {},
        owner_id: "user-1",
        created_at: "2026-06-24T00:00:00.000Z",
        updated_at: "2026-06-24T00:00:00.000Z",
      },
    ]);
    mockedGetWorkspace.mockResolvedValue({
      id: "workspace-db-1",
      name: "BusinessOS",
      slug: "businessos",
      description: null,
      logo_url: null,
      plan_type: "free",
      max_members: 5,
      max_projects: 10,
      max_storage_gb: 1,
      settings: {},
      owner_id: "user-1",
      created_at: "2026-06-24T00:00:00.000Z",
      updated_at: "2026-06-24T00:00:00.000Z",
    });
    mockedFetchKnowledgeWorkspaces.mockResolvedValue([
      { slug: "businessos", name: "BusinessOS" },
      { slug: "agency-miosa", name: "Agency Miosa" },
    ]);

    await initializeWorkspaces();

    expect(get(workspaces)).toEqual([
      expect.objectContaining({
        id: "workspace-db-1",
        slug: "businessos",
      }),
    ]);
    expect(mockedFetchKnowledgeWorkspaces).not.toHaveBeenCalled();
  });
});
