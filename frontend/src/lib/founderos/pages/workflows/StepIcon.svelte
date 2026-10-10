<!-- prod WorkflowTree stepIcon(): human → User, planned → CircleDashed, live → Zap, else Bot -->
<script lang="ts">
	import { Bot, CircleDashed, User, Zap } from '$lib/founderos/icons';
	import type { WorkflowStep } from './types';
	let { step, class: className = '', color }: { step: WorkflowStep; class?: string; color: string } = $props();
	const Icon = $derived(step.ownerKind === 'human' ? User : step.automation?.state === 'suggested' ? CircleDashed : step.automation?.state === 'live' ? Zap : Bot);
</script>

<Icon class={className} strokeWidth={1.7} style="color: {color}" />
