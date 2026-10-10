import type { RichText, TextAnnotations } from "../entities/types";

const defaultAnnotations: TextAnnotations = {
  bold: false,
  italic: false,
  strikethrough: false,
  underline: false,
  code: false,
  color: "default",
};

function textSegment(
  text: string,
  annotations: Partial<TextAnnotations> = {},
  href: string | null = null,
): RichText {
  return {
    type: "text",
    text: { content: text, link: href },
    annotations: { ...defaultAnnotations, ...annotations },
    plain_text: text,
    href,
  };
}

export function plainTextToRichText(text: string): RichText[] {
  return text ? [textSegment(text)] : [];
}

export function extractRichTextPlainText(content: RichText[] | string | null): string {
  if (!content) return "";
  if (typeof content === "string") return content;
  return content.map((rt) => rt.plain_text || "").join("");
}

export function parseInlineMarkdownToRichText(text: string): RichText[] {
  if (!text) return [];

  const segments: RichText[] = [];
  let buffer = "";
  let i = 0;

  const flush = () => {
    if (!buffer) return;
    segments.push(textSegment(buffer));
    buffer = "";
  };

  while (i < text.length) {
    const rest = text.slice(i);

    const link = rest.match(/^\[([^\]]+)\]\(([^)\s]+)\)/);
    if (link) {
      flush();
      segments.push(textSegment(link[1], {}, link[2]));
      i += link[0].length;
      continue;
    }

    const patterns: Array<[RegExp, Partial<TextAnnotations>]> = [
      [/^\*\*([^*]+)\*\*/, { bold: true }],
      [/^__([^_]+)__/, { bold: true }],
      [/^~~([^~]+)~~/, { strikethrough: true }],
      [/^`([^`]+)`/, { code: true }],
      [/^\*([^*]+)\*/, { italic: true }],
      [/^_([^_]+)_/, { italic: true }],
    ];

    let matched = false;
    for (const [pattern, annotations] of patterns) {
      const match = rest.match(pattern);
      if (!match) continue;
      flush();
      segments.push(textSegment(match[1], annotations));
      i += match[0].length;
      matched = true;
      break;
    }

    if (!matched) {
      buffer += text[i];
      i++;
    }
  }

  flush();
  return segments.length > 0 ? segments : plainTextToRichText(text);
}

function escapeHtml(value: string): string {
  return value
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

export function richTextToHTML(content: RichText[] | string | null): string {
  if (!content) return "";
  if (typeof content === "string") return escapeHtml(content);

  return content
    .map((segment) => {
      let html = escapeHtml(segment.plain_text || "");
      if (segment.annotations.code) html = `<code class="inline-code">${html}</code>`;
      if (segment.annotations.bold) html = `<strong>${html}</strong>`;
      if (segment.annotations.italic) html = `<em>${html}</em>`;
      if (segment.annotations.underline) html = `<u>${html}</u>`;
      if (segment.annotations.strikethrough) html = `<s>${html}</s>`;
      if (segment.href) html = `<a href="${escapeHtml(segment.href)}">${html}</a>`;
      return html;
    })
    .join("");
}

function escapeMarkdown(value: string): string {
  return value.replace(/\\/g, "\\\\").replace(/`/g, "\\`");
}

export function richTextToMarkdown(content: RichText[] | string | null): string {
  if (!content) return "";
  if (typeof content === "string") return content;

  return content
    .map((segment) => {
      let text = escapeMarkdown(segment.plain_text || "");
      if (segment.annotations.code) text = `\`${text}\``;
      if (segment.annotations.bold) text = `**${text}**`;
      if (segment.annotations.italic) text = `_${text}_`;
      if (segment.annotations.strikethrough) text = `~~${text}~~`;
      if (segment.href) text = `[${text}](${segment.href})`;
      return text;
    })
    .join("");
}
