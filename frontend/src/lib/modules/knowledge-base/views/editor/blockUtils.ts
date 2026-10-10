import type { RichText } from "../../entities/types";
import { plainTextToRichText } from "../../utils/rich-text";

/**
 * Parses innerHTML from a contenteditable element into a RichText[] array.
 * Handles bold, italic, underline, strikethrough, code, and link annotations.
 */
export function parseHTMLToRichText(
  html: string,
  plainText: string,
): RichText[] {
  const segments: RichText[] = [];

  if (html === plainText || !html.includes("<")) {
    return plainTextToRichText(plainText);
  }

  const temp = document.createElement("div");
  temp.innerHTML = html;

  function processNode(node: Node, annotations: RichText["annotations"]): void {
    if (node.nodeType === Node.TEXT_NODE) {
      const text = node.textContent || "";
      if (text) {
        segments.push({
          type: "text",
          text: { content: text, link: null },
          annotations: { ...annotations },
          plain_text: text,
          href: null,
        });
      }
    } else if (node.nodeType === Node.ELEMENT_NODE) {
      const element = node as Element;
      const newAnnotations = { ...annotations };

      const tagName = element.tagName.toLowerCase();
      if (tagName === "b" || tagName === "strong") newAnnotations.bold = true;
      if (tagName === "i" || tagName === "em") newAnnotations.italic = true;
      if (tagName === "u") newAnnotations.underline = true;
      if (tagName === "s" || tagName === "strike" || tagName === "del")
        newAnnotations.strikethrough = true;
      if (tagName === "code") newAnnotations.code = true;

      if (tagName === "a") {
        const href = element.getAttribute("href");
        if (href) {
          const text = element.textContent || "";
          segments.push({
            type: "text",
            text: { content: text, link: href },
            annotations: newAnnotations,
            plain_text: text,
            href,
          });
          return;
        }
      }

      node.childNodes.forEach((child) => processNode(child, newAnnotations));
    }
  }

  const defaultAnnotations: RichText["annotations"] = {
    bold: false,
    italic: false,
    strikethrough: false,
    underline: false,
    code: false,
    color: "default",
  };

  temp.childNodes.forEach((child) => processNode(child, defaultAnnotations));

  return segments.length > 0
    ? segments
    : plainTextToRichText(plainText);
}
