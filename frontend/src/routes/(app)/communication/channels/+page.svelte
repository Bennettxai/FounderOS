<script lang="ts">
	import { onMount } from 'svelte';
	import { toast } from 'svelte-sonner';
	import { MessageSquare } from 'lucide-svelte';

	import ChannelsSidebar from '$lib/components/comms/channels/ChannelsSidebar.svelte';
	import ChannelsToolbar from '$lib/components/comms/channels/ChannelsToolbar.svelte';
	import ChannelMessages from '$lib/components/comms/channels/ChannelMessages.svelte';
	import ChannelCompose from '$lib/components/comms/channels/ChannelCompose.svelte';
	import ChannelThreadDrawer from '$lib/components/comms/channels/ChannelThreadDrawer.svelte';
	import ChannelTypingIndicator from '$lib/components/comms/channels/ChannelTypingIndicator.svelte';
	import ChannelEmptyState from '$lib/components/comms/channels/ChannelEmptyState.svelte';
	import WorkspaceRouteControl from '$lib/components/comms/channels/WorkspaceRouteControl.svelte';
	import PillButton from '$lib/components/ui/PillButton.svelte';

	import {
		filterByWorkspace,
		type WorkspaceScope,
	} from '$lib/components/comms/channels/commsChannelsUtils';
	import {
		getWorkspaces as getCommsWorkspaces,
		deleteCommunicationRoute,
		getUnifiedChannels,
		getUnifiedMessages,
		getThreadMessages,
		onChannelUpdated,
		onMessageReceived,
		saveCommunicationRoute,
		sendUnifiedMessage,
		syncWorkspace,
		type ChannelProvider,
		type CommsChannel,
		type CommsMessage,
		type CommsWorkspace,
		type CommunicationRouteScope,
	} from '$lib/api/comms';
	import { getWorkspaces as getBusinessWorkspaces, type Workspace } from '$lib/api/workspaces';
	import { initiateSlackAuth } from '$lib/api/slack';

	// ─── Page state ───
	let workspaces = $state<CommsWorkspace[]>([]);
	let businessWorkspaces = $state<Workspace[]>([]);
	let channels = $state<CommsChannel[]>([]);
	let workspaceScope = $state<WorkspaceScope>('all');
	let selectedChannelId = $state<string | null>(null);
	let messagesByChannel = $state<Record<string, CommsMessage[]>>({});
	let isBootstrapping = $state(true);
	let isLoadingChannels = $state(false);
	let isLoadingMessages = $state(false);
	let isSyncing = $state(false);
	let isSavingRoute = $state(false);
	let messagesError = $state<string | null>(null);
	let composeValue = $state('');
	let composeIsSending = $state(false);
	let threadOpen = $state<{ channelId: string; threadTs: string } | null>(null);
	let threadMessages = $state<CommsMessage[]>([]);
	let threadIsLoading = $state(false);
	let threadError = $state<string | null>(null);
	let threadReplyValue = $state('');
	let threadReplyIsSending = $state(false);
	// Typing indicator UI shell — realtime (Fantem) flips this when presence
	// events arrive. Keyed per channel id so the indicator clears when you
	// switch channels.
	let typingByChannel = $state<Record<string, string[]>>({});
	let messagesScrolled = $state(false);

	// Optimistic-send IDs the realtime echo should swap-in-place rather than
	// duplicate. Keyed per channel so a parallel send in another channel
	// doesn't cause cross-talk.
	let pendingSendIdsByChannel = $state<Record<string, Set<string>>>({});

	const DRAFTS_KEY = 'comms.channels.drafts.v1';

	// ─── Hardcoded bypass data (UI dev only) ───
	let useMockData = $state(false);

	const HC_HOURS = (n: number) => new Date(Date.now() - n * 3_600_000).toISOString();
	const HC_MINUTES = (n: number) => new Date(Date.now() - n * 60_000).toISOString();

	const HARDCODED_WORKSPACES: CommsWorkspace[] = [
		{
			provider: 'slack',
			workspace_id: 'mock-slack-acme',
			workspace_name: 'Acme',
			connected_at: new Date(Date.now() - 30 * 86_400_000).toISOString(),
			status: 'healthy',
		},
	];

	const HARDCODED_CHANNELS: CommsChannel[] = [
		{ id: 'hc-c-1', provider: 'slack', workspace_id: 'mock-slack-acme', workspace_name: 'Acme', external_id: 'C001', name: 'general', topic: 'Engineering announcements', is_private: false, is_archived: false, is_dm: false, member_count: 42, unread_count: 12, last_message_at: HC_MINUTES(8) },
		{ id: 'hc-c-2', provider: 'slack', workspace_id: 'mock-slack-acme', workspace_name: 'Acme', external_id: 'C002', name: 'design', topic: '🎨 Design system + token enforcement', is_private: false, is_archived: false, is_dm: false, member_count: 8, unread_count: 3, last_message_at: HC_HOURS(2) },
		{ id: 'hc-c-3', provider: 'slack', workspace_id: 'mock-slack-acme', workspace_name: 'Acme', external_id: 'C003', name: 'engineering', topic: 'Backend, frontend, agent sub-thread', is_private: false, is_archived: false, is_dm: false, member_count: 24, unread_count: 0, last_message_at: HC_HOURS(5) },
		{ id: 'hc-c-4', provider: 'slack', workspace_id: 'mock-slack-acme', workspace_name: 'Acme', external_id: 'C004', name: 'launch', topic: 'Q3 launch checklist', is_private: true, is_archived: false, is_dm: false, member_count: 6, unread_count: 7, last_message_at: HC_HOURS(1) },
		{ id: 'hc-c-5', provider: 'slack', workspace_id: 'mock-slack-acme', workspace_name: 'Acme', external_id: 'C005', name: 'marketing', is_private: false, is_archived: false, is_dm: false, member_count: 15, unread_count: 0, last_message_at: HC_HOURS(20) },
		{ id: 'hc-d-1', provider: 'slack', workspace_id: 'mock-slack-acme', workspace_name: 'Acme', external_id: 'D001', name: 'Alice Chen', is_private: false, is_archived: false, is_dm: true, member_count: 2, unread_count: 1, last_message_at: HC_MINUTES(12), presence: 'online' },
		{ id: 'hc-d-2', provider: 'slack', workspace_id: 'mock-slack-acme', workspace_name: 'Acme', external_id: 'D002', name: 'Bob, Carol', is_private: false, is_archived: false, is_dm: true, member_count: 3, unread_count: 0, last_message_at: HC_HOURS(3), presence: 'unknown' },
		{ id: 'hc-d-3', provider: 'slack', workspace_id: 'mock-slack-acme', workspace_name: 'Acme', external_id: 'D003', name: 'Sarah Wong', is_private: false, is_archived: false, is_dm: true, member_count: 2, unread_count: 0, last_message_at: HC_HOURS(26), presence: 'away' },
		{ id: 'hc-d-4', provider: 'slack', workspace_id: 'mock-slack-acme', workspace_name: 'Acme', external_id: 'D004', name: 'Marcus Torres', is_private: false, is_archived: false, is_dm: true, member_count: 2, unread_count: 0, last_message_at: HC_HOURS(48), presence: 'offline' },
	];

	function makeMockMessage(
		id: string,
		channel_id: string,
		sender: string,
		content: string,
		hoursAgo: number,
		extras: Partial<CommsMessage> = {},
	): CommsMessage {
		return {
			id,
			channel_id,
			external_id: id,
			sender_id: sender.toLowerCase().replace(/\s+/g, '-'),
			sender_name: sender,
			sender_avatar: undefined,
			content,
			content_html: undefined,
			attachments: [],
			reactions: [],
			mentions: [],
			thread_ts: undefined,
			parent_message_id: undefined,
			reply_count: 0,
			is_thread_root: false,
			is_edited: false,
			is_deleted: false,
			sent_at: HC_HOURS(hoursAgo),
			edited_at: undefined,
			...extras,
		};
	}

	const HARDCODED_MESSAGES: Record<string, CommsMessage[]> = {
		'hc-c-1': [
			makeMockMessage('m1', 'hc-c-1', 'Sarah Chen', 'Heads up — staging is back up after the migration.', 24),
			makeMockMessage('m2', 'hc-c-1', 'Sarah Chen', 'Verified the auth path end-to-end.', 24),
			makeMockMessage('m3', 'hc-c-1', 'Marcus Torres', 'Nice. Anything we need to roll forward in prod?', 23),
			makeMockMessage('m4', 'hc-c-1', 'Sarah Chen', 'One config flip on the worker pool. Will PR this afternoon.', 23),
			makeMockMessage('m5', 'hc-c-1', 'Alice Chen', 'Hey, can someone review the Q3 board deck before Friday?', 2, {
				is_thread_root: true,
				thread_ts: 'm5',
				reply_count: 3,
				reactions: [
					{ emoji: '👍', count: 2, reacted_by_me: false },
					{ emoji: '🎉', count: 1, reacted_by_me: true },
				],
			}),
			makeMockMessage('m5r1', 'hc-c-1', 'Bob Park', 'Looking now.', 2, { thread_ts: 'm5' }),
			makeMockMessage('m5r2', 'hc-c-1', 'Bob Park', 'Done — left comments on slides 4, 11, 18.', 2, { thread_ts: 'm5' }),
			makeMockMessage('m5r3', 'hc-c-1', 'Alice Chen', 'Thanks! Will incorporate.', 1, { thread_ts: 'm5' }),
			makeMockMessage('m6', 'hc-c-1', 'Marcus Torres', 'Pushed the rate-limiter fix. Watching SORX dashboards.', 0.5, { is_edited: true }),
			makeMockMessage('m7', 'hc-c-1', 'Marcus Torres', "If anyone sees 429s in the next hour, ping me — there's a fallback path I want to verify.", 0.4),
			makeMockMessage('m8', 'hc-c-1', 'Jordan Lee', "Looks clean from where I'm sitting. Nice work.", 0.2),
		],
		'hc-c-2': [
			makeMockMessage('m20', 'hc-c-2', 'Leah', 'Email v2 spec landed. Tokens are clean, structure parallels calendar.', 6),
			makeMockMessage('m21', 'hc-c-2', 'Leon', 'Channels spec matches. Working on Wave 2 now.', 4),
			makeMockMessage('m22', 'hc-c-2', 'Jordan Lee', 'Both look great. Ship it.', 2),
		],
		'hc-c-3': [
			makeMockMessage('m30', 'hc-c-3', 'Ghost', 'Gmail tool-handler stubs are wired to the real services.', 5),
			makeMockMessage('m31', 'hc-c-3', 'Fantem', 'Confirmed — the email tab actually works now.', 5),
		],
		'hc-c-4': [
			makeMockMessage('m40', 'hc-c-4', 'Jordan Lee', 'Q3 launch checklist:', 1),
			makeMockMessage('m41', 'hc-c-4', 'Jordan Lee', '1. Marketing site copy reviewed ✅\n2. Pricing page rev ✅\n3. Support docs — in flight\n4. Status page on standby', 1),
		],
		'hc-c-5': [
			makeMockMessage('m50', 'hc-c-5', 'Marketing Bot', "Tomorrow's social schedule is queued.", 20),
		],
		'hc-d-1': [
			makeMockMessage('d10', 'hc-d-1', 'Alice Chen', 'Free for 15 min before standup?', 0.2),
		],
		'hc-d-2': [
			makeMockMessage('d20', 'hc-d-2', 'Bob Park', 'Ramen at 12:30 still on?', 3),
			makeMockMessage('d21', 'hc-d-2', 'Carol', 'In!', 3),
		],
		'hc-d-3': [
			makeMockMessage('d30', 'hc-d-3', 'Sarah Wong', 'Sent over the deck — let me know what you think.', 26),
		],
	};

	// ─── Helpers ───
	function errorText(e: unknown, fallback: string): string {
		if (e instanceof Error && e.message) return e.message;
		return fallback;
	}

	const scopedChannels = $derived(
		filterByWorkspace(channels, workspaceScope),
	);

	const selectedChannel = $derived<CommsChannel | null>(
		channels.find((c) => c.id === selectedChannelId) ?? null,
	);

	const visibleMessages = $derived<CommsMessage[]>(
		selectedChannelId ? messagesByChannel[selectedChannelId] ?? [] : [],
	);

	const typingNames = $derived<string[]>(
		selectedChannelId ? typingByChannel[selectedChannelId] ?? [] : [],
	);

	// ─── Drafts (per-channel localStorage) ───
	let draftStore = $state<Record<string, string>>({});

	function loadDrafts() {
		try {
			const raw = localStorage.getItem(DRAFTS_KEY);
			if (raw) draftStore = JSON.parse(raw);
		} catch {
			// localStorage unavailable; defaults are fine.
		}
	}

	function persistDrafts(next: Record<string, string>) {
		draftStore = next;
		try {
			localStorage.setItem(DRAFTS_KEY, JSON.stringify(next));
		} catch {
			// Best-effort persistence.
		}
	}

	function readDraftForChannel(channelId: string): string {
		return draftStore[channelId] ?? '';
	}

	function writeDraftForChannel(channelId: string, value: string) {
		const next = { ...draftStore };
		if (value.trim()) next[channelId] = value;
		else delete next[channelId];
		persistDrafts(next);
	}

	// Sync compose value when active channel changes — load that channel's draft.
	$effect(() => {
		const id = selectedChannelId;
		if (!id) {
			composeValue = '';
			return;
		}
		composeValue = readDraftForChannel(id);
	});

	// ─── Load workspaces, channels, messages ───
	async function loadWorkspaces() {
		if (useMockData) {
			workspaces = HARDCODED_WORKSPACES;
			return;
		}
		try {
			workspaces = await getCommsWorkspaces();
		} catch (e) {
			toast.error(errorText(e, 'Failed to load workspaces'));
		}
	}

	async function loadBusinessWorkspaces() {
		try {
			businessWorkspaces = (await getBusinessWorkspaces()).filter(
				(workspace) => !workspace.id.startsWith('kb:'),
			);
		} catch (e) {
			toast.error(errorText(e, 'Failed to load workspace routes'));
		}
	}

	async function loadChannels() {
		if (useMockData) {
			channels = HARDCODED_CHANNELS;
			return;
		}
		if (workspaces.length === 0) {
			channels = [];
			return;
		}
		isLoadingChannels = true;
		try {
			const res = await getUnifiedChannels({ type: 'all' });
			channels = res.channels;
		} catch (e) {
			toast.error(errorText(e, 'Failed to load channels'));
		} finally {
			isLoadingChannels = false;
		}
	}

	async function loadMessages(channelId: string) {
		if (useMockData) {
			messagesByChannel = {
				...messagesByChannel,
				[channelId]: HARDCODED_MESSAGES[channelId] ?? [],
			};
			return;
		}
		const channel = channels.find((c) => c.id === channelId);
		if (!channel) return;
		isLoadingMessages = true;
		messagesError = null;
		try {
			const res = await getUnifiedMessages({
				channel_id: channelId,
				provider: channel.provider,
				limit: 50,
			});
			messagesByChannel = {
				...messagesByChannel,
				[channelId]: res.messages,
			};
		} catch (e) {
			messagesError = errorText(e, 'Failed to load messages');
			toast.error(messagesError);
		} finally {
			isLoadingMessages = false;
		}
	}

	function bypassAuthForUI() {
		useMockData = true;
		isBootstrapping = false;
		workspaces = HARDCODED_WORKSPACES;
		channels = HARDCODED_CHANNELS;
		messagesByChannel = HARDCODED_MESSAGES;
		// Auto-select #general so the user lands on populated state.
		selectedChannelId = 'hc-c-1';
		// Demo the typing indicator UI shell against #general (Wave 3 polish —
		// realtime will populate this from presence events later).
		typingByChannel = { 'hc-c-1': ['Sarah Chen'] };
	}

	onMount(async () => {
		loadDrafts();
		await Promise.all([loadWorkspaces(), loadBusinessWorkspaces()]);
		await loadChannels();
		isBootstrapping = false;
	});

	// ─── Realtime subscriptions ───
	// New messages append to the selected channel; non-selected channels get
	// an unread bump + last_message_at update so the sidebar reorders without
	// changing the open view. Channel metadata changes patch in place.
	$effect(() => {
		if (useMockData) return;
		const offMessage = onMessageReceived(({ message }) => {
			handleStreamMessage(message);
		});
		const offChannel = onChannelUpdated(({ id, changes }) => {
			channels = channels.map((c) => (c.id === id ? { ...c, ...changes } : c));
		});
		return () => {
			offMessage();
			offChannel();
		};
	});

	function handleStreamMessage(message: CommsMessage) {
		const targetChannelId = message.channel_id;
		const isOpen = selectedChannelId === targetChannelId;
		const existing = messagesByChannel[targetChannelId];
		if (existing) {
			messagesByChannel = {
				...messagesByChannel,
				[targetChannelId]: mergeIncomingMessage(message, existing, targetChannelId),
			};
		} else if (isOpen) {
			// Channel is open but messages haven't loaded yet — let the loader pick
			// it up; stream just primes the cache for the next subscriber.
			messagesByChannel = {
				...messagesByChannel,
				[targetChannelId]: [message],
			};
		}
		channels = channels.map((c) => {
			if (c.id !== targetChannelId) return c;
			return {
				...c,
				last_message_at: message.sent_at,
				unread_count: isOpen ? c.unread_count : (c.unread_count ?? 0) + 1,
			};
		});
	}

	function mergeIncomingMessage(
		incoming: CommsMessage,
		current: CommsMessage[],
		channelId: string,
	): CommsMessage[] {
		const pending = pendingSendIdsByChannel[channelId];
		if (pending && pending.size > 0) {
			const twin = matchOptimisticTwin(incoming, current, pending);
			if (twin) {
				const next = new Set(pending);
				next.delete(twin.id);
				pendingSendIdsByChannel = { ...pendingSendIdsByChannel, [channelId]: next };
				return current.map((m) => (m.id === twin.id ? incoming : m));
			}
		}
		const existingIdx = current.findIndex((m) => m.id === incoming.id);
		if (existingIdx !== -1) {
			const next = current.slice();
			next[existingIdx] = { ...current[existingIdx], ...incoming };
			return next;
		}
		return [...current, incoming];
	}

	function matchOptimisticTwin(
		incoming: CommsMessage,
		current: CommsMessage[],
		pending: Set<string>,
	): CommsMessage | null {
		const incomingMs = new Date(incoming.sent_at).getTime();
		for (const id of pending) {
			const candidate = current.find((m) => m.id === id);
			if (!candidate) continue;
			if (candidate.content !== incoming.content) continue;
			if (candidate.sender_id && incoming.sender_id && candidate.sender_id !== incoming.sender_id) {
				continue;
			}
			// Accept any echo within 60s of the optimistic timestamp — handles
			// clock skew without making accidental matches across older sends.
			const candidateMs = new Date(candidate.sent_at).getTime();
			if (Math.abs(candidateMs - incomingMs) > 60_000) continue;
			return candidate;
		}
		return null;
	}

	// ─── Handlers ───
	function handleChangeWorkspaceScope(scope: WorkspaceScope) {
		workspaceScope = scope;
		// Clear selection if the current channel no longer belongs to scope.
		if (selectedChannel && !scopedChannels.find((c) => c.id === selectedChannel.id)) {
			selectedChannelId = null;
			closeThread();
		}
	}

	async function handleSelectChannel(channel: CommsChannel) {
		selectedChannelId = channel.id;
		closeThread();
		if (!messagesByChannel[channel.id] || !useMockData) {
			await loadMessages(channel.id);
		}
	}

	async function handleSync() {
		if (!workspaces.length) return;
		isSyncing = true;
		try {
			await Promise.all(workspaces.map((w) => syncWorkspace(w)));
			await loadChannels();
			if (selectedChannelId) await loadMessages(selectedChannelId);
		} catch (e) {
			toast.error(errorText(e, 'Failed to sync'));
		} finally {
			isSyncing = false;
		}
	}

	async function handleSaveRoute(workspaceId: string, scope: CommunicationRouteScope) {
		const channel = selectedChannel;
		if (!channel) return;
		isSavingRoute = true;
		try {
			const result = await saveCommunicationRoute({
				provider: channel.provider,
				scope,
				external_id: scope === 'conversation' ? channel.id : undefined,
				workspace_id: workspaceId,
				backfill_limit: channel.provider === 'whatsapp' && scope === 'conversation' ? 200 : 0,
			});
			await Promise.all([loadWorkspaces(), loadChannels()]);
			const indexed = result.indexed > 0 ? ` ${result.indexed} messages queued for indexing.` : '';
			toast.success(`Routed to ${result.route.workspace_name}.${indexed}`);
		} catch (e) {
			toast.error(errorText(e, 'Failed to save workspace route'));
		} finally {
			isSavingRoute = false;
		}
	}

	async function handleClearRoute(scope: CommunicationRouteScope) {
		const channel = selectedChannel;
		if (!channel) return;
		isSavingRoute = true;
		try {
			await deleteCommunicationRoute(
				channel.provider,
				scope,
				scope === 'conversation' ? channel.id : undefined,
			);
			await Promise.all([loadWorkspaces(), loadChannels()]);
			toast.success(
				scope === 'account'
					? 'Provider default cleared. Conversation-specific routes remain active.'
					: 'Conversation override removed. The provider default applies when configured.',
			);
		} catch (e) {
			toast.error(errorText(e, 'Failed to clear workspace route'));
		} finally {
			isSavingRoute = false;
		}
	}

	function handleComposeChange(value: string) {
		composeValue = value;
		if (selectedChannelId) writeDraftForChannel(selectedChannelId, value);
	}

	async function handleSend() {
		if (!selectedChannelId) return;
		const channel = channels.find((c) => c.id === selectedChannelId);
		if (!channel) return;
		if (channel.read_only) {
			toast.info('This local WhatsApp history is read-only in BusinessOS.');
			return;
		}
		const text = composeValue.trim();
		if (!text) return;

		// Optimistic: append a "pending" message immediately.
		const tempId = `temp-${crypto.randomUUID()}`;
		const optimistic: CommsMessage = {
			id: tempId,
			channel_id: channel.id,
			external_id: tempId,
			sender_id: 'me',
			sender_name: 'You',
			sender_avatar: undefined,
			content: text,
			content_html: undefined,
			attachments: [],
			reactions: [],
			mentions: [],
			thread_ts: undefined,
			parent_message_id: undefined,
			reply_count: 0,
			is_thread_root: false,
			is_edited: false,
			is_deleted: false,
			sent_at: new Date().toISOString(),
			edited_at: undefined,
		};
		messagesByChannel = {
			...messagesByChannel,
			[channel.id]: [...(messagesByChannel[channel.id] ?? []), optimistic],
		};
		const channelPending = new Set(pendingSendIdsByChannel[channel.id] ?? []);
		channelPending.add(tempId);
		pendingSendIdsByChannel = { ...pendingSendIdsByChannel, [channel.id]: channelPending };
		composeValue = '';
		writeDraftForChannel(channel.id, '');
		composeIsSending = true;

		try {
			if (!useMockData) {
				await sendUnifiedMessage({
					channel_id: channel.id,
					provider: channel.provider,
					text,
				});
				// The realtime stream's message.received echo will swap the
				// optimistic row for the canonical one via mergeIncomingMessage.
			}
		} catch (e) {
			toast.error(errorText(e, 'Failed to send message'));
			messagesByChannel = {
				...messagesByChannel,
				[channel.id]: (messagesByChannel[channel.id] ?? []).filter(
					(m) => m.id !== tempId,
				),
			};
			const rollback = new Set(pendingSendIdsByChannel[channel.id] ?? []);
			rollback.delete(tempId);
			pendingSendIdsByChannel = { ...pendingSendIdsByChannel, [channel.id]: rollback };
			composeValue = text;
			writeDraftForChannel(channel.id, text);
		} finally {
			composeIsSending = false;
		}
	}

	function handleConnectSlack() {
		if (useMockData) {
			bypassAuthForUI();
			return;
		}
		(async () => {
			try {
				const res = await initiateSlackAuth();
				if (res.auth_url) window.location.href = res.auth_url;
			} catch (e) {
				toast.error(errorText(e, 'Failed to start Slack OAuth'));
			}
		})();
	}

	function handleAddWorkspace(provider: ChannelProvider) {
		if (provider === 'slack') {
			handleConnectSlack();
		} else {
			toast.info('Microsoft Teams support is on the roadmap.');
		}
	}

	// ─── Thread drawer ───
	async function handleOpenThread(message: CommsMessage) {
		if (!selectedChannelId) return;
		const threadTs = message.thread_ts ?? message.external_id;
		threadOpen = { channelId: selectedChannelId, threadTs };
		threadIsLoading = true;
		threadError = null;
		threadMessages = [];
		try {
			if (useMockData) {
				const all = HARDCODED_MESSAGES[selectedChannelId] ?? [];
				const root = all.find(
					(m) => m.is_thread_root && m.external_id === threadTs,
				);
				const replies = all.filter((m) => m.thread_ts === threadTs && m !== root);
				threadMessages = root ? [root, ...replies] : replies;
			} else {
				const channel = channels.find((c) => c.id === selectedChannelId);
				const res = await getThreadMessages(
					selectedChannelId,
					threadTs,
					channel?.provider,
				);
				threadMessages = res.messages;
			}
		} catch (e) {
			threadError = errorText(e, 'Failed to load thread');
		} finally {
			threadIsLoading = false;
		}
	}

	function closeThread() {
		threadOpen = null;
		threadMessages = [];
		threadReplyValue = '';
		threadError = null;
	}

	async function handleSendThreadReply() {
		const open = threadOpen;
		if (!open) return;
		const text = threadReplyValue.trim();
		if (!text) return;
		const channel = channels.find((c) => c.id === open.channelId);
		if (!channel) return;
		threadReplyIsSending = true;
		try {
			if (useMockData) {
				const reply: CommsMessage = {
					id: `temp-${crypto.randomUUID()}`,
					channel_id: channel.id,
					external_id: `temp-${crypto.randomUUID()}`,
					sender_id: 'me',
					sender_name: 'You',
					sender_avatar: undefined,
					content: text,
					content_html: undefined,
					attachments: [],
					reactions: [],
					mentions: [],
					thread_ts: open.threadTs,
					parent_message_id: undefined,
					reply_count: 0,
					is_thread_root: false,
					is_edited: false,
					is_deleted: false,
					sent_at: new Date().toISOString(),
					edited_at: undefined,
				};
				threadMessages = [...threadMessages, reply];
				threadReplyValue = '';
			} else {
				await sendUnifiedMessage({
					channel_id: channel.id,
					provider: channel.provider,
					text,
					thread_ts: open.threadTs,
				});
				threadReplyValue = '';
				const res = await getThreadMessages(
					channel.id,
					open.threadTs,
					channel.provider,
				);
				threadMessages = res.messages;
				// Refresh the main stream so the parent's reply_count updates.
				await loadMessages(channel.id);
			}
		} catch (e) {
			toast.error(errorText(e, 'Failed to send reply'));
		} finally {
			threadReplyIsSending = false;
		}
	}

	// ─── Reactions / quick-reply (UI hooks; backend wiring lands when Ghost ships) ───
	function handleToggleReaction(_message: CommsMessage, _emoji: string) {
		toast.info('Reactions are coming with the next backend update.');
	}

	function handlePickEmoji(_message: CommsMessage) {
		toast.info('Emoji picker is coming with the next backend update.');
	}

	function handleReplyInThread(message: CommsMessage) {
		// Replying on a non-thread message starts a new thread on it.
		const synthetic: CommsMessage = message.is_thread_root
			? message
			: { ...message, is_thread_root: true, thread_ts: message.external_id };
		handleOpenThread(synthetic);
	}

	function handleRetryThread() {
		const open = threadOpen;
		if (!open) return;
		const root = threadMessages.find((m) => m.is_thread_root) ?? threadMessages[0];
		if (root) {
			handleOpenThread(root);
			return;
		}
		// Fall back: build a shim from the open marker so we can re-fire the call.
		handleOpenThread({
			id: open.threadTs,
			channel_id: open.channelId,
			external_id: open.threadTs,
			content: '',
			attachments: [],
			reactions: [],
			mentions: [],
			reply_count: 0,
			is_thread_root: true,
			thread_ts: open.threadTs,
			is_edited: false,
			is_deleted: false,
			sent_at: new Date().toISOString(),
		} as CommsMessage);
	}
