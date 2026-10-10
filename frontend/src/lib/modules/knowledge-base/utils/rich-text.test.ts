import { describe, expect, it } from "vitest";
import {
  extractRichTextPlainText,
  parseInlineMarkdownToRichText,
  richTextToHTML,
  richTextToMarkdown,
} from "./rich-text";

describe("knowledge-base rich text utilities", () => {
  it("parses inline markdown marks into rich text annotations", () => {
    const richText = parseInlineMarkdownToRichText(
      "Plan **Monday** with `signals/YYYY-MM-DD.md` and [docs](https://example.com).",
    );

    expect(extractRichTextPlainText(richText)).toBe(
      "Plan Monday with signals/YYYY-MM-DD.md and docs.",
    );
    expect(richText.some((segment) => segment.plain_text === "Monday" && segment.annotations.bold)).toBe(true);
    expect(
      richText.some(
        (segment) => segment.plain_text === "signals/YYYY-MM-DD.md" && segment.annotations.code,
      ),
    ).toBe(true);
    expect(richText.some((segment) => segment.plain_text === "docs" && segment.href)).toBe(true);
  });

  it("renders rich text as editor-safe html", () => {
    const richText = parseInlineMarkdownToRichText("**Bold** and `code`");

    expect(richTextToHTML(richText)).toContain("<strong>Bold</strong>");
    expect(richTextToHTML(richText)).toContain('<code class="inline-code">code</code>');
  });

  it("serializes annotations back to markdown", () => {
    const richText = parseInlineMarkdownToRichText("**Bold** and ~~done~~");

    expect(richTextToMarkdown(richText)).toBe("**Bold** and ~~done~~");
  });
});
