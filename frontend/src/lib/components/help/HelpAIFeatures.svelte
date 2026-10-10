<script lang="ts">
	import {
		Sparkles,
		FileSearch,
		BarChart3,
		PenTool,
		Hammer,
		Lightbulb,
		MessageSquare,
		Zap,
		Mic,
		Wand2
	} from 'lucide-svelte';
</script>

<!-- AI Features -->
<section id="ai-features" class="help-section">
	<div class="section-header">
		<div class="section-icon gradient-purple">
			<Sparkles size={20} strokeWidth={2} />
		</div>
		<div>
			<h1>AI Features</h1>
			<p class="section-subtitle">Intelligent assistance tailored to your business</p>
		</div>
	</div>

	<div id="focus-modes" class="subsection">
		<h2>Focus Modes</h2>
		<p class="subsection-intro">
			Focus Modes transform the AI into specialized agents optimized for different types of work.
			Each mode has unique prompts, capabilities, and configurable options.
		</p>

		<div class="mode-grid">
			<div class="mode-card research">
				<div class="mode-header">
					<FileSearch size={20} strokeWidth={2} />
					<h3>Research Mode</h3>
				</div>
				<p>Search the web and your documents for information. Perfect for market research,
				competitive analysis, or finding specific facts.</p>
				<div class="mode-options">
					<h5>Options:</h5>
					<ul>
						<li><strong>Scope:</strong> Web Only, Documents Only, or All Sources</li>
						<li><strong>Depth:</strong> Quick Scan or Thorough Investigation</li>
						<li><strong>Output:</strong> Summary, Bullet Points, or Full Report</li>
					</ul>
				</div>
				<div class="mode-example">
					<span class="example-label">Example:</span>
					"Research the top 5 competitors in the CRM space and summarize their pricing models"
				</div>
			</div>

			<div class="mode-card analyze">
				<div class="mode-header">
					<BarChart3 size={20} strokeWidth={2} />
					<h3>Analyze Mode</h3>
				</div>
				<p>Deep analysis of data, situations, or documents. Get insights, identify patterns,
				and make data-driven decisions.</p>
				<div class="mode-options">
					<h5>Options:</h5>
					<ul>
						<li><strong>Approach:</strong> Validate, Compare, or Forecast</li>
						<li><strong>Depth:</strong> Surface Level to Deep Dive</li>
						<li><strong>Output:</strong> Key Findings or Interactive Dashboard</li>
					</ul>
				</div>
				<div class="mode-example">
					<span class="example-label">Example:</span>
					"Analyze our Q4 sales data and identify which product categories are underperforming"
				</div>
			</div>

			<div class="mode-card write">
				<div class="mode-header">
					<PenTool size={20} strokeWidth={2} />
					<h3>Write Mode</h3>
				</div>
				<p>Create professional content - documents, emails, proposals, blog posts,
				presentations, and more.</p>
				<div class="mode-options">
					<h5>Options:</h5>
					<ul>
						<li><strong>Format:</strong> Document, Email, Slides, or Spreadsheet</li>
						<li><strong>Tone:</strong> Professional, Casual, or Persuasive</li>
						<li><strong>Process:</strong> Step-by-Step or First Draft</li>
					</ul>
				</div>
				<div class="mode-example">
					<span class="example-label">Example:</span>
					"Write a professional proposal for our new consulting service targeting SMBs"
				</div>
			</div>

			<div class="mode-card build">
				<div class="mode-header">
					<Hammer size={20} strokeWidth={2} />
					<h3>Build Mode</h3>
				</div>
				<p>Strategy, planning, and framework creation. Build SOPs, project plans,
				business frameworks, and operational procedures.</p>
				<div class="mode-options">
					<h5>Options:</h5>
					<ul>
						<li><strong>Type:</strong> Framework, SOP, or Plan</li>
						<li><strong>Detail:</strong> High-Level Outline or Detailed Steps</li>
						<li><strong>Format:</strong> Document, Checklist, or Diagram</li>
					</ul>
				</div>
				<div class="mode-example">
					<span class="example-label">Example:</span>
					"Build an SOP for our customer onboarding process with detailed checklists"
				</div>
			</div>
		</div>

		<div class="tip-card">
			<Lightbulb size={20} strokeWidth={2} />
			<div>
				<strong>Pro Tip:</strong> Use "Do More" mode for general conversation without a specific focus.
				The AI adapts to your needs and can switch modes mid-conversation if needed.
			</div>
		</div>
	</div>

	<div id="ai-chat" class="subsection">
		<h2>AI Chat Interface</h2>
		<div class="feature-list">
			<div class="feature-item">
				<div class="item-icon"><MessageSquare size={20} strokeWidth={2} /></div>
				<div class="item-content">
					<h4>Full Chat Module</h4>
					<p>Open the Chat module for extended conversations. Your chat history is saved,
					and you can create multiple conversation threads for different topics.</p>
				</div>
			</div>
			<div class="feature-item">
				<div class="item-icon"><Zap size={20} strokeWidth={2} /></div>
				<div class="item-content">
					<h4>Quick Chat Bar</h4>
					<p>The chat bar above your dock provides instant access. Ask quick questions
					without leaving your current work. Responses appear inline.</p>
				</div>
			</div>
			<div class="feature-item">
				<div class="item-icon"><Mic size={20} strokeWidth={2} /></div>
				<div class="item-content">
					<h4>Voice Input</h4>
					<p>Click the microphone to speak your message. Audio is transcribed in real-time
					and you can edit before sending.</p>
				</div>
			</div>
			<div class="feature-item">
				<div class="item-icon"><Wand2 size={20} strokeWidth={2} /></div>
				<div class="item-content">
					<h4>AI Artifacts</h4>
					<p>When AI generates documents, code, or structured content, it appears as an
					"artifact" you can save directly to your Contexts.</p>
				</div>
			</div>
		</div>
	</div>

	<div id="ai-contexts" class="subsection">
		<h2>AI Context Awareness</h2>
		<p class="subsection-intro">
			The AI in Business OS isn't just a generic chatbot - it understands your business structure
			and can access your stored knowledge.
		</p>
		<div class="info-card">
			<h4>How Context Works</h4>
			<ul>
				<li><strong>Active Node:</strong> Set a node as "active" and AI conversations will be
				focused on that area of your business</li>
				<li><strong>Document Context:</strong> Reference specific Contexts in chat and AI will
				read and understand them</li>
				<li><strong>Project Awareness:</strong> AI knows about your projects, tasks, and team
				when providing suggestions</li>
				<li><strong>Conversation Memory:</strong> Within a chat session, AI remembers what
				you've discussed and builds on previous responses</li>
			</ul>
		</div>
	</div>
</section>
