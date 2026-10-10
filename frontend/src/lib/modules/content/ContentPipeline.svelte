<script lang="ts">
	import type { ContentItem } from '$lib/api/content';
	import { CalendarDays, Clapperboard, ExternalLink, Film, Megaphone, MessageSquare, Pencil, Plus, Send, Trash2, Users } from 'lucide-svelte';
	import {
		CONTENT_THEME_COLORS,
		CONTENT_THEMES,
		DEFAULT_CARD_SETTINGS,
		analyticsEngagement,
		formatMetric,
		hasAnalytics,
		INITIAL_STAGE_CARD_LIMIT,
		STAGE_CARD_INCREMENT,
		STAGE_META,
		STAGES,
		type ContentBoard,
		type ContentCardSettings,
		type StageId
	} from './model';

	type Props = {
		boards: ContentBoard[];
		stages?: StageId[];
		onNew: (stage: StageId, profile?: string, workstream?: string, theme?: string) => void;
		onEdit: (item: ContentItem) => void;
		onPost: (item: ContentItem) => void;
		onRemove: (item: ContentItem) => void;
		onMove: (item: ContentItem, stage: StageId, workstream?: string, beforeItemId?: string, theme?: string) => void | Promise<void>;
		onReorderWorkstream?: (profile: string, workstream: string, beforeWorkstream: string) => void;
		topicsByWorkstream?: Record<string, string[]>;
		onAddTopic?: (profile: string, workstream: string) => void;
		onRenameTopic?: (profile: string, workstream: string, theme: string) => void;
		onRemoveTopic?: (profile: string, workstream: string, theme: string) => void;
		themeNames?: string[];
		themeColors?: Record<string, string>;
		cardSettings?: ContentCardSettings;
	};

	let {
		boards,
		stages = STAGES,
		onNew,
		onEdit,
		onPost,
		onRemove,
		onMove,
		onReorderWorkstream,
		topicsByWorkstream = {},
		onAddTopic,
		onRenameTopic,
		onRemoveTopic,
		themeNames = CONTENT_THEMES,
		themeColors = CONTENT_THEME_COLORS,
		cardSettings = DEFAULT_CARD_SETTINGS
	}: Props = $props();
	let draggingItemId = $state<string | null>(null);
	let draggingWorkstream = $state<{ profile: string; name: string } | null>(null);
	let expanded = $state<Record<string, number>>({});
	let activeThemes = $state<Record<string, string>>({});
	let lastCreateActionAt = 0;
	const ALL_TOPICS = 'all';

	function key(profile: string, workstream: string, stage: StageId, theme = ALL_TOPICS) {
		return `${profile}::${workstream}::${theme}::${stage}`;
	}

	function limit(profile: string, workstream: string, stage: StageId, theme = ALL_TOPICS) {
		return expanded[key(profile, workstream, stage, theme)] ?? INITIAL_STAGE_CARD_LIMIT;
	}

	function setLimit(profile: string, workstream: string, stage: StageId, value: number, theme = ALL_TOPICS) {
		expanded = { ...expanded, [key(profile, workstream, stage, theme)]: value };
	}

	function themeStateKey(profile: string, workstream: string) {
		return `${profile}::${workstream}`;
	}

	function activeTheme(profile: string, workstream: string) {
		return activeThemes[themeStateKey(profile, workstream)] ?? ALL_TOPICS;
	}

	function setActiveTheme(profile: string, workstream: string, theme: string) {
		activeThemes = { ...activeThemes, [themeStateKey(profile, workstream)]: theme };
	}

	function itemTheme(item: ContentItem) {
		return item.theme?.trim() || 'Uncategorized';
	}

	function themeOptions(profile: string, workstream: string, items: ContentItem[]) {
		const configured = topicsByWorkstream[themeStateKey(profile, workstream)] ?? activeThemeList();
		const extras = items.map(itemTheme).filter((theme) => !configured.includes(theme));
		return [...configured, ...extras];
	}

	function safeActiveTheme(profile: string, workstream: string, items: ContentItem[]) {
		const selected = activeTheme(profile, workstream);
		if (selected === ALL_TOPICS) return selected;
		return themeOptions(profile, workstream, items).includes(selected) ? selected : ALL_TOPICS;
	}

	function canEditTopic(theme: string) {
		return theme !== 'Uncategorized';
	}

	function activeThemeList() {
		return [...themeNames, 'Uncategorized'];
	}

	function themeColor(theme: string) {
		return themeColors[theme] || CONTENT_THEME_COLORS[theme] || '#0f766e';
	}

	function themeCount(items: ContentItem[], theme: string) {
		if (theme === ALL_TOPICS) return items.length;
		return items.filter((item) => itemTheme(item) === theme).length;
	}

	function themedColumns(columns: ContentBoard['workstreams'][number]['columns'], theme: string) {
		if (theme === ALL_TOPICS) return columns;
		return columns.map((column) => ({ ...column, items: column.items.filter((item) => itemTheme(item) === theme) }));
	}

	function themeForWrite(theme: string) {
		return theme === ALL_TOPICS ? undefined : theme;
	}

	function startDrag(event: DragEvent, item: ContentItem) {
		draggingItemId = item.id;
		draggingWorkstream = null;
		event.dataTransfer?.setData('text/plain', item.id);
		event.dataTransfer?.setData('application/x-content-card', item.id);
		if (event.dataTransfer) event.dataTransfer.effectAllowed = 'move';
	}

	function startWorkstreamDrag(event: DragEvent, profile: string, name: string) {
		draggingWorkstream = { profile, name };
		draggingItemId = null;
		event.dataTransfer?.setData('text/plain', name);
		event.dataTransfer?.setData('application/x-content-workstream', JSON.stringify({ profile, name }));
		if (event.dataTransfer) event.dataTransfer.effectAllowed = 'move';
	}

	function allowDrop(event: DragEvent) {
		event.preventDefault();
		if (event.dataTransfer) event.dataTransfer.dropEffect = 'move';
		scrollNearDragEdge(event);
	}

	function scrollNearDragEdge(event: DragEvent) {
		const scrollEl = document.querySelector('.contentos') as HTMLElement | null;
		const rect = scrollEl?.getBoundingClientRect();
		const top = rect?.top ?? 0;
		const bottom = rect?.bottom ?? window.innerHeight;
		const edge = 88;
		const speed = 24;
		let delta = 0;
		if (event.clientY < top + edge) delta = -speed;
		if (event.clientY > bottom - edge) delta = speed;
		if (!delta) return;
		if (scrollEl) scrollEl.scrollBy({ top: delta });
		else window.scrollBy({ top: delta });
	}

	function drop(event: DragEvent, stage: StageId, workstream: string, theme = ALL_TOPICS) {
		event.preventDefault();
		const id = event.dataTransfer?.getData('text/plain') || draggingItemId;
		draggingItemId = null;
		const item = boards.flatMap((board) => board.items).find((candidate) => candidate.id === id);
		if (item) void onMove(item, stage, workstream, undefined, themeForWrite(theme));
	}

	function dropOnCard(event: DragEvent, target: ContentItem, stage: StageId, workstream: string, theme = ALL_TOPICS) {
		event.preventDefault();
		event.stopPropagation();
		const id = event.dataTransfer?.getData('application/x-content-card') || draggingItemId;
		draggingItemId = null;
		if (!id || id === target.id) return;
		const item = boards.flatMap((board) => board.items).find((candidate) => candidate.id === id);
		if (item) void onMove(item, stage, workstream, target.id, themeForWrite(theme));
	}

	function dropWorkstream(event: DragEvent, profile: string, beforeWorkstream: string) {
		const raw = event.dataTransfer?.getData('application/x-content-workstream');
		const dragged = raw ? JSON.parse(raw) as { profile: string; name: string } : draggingWorkstream;
		if (!dragged || dragged.profile !== profile || dragged.name === beforeWorkstream) return;
		event.preventDefault();
		event.stopPropagation();
		draggingWorkstream = null;
		onReorderWorkstream?.(profile, dragged.name, beforeWorkstream);
	}

	function openDetails(event: Event, item: ContentItem) {
		event.preventDefault();
		event.stopPropagation();
		draggingItemId = null;
		onEdit(item);
	}

	function openDetailsAfterPress(event: Event, item: ContentItem) {
		event.preventDefault();
		event.stopPropagation();
		if (draggingItemId && draggingItemId !== item.id) return;
		draggingItemId = null;
		onEdit(item);
	}

	function keepControlClick(event: Event) {
		event.stopPropagation();
	}

	function runCreateAction(event: Event, action: () => void) {
		event.preventDefault();
		event.stopPropagation();
		const now = Date.now();
		if (now - lastCreateActionAt < 180) return;
		lastCreateActionAt = now;
		action();
	}

	function startCreateAction(event: PointerEvent, action: () => void) {
		runCreateAction(event, action);
	}
