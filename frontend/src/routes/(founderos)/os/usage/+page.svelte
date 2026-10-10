<!-- /usage (spec 6.11): the token-burn board in the Brand Deals look. The title
     row is static; every number below it is the live board's, polled every 10s. -->
<script lang="ts">
	import { founderosUrl } from '$lib/founderos/api';
	import { Slab, SlabTitle } from '$lib/founderos/kit';
	import UsageBoard from '$lib/founderos/pages/usage/UsageBoard.svelte';
	import { PILL } from '$lib/founderos/pages/usage/ui';
</script>

<Slab>
	<SlabTitle
		eyebrow="token burn · one plan per provider"
		title="Usage"
		meta="Claude · ChatGPT / Codex · Ollama · local file parsing only · no paid calls · refreshes every 10s"
	>
		{#snippet right()}
			<a href={founderosUrl('/pages/usage')} target="_blank" rel="noreferrer" class={PILL}>Raw reading</a>
		{/snippet}
	</SlabTitle>
	<UsageBoard />
</Slab>
