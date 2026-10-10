<script lang="ts">
	interface UpcomingMeeting {
		id: string;
		title: string;
		start: string;
	}

	interface PullProgress {
		status: string;
		total?: number;
		completed?: number;
	}

	interface Props {
		isPulling: boolean;
		pullingModel: string;
		pullProgress: PullProgress | null;
		isMeetingMode: boolean;
		upcomingMeeting: UpcomingMeeting | null;
		onStartMeetingRecording: () => void;
		onStopMeetingRecording: () => void;
	}

	let {
		isPulling,
		pullingModel,
		pullProgress,
		isMeetingMode,
		upcomingMeeting,
		onStartMeetingRecording,
		onStopMeetingRecording,
	}: Props = $props();

	function formatTime(dateStr: string): string {
		return new Date(dateStr).toLocaleTimeString('en-US', {
			hour: 'numeric',
			minute: '2-digit'
		});
	}
</script>

<!-- Pull progress banner -->
{#if isPulling && pullProgress}
	<div class="pull-banner">
		<div class="pull-info">
			<div class="mini-spinner"></div>
			<span>Pulling {pullingModel}...</span>
		</div>
		<span class="pull-status">{pullProgress.status}</span>
		{#if pullProgress.total && pullProgress.completed}
			<div class="pull-bar">
				<div class="pull-bar-fill" style="width: {Math.round((pullProgress.completed / pullProgress.total) * 100)}%"></div>
			</div>
		{/if}
	</div>
{/if}

<!-- Upcoming meeting banner -->
{#if upcomingMeeting && !isMeetingMode}
	<div class="meeting-banner">
		<div class="meeting-info">
			<span class="meeting-time">{formatTime(upcomingMeeting.start)}</span>
			<span class="meeting-title">{upcomingMeeting.title}</span>
		</div>
		<button class="meeting-record-btn" onclick={onStartMeetingRecording}>
			<svg viewBox="0 0 24 24" fill="currentColor">
				<circle cx="12" cy="12" r="6"/>
			</svg>
			Record
		</button>
	</div>
{/if}

<!-- Meeting mode controls -->
{#if isMeetingMode}
	<div class="meeting-controls">
		<button class="stop-recording-btn" onclick={onStopMeetingRecording}>
			<svg viewBox="0 0 24 24" fill="currentColor">
				<rect x="6" y="6" width="12" height="12" rx="2"/>
			</svg>
			Stop Recording
		</button>
	</div>
{/if}

<style>
	.pull-banner {
		display: flex;
		flex-direction: column;
		gap: 6px;
		padding: 10px 16px;
		background: linear-gradient(135deg, #f3f4f6 0%, #e5e7eb 100%);
		border-bottom: 1px solid rgba(0, 0, 0, 0.08);
	}

	.pull-info {
		display: flex;
		align-items: center;
		gap: 8px;
		font-size: 13px;
		font-weight: 500;
	}

	.pull-status {
		font-size: 11px;
		color: #666;
	}

	.pull-bar {
		height: 4px;
		background: #d1d5db;
		border-radius: 2px;
		overflow: hidden;
	}

	.pull-bar-fill {
		height: 100%;
		background: linear-gradient(90deg, #111 0%, #444 100%);
		transition: width 0.3s;
	}

	.mini-spinner {
		width: 14px;
		height: 14px;
		border: 2px solid #e5e7eb;
		border-top-color: #111;
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}

	.meeting-banner {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 10px 16px;
		background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
		color: white;
	}

	.meeting-info {
		display: flex;
		align-items: center;
		gap: 10px;
	}

	.meeting-time {
		font-weight: 600;
		font-size: 13px;
	}

	.meeting-title {
		font-size: 13px;
		opacity: 0.9;
		max-width: 180px;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.meeting-record-btn {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 6px 12px;
		background: rgba(255, 255, 255, 0.2);
		border: none;
		border-radius: 6px;
		color: white;
		font-size: 12px;
		font-weight: 500;
		cursor: pointer;
		transition: background 0.15s;
	}

	.meeting-record-btn:hover {
		background: rgba(255, 255, 255, 0.3);
	}

	.meeting-record-btn svg {
		width: 12px;
		height: 12px;
	}

	.meeting-controls {
		padding: 10px 16px;
		background: #fef2f2;
		border-bottom: 1px solid #fecaca;
	}

	.stop-recording-btn {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 8px 16px;
		background: #ef4444;
		border: none;
		border-radius: 6px;
		color: white;
		font-size: 13px;
		font-weight: 500;
		cursor: pointer;
		width: 100%;
		justify-content: center;
	}

	.stop-recording-btn:hover {
		background: #dc2626;
	}

	.stop-recording-btn svg {
		width: 14px;
		height: 14px;
	}
</style>
