import { useEffect, useRef, useState } from "react";
import { fetchWorkspaces } from "../lib/client";
import type { Workspace } from "../lib/types";

interface UseWorkspacesResult {
  workspaces: Workspace[];
  isLoading: boolean;
  error: string | null;
  reload: () => void;
}

/**
 * SWR-style hook that fetches the workspace list once and caches it for the
 * lifetime of the component. Calling `reload()` forces a re-fetch.
 */
export function useWorkspaces(tenant?: string): UseWorkspacesResult {
  const [workspaces, setWorkspaces] = useState<Workspace[]>([]);
  const [isLoading, setIsLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);
  const [tick, setTick] = useState<number>(0);

  // Keep the latest tenant ref so the effect always sees the current value
  const tenantRef = useRef<string | undefined>(tenant);
  tenantRef.current = tenant;

  useEffect(() => {
    let cancelled = false;

    async function load(): Promise<void> {
      setIsLoading(true);
      setError(null);

      const result = await fetchWorkspaces(tenantRef.current);

      if (cancelled) return;

      if (result.ok) {
        setWorkspaces(result.data.workspaces);
      } else {
        setError(result.error);
        setWorkspaces([]);
      }

      setIsLoading(false);
    }

    void load();

    return () => {
      cancelled = true;
    };
  }, [tick]);

  function reload(): void {
    setTick((n) => n + 1);
  }

  return { workspaces, isLoading, error, reload };
}
