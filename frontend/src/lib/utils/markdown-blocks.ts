import type { EditorBlock, BlockType } from '$lib/stores/editor';

// Generate unique block ID
function generateBlockId(): string {
	return Math.random().toString(36).substring(2, 11);
}

// Strip inline markdown formatting (bold, italic, etc.)
function stripInlineFormatting(text: string): string {
	return text
		// Bold: **text** or __text__
		.replace(/\*\*([^*]+)\*\*/g, '$1')
		.replace(/__([^_]+)__/g, '$1')
		// Italic: *text* or _text_
		.replace(/\*([^*]+)\*/g, '$1')
		.replace(/_([^_]+)_/g, '$1')
		// Strikethrough: ~~text~~
		.replace(/~~([^~]+)~~/g, '$1')
		// Inline code: `code`
		.replace(/`([^`]+)`/g, '$1')
		// Links: [text](url) -> text
		.replace(/\[([^\]]+)\]\([^)]+\)/g, '$1');
}

// Create an empty block of given type
function createBlock(type: BlockType, content: string = '', properties?: EditorBlock['properties'], preserveFormatting: boolean = false): EditorBlock {
	return {
		id: generateBlockId(),
		type,
		content: preserveFormatting ? content : stripInlineFormatting(content),
		properties
	};
}

/**
 * Convert markdown string to EditorBlock array
 */
export function markdownToBlocks(markdown: string): EditorBlock[] {
	if (!markdown || !markdown.trim()) {
		return [createBlock('paragraph', '')];
	}

	const blocks: EditorBlock[] = [];
	const lines = markdown.split('\n');
	let i = 0;

	while (i < lines.length) {
		const line = lines[i];

		// Code block (fenced) - preserve content as-is
		if (line.startsWith('```')) {
			const language = line.slice(3).trim() || 'plaintext';
			const codeLines: string[] = [];
			i++;
			while (i < lines.length && !lines[i].startsWith('```')) {
				codeLines.push(lines[i]);
				i++;
			}
			blocks.push(createBlock('code', codeLines.join('\n'), { language }, true));
			i++; // Skip closing ```
			continue;
		}

		// Divider
		if (line.match(/^---+\s*$/) || line.match(/^\*\*\*+\s*$/) || line.match(/^___+\s*$/)) {
			blocks.push(createBlock('divider', ''));
			i++;
			continue;
		}

		// Heading 1
		if (line.startsWith('# ')) {
			blocks.push(createBlock('heading1', line.slice(2).trim()));
			i++;
			continue;
		}

		// Heading 2
		if (line.startsWith('## ')) {
			blocks.push(createBlock('heading2', line.slice(3).trim()));
			i++;
			continue;
		}

		// Heading 3
		if (line.startsWith('### ')) {
			blocks.push(createBlock('heading3', line.slice(4).trim()));
			i++;
			continue;
		}

		// Todo (unchecked)
		if (line.match(/^[-*]\s*\[\s*\]/)) {
			const content = line.replace(/^[-*]\s*\[\s*\]\s*/, '').trim();
			blocks.push(createBlock('todo', content, { checked: false }));
			i++;
			continue;
		}

		// Todo (checked)
		if (line.match(/^[-*]\s*\[[xX]\]/)) {
			const content = line.replace(/^[-*]\s*\[[xX]\]\s*/, '').trim();
			blocks.push(createBlock('todo', content, { checked: true }));
			i++;
			continue;
		}

		// Bullet list
		if (line.match(/^[-*]\s+/)) {
			const content = line.replace(/^[-*]\s+/, '').trim();
			blocks.push(createBlock('bulletList', content));
			i++;
			continue;
		}

		// Numbered list
		if (line.match(/^\d+\.\s+/)) {
			const content = line.replace(/^\d+\.\s+/, '').trim();
			blocks.push(createBlock('numberedList', content));
			i++;
			continue;
		}

		// Quote
		if (line.startsWith('> ')) {
			const content = line.slice(2).trim();
			blocks.push(createBlock('quote', content));
			i++;
			continue;
		}

		// Empty line - skip
		if (line.trim() === '') {
			i++;
			continue;
		}

		// Regular paragraph
		blocks.push(createBlock('paragraph', line.trim()));
		i++;
	}

	// Ensure at least one block
	if (blocks.length === 0) {
		blocks.push(createBlock('paragraph', ''));
	}

	return blocks;
}

/**
 * Convert EditorBlock array back to markdown string
 */
export function blocksToMarkdown(blocks: EditorBlock[]): string {
	if (!blocks || blocks.length === 0) {
		return '';
	}

	const lines: string[] = [];

	for (const block of blocks) {
		switch (block.type) {
			case 'heading1':
				lines.push(`# ${block.content}`);
				lines.push('');
				break;

			case 'heading2':
				lines.push(`## ${block.content}`);
				lines.push('');
				break;

			case 'heading3':
				lines.push(`### ${block.content}`);
				lines.push('');
				break;

			case 'bulletList':
				lines.push(`- ${block.content}`);
				break;

			case 'numberedList':
				lines.push(`1. ${block.content}`);
				break;

			case 'todo':
				const checkbox = block.properties?.checked ? '[x]' : '[ ]';
				lines.push(`- ${checkbox} ${block.content}`);
				break;

			case 'quote':
				lines.push(`> ${block.content}`);
				lines.push('');
				break;

			case 'code':
				const language = block.properties?.language || '';
				lines.push(`\`\`\`${language}`);
				lines.push(block.content);
				lines.push('```');
				lines.push('');
				break;

			case 'divider':
				lines.push('---');
				lines.push('');
				break;

			case 'callout':
				// Render callout as a blockquote with emoji prefix
				const calloutType = block.properties?.calloutType || 'info';
				const emoji = { info: 'ℹ️', warning: '⚠️', success: '✅', error: '❌' }[calloutType] || 'ℹ️';
				lines.push(`> ${emoji} ${block.content}`);
				lines.push('');
				break;

			case 'paragraph':
			default:
				if (block.content) {
					lines.push(block.content);
					lines.push('');
				}
				break;
		}
	}

	// Clean up trailing empty lines and normalize
	return lines.join('\n').replace(/\n{3,}/g, '\n\n').trim();
}
