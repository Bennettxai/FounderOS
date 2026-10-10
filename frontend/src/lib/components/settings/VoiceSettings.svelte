<script lang="ts">
	// Voice settings tab — currently a navigation hub to the full voice notes page.
	// Future: expose audio input device selection, transcription quality, etc.
</script>

<div class="space-y-4">
	<div class="card text-center py-6">
		<div class="w-12 h-12 mx-auto mb-3 rounded-full flex items-center justify-center" style="background: var(--bos-status-success-bg)">
			<svg class="w-6 h-6" style="color: var(--bos-status-success)" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="M12 1a3 3 0 0 0-3 3v8a3 3 0 0 0 6 0V4a3 3 0 0 0-3-3z"/>
				<path d="M19 10v2a7 7 0 0 1-14 0v-2"/>
				<line x1="12" y1="19" x2="12" y2="23"/>
				<line x1="8" y1="23" x2="16" y2="23"/>
			</svg>
		</div>
		<h2 class="text-xs font-semibold uppercase tracking-wide st-label mb-2">Voice Notes</h2>
		<p class="text-sm st-muted mb-4 max-w-md mx-auto">
			View your voice transcription history, track speaking stats, and browse notes by date or project.
		</p>
		<a href="/voice-notes" class="inline-flex items-center gap-2 btn btn-primary">
			<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
				<path d="M12 1a3 3 0 0 0-3 3v8a3 3 0 0 0 6 0V4a3 3 0 0 0-3-3z"/>
				<path d="M19 10v2a7 7 0 0 1-14 0v-2"/>
			</svg>
			View Voice Notes
		</a>
	</div>

	<div class="card">
		<h2 class="text-xs font-semibold uppercase tracking-wide st-label mb-3">About Voice Notes</h2>
		<div class="space-y-4 text-sm st-label">
			<div class="flex items-start gap-3">
				<div class="w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0" style="background: var(--bos-status-info-bg)">
					<svg class="w-4 h-4" style="color: var(--bos-status-info)" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<circle cx="12" cy="12" r="10"/>
						<polyline points="12 6 12 12 16 14"/>
					</svg>
				</div>
				<div>
					<p class="font-medium st-title">Track Your Speaking</p>
					<p>See duration, word count, and words per minute for each recording.</p>
				</div>
			</div>
			<div class="flex items-start gap-3">
				<div class="w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0" style="background: color-mix(in srgb, var(--bos-category-ai) 12%, transparent)">
					<svg class="w-4 h-4" style="color: var(--bos-category-ai)" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<rect x="3" y="4" width="18" height="18" rx="2" ry="2"/>
						<line x1="16" y1="2" x2="16" y2="6"/>
						<line x1="8" y1="2" x2="8" y2="6"/>
						<line x1="3" y1="10" x2="21" y2="10"/>
					</svg>
				</div>
				<div>
					<p class="font-medium st-title">Browse by Date</p>
					<p>Notes organized by today, yesterday, and older dates for easy review.</p>
				</div>
			</div>
			<div class="flex items-start gap-3">
				<div class="w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0" style="background: var(--bos-status-warning-bg)">
					<svg class="w-4 h-4" style="color: var(--bos-status-warning)" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
					</svg>
				</div>
				<div>
					<p class="font-medium st-title">Filter by Project</p>
					<p>View notes associated with specific projects and contexts.</p>
				</div>
			</div>
		</div>
	</div>
</div>

<style>
	.st-title { color: var(--dt); }
	.st-muted { color: var(--dt3); }
	.st-label { color: var(--dt2); }
</style>
