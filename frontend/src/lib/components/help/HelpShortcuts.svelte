<script lang="ts">
	import { Keyboard } from 'lucide-svelte';
</script>

<!-- Keyboard Shortcuts -->
<section id="shortcuts" class="help-section">
	<div class="section-header">
		<div class="section-icon gradient-gray">
			<Keyboard size={24} />
		</div>
		<div>
			<h1>Keyboard Shortcuts</h1>
			<p class="section-subtitle">Work faster with these keyboard combinations</p>
		</div>
	</div>

	<div class="shortcuts-container">
		<div class="shortcut-category">
			<h3>Global</h3>
			<div class="shortcut-list">
				<div class="shortcut-row">
					<div class="keys"><kbd>Cmd</kbd> + <kbd>Space</kbd></div>
					<div class="desc">Open Spotlight Search</div>
				</div>
				<div class="shortcut-row">
					<div class="keys"><kbd>Cmd</kbd> + <kbd>Shift</kbd> + <kbd>Space</kbd></div>
					<div class="desc">Open Quick Chat popup</div>
				</div>
				<div class="shortcut-row">
					<div class="keys"><kbd>Cmd</kbd> + <kbd>W</kbd></div>
					<div class="desc">Close current window</div>
				</div>
				<div class="shortcut-row">
					<div class="keys"><kbd>Cmd</kbd> + <kbd>M</kbd></div>
					<div class="desc">Minimize current window</div>
				</div>
				<div class="shortcut-row">
					<div class="keys"><kbd>Cmd</kbd> + <kbd>`</kbd></div>
					<div class="desc">Cycle between windows</div>
				</div>
				<div class="shortcut-row">
					<div class="keys"><kbd>Esc</kbd></div>
					<div class="desc">Close modals / Deselect</div>
				</div>
			</div>
		</div>

		<div class="shortcut-category">
			<h3>Chat</h3>
			<div class="shortcut-list">
				<div class="shortcut-row">
					<div class="keys"><kbd>Enter</kbd></div>
					<div class="desc">Send message</div>
				</div>
				<div class="shortcut-row">
					<div class="keys"><kbd>Shift</kbd> + <kbd>Enter</kbd></div>
					<div class="desc">New line in message</div>
				</div>
				<div class="shortcut-row">
					<div class="keys"><kbd>Cmd</kbd> + <kbd>Enter</kbd></div>
					<div class="desc">Expand to full chat</div>
				</div>
			</div>
		</div>

		<div class="shortcut-category">
			<h3>Spotlight</h3>
			<div class="shortcut-list">
				<div class="shortcut-row">
					<div class="keys"><kbd>Tab</kbd></div>
					<div class="desc">Switch Search/Chat mode</div>
				</div>
				<div class="shortcut-row">
					<div class="keys"><kbd>↑</kbd> / <kbd>↓</kbd></div>
					<div class="desc">Navigate results</div>
				</div>
				<div class="shortcut-row">
					<div class="keys"><kbd>Enter</kbd></div>
					<div class="desc">Select / Send</div>
				</div>
			</div>
		</div>

		<div class="shortcut-category">
			<h3>3D Desktop</h3>
			<div class="shortcut-list">
				<div class="shortcut-row">
					<div class="keys"><kbd>Esc</kbd></div>
					<div class="desc">Unfocus / Exit 3D mode</div>
				</div>
				<div class="shortcut-row">
					<div class="keys"><kbd>Space</kbd></div>
					<div class="desc">Toggle orb/grid view</div>
				</div>
				<div class="shortcut-row">
					<div class="keys"><kbd>←</kbd> / <kbd>→</kbd></div>
					<div class="desc">Navigate windows</div>
				</div>
				<div class="shortcut-row">
					<div class="keys"><kbd>+</kbd> / <kbd>-</kbd></div>
					<div class="desc">Resize focused window</div>
				</div>
				<div class="shortcut-row">
					<div class="keys"><kbd>1</kbd> - <kbd>9</kbd></div>
					<div class="desc">Focus window by index</div>
				</div>
			</div>
		</div>
	</div>
</section>
