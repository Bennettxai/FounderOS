<!-- The org board's 6px LED (app/org STATUS_DOT, lib/org-live LIVE_DOT): a
     filled round dot, or prod's hollow ring for planned. Not the kit Dot,
     whose states are the green/amber connector LEDs. -->
<script lang="ts">
	import type { OrgDotLook } from './org';

	let { look }: { look: OrgDotLook } = $props();
</script>

<span
	data-part="org-dot"
	aria-hidden="true"
	class="inline-block h-1.5 w-1.5 shrink-0 rounded-full"
	class:animate-pulse={look.pulse}
	class:border={!look.fill}
	style={look.fill ? `background: ${look.fill}` : 'background: transparent; border-color: var(--bn-text-3)'}
></span>
