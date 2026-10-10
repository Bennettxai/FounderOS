import { request } from "./base";
import type { DesktopSpace } from "$lib/stores/windowStore";

export interface WorkspaceDesktopSpaceRecord {
  id: string;
  workspace_id: string;
  name: string;
  kind: "personal" | "team" | "workspace";
  config: DesktopSpace;
  created_by: string | null;
  created_at: string;
  updated_at: string;
}

export async function listWorkspaceDesktopSpaces(
  workspaceId: string,
): Promise<{ spaces: WorkspaceDesktopSpaceRecord[]; count: number }> {
  return request<{ spaces: WorkspaceDesktopSpaceRecord[]; count: number }>(
    `/workspaces/${encodeURIComponent(workspaceId)}/desktop-spaces`,
    { skipCache: true },
  );
}

export function saveWorkspaceDesktopSpace(
  workspaceId: string,
  space: DesktopSpace,
): Promise<WorkspaceDesktopSpaceRecord> {
  return request<WorkspaceDesktopSpaceRecord>(
    `/workspaces/${encodeURIComponent(workspaceId)}/desktop-spaces`,
    {
      method: "PUT",
      body: {
        id: space.id,
        name: space.name,
        kind: space.kind,
        config: space,
      },
    },
  );
}

export function deleteWorkspaceDesktopSpace(
  workspaceId: string,
  desktopSpaceId: string,
): Promise<unknown> {
  return request(
    `/workspaces/${encodeURIComponent(workspaceId)}/desktop-spaces/${encodeURIComponent(desktopSpaceId)}`,
    { method: "DELETE" },
  );
}