</script>

{#if boards.length === 0}
	<div class="empty"><p>No content matches the current filters.</p></div>
{:else}
	<div class="boards" aria-label="Content production boards">
		{#each boards as board (board.profile)}
			<section class="profile-board" aria-label={`${board.profile} content board`}>
				<header class="profile-head">
					<div><h2>{board.profile}</h2><p>{board.items.length} content {board.items.length === 1 ? 'piece' : 'pieces'}</p></div>
					<button type="button" draggable="false" onpointerdown={(event) => startCreateAction(event, () => onNew('idea', board.profile))} onmousedown={(event) => runCreateAction(event, () => onNew('idea', board.profile))} onclick={(event) => runCreateAction(event, () => onNew('idea', board.profile))}><Plus size={14} />New content</button>
				</header>
				<div class="workstreams">
					{#each board.workstreams as workstream (workstream.name)}
						<section role="group" aria-label={`${workstream.name} pipeline`} class:dragging={draggingWorkstream?.profile === board.profile && draggingWorkstream?.name === workstream.name} class="workstream" ondragover={allowDrop} ondrop={(event) => dropWorkstream(event, board.profile, workstream.name)}>
							<header role="button" tabindex="0" aria-label={`Reorder ${workstream.name}`} class="workstream-head" draggable="true" ondragstart={(event) => startWorkstreamDrag(event, board.profile, workstream.name)} ondragend={() => (draggingWorkstream = null)}>
								<div><h3>{workstream.name}</h3><p>{workstream.items.length} {workstream.items.length === 1 ? 'card' : 'cards'}</p></div>
								<button type="button" draggable="false" onpointerdown={(event) => startCreateAction(event, () => onNew('idea', board.profile, workstream.name))} onmousedown={(event) => runCreateAction(event, () => onNew('idea', board.profile, workstream.name))} onclick={(event) => runCreateAction(event, () => onNew('idea', board.profile, workstream.name))}><Plus size={14} />New</button>
							</header>
							{#if true}
								{@const selectedTheme = safeActiveTheme(board.profile, workstream.name, workstream.items)}
								<div class="topic-strip" role="tablist" aria-label={`${workstream.name} topics`}>
									<button class:active={selectedTheme === ALL_TOPICS} type="button" role="tab" draggable="false" onpointerdown={keepControlClick} onmousedown={keepControlClick} aria-selected={selectedTheme === ALL_TOPICS} onclick={(event) => { event.stopPropagation(); setActiveTheme(board.profile, workstream.name, ALL_TOPICS); }}>
										<span>All topics</span><strong>{themeCount(workstream.items, ALL_TOPICS)}</strong>
									</button>
									{#each themeOptions(board.profile, workstream.name, workstream.items) as theme}
										<div class:active={selectedTheme === theme} class="topic-pill" style={`--theme-color: ${themeColor(theme)}`}>
											<button class="topic-main" type="button" role="tab" draggable="false" onpointerdown={keepControlClick} onmousedown={keepControlClick} aria-selected={selectedTheme === theme} onclick={(event) => { event.stopPropagation(); setActiveTheme(board.profile, workstream.name, theme); }}>
												<span>{theme}</span><strong>{themeCount(workstream.items, theme)}</strong>
											</button>
											{#if canEditTopic(theme)}
												<button class="topic-tool" type="button" draggable="false" onpointerdown={keepControlClick} onmousedown={keepControlClick} title={`Rename ${theme}`} aria-label={`Rename ${theme}`} onclick={(event) => { event.stopPropagation(); onRenameTopic?.(board.profile, workstream.name, theme); }}>
													<Pencil size={11} />
												</button>
												<button class="topic-tool danger" type="button" draggable="false" onpointerdown={keepControlClick} onmousedown={keepControlClick} title={`Remove ${theme}`} aria-label={`Remove ${theme}`} onclick={(event) => { event.stopPropagation(); onRemoveTopic?.(board.profile, workstream.name, theme); }}>
													<Trash2 size={11} />
												</button>
											{/if}
										</div>
									{/each}
									<button class="topic-add" type="button" draggable="false" onpointerdown={(event) => startCreateAction(event, () => onAddTopic?.(board.profile, workstream.name))} onmousedown={(event) => runCreateAction(event, () => onAddTopic?.(board.profile, workstream.name))} aria-label={`Add topic to ${workstream.name}`} onclick={(event) => runCreateAction(event, () => onAddTopic?.(board.profile, workstream.name))}>
										<Plus size={12} />Topic
									</button>
								</div>
								<div class="stage-grid">
									{#each themedColumns(workstream.columns, selectedTheme) as column (column.stage)}
										<section role="group" aria-label={`${workstream.name} ${STAGE_META[column.stage].label}`} class="stage" style={`--stage-color: ${STAGE_META[column.stage].color}`} ondragover={allowDrop} ondrop={(event) => drop(event, column.stage, workstream.name, selectedTheme)}>
											<header><span>{STAGE_META[column.stage].label}</span><strong>{column.items.length}</strong></header>
											<div class="cards">
												{#each column.items.slice(0, limit(board.profile, workstream.name, column.stage, selectedTheme)) as item (item.id)}
													<article class:dragging={draggingItemId === item.id} class:compact={cardSettings.compact} style={`--theme-color: ${themeColor(itemTheme(item))}`} draggable="true" ondragstart={(event) => startDrag(event, item)} ondragend={() => (draggingItemId = null)} ondragover={allowDrop} ondrop={(event) => dropOnCard(event, item, column.stage, workstream.name, selectedTheme)}>
														<div class="card-top"><span>{item.content_type}</span>{#if item.channel}<b>{item.channel}</b>{/if}</div>
														<button class="title" type="button" draggable="false" onpointerdown={keepControlClick} onmousedown={keepControlClick} onmouseup={(event) => openDetailsAfterPress(event, item)} onclick={(event) => openDetailsAfterPress(event, item)}>{item.title}</button>
														{#if cardSettings.showHook && item.hook}<p class="hook">{item.hook}</p>{/if}
														{#if item.theme}<div class="theme">{item.theme}</div>{/if}
														{#if cardSettings.showMeta}
															<div class="meta">
																{#if item.client}<span><Users size={12} />{item.client}</span>{/if}
																{#if item.campaign}<span><Megaphone size={12} />{item.campaign}</span>{/if}
																{#if item.editor}<span><Film size={12} />{item.editor}</span>{/if}
																{#if item.due_date}<span><CalendarDays size={12} />Film {item.due_date}</span>{/if}
																{#if item.publish_date}<span><CalendarDays size={12} />Post {item.publish_date}</span>{/if}
															</div>
														{/if}
														{#if cardSettings.showLinks}
															<div class="links">
																{#if item.asset_link}<a href={item.asset_link} target="_blank" rel="noopener noreferrer"><Clapperboard size={13} />Asset</a>{/if}
																{#if item.review_link}<a href={item.review_link} target="_blank" rel="noopener noreferrer"><MessageSquare size={13} />Review</a>{/if}
																{#if item.link}<a href={item.link} target="_blank" rel="noopener noreferrer"><ExternalLink size={13} />Live</a>{/if}
															</div>
														{/if}
														{#if cardSettings.showAnalytics && hasAnalytics(item)}
															<div class="analytics" aria-label="Content analytics">
																<span>Views {formatMetric(item.views)}</span>
																<span>Eng {formatMetric(analyticsEngagement(item))}</span>
																<span>Saves {formatMetric(item.saves)}</span>
																<span>Shares {formatMetric(item.shares)}</span>
															</div>
														{/if}
														<div class="card-actions">
															<select value={column.stage} draggable="false" onpointerdown={keepControlClick} onmousedown={keepControlClick} onchange={(event) => void onMove(item, (event.currentTarget as HTMLSelectElement).value as StageId)} aria-label="Move stage">
																{#each stages as stage}<option value={stage}>{STAGE_META[stage].label}</option>{/each}
															</select>
															{#if column.stage === 'to_post'}<button class="icon post" type="button" draggable="false" title="Post" aria-label={`Post ${item.title}`} onpointerdown={keepControlClick} onmousedown={keepControlClick} onclick={(event) => { event.stopPropagation(); onPost(item); }}><Send size={14} /></button>{/if}
															<button class="icon" type="button" draggable="false" title="Edit" aria-label={`Edit ${item.title}`} onpointerdown={keepControlClick} onmousedown={keepControlClick} onmouseup={(event) => openDetailsAfterPress(event, item)} onclick={(event) => openDetailsAfterPress(event, item)}><Pencil size={14} /></button>
															<button class="icon danger" type="button" draggable="false" title="Delete" aria-label={`Delete ${item.title}`} onpointerdown={keepControlClick} onmousedown={keepControlClick} onclick={(event) => { event.stopPropagation(); onRemove(item); }}><Trash2 size={14} /></button>
														</div>
													</article>
												{/each}
												{#if column.items.length > INITIAL_STAGE_CARD_LIMIT}
													<div class="more">
														<span>{Math.min(limit(board.profile, workstream.name, column.stage, selectedTheme), column.items.length)} of {column.items.length}</span>
														{#if limit(board.profile, workstream.name, column.stage, selectedTheme) < column.items.length}
																<button type="button" draggable="false" onpointerdown={keepControlClick} onmousedown={keepControlClick} onclick={(event) => { event.stopPropagation(); setLimit(board.profile, workstream.name, column.stage, Math.min(column.items.length, limit(board.profile, workstream.name, column.stage, selectedTheme) + STAGE_CARD_INCREMENT), selectedTheme); }}>Show more</button>
															{:else}<button type="button" draggable="false" onpointerdown={keepControlClick} onmousedown={keepControlClick} onclick={(event) => { event.stopPropagation(); setLimit(board.profile, workstream.name, column.stage, INITIAL_STAGE_CARD_LIMIT, selectedTheme); }}>Collapse</button>{/if}
													</div>
												{/if}
													<button class="new-card" type="button" draggable="false" onpointerdown={(event) => startCreateAction(event, () => onNew(column.stage, board.profile, workstream.name, themeForWrite(selectedTheme)))} onmousedown={(event) => runCreateAction(event, () => onNew(column.stage, board.profile, workstream.name, themeForWrite(selectedTheme)))} onclick={(event) => runCreateAction(event, () => onNew(column.stage, board.profile, workstream.name, themeForWrite(selectedTheme)))}><Plus size={14} />New page</button>
											</div>
										</section>
									{/each}
								</div>
							{/if}
						</section>
					{/each}
				</div>
			</section>
		{/each}
	</div>
{/if}

<style>
	.boards, .workstreams { display: grid; gap: 16px; }
	.boards { padding: 16px; }
	.profile-board { min-width: 0; border: 1px solid var(--dbd); border-radius: 8px; overflow: hidden; }
	.profile-head, .workstream-head { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 14px 16px; border-bottom: 1px solid var(--dbd); }
	.profile-head h2, .workstream-head h3 { margin: 0; color: var(--dt); letter-spacing: 0; }
	.profile-head h2 { font-size: 1rem; } .workstream-head h3 { font-size: .88rem; }
	.profile-head p, .workstream-head p { margin: 3px 0 0; color: var(--dt3); font-size: .72rem; }
	.profile-head button, .workstream-head button, .new-card, .more button { display: inline-flex; align-items: center; gap: 5px; border: 1px solid var(--dbd); border-radius: 6px; background: transparent; color: var(--dt2); font: inherit; font-size: .74rem; cursor: pointer; }
	.profile-head button, .workstream-head button { min-height: 30px; padding: 5px 9px; }
	.workstreams { padding: 12px; background: color-mix(in srgb, var(--dt) 1.5%, var(--dbg)); }
	.workstream { min-width: 0; border: 1px solid var(--dbd); border-radius: 7px; background: var(--dbg); overflow: hidden; }
	.workstream.dragging { opacity: .55; }
	.workstream-head { cursor: grab; }
	.workstream-head:active { cursor: grabbing; }
	.topic-strip { display: flex; gap: 6px; min-width: 0; padding: 9px 10px; border-bottom: 1px solid var(--dbd); overflow-x: auto; background: color-mix(in srgb, var(--dt) 1.5%, transparent); }
	.topic-strip button, .topic-pill { display: inline-flex; align-items: center; flex: 0 0 auto; min-height: 28px; border: 1px solid var(--dbd); border-radius: 999px; background: var(--dbg); color: var(--dt2); font: inherit; font-size: .68rem; font-weight: 720; }
	.topic-strip button { gap: 7px; max-width: 190px; padding: 5px 8px; cursor: pointer; }
	.topic-pill { gap: 2px; max-width: 240px; padding: 0 3px 0 0; overflow: hidden; }
	.topic-pill.active, .topic-strip > button.active { border-color: color-mix(in srgb, var(--theme-color, #0f766e) 45%, var(--dbd)); color: var(--theme-color, #0f766e); background: color-mix(in srgb, var(--theme-color, #0f766e) 8%, var(--dbg)); }
	.topic-main { min-width: 0; border: 0 !important; background: transparent !important; }
	.topic-tool { display: grid !important; width: 22px; height: 22px; min-height: 22px !important; place-items: center; padding: 0 !important; border: 0 !important; color: var(--dt3) !important; }
	.topic-tool:hover { color: #0f766e !important; background: color-mix(in srgb, var(--dt) 5%, transparent) !important; }
	.topic-tool.danger:hover { color: #dc2626 !important; }
	.topic-add { border-style: dashed !important; }
	.topic-strip span { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
	.topic-strip strong { display: grid; min-width: 18px; height: 18px; place-items: center; border-radius: 999px; background: color-mix(in srgb, var(--dt) 7%, transparent); color: var(--dt3); font-size: .61rem; }
	.topic-pill.active strong, .topic-strip > button.active strong { color: var(--theme-color, #0f766e); background: color-mix(in srgb, var(--theme-color, #0f766e) 12%, transparent); }
	.stage-grid { display: grid; grid-template-columns: repeat(9, minmax(220px, 1fr)); gap: 8px; padding: 10px; overflow-x: auto; }
	.stage { min-width: 220px; border: 1px solid var(--dbd); border-top: 2px solid var(--stage-color); border-radius: 7px; background: color-mix(in srgb, var(--dt) 2%, var(--dbg)); }
	.stage > header { display: flex; align-items: center; justify-content: space-between; padding: 9px 10px; border-bottom: 1px solid var(--dbd); }
	.stage > header span { color: var(--dt2); font-size: .7rem; font-weight: 750; text-transform: uppercase; }
	.stage > header strong { color: var(--dt3); font-size: .72rem; }
	.cards { display: grid; gap: 7px; padding: 7px; }
	article { min-width: 0; padding: 10px; border: 1px solid var(--dbd); border-left: 3px solid color-mix(in srgb, var(--theme-color, var(--dbd)) 55%, var(--dbd)); border-radius: 7px; background: var(--dbg); transition: opacity .15s; }
	article.compact { padding: 8px; }
	article.compact .hook, article.compact .meta, article.compact .links, article.compact .analytics { display: none; }
	article.dragging { opacity: .45; }
	.card-top { display: flex; justify-content: space-between; gap: 6px; }
	.card-top span, .card-top b { color: var(--dt3); font-size: .65rem; font-weight: 700; text-transform: capitalize; }
	.title { display: block; width: 100%; margin: 7px 0 0; padding: 0; border: 0; background: transparent; color: var(--dt); font: inherit; font-size: .8rem; font-weight: 700; text-align: left; cursor: pointer; }
	.hook { display: -webkit-box; margin: 6px 0 0; overflow: hidden; color: var(--dt2); font-size: .7rem; line-height: 1.4; -webkit-box-orient: vertical; -webkit-line-clamp: 2; }
	.theme { display: inline-flex; width: fit-content; max-width: 100%; margin-top: 7px; padding: 3px 6px; border: 1px solid color-mix(in srgb, var(--theme-color, #0f766e) 24%, var(--dbd)); border-radius: 5px; color: var(--theme-color, #0f766e); background: color-mix(in srgb, var(--theme-color, #0f766e) 7%, transparent); font-size: .63rem; font-weight: 760; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
	.meta { display: grid; gap: 4px; margin-top: 8px; }
	.meta span, .links a { display: flex; align-items: center; gap: 4px; overflow: hidden; color: var(--dt3); font-size: .66rem; text-overflow: ellipsis; white-space: nowrap; }
	.links { display: flex; flex-wrap: wrap; gap: 8px; margin-top: 8px; }
	.links a { color: #0f766e; text-decoration: none; }
	.analytics { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 4px; margin-top: 8px; padding: 7px; border: 1px solid var(--dbd); border-radius: 6px; background: color-mix(in srgb, var(--dt) 2%, transparent); }
	.analytics span { overflow: hidden; color: var(--dt2); font-size: .64rem; font-weight: 680; text-overflow: ellipsis; white-space: nowrap; }
	.card-actions { display: flex; align-items: center; gap: 5px; margin-top: 9px; }
	.card-actions select { min-width: 0; flex: 1; height: 28px; border: 1px solid var(--dbd); border-radius: 5px; background: var(--dbg); color: var(--dt2); font: inherit; font-size: .67rem; }
	.icon { display: grid; flex: 0 0 28px; width: 28px; height: 28px; place-items: center; padding: 0; border: 1px solid var(--dbd); border-radius: 5px; background: transparent; color: var(--dt3); cursor: pointer; }
	.icon.post { color: #0f766e; } .icon.danger:hover { color: #dc2626; }
	.new-card { justify-content: center; min-height: 30px; border-style: dashed; }
	.more { display: flex; align-items: center; justify-content: space-between; gap: 5px; color: var(--dt3); font-size: .65rem; }
	.more button { padding: 3px 5px; }
	.empty { display: grid; place-items: center; min-height: 300px; color: var(--dt3); font-size: .84rem; }
	button { -webkit-user-select: none; user-select: none; }
</style>
