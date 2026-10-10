/**
 * markdownRenderer.ts
 *
 * Shared markdown-to-HTML renderer for the BOS frontend.
 *
 * Security model:
 *   1. HTML-escape the raw input first (eliminates injected tags/attributes).
 *   2. Apply regex-based markdown transformations on the escaped string.
 *   3. Run DOMPurify as a final sanitization pass (SSR-safe: no-op on server).
 *
 * Usage:
 *   import { renderMarkdown } from '$lib/utils/markdownRenderer';
 *
 *   // Full mode — bold, italic, inline code, H1-H3, bullet/numbered lists, paragraphs
 *   {@html renderMarkdown(message.content)}
 *
 *   // Simple mode — bold, inline code, line breaks only (e.g. OSA response stream)
 *   {@html renderMarkdown(message.content, { simple: true })}
 */

import { browser } from "$app/environment";
import DOMPurify from "dompurify";

export interface MarkdownOptions {
  /** When true, only bold, inline code, and line breaks are rendered (no headers, lists, italic). */
  simple?: boolean;
}

/**
 * Escape the five HTML special characters to prevent injection before regex
 * transformations re-introduce safe markup.
 */
function escapeHtml(text: string): string {
  return text
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");
}

/**
 * Wrap consecutive `<li>` elements that carry `list-disc` with a `<ul>`,
 * and those that carry `list-decimal` with an `<ol>`.
 *
 * This is applied as a post-processing step after all inline replacements so
 * that the individual `<li>` lines produced by the regex pass are properly
 * grouped into their parent list containers.
 */
function wrapLists(html: string): string {
  // Wrap unordered list items
  html = html.replace(
    /(<li class="ml-4 list-disc">.*?<\/li>)(\s*<li class="ml-4 list-disc">.*?<\/li>)*/gs,
    (match) => `<ul class="mb-2 mt-1 space-y-0.5">${match}</ul>`,
  );

  // Wrap ordered list items
  html = html.replace(
    /(<li class="ml-4 list-decimal">.*?<\/li>)(\s*<li class="ml-4 list-decimal">.*?<\/li>)*/gs,
    (match) => `<ol class="mb-2 mt-1 space-y-0.5">${match}</ol>`,
  );

  return html;
}

/**
 * Render markdown text to sanitized HTML.
 *
 * @param text    Raw markdown string (may contain user-supplied content).
 * @param options Optional rendering options.
 * @returns       Sanitized HTML string safe for use with `{@html ...}`.
 */
export function renderMarkdown(
  text: string,
  options: MarkdownOptions = {},
): string {
  if (!text) return "";

  const { simple = false } = options;

  // Step 1 — escape HTML to neutralise any injected markup
  let html = escapeHtml(text);

  if (simple) {
    // Simple mode: bold, inline code, line breaks only
    html = html
      .replace(/\*\*(.+?)\*\*/g, "<strong>$1</strong>")
      .replace(
        /`([^`]+)`/g,
        '<code class="bg-gray-100 px-1 py-0.5 rounded text-sm font-mono dark:bg-gray-700">$1</code>',
      )
      .replace(/\n/g, "<br />");
  } else {
    // Full mode: headers, bold, italic, inline code, lists, paragraphs

    // Headers (must run before bold/italic to avoid double-processing)
    html = html
      .replace(
        /^### (.*$)/gm,
        '<h3 class="text-base font-semibold mt-4 mb-2">$1</h3>',
      )
      .replace(
        /^## (.*$)/gm,
        '<h2 class="text-lg font-semibold mt-4 mb-2">$1</h2>',
      )
      .replace(/^# (.*$)/gm, '<h1 class="text-xl font-bold mt-4 mb-2">$1</h1>');

    // Lists — individual items (wrapping happens in wrapLists below)
    html = html
      .replace(/^- (.*$)/gm, '<li class="ml-4 list-disc">$1</li>')
      .replace(/^\d+\. (.*$)/gm, '<li class="ml-4 list-decimal">$1</li>');

    // Inline: bold, then italic, then code
    html = html
      .replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>")
      .replace(/\*([^*]+)\*/g, "<em>$1</em>")
      .replace(
        /`([^`]+)`/g,
        '<code class="bg-gray-100 px-1 py-0.5 rounded text-sm font-mono dark:bg-gray-700">$1</code>',
      );

    // Paragraphs and line breaks
    // Double newline → paragraph break; single newline → <br>
    html = html
      .replace(/\n\n/g, '</p><p class="mb-2">')
      .replace(/\n/g, "<br />");

    // Wrap bare list items in <ul>/<ol> containers
    html = wrapLists(html);
  }

  // Step 3 — DOMPurify final pass (browser only; SSR returns as-is)
  if (browser) {
    html = DOMPurify.sanitize(html, {
      ALLOWED_TAGS: [
        "strong",
        "em",
        "code",
        "br",
        "p",
        "h1",
        "h2",
        "h3",
        "ul",
        "ol",
        "li",
      ],
      ALLOWED_ATTR: ["class"],
    });
  }

  return html;
}
