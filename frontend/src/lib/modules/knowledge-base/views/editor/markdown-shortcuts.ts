import type { BlockProperties, BlockType } from "../../entities/types";

export interface MarkdownShortcutApplication {
	type: BlockType;
	properties: Partial<BlockProperties>;
}

interface MarkdownShortcutDefinition {
	id: string;
	matches: (text: string) => boolean;
	application: MarkdownShortcutApplication;
}

export const MARKDOWN_SHORTCUTS: readonly MarkdownShortcutDefinition[] = [
	{
		id: "heading_1",
		matches: (text) => text === "#",
		application: { type: "heading_1", properties: {} },
	},
	{
		id: "heading_2",
		matches: (text) => text === "##",
		application: { type: "heading_2", properties: {} },
	},
	{
		id: "heading_3",
		matches: (text) => text === "###",
		application: { type: "heading_3", properties: {} },
	},
	{
		id: "bulleted_list",
		matches: (text) => text === "-" || text === "*",
		application: { type: "bulleted_list", properties: {} },
	},
	{
		id: "numbered_list",
		matches: (text) => /^\d+\.$/.test(text),
		application: { type: "numbered_list", properties: {} },
	},
	{
		id: "quote",
		matches: (text) => text === ">",
		application: { type: "quote", properties: {} },
	},
	{
		id: "unchecked_to_do",
		matches: (text) => text === "[]" || text === "[ ]",
		application: { type: "to_do", properties: { checked: false } },
	},
	{
		id: "checked_to_do",
		matches: (text) => /^\[x\]$/i.test(text),
		application: { type: "to_do", properties: { checked: true } },
	},
	{
		id: "divider",
		matches: (text) => /^(---+|\*\*\*+|___+)$/.test(text),
		application: { type: "divider", properties: { divider_style: "solid" } },
	},
];

export function detectMarkdownShortcut(text: string): MarkdownShortcutApplication | null {
	const shortcut = MARKDOWN_SHORTCUTS.find((definition) => definition.matches(text));

	if (!shortcut) {
		return null;
	}

	return {
		type: shortcut.application.type,
		properties: { ...shortcut.application.properties },
	};
}
