import { useCallback, useEffect, useReducer, useRef, useState } from "react";
import {
  clipMemory,
  listMemories,
  listWorkspaces,
  searchMemories,
} from "@/lib/client";
import { getSettings, saveSettings } from "@/lib/storage";
import type {
  ExtensionMessage,
  Memory,
  PageInfo,
  SearchResult,
  Workspace,
} from "@/lib/types";

// ─── State ────────────────────────────────────────────────────────────────────

interface AppState {
  engineUrl: string;
  workspaces: Workspace[];
  activeWorkspace: string;
  searchQuery: string;
  searchResults: SearchResult[];
  recentClips: Memory[];
  isClipping: boolean;
  isSearching: boolean;
  clipStatus: "idle" | "success" | "error";
  clipError: string;
  loadError: string;
  pageInfo: PageInfo | null;
  view: "main" | "profile";
}

type Action =
  | { type: "SET_ENGINE_URL"; url: string }
  | { type: "SET_WORKSPACES"; workspaces: Workspace[] }
  | { type: "SET_WORKSPACE"; workspace: string }
  | { type: "SET_SEARCH_QUERY"; query: string }
  | { type: "SET_SEARCH_RESULTS"; results: SearchResult[] }
  | { type: "SET_RECENT_CLIPS"; clips: Memory[] }
  | { type: "SET_CLIPPING"; value: boolean }
  | { type: "SET_SEARCHING"; value: boolean }
  | { type: "SET_CLIP_STATUS"; status: AppState["clipStatus"]; error?: string }
  | { type: "SET_LOAD_ERROR"; error: string }
  | { type: "SET_PAGE_INFO"; info: PageInfo }
  | { type: "SET_VIEW"; view: AppState["view"] };

function reducer(state: AppState, action: Action): AppState {
  switch (action.type) {
    case "SET_ENGINE_URL":
      return { ...state, engineUrl: action.url };
    case "SET_WORKSPACES":
      return { ...state, workspaces: action.workspaces };
    case "SET_WORKSPACE":
      return { ...state, activeWorkspace: action.workspace };
    case "SET_SEARCH_QUERY":
      return { ...state, searchQuery: action.query };
    case "SET_SEARCH_RESULTS":
      return { ...state, searchResults: action.results };
    case "SET_RECENT_CLIPS":
      return { ...state, recentClips: action.clips };
    case "SET_CLIPPING":
      return { ...state, isClipping: action.value };
    case "SET_SEARCHING":
      return { ...state, isSearching: action.value };
    case "SET_CLIP_STATUS":
      return {
        ...state,
        clipStatus: action.status,
        clipError: action.error ?? "",
      };
    case "SET_LOAD_ERROR":
      return { ...state, loadError: action.error };
    case "SET_PAGE_INFO":
      return { ...state, pageInfo: action.info };
    case "SET_VIEW":
      return { ...state, view: action.view };
    default:
      return state;
  }
}

const INITIAL_STATE: AppState = {
  engineUrl: "",
  workspaces: [],
  activeWorkspace: "",
  searchQuery: "",
  searchResults: [],
  recentClips: [],
  isClipping: false,
  isSearching: false,
  clipStatus: "idle",
  clipError: "",
  loadError: "",
  pageInfo: null,
  view: "main",
};

// ─── Helpers ──────────────────────────────────────────────────────────────────

function truncate(text: string, max: number): string {
  return text.length > max ? text.slice(0, max) + "…" : text;
}

function timeAgo(iso?: string): string {
  if (!iso) return "";
  const diff = Date.now() - new Date(iso).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  return `${Math.floor(hrs / 24)}d ago`;
}

async function getActiveTabInfo(): Promise<PageInfo | null> {
  try {
    const [tab] = await chrome.tabs.query({
      active: true,
      currentWindow: true,
    });
    if (!tab?.id) return null;

    const msg: ExtensionMessage = { type: "GET_PAGE_INFO" };
    const info = await chrome.tabs.sendMessage(tab.id, msg).catch(() => null);
    return info as PageInfo | null;
  } catch {
    return null;
  }
}

// ─── Component ────────────────────────────────────────────────────────────────

