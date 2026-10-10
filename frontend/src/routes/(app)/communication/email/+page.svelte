<script lang="ts">
	import { onMount } from 'svelte';
	import { toast } from 'svelte-sonner';
	import { Mail } from 'lucide-svelte';

	import EmailSidebar, { type ProviderScope } from '$lib/components/comms/email/EmailSidebar.svelte';
	import EmailStatusBanner from '$lib/components/comms/email/EmailStatusBanner.svelte';
	import EmailToolbar from '$lib/components/comms/email/EmailToolbar.svelte';
	import EmailList from '$lib/components/comms/email/EmailList.svelte';
	import EmailThreadView from '$lib/components/comms/email/EmailThreadView.svelte';
	import EmailEmptyState from '$lib/components/comms/email/EmailEmptyState.svelte';
	import EmailComposeModal, {
		type ComposeAttachment,
		type ComposeDraft,
	} from '$lib/components/comms/email/EmailComposeModal.svelte';
	import PillButton from '$lib/components/ui/PillButton.svelte';

	import {
		FOLDERS,
		buildLocalContactIndex,
		groupByThread,
		searchLocalContacts,
	} from '$lib/components/comms/email/commsEmailUtils';
	import { bindShortcuts } from '$lib/components/comms/commsKeyboard';
	import {
		getConnectedAccounts,
		getUnifiedInbox,
		onEmailReceived,
		onEmailUpdated,
		saveDraftRemote,
		searchContacts,
		sendUnifiedEmail,
		syncProvider,
		type EmailAccount,
		type EmailFolder,
		type EmailProvider,
		type EmailThread,
		type UnifiedEmail,
	} from '$lib/api/comms';
	import {
		archiveEmail as gmailArchive,
		deleteEmail as gmailDelete,
		markAsRead as gmailMarkRead,
		requestGmailAccess,
	} from '$lib/api/gmail';

	let accounts = $state<EmailAccount[]>([]);
	let providerScope = $state<ProviderScope>('all');
	let currentFolder = $state<EmailFolder>('inbox');
	let emails = $state<UnifiedEmail[]>([]);
	let selectedThreadId = $state<string | null>(null);
	let isBootstrapping = $state(true);

	// ─── Hardcoded bypass data (UI dev only) ───
	// Click "Skip auth" on the connect screen to render with these.
	const HARDCODED_ACCOUNTS: EmailAccount[] = [
		{
			provider: 'gmail',
			email: 'mock-user@miosa.dev',
			account_id: 'hc-acct-1',
			connected_at: new Date(Date.now() - 30 * 86_400_000).toISOString(),
			last_sync: new Date().toISOString(),
			status: 'healthy',
			unread_count: 5,
		},
	];
	const HC_HOURS = (n: number) => new Date(Date.now() - n * 3_600_000).toISOString();
	const HC_DAYS = (n: number) => new Date(Date.now() - n * 86_400_000).toISOString();
	const HARDCODED_EMAILS: UnifiedEmail[] = [
		{ id: 'hc-1', user_id: 'hc', provider: 'gmail', external_id: 'hc-1', thread_id: 't-1', subject: 'Q3 board deck — final review needed by Friday', snippet: "Attaching the latest cut of the Q3 board deck. The financial appendix is updated through last week's actuals.", from_email: 'sarah.chen@example.com', from_name: 'Sarah Chen', to_emails: [{ email: 'me@example.com' }], body_text: "Hi team,\n\nAttaching the latest cut of the Q3 board deck. Three things I want your eyes on:\n1. Slide 4 — does the customer-acquisition narrative land?\n2. Slide 11 — the burn-multiple math.\n3. Slide 18 — risk factors.\n\nNeed final sign-off by Friday EOD.\n\nSarah", is_read: false, is_starred: true, is_important: true, is_draft: false, is_sent: false, is_archived: false, is_trash: false, labels: ['INBOX', 'IMPORTANT'], date: HC_HOURS(2) },
		{ id: 'hc-2', user_id: 'hc', provider: 'gmail', external_id: 'hc-2', thread_id: 't-2', subject: 'Re: Onboarding flow — agent loop is still hitting the rate limiter', snippet: 'Tested it again this morning with the throttle bumped to 100/min and it still pegged the limit on the second batch.', from_email: 'marcus.t@example.com', from_name: 'Marcus Torres', to_emails: [{ email: 'me@example.com' }], body_text: "Tested it again this morning. My hunch: the retry loop in the worker isn't honoring the backoff header. Going to instrument it.\n\nMarcus", is_read: false, is_starred: false, is_important: false, is_draft: false, is_sent: false, is_archived: false, is_trash: false, labels: ['INBOX'], date: HC_HOURS(5) },
		{ id: 'hc-3', user_id: 'hc', provider: 'gmail', external_id: 'hc-3', thread_id: 't-3', subject: 'Welcome to BusinessOS!', snippet: "Thanks for creating your first workspace. Here's what to do in your first 24 hours.", from_email: 'hello@example.com', from_name: 'BusinessOS Team', to_emails: [{ email: 'me@example.com' }], body_html: `<div><h2>Welcome to BusinessOS</h2><p>Your agents have a home.</p><ol><li>Connect your workspace</li><li>Add your first context document</li><li>Open Command and run your first task</li></ol></div>`, is_read: true, is_starred: false, is_important: false, is_draft: false, is_sent: false, is_archived: false, is_trash: false, labels: ['INBOX'], date: HC_DAYS(1) },
		{ id: 'hc-4', user_id: 'hc', provider: 'gmail', external_id: 'hc-4', thread_id: 't-1', subject: 'Re: Q3 board deck — final review needed by Friday', snippet: 'Slide 4 narrative is solid. On 11, the burn-multiple is correct but the chart axis is misleading — flatten it.', from_email: 'jordan.lee@example.com', from_name: 'Jordan Lee', to_emails: [{ email: 'me@example.com' }], body_text: 'Slide 4 narrative is solid.\n\nOn 11, flatten the chart axis. 18 looks complete.\n\nJordan', is_read: true, is_starred: false, is_important: false, is_draft: false, is_sent: false, is_archived: false, is_trash: false, labels: ['INBOX'], date: HC_HOURS(1) },
		{ id: 'hc-5', user_id: 'hc', provider: 'gmail', external_id: 'hc-5', thread_id: 't-5', subject: 'Design system review notes', snippet: "Three drift points — none structural, all spacing and color cleanup. Fixups tonight.", from_email: 'leah@example.com', from_name: 'Leah', to_emails: [{ email: 'me@example.com' }], body_text: "1. Email status background should use the shared error token.\n2. Channel avatars should use the shared default token.\n3. Compose modal should use the standard radius.\n\nLeah", is_read: false, is_starred: true, is_important: false, is_draft: false, is_sent: false, is_archived: false, is_trash: false, labels: ['INBOX'], date: HC_HOURS(9) },
		{ id: 'hc-6', user_id: 'hc', provider: 'gmail', external_id: 'hc-6', thread_id: 't-6', subject: 'Lunch tomorrow?', snippet: 'Free 12:30? Trying that new ramen spot on 4th.', from_email: 'alex@example.com', from_name: 'Alex Park', to_emails: [{ email: 'me@example.com' }], body_text: 'Free 12:30? Trying that new ramen spot on 4th.', is_read: false, is_starred: false, is_important: false, is_draft: false, is_sent: false, is_archived: false, is_trash: false, labels: ['INBOX'], date: HC_HOURS(14) },
		{ id: 'hc-7', user_id: 'hc', provider: 'gmail', external_id: 'hc-7', thread_id: 't-7', subject: 'Sent: thanks for the demo', snippet: 'Appreciated the walkthrough — the workspace architecture is impressive.', from_email: 'me@example.com', from_name: 'Me', to_emails: [{ email: 'demos@example.com' }], body_text: 'Appreciated the walkthrough — the workspace architecture is impressive. Following up with our team this week.', is_read: true, is_starred: false, is_important: false, is_draft: false, is_sent: true, is_archived: false, is_trash: false, labels: ['SENT'], date: HC_DAYS(1) },
		{ id: 'hc-8', user_id: 'hc', provider: 'gmail', external_id: 'hc-8', thread_id: 't-8', subject: 'Draft — pricing page revision', snippet: 'Working on the pricing page rewrite. Three tiers instead of two.', from_email: 'me@example.com', from_name: 'Me', to_emails: [{ email: '' }], body_text: '(Draft)\n\nThree tiers. TODO: finalize Pro pricing, decide Enterprise gating.', is_read: true, is_starred: false, is_important: false, is_draft: true, is_sent: false, is_archived: false, is_trash: false, labels: ['DRAFT'], date: HC_HOURS(3) },
	];
	let useMockData = $state(false);
	let isLoadingList = $state(false);
	let isSyncing = $state(false);
	let listError = $state<string | null>(null);
	let threadError = $state<string | null>(null);
	let searchQuery = $state('');
	let composeOpen = $state(false);
	let composeIsSending = $state(false);
	let composeError = $state<string | null>(null);
	let composeDraft = $state<ComposeDraft>(emptyDraft());

	function emptyDraft(): ComposeDraft {
		return { to: [], cc: [], bcc: [], subject: '', body: '', attachments: [] };
	}

	const providerFilter = $derived<EmailProvider[] | undefined>(
		providerScope === 'all' ? undefined : [providerScope],
	);

	const showProviderBadge = $derived(providerScope === 'all' && accounts.length > 1);

	const folderItems = $derived(
		FOLDERS.map((f) => ({
			id: f.id,
			label: f.label,
			count: f.id === 'inbox' ? unreadCountForScope() : undefined,
		})),
	);

	function unreadCountForScope(): number {
		return emails.filter((e) => !e.is_read && !e.is_archived && !e.is_trash).length;
	}

	const filteredEmails = $derived.by(() => {
		const q = searchQuery.trim().toLowerCase();
		if (!q) return emails;
		return emails.filter(
			(e) =>
				(e.subject ?? '').toLowerCase().includes(q) ||
				(e.from_email ?? '').toLowerCase().includes(q) ||
				(e.from_name ?? '').toLowerCase().includes(q) ||
				(e.snippet ?? '').toLowerCase().includes(q),
		);
	});

	const threads = $derived(groupByThread(filteredEmails));

	const selectedThread = $derived<EmailThread | null>(
		threads.find((t) => t.id === selectedThreadId) ?? null,
	);

	const folderLabel = $derived(
		FOLDERS.find((f) => f.id === currentFolder)?.label ?? currentFolder,
	);

	let recipientQuery = $state('');
	let remoteSuggestions = $state<{ email: string; name?: string }[]>([]);
	const localContactIndex = $derived(buildLocalContactIndex(emails));
	const localSuggestions = $derived(
		searchLocalContacts(localContactIndex, recipientQuery, 8),
	);
	const recipientSuggestions = $derived.by(() => {
		const seen = new Set<string>();
		const merged: { email: string; name?: string }[] = [];
		for (const s of [...remoteSuggestions, ...localSuggestions]) {
			const key = s.email.toLowerCase();
			if (seen.has(key)) continue;
			seen.add(key);
			merged.push(s);
			if (merged.length >= 8) break;
		}
		return merged;
	});

	let contactSearchTimer: ReturnType<typeof setTimeout> | null = null;
	let contactSearchSeq = 0;
	function scheduleContactSearch(query: string) {
		if (contactSearchTimer) clearTimeout(contactSearchTimer);
		const trimmed = query.trim();
		if (!trimmed) {
			remoteSuggestions = [];
			return;
		}
		const seq = ++contactSearchSeq;
		contactSearchTimer = setTimeout(async () => {
			try {
				const results = await searchContacts(trimmed, 8);
				if (seq !== contactSearchSeq) return;
				remoteSuggestions = results.map(({ email, name }) => ({ email, name }));
			} catch {
				// Quiet — local suggestions still cover the field.
			}
		}, 200);
	}

	const DRAFT_STORAGE_KEY = 'comms.email.draftV1';
	let draftSaveTimer: ReturnType<typeof setTimeout> | null = null;
	let autoSaveStatus = $state<'idle' | 'saving' | 'saved' | 'error'>('idle');
	let autoSaveAt = $state<string | null>(null);
	let remoteDraftId = $state<string | null>(null);

	// IDs of optimistically-inserted Sent rows whose backend confirm hasn't
	// landed yet. Used to swap-or-keep when the realtime echo arrives.
	let pendingSentIds = $state<Set<string>>(new Set());

	function showError(message: string, fallback: string): void {
		toast.error(message || fallback);
	}

	function errorText(e: unknown, fallback: string): string {
		if (e instanceof Error && e.message) return e.message;
		return fallback;
	}

	async function loadAccounts() {
		if (useMockData) {
			accounts = HARDCODED_ACCOUNTS;
			return;
		}
		try {
			accounts = await getConnectedAccounts();
		} catch (e) {
			showError(errorText(e, "Couldn't load your accounts"), "Couldn't load your accounts");
		}
	}

	async function loadEmails() {
		if (accounts.length === 0) {
			emails = [];
			return;
		}
		if (useMockData) {
			emails = HARDCODED_EMAILS.filter((e) => {
				switch (currentFolder) {
					case 'inbox': return !e.is_archived && !e.is_trash && !e.is_draft && !e.is_sent;
					case 'sent': return e.is_sent;
					case 'drafts': return e.is_draft;
					case 'starred': return e.is_starred;
					case 'archive': return e.is_archived;
					case 'trash': return e.is_trash;
				}
			});
			return;
		}
		isLoadingList = true;
		listError = null;
		try {
			const res = await getUnifiedInbox({
				providers: providerFilter,
				folder: currentFolder,
				limit: 50,
			});
			emails = res.emails;
		} catch (e) {
			const message = errorText(e, "Couldn't load your inbox");
			listError = message;
			toast.error(message);
		} finally {
			isLoadingList = false;
		}
	}

	function bypassAuthForUI() {
		useMockData = true;
		isBootstrapping = false;
		accounts = HARDCODED_ACCOUNTS;
		loadEmails();
	}

	$effect(() => {
		// Reload when provider scope or folder changes.
		providerScope;
		currentFolder;
		if (!isBootstrapping) loadEmails();
	});

	// j/k navigate the open thread; first j with no selection lands on the top row.
	function navigateThread(dir: 1 | -1) {
		if (!threads.length) return;
		const idx = selectedThreadId
			? threads.findIndex((t) => t.id === selectedThreadId)
			: -1;
		if (idx === -1) {
			void handleSelectThread(dir === 1 ? threads[0] : threads[threads.length - 1]);
			return;
		}
		const next = (idx + dir + threads.length) % threads.length;
		void handleSelectThread(threads[next]);
	}

	function focusSearch() {
		const el = document.querySelector<HTMLInputElement>('.cm-email-toolbar__search-input');
		el?.focus();
		el?.select();
	}

	// Window-level shortcuts. Suspended while compose modal is open — the modal
	// owns its own Esc + Mod+Enter inside that scope.
	$effect(() => {
		if (composeOpen || isBootstrapping || accounts.length === 0) return;
		return bindShortcuts([
			{ key: 'j', description: 'Next thread', handler: () => navigateThread(1) },
			{ key: 'k', description: 'Previous thread', handler: () => navigateThread(-1) },
			{ key: 'Enter', description: 'Open selected thread', handler: () => {
				const t = selectedThread;
				if (t) void handleSelectThread(t);
			} },
			{ key: 'Escape', description: 'Close thread', handler: () => {
				if (selectedThreadId) selectedThreadId = null;
			} },
			{ key: 'e', description: 'Archive selected thread', handler: () => {
				if (selectedThread) void handleArchive(selectedThread);
			} },
			{ key: '#', description: 'Delete selected thread', handler: () => {
				if (selectedThread) void handleDelete(selectedThread);
			} },
			{ key: 'r', description: 'Reply to selected thread', handler: () => {
				if (selectedThread) openReply(selectedThread, false);
			} },
			{ key: 'c', description: 'Compose new email', handler: openCompose },
			{ key: '/', description: 'Focus search', handler: focusSearch },
			{ key: '?', description: 'Show keyboard shortcuts', handler: () => {
				toast.info('Shortcuts: j/k navigate · Enter open · e archive · # delete · r reply · c compose · / search · Esc close');
			} },
		]);
	});

	onMount(async () => {
		await loadAccounts();
		await loadEmails();
		isBootstrapping = false;
	});

	// ─── Realtime subscriptions ───
	// Stream lifecycle is owned by the comms layout; here we only wire / unwire
	// per-event handlers. Cleanup runs on component destroy via $effect's
	// teardown return.
	$effect(() => {
		if (useMockData) return;
		const offReceived = onEmailReceived(({ email }) => {
			// Drop events the current view wouldn't render anyway — keeps the
			// list quiet during noisy account-wide syncs while still allowing
			// folder switches to fetch fresh.
			if (!emailMatchesScope(email)) return;
			emails = mergeIncomingEmail(email, emails);
		});
		const offUpdated = onEmailUpdated(({ id, changes }) => {
			emails = emails.map((m) => (m.id === id ? { ...m, ...changes } : m));
		});
		return () => {
			offReceived();
			offUpdated();
		};
	});

	function emailMatchesScope(email: UnifiedEmail): boolean {
		if (providerFilter && !providerFilter.includes(email.provider)) return false;
		switch (currentFolder) {
			case 'inbox':
				return !email.is_archived && !email.is_trash && !email.is_draft && !email.is_sent;
			case 'sent':
				return email.is_sent;
			case 'drafts':
				return email.is_draft;
			case 'starred':
				return email.is_starred;
			case 'archive':
				return email.is_archived;
			case 'trash':
				return email.is_trash;
			default:
				return true;
		}
	}

	// Reconcile an incoming email against the current list — replace the
	// matching optimistic row when the realtime echo arrives, otherwise
	// dedup by id and prepend.
	function mergeIncomingEmail(
		incoming: UnifiedEmail,
		current: UnifiedEmail[],
	): UnifiedEmail[] {
		const optimisticTwin = matchOptimisticTwin(incoming, current);
		if (optimisticTwin) {
			pendingSentIds.delete(optimisticTwin.id);
			pendingSentIds = new Set(pendingSentIds);
			return current.map((m) => (m.id === optimisticTwin.id ? incoming : m));
		}
		const existingIdx = current.findIndex((m) => m.id === incoming.id);
		if (existingIdx !== -1) {
			const next = current.slice();
			next[existingIdx] = { ...current[existingIdx], ...incoming };
			return next;
		}
		return [incoming, ...current];
	}

	function matchOptimisticTwin(
		incoming: UnifiedEmail,
		current: UnifiedEmail[],
	): UnifiedEmail | null {
		if (!incoming.is_sent || pendingSentIds.size === 0) return null;
		const incomingTo = new Set((incoming.to_emails ?? []).map((r) => r.email));
		for (const id of pendingSentIds) {
			const candidate = current.find((m) => m.id === id);
			if (!candidate) continue;
			if (candidate.subject !== incoming.subject) continue;
			const candidateTo = new Set((candidate.to_emails ?? []).map((r) => r.email));
			if (candidateTo.size !== incomingTo.size) continue;
			let match = true;
			for (const addr of candidateTo) {
				if (!incomingTo.has(addr)) {
					match = false;
					break;
				}
			}
			if (match) return candidate;
		}
		return null;
	}

	function handleSearchChange(value: string) {
		searchQuery = value;
	}

	function handleProviderChange(scope: ProviderScope) {
		providerScope = scope;
		selectedThreadId = null;
	}

	function handleFolderChange(folder: EmailFolder) {
		currentFolder = folder;
		selectedThreadId = null;
	}

	async function handleSync() {
		isSyncing = true;
		try {
			const targets: EmailProvider[] = providerFilter ?? ['gmail'];
			await Promise.all(targets.map((p) => syncProvider(p)));
			await loadEmails();
		} catch (e) {
			showError(errorText(e, "Sync didn't finish — try again"), "Sync didn't finish");
		} finally {
			isSyncing = false;
		}
	}

	async function handleSelectThread(thread: EmailThread) {
		selectedThreadId = thread.id;
		threadError = null;
		const unread = thread.messages.filter((m) => !m.is_read);
		if (!unread.length) return;
		// Optimistic: mark read locally.
		emails = emails.map((m) =>
			thread.messages.some((tm) => tm.id === m.id) ? { ...m, is_read: true } : m,
		);
		try {
			await Promise.all(
				unread
					.filter((m) => m.provider === 'gmail')
					.map((m) => gmailMarkRead(m.id)),
			);
		} catch (e) {
			toast.error(errorText(e, "Couldn't mark as read"));
		}
	}

	async function handleToggleStar(thread: EmailThread) {
		const next = !thread.starred;
		emails = emails.map((m) =>
			thread.messages.some((tm) => tm.id === m.id) ? { ...m, is_starred: next } : m,
		);
		// Backend star toggle requires Ghost's wave 2 endpoint; surface a hint until then.
		toast.info(next ? 'Starred (saved locally for now)' : 'Unstarred');
	}

	async function handleArchive(thread: EmailThread) {
		emails = emails.filter((m) => !thread.messages.some((tm) => tm.id === m.id));
		selectedThreadId = null;
		try {
			await Promise.all(
				thread.messages
					.filter((m) => m.provider === 'gmail')
					.map((m) => gmailArchive(m.id)),
			);
			toast.success('Archived');
		} catch (e) {
			showError(errorText(e, "Couldn't archive — try again"), "Couldn't archive");
			await loadEmails();
		}
	}

	async function handleDelete(thread: EmailThread) {
		emails = emails.filter((m) => !thread.messages.some((tm) => tm.id === m.id));
		selectedThreadId = null;
		try {
			await Promise.all(
				thread.messages
					.filter((m) => m.provider === 'gmail')
					.map((m) => gmailDelete(m.id)),
			);
			toast.success('Moved to Trash');
		} catch (e) {
			showError(errorText(e, "Couldn't move to Trash — try again"), "Couldn't move to Trash");
			await loadEmails();
		}
	}

	function openCompose() {
		const restored = loadLocalDraft();
		composeDraft = restored ?? emptyDraft();
		composeError = null;
		autoSaveStatus = restored ? 'saved' : 'idle';
		composeOpen = true;
	}

	function closeCompose() {
		composeOpen = false;
		if (draftSaveTimer) clearTimeout(draftSaveTimer);
	}

	function loadLocalDraft(): ComposeDraft | null {
		try {
			const raw = localStorage.getItem(DRAFT_STORAGE_KEY);
			if (!raw) return null;
			const parsed = JSON.parse(raw) as ComposeDraft & { saved_at?: string };
			if (!parsed.subject && !parsed.body && parsed.to.length === 0) return null;
			if (parsed.saved_at) autoSaveAt = parsed.saved_at;
			return {
				to: parsed.to ?? [],
				cc: parsed.cc ?? [],
				bcc: parsed.bcc ?? [],
				subject: parsed.subject ?? '',
				body: parsed.body ?? '',
				attachments: [],
				from_account_id: parsed.from_account_id,
			};
		} catch {
			return null;
		}
	}

	function clearLocalDraft() {
		try {
			localStorage.removeItem(DRAFT_STORAGE_KEY);
		} catch {
			// Best-effort.
		}
		autoSaveAt = null;
		autoSaveStatus = 'idle';
		remoteDraftId = null;
	}

	$effect(() => {
		// Debounced draft auto-save while compose is open.
		if (!composeOpen) return;
		const snapshot = composeDraft;
		if (
			!snapshot.subject.trim() &&
			!snapshot.body.trim() &&
			snapshot.to.length === 0 &&
			snapshot.cc.length === 0 &&
			snapshot.bcc.length === 0
		) {
			return;
		}
		if (draftSaveTimer) clearTimeout(draftSaveTimer);
		draftSaveTimer = setTimeout(() => {
			void persistDraft(snapshot);
		}, 2000);
	});

	async function persistDraft(snapshot: ComposeDraft) {
		autoSaveStatus = 'saving';
		const stamp = new Date().toISOString();
		try {
			localStorage.setItem(
				DRAFT_STORAGE_KEY,
				JSON.stringify({ ...snapshot, attachments: [], saved_at: stamp }),
			);
		} catch {
			// Storage full or unavailable — fall through to remote.
		}
		try {
			const provider = (snapshot.from_account_id
				? accounts.find((a) => a.account_id === snapshot.from_account_id)?.provider
				: accounts[0]?.provider) ?? 'gmail';
			const saved = await saveDraftRemote({
				id: remoteDraftId ?? undefined,
				provider,
				to: snapshot.to,
				cc: snapshot.cc,
				bcc: snapshot.bcc,
				subject: snapshot.subject,
				body: snapshot.body,
				attachments_meta: snapshot.attachments.map((a) => ({
					filename: a.filename,
					size: a.size,
					mime_type: a.mime_type,
				})),
			});
			if (saved) remoteDraftId = saved.id;
			autoSaveStatus = 'saved';
			autoSaveAt = stamp;
		} catch {
			autoSaveStatus = 'error';
		}
	}

	function quoteBody(latest: UnifiedEmail): string {
		const stamp = new Date(latest.date).toLocaleString();
		const sender = latest.from_name || latest.from_email;
		return `\n\n---\nOn ${stamp}, ${sender} wrote:\n${latest.body_text || latest.snippet || ''}`;
	}

	function openReply(thread: EmailThread, all: boolean) {
		const latest = thread.messages[0];
		const subject = thread.subject.startsWith('Re:') ? thread.subject : `Re: ${thread.subject}`;
		const to = [latest.from_email];
		const cc = all
			? Array.from(
					new Set(
						(latest.to_emails ?? []).map((r) => r.email).filter((e) => e !== latest.from_email),
					),
				)
			: [];
		composeDraft = {
			...emptyDraft(),
			subject,
			to,
			cc,
			body: quoteBody(latest),
		};
		composeError = null;
		composeOpen = true;
	}

	function openForward(thread: EmailThread) {
		const latest = thread.messages[0];
		const subject = thread.subject.startsWith('Fwd:') ? thread.subject : `Fwd: ${thread.subject}`;
		composeDraft = {
			...emptyDraft(),
			subject,
			body: `\n\n---\nForwarded message:\nFrom: ${latest.from_name || latest.from_email} <${latest.from_email}>\nDate: ${new Date(latest.date).toLocaleString()}\nSubject: ${latest.subject}\n\n${latest.body_text || latest.snippet || ''}`,
		};
		composeError = null;
		composeOpen = true;
	}

	function buildOptimisticSent(
		provider: EmailProvider,
		account: EmailAccount | undefined,
		draft: ComposeDraft,
	): UnifiedEmail {
		return {
			id: `optimistic-${crypto.randomUUID()}`,
			user_id: account?.account_id ?? 'me',
			provider,
			external_id: '',
			thread_id: undefined,
			subject: draft.subject || '(no subject)',
			snippet: draft.body.slice(0, 140),
			from_email: account?.email ?? 'me@local',
			from_name: 'Me',
			to_emails: draft.to.map((email) => ({ email })),
			cc_emails: draft.cc.length ? draft.cc.map((email) => ({ email })) : undefined,
			bcc_emails: draft.bcc.length ? draft.bcc.map((email) => ({ email })) : undefined,
			body_text: draft.body,
			attachments: draft.attachments.map((a) => ({
				id: a.id,
				filename: a.filename,
				mime_type: a.mime_type,
				size: a.size,
			})),
			is_read: true,
			is_starred: false,
			is_important: false,
			is_draft: false,
			is_sent: true,
			is_archived: false,
			is_trash: false,
			labels: ['SENT'],
			date: new Date().toISOString(),
		};
	}

	async function handleSend() {
		composeIsSending = true;
		composeError = null;
		const provider =
			(composeDraft.from_account_id
				? accounts.find((a) => a.account_id === composeDraft.from_account_id)?.provider
				: accounts[0]?.provider) ?? 'gmail';
		const account =
			accounts.find(
				(a) =>
					(composeDraft.from_account_id
						? a.account_id === composeDraft.from_account_id
						: a.provider === provider),
			);
		const files = composeDraft.attachments
			.map((a) => a.local)
			.filter((f): f is File => !!f);
		const draftSnapshot = composeDraft;
		const optimistic = buildOptimisticSent(provider, account, draftSnapshot);
		// Optimistic insert + close the modal immediately. A failure rolls both back.
		emails = [optimistic, ...emails];
		pendingSentIds = new Set([...pendingSentIds, optimistic.id]);
		composeOpen = false;
		clearLocalDraft();
		try {
			await sendUnifiedEmail({
				provider,
				to: draftSnapshot.to,
				cc: draftSnapshot.cc.length ? draftSnapshot.cc : undefined,
				bcc: draftSnapshot.bcc.length ? draftSnapshot.bcc : undefined,
				subject: draftSnapshot.subject,
				body: draftSnapshot.body,
				is_html: false,
				attachments: files.length ? files : undefined,
			});
			toast.success('Email sent');
		} catch (e) {
			emails = emails.filter((m) => m.id !== optimistic.id);
			pendingSentIds.delete(optimistic.id);
			pendingSentIds = new Set(pendingSentIds);
			composeDraft = draftSnapshot;
			composeError = errorText(e, "Couldn't send — check your connection and try again");
			composeOpen = true;
		} finally {
			composeIsSending = false;
		}
	}

	function handlePickAttachments(files: FileList) {
		const additions: ComposeAttachment[] = Array.from(files).map((file) => ({
			id: `local-${crypto.randomUUID()}`,
			filename: file.name,
			mime_type: file.type || 'application/octet-stream',
			size: file.size,
			local: file,
		}));
		composeDraft = {
			...composeDraft,
			attachments: [...composeDraft.attachments, ...additions],
		};
	}

	function handleRemoveAttachment(attachment: ComposeAttachment) {
		composeDraft = {
			...composeDraft,
			attachments: composeDraft.attachments.filter((a) => a.id !== attachment.id),
		};
	}

	function handleRecipientQueryChange(_field: 'to' | 'cc' | 'bcc', q: string) {
		recipientQuery = q;
		scheduleContactSearch(q);
	}

	async function handleConnectGmail() {
		try {
			const result = await requestGmailAccess();
			if (result.auth_url) window.location.href = result.auth_url;
		} catch (e) {
			showError(errorText(e, "Couldn't start Gmail sign-in"), "Couldn't start Gmail sign-in");
		}
	}

	function handleAddAccount(provider: EmailProvider) {
		if (provider === 'gmail') handleConnectGmail();
		else toast.info('Connect Outlook from Settings → Integrations');
	}

	function handleReconnect(provider: EmailProvider) {
		handleAddAccount(provider);
	}
