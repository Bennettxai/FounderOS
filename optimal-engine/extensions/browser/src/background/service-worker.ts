import { clipMemory } from "@/lib/client";
import { getSettings } from "@/lib/storage";
import type { ExtensionMessage, PageInfo } from "@/lib/types";

// ─── Context menus ────────────────────────────────────────────────────────────

const MENU_CLIP_SELECTION = "oe_clip_selection";
const MENU_CLIP_LINK = "oe_clip_link";

function createContextMenus() {
  chrome.contextMenus.removeAll(() => {
    chrome.contextMenus.create({
      id: MENU_CLIP_SELECTION,
      title: "Save to Optimal Engine",
      contexts: ["selection"],
    });
    chrome.contextMenus.create({
      id: MENU_CLIP_LINK,
      title: "Clip URL to Optimal Engine",
      contexts: ["link"],
    });
  });
}

chrome.runtime.onInstalled.addListener(createContextMenus);
chrome.runtime.onStartup.addListener(createContextMenus);

// ─── Context menu click handler ───────────────────────────────────────────────

chrome.contextMenus.onClicked.addListener(async (info, tab) => {
  const settings = await getSettings();
  const workspace = settings.defaultWorkspace;

  if (!settings.engineUrl) {
    chrome.runtime.openOptionsPage();
    return;
  }

  if (info.menuItemId === MENU_CLIP_SELECTION && info.selectionText) {
    const content = info.selectionText.trim();
    const url = info.pageUrl ?? tab?.url ?? "";
    const title = tab?.title ?? url;

    await clipMemory({
      content,
      workspace,
      citation_uri: url,
      metadata: {
        url,
        title,
        clipped_at: new Date().toISOString(),
        source: "context-menu-selection",
      },
    });

    showNotification("Clipped!", `Selection saved to workspace "${workspace}"`);
  }

  if (info.menuItemId === MENU_CLIP_LINK && info.linkUrl) {
    const url = info.linkUrl;

    await clipMemory({
      content: url,
      workspace,
      citation_uri: url,
      metadata: {
        url,
        title: url,
        clipped_at: new Date().toISOString(),
        source: "context-menu-link",
      },
    });

    showNotification("Clipped!", `URL saved to workspace "${workspace}"`);
  }
});

// ─── Message router ───────────────────────────────────────────────────────────

chrome.runtime.onMessage.addListener(
  (message: ExtensionMessage, _sender, sendResponse) => {
    handleMessage(message)
      .then(sendResponse)
      .catch((err: unknown) => {
        sendResponse({
          ok: false,
          error: err instanceof Error ? err.message : "Unknown error",
        });
      });
    // Return true to keep message channel open for async response
    return true;
  },
);

async function handleMessage(
  message: ExtensionMessage,
): Promise<Record<string, unknown>> {
  const settings = await getSettings();

  if (message.type === "CLIP_PAGE" || message.type === "CLIP_SELECTION") {
    const info = message.payload as PageInfo;
    const content =
      message.type === "CLIP_SELECTION" && info.selectedText
        ? info.selectedText
        : info.bodyText.slice(0, 1000);

    const result = await clipMemory({
      content,
      workspace: settings.defaultWorkspace,
      citation_uri: info.url,
      metadata: {
        url: info.url,
        title: info.title,
        clipped_at: new Date().toISOString(),
        source:
          message.type === "CLIP_SELECTION" ? "popup-selection" : "popup-page",
      },
    });

    return result as Record<string, unknown>;
  }

  return { ok: false, error: `Unhandled message type: ${message.type}` };
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

function showNotification(title: string, message: string) {
  // Basic notification — only shown if "notifications" permission is added
  // Currently we silently succeed; this is a no-op but ready to enable.
  void title;
  void message;
}
