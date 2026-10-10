<script lang="ts">
	import { fade } from 'svelte/transition';
</script>

<div class="flex justify-start" in:fade={{ duration: 200 }}>
	<div class="max-w-[85%]">
		<div class="flex items-center gap-2 mb-2">
			<div class="w-7 h-7 rounded-lg bg-gray-100 flex items-center justify-center">
				<svg class="w-4 h-4 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
				</svg>
			</div>
			<span class="text-sm font-medium text-gray-700">Assistant</span>
		</div>

		<div class="bg-gray-50 border border-gray-100 rounded-2xl rounded-tl-md px-5 py-4">
			<div class="flex items-center gap-1">
				<span class="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style="animation-delay: 0ms"></span>
				<span class="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style="animation-delay: 150ms"></span>
				<span class="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style="animation-delay: 300ms"></span>
			</div>
		</div>
	</div>
</div>

<style>
	@keyframes bounce {
		0%, 80%, 100% { transform: translateY(0); }
		40% { transform: translateY(-6px); }
	}
</style>
