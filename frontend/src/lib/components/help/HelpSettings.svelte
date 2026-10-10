<script lang="ts">
	import { Settings, Bot, Calendar, Palette, Users } from 'lucide-svelte';
</script>

<!-- Settings & Configuration -->
<section id="settings-help" class="help-section">
	<div class="section-header">
		<div class="section-icon gradient-slate">
			<Settings size={24} />
		</div>
		<div>
			<h1>Settings &amp; Configuration</h1>
			<p class="section-subtitle">Customize Business OS to your needs</p>
		</div>
	</div>

	<div class="settings-grid">
		<div class="setting-card">
			<Bot size={20} />
			<h4>AI Configuration</h4>
			<p>Choose your AI provider (Anthropic, OpenAI, etc.), select models, and configure
			API keys for each service.</p>
		</div>
		<div class="setting-card">
			<Calendar size={20} />
			<h4>Integrations</h4>
			<p>Connect Google Calendar, set up webhooks, and configure third-party integrations.</p>
		</div>
		<div class="setting-card">
			<Palette size={20} />
			<h4>Appearance</h4>
			<p>Desktop backgrounds, icon styles, window behavior, and visual preferences.</p>
		</div>
		<div class="setting-card">
			<Users size={20} />
			<h4>Team Settings</h4>
			<p>Manage team members, roles, permissions, and workspace settings.</p>
		</div>
	</div>
</section>
