// Resolves the "BusinessOS home" - the folder the built-in terminal opens in
// and agents cd into for context. The main process auto-detects it (dev repo
// root, or ~/BusinessOS for packaged users) and lets the user override it;
// this is the renderer-side accessor with a one-time cache.
//
// Falls back to "~" on web / when the desktop bridge is unavailable, so the
// terminal still starts somewhere sane.

interface DesktopBridge {
  getBusinessOSHome?: () => Promise<string>;
  setBusinessOSHome?: (dir: string) => Promise<{ ok: boolean; home?: string }>;
}

function bridge(): DesktopBridge | null {
  if (typeof window === "undefined") return null;
  return (window as unknown as { electron?: DesktopBridge }).electron ?? null;
}

let cached: string | null = null;

// Returns an ABSOLUTE path, or "" when the desktop bridge can't resolve one.
// Never returns the literal "~" - callers must not `cd "~"` (quoted tilde does
// not expand and the cd fails). An empty string means "leave the shell where it
// already is" (the PTY already spawns in the resolved cwd).
export async function getBusinessOSHome(): Promise<string> {
  if (cached) return cached;
  const b = bridge();
  if (b?.getBusinessOSHome) {
    try {
      const home = await b.getBusinessOSHome();
      if (home && home !== "~") {
        cached = home;
        return home;
      }
    } catch {
      // fall through to default
    }
  }
  return "";
}

export async function setBusinessOSHome(dir: string): Promise<boolean> {
  const b = bridge();
  if (!b?.setBusinessOSHome) return false;
  try {
    const res = await b.setBusinessOSHome(dir);
    if (res.ok) {
      cached = res.home ?? dir;
      return true;
    }
  } catch {
    // ignore
  }
  return false;
}