</script>

{#if isBootstrapping}
	<div class="cm-channels-page cm-channels-page--loading">
		<ChannelEmptyState variant="loading" title="Loading channels…" />
	</div>
{:else if workspaces.length === 0}
	<div class="cm-channels-page cm-channels-page--unconnected">
		<div class="cm-channels-page__connect">
			<div class="cm-channels-page__connect-icon" aria-hidden="true">
				<MessageSquare size={32} />
			</div>
			<h2 class="cm-channels-page__connect-title">Connect your team chat</h2>
			<p class="cm-channels-page__connect-body">
				Connect Slack to view channels, DMs, and threads from BusinessOS.
				Microsoft Teams support lands with the next backend update.
			</p>
			<div class="cm-channels-page__connect-actions">
				<PillButton variant="cta" size="md" onclick={handleConnectSlack}>
					<MessageSquare size={16} />
					<span style="margin-left: var(--space-2);">Connect Slack</span>
				</PillButton>
				<PillButton
					variant="soft"
					size="md"
					onclick={() => handleAddWorkspace('teams')}
				>
					Connect Teams
				</PillButton>
			</div>
			<button
				type="button"
				class="cm-channels-page__bypass"
				onclick={bypassAuthForUI}
				aria-label="Skip auth and view UI with mock data"
			>
				Skip auth — show UI with mock data →
			</button>
		</div>
	</div>
{:else}
	<div class="cm-channels-page">
		<div class="cm-channels-page__panes">
			<ChannelsSidebar
				{workspaces}
				channels={scopedChannels}
				{workspaceScope}
				{selectedChannelId}
				{isSyncing}
				onSelectChannel={handleSelectChannel}
				onChangeWorkspaceScope={handleChangeWorkspaceScope}
				onAddWorkspace={handleAddWorkspace}
				onSync={handleSync}
			/>
			<div class="cm-channels-page__main">
				<ChannelsToolbar
					channel={selectedChannel}
					syncState={isSyncing ? 'syncing' : 'idle'}
					isScrolled={messagesScrolled}
					onSync={handleSync}
				/>
				{#if selectedChannel}
					<WorkspaceRouteControl
						channel={selectedChannel}
						workspaces={businessWorkspaces}
						isSaving={isSavingRoute}
						onSave={handleSaveRoute}
						onClear={handleClearRoute}
					/>
				{/if}
				<ChannelMessages
					channel={selectedChannel}
					messages={visibleMessages}
					isLoading={isLoadingMessages}
					error={messagesError}
					selfUserId="me"
					onOpenThread={handleOpenThread}
					onReplyInThread={handleReplyInThread}
					onToggleReaction={handleToggleReaction}
					onPickEmoji={handlePickEmoji}
					onRetry={() => selectedChannelId && loadMessages(selectedChannelId)}
					onScrolledChange={(s) => (messagesScrolled = s)}
				/>
				<ChannelTypingIndicator names={typingNames} />
				{#if selectedChannel?.read_only}
					<div class="cm-channels-page__read-only" role="status">
						<strong>Read-only local history</strong>
						<span>Sending stays disabled. Routed history can still be indexed into Optimal Engine.</span>
					</div>
				{:else if selectedChannel}
					<ChannelCompose
						channel={selectedChannel}
						value={composeValue}
						isSending={composeIsSending}
						onChange={handleComposeChange}
						onSend={handleSend}
					/>
				{/if}
			</div>
			{#if threadOpen}
				<ChannelThreadDrawer
					channel={selectedChannel}
					messages={threadMessages}
					isLoading={threadIsLoading}
					error={threadError}
					selfUserId="me"
					replyValue={threadReplyValue}
					replyIsSending={threadReplyIsSending}
					onClose={closeThread}
					onChangeReply={(v) => (threadReplyValue = v)}
					onSendReply={handleSendThreadReply}
					onToggleReaction={handleToggleReaction}
					onPickEmoji={handlePickEmoji}
					onRetry={handleRetryThread}
				/>
			{/if}
		</div>
	</div>
{/if}

<style>
	.cm-channels-page {
		display: flex;
		flex-direction: column;
		flex: 1 1 0%;
		min-height: 0;
		min-width: 0;
		background: var(--dbg);
		overflow: hidden;
	}

	.cm-channels-page__read-only {
		flex-shrink: 0;
		display: flex;
		align-items: center;
		gap: var(--space-2);
		padding: var(--space-3) var(--space-4);
		border-top: 1px solid var(--dbd);
		background: var(--dbg2);
		color: var(--dt3);
		font-size: var(--text-xs);
	}

	.cm-channels-page__read-only strong {
		color: var(--dt);
		font-weight: var(--font-semibold);
	}

	.cm-channels-page--loading,
	.cm-channels-page--unconnected {
		align-items: center;
		justify-content: center;
	}

	.cm-channels-page__connect {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: var(--space-3);
		padding: var(--space-12);
		text-align: center;
		max-width: 420px;
	}

	.cm-channels-page__connect-icon {
		width: 64px;
		height: 64px;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		border-radius: var(--radius-2xl);
		background: var(--dbg2);
		color: var(--dt2);
		margin-bottom: var(--space-2);
	}

	.cm-channels-page__connect-title {
		margin: 0;
		font-size: var(--text-lg);
		font-weight: var(--font-semibold);
		color: var(--dt);
	}

	.cm-channels-page__connect-body {
		margin: 0;
		font-size: var(--text-sm);
		color: var(--dt3);
		line-height: 1.5;
	}

	.cm-channels-page__connect-actions {
		display: flex;
		gap: var(--space-2);
		margin-top: var(--space-2);
	}

	.cm-channels-page__bypass {
		margin-top: var(--space-4);
		background: transparent;
		border: none;
		padding: var(--space-2) var(--space-3);
		font-size: var(--text-xs);
		font-family: inherit;
		color: var(--dt3);
		cursor: pointer;
		border-radius: var(--radius-sm);
		transition: color var(--bos-transition-fast), background var(--bos-transition-fast);
	}

	.cm-channels-page__bypass:hover {
		color: var(--dt2);
		background: var(--dbg2);
	}

	.cm-channels-page__panes {
		flex: 1;
		min-height: 0;
		display: flex;
		position: relative;
	}

	.cm-channels-page__main {
		flex: 1;
		display: flex;
		flex-direction: column;
		min-width: 0;
		min-height: 0;
	}

	@media (prefers-reduced-motion: reduce) {
		.cm-channels-page__bypass {
			transition: none;
		}
	}
</style>
		saveCommunicationRoute,
