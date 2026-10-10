import type { ExtensionMessage, PageInfo } from "@/lib/types";

/**
 * Content script — injected into every page at document_idle.
 * Responsibilities:
 *   1. Respond to GET_PAGE_INFO requests from the popup/background.
 *   2. Capture the current text selection.
 */

function getPageInfo(): PageInfo {
  const selectedText = window.getSelection()?.toString().trim() ?? "";

  // Grab meaningful body text — prefer article/main, fall back to body.
  const contentEl =
    document.querySelector("article") ??
    document.querySelector("main") ??
    document.body;

  const bodyText = (contentEl?.innerText ?? "").replace(/\s+/g, " ").trim();

  return {
    url: location.href,
    title: document.title,
    selectedText,
    bodyText: bodyText.slice(0, 4000),
  };
}

chrome.runtime.onMessage.addListener(
  (message: ExtensionMessage, _sender, sendResponse) => {
    if (message.type === "GET_PAGE_INFO") {
      sendResponse(getPageInfo());
    }
    // Return false — synchronous response is fine here.
    return false;
  },
);
