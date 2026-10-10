<script lang="ts">
	import { Briefcase, Phone, Mail, Users } from 'lucide-svelte';
</script>

<!-- Clients & CRM -->
<section id="clients" class="help-section">
	<div class="section-header">
		<div class="section-icon gradient-indigo">
			<Briefcase size={24} />
		</div>
		<div>
			<h1>Clients &amp; CRM</h1>
			<p class="section-subtitle">Manage relationships and grow your business</p>
		</div>
	</div>

	<div id="client-profiles" class="subsection">
		<h2>Client Profiles</h2>
		<div class="feature-grid two-col">
			<div class="info-card">
				<h4>Profile Information</h4>
				<ul>
					<li>Company name and logo</li>
					<li>Industry and company size</li>
					<li>Contact information</li>
					<li>Custom fields and tags</li>
					<li>Notes and internal comments</li>
				</ul>
			</div>
			<div class="info-card">
				<h4>Status Tracking</h4>
				<ul>
					<li><strong>Lead:</strong> Initial contact, not yet qualified</li>
					<li><strong>Prospect:</strong> Qualified, actively pursuing</li>
					<li><strong>Active:</strong> Current paying client</li>
					<li><strong>Inactive:</strong> Paused or dormant relationship</li>
					<li><strong>Churned:</strong> Former client</li>
				</ul>
			</div>
		</div>
	</div>

	<div id="deals-pipeline" class="subsection">
		<h2>Deals Pipeline</h2>
		<p class="subsection-intro">
			Track sales opportunities through your pipeline with a visual Kanban board.
		</p>
		<div class="pipeline-stages">
			<div class="stage">
				<div class="stage-dot" style="background: var(--bos-status-info);"></div>
				<h5>Qualification</h5>
				<p>Initial assessment of fit and potential</p>
			</div>
			<div class="stage-arrow">→</div>
			<div class="stage">
				<div class="stage-dot" style="background: var(--bos-status-warning);"></div>
				<h5>Proposal</h5>
				<p>Preparing and presenting your offer</p>
			</div>
			<div class="stage-arrow">→</div>
			<div class="stage">
				<div class="stage-dot" style="background: var(--bos-category-ai);"></div>
				<h5>Negotiation</h5>
				<p>Discussing terms and finalizing details</p>
			</div>
			<div class="stage-arrow">→</div>
			<div class="stage">
				<div class="stage-dot" style="background: var(--bos-status-success);"></div>
				<h5>Closed</h5>
				<p>Deal won (or lost) - final outcome</p>
			</div>
		</div>
		<div class="info-card" style="margin-top: 20px;">
			<h4>Deal Properties</h4>
			<ul>
				<li><strong>Value:</strong> Expected deal amount</li>
				<li><strong>Probability:</strong> Likelihood of closing (affects forecasting)</li>
				<li><strong>Expected Close:</strong> Target closing date</li>
				<li><strong>Owner:</strong> Team member responsible</li>
			</ul>
		</div>
	</div>

	<div id="interactions" class="subsection">
		<h2>Interaction Tracking</h2>
		<div class="feature-grid three-col">
			<div class="feature-card compact">
				<Phone size={20} class="card-icon-inline" />
				<h4>Calls</h4>
				<p>Log phone calls with notes and outcomes</p>
			</div>
			<div class="feature-card compact">
				<Mail size={20} class="card-icon-inline" />
				<h4>Emails</h4>
				<p>Track email correspondence</p>
			</div>
			<div class="feature-card compact">
				<Users size={20} class="card-icon-inline" />
				<h4>Meetings</h4>
				<p>Record meeting notes and action items</p>
			</div>
		</div>
	</div>
</section>