export default function App() {
  const [state, dispatch] = useReducer(reducer, INITIAL_STATE);
  const searchTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // ── Bootstrap ──────────────────────────────────────────────────────────────

  useEffect(() => {
    void (async () => {
      const settings = await getSettings();
      dispatch({ type: "SET_ENGINE_URL", url: settings.engineUrl });

      if (!settings.engineUrl) return;

      dispatch({
        type: "SET_WORKSPACE",
        workspace: settings.defaultWorkspace,
      });

      // Load workspaces + page info in parallel
      const [wsResult, pageInfo] = await Promise.all([
        listWorkspaces(),
        getActiveTabInfo(),
      ]);

      if (wsResult.ok) {
        dispatch({ type: "SET_WORKSPACES", workspaces: wsResult.value });
      } else {
        dispatch({
          type: "SET_LOAD_ERROR",
          error: `Could not load workspaces: ${wsResult.error}`,
        });
      }

      if (pageInfo) {
        dispatch({ type: "SET_PAGE_INFO", info: pageInfo });
      }

      // Load recent clips
      if (settings.defaultWorkspace) {
        const clipsResult = await listMemories(settings.defaultWorkspace, 10);
        if (clipsResult.ok) {
          dispatch({ type: "SET_RECENT_CLIPS", clips: clipsResult.value });
        }
      }
    })();
  }, []);

  // Reload recent clips when workspace changes
  useEffect(() => {
    if (!state.activeWorkspace || !state.engineUrl) return;
    void listMemories(state.activeWorkspace, 10).then((result) => {
      if (result.ok) {
        dispatch({ type: "SET_RECENT_CLIPS", clips: result.value });
      }
    });
  }, [state.activeWorkspace, state.engineUrl]);

  // ── Workspace change ───────────────────────────────────────────────────────

  const handleWorkspaceChange = useCallback(async (workspace: string) => {
    dispatch({ type: "SET_WORKSPACE", workspace });
    await saveSettings({ defaultWorkspace: workspace });
  }, []);

  // ── Search ─────────────────────────────────────────────────────────────────

  const handleSearchChange = useCallback(
    (query: string) => {
      dispatch({ type: "SET_SEARCH_QUERY", query });

      if (searchTimerRef.current) clearTimeout(searchTimerRef.current);

      if (!query.trim()) {
        dispatch({ type: "SET_SEARCH_RESULTS", results: [] });
        return;
      }

      searchTimerRef.current = setTimeout(async () => {
        dispatch({ type: "SET_SEARCHING", value: true });
        const result = await searchMemories(query, state.activeWorkspace, 20);
        dispatch({ type: "SET_SEARCHING", value: false });
        if (result.ok) {
          dispatch({ type: "SET_SEARCH_RESULTS", results: result.value });
        } else {
          dispatch({ type: "SET_SEARCH_RESULTS", results: [] });
        }
      }, 300);
    },
    [state.activeWorkspace],
  );

  // ── Clip ───────────────────────────────────────────────────────────────────

  const handleClip = useCallback(
    async (mode: "page" | "selection") => {
      if (!state.pageInfo) return;
      dispatch({ type: "SET_CLIPPING", value: true });
      dispatch({ type: "SET_CLIP_STATUS", status: "idle" });

      const content =
        mode === "selection" && state.pageInfo.selectedText
          ? state.pageInfo.selectedText
          : state.pageInfo.bodyText.slice(0, 1000);

      const result = await clipMemory({
        content,
        workspace: state.activeWorkspace,
        citation_uri: state.pageInfo.url,
        metadata: {
          url: state.pageInfo.url,
          title: state.pageInfo.title,
          clipped_at: new Date().toISOString(),
          source: mode === "selection" ? "popup-selection" : "popup-page",
        },
      });

      dispatch({ type: "SET_CLIPPING", value: false });

      if (result.ok) {
        dispatch({ type: "SET_CLIP_STATUS", status: "success" });
        // Refresh recent clips
        const clipsResult = await listMemories(state.activeWorkspace, 10);
        if (clipsResult.ok) {
          dispatch({ type: "SET_RECENT_CLIPS", clips: clipsResult.value });
        }
        setTimeout(
          () => dispatch({ type: "SET_CLIP_STATUS", status: "idle" }),
          2500,
        );
      } else {
        dispatch({
          type: "SET_CLIP_STATUS",
          status: "error",
          error: result.error,
        });
      }
    },
    [state.pageInfo, state.activeWorkspace],
  );

  // ── No engine configured ───────────────────────────────────────────────────

  if (!state.engineUrl) {
    return (
      <div className="flex h-full min-h-[200px] flex-col items-center justify-center gap-3 p-6 text-center">
        <EngineIcon className="h-10 w-10 text-accent opacity-60" />
        <p className="text-sm text-text-secondary">No engine configured.</p>
        <button
          className="btn-primary"
          onClick={() => chrome.runtime.openOptionsPage()}
        >
          Open Options
        </button>
      </div>
    );
  }

  const hasSelection = Boolean(state.pageInfo?.selectedText);
  const showSearchResults = state.searchQuery.trim().length > 0;

  return (
    <div
      className="flex flex-col"
      style={{ width: 380, minHeight: 480, maxHeight: 600 }}
    >
      {/* ── Header ── */}
      <header className="flex items-center justify-between border-b border-bg-border px-4 py-3">
        <div className="flex items-center gap-2">
          <EngineIcon className="h-5 w-5 text-accent" />
          <span className="text-sm font-semibold tracking-tight text-text-primary">
            Optimal Engine
          </span>
        </div>
        <button
          className="btn-ghost px-2 py-1 text-xs"
          onClick={() => chrome.runtime.openOptionsPage()}
          aria-label="Open options"
        >
          Settings
        </button>
      </header>

      {/* ── Workspace selector ── */}
      <div className="border-b border-bg-border px-4 py-2">
        <select
          className="select text-xs"
          value={state.activeWorkspace}
          onChange={(e) => void handleWorkspaceChange(e.target.value)}
          aria-label="Select workspace"
        >
          {state.workspaces.length === 0 && (
            <option value="">No workspaces found</option>
          )}
          {state.workspaces.map((ws) => (
            <option key={ws.id} value={ws.id}>
              {ws.name}
            </option>
          ))}
        </select>
        {state.loadError && (
          <p className="mt-1 text-xs text-error">{state.loadError}</p>
        )}
      </div>

      {/* ── Search ── */}
      <div className="border-b border-bg-border px-4 py-3">
        <div className="relative">
          <SearchIcon className="absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-text-muted" />
          <input
            className="input pl-8 text-xs"
            type="search"
            placeholder="Search memories…"
            value={state.searchQuery}
            onChange={(e) => handleSearchChange(e.target.value)}
            aria-label="Search memories"
          />
          {state.isSearching && (
            <SpinnerIcon className="absolute right-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 animate-spin text-accent" />
          )}
        </div>
      </div>

      {/* ── Scrollable body ── */}
      <div className="flex-1 overflow-y-auto">
        {/* Search results */}
        {showSearchResults && (
          <section className="px-4 py-3">
            {state.searchResults.length === 0 && !state.isSearching ? (
              <EmptyState message="No results found." />
            ) : (
              <ul className="flex flex-col gap-2" role="list">
                {state.searchResults.map((r) => (
                  <SearchResultItem key={r.id} result={r} />
                ))}
              </ul>
            )}
          </section>
        )}

        {/* Main view — clip + recent */}
        {!showSearchResults && (
          <>
            {/* Clip actions */}
            <section className="border-b border-bg-border px-4 py-3">
              {state.pageInfo && (
                <div className="mb-2">
                  <p className="truncate text-xs font-medium text-text-primary">
                    {state.pageInfo.title}
                  </p>
                  <p className="truncate text-xs text-text-muted">
                    {state.pageInfo.url}
                  </p>
                </div>
              )}

              <div className="flex gap-2">
                <button
                  className="btn-primary flex-1 text-xs"
                  onClick={() => void handleClip("page")}
                  disabled={state.isClipping || !state.pageInfo}
                  aria-label="Clip this page"
                >
                  {state.isClipping ? (
                    <SpinnerIcon className="h-3.5 w-3.5 animate-spin" />
                  ) : (
                    <ClipIcon className="h-3.5 w-3.5" />
                  )}
                  Clip page
                </button>

                {hasSelection && (
                  <button
                    className="btn-ghost flex-1 border border-bg-border text-xs"
                    onClick={() => void handleClip("selection")}
                    disabled={state.isClipping}
                    aria-label="Clip selected text"
                  >
                    <SelectionIcon className="h-3.5 w-3.5" />
                    Clip selection
                  </button>
                )}
              </div>

              {/* Clip status */}
              {state.clipStatus === "success" && (
                <p className="mt-2 flex items-center gap-1 text-xs text-success">
                  <CheckIcon className="h-3.5 w-3.5" />
                  Saved to engine
                </p>
              )}
              {state.clipStatus === "error" && (
                <p className="mt-2 text-xs text-error">
                  Error: {state.clipError}
                </p>
              )}
            </section>

            {/* Recent clips */}
            <section className="px-4 py-3">
              <h2 className="mb-2 text-xs font-semibold uppercase tracking-wider text-text-muted">
                Recent clips
              </h2>
              {state.recentClips.length === 0 ? (
                <EmptyState message="No clips yet in this workspace." />
              ) : (
                <ul className="flex flex-col gap-2" role="list">
                  {state.recentClips.map((clip) => (
                    <RecentClipItem key={clip.id} clip={clip} />
                  ))}
                </ul>
              )}
            </section>
          </>
        )}
      </div>
    </div>
  );
}

