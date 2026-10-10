<script lang="ts">
	import { page } from '$app/stores';
	import { tick } from 'svelte';
	import { currentWorkspace, currentWorkspaceMembers } from '$lib/stores/workspaces';
	import ContentCalendarEditor from '$lib/components/calendar/ContentCalendarEditor.svelte';
	import {
		createContentItem,
		deleteContentItem,
		getContent,
		updateContentItem,
		type ContentItem,
		type ContentItemInput
	} from '$lib/api/content';
	import {
		ContentEditorModal,
		ContentOverview,
		ContentPipeline,
		ContentPostModal,
		CHANNELS,
		DEFAULT_WORKSTREAM,
		DEFAULT_CARD_SETTINGS,
		CLIENT_CONTENT_STAGES,
		CONTENT_THEMES,
		CONTENT_THEME_COLORS,
		defaultThemeConfig,
		OWNED_STRATEGIC_CONTENT_TOPICS,
		OWNED_CONTENT_STAGES,
		OWNED_CONTENT_WORKSTREAMS,
		STAGE_META,
		TYPES,
		UNASSIGNED_PROFILE,
		buildContentBoards,
		contentProfile,
		contentWorkstream,
		emptyContentForm,
		normalizeStatus,
		profileInScope,
		scopeProfiles,
		unique,
		workspaceMemberLabel,
		type ContentBoard,
		type ContentCardSettings,
		type ContentForm,
		type ContentThemeConfig,
		type MemberWithProfile,
		type ContentScope,
		type StageId
	} from '$lib/modules/content';
	import { CalendarDays, FolderKanban, LineChart, Loader2, Palette, Plus, RotateCcw, Search, SlidersHorizontal, Trash2, X } from 'lucide-svelte';

	type View = 'home' | 'pipeline';
	type ContentUiOrder = {
		cardOrder: string[];
		workstreamOrder: Record<string, string[]>;
		topicOrder: Record<string, string[]>;
		themeCatalog?: ContentThemeConfig[];
		cardSettings?: ContentCardSettings;
	};
	type TopicReassign = {
		profile: string;
		workstream: string;
		topic: string;
		target: string;
		selected: string[];
	};
	type TopicEditor = {
		mode: 'add' | 'rename';
		profile: string;
		workstream: string;
		topic?: string;
		value: string;
	};

	let items = $state<ContentItem[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let query = $state('');
	let activeView = $state<View>('home');
	let activeScope = $state<ContentScope>('my');
	let workstreamFilter = $state('all');
	let profileFilter = $state('all');
	let assigneeFilter = $state('all');
	let searchTimer: ReturnType<typeof setTimeout>;
	let workspaceId = $state<string | null | undefined>(undefined);
	let showEditor = $state(false);
	let editing = $state<ContentItem | null>(null);
	let form = $state<ContentForm>(emptyContentForm());
	let posting = $state<ContentItem | null>(null);
	let liveLink = $state('');
	let saving = $state(false);
	let contentScrollEl = $state<HTMLElement | null>(null);
	let cardOrder = $state<string[]>([]);
	let workstreamOrder = $state<Record<string, string[]>>({});
	let topicOrder = $state<Record<string, string[]>>({});
	let themeCatalog = $state<ContentThemeConfig[]>(defaultThemeConfig());
	let cardSettings = $state<ContentCardSettings>({ ...DEFAULT_CARD_SETTINGS });
	let showCustomize = $state(false);
	let editingTopic = $state<TopicEditor | null>(null);
	let reassigningTopic = $state<TopicReassign | null>(null);

	$effect(() => {
		const id = $currentWorkspace?.id ?? null;
		if (id !== workspaceId) {
			workspaceId = id;
			query = '';
			workstreamFilter = 'all';
			profileFilter = 'all';
			assigneeFilter = 'all';
			loadUiOrder(id, activeScope);
			void load();
		}
	});

	$effect(() => {
		const requested = $page.url.searchParams.get('view');
		if (requested === 'pipeline' || requested === 'home') activeView = requested;
	});

	$effect(() => {
		const requested = $page.url.searchParams.get('scope');
		const nextScope: ContentScope = requested === 'clients' ? 'clients' : 'my';
		if (activeScope !== nextScope) {
			activeScope = nextScope;
			profileFilter = 'all';
			workstreamFilter = 'all';
			assigneeFilter = 'all';
			loadUiOrder(workspaceId ?? null, nextScope);
		}
	});

	const workspaceName = $derived($currentWorkspace?.name || 'Selected workspace');
	const configuredContentProfiles = $derived(
		(($currentWorkspace?.settings?.content_profiles as { owned?: unknown; clients?: unknown } | undefined) ?? {})
	);
	const ownedProfileOptions = $derived(
		unique(
			(Array.isArray(configuredContentProfiles.owned)
				? configuredContentProfiles.owned.filter((value): value is string => typeof value === 'string' && value.trim().length > 0)
				: [workspaceName])
		)
	);
	const configuredClientProfiles = $derived(
		Array.isArray(configuredContentProfiles.clients)
			? configuredContentProfiles.clients.filter((value): value is string => typeof value === 'string' && value.trim().length > 0)
			: []
	);
	const discoveredProfiles = $derived(unique(items.map(contentProfile).filter((profile) => profile !== UNASSIGNED_PROFILE)));
	const clientProfileOptions = $derived(
		unique([...configuredClientProfiles, ...discoveredProfiles.filter((profile) => !ownedProfileOptions.includes(profile))])
	);
	const profileOptions = $derived(scopeProfiles(activeScope, ownedProfileOptions, clientProfileOptions));
	const scopedItems = $derived(
		items.filter((item) => profileInScope(contentProfile(item), activeScope, ownedProfileOptions, clientProfileOptions))
	);
	const pipelineStages = $derived(activeScope === 'clients' ? CLIENT_CONTENT_STAGES : OWNED_CONTENT_STAGES);
	const stageLabels = $derived(Object.fromEntries(pipelineStages.map((stage) => [stage, STAGE_META[stage].label])) as Record<string, string>);
	const profiles = $derived(unique([...profileOptions, ...scopedItems.map(contentProfile).filter((profile) => profile !== UNASSIGNED_PROFILE)]));
	const workstreamSeed = $derived(activeScope === 'my' ? OWNED_CONTENT_WORKSTREAMS : [DEFAULT_WORKSTREAM]);
	const workstreams = $derived(unique([...workstreamSeed, ...scopedItems.map(contentWorkstream)]));
	const themeNames = $derived(themeCatalog.map((theme) => theme.name));
	const themeColors = $derived(Object.fromEntries(themeCatalog.map((theme) => [theme.name, theme.color])));
	const workspaceMembers = $derived(Array.isArray($currentWorkspaceMembers) ? ($currentWorkspaceMembers as MemberWithProfile[]) : []);
	const memberOptions = $derived(
		unique(
			workspaceMembers
				.filter((member) => member.status === 'active')
				.map(workspaceMemberLabel)
		)
	);
	const assignees = $derived(unique(scopedItems.flatMap((item) => [item.owner?.trim(), item.editor?.trim()])));
	const visibleItems = $derived(
		scopedItems.filter((item) => {
			const assigneesForItem = [item.owner?.trim(), item.editor?.trim()].filter(Boolean);
			return (
				(workstreamFilter === 'all' || contentWorkstream(item) === workstreamFilter) &&
				(profileFilter === 'all' || contentProfile(item) === profileFilter) &&
				(assigneeFilter === 'all' || assigneesForItem.includes(assigneeFilter))
			);
		})
	);
	const orderedVisibleItems = $derived(orderItems(visibleItems));
	const rawBoards = $derived(
		buildContentBoards(
			orderedVisibleItems,
			profileFilter === 'all' ? profileOptions : [profileFilter],
			pipelineStages,
			ownedProfileOptions
		)
	);
	const boards = $derived(orderBoards(rawBoards));
	const topicsByWorkstream = $derived(
		Object.fromEntries(
			boards.flatMap((board) =>
				board.workstreams.map((workstream) => [topicConfigKey(board.profile, workstream.name), topicsForWorkstream(board.profile, workstream.name, workstream.items)])
			)
		)
	);
	const reassignCards = $derived(
		reassigningTopic
			? items.filter(
					(item) =>
						contentProfile(item) === reassigningTopic?.profile &&
						contentWorkstream(item) === reassigningTopic?.workstream &&
						itemTheme(item) === reassigningTopic?.topic
				)
			: []
	);
	const reassignTargets = $derived(
		reassigningTopic
			? topicsForWorkstream(reassigningTopic.profile, reassigningTopic.workstream, items.filter((item) => contentProfile(item) === reassigningTopic?.profile && contentWorkstream(item) === reassigningTopic?.workstream)).filter(
					(topic) => topic !== reassigningTopic?.topic
				)
			: []
	);

	function uiOrderKey(id: string | null | undefined = workspaceId, scope: ContentScope = activeScope) {
		return `contentos-ui-order:${id || 'no-workspace'}:${scope}`;
	}

	function loadUiOrder(id: string | null | undefined = workspaceId, scope: ContentScope = activeScope) {
		if (typeof localStorage === 'undefined') return;
		try {
			const parsed = JSON.parse(localStorage.getItem(uiOrderKey(id, scope)) || '{}') as Partial<ContentUiOrder>;
			cardOrder = Array.isArray(parsed.cardOrder) ? parsed.cardOrder : [];
			workstreamOrder = parsed.workstreamOrder && typeof parsed.workstreamOrder === 'object' ? parsed.workstreamOrder : {};
			topicOrder = parsed.topicOrder && typeof parsed.topicOrder === 'object' ? parsed.topicOrder : {};
			themeCatalog = sanitizeThemeCatalog(parsed.themeCatalog);
			cardSettings = { ...DEFAULT_CARD_SETTINGS, ...(parsed.cardSettings ?? {}) };
		} catch {
			cardOrder = [];
			workstreamOrder = {};
			topicOrder = {};
			themeCatalog = defaultThemeConfig();
			cardSettings = { ...DEFAULT_CARD_SETTINGS };
		}
	}

	function saveUiOrder(
		nextCardOrder = cardOrder,
		nextWorkstreamOrder = workstreamOrder,
		nextTopicOrder = topicOrder,
		nextThemeCatalog = themeCatalog,
		nextCardSettings = cardSettings
	) {
		if (typeof localStorage === 'undefined') return;
		localStorage.setItem(
			uiOrderKey(),
			JSON.stringify({
				cardOrder: nextCardOrder,
				workstreamOrder: nextWorkstreamOrder,
				topicOrder: nextTopicOrder,
				themeCatalog: nextThemeCatalog,
				cardSettings: nextCardSettings
			})
		);
	}

	function sanitizeThemeCatalog(value: unknown): ContentThemeConfig[] {
		const fallback = defaultThemeConfig();
		if (!Array.isArray(value)) return fallback;
		const seen = new Set<string>();
		const sanitized = value
			.map((theme) => {
				if (!theme || typeof theme !== 'object') return null;
				const candidate = theme as Partial<ContentThemeConfig>;
				const name = candidate.name?.trim();
				const color = candidate.color?.trim();
				if (!name || seen.has(name.toLowerCase())) return null;
				seen.add(name.toLowerCase());
				return { name, color: color && /^#[0-9a-f]{6}$/i.test(color) ? color : CONTENT_THEME_COLORS[name] || '#0f766e' };
			})
			.filter(Boolean) as ContentThemeConfig[];
		return sanitized.length ? sanitized : fallback;
	}

	function topicConfigKey(profile: string, workstream: string) {
		return `${profile}::${workstream}`;
	}

	function uniqueInOrder(values: string[]) {
		return Array.from(new Set(values.map((value) => value.trim()).filter(Boolean)));
	}

	function itemTheme(item: ContentItem) {
		return item.theme?.trim() || 'Uncategorized';
	}

	function defaultTopicsForWorkstream(workstream: string) {
		if (workstream === 'Paid Ads') return ['iPhone Talking Ad', 'Professional Camera Ad', 'Carousel Ad', 'Offer Test', 'Retargeting'];
		if (workstream === 'Organic Content' || workstream === 'LinkedIn Posts' || workstream === 'Trial Reels') {
			return themeNames.length ? themeNames : OWNED_STRATEGIC_CONTENT_TOPICS;
		}
		return themeNames.length ? themeNames : CONTENT_THEMES;
	}

	function topicsForWorkstream(profile: string, workstream: string, workstreamItems: ContentItem[]) {
		const key = topicConfigKey(profile, workstream);
		const hasSavedTopics = Object.prototype.hasOwnProperty.call(topicOrder, key);
		const configured = hasSavedTopics ? topicOrder[key] : defaultTopicsForWorkstream(workstream);
		const cardTopics = workstreamItems.map(itemTheme);
		return uniqueInOrder([...configured, ...cardTopics, 'Uncategorized']);
	}

	function setTopics(profile: string, workstream: string, topics: string[]) {
		const key = topicConfigKey(profile, workstream);
		const nextTopics = uniqueInOrder(topics);
		const nextTopicOrder = { ...topicOrder, [key]: nextTopics };
		topicOrder = nextTopicOrder;
		saveUiOrder(cardOrder, workstreamOrder, nextTopicOrder);
	}

	function saveThemeCatalog(nextCatalog: ContentThemeConfig[]) {
		const sanitized = sanitizeThemeCatalog(nextCatalog);
		themeCatalog = sanitized;
		saveUiOrder(cardOrder, workstreamOrder, topicOrder, sanitized, cardSettings);
	}

	function updateTheme(index: number, patch: Partial<ContentThemeConfig>) {
		const next = themeCatalog.map((theme, position) => (position === index ? { ...theme, ...patch } : theme));
		saveThemeCatalog(next);
	}

	function addTheme() {
		const base = 'New theme';
		let counter = 1;
		let name = base;
		const existing = new Set(themeCatalog.map((theme) => theme.name.toLowerCase()));
		while (existing.has(name.toLowerCase())) {
			counter += 1;
			name = `${base} ${counter}`;
		}
		saveThemeCatalog([...themeCatalog, { name, color: '#0f766e' }]);
	}

	function removeTheme(name: string) {
		if (!confirm(`Remove "${name}" from your theme list? Existing cards keep the text unless you edit them.`)) return;
		saveThemeCatalog(themeCatalog.filter((theme) => theme.name !== name));
	}

	function resetThemes() {
		if (!confirm('Reset themes to the workspace defaults?')) return;
		const nextCatalog = defaultThemeConfig();
		themeCatalog = nextCatalog;
		saveUiOrder(cardOrder, workstreamOrder, topicOrder, nextCatalog, cardSettings);
	}

	function updateCardSetting<K extends keyof ContentCardSettings>(key: K, value: ContentCardSettings[K]) {
		const nextSettings = { ...cardSettings, [key]: value };
		cardSettings = nextSettings;
		saveUiOrder(cardOrder, workstreamOrder, topicOrder, themeCatalog, nextSettings);
	}

	function addTopic(profile: string, workstream: string) {
		editingTopic = { mode: 'add', profile, workstream, value: '' };
	}

	function renameTopic(profile: string, workstream: string, topic: string) {
		if (topic === 'Uncategorized') return;
		editingTopic = { mode: 'rename', profile, workstream, topic, value: topic };
	}

	async function saveTopicEditor(event: Event) {
		event.preventDefault();
		if (!editingTopic) return;
		const nextTopic = editingTopic.value.trim();
		if (!nextTopic) return;
		const { mode, profile, workstream, topic } = editingTopic;
		const workstreamItems = items.filter((item) => contentProfile(item) === profile && contentWorkstream(item) === workstream);
		const currentTopics = topicsForWorkstream(profile, workstream, workstreamItems);
		if (currentTopics.some((candidate) => candidate !== topic && candidate.toLowerCase() === nextTopic.toLowerCase())) {
			error = 'That topic already exists.';
			return;
		}
		if (mode === 'add') {
			setTopics(profile, workstream, [...currentTopics.filter((candidate) => candidate !== 'Uncategorized'), nextTopic, 'Uncategorized']);
			editingTopic = null;
			error = null;
			return;
		}
		if (!workspaceId || !topic || nextTopic === topic) {
			editingTopic = null;
			return;
		}
		const matching = workstreamItems.filter((item) => itemTheme(item) === topic);
		setTopics(profile, workstream, currentTopics.map((candidate) => (candidate === topic ? nextTopic : candidate)));
		error = null;
		try {
			for (const item of matching) {
				const updated = await updateContentItem(item.id, payload({ ...formFromItem(item), theme: nextTopic }), workspaceId);
				items = items.map((candidate) => (candidate.id === updated.id ? updated : candidate));
			}
			editingTopic = null;
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'Failed to rename topic cards';
			await load();
		}
	}

	function removeTopic(profile: string, workstream: string, topic: string) {
		if (topic === 'Uncategorized') return;
		const workstreamItems = items.filter((item) => contentProfile(item) === profile && contentWorkstream(item) === workstream);
		const matching = workstreamItems.filter((item) => itemTheme(item) === topic);
		if (matching.length > 0) {
			const targets = topicsForWorkstream(profile, workstream, workstreamItems).filter((candidate) => candidate !== topic);
			reassigningTopic = {
				profile,
				workstream,
				topic,
				target: targets[0] ?? 'Uncategorized',
				selected: matching.map((item) => item.id)
			};
			return;
		}
		setTopics(profile, workstream, topicsForWorkstream(profile, workstream, workstreamItems).filter((candidate) => candidate !== topic));
	}

	function toggleReassignCard(id: string) {
		if (!reassigningTopic) return;
		const selected = reassigningTopic.selected.includes(id)
			? reassigningTopic.selected.filter((candidate) => candidate !== id)
			: [...reassigningTopic.selected, id];
		reassigningTopic = { ...reassigningTopic, selected };
	}

	function selectAllReassignCards() {
		if (!reassigningTopic) return;
		reassigningTopic = { ...reassigningTopic, selected: reassignCards.map((item) => item.id) };
	}

	async function moveTopicCards(targetOverride?: string, idsOverride?: string[]) {
		if (!workspaceId || !reassigningTopic) return;
		const target = targetOverride ?? reassigningTopic.target;
		const selectedIds = idsOverride ?? reassigningTopic.selected;
		if (!target || selectedIds.length === 0) return;
		const selected = reassignCards.filter((item) => selectedIds.includes(item.id));
		error = null;
		try {
			for (const item of selected) {
				const updated = await updateContentItem(item.id, payload({ ...formFromItem(item), theme: target === 'Uncategorized' ? '' : target }), workspaceId);
				items = items.map((candidate) => (candidate.id === updated.id ? updated : candidate));
			}
			const remaining = reassignCards.filter((item) => !selectedIds.includes(item.id));
			if (remaining.length === 0) {
				const currentWorkstreamItems = items.filter((item) => contentProfile(item) === reassigningTopic?.profile && contentWorkstream(item) === reassigningTopic?.workstream);
				setTopics(
					reassigningTopic.profile,
					reassigningTopic.workstream,
					topicsForWorkstream(reassigningTopic.profile, reassigningTopic.workstream, currentWorkstreamItems).filter((topic) => topic !== reassigningTopic?.topic)
				);
				reassigningTopic = null;
			} else {
				reassigningTopic = { ...reassigningTopic, selected: remaining.map((item) => item.id) };
			}
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'Failed to reassign topic cards';
			await load();
		}
	}

	function orderedIds(existing: string[], ids: string[]) {
		const available = new Set(ids);
		return [...existing.filter((id) => available.has(id)), ...ids.filter((id) => !existing.includes(id))];
	}

	function moveBefore(values: string[], moving: string, before?: string) {
		const withoutMoving = values.filter((value) => value !== moving);
		const targetIndex = before ? withoutMoving.indexOf(before) : -1;
		if (targetIndex === -1) return [...withoutMoving, moving];
		return [...withoutMoving.slice(0, targetIndex), moving, ...withoutMoving.slice(targetIndex)];
	}

	function orderItems(nextItems: ContentItem[]) {
		const order = orderedIds(cardOrder, nextItems.map((item) => item.id));
		const index = new Map(order.map((id, position) => [id, position]));
		return [...nextItems].sort((a, b) => (index.get(a.id) ?? Number.MAX_SAFE_INTEGER) - (index.get(b.id) ?? Number.MAX_SAFE_INTEGER));
	}

	function orderBoards(nextBoards: ContentBoard[]) {
		const profileIndex = new Map(profileOptions.map((profile, position) => [profile, position]));
		const sortedBoards = [...nextBoards].sort((a, b) => {
			if (profileFilter === 'all' && a.items.length !== b.items.length) return b.items.length - a.items.length;
			return (profileIndex.get(a.profile) ?? Number.MAX_SAFE_INTEGER) - (profileIndex.get(b.profile) ?? Number.MAX_SAFE_INTEGER);
		});
		return sortedBoards.map((board) => {
			const order = orderedIds(workstreamOrder[board.profile] ?? [], board.workstreams.map((workstream) => workstream.name));
			const index = new Map(order.map((name, position) => [name, position]));
			return {
				...board,
				workstreams: [...board.workstreams].sort((a, b) => (index.get(a.name) ?? Number.MAX_SAFE_INTEGER) - (index.get(b.name) ?? Number.MAX_SAFE_INTEGER))
			};
		});
	}

	async function load() {
		if (workspaceId === undefined) return;
		if (!workspaceId) {
			items = [];
			loading = false;
			error = null;
			return;
		}
		loading = true;
		error = null;
		try {
			const response = await getContent(query.trim() || undefined, workspaceId);
			items = response.items;
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'Failed to load content';
		} finally {
			loading = false;
		}
	}

	function onSearch() {
		clearTimeout(searchTimer);
		searchTimer = setTimeout(() => void load(), 250);
	}

	function clearSearch() {
		query = '';
		clearTimeout(searchTimer);
		void load();
	}

	function clearFilters() {
		workstreamFilter = 'all';
		profileFilter = 'all';
		assigneeFilter = 'all';
		clearSearch();
	}

	function payload(next: ContentForm): ContentItemInput {
		return {
			title: next.title.trim(),
			content_type: next.content_type,
			status: next.status,
			hook: next.hook.trim(),
			body: next.body.trim(),
			caption: next.caption.trim(),
			cta: next.cta.trim(),
			channel: next.channel.trim(),
			link: next.link.trim(),
			category: next.category.trim() || DEFAULT_WORKSTREAM,
			theme: next.theme.trim(),
			client: next.client.trim(),
			campaign: next.campaign.trim(),
			owner: next.owner.trim(),
			editor: next.editor.trim(),
			priority: next.priority,
			due_date: next.due_date.trim(),
			publish_date: next.publish_date.trim(),
			asset_link: next.asset_link.trim(),
			review_link: next.review_link.trim(),
			revision_notes: next.revision_notes.trim(),
			notes: next.notes.trim(),
			views: next.views || 0,
			reach: next.reach || 0,
			likes: next.likes || 0,
			comments: next.comments || 0,
			saves: next.saves || 0,
			shares: next.shares || 0,
			reposts: next.reposts || 0,
			follows: next.follows || 0,
			profile_activity: next.profile_activity || 0,
			accounts_engaged: next.accounts_engaged || 0,
			avg_watch_time_seconds: next.avg_watch_time_seconds || 0,
			retention_rate: next.retention_rate || 0,
			analytics_notes: next.analytics_notes.trim()
		};
	}

	function formFromItem(item: ContentItem): ContentForm {
		return {
			title: item.title,
			content_type: item.content_type,
			status: normalizeStatus(item.status),
			hook: item.hook ?? '',
			body: item.body ?? '',
			caption: item.caption ?? '',
			cta: item.cta ?? '',
			channel: item.channel ?? '',
			link: item.link ?? '',
			category: contentWorkstream(item),
			theme: item.theme ?? '',
			client: item.client ?? '',
			campaign: item.campaign ?? '',
			owner: item.owner ?? '',
			editor: item.editor ?? '',
			priority: item.priority ?? 'normal',
			due_date: item.due_date ?? '',
			publish_date: item.publish_date ?? '',
			asset_link: item.asset_link ?? '',
			review_link: item.review_link ?? '',
			revision_notes: item.revision_notes ?? '',
			notes: item.notes ?? '',
			views: item.views || 0,
			reach: item.reach || 0,
			likes: item.likes || 0,
			comments: item.comments || 0,
			saves: item.saves || 0,
			shares: item.shares || 0,
			reposts: item.reposts || 0,
			follows: item.follows || 0,
			profile_activity: item.profile_activity || 0,
			accounts_engaged: item.accounts_engaged || 0,
			avg_watch_time_seconds: item.avg_watch_time_seconds || 0,
			retention_rate: item.retention_rate || 0,
			analytics_notes: item.analytics_notes ?? ''
		};
	}

	function captureScroll() {
		return {
			top: contentScrollEl?.scrollTop ?? 0,
			left: contentScrollEl?.scrollLeft ?? 0,
			windowX: window.scrollX,
			windowY: window.scrollY
		};
	}

	function restoreScroll(snapshot: ReturnType<typeof captureScroll>) {
		void tick().then(() => {
			requestAnimationFrame(() => {
				if (contentScrollEl) {
					contentScrollEl.scrollTop = snapshot.top;
					contentScrollEl.scrollLeft = snapshot.left;
				}
				window.scrollTo(snapshot.windowX, snapshot.windowY);
			});
		});
	}

	function openNew(stage: StageId = 'idea', profile?: string, workstream?: string, theme?: string) {
		editing = null;
		const defaultProfile = activeScope === 'clients' ? (profileOptions[0] ?? '') : '';
		form = {
			...emptyContentForm(),
			status: stage,
			client: profile === UNASSIGNED_PROFILE ? '' : profile || (profileFilter === 'all' ? defaultProfile : profileFilter),
			category: workstream || (workstreamFilter === 'all' ? DEFAULT_WORKSTREAM : workstreamFilter),
			theme: theme ?? ''
		};
		showEditor = true;
	}

	function openEdit(item: ContentItem) {
		editing = item;
		form = formFromItem(item);
		showEditor = true;
	}

	async function save(event: Event) {
		event.preventDefault();
		if (!workspaceId || !form.title.trim()) return;
		saving = true;
		error = null;
		try {
			if (editing) await updateContentItem(editing.id, payload(form), workspaceId);
			else await createContentItem(payload(form), workspaceId);
			showEditor = false;
			await load();
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'Failed to save content';
		} finally {
			saving = false;
		}
	}

	async function move(item: ContentItem, status: StageId, workstream?: string, beforeItemId?: string, theme?: string) {
		if (!workspaceId) return;
		const next = formFromItem(item);
		const nextTheme = theme === undefined ? next.theme : theme === 'Uncategorized' ? '' : theme;
		const scrollSnapshot = captureScroll();
		const previousOrder = cardOrder;
		const currentOrder = orderedIds(cardOrder, visibleItems.map((candidate) => candidate.id));
		const nextOrder = moveBefore(currentOrder, item.id, beforeItemId);
		cardOrder = nextOrder;
		saveUiOrder(nextOrder);
		error = null;
		if (normalizeStatus(item.status) === status && (!workstream || next.category === workstream) && (theme === undefined || next.theme === nextTheme)) {
			restoreScroll(scrollSnapshot);
			return;
		}
		try {
			const updated = await updateContentItem(item.id, payload({ ...next, status, category: workstream || next.category, theme: nextTheme }), workspaceId);
			items = items.map((candidate) => (candidate.id === updated.id ? updated : candidate));
			restoreScroll(scrollSnapshot);
		} catch (cause) {
			cardOrder = previousOrder;
			saveUiOrder(previousOrder);
			error = cause instanceof Error ? cause.message : 'Failed to move content';
		}
	}

	function reorderWorkstream(profile: string, workstream: string, beforeWorkstream: string) {
		const board = boards.find((candidate) => candidate.profile === profile);
		if (!board) return;
		const currentOrder = orderedIds(workstreamOrder[profile] ?? [], board.workstreams.map((candidate) => candidate.name));
		const nextOrder = moveBefore(currentOrder, workstream, beforeWorkstream);
		const nextWorkstreamOrder = { ...workstreamOrder, [profile]: nextOrder };
		workstreamOrder = nextWorkstreamOrder;
		saveUiOrder(cardOrder, nextWorkstreamOrder);
	}

	function openPost(item: ContentItem) {
		posting = item;
		liveLink = item.link ?? '';
	}

	async function markPosted() {
		if (!workspaceId || !posting || !liveLink.trim()) return;
		saving = true;
		error = null;
		try {
			await updateContentItem(posting.id, payload({ ...formFromItem(posting), status: 'posted', link: liveLink }), workspaceId);
			posting = null;
			await load();
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'Failed to mark content as posted';
		} finally {
			saving = false;
		}
	}

	async function remove(item: ContentItem) {
		if (!workspaceId || !confirm(`Delete "${item.title}"?`)) return;
		error = null;
		try {
			await deleteContentItem(item.id, workspaceId);
			items = items.filter((candidate) => candidate.id !== item.id);
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'Failed to delete content';
		}
	}
</script>

<svelte:head><title>ContentOS - BusinessOS</title></svelte:head>

<div class="contentos" bind:this={contentScrollEl}>
	<header class="topbar">
		<div class="title-wrap"><div class="module-icon"><FolderKanban size={18} /></div><div><h1>ContentOS</h1><p>Workspace content planning, production, review, and publishing.</p></div><span class="count">{scopedItems.length}</span></div>
		<div class="tools">
			<label class="search"><Search size={15} /><input bind:value={query} oninput={onSearch} placeholder="Search content..." aria-label="Search content" />{#if query}<button aria-label="Clear search" onclick={clearSearch}><X size={13} /></button>{/if}</label>
		</div>
	</header>

	<nav class="viewbar" aria-label="ContentOS views">
		<div class="segmented"><button class:active={activeView === 'home'} onclick={() => (activeView = 'home')}><LineChart size={15} />Home</button><button class:active={activeView === 'pipeline'} onclick={() => (activeView = 'pipeline')}><FolderKanban size={15} />Pipeline</button></div>
		<div class="view-actions">
			<button type="button" onclick={() => (showCustomize = true)}><SlidersHorizontal size={14} />Customize board</button>
			<a href="/calendar?mode=content"><CalendarDays size={14} />Open Content Calendar</a>
		</div>
	</nav>

	<section class="filters" aria-label="Content filters">
		<select bind:value={workstreamFilter} aria-label="Content workstream"><option value="all">All workstreams</option>{#each workstreams as value}<option value={value}>{value}</option>{/each}</select>
		<select bind:value={profileFilter} aria-label="Profile or brand"><option value="all">All profiles</option>{#each profiles as value}<option value={value}>{value}</option>{/each}</select>
		<select bind:value={assigneeFilter} aria-label="Owner or editor"><option value="all">All owners/editors</option>{#each assignees as value}<option value={value}>{value}</option>{/each}</select>
		{#if query || workstreamFilter !== 'all' || profileFilter !== 'all' || assigneeFilter !== 'all'}<button onclick={clearFilters}>Clear</button>{/if}
	</section>

	{#if error}<div class="banner">{error}</div>{/if}
	{#if loading}
		<div class="state"><span class="spinner"><Loader2 size={20} /></span>Loading ContentOS...</div>
	{:else if !workspaceId}
		<div class="state"><FolderKanban size={27} strokeWidth={1.4} /><p>Select a workspace to load and save its content.</p></div>
	{:else if activeView === 'home'}
		<ContentOverview items={visibleItems} {workspaceName} knownProfiles={profiles} scope={activeScope} onNew={openNew} onOpenPipeline={() => (activeView = 'pipeline')} onEdit={openEdit} />
	{:else}
		<ContentPipeline
			{boards}
			stages={pipelineStages}
			{topicsByWorkstream}
			{themeNames}
			{themeColors}
			{cardSettings}
			onNew={openNew}
			onEdit={openEdit}
			onPost={openPost}
			onRemove={remove}
			onMove={move}
			onReorderWorkstream={reorderWorkstream}
			onAddTopic={addTopic}
			onRenameTopic={renameTopic}
			onRemoveTopic={removeTopic}
		/>
	{/if}
</div>

{#if showEditor && editing}
	<ContentCalendarEditor
		bind:form
		contentProfiles={profiles}
		{workstreams}
		stages={pipelineStages}
		stageLabels={stageLabels}
		types={TYPES}
		channels={CHANNELS}
		saving={saving}
		deleting={false}
		onClose={() => (showEditor = false)}
		onSave={save}
		onDelete={async () => {
			if (!editing) return;
			await remove(editing);
			showEditor = false;
		}}
	/>
{:else if showEditor}
	<ContentEditorModal bind:form {editing} {profiles} {workstreams} {memberOptions} themeOptions={themeNames} stages={pipelineStages} {saving} onClose={() => (showEditor = false)} onSave={save} />
	{/if}
	{#if posting}<ContentPostModal item={posting} bind:liveLink {saving} onClose={() => (posting = null)} onConfirm={markPosted} />{/if}

	{#if showCustomize}
		<div class="modal-backdrop" role="presentation">
			<section class="customize-modal" aria-label="Customize ContentOS board">
				<header>
					<div>
						<h2>Customize board</h2>
						<p>Edit themes, colors, and card display for this workspace view.</p>
					</div>
					<button class="ghost" type="button" onclick={() => (showCustomize = false)}>Done</button>
				</header>
				<div class="customize-grid">
					<section class="customize-panel">
						<div class="panel-title"><Palette size={15} /><span>Themes</span></div>
						<div class="theme-editor-list">
							{#each themeCatalog as theme, index (theme.name)}
								<div class="theme-editor-row">
									<input class="swatch" type="color" value={theme.color} aria-label={`${theme.name} color`} oninput={(event) => updateTheme(index, { color: (event.currentTarget as HTMLInputElement).value })} />
									<input class="theme-name" value={theme.name} aria-label={`${theme.name} name`} onblur={(event) => updateTheme(index, { name: (event.currentTarget as HTMLInputElement).value })} />
									<button class="icon ghost danger" type="button" aria-label={`Remove ${theme.name}`} onclick={() => removeTheme(theme.name)}><Trash2 size={14} /></button>
								</div>
							{/each}
						</div>
						<div class="customize-actions">
							<button type="button" onclick={addTheme}><Plus size={14} />Theme</button>
							<button type="button" onclick={resetThemes}><RotateCcw size={14} />Reset</button>
						</div>
					</section>
					<section class="customize-panel">
						<div class="panel-title"><SlidersHorizontal size={15} /><span>Card display</span></div>
						<label class="toggle-row"><input type="checkbox" checked={cardSettings.compact} onchange={(event) => updateCardSetting('compact', (event.currentTarget as HTMLInputElement).checked)} /><span>Compact cards</span></label>
						<label class="toggle-row"><input type="checkbox" checked={cardSettings.showHook} onchange={(event) => updateCardSetting('showHook', (event.currentTarget as HTMLInputElement).checked)} /><span>Show hook preview</span></label>
						<label class="toggle-row"><input type="checkbox" checked={cardSettings.showMeta} onchange={(event) => updateCardSetting('showMeta', (event.currentTarget as HTMLInputElement).checked)} /><span>Show profile, campaign, dates</span></label>
						<label class="toggle-row"><input type="checkbox" checked={cardSettings.showLinks} onchange={(event) => updateCardSetting('showLinks', (event.currentTarget as HTMLInputElement).checked)} /><span>Show asset/review/live links</span></label>
						<label class="toggle-row"><input type="checkbox" checked={cardSettings.showAnalytics} onchange={(event) => updateCardSetting('showAnalytics', (event.currentTarget as HTMLInputElement).checked)} /><span>Show analytics block</span></label>
					</section>
				</div>
			</section>
		</div>
	{/if}

	{#if editingTopic}
		<div class="modal-backdrop" role="presentation">
			<form class="topic-modal topic-modal--small" aria-label={editingTopic.mode === 'add' ? 'Add topic' : 'Rename topic'} onsubmit={saveTopicEditor}>
				<header>
					<div>
						<h2>{editingTopic.mode === 'add' ? 'Add topic' : 'Rename topic'}</h2>
						<p>{editingTopic.workstream} · {editingTopic.profile}</p>
					</div>
					<button class="ghost" type="button" onclick={() => (editingTopic = null)}>Cancel</button>
				</header>
				<div class="topic-editor-body">
					<label>
						<span>Topic name</span>
						<input bind:value={editingTopic.value} autofocus placeholder="Topic name" />
					</label>
				</div>
				<div class="topic-actions topic-actions--end">
					<button class="ghost" type="button" onclick={() => (editingTopic = null)}>Cancel</button>
					<button type="submit">Save topic</button>
				</div>
			</form>
		</div>
	{/if}

	{#if reassigningTopic}
		<div class="modal-backdrop" role="presentation">
			<section class="topic-modal" aria-label={`Reassign ${reassigningTopic.topic} cards`}>
			<header>
				<div>
					<h2>Move cards before deleting topic</h2>
					<p>{reassignCards.length} {reassignCards.length === 1 ? 'card is' : 'cards are'} still in "{reassigningTopic.topic}". Choose where they should go.</p>
				</div>
				<button class="ghost" type="button" onclick={() => (reassigningTopic = null)}>Cancel</button>
			</header>
			<div class="topic-actions">
				<label>
					<span>Move selected to</span>
					<select value={reassigningTopic.target} aria-label="Reassign target topic" onchange={(event) => (reassigningTopic = reassigningTopic ? { ...reassigningTopic, target: (event.currentTarget as HTMLSelectElement).value } : null)}>
						{#each reassignTargets as target}<option value={target}>{target}</option>{/each}
					</select>
				</label>
				<button type="button" onclick={selectAllReassignCards}>Select all</button>
				<button type="button" onclick={() => void moveTopicCards()}>Move selected</button>
				<button type="button" onclick={() => void moveTopicCards('Uncategorized', reassignCards.map((item) => item.id))}>Move all to Uncategorized</button>
			</div>
			<div class="topic-card-list">
				{#each reassignCards as item (item.id)}
					<label class="topic-card-row">
						<input type="checkbox" checked={reassigningTopic.selected.includes(item.id)} onchange={() => toggleReassignCard(item.id)} />
						<span>
							<strong>{item.title}</strong>
							<small>{item.channel || item.content_type} · {normalizeStatus(item.status)}</small>
						</span>
					</label>
				{/each}
			</div>
		</section>
	</div>
{/if}

<style>
	:global(body) { overflow-x: hidden; }
	.contentos {
		height: 100%;
		min-height: 0;
		overflow-x: hidden;
		overflow-y: auto;
		background: var(--dbg);
		color: var(--dt);
	}
	.topbar { display: flex; align-items: center; justify-content: space-between; gap: 16px; min-height: 68px; padding: 12px 16px; border-bottom: 1px solid var(--dbd); }
	.title-wrap, .tools, .search, .viewbar, .view-actions, .segmented, .filters { display: flex; align-items: center; }
	.title-wrap { min-width: 0; gap: 10px; }
	.module-icon { display: grid; width: 34px; height: 34px; flex: 0 0 auto; place-items: center; border: 1px solid var(--dbd); border-radius: 7px; color: #0f766e; }
	h1 { margin: 0; font-size: 1rem; letter-spacing: 0; }
	.title-wrap p { margin: 2px 0 0; color: var(--dt3); font-size: .72rem; }
	.count { min-width: 24px; padding: 3px 6px; border-radius: 6px; background: color-mix(in srgb, var(--dt) 7%, transparent); color: var(--dt2); font-size: .68rem; font-weight: 750; text-align: center; }
	.tools { gap: 8px; }
	.search { gap: 6px; min-height: 34px; padding: 0 8px; border: 1px solid var(--dbd); border-radius: 7px; color: var(--dt3); }
	.search input { width: 210px; border: 0; outline: 0; background: transparent; color: var(--dt); font: inherit; font-size: .78rem; }
	.search button { display: grid; width: 22px; height: 22px; place-items: center; padding: 0; border: 0; background: transparent; color: var(--dt3); cursor: pointer; }
	button, .viewbar a { display: inline-flex; align-items: center; justify-content: center; gap: 6px; min-height: 32px; padding: 6px 10px; border: 1px solid var(--dbd); border-radius: 7px; background: transparent; color: var(--dt2); font: inherit; font-size: .76rem; font-weight: 680; text-decoration: none; cursor: pointer; }
	button.primary { min-height: 34px; border-color: var(--dt); background: var(--dt); color: var(--dbg); }
	.viewbar { justify-content: space-between; gap: 12px; padding: 9px 16px; border-bottom: 1px solid var(--dbd); }
	.view-actions { gap: 8px; flex-wrap: wrap; justify-content: flex-end; }
	.segmented { padding: 3px; border: 1px solid var(--dbd); border-radius: 7px; background: color-mix(in srgb, var(--dt) 2%, transparent); }
	.segmented button { border: 0; min-height: 28px; }
	.segmented button.active { background: var(--dbg); color: var(--dt); box-shadow: 0 1px 3px rgb(0 0 0 / .1); }
	.filters { gap: 8px; padding: 9px 16px; border-bottom: 1px solid var(--dbd); }
	.filters select { min-width: 150px; height: 32px; padding: 0 8px; border: 1px solid var(--dbd); border-radius: 6px; background: var(--dbg); color: var(--dt2); font: inherit; font-size: .74rem; }
	.banner { margin: 12px 16px 0; padding: 10px 12px; border: 1px solid color-mix(in srgb, #dc2626 25%, var(--dbd)); border-radius: 7px; background: color-mix(in srgb, #dc2626 6%, var(--dbg)); color: #dc2626; font-size: .78rem; }
	.state { display: grid; min-height: 360px; place-items: center; align-content: center; gap: 9px; color: var(--dt3); font-size: .8rem; text-align: center; }
	.state p { margin: 0; }
	.modal-backdrop { position: fixed; inset: 0; z-index: 80; display: grid; place-items: center; padding: 18px; background: rgb(0 0 0 / .42); }
	.customize-modal { width: min(860px, 100%); max-height: min(760px, calc(100vh - 36px)); overflow: hidden; border: 1px solid var(--dbd); border-radius: 10px; background: var(--dbg); box-shadow: 0 24px 80px rgb(0 0 0 / .25); }
	.customize-modal > header { display: flex; align-items: flex-start; justify-content: space-between; gap: 14px; padding: 16px; border-bottom: 1px solid var(--dbd); }
	.customize-modal h2 { margin: 0; color: var(--dt); font-size: 1rem; letter-spacing: 0; }
	.customize-modal p { margin: 4px 0 0; color: var(--dt3); font-size: .76rem; }
	.customize-grid { display: grid; grid-template-columns: minmax(0, 1.4fr) minmax(260px, .9fr); gap: 12px; max-height: calc(100vh - 130px); padding: 14px; overflow: auto; }
	.customize-panel { min-width: 0; border: 1px solid var(--dbd); border-radius: 8px; background: color-mix(in srgb, var(--dt) 1.5%, transparent); }
	.panel-title { display: flex; align-items: center; gap: 7px; padding: 11px 12px; border-bottom: 1px solid var(--dbd); color: var(--dt2); font-size: .78rem; font-weight: 760; }
	.theme-editor-list { display: grid; gap: 7px; padding: 10px; }
	.theme-editor-row { display: grid; grid-template-columns: 30px minmax(0, 1fr) 30px; align-items: center; gap: 8px; }
	.swatch { width: 30px; height: 30px; padding: 0; overflow: hidden; border: 1px solid var(--dbd); border-radius: 6px; background: transparent; cursor: pointer; }
	.theme-name { min-width: 0; height: 30px; border: 1px solid var(--dbd); border-radius: 6px; padding: 0 9px; background: var(--dbg); color: var(--dt); font: inherit; font-size: .76rem; }
	.theme-name:focus { outline: 0; border-color: #0f766e; box-shadow: 0 0 0 2px color-mix(in srgb, #0f766e 12%, transparent); }
	.customize-actions { display: flex; gap: 8px; padding: 10px; border-top: 1px solid var(--dbd); }
	.toggle-row { display: flex; align-items: center; gap: 9px; padding: 11px 12px; border-bottom: 1px solid var(--dbd); color: var(--dt2); font-size: .78rem; font-weight: 680; }
	.toggle-row:last-child { border-bottom: 0; }
	.toggle-row input { width: 15px; height: 15px; margin: 0; accent-color: #0f766e; }
	.icon.ghost { width: 30px; min-height: 30px; padding: 0; }
	.ghost.danger:hover { color: #dc2626; }
		.topic-modal { width: min(680px, 100%); max-height: min(720px, calc(100vh - 36px)); display: grid; grid-template-rows: auto auto minmax(0, 1fr); overflow: hidden; border: 1px solid var(--dbd); border-radius: 10px; background: var(--dbg); box-shadow: 0 24px 80px rgb(0 0 0 / .25); }
		.topic-modal--small { max-width: 460px; grid-template-rows: auto auto auto; }
		.topic-modal header { display: flex; align-items: flex-start; justify-content: space-between; gap: 14px; padding: 16px; border-bottom: 1px solid var(--dbd); }
		.topic-modal h2 { margin: 0; color: var(--dt); font-size: 1rem; letter-spacing: 0; }
		.topic-modal p { margin: 4px 0 0; color: var(--dt3); font-size: .76rem; line-height: 1.45; }
		.topic-actions { display: flex; align-items: flex-end; flex-wrap: wrap; gap: 8px; padding: 12px 16px; border-bottom: 1px solid var(--dbd); background: color-mix(in srgb, var(--dt) 2%, transparent); }
		.topic-actions--end { justify-content: flex-end; border-top: 1px solid var(--dbd); border-bottom: 0; }
		.topic-editor-body { padding: 16px; }
		.topic-editor-body label { display: grid; gap: 6px; color: var(--dt3); font-size: .7rem; font-weight: 720; }
		.topic-editor-body input { height: 36px; border: 1px solid var(--dbd); border-radius: 7px; padding: 0 10px; background: var(--dbg); color: var(--dt); font: inherit; }
		.topic-actions label { display: grid; gap: 5px; min-width: 190px; color: var(--dt3); font-size: .68rem; font-weight: 720; }
	.topic-actions select { height: 32px; padding: 0 8px; border: 1px solid var(--dbd); border-radius: 7px; background: var(--dbg); color: var(--dt); font: inherit; font-size: .76rem; }
	.topic-card-list { display: grid; gap: 7px; min-height: 0; padding: 12px 16px 16px; overflow: auto; }
	.topic-card-row { display: flex; align-items: flex-start; gap: 9px; padding: 10px; border: 1px solid var(--dbd); border-radius: 7px; background: color-mix(in srgb, var(--dt) 1.5%, transparent); }
	.topic-card-row input { margin-top: 2px; }
	.topic-card-row span { display: grid; gap: 3px; min-width: 0; }
	.topic-card-row strong { overflow: hidden; color: var(--dt); font-size: .78rem; text-overflow: ellipsis; white-space: nowrap; }
	.topic-card-row small { color: var(--dt3); font-size: .66rem; }
	.spinner { display: inline-flex; animation: spin .8s linear infinite; } @keyframes spin { to { transform: rotate(360deg); } }
	@media (max-width: 720px) { .topbar { align-items: flex-start; flex-direction: column; } .tools, .search, .search input { width: 100%; } .tools .primary { flex: 0 0 auto; } .viewbar { align-items: flex-start; flex-direction: column; } .view-actions { justify-content: flex-start; } .customize-grid { grid-template-columns: 1fr; } .filters { align-items: stretch; flex-direction: column; } .filters select { width: 100%; } .title-wrap p { display: none; } }
</style>
