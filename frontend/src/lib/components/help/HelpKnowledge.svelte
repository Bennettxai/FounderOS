<script lang="ts">
	import {
		Brain,
		Briefcase,
		FolderKanban,
		GraduationCap,
		Settings,
		Target
	} from 'lucide-svelte';
</script>

<!-- Knowledge & Data -->
<section id="knowledge" class="help-section">
	<div class="section-header">
		<div class="section-icon gradient-teal">
			<Brain size={24} />
		</div>
		<div>
			<h1>Knowledge &amp; Data</h1>
			<p class="section-subtitle">Organize and leverage your business intelligence</p>
		</div>
	</div>

	<div id="nodes" class="subsection">
		<h2>Nodes System</h2>
		<p class="subsection-intro">
			Nodes are the cognitive backbone of Business OS - a hierarchical structure for organizing
			and managing different areas of your business.
		</p>

		<div class="node-types">
			<div class="node-type">
				<div class="node-icon business"><Briefcase size={20} /></div>
				<h4>Business Nodes</h4>
				<p>Top-level units like departments, divisions, or major business areas</p>
			</div>
			<div class="node-type">
				<div class="node-icon project"><FolderKanban size={20} /></div>
				<h4>Project Nodes</h4>
				<p>Specific initiatives, products, or campaigns within your business</p>
			</div>
			<div class="node-type">
				<div class="node-icon learning"><GraduationCap size={20} /></div>
				<h4>Learning Nodes</h4>
				<p>Knowledge areas, training programs, skill development tracks</p>
			</div>
			<div class="node-type">
				<div class="node-icon operational"><Settings size={20} /></div>
				<h4>Operational Nodes</h4>
				<p>Day-to-day processes, SOPs, and recurring operations</p>
			</div>
		</div>

		<div class="info-card">
			<h4>Node Properties</h4>
			<ul>
				<li><strong>Health Status:</strong> Healthy, Needs Attention, Critical, or Not Started</li>
				<li><strong>Purpose:</strong> What this node represents and its goals</li>
				<li><strong>Current Status:</strong> What's happening right now</li>
				<li><strong>Weekly Focus:</strong> This week's priorities and targets</li>
				<li><strong>Decision Queue:</strong> Pending decisions requiring attention</li>
				<li><strong>Delegation Ready:</strong> Tasks ready to hand off</li>
			</ul>
		</div>

		<div class="tip-card">
			<Target size={18} />
			<div>
				<strong>Active Node:</strong> Set a node as "active" to focus AI conversations on that
				area. The AI will use the node's context when providing assistance.
			</div>
		</div>
	</div>

	<div id="contexts" class="subsection">
		<h2>Contexts (Documents)</h2>
		<p class="subsection-intro">
			Rich documents with a Notion-like block editor. Store business knowledge that AI can
			reference in conversations.
		</p>

		<div class="feature-grid two-col">
			<div class="info-card">
				<h4>Block Types</h4>
				<ul>
					<li><strong>Text:</strong> Paragraphs, headings (H1, H2, H3)</li>
					<li><strong>Lists:</strong> Bullet, numbered, and to-do checkboxes</li>
					<li><strong>Media:</strong> Images, embeds, code blocks with syntax highlighting</li>
					<li><strong>Structure:</strong> Quotes, callouts, dividers, tables</li>
					<li><strong>AI:</strong> Artifact blocks for AI-generated content</li>
				</ul>
			</div>
			<div class="info-card">
				<h4>Document Features</h4>
				<ul>
					<li><strong>Slash Commands:</strong> Type <code>/</code> to insert any block type</li>
					<li><strong>Properties:</strong> Add custom fields (text, select, date, etc.)</li>
					<li><strong>Relations:</strong> Link to other contexts, projects, or clients</li>
					<li><strong>Sharing:</strong> Generate public links to share documents</li>
					<li><strong>Templates:</strong> Create and use document templates</li>
				</ul>
			</div>
		</div>
	</div>

	<div id="knowledge-graph" class="subsection">
		<h2>Knowledge Graph</h2>
		<p class="subsection-intro">
			Visualize connections between your business entities in an interactive 3D graph.
		</p>
		<div class="info-card">
			<h4>Graph Features</h4>
			<ul>
				<li>See relationships between nodes, contexts, projects, and clients</li>
				<li>Interactive 3D visualization with orbit controls</li>
				<li>Click nodes to view details and navigate</li>
				<li>Filter by entity type or connection strength</li>
				<li>Discover hidden relationships in your business data</li>
			</ul>
		</div>
	</div>
</section>