// ─── Sub-components ───────────────────────────────────────────────────────────

function SearchResultItem({ result }: { result: SearchResult }) {
  const url = result.metadata?.url as string | undefined;
  return (
    <li className="card group cursor-default">
      <p className="text-xs text-text-primary">
        {truncate(result.content, 140)}
      </p>
      {url && (
        <a
          href={url}
          target="_blank"
          rel="noopener noreferrer"
          className="mt-1 block truncate text-xs text-accent hover:underline"
          aria-label={`Open source: ${url}`}
        >
          {url}
        </a>
      )}
      {result.score !== undefined && (
        <span className="tag mt-1">score: {result.score.toFixed(2)}</span>
      )}
    </li>
  );
}

function RecentClipItem({ clip }: { clip: Memory }) {
  const url = clip.metadata?.url as string | undefined;
  const title = clip.metadata?.title as string | undefined;
  return (
    <li className="card">
      <div className="flex items-start justify-between gap-2">
        <p className="flex-1 text-xs text-text-primary">
          {truncate(title ?? clip.content, 80)}
        </p>
        <span className="shrink-0 text-xs text-text-muted">
          {timeAgo(
            clip.created_at ??
              (clip.metadata?.clipped_at as string | undefined),
          )}
        </span>
      </div>
      {url && (
        <a
          href={url}
          target="_blank"
          rel="noopener noreferrer"
          className="mt-0.5 block truncate text-xs text-text-muted hover:text-accent"
          aria-label={`Open clipped page: ${url}`}
        >
          {url}
        </a>
      )}
    </li>
  );
}

