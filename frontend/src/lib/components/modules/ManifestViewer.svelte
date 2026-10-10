<script lang="ts">
	import type { ModuleManifest } from '$lib/types/modules';

	interface Props {
		manifest: ModuleManifest;
	}

	let { manifest }: Props = $props();

	const jsonString = $derived(JSON.stringify(manifest, null, 2));
</script>

<div class="am-manifest">
	<pre class="am-manifest__code"><code>{jsonString}</code></pre>
</div>

<style>
	.am-manifest {
		background: #1a1a1a;
		border-radius: 12px;
		padding: 16px;
		overflow-x: auto;
		border: 1px solid var(--dbd, #333);
	}
	.am-manifest__code {
		font-size: 13px;
		color: #e5e5e5;
		font-family: monospace;
		margin: 0;
	}
</style>
