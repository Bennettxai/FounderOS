<script lang="ts">
	let { line1, line2 }: { line1: string; line2: string } = $props();
</script>

<div class="flex h-full min-h-[200px] flex-col items-center justify-center text-center" data-part="empty">
	<p class="bn-muted text-[13px]">{line1}</p>
	<p class="bn-dim mt-1 max-w-md text-[12px]">{line2}</p>
</div>