</script>

{#if isBootstrapping}
	<div class="cm-email-page cm-email-page--loading">
		<EmailEmptyState variant="loading" title="Loading email…" />
	</div>
{:else if accounts.length === 0}
	<div class="cm-email-page cm-email-page--unconnected">
		<div class="cm-email-page__connect">
			<div class="cm-email-page__connect-icon" aria-hidden="true">
				<Mail size={32} />
			</div>
			<h2 class="cm-email-page__connect-title">Connect your inbox</h2>
			<p class="cm-email-page__connect-body">
				Connect Gmail or Outlook to read and send mail from BusinessOS.
			</p>
			<div class="cm-email-page__connect-actions">
				<PillButton variant="cta" size="md" onclick={handleConnectGmail}>
					<Mail size={16} />
					<span style="margin-left: var(--space-2);">Connect Gmail</span>
				</PillButton>
				<PillButton variant="soft" size="md" onclick={() => handleAddAccount('outlook')}>
					Connect Outlook
				</PillButton>
			</div>
			<button
				type="button"
				class="cm-email-page__bypass"
				onclick={bypassAuthForUI}
				aria-label="Skip auth and view UI with mock data"
			>
				Skip auth — show UI with mock data →
			</button>
		</div>
	</div>
{:else}
	<div class="cm-email-page">
		<EmailStatusBanner
			{accounts}
			{currentFolder}
			{isSyncing}
			onReconnect={handleReconnect}
			onSyncNow={handleSync}
		/>
		<div class="cm-email-page__panes">
			<EmailSidebar
				{providerScope}
				{accounts}
				folders={folderItems}
				{currentFolder}
				onChangeProviderScope={handleProviderChange}
				onChangeFolder={handleFolderChange}
				onCompose={openCompose}
				onAddAccount={handleAddAccount}
			/>
			<div class="cm-email-page__list-pane">
				<EmailToolbar
					folder={currentFolder}
					{folderLabel}
					unreadCount={unreadCountForScope()}
					{searchQuery}
					syncState={isSyncing ? 'syncing' : 'idle'}
					onSearchChange={handleSearchChange}
					onSync={handleSync}
				/>
				<EmailList
					{threads}
					{selectedThreadId}
					isLoading={isLoadingList}
					isRefreshing={isSyncing}
					{showProviderBadge}
					{searchQuery}
					folder={currentFolder}
					{folderLabel}
					errorMessage={listError}
					onSelectThread={handleSelectThread}
					onToggleStar={handleToggleStar}
					onRetry={loadEmails}
					onClearSearch={() => (searchQuery = '')}
					onSync={handleSync}
				/>
			</div>
			<div class="cm-email-page__preview-pane">
				<EmailThreadView
					thread={selectedThread}
					isLoading={false}
					errorMessage={threadError}
					{folderLabel}
					onReply={(t) => openReply(t, false)}
					onReplyAll={(t) => openReply(t, true)}
					onForward={openForward}
					onArchive={handleArchive}
					onDelete={handleDelete}
					onToggleStar={handleToggleStar}
					onCompose={openCompose}
					onRetry={() => (threadError = null)}
				/>
			</div>
		</div>
	</div>

	<EmailComposeModal
		open={composeOpen}
		draft={composeDraft}
		{accounts}
		recipientSuggestions={recipientSuggestions}
		isSending={composeIsSending}
		errorMessage={composeError}
		{autoSaveStatus}
		{autoSaveAt}
		onClose={closeCompose}
		onChange={(next) => (composeDraft = next)}
		onSend={handleSend}
		onDiscard={() => {
			clearLocalDraft();
			closeCompose();
		}}
		onPickAttachments={handlePickAttachments}
		onRemoveAttachment={handleRemoveAttachment}
		onRecipientQueryChange={handleRecipientQueryChange}
	/>
{/if}

<style>
	.cm-email-page {
		display: flex;
		flex-direction: column;
		flex: 1 1 0%;
		min-height: 0;
		min-width: 0;
		background: var(--dbg);
		overflow: hidden;
	}

	.cm-email-page--loading,
	.cm-email-page--unconnected {
		align-items: center;
		justify-content: center;
	}

	.cm-email-page__connect {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: var(--space-3);
		padding: var(--space-12);
		text-align: center;
		max-width: 420px;
	}

	.cm-email-page__connect-icon {
		width: 64px;
		height: 64px;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		border-radius: var(--radius-2xl);
		background: var(--dbg2);
		color: var(--bos-v2-icon-secondary, var(--dt2));
		margin-bottom: var(--space-2);
	}

	.cm-email-page__connect-title {
		margin: 0;
		font-size: var(--text-lg);
		font-weight: var(--font-semibold);
		color: var(--dt);
	}

	.cm-email-page__connect-body {
		margin: 0;
		font-size: var(--text-sm);
		color: var(--dt3);
		line-height: 1.5;
	}

	.cm-email-page__connect-actions {
		display: flex;
		gap: var(--space-2);
		margin-top: var(--space-2);
	}

	.cm-email-page__bypass {
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

	.cm-email-page__bypass:hover {
		color: var(--dt2);
		background: var(--dbg2);
	}

	.cm-email-page__panes {
		flex: 1;
		min-height: 0;
		display: flex;
	}

	.cm-email-page__list-pane {
		width: 340px;
		flex-shrink: 0;
		display: flex;
		flex-direction: column;
		min-height: 0;
		border-right: 1px solid var(--dbd);
	}

	.cm-email-page__preview-pane {
		flex: 1;
		display: flex;
		flex-direction: column;
		min-height: 0;
	}
</style>
