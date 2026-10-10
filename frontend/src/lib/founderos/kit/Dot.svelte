<!-- Round LED status dot (FounderOS v1 terminal.tsx Dot). Only an ok dot blinks. -->
<script lang="ts">
	import './kit.css';
	import { dotState } from './format';

	let { state, pulse = false }: { state: string; pulse?: boolean } = $props();
	const cls = $derived(dotState(state));
</script>

<span class="bn-dot {cls}" class:pulse={pulse && cls === 'ok'} data-state={cls} aria-hidden="true"></span>
