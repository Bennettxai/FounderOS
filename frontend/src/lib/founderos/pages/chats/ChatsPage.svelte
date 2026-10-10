<!-- /os/chats (FounderOS v1 app/chats/page.tsx): the chat hub. Every
     conversation in one rail (the board Conductor pinned first), the open
     thread beside it, and a direct line to any agent via New chat. The page
     owns one viewport of height on wide screens; the hub scrolls inside. -->
<script lang="ts">
	import { onMount } from 'svelte';
	import { founderosFetch } from '$lib/founderos/api';
	import PageHeader from '$lib/founderos/kit/PageHeader.svelte';
	import Rise from '$lib/founderos/kit/Rise.svelte';
	import ChatHub from './ChatHub.svelte';
	import type { ChatsView } from './types';

	let view = $state<ChatsView | null>(null);
	let error = $state<string | null>(null);

	onMount(() => {
		founderosFetch<ChatsView>('/pages/chats')
			.then((v) => (view = v))
			.catch((err) => (error = err instanceof Error ? err.message : String(err)));
	});
</script>

<div class="flex flex-col xl:h-[calc(100dvh-9rem)]">
	<PageHeader eyebrow="every conversation" title="Chats" />
	<Rise i={1} class="flex min-h-0 flex-1 flex-col">
		{#if error}
			<p class="font-mono text-[11px]" style:color="var(--bn-err)">Could not load the chats: {error}</p>
		{:else if !view}
			<div class="bn-skeleton min-h-[560px] flex-1 rounded-2xl" aria-hidden="true"></div>
		{:else}
			<ChatHub initialSummaries={view.summaries} roster={view.roster} conductorModel={view.conductorModel} boardUrl={view.boardUrl} />
		{/if}
	</Rise>
</div>
