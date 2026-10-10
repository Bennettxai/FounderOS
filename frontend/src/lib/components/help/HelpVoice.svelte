<script lang="ts">
	import { Mic, Lock } from 'lucide-svelte';
</script>

<!-- Voice Features -->
<section id="voice" class="help-section">
	<div class="section-header">
		<div class="section-icon gradient-red">
			<Mic size={24} />
		</div>
		<div>
			<h1>Voice Features</h1>
			<p class="section-subtitle">Hands-free interaction with Business OS</p>
		</div>
	</div>

	<div class="feature-grid two-col">
		<div class="info-card">
			<h4>How to Use Voice Input</h4>
			<ol>
				<li>Click the microphone icon in chat or Spotlight</li>
				<li>Speak your message clearly</li>
				<li>A waveform visualizes your audio</li>
				<li>Click stop when finished</li>
				<li>Review and edit the transcription</li>
				<li>Press Enter to send</li>
			</ol>
		</div>
		<div class="info-card">
			<h4>Tips for Best Results</h4>
			<ul>
				<li>Use a quiet environment when possible</li>
				<li>Speak at a natural pace</li>
				<li>Punctuation is added automatically</li>
				<li>You can always edit before sending</li>
				<li>Works in Quick Chat and full Chat module</li>
			</ul>
		</div>
	</div>

	<div class="privacy-note">
		<Lock size={16} />
		<p>
			<strong>Privacy:</strong> Audio is processed securely and not stored permanently.
			Transcription happens on secure servers with enterprise-grade encryption.
		</p>
	</div>
</section>
