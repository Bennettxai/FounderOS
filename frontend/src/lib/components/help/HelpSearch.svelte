<script lang="ts">
	import { Search } from 'lucide-svelte';
</script>

<!-- Spotlight Search -->
<section id="search" class="help-section">
	<div class="section-header">
		<div class="section-icon gradient-yellow">
			<Search size={24} />
		</div>
		<div>
			<h1>Spotlight Search</h1>
			<p class="section-subtitle">Your command center for everything</p>
		</div>
	</div>

	<div class="spotlight-preview">
		<div class="spotlight-mock">
			<div class="mock-search">
				<Search size={18} />
				<span>Search or chat...</span>
			</div>
		</div>
	</div>

	<div class="feature-grid two-col">
		<div class="info-card">
			<h4>Search Mode</h4>
			<ul>
				<li>Find apps, projects, tasks, clients, documents</li>
				<li>Results grouped by category</li>
				<li>Use arrow keys to navigate</li>
				<li>Press Enter to open selected item</li>
				<li>Recent searches shown by default</li>
			</ul>
		</div>
		<div class="info-card">
			<h4>Chat Mode</h4>
			<ul>
				<li>Press Tab to switch to Chat</li>
				<li>Select a context for relevant answers</li>
				<li>Voice input available</li>
				<li>Responses stream in real-time</li>
				<li>Quick access to AI assistance</li>
			</ul>
		</div>
	</div>

	<div class="commands-section">
		<h3>Quick Commands</h3>
		<p>Type these in Spotlight for instant actions:</p>
		<div class="commands-grid">
			<div class="command"><code>/task</code> Create new task</div>
			<div class="command"><code>/note</code> Quick note</div>
			<div class="command"><code>/project</code> New project</div>
			<div class="command"><code>/meet</code> Schedule meeting</div>
			<div class="command"><code>/remind</code> Set reminder</div>
			<div class="command"><code>/client</code> New client</div>
		</div>
	</div>
</section>