function EmptyState({ message }: { message: string }) {
  return <p className="py-4 text-center text-xs text-text-muted">{message}</p>;
}

// ─── Icons (inline SVG — zero extra deps) ─────────────────────────────────────

function EngineIcon({ className }: { className?: string }) {
  return (
    <svg
      className={className}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={1.5}
      aria-hidden="true"
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M9.75 3.104v5.714a2.25 2.25 0 0 1-.659 1.591L5 14.5M9.75 3.104c-.251.023-.501.05-.75.082m.75-.082a24.301 24.301 0 0 1 4.5 0m0 0v5.714c0 .597.237 1.17.659 1.591L19.8 15.3M14.25 3.104c.251.023.501.05.75.082M19.8 15.3l-1.57.393A9.065 9.065 0 0 1 12 15a9.065 9.065 0 0 0-6.23-.693L5 14.5m14.8.8 1.402 1.402c1.232 1.232.65 3.318-1.067 3.611A48.309 48.309 0 0 1 12 21c-2.773 0-5.491-.235-8.135-.687-1.718-.293-2.3-2.379-1.067-3.61L5 14.5"
      />
    </svg>
  );
}

function SearchIcon({ className }: { className?: string }) {
  return (
    <svg
      className={className}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      aria-hidden="true"
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="m21 21-5.197-5.197m0 0A7.5 7.5 0 1 0 5.196 5.196a7.5 7.5 0 0 0 10.607 10.607Z"
      />
    </svg>
  );
}

function SpinnerIcon({ className }: { className?: string }) {
  return (
    <svg
      className={className}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      aria-hidden="true"
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0 3.181 3.183a8.25 8.25 0 0 0 13.803-3.7M4.031 9.865a8.25 8.25 0 0 1 13.803-3.7l3.181 3.182m0-4.991v4.99"
      />
    </svg>
  );
}

function ClipIcon({ className }: { className?: string }) {
  return (
    <svg
      className={className}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={1.75}
      aria-hidden="true"
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M12 16.5V9.75m0 0 3 3m-3-3-3 3M6.75 19.5a4.5 4.5 0 0 1-1.41-8.775 5.25 5.25 0 0 1 10.233-2.33 3 3 0 0 1 3.758 3.848A3.752 3.752 0 0 1 18 19.5H6.75Z"
      />
    </svg>
  );
}

function SelectionIcon({ className }: { className?: string }) {
  return (
    <svg
      className={className}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={1.75}
      aria-hidden="true"
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="M8.25 6.75h12M8.25 12h12m-12 5.25h12M3.75 6.75h.007v.008H3.75V6.75Zm.375 0a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0ZM3.75 12h.007v.008H3.75V12Zm.375 0a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0Zm-.375 5.25h.007v.008H3.75v-.008Zm.375 0a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0Z"
      />
    </svg>
  );
}

function CheckIcon({ className }: { className?: string }) {
  return (
    <svg
      className={className}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      aria-hidden="true"
    >
      <path
        strokeLinecap="round"
        strokeLinejoin="round"
        d="m4.5 12.75 6 6 9-13.5"
      />
    </svg>
  );
}
