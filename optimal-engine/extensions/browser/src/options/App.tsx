import { useEffect, useReducer } from "react";
import { getSettings, saveSettings } from "@/lib/storage";
import { listWorkspaces } from "@/lib/client";
import type { ExtensionSettings, Workspace } from "@/lib/types";
import { DEFAULT_SETTINGS } from "@/lib/types";

// ─── State ────────────────────────────────────────────────────────────────────

interface OptionsState {
  form: ExtensionSettings;
  workspaces: Workspace[];
  status: "idle" | "saving" | "saved" | "error";
  errorMsg: string;
  wsLoading: boolean;
  wsError: string;
}

type Action =
  | { type: "INIT"; settings: ExtensionSettings }
  | {
      type: "SET_FIELD";
      field: keyof ExtensionSettings;
      value: string | boolean;
    }
  | { type: "SET_WORKSPACES"; workspaces: Workspace[] }
  | { type: "SET_WS_LOADING"; value: boolean }
  | { type: "SET_WS_ERROR"; error: string }
  | { type: "SET_STATUS"; status: OptionsState["status"]; error?: string };

function reducer(state: OptionsState, action: Action): OptionsState {
  switch (action.type) {
    case "INIT":
      return { ...state, form: action.settings };
    case "SET_FIELD":
      return {
        ...state,
        form: { ...state.form, [action.field]: action.value },
      };
    case "SET_WORKSPACES":
      return { ...state, workspaces: action.workspaces };
    case "SET_WS_LOADING":
      return { ...state, wsLoading: action.value };
    case "SET_WS_ERROR":
      return { ...state, wsError: action.error };
    case "SET_STATUS":
      return { ...state, status: action.status, errorMsg: action.error ?? "" };
    default:
      return state;
  }
}

const INITIAL: OptionsState = {
  form: DEFAULT_SETTINGS,
  workspaces: [],
  status: "idle",
  errorMsg: "",
  wsLoading: false,
  wsError: "",
};

// ─── Component ────────────────────────────────────────────────────────────────

