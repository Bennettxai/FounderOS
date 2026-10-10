<script lang="ts">
	import {
		Rocket,
		Monitor,
		Bot,
		Layers,
		Brain,
		Network,
		FileText,
		Target,
		FolderKanban
	} from 'lucide-svelte';
</script>

<!-- Getting Started -->
<section id="getting-started" class="help-section">
	<div class="section-header">
		<div class="section-icon gradient-blue">
			<Rocket size={24} />
		</div>
		<div>
			<h1>Getting Started with Business OS</h1>
			<p class="section-subtitle">Your all-in-one operating system for managing your business</p>
		</div>
	</div>

	<div class="intro-card">
		<p>
			Business OS is an AI-powered workspace that combines project management, team collaboration,
			client management, and intelligent assistance in a beautiful desktop-like interface. Think of it
			as macOS meets Notion meets your personal AI assistant.
		</p>
	</div>

	<div id="overview" class="subsection">
		<h2>Platform Overview</h2>
		<div class="feature-grid four-col">
			<div class="feature-card">
				<div class="card-icon blue">
					<Monitor size={20} />
				</div>
				<h3>Desktop Environment</h3>
				<p>A familiar macOS-inspired interface with draggable windows, a dock for quick access,
				customizable backgrounds, and even an experimental 3D spatial mode.</p>
			</div>
			<div class="feature-card">
				<div class="card-icon purple">
					<Bot size={20} />
				</div>
				<h3>AI-First Design</h3>
				<p>Every module is enhanced with AI. Chat with specialized Focus Modes, get intelligent
				suggestions, and let AI help you analyze, write, research, and build.</p>
			</div>
			<div class="feature-card">
				<div class="card-icon green">
					<Layers size={20} />
				</div>
				<h3>Integrated Modules</h3>
				<p>Tasks, Projects, Team, Clients, Calendar, Documents - all connected and aware of each
				other. Create a task from chat, link it to a project, assign to team.</p>
			</div>
			<div class="feature-card">
				<div class="card-icon orange">
					<Brain size={20} />
				</div>
				<h3>Knowledge System</h3>
				<p>Organize your business knowledge in Nodes and Contexts. The AI understands your
				business structure and provides contextual assistance.</p>
			</div>
		</div>
	</div>

	<div id="first-steps" class="subsection">
		<h2>First Steps</h2>
		<div class="steps-list">
			<div class="step-item">
				<div class="step-number">1</div>
				<div class="step-content">
					<h4>Explore the Desktop</h4>
					<p>Click icons on the desktop or dock to open modules. Drag windows to move them,
					resize by dragging edges. Try <kbd>Cmd</kbd> + <kbd>Space</kbd> for Spotlight search.</p>
				</div>
			</div>
			<div class="step-item">
				<div class="step-number">2</div>
				<div class="step-content">
					<h4>Try the AI Chat</h4>
					<p>Open the Chat module or use the quick chat bar above the dock. Select a Focus Mode
					(Research, Analyze, Write, Build) for specialized assistance.</p>
				</div>
			</div>
			<div class="step-item">
				<div class="step-number">3</div>
				<div class="step-content">
					<h4>Create a Node</h4>
					<p>Nodes are the backbone of your business structure. Create a node for your main
					business, then add child nodes for departments, projects, or areas of focus.</p>
				</div>
			</div>
			<div class="step-item">
				<div class="step-number">4</div>
				<div class="step-content">
					<h4>Add a Project &amp; Tasks</h4>
					<p>Create a project, then add tasks with priorities and due dates. Use the Kanban
					board to visualize progress or switch to list/calendar views.</p>
				</div>
			</div>
			<div class="step-item">
				<div class="step-number">5</div>
				<div class="step-content">
					<h4>Customize Your Space</h4>
					<p>Right-click the desktop for settings. Choose from 50+ backgrounds, 15 icon styles,
					adjust sizes, and make Business OS feel like home.</p>
				</div>
			</div>
		</div>
	</div>

	<div id="core-concepts" class="subsection">
		<h2>Core Concepts</h2>
		<div class="concept-grid">
			<div class="concept-card">
				<h4><Network size={16} /> Nodes</h4>
				<p>Hierarchical units representing areas of your business. Each node has a health status,
				purpose, and can contain sub-nodes. Set an "active" node to focus AI context.</p>
			</div>
			<div class="concept-card">
				<h4><FileText size={16} /> Contexts</h4>
				<p>Rich documents with a Notion-like block editor. Store business knowledge, SOPs,
				meeting notes - anything the AI should know about when helping you.</p>
			</div>
			<div class="concept-card">
				<h4><Target size={16} /> Focus Modes</h4>
				<p>Specialized AI agents for different tasks: Research (find info), Analyze (deep dive),
				Write (create content), Build (plan &amp; strategy).</p>
			</div>
			<div class="concept-card">
				<h4><FolderKanban size={16} /> Projects &amp; Tasks</h4>
				<p>Organize work into projects with status tracking. Tasks have priorities, due dates,
				assignees, and can be viewed as Kanban boards, lists, or calendar.</p>
			</div>
		</div>
	</div>
</section>
