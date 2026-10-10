<!-- FounderOS v1 components/AgentAvatar.tsx fallback: a round bot glyph. Prod
     shows a photo only when the roster has one; the port has none, so every
     owner reads as the bot, as it does on prod today. -->
<script lang="ts">
	import { Bot } from '$lib/founderos/icons';
	let { size = 24 }: { size?: number } = $props();
</script>

<span data-part="avatar" class="bn-border bn-dim grid shrink-0 place-items-center rounded-full border" style="width: {size}px; height: {size}px; background: var(--bn-surface)" aria-hidden="true">
	<Bot size={Math.max(11, Math.round(size * 0.55))} strokeWidth={1.7} />
</span>
