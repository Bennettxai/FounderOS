<script lang="ts">
	import {
		Monitor,
		Move,
		Maximize2,
		Layers,
		Keyboard,
		Palette,
		Grid3X3,
		Lightbulb
	} from 'lucide-svelte';
</script>

<!-- Desktop Environment -->
<section id="desktop" class="help-section">
	<div class="section-header">
		<div class="section-icon gradient-green">
			<Monitor size={24} />
		</div>
		<div>
			<h1>Desktop Environment</h1>
			<p class="section-subtitle">A familiar, powerful workspace</p>
		</div>
	</div>

	<div id="windows" class="subsection">
		<h2>Window Management</h2>
		<div class="feature-grid three-col">
			<div class="feature-card compact">
				<Move size={20} class="card-icon-inline" />
				<h4>Move Windows</h4>
				<p>Drag the title bar to move windows anywhere on your desktop.</p>
			</div>
			<div class="feature-card compact">
				<Maximize2 size={20} class="card-icon-inline" />
				<h4>Resize Windows</h4>
				<p>Drag edges or corners to resize. Double-click title bar to maximize.</p>
			</div>
			<div class="feature-card compact">
				<Layers size={20} class="card-icon-inline" />
				<h4>Window Snapping</h4>
				<p>Drag to screen edges to snap windows to half or quarter positions.</p>
			</div>
		</div>

		<div class="keyboard-hint">
			<Keyboard size={16} />
			<span><kbd>Cmd</kbd> + <kbd>W</kbd> closes the current window,
			<kbd>Cmd</kbd> + <kbd>`</kbd> cycles between open windows</span>
		</div>
	</div>

	<div id="dock" class="subsection">
		<h2>Dock &amp; Navigation</h2>
		<div class="info-card">
			<h4>Using the Dock</h4>
			<ul>
				<li><strong>Open Apps:</strong> Click any icon to open or focus that module</li>
				<li><strong>Pin to Dock:</strong> Right-click desktop icons → "Add to Dock"</li>
				<li><strong>Running Indicator:</strong> A dot appears under running applications</li>
				<li><strong>Quick Chat:</strong> The chat bar above the dock for instant AI access</li>
				<li><strong>Tooltips:</strong> Hover over icons to see module names</li>
			</ul>
		</div>
	</div>

	<div id="customization" class="subsection">
		<h2>Desktop Customization</h2>
		<p class="subsection-intro">
			Right-click anywhere on the desktop and select "Desktop Settings" to personalize your workspace.
		</p>

		<div class="feature-grid two-col">
			<div class="customization-card">
				<Palette size={20} />
				<h4>Backgrounds</h4>
				<p>Choose from 50+ options:</p>
				<ul>
					<li><strong>Solid Colors:</strong> Clean, minimal backgrounds</li>
					<li><strong>Gradients:</strong> Sunrise, Ocean, Aurora, Sunset, Forest</li>
					<li><strong>Patterns:</strong> Dots, Grid, Lines, Blueprint, Topography</li>
					<li><strong>Custom:</strong> Upload your own images</li>
				</ul>
			</div>
			<div class="customization-card">
				<Grid3X3 size={20} />
				<h4>Icon Styles</h4>
				<p>15 unique icon themes:</p>
				<ul>
					<li><strong>Modern:</strong> macOS, Glassmorphism, Minimal</li>
					<li><strong>Retro:</strong> Windows 95, Pixel Art, Classic</li>
					<li><strong>Creative:</strong> Neon, Gradient, Outlined</li>
					<li><strong>Size:</strong> Adjustable from 32px to 128px</li>
				</ul>
			</div>
		</div>

		<div class="tip-card">
			<Lightbulb size={18} />
			<div>
				<strong>Pro Tip:</strong> Enable the "Noise Texture" option for a subtle film grain
				effect that adds visual depth to gradient backgrounds.
			</div>
		</div>
	</div>

	<div id="3d-mode" class="subsection">
		<h2>3D Desktop Mode (Experimental)</h2>
		<p class="subsection-intro">
			Experience your workspace in a new dimension with the experimental 3D desktop mode.
		</p>

		<div class="info-card highlight">
			<h4>3D Mode Features</h4>
			<ul>
				<li><strong>Spatial Layout:</strong> Windows orbit around a central sphere in 3D space</li>
				<li><strong>Focus View:</strong> Click any window to bring it front and center</li>
				<li><strong>Orbit Controls:</strong> Click and drag to rotate the view</li>
				<li><strong>Navigation:</strong> Use arrow keys to move between windows when focused</li>
				<li><strong>Resize:</strong> Use +/- buttons or keyboard shortcuts to resize focused windows</li>
			</ul>
			<div class="keyboard-hint" style="margin-top: 12px;">
				<Keyboard size={16} />
				<span><kbd>Esc</kbd> to unfocus/exit, <kbd>Space</kbd> toggles view mode,
				<kbd>+</kbd>/<kbd>-</kbd> resize</span>
			</div>
		</div>
	</div>
</section>
