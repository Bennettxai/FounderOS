<script lang="ts">
	import { editor, type EditorBlock } from '$lib/stores/editor';

	interface Props {
		block: EditorBlock;
		readonly: boolean;
		isEmpty: boolean;
		onFocus: () => void;
		onBlur: (e: FocusEvent) => void;
		onInput: (e: Event) => void;
		onKeydown: (e: KeyboardEvent) => void;
		onBindElement: (el: HTMLElement | null) => void;
	}

	let {
		block,
		readonly,
		isEmpty,
		onFocus,
		onBlur,
		onInput,
		onKeydown,
		onBindElement,
	}: Props = $props();

	const LANGUAGES = [
		{ id: 'plain', label: 'Plain Text' },
		{ id: 'javascript', label: 'JavaScript' },
		{ id: 'typescript', label: 'TypeScript' },
		{ id: 'python', label: 'Python' },
		{ id: 'go', label: 'Go' },
		{ id: 'rust', label: 'Rust' },
		{ id: 'java', label: 'Java' },
		{ id: 'c', label: 'C' },
		{ id: 'cpp', label: 'C++' },
		{ id: 'csharp', label: 'C#' },
		{ id: 'ruby', label: 'Ruby' },
		{ id: 'php', label: 'PHP' },
		{ id: 'swift', label: 'Swift' },
		{ id: 'kotlin', label: 'Kotlin' },
		{ id: 'sql', label: 'SQL' },
		{ id: 'html', label: 'HTML' },
		{ id: 'css', label: 'CSS' },
		{ id: 'json', label: 'JSON' },
		{ id: 'yaml', label: 'YAML' },
		{ id: 'markdown', label: 'Markdown' },
		{ id: 'bash', label: 'Bash' },
		{ id: 'shell', label: 'Shell' },
		{ id: 'dockerfile', label: 'Dockerfile' },
		{ id: 'graphql', label: 'GraphQL' },
		{ id: 'svelte', label: 'Svelte' },
		{ id: 'vue', label: 'Vue' },
		{ id: 'jsx', label: 'JSX' },
		{ id: 'tsx', label: 'TSX' },
		{ id: 'xml', label: 'XML' },
		{ id: 'toml', label: 'TOML' },
	];

	let showLanguagePicker = $state(false);
	let languageSearchQuery = $state('');
	let languagePickerRef: HTMLDivElement | null = $state(null);

	let filteredLanguages = $derived(
		languageSearchQuery
			? LANGUAGES.filter(lang =>
				lang.label.toLowerCase().includes(languageSearchQuery.toLowerCase()) ||
				lang.id.toLowerCase().includes(languageSearchQuery.toLowerCase())
			)
			: LANGUAGES
	);

	function selectLanguage(langId: string) {
		editor.updateBlock(block.id, block.content, { ...block.properties, language: langId });
		showLanguagePicker = false;
		languageSearchQuery = '';
	}

	function getLanguageLabel(langId: string | undefined): string {
		if (!langId) return 'Plain Text';
		const found = LANGUAGES.find(l => l.id === langId);
		return found?.label || langId;
	}

	function handleLanguagePickerKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			showLanguagePicker = false;
			languageSearchQuery = '';
		}
	}

	function handleLanguagePickerClickOutside(e: MouseEvent) {
		if (languagePickerRef && !languagePickerRef.contains(e.target as Node)) {
			showLanguagePicker = false;
			languageSearchQuery = '';
		}
	}

	$effect(() => {
		if (showLanguagePicker) {
			globalThis.document.addEventListener('click', handleLanguagePickerClickOutside);
			return () => globalThis.document.removeEventListener('click', handleLanguagePickerClickOutside);
		}
	});

	// Svelte 5 action to bind the element reference to the parent
	function bindElement(node: HTMLElement) {
		onBindElement(node);
		return {
			destroy() {
				onBindElement(null);
			}
		};
	}
</script>

<div class="code-block rounded-md overflow-hidden border border-gray-200 dark:border-transparent">
	<!-- Language selector bar -->
	<div class="flex items-center justify-between px-3 py-1.5 bg-gray-100 dark:bg-[var(--bos-v2-layer-background-secondary)] border-b border-gray-200 dark:border-[var(--bos-v2-layer-insideBorder-border)]">
		<!-- Language picker dropdown -->
		<div class="relative" bind:this={languagePickerRef}>
			<button
				onclick={(e) => { e.stopPropagation(); showLanguagePicker = !showLanguagePicker; }}
				class="flex items-center gap-1.5 text-xs text-gray-600 dark:text-gray-400 font-mono hover:text-gray-800 dark:hover:text-gray-200 transition-colors"
				tabindex="-1"
			>
				<span>{getLanguageLabel(block.properties?.language as string | undefined)}</span>
				<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
				</svg>
			</button>

			{#if showLanguagePicker}
				<div class="absolute left-0 top-full mt-1 w-48 max-h-64 bg-white dark:bg-[var(--bos-v2-layer-background-overlayPanel)] rounded-lg shadow-xl border border-gray-200 dark:border-[var(--bos-v2-layer-insideBorder-border)] overflow-hidden z-50">
					<!-- Search input -->
					<div class="p-2 border-b border-gray-200 dark:border-[var(--bos-v2-layer-insideBorder-border)]">
						<input
							type="text"
							bind:value={languageSearchQuery}
							onkeydown={handleLanguagePickerKeydown}
							placeholder="Search languages..."
							class="w-full px-2 py-1.5 text-xs bg-gray-50 dark:bg-[var(--bos-v2-layer-background-primary)] border border-gray-200 dark:border-[var(--bos-v2-layer-insideBorder-border)] rounded text-gray-700 dark:text-gray-200 placeholder:text-gray-400 focus:outline-none focus:ring-1 focus:ring-blue-500"
						/>
					</div>
					<!-- Language list -->
					<div class="overflow-y-auto max-h-48">
						{#each filteredLanguages as lang}
							<button
								onclick={() => selectLanguage(lang.id)}
								class="w-full px-3 py-2 text-left text-xs text-gray-700 dark:text-gray-200 hover:bg-gray-100 dark:hover:bg-[var(--dbg2)] transition-colors flex items-center justify-between"
							>
								<span>{lang.label}</span>
								{#if block.properties?.language === lang.id}
									<svg class="w-3 h-3 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
									</svg>
								{/if}
							</button>
						{/each}
						{#if filteredLanguages.length === 0}
							<div class="px-3 py-4 text-xs text-gray-500 text-center">
								No languages found
							</div>
						{/if}
					</div>
				</div>
			{/if}
		</div>
		<button
			onclick={() => {
				if (block.content) {
					navigator.clipboard.writeText(block.content);
				}
			}}
			class="text-xs text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 transition-colors"
			tabindex="-1"
		>
			Copy
		</button>
	</div>
	<pre
		use:bindElement
		contenteditable={!readonly}
		data-block-id={block.id}
		data-placeholder="// code..."
		onfocus={onFocus}
		onblur={onBlur}
		oninput={onInput}
		onkeydown={onKeydown}
		class="bg-gray-50 dark:bg-[var(--bos-v2-layer-background-primary)] text-gray-800 dark:text-[var(--dt)] font-mono text-sm p-4 outline-none min-h-[2.5em] whitespace-pre-wrap block-editable"
		class:is-empty={isEmpty}
	></pre>
</div>
