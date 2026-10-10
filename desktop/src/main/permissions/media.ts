// Camera and microphone for the app's own pages (3D OS hand gestures, voice).
// macOS only shows its camera prompt when the app asks for it, so a media
// request from a trusted page is turned into systemPreferences.askForMediaAccess
// for each device it wants. Any other page is refused.

export type MediaDevice = "camera" | "microphone";

export interface MediaRequest {
  permission: string;
  origin: string;
  mediaTypes: string[];
  trustedOrigins: string[];
}

/** null: not a media request, leave it to Electron's default. */
export function mediaPermission(req: MediaRequest): { allow: boolean; ask: MediaDevice[] } | null {
  if (req.permission !== "media") return null;
  if (!req.trustedOrigins.includes(req.origin)) return { allow: false, ask: [] };
  const ask: MediaDevice[] = [];
  if (req.mediaTypes.includes("video")) ask.push("camera");
  if (req.mediaTypes.includes("audio")) ask.push("microphone");
  return { allow: true, ask };
}

/** scheme://host, also for app:// where URL.origin is "null". */
export function originOf(url: string): string {
  try {
    const u = new URL(url);
    return `${u.protocol}//${u.host}`;
  } catch {
    return "";
  }
}