export default function OptionsApp() {
  const [state, dispatch] = useReducer(reducer, INITIAL);

  // Load saved settings on mount
  useEffect(() => {
    void getSettings().then((s) => dispatch({ type: "INIT", settings: s }));
  }, []);

  // Fetch workspaces whenever engineUrl changes (debounced via effect dep)
  useEffect(() => {
    const url = state.form.engineUrl.trim();
    if (!url) return;

    dispatch({ type: "SET_WS_LOADING", value: true });
    dispatch({ type: "SET_WS_ERROR", error: "" });

    void listWorkspaces().then((result) => {
      dispatch({ type: "SET_WS_LOADING", value: false });
      if (result.ok) {
        dispatch({ type: "SET_WORKSPACES", workspaces: result.value });
      } else {
        dispatch({ type: "SET_WS_ERROR", error: result.error });
      }
    });
  }, [state.form.engineUrl]);

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    dispatch({ type: "SET_STATUS", status: "saving" });
    try {
      await saveSettings(state.form);
      dispatch({ type: "SET_STATUS", status: "saved" });
      setTimeout(() => dispatch({ type: "SET_STATUS", status: "idle" }), 2000);
    } catch (err) {
      dispatch({
        type: "SET_STATUS",
        status: "error",
        error: err instanceof Error ? err.message : "Save failed",
      });
    }
  };

  const field = (key: keyof ExtensionSettings) => ({
    value: String(state.form[key]),
    onChange: (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) =>
      dispatch({ type: "SET_FIELD", field: key, value: e.target.value }),
  });

  return (
    <div className="px-8 py-8">
      {/* Header */}
      <header className="mb-8 flex items-center gap-3 border-b border-bg-border pb-5">
        <div>
          <h1 className="text-lg font-semibold tracking-tight text-text-primary">
            Optimal Engine — Web Clipper
          </h1>
          <p className="text-sm text-text-secondary">
            Configure your connection to the engine
          </p>
        </div>
      </header>

      <form
        onSubmit={(e) => void handleSave(e)}
        className="flex flex-col gap-6"
      >
        {/* Engine URL */}
        <fieldset className="flex flex-col gap-2">
          <label
            htmlFor="engineUrl"
            className="text-sm font-medium text-text-primary"
          >
            Engine URL
          </label>
          <input
            id="engineUrl"
            type="url"
            className="input"
            placeholder="http://localhost:4200"
            required
            {...field("engineUrl")}
          />
          <p className="text-xs text-text-muted">
            Base URL of your running Optimal Engine instance.
          </p>
        </fieldset>

        {/* Default workspace */}
        <fieldset className="flex flex-col gap-2">
          <label
            htmlFor="defaultWorkspace"
            className="text-sm font-medium text-text-primary"
          >
            Default workspace
          </label>
          {state.wsLoading ? (
            <p className="text-xs text-text-muted">Loading workspaces…</p>
          ) : state.workspaces.length > 0 ? (
            <select
              id="defaultWorkspace"
              className="select"
              value={state.form.defaultWorkspace}
              onChange={(e) =>
                dispatch({
                  type: "SET_FIELD",
                  field: "defaultWorkspace",
                  value: e.target.value,
                })
              }
            >
              <option value="">— none —</option>
              {state.workspaces.map((ws) => (
                <option key={ws.id} value={ws.id}>
                  {ws.name}
                </option>
              ))}
            </select>
          ) : (
            <input
              id="defaultWorkspace"
              type="text"
              className="input"
              placeholder="my-workspace"
              {...field("defaultWorkspace")}
            />
          )}
          {state.wsError && (
            <p className="text-xs text-error">
              Could not load workspaces: {state.wsError}
            </p>
          )}
          <p className="text-xs text-text-muted">
            Clips will be saved here by default.
          </p>
        </fieldset>

        {/* API key */}
        <fieldset className="flex flex-col gap-2">
          <label
            htmlFor="apiKey"
            className="text-sm font-medium text-text-primary"
          >
            API key{" "}
            <span className="font-normal text-text-muted">(optional)</span>
          </label>
          <input
            id="apiKey"
            type="password"
            className="input font-mono"
            placeholder="sk-…"
            autoComplete="off"
            {...field("apiKey")}
          />
          <p className="text-xs text-text-muted">
            Sent as{" "}
            <code className="rounded bg-bg-elevated px-1 text-accent">
              Authorization: Bearer
            </code>{" "}
            on every request. Leave blank for local engines.
          </p>
        </fieldset>

        {/* Auto-tag */}
        <fieldset className="flex items-start gap-3">
          <input
            id="autoTag"
            type="checkbox"
            className="mt-0.5 h-4 w-4 cursor-pointer accent-accent"
            checked={state.form.autoTag}
            onChange={(e) =>
              dispatch({
                type: "SET_FIELD",
                field: "autoTag",
                value: e.target.checked,
              })
            }
          />
          <div>
            <label
              htmlFor="autoTag"
              className="cursor-pointer text-sm font-medium text-text-primary"
            >
              Auto-tag clips
            </label>
            <p className="text-xs text-text-muted">
              Automatically add domain and date tags when clipping.
            </p>
          </div>
        </fieldset>

        {/* Submit */}
        <div className="flex items-center gap-4 border-t border-bg-border pt-4">
          <button
            type="submit"
            className="btn-primary px-5 py-2"
            disabled={state.status === "saving"}
          >
            {state.status === "saving" ? "Saving…" : "Save settings"}
          </button>

          {state.status === "saved" && (
            <p className="text-sm text-success">Settings saved.</p>
          )}
          {state.status === "error" && (
            <p className="text-sm text-error">{state.errorMsg}</p>
          )}
        </div>
      </form>
    </div>
  );
}
