import { describe, expect, it } from "vitest";
import { detectMarkdownShortcut } from "./markdown-shortcuts";

describe("markdown shortcut detection", () => {
	it.each([
		["#", { type: "heading_1", properties: {} }],
		["##", { type: "heading_2", properties: {} }],
		["###", { type: "heading_3", properties: {} }],
		["-", { type: "bulleted_list", properties: {} }],
		["*", { type: "bulleted_list", properties: {} }],
		["1.", { type: "numbered_list", properties: {} }],
		["42.", { type: "numbered_list", properties: {} }],
		[">", { type: "quote", properties: {} }],
		["[]", { type: "to_do", properties: { checked: false } }],
		["[ ]", { type: "to_do", properties: { checked: false } }],
		["[x]", { type: "to_do", properties: { checked: true } }],
		["[X]", { type: "to_do", properties: { checked: true } }],
		["---", { type: "divider", properties: { divider_style: "solid" } }],
		["****", { type: "divider", properties: { divider_style: "solid" } }],
		["____", { type: "divider", properties: { divider_style: "solid" } }],
	])("maps %s to application data", (text, application) => {
		expect(detectMarkdownShortcut(text)).toEqual(application);
	});

	it.each(["", " #", "# ", "####", "1", "1. item", "[y]", "--", "**", "__", "plain text"])(
		"does not match %s",
		(text) => {
			expect(detectMarkdownShortcut(text)).toBeNull();
		},
	);

	it("returns a fresh properties object", () => {
		const first = detectMarkdownShortcut("[x]");
		const second = detectMarkdownShortcut("[x]");

		expect(first?.properties).toEqual({ checked: true });
		expect(first?.properties).not.toBe(second?.properties);
	});
});
