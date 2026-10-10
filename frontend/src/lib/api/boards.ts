// Boards API - the Views/Boards composition layer (Phase B).
// A board is a workspace-scoped surface composing several module views
// (projects, tasks, team, deals, clients) filtered to one subject
// (typically a client). See .claude plan "Views & Boards".
//
// Conventions match the sibling api files (modules.ts): the shared `request`
// helper injects the X-Workspace-ID header, CSRF token on mutations, and
// invalidates the GET cache after writes.

import { request } from "./base";

// ── Types ─────────────────────────────────────────────────────────────────────

export type BoardView = "projects" | "tasks" | "team" | "deals" | "clients";

export interface BoardLayoutEntry {
  view: BoardView;
  filters?: {
    client_id?: string;
  };
  group_by?: string;
}

export interface Board {
  id: string;
  workspace_id: string;
  name: string;
  kind: string;
  subject_type: string;
  subject_id: string;
  layout: BoardLayoutEntry[];
  is_pinned: boolean;
  position: number;
  created_at: string;
  updated_at: string;
}

export interface BoardSection {
  view: BoardView;
  items: unknown[];
  count: number;
}

export interface BoardData {
  board: Board;
  sections: BoardSection[];
}

export interface CreateBoardInput {
  name: string;
  kind?: string;
  subject_type?: string;
  subject_id?: string;
  layout?: BoardLayoutEntry[];
  is_pinned?: boolean;
  position?: number;
}

export interface UpdateBoardInput {
  name?: string;
  kind?: string;
  subject_type?: string;
  subject_id?: string;
  layout?: BoardLayoutEntry[];
  is_pinned?: boolean;
  position?: number;
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// Backends in this repo return lists either as a plain array, { boards: [...] },
// or the paginated { data: [...] } envelope. Normalize all three.
function unwrapBoardList(raw: unknown): Board[] {
  if (Array.isArray(raw)) return raw as Board[];
  if (raw && typeof raw === "object") {
    const obj = raw as Record<string, unknown>;
    if (Array.isArray(obj.boards)) return obj.boards as Board[];
    if (Array.isArray(obj.data)) return obj.data as Board[];
  }
  return [];
}

// ── API functions ─────────────────────────────────────────────────────────────

export async function listBoards(pinnedOnly?: boolean): Promise<Board[]> {
  const query = pinnedOnly ? "?pinned=true" : "";
  const raw = await request<unknown>(`/boards${query}`, { skipCache: true });
  return unwrapBoardList(raw);
}

export async function createBoard(input: CreateBoardInput): Promise<Board> {
  return request(`/boards`, {
    method: "POST",
    body: input,
  });
}

export async function getBoard(id: string): Promise<Board> {
  return request(`/boards/${id}`);
}

export async function updateBoard(
  id: string,
  patch: UpdateBoardInput,
): Promise<Board> {
  return request(`/boards/${id}`, {
    method: "PUT",
    body: patch,
  });
}

export async function deleteBoard(id: string): Promise<{ message: string }> {
  return request(`/boards/${id}`, {
    method: "DELETE",
  });
}

export async function pinBoard(id: string, pinned: boolean): Promise<Board> {
  // Backend contract: POST /boards/:id/pin with {"pinned": bool} (boards.go PinBoard).
  return request(`/boards/${id}/pin`, {
    method: "POST",
    body: { pinned },
  });
}

export async function getBoardData(id: string): Promise<BoardData> {
  return request(`/boards/${id}/data`, { skipCache: true });
}
