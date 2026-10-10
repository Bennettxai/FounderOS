<!-- Home's copy of the Needs you queue (FounderOS v1 components/HomeNeedsYou.tsx):
     the same NeedsYouList the Agents tab mounts, on the same payload
     (GET /pages/board/deliverables), filling its row-mate's box. This wrapper
     owns the fetch and the per-row opened store, as prod's useDeliverables
     does; the list answers in place (the outward call goes through the bridge
     guard), confirms on the OS-wide toast stack and opens the slide-in review. -->
<script lang="ts">
	import { onMount } from 'svelte';
	import { FounderosApiError, founderosFetch } from '$lib/founderos/api';
	import NeedsYouList from '../agents/NeedsYouList.svelte';
	import type { DeliverableItem, DeliverablesBody } from '../agents/types';
	import { OPENED_KEY, markManyOpened, markOpened, parseOpenedMap, pruneOpened, readStore, writeStore, type OpenedMap } from './needs-queue';

	type Body = DeliverablesBody & { boardUrl?: string | null };

	let body = $state<Body | null>(null);
	let error = $state<string | null>(null);
	let refreshing = $state(false);
	let opened = $state<OpenedMap | null>(null);

	async function reload() {
		refreshing = true;
		try {
			const next = await founderosFetch<Body>('/pages/board/deliverables');
			body = next;
			error = null;
			adopt(next);
		} catch (e) {
			if (!body) error = e instanceof FounderosApiError && e.status === 404 ? 'not on this bridge' : e instanceof Error ? e.message : 'unreachable';
		} finally {
			refreshing = false;
		}
	}

	/** First load in a fresh browser adopts everything silently: a wall of dots on
	    every file is noise. After that, prune ids that are gone. */
	function adopt(b: Body) {
		const all = (b.groups ?? []).flatMap((g) => g.items ?? []);
		if (opened === null) {
			opened = markManyOpened({}, all.map((i) => ({ id: i.id, revision: i.revision })));
			writeStore(OPENED_KEY, opened);
			return;
		}
		const pruned = pruneOpened(opened, all.map((i) => i.id));
		if (Object.keys(pruned).length !== Object.keys(opened).length) {
			opened = pruned;
			writeStore(OPENED_KEY, pruned);
		}
	}

	function markItemOpened(item: DeliverableItem) {
		opened = markOpened(opened ?? {}, item.id, item.revision);
		writeStore(OPENED_KEY, opened);
	}

	onMount(() => {
		opened = parseOpenedMap(readStore(OPENED_KEY));
		void reload();
	});
</script>

<NeedsYouList data={body} {error} boardUrl={body?.boardUrl ?? null} {refreshing} {opened} onOpen={markItemOpened} onReload={reload} fill />
