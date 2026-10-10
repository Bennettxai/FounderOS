<!-- The /os/brain knowledge graph: a Svelte port of FounderOS v1
     components/KnowledgeGraph.tsx. The operator at the core (rendered as the
     Optimal Engine memory constellation), pillars, their SOP tasks, the one
     worker who does each, and their tools: concentric, with live d3-force
     physics, a slowly turning orbital backdrop and a drifting grid. Hover any
     node to trace its pillar chain; click a pillar to grow it into a
     bottom-to-top tree on the turning department wheel (← / → step it); click
     a task, worker or tool for its card; click the core to dive into the
     memory. The Fullscreen tab opens the department wheel over the whole
     screen. Loaded lazily by BrainGraphView behind a same-size skeleton. -->
<script lang="ts">
	import { onMount, untrack } from 'svelte';
	import { forceCollide, forceLink, forceManyBody, forceRadial, forceSimulation, forceX, forceY, type Simulation } from 'd3-force';
	import { ArrowLeft, ChevronLeft, ChevronRight, Maximize2, Minimize2, X } from '$lib/founderos/icons';
	import './kg.css';
	import GraphDirectory from './GraphDirectory.svelte';
	import GraphNodeCard from './GraphNodeCard.svelte';
	import KindIcon from './KindIcon.svelte';
	import KnowledgeGraphFullscreen from './KnowledgeGraphFullscreen.svelte';
	import AgentHarnessCard from './detail/AgentHarnessCard.svelte';
	import GraphHumanDetailCard from './detail/GraphHumanDetailCard.svelte';
	import HeadDetailCard from './detail/HeadDetailCard.svelte';
	import MemoryCoreCard from './detail/MemoryCoreCard.svelte';
	import MemoryNoteCard from './detail/MemoryNoteCard.svelte';
	import SopTaskDetailCard from './detail/SopTaskDetailCard.svelte';
	import ToolDetailCard from './detail/ToolDetailCard.svelte';
	import { ACTION_LENSES, ENTITY_LENSES, FUNCTION_LENSES, lensNodeSet, type Lens } from './graph-lens';
	import { SELF_ID, buildToolWiki, headForDepartment, orderGraphDepartments, prettifySlug, themedNodeColor, toolSlugOf } from './kg';
	import { cameraRect, lerpRect, memoryNodePos, pickRestTier, R_CORE, type MemoryGraph, type MemoryNode, type Rect } from './memory-core';
	import { searchMemoryNotes } from './memory-search';
	import { playbookFor } from './sop-playbooks';
	import {
		branchPath,
		branchWidth,
		cyclicDeltaF,
		edgeArc,
		focusWheel,
		radialRestLayout,
		responsiveRingR,
		rotateAbout,
		shortestAngleDelta,
		treeLayout,
		wheelPoint,
		wheelStageGeom,
		wheelStageSpot,
		type Pt,
		type TreeLayoutResult
	} from './tree-layout';
	import type { BrainPage, DirectoryGroup, KGNode, KGNodeKind } from './types';

	let {
		page,
		memory = null,
		memoryError = null,
		activePillars,
		onPanel,
		onShowAllPillars,
		fill = true
	}: {
		page: BrainPage;
		memory?: MemoryGraph | null;
		memoryError?: string | null;
		/** department ids still switched on by the pillar chips; undefined = no filter */
		activePillars?: string[];
		/** told whether the graph has been tapped into (a focused pillar, the open
		 *  core or a docked card), so the satellites can step out of the way */
		onPanel?: (quiet: boolean) => void;
		/** wired = the directory's empty state can switch every pillar back on */
		onShowAllPillars?: () => void;
		fill?: boolean;
	} = $props();

	// ── geometry (KnowledgeGraph.tsx constants) ──
	const W = 880;
	const H = 600;
	const CX = W / 2;
	const CY = H / 2;
	const RING_R = responsiveRingR(W, H);
	const BOARD_R = RING_R[1] * 0.68;
	const MARGIN = 78;
	const FOCUS_WHEEL = focusWheel(W, H, RING_R);
	const WHEEL_GEOM = wheelStageGeom(W, H);
	const RIM_DELTA_DEG = (WHEEL_GEOM.delta * 180) / Math.PI;

	const CAT: Record<KGNodeKind, { color: string; label: string; r: number }> = {
		self: { color: 'var(--bn-text)', label: 'Engine', r: 18 },
		team: { color: 'var(--bn-brain-1)', label: 'Pillars', r: 15 },
		board: { color: 'var(--bn-text)', label: 'Board agents', r: 10 },
		task: { color: 'var(--bn-text-2)', label: 'SOP tasks', r: 7 },
		person: { color: 'var(--bn-kg-person, var(--bn-warn))', label: 'Humans', r: 10 },
		employee: { color: 'var(--bn-kg-employee, var(--bn-accent))', label: 'AI agents', r: 10 },
		tool: { color: 'var(--bn-kg-tool)', label: 'Tools', r: 7.5 }
	};
	const TIER_OPACITY: Record<KGNodeKind, number> = { self: 1, team: 1, board: 0.98, person: 0.98, employee: 0.98, task: 0.94, tool: 0.94 };
	const LEGEND_KINDS: KGNodeKind[] = ['team', 'board', 'task', 'person', 'employee', 'tool'];
	const nodeColor = (n: KGNode) => (n.color ? themedNodeColor(n.color) : CAT[n.kind].color);
	const EDGE_COLOR: Record<string, string> = {
		pillar: 'var(--bn-text)',
		sop: 'var(--bn-text-2)',
		does: 'var(--bn-accent)',
		member: 'var(--bn-text-2)',
		uses: 'var(--bn-brain-2)',
		reports: 'var(--bn-accent)',
		board: 'var(--bn-text-2)'
	};
	const shortLabel = (n: KGNode) => (n.kind === 'task' && n.label.length > 20 ? `${n.label.slice(0, 18).trimEnd()}…` : n.label);
	const agoLabel = (iso: string): string => {
		const ms = Date.now() - new Date(iso).getTime();
		if (!Number.isFinite(ms) || ms < 0) return '';
		const m = Math.floor(ms / 60_000);
		if (m < 1) return 'just now';
		if (m < 60) return `${m}m ago`;
		const h = Math.floor(m / 60);
		if (h < 24) return `${h}h ago`;
		return `${Math.floor(h / 24)}d ago`;
	};

	// ── the memory core ──
	const CORE_SCALE_TREE = 38 / R_CORE;
	const CORE_SCALE_EXPANDED = 96 / R_CORE;
	const TEAM_PUSH_EXPANDED = 1.6;
	const hashStr = (s: string) => {
		let h = 0;
		for (const ch of s) h = (h * 31 + ch.charCodeAt(0)) >>> 0;
		return h;
	};
	const HUB_COLOR = 'var(--bn-kg-mem, #e35c35)';
	const SYNAPSE_COLOR = 'var(--kg-synapse, #ffb08a)';
	const SYNAPSE_N = 14;
	const memColor = (_m: MemoryNode) => HUB_COLOR;
	const HEX_PTS_CACHE = new Map<number, string>();
	const hexPts = (r: number): string => {
		const key = Math.round(r * 100);
		let s = HEX_PTS_CACHE.get(key);
		if (!s) {
			s = Array.from({ length: 6 }, (_, k) => {
				const a = (k * Math.PI) / 3 - Math.PI / 2;
				return `${(r * Math.cos(a)).toFixed(3)},${(r * Math.sin(a)).toFixed(3)}`;
			}).join(' ');
			HEX_PTS_CACHE.set(key, s);
		}
		return s;
	};
	const MEM_LAYERS = [
		'--kg-ddx: 1.1px; --kg-ddy: -0.8px; animation: kg-note-drift 19s ease-in-out infinite alternate, kg-breathe 6.5s ease-in-out infinite alternate',
		'--kg-ddx: -0.9px; --kg-ddy: 1.2px; animation: kg-note-drift 24s ease-in-out -8s infinite alternate, kg-breathe 8.5s ease-in-out -3s infinite alternate',
		'--kg-ddx: 0.7px; --kg-ddy: 1px; animation: kg-note-drift 29s ease-in-out -15s infinite alternate, kg-breathe 11s ease-in-out -6s infinite alternate'
	];
	const MEM_STIRS = [
		'--kg-sdx: 2.6px; --kg-sdy: 1.8px; animation: kg-stir 1.7s ease-in-out infinite alternate',
		'--kg-sdx: -2.2px; --kg-sdy: 2.4px; animation: kg-stir 2.1s ease-in-out -0.6s infinite alternate',
		'--kg-sdx: 1.9px; --kg-sdy: -2.5px; animation: kg-stir 2.5s ease-in-out -1.2s infinite alternate'
	];
	const memLayerOf = (id: string) => hashStr(id) % MEM_LAYERS.length;
	const memNodeR = (n: MemoryNode) =>
		n.type === 'folder' ? 2.4 : n.links === 0 ? 0.45 : 0.45 + Math.min(0.95, n.links * 0.1 + (n.wordCount ?? 0) / 4000);
	const CAM_EASE = 0.075;
	const CAM_EASE_HOME = 0.3;
	const fixedLabel = (px: number, groupScale = 1) => `font-size: calc(${(px / groupScale).toFixed(3)}px * var(--kg-cam-k, 1))`;

	type SimNode = KGNode & { x: number; y: number; vx?: number; vy?: number; fx?: number | null; fy?: number | null };
	type SimLink = { source: SimNode | string; target: SimNode | string; kind: string };

	const graph = $derived(page.graph);
	const agents = $derived(page.agents);
	const people = $derived(page.people);
	const tasks = $derived(page.tasks);
	const departments = $derived(page.departments);
	const runsByAgent = $derived(page.runsByAgent);
	const boardLeads = $derived(page.boardLeads);
	const boardAgents = $derived(page.boardAgents);
	const memoryOn = $derived(!!memory && memory.nodes.length > 0);

	// ── interaction state ──
	let hoverId = $state<string | null>(null);
	let cardId = $state<string | null>(null);
	let focusId = $state<string | null>(null);
	let selectedAgentId = $state<string | null>(null);
	let selectedToolId = $state<string | null>(null);
	let selectedTaskId = $state<string | null>(null);
	let selectedHumanId = $state<string | null>(null);
	let selectedHeadId = $state<string | null>(null);
	let selectedBoardId = $state<string | null>(null);
	let coreExpanded = $state(false);
	let selectedMemoryId = $state<string | null>(null);
	let memHoverId = $state<string | null>(null);
	let memQuery = $state('');
	let memSearchEl = $state<HTMLInputElement | null>(null);
	let fullscreen = $state(false);
	let directoryCollapsed = $state(false);
	let detailExpanded = $state(false);
	let lensId = $state<string | null>(null);

	// ── the sim's live state (mutated by d3, painted once per frame) ──
	let sim: Simulation<SimNode, undefined> | null = null;
	let nodes = $state.raw<SimNode[]>([]);
	let links = $state.raw<SimLink[]>([]); // reassigned when the graph loads; the web must re-render
	let pos = $state.raw<Map<string, Pt>>(new Map());
	let svgEl = $state<SVGSVGElement | null>(null);
	let drag: { id: string; moved: boolean; startX: number; startY: number } | null = null;
	let suppressClick = false;
	let userView: Rect | null = null;
	let pan: { px: number; py: number; x: number; y: number; k: number; moved: boolean } | null = null;
	let panSuppress = false;

	const agentById = $derived(new Map(agents.map((a) => [`emp:${a.id}`, a])));
	const personById = $derived(new Map(people.map((p) => [`person:${p.id}`, p])));
	const taskById = $derived(new Map(tasks.map((t) => [`task:${t.id}`, t])));

	// adjacency + relationship maps: team —sop→ task —does→ worker —uses→ tool
	const rel = $derived.by(() => {
		const adjacency = new Map<string, Set<string>>();
		const tasksOfTeam = new Map<string, string[]>();
		const teamOfTask = new Map<string, string>();
		const workerOfTask = new Map<string, string>();
		const taskOfWorker = new Map<string, string>();
		const teamOfWorker = new Map<string, string>();
		const workersOfTeam = new Map<string, string[]>();
		const toolsOfWorker = new Map<string, string[]>();
		const workersOfTool = new Map<string, string[]>();
		const byId = new Map(graph.nodes.map((n) => [n.id, n]));
		const push = (m: Map<string, string[]>, k: string, v: string) => (m.get(k) ?? m.set(k, []).get(k)!).push(v);
		for (const n of graph.nodes) adjacency.set(n.id, new Set([n.id]));
		for (const e of graph.edges) {
			adjacency.get(e.source)?.add(e.target);
			adjacency.get(e.target)?.add(e.source);
			if (e.kind === 'sop') {
				teamOfTask.set(e.target, e.source);
				push(tasksOfTeam, e.source, e.target);
			}
			if (e.kind === 'does') {
				workerOfTask.set(e.source, e.target);
				taskOfWorker.set(e.target, e.source);
			}
			if (e.kind === 'member') teamOfWorker.set(e.source, e.target);
			if (e.kind === 'uses') {
				push(toolsOfWorker, e.source, e.target);
				push(workersOfTool, e.target, e.source);
			}
		}
		for (const [task, worker] of workerOfTask) {
			const team = teamOfTask.get(task);
			if (team) teamOfWorker.set(worker, team);
		}
		for (const [worker, team] of teamOfWorker) push(workersOfTeam, team, worker);
		return { adjacency, byId, tasksOfTeam, teamOfTask, workerOfTask, taskOfWorker, teamOfWorker, workersOfTeam, toolsOfWorker, workersOfTool };
	});
	const byId = $derived(rel.byId);
	const isWorker = (kind: KGNodeKind) => kind === 'employee' || kind === 'person';

	function teamForFocus(id: string | null): string | null {
		if (!id) return null;
		const n = rel.byId.get(id);
		if (!n) return null;
		if (n.kind === 'team') return n.id;
		if (n.kind === 'task') return rel.teamOfTask.get(n.id) ?? null;
		if (isWorker(n.kind)) return rel.teamOfWorker.get(n.id) ?? null;
		if (n.kind === 'tool') {
			const w = (rel.workersOfTool.get(n.id) ?? [])[0];
			return w ? rel.teamOfWorker.get(w) ?? null : null;
		}
		return null;
	}
	function chainOfWorker(w: string, set: Set<string>) {
		set.add(w);
		const task = rel.taskOfWorker.get(w);
		if (task) set.add(task);
		const team = rel.teamOfWorker.get(w);
		if (team) set.add(team);
		for (const tool of rel.toolsOfWorker.get(w) ?? []) set.add(tool);
	}
	function litFor(id: string): Set<string> {
		const node = rel.byId.get(id);
		const set = new Set<string>([id]);
		if (!node) return set;
		if (node.kind === 'team') {
			set.add(SELF_ID);
			for (const w of rel.workersOfTeam.get(id) ?? []) chainOfWorker(w, set);
			for (const t of rel.tasksOfTeam.get(id) ?? []) set.add(t);
		} else if (node.kind === 'task') {
			set.add(SELF_ID);
			const team = rel.teamOfTask.get(id);
			if (team) set.add(team);
			const w = rel.workerOfTask.get(id);
			if (w) chainOfWorker(w, set);
		} else if (isWorker(node.kind)) {
			set.add(SELF_ID);
			chainOfWorker(id, set);
		} else if (node.kind === 'board') {
			set.add(SELF_ID);
		} else if (node.kind === 'tool') {
			for (const w of rel.workersOfTool.get(id) ?? []) chainOfWorker(w, set);
		} else {
			for (const m of rel.adjacency.get(id) ?? []) set.add(m);
		}
		return set;
	}

	const focusTeamId = $derived(teamForFocus(focusId));
	const focusSet = $derived.by(() => {
		if (!focusTeamId) return null;
		const set = new Set<string>([SELF_ID, focusTeamId]);
		for (const t of rel.tasksOfTeam.get(focusTeamId) ?? []) set.add(t);
		for (const w of rel.workersOfTeam.get(focusTeamId) ?? []) {
			set.add(w);
			for (const tool of rel.toolsOfWorker.get(w) ?? []) set.add(tool);
		}
		return set;
	});

	// one upright tree per pillar: the wheel mounts every department expanded
	const allTrees = $derived.by(() => {
		const byLabel = (a: string, b: string) => (rel.byId.get(a)?.label ?? '').localeCompare(rel.byId.get(b)?.label ?? '');
		const m = new Map<string, TreeLayoutResult>();
		for (const team of graph.nodes.filter((n) => n.kind === 'team')) {
			const taskIds = (rel.tasksOfTeam.get(team.id) ?? []).slice().sort(byLabel);
			const workerByTask: Record<string, string> = {};
			const toolsByWorker: Record<string, string[]> = {};
			for (const t of taskIds) {
				const w = rel.workerOfTask.get(t);
				if (!w) continue;
				workerByTask[t] = w;
				toolsByWorker[w] = (rel.toolsOfWorker.get(w) ?? []).slice().sort(byLabel);
			}
			m.set(team.id, treeLayout({ selfId: SELF_ID, teamId: team.id, taskIds, workerByTask, toolsByWorker, width: W, height: H, margin: MARGIN }));
		}
		return m;
	});
	const focusTree = $derived(focusTeamId ? allTrees.get(focusTeamId) ?? null : null);

	// the symmetric resting sunburst, board seats in the widest pillar gaps
	const restLayout = $derived.by(() => {
		const teams = orderGraphDepartments(
			graph.nodes.filter((n) => n.kind === 'team'),
			(t) => t.id.replace('team:', '')
		);
		const toolsByPillar = new Map<string, string[]>();
		for (const n of graph.nodes) {
			if (n.kind !== 'tool') continue;
			const users = rel.workersOfTool.get(n.id) ?? [];
			const team = users.length ? rel.teamOfWorker.get(users[0]) ?? null : null;
			if (team) (toolsByPillar.get(team) ?? toolsByPillar.set(team, []).get(team)!).push(n.id);
		}
		const pillars = teams.map((t) => ({
			teamId: t.id,
			taskIds: rel.tasksOfTeam.get(t.id) ?? [],
			workerIds: rel.workersOfTeam.get(t.id) ?? [],
			toolIds: toolsByPillar.get(t.id) ?? []
		}));
		const layout = radialRestLayout({ selfId: SELF_ID, pillars, ringR: RING_R, cx: CX, cy: CY });
		const board = graph.nodes.filter((n) => n.kind === 'board');
		if (board.length) {
			const TAU = Math.PI * 2;
			const pillarAngles = graph.nodes
				.filter((n) => n.kind === 'team')
				.map((t) => layout.positions.get(t.id))
				.filter((p): p is Pt => !!p)
				.map((p) => Math.atan2(p.y - CY, p.x - CX))
				.sort((a, b) => a - b);
			const gaps = pillarAngles.map((a, i) => ({ start: a, span: (pillarAngles[(i + 1) % pillarAngles.length] - a + TAU) % TAU || TAU }));
			const counts = gaps.map(() => 0);
			for (let s = 0; s < board.length; s++) {
				let best = 0;
				for (let i = 1; i < gaps.length; i++) if (gaps[i].span / (counts[i] + 1) > gaps[best].span / (counts[best] + 1)) best = i;
				if (gaps.length) counts[best]++;
			}
			const spots: number[] = [];
			gaps.forEach((g, i) => {
				for (let j = 0; j < counts[i]; j++) spots.push(g.start + g.span * ((j + 1) / (counts[i] + 1)));
			});
			spots.sort((a, b) => a - b);
			board.forEach((n, i) => {
				const a = spots[i] ?? -Math.PI / 2 + ((i + 0.5) / board.length) * Math.PI * 2;
				layout.positions.set(n.id, { x: CX + BOARD_R * Math.cos(a), y: CY + BOARD_R * Math.sin(a) });
			});
		}
		return layout;
	});

	// staggered label rows inside a focused tree
	const labelDy = $derived.by(() => {
		const m = new Map<string, number>();
		if (!focusTree) return m;
		const byDepth = new Map<number, { id: string; x: number }[]>();
		for (const [id, p] of focusTree.positions) {
			if (p.depth < 2) continue;
			(byDepth.get(p.depth) ?? byDepth.set(p.depth, []).get(p.depth)!).push({ id, x: p.x });
		}
		for (const entries of byDepth.values()) {
			entries.sort((a, b) => a.x - b.x);
			entries.forEach((e, i) => m.set(e.id, i % 2 === 0 ? 0 : 11));
		}
		return m;
	});

	const deptList = $derived(
		orderGraphDepartments(
			graph.nodes.filter((n) => n.kind === 'team'),
			(t) => t.id.replace('team:', '')
		).map((t) => {
			const deptId = t.id.replace('team:', '');
			return { teamId: t.id, deptId, name: t.label, color: t.color ? themedNodeColor(t.color) : 'var(--bn-brain-1)', tagline: departments.find((d) => d.id === deptId)?.tagline ?? '' };
		})
	);
	const currentDept = $derived(deptList.find((d) => d.teamId === focusTeamId) ?? null);
	const flankTeams = $derived.by(() => {
		if (!focusTeamId) return null;
		const order = deptList.map((d) => d.teamId);
		const fi = order.indexOf(focusTeamId);
		if (fi < 0 || order.length < 2) return null;
		return new Set([order[(fi + 1) % order.length], order[(fi - 1 + order.length) % order.length]]);
	});

	// ── the wheel turn + physics (plain values the d3 accessors read live) ──
	let wheel = 0;
	let wheelTarget = 0;
	let stagePhase = 0;
	let stageTarget = 0;
	let stageVel = 0;
	let settleBoost = 0;
	let rimGuideEl = $state<SVGGElement | null>(null);
	let homeTween: { t0: number; from: Map<string, Pt> } | null = null;
	const HOME_TWEEN_MS = 820;
	const spreadK = () => Math.max(0.7, Math.min(1.18, 0.7 + (60 - 10) / 180));
	function homeSpotOf(d: SimNode): Pt | null {
		const r = restLayout.positions.get(d.id);
		if (!r) return null;
		const p = rotateAbout(r, { x: CX, y: CY }, wheel);
		const k = spreadK();
		return { x: (p.x - CX) * k + CX, y: (p.y - CY) * k + CY };
	}

	/* eslint-disable @typescript-eslint/no-explicit-any */
	function configure(s: Simulation<SimNode, undefined>) {
		const lay = () => focusTree?.positions ?? null;
		const focused = () => !!lay();
		const tgt = (d: SimNode) => lay()?.get(d.id) ?? null;
		const rest = (d: SimNode) => restLayout.positions.get(d.id) ?? null;
		const restStrength = () => 0.32;
		const pushK = (d: SimNode) => (coreExpanded && d.kind === 'team' ? TEAM_PUSH_EXPANDED : 1);
		const spun = (d: SimNode) => {
			const r = rest(d);
			if (!r) return null;
			if (focused()) return wheelPoint(r, { x: CX, y: CY }, FOCUS_WHEEL, wheel);
			return rotateAbout(r, { x: CX, y: CY }, wheel);
		};
		const rimOffset = (teamId: string) => {
			const order = deptList.map((d) => d.teamId);
			const ti = order.indexOf(teamId);
			if (ti < 0 || order.length === 0) return null;
			const n = order.length;
			const phase = ((stagePhase % n) + n) % n;
			return cyclicDeltaF(phase, ti, n);
		};
		const rim = (d: SimNode) => {
			if (!focused()) return null;
			const teamId = d.kind === 'team' ? d.id : teamForFocus(d.id);
			if (!teamId) return null;
			const o = rimOffset(teamId);
			if (o === null) return null;
			const tree = allTrees.get(teamId);
			const home = tree?.positions.get(d.id) ?? tree?.positions.get(teamId) ?? wheelStageSpot(0, W, H);
			return rotateAbout(home, WHEEL_GEOM.hub, o * WHEEL_GEOM.delta);
		};
		const restX = (d: SimNode) => {
			const r = spun(d);
			return r ? (focused() ? r.x : (r.x - CX) * spreadK() * pushK(d) + CX) : CX;
		};
		const restY = (d: SimNode) => {
			const r = spun(d);
			return r ? (focused() ? r.y : (r.y - CY) * spreadK() * pushK(d) + CY) : CY;
		};
		(s.force('charge') as any).strength((d: SimNode) => (tgt(d) ? -34 : focused() ? 0 : -150 * 0.3));
		(s.force('link') as any).distance(() => 60).strength(focused() ? () => 0 : () => 0.06);
		const targetOf = (d: SimNode) => rim(d) ?? tgt(d);
		const staged = (d: SimNode) => focused() && !!(rim(d) ?? tgt(d));
		(s.force('x') as any).x((d: SimNode) => restX(d)).strength((d: SimNode) => (staged(d) ? 0 : Math.max(restStrength(), settleBoost)));
		(s.force('y') as any).y((d: SimNode) => restY(d)).strength((d: SimNode) => (staged(d) ? 0 : Math.max(restStrength(), settleBoost)));
		{
			let stageNodes: SimNode[] = [];
			const stageForce = ((alpha: number) => {
				if (!focused()) return;
				for (const d of stageNodes) {
					const t = targetOf(d);
					if (!t) continue;
					const k = (tgt(d) ? 0.9 : 0.65) * alpha;
					d.vx = (d.vx ?? 0) + (t.x - (d.x ?? 0)) * k;
					d.vy = (d.vy ?? 0) + (t.y - (d.y ?? 0)) * k;
				}
			}) as any;
			stageForce.initialize = (ns: SimNode[]) => {
				stageNodes = ns;
			};
			s.force('stage', stageForce);
		}
		(s.force('radial') as any).strength((d: SimNode) => (tgt(d) || rest(d) ? 0 : 0.4));
		(s.force('collide') as any).radius((d: SimNode) => (tgt(d) ? CAT[d.kind].r + 4 : focused() ? 0.5 : CAT[d.kind].r + 3));
	}
	/* eslint-enable @typescript-eslint/no-explicit-any */

	// one paint per animation frame while d3 ticks freely underneath
	let paintQueued = false;
	function paint() {
		if (paintQueued) return;
		paintQueued = true;
		const go = () => {
			paintQueued = false;
			pos = new Map(nodes.map((n) => [n.id, { x: n.x, y: n.y }]));
		};
		if (typeof requestAnimationFrame === 'function') requestAnimationFrame(go);
		else setTimeout(go, 16);
	}

	// build the simulation when the graph changes, seeded on the clean sunburst
	$effect(() => {
		const g = graph;
		return untrack(() => {
			const rest = restLayout.positions;
			const ns: SimNode[] = g.nodes.map((n, i) => {
				const r = rest.get(n.id);
				if (r) return { ...n, x: r.x, y: r.y };
				const peers = g.nodes.filter((m) => m.ring === n.ring).length || 1;
				const a = (i / peers) * Math.PI * 2;
				return { ...n, x: CX + Math.cos(a) * (RING_R[n.ring] || 1), y: CY + Math.sin(a) * (RING_R[n.ring] || 1) };
			});
			links = g.edges.map((e) => ({ source: e.source, target: e.target, kind: e.kind }));
			nodes = ns;
			pos = new Map(ns.map((n) => [n.id, { x: n.x, y: n.y }]));
			const s = forceSimulation(ns)
				.velocityDecay(0.62)
				.alphaDecay(0.015)
				.force('link', forceLink<SimNode, SimLink & { source: SimNode | string; target: SimNode | string }>(links as never).id((d) => d.id))
				.force('charge', forceManyBody())
				.force('radial', forceRadial<SimNode>((d) => RING_R[d.ring], CX, CY))
				.force('x', forceX<SimNode>(CX))
				.force('y', forceY<SimNode>(CY))
				.force('collide', forceCollide<SimNode>(10))
				.on('tick', paint);
			configure(s);
			sim = s;
			return () => {
				s.stop();
				if (sim === s) sim = null;
			};
		});
	});

	// turn the wheel so the focused pillar faces the stage
	let prevFocusTeam: string | null = null;
	$effect(() => {
		const ft = focusTeamId;
		untrack(() => {
			if (!ft) {
				prevFocusTeam = null;
				return;
			}
			const order = deptList.map((d) => d.teamId);
			const idx = order.indexOf(ft);
			if (idx >= 0 && order.length > 0) {
				if (!prevFocusTeam) {
					stagePhase = idx;
					stageTarget = idx;
				} else {
					const n = order.length;
					const phaseMod = ((stageTarget % n) + n) % n;
					stageTarget += cyclicDeltaF(phaseMod, idx, n);
				}
			}
			prevFocusTeam = ft;
			const home = restLayout.positions.get(ft);
			if (!home) return;
			const homeAngle = Math.atan2(home.y - CY, home.x - CX);
			wheelTarget += shortestAngleDelta(homeAngle + wheelTarget, FOCUS_WHEEL.stage);
			sim?.alpha(0.16).restart();
		});
	});

	// reconfigure + soft reheat on focus / vault changes (the vault defers it until the zoom lands)
	let prevExpanded = false;
	$effect(() => {
		void focusId;
		const open = coreExpanded;
		return untrack(() => {
			const s = sim;
			if (!s) return;
			configure(s);
			const justOpened = open && !prevExpanded;
			prevExpanded = open;
			if (justOpened) {
				const t = setTimeout(() => sim?.alpha(0.16).restart(), 950);
				return () => clearTimeout(t);
			}
			s.alpha(0.16).restart();
		});
	});

	// the vault closing drops any stale note hover and its search
	$effect(() => {
		if (!coreExpanded) {
			untrack(() => {
				memHoverId = null;
				memQuery = '';
			});
		}
	});

	// pause the core's animations while it opens/closes, rasterize fast while the camera flies
	let coreGEl = $state<SVGGElement | null>(null);
	let firstExpand = true;
	$effect(() => {
		void coreExpanded;
		return untrack(() => {
			if (firstExpand) {
				firstExpand = false;
				return;
			}
			const el = coreGEl;
			const svg = svgEl;
			if (!el) return;
			el.classList.add('kg-transitioning');
			svg?.classList.add('kg-fast-raster');
			const t = setTimeout(() => {
				el.classList.remove('kg-transitioning');
				svg?.classList.remove('kg-fast-raster');
			}, 1250);
			return () => {
				clearTimeout(t);
				el.classList.remove('kg-transitioning');
				svg?.classList.remove('kg-fast-raster');
			};
		});
	});

	// ── the constellation (local coords about the self anchor) ──
	const coreScale = $derived(coreExpanded ? CORE_SCALE_EXPANDED : focusTree ? CORE_SCALE_TREE : 1);
	const memLayout = $derived.by(() => {
		const m = new Map<string, Pt>();
		for (const n of memory?.nodes ?? []) m.set(n.id, memoryNodePos(n, { x: 0, y: 0 }, R_CORE));
		return m;
	});
	const memById = $derived(new Map((memory?.nodes ?? []).map((n) => [n.id, n])));
	const memHits = $derived(coreExpanded && memory ? searchMemoryNotes(memory.nodes, memQuery) : []);
	const memVisible = $derived(memoryOn && memory ? (coreExpanded ? memory.nodes : pickRestTier(memory.nodes)) : []);
	const memVisibleIds = $derived(new Set(memVisible.map((n) => n.id)));
	const memRestIds = $derived(coreExpanded && memory ? new Set(pickRestTier(memory.nodes).map((n) => n.id)) : memVisibleIds);
	const memLayers = $derived.by(() => {
		const layers: MemoryNode[][] = [[], [], []];
		for (const m of memVisible) layers[memLayerOf(m.id)].push(m);
		return layers;
	});
	const memEdges = $derived(memoryOn && memory ? memory.edges.filter((e) => memVisibleIds.has(e.source) && memVisibleIds.has(e.target) && memLayout.has(e.source) && memLayout.has(e.target)) : []);
	let sparkSegs: [number, number, number, number][] = [];
	$effect(() => {
		sparkSegs = memEdges.map((e) => {
			const s = memLayout.get(e.source)!;
			const t = memLayout.get(e.target)!;
			return [s.x, s.y, t.x, t.y] as [number, number, number, number];
		});
	});
	const memAdj = $derived.by(() => {
		const m = new Map<string, string[]>();
		const add = (a: string, b: string) => (m.get(a) ?? m.set(a, []).get(a)!).push(b);
		for (const e of memory?.edges ?? []) {
			add(e.source, e.target);
			add(e.target, e.source);
		}
		return m;
	});
	const memRotDeg = () => (memRot * 180) / Math.PI;
	let memRot = 0;
	let memRotGEl = $state<SVGGElement | null>(null);
	let memOverlayGEl = $state<SVGGElement | null>(null);
	let synapseRotGEl = $state<SVGGElement | null>(null);
	const sparkEls: (SVGCircleElement | null)[] = $state([]);
	const commEls: Record<string, SVGCircleElement | null> = $state({});
	const commTeams = $derived(
		orderGraphDepartments(
			graph.nodes.filter((n) => n.kind === 'team'),
			(t) => t.id.replace('team:', '')
		).map((t) => ({ id: t.id, color: t.color ? themedNodeColor(t.color) : 'var(--bn-brain-1)' }))
	);

	// ── lenses + pillar chips ──
	const lensLit = $derived(lensId ? lensNodeSet(lensId, { nodes: graph.nodes, teamOf: (id) => teamForFocus(id) }) : null);
	const pillarLit = $derived.by(() => {
		if (!activePillars || activePillars.length === departments.length) return null;
		const on = new Set(activePillars.map((id) => `team:${id}`));
		const set = new Set<string>();
		for (const n of graph.nodes) {
			const team = teamForFocus(n.id);
			if (team && on.has(team)) set.add(n.id);
		}
		if (set.size > 0) set.add(SELF_ID);
		return set;
	});
	const bothLit = $derived(lensLit && pillarLit ? new Set([...lensLit].filter((id) => pillarLit.has(id))) : lensLit ?? pillarLit);
	const lit = $derived(focusSet ?? (hoverId ? litFor(hoverId) : null) ?? bothLit);
	const focusedTeam = $derived(focusTeamId ? byId.get(focusTeamId) ?? null : null);

	// exit crossfade: a closing tree keeps its skeleton ~280ms while riding the nodes home
	let exitTree = $state<TreeLayoutResult | null>(null);
	let prevTree: TreeLayoutResult | null = null;
	$effect(() => {
		const ft = focusTree;
		return untrack(() => {
			if (ft) {
				prevTree = ft;
				exitTree = null;
				return;
			}
			if (prevTree) {
				exitTree = prevTree;
				prevTree = null;
				const t = setTimeout(() => (exitTree = null), 280);
				return () => clearTimeout(t);
			}
		});
	});

	// ── the camera rAF: wheel ease, go-home glide, stage phase, orbit, sparks, pulses, viewBox ──
	let lastActive = typeof performance !== 'undefined' ? performance.now() : 0;
	const selectedOrgId = $derived(selectedAgentId ?? selectedHumanId ?? selectedHeadId ?? selectedBoardId ?? selectedTaskId ?? selectedToolId);
	onMount(() => {
		const reduced = typeof window !== 'undefined' && !!window.matchMedia?.('(prefers-reduced-motion: reduce)').matches;
		let cur: Rect = { x: 0, y: 0, w: W, h: H };
		let raf = 0;
		let lastT = performance.now();
		let lastRotDeg = NaN;
		let frame = 0;
		const ORBIT_S = 150;
		const IDLE_MS = 15_000;
		const posOf = (id: string | null): Pt | null => {
			if (!id) return null;
			const n = nodes.find((m) => m.id === id);
			return n ? { x: n.x, y: n.y } : null;
		};
		const step = () => {
			const c = { focusTree: !!focusTree, selectedOrgId, coreExpanded, selectedMemoryId, coreScale };
			const nowT = performance.now();
			const wd = shortestAngleDelta(wheel, wheelTarget);
			if (Math.abs(wd) > 0.0005) wheel += wd * 0.075;
			const tween = homeTween;
			if (tween && (c.focusTree || c.coreExpanded || c.selectedOrgId || c.selectedMemoryId)) {
				for (const n of nodes) {
					n.fx = null;
					n.fy = null;
				}
				homeTween = null;
			} else if (tween) {
				const k = Math.min(1, (nowT - tween.t0) / HOME_TWEEN_MS);
				const e = 1 - Math.pow(1 - k, 3);
				for (const n of nodes) {
					const to = homeSpotOf(n);
					const from = tween.from.get(n.id);
					if (!to || !from) continue;
					n.fx = n.x = from.x + (to.x - from.x) * e;
					n.fy = n.y = from.y + (to.y - from.y) * e;
					n.vx = 0;
					n.vy = 0;
				}
				if (k >= 1) {
					for (const n of nodes) {
						n.fx = null;
						n.fy = null;
					}
					homeTween = null;
					sim?.alpha(0.05).restart();
				}
			}
			const sd = stageTarget - stagePhase;
			stageVel += (sd * 0.075 - stageVel) * 0.09;
			if (Math.abs(sd) > 0.0008 || Math.abs(stageVel) > 0.0004) {
				stagePhase += stageVel;
				if (sim && Math.abs(sd) > 0.02 && sim.alpha() < 0.05) sim.alpha(0.1).restart();
				if (c.focusTree) rimGuideEl?.setAttribute('transform', `rotate(${(-stagePhase * RIM_DELTA_DEG).toFixed(3)} ${FOCUS_WHEEL.hub.x} ${FOCUS_WHEEL.hub.y})`);
			}
			frame = (frame + 1) % 3;
			const idle = nowT - lastActive > IDLE_MS && !c.focusTree && !c.coreExpanded && !c.selectedOrgId && !c.selectedMemoryId;
			if (idle && frame !== 0) {
				raf = requestAnimationFrame(step);
				return;
			}
			const dt = Math.min(0.1, (nowT - lastT) / 1000);
			lastT = nowT;
			if (!reduced && !c.coreExpanded) memRot = (memRot + (2 * Math.PI * dt) / ORBIT_S) % (2 * Math.PI);
			const rotDeg = memRotDeg();
			if (rotDeg !== lastRotDeg) {
				lastRotDeg = rotDeg;
				for (const g of [memRotGEl, memOverlayGEl]) {
					if (!g) continue;
					g.setAttribute('transform', `rotate(${rotDeg})`);
					for (const el of g.getElementsByClassName('kg-mem-upright')) el.setAttribute('transform', `rotate(${-rotDeg})`);
				}
				synapseRotGEl?.setAttribute('transform', `rotate(${rotDeg})`);
			}
			if (!reduced && sparkSegs.length) {
				for (let i = 0; i < SYNAPSE_N; i++) {
					const el = sparkEls[i];
					if (!el) continue;
					const period = 2400 + ((i * 379) % 1700);
					const t = nowT + i * 911;
					const cycle = Math.floor(t / period);
					const seg = sparkSegs[(cycle * 131 + i * 37) % sparkSegs.length];
					const u = (t % period) / period;
					el.setAttribute('cx', String(seg[0] + (seg[2] - seg[0]) * u));
					el.setAttribute('cy', String(seg[1] + (seg[3] - seg[1]) * u));
					el.setAttribute('opacity', String(0.95 * Math.sin(Math.PI * u)));
				}
			}
			const coreCenter = posOf(SELF_ID) ?? { x: CX, y: CY };
			const proj = c.selectedMemoryId ? memById.get(c.selectedMemoryId) : null;
			const th = memRot;
			const rProj = proj ? { vx: proj.vx * Math.cos(th) - proj.vy * Math.sin(th), vy: proj.vx * Math.sin(th) + proj.vy * Math.cos(th) } : null;
			const target =
				userView ??
				cameraRect(
					{ w: W, h: H },
					{
						focusedTeam: c.focusTree,
						coreExpanded: c.coreExpanded,
						coreCenter,
						selectedNodePos: posOf(c.selectedOrgId),
						memorySelectedPos: rProj ? memoryNodePos(rProj, coreCenter, R_CORE * c.coreScale) : null
					}
				);
			const goingHome = !userView && !c.focusTree && !c.coreExpanded && !c.selectedOrgId && !c.selectedMemoryId;
			const next = lerpRect(cur, target, reduced ? 1 : goingHome ? CAM_EASE_HOME : CAM_EASE);
			if (next !== cur) {
				cur = next;
				if (svgEl) {
					svgEl.setAttribute('viewBox', `${cur.x} ${cur.y} ${cur.w} ${cur.h}`);
					svgEl.style.setProperty('--kg-cam-k', String(cur.w / W));
				}
			}
			if (!reduced) {
				const now = performance.now();
				const selfPos = posOf(SELF_ID);
				for (const [key, el] of Object.entries(commEls)) {
					if (!el) continue;
					const sep = key.lastIndexOf(':');
					const teamPos = posOf(key.slice(0, sep));
					const dir = key.slice(sep + 1);
					if (!selfPos || !teamPos) continue;
					const dx = teamPos.x - selfPos.x;
					const dy = teamPos.y - selfPos.y;
					const len = Math.hypot(dx, dy) || 1;
					const mx = (selfPos.x + teamPos.x) / 2 + (-dy / len) * 0.12 * len;
					const my = (selfPos.y + teamPos.y) / 2 + (dx / len) * 0.12 * len;
					const seed = (hashStr(key.slice(0, sep)) % 100) / 100;
					const u = dir === 'out' ? (now / 2600 + seed) % 1 : 1 - ((now / 3300 + seed * 1.7) % 1);
					const a = 1 - u;
					el.setAttribute('transform', `translate(${a * a * selfPos.x + 2 * a * u * mx + u * u * teamPos.x},${a * a * selfPos.y + 2 * a * u * my + u * u * teamPos.y})`);
					el.setAttribute('opacity', String(0.9 * Math.sin(Math.PI * u)));
				}
			}
			raf = requestAnimationFrame(step);
		};
		raf = requestAnimationFrame(step);
		const wake = () => (lastActive = performance.now());
		const WAKE = ['pointermove', 'pointerdown', 'keydown', 'wheel'] as const;
		for (const ev of WAKE) window.addEventListener(ev, wake, { passive: true });
		return () => {
			cancelAnimationFrame(raf);
			for (const ev of WAKE) window.removeEventListener(ev, wake);
		};
	});

	// ── keys: Escape walks back out (card → focus → home), ← / → step pillars, / searches the vault ──
	const hasDetail = $derived(!!(selectedAgentId || selectedToolId || selectedTaskId || selectedHumanId || selectedMemoryId));
	const typing = (e: KeyboardEvent) => {
		const t = e.target as HTMLElement | null;
		return t?.tagName === 'INPUT' || t?.tagName === 'TEXTAREA' || !!t?.isContentEditable;
	};
	function onWindowKey(e: KeyboardEvent) {
		if (e.key === '/' && !typing(e) && coreExpanded) {
			e.preventDefault();
			memSearchEl?.focus();
			return;
		}
		if (fullscreen || typing(e)) return;
		if (e.key === 'Escape' && (hasDetail || !!focusId || coreExpanded)) {
			if (hasDetail) clearDetail();
			else clearAll();
			return;
		}
		if ((e.key === 'ArrowLeft' || e.key === 'ArrowRight') && focusTree) stepDept(e.key === 'ArrowLeft' ? -1 : 1);
	}

	// ── selection / panel data ──
	const selectedAgent = $derived(selectedAgentId ? agentById.get(selectedAgentId) ?? null : null);
	const selectedAgentTaskId = $derived(selectedAgentId ? rel.taskOfWorker.get(selectedAgentId) ?? null : null);
	const selectedAgentTask = $derived(selectedAgentTaskId ? taskById.get(selectedAgentTaskId) ?? null : null);
	const selectedAgentParent = $derived(selectedAgent?.parentId ? agents.find((a) => a.id === selectedAgent.parentId) ?? null : null);
	const selectedAgentSubs = $derived(selectedAgent ? agents.filter((a) => a.parentId === selectedAgent.id).map((a) => ({ id: a.id, name: a.name })) : []);
	const selectedAgentRun = $derived(selectedAgent ? runsByAgent[selectedAgent.id] ?? null : null);
	const agentHeadName = $derived.by(() => {
		if (!selectedAgentId) return null;
		const team = rel.teamOfWorker.get(selectedAgentId);
		return team ? headForDepartment(team.replace('team:', ''))?.name ?? null : null;
	});
	const toolWiki = $derived(
		selectedToolId
			? buildToolWiki(
					toolSlugOf(selectedToolId),
					(rel.workersOfTool.get(selectedToolId) ?? []).map((w) => agentById.get(w)?.name ?? personById.get(w)?.name ?? byId.get(w)?.label ?? w)
				)
			: null
	);
	const toolChips = (workerId: string | null) =>
		(workerId ? rel.toolsOfWorker.get(workerId) ?? [] : []).map((t) => {
			const slug = toolSlugOf(t);
			return { slug, name: prettifySlug(slug), mcp: buildToolWiki(slug).mcp };
		});
	const selectedTask = $derived(selectedTaskId ? taskById.get(selectedTaskId) ?? null : null);
	const selectedTaskWorker = $derived(selectedTaskId ? rel.workerOfTask.get(selectedTaskId) ?? null : null);
	const selectedTaskWorkerNode = $derived(selectedTaskWorker ? byId.get(selectedTaskWorker) ?? null : null);
	const selectedHuman = $derived(selectedHumanId ? personById.get(selectedHumanId) ?? null : null);
	const selectedHumanTaskId = $derived(selectedHumanId ? rel.taskOfWorker.get(selectedHumanId) ?? null : null);
	const selectedHumanTask = $derived(selectedHumanTaskId ? taskById.get(selectedHumanTaskId) ?? null : null);
	const selectedMemory = $derived(selectedMemoryId ? memory?.nodes.find((n) => n.id === selectedMemoryId) ?? null : null);
	const headTeam = $derived(selectedHeadId ? byId.get(selectedHeadId) ?? null : null);
	const headSops = $derived(
		selectedHeadId
			? (rel.tasksOfTeam.get(selectedHeadId) ?? [])
					.map((tid) => taskById.get(tid))
					.filter((t) => !!t)
					.map((t) => ({ id: `task:${t!.id}`, title: t!.title, skillName: playbookFor(t!).skill.name }))
			: []
	);
	const boardNode = $derived(selectedBoardId ? byId.get(selectedBoardId) ?? null : null);
	const boardLive = $derived(selectedBoardId ? boardAgents.find((a) => `board:${a.id}` === selectedBoardId) ?? null : null);
	// which card holds the slot, in production's precedence
	const detailKind = $derived(
		toolWiki ? 'tool' : selectedAgent ? 'agent' : headTeam ? 'head' : boardNode ? 'board' : selectedTask ? 'task' : selectedHuman ? 'human' : selectedMemory ? 'memory' : coreExpanded ? 'core' : null
	);
	const detailOpen = $derived(detailKind !== null);

	$effect(() => {
		onPanel?.(!!focusSet || coreExpanded || (detailOpen && !detailExpanded));
	});

	// ── interaction helpers ──
	function clearDetail() {
		selectedAgentId = null;
		selectedToolId = null;
		selectedTaskId = null;
		selectedHumanId = null;
		selectedHeadId = null;
		selectedBoardId = null;
		selectedMemoryId = null;
		detailExpanded = false;
	}
	let settleTimer: ReturnType<typeof setTimeout> | null = null;
	function clearAll() {
		userView = null;
		focusId = null;
		coreExpanded = false;
		clearDetail();
		wheelTarget = wheel;
		for (const n of nodes) {
			n.fx = null;
			n.fy = null;
		}
		settleBoost = 0.32;
		if (settleTimer) clearTimeout(settleTimer);
		settleTimer = setTimeout(() => {
			settleBoost = 0;
			settleTimer = null;
		}, 1100);
		const reducedMotion = typeof window !== 'undefined' && !!window.matchMedia?.('(prefers-reduced-motion: reduce)').matches;
		if (reducedMotion) {
			for (const n of nodes) {
				const h = homeSpotOf(n);
				if (!h) continue;
				n.x = h.x;
				n.y = h.y;
				n.vx = 0;
				n.vy = 0;
			}
			homeTween = null;
			sim?.alpha(0.02).restart();
			return;
		}
		homeTween = { t0: performance.now(), from: new Map(nodes.map((n) => [n.id, { x: n.x ?? CX, y: n.y ?? CY }])) };
		sim?.alpha(0.35).restart();
	}
	function navDept(teamId: string) {
		focusId = teamId;
		clearDetail();
	}
	function stepDept(dir: number) {
		const ids = deptList.map((d) => d.teamId);
		if (ids.length === 0) return;
		const i = ids.indexOf(focusTeamId ?? '');
		const next = i < 0 ? (dir > 0 ? 0 : ids.length - 1) : (i + dir + ids.length) % ids.length;
		navDept(ids[next]);
	}
	function selectAgent(id: string) {
		focusId = teamForFocus(id) ?? id;
		clearDetail();
		selectedAgentId = id;
	}
	function selectHuman(id: string) {
		focusId = teamForFocus(id) ?? id;
		clearDetail();
		selectedHumanId = id;
	}
	const selectWorker = (id: string) => (personById.has(id) ? selectHuman : selectAgent)(id);
	function selectTask(id: string) {
		focusId = teamForFocus(id) ?? focusId;
		clearDetail();
		selectedTaskId = id;
	}
	function selectTool(id: string) {
		focusId = teamForFocus(id) ?? focusId;
		clearDetail();
		selectedToolId = id;
	}
	function selectToolSlug(slug: string) {
		const dept = focusTeamId?.replace('team:', '');
		const local = dept ? `tool:${slug}@${dept}` : null;
		const target = (local && byId.has(local) && local) || (byId.has(`tool:${slug}`) && `tool:${slug}`) || graph.nodes.find((n) => n.kind === 'tool' && toolSlugOf(n.id) === slug)?.id;
		if (target) selectTool(target);
	}

	// the everything-index, filtered by the pillar chips
	const directory = $derived.by(() => {
		const groups = page.directory;
		if (!activePillars || activePillars.length === departments.length) return groups;
		const on = new Set(activePillars);
		return groups.map((g) => ({ ...g, rows: g.rows.filter((r) => r.deptIds.some((d) => on.has(d))) }));
	});
	function pickFromDirectory(kind: DirectoryGroup['kind'], id: string) {
		userView = null;
		if (kind === 'tool') selectToolSlug(id);
		else if (kind === 'task') selectTask(id);
		else selectWorker(id);
	}
	function hoverFromDirectory(kind: DirectoryGroup['kind'], id: string | null) {
		if (!id) {
			hoverId = null;
			return;
		}
		hoverId = kind === 'tool' ? graph.nodes.find((n) => n.kind === 'tool' && toolSlugOf(n.id) === id)?.id ?? null : id;
	}

	// the sticky node hover card names whatever the pointer last touched
	$effect(() => {
		if (hoverId) cardId = hoverId;
	});
	const cardNode = $derived(cardId ? byId.get(cardId) ?? null : null);
	function openCardNode() {
		const n = cardNode;
		if (!n) return;
		if (n.kind === 'tool') selectToolSlug(toolSlugOf(n.id));
		else if (n.kind === 'task') selectTask(n.id);
		else if (isWorker(n.kind)) selectWorker(n.id);
		else if (n.kind === 'team') navDept(n.id);
	}

	function onNodeClick(n: KGNode) {
		userView = null;
		if (n.kind === 'self') {
			if (memoryOn) {
				const entering = !coreExpanded;
				focusId = null;
				clearDetail();
				coreExpanded = entering;
			} else clearAll();
			return;
		}
		coreExpanded = false;
		selectedMemoryId = null;
		if (n.kind === 'employee') {
			const was = selectedAgentId === n.id;
			focusId = n.id;
			clearDetail();
			if (!was) selectedAgentId = n.id;
		} else if (n.kind === 'person') {
			const was = selectedHumanId === n.id;
			focusId = n.id;
			clearDetail();
			if (!was) selectedHumanId = n.id;
		} else if (n.kind === 'task') {
			const was = selectedTaskId === n.id;
			focusId = teamForFocus(n.id) ?? focusId;
			clearDetail();
			if (!was) selectedTaskId = n.id;
		} else if (n.kind === 'tool') {
			const was = selectedToolId === n.id;
			focusId = teamForFocus(n.id) ?? focusId;
			clearDetail();
			if (!was) selectedToolId = n.id;
		} else if (n.kind === 'board') {
			const was = selectedBoardId === n.id;
			focusId = null;
			clearDetail();
			if (!was) selectedBoardId = n.id;
		} else {
			// the pillar IS the department head: expand it AND pull up its exec card
			const entering = focusId !== n.id;
			clearDetail();
			focusId = entering ? n.id : null;
			if (n.kind === 'team' && entering) selectedHeadId = n.id;
		}
	}

	// ── node dragging + the manual camera (wheel zoom, canvas pan) ──
	const simNode = (id: string) => nodes.find((n) => n.id === id) ?? null;
	function toSvgPoint(clientX: number, clientY: number): Pt | null {
		const ctm = svgEl?.getScreenCTM?.();
		if (!svgEl || !ctm) return null;
		const pt = new DOMPoint(clientX, clientY).matrixTransform(ctm.inverse());
		return { x: pt.x, y: pt.y };
	}
	$effect(() => {
		const svg = svgEl;
		if (!svg) return;
		const onWheel = (e: WheelEvent) => {
			e.preventDefault();
			const vb = svg.viewBox.baseVal;
			const ctm = svg.getScreenCTM?.();
			const p = ctm ? new DOMPoint(e.clientX, e.clientY).matrixTransform(ctm.inverse()) : { x: CX, y: CY };
			const f = Math.min(2, Math.max(0.5, Math.exp(e.deltaY * 0.0012)));
			const w = Math.min(W * 3, Math.max(W * 0.12, vb.width * f));
			const k = w / vb.width;
			userView = { x: p.x - (p.x - vb.x) * k, y: p.y - (p.y - vb.y) * k, w, h: vb.height * k };
		};
		svg.addEventListener('wheel', onWheel, { passive: false });
		return () => svg.removeEventListener('wheel', onWheel);
	});
	function onCanvasPointerDown(e: PointerEvent) {
		if (!svgEl || e.button !== 0) return;
		const vb = svgEl.viewBox.baseVal;
		const ctm = svgEl.getScreenCTM?.();
		pan = { px: e.clientX, py: e.clientY, x: vb.x, y: vb.y, k: ctm?.a ? 1 / ctm.a : 1, moved: false };
		try {
			(e.currentTarget as Element).setPointerCapture?.(e.pointerId);
		} catch {
			/* best-effort */
		}
	}
	function onCanvasPointerMove(e: PointerEvent) {
		const p = pan;
		if (!p || !svgEl) return;
		const dx = e.clientX - p.px;
		const dy = e.clientY - p.py;
		if (!p.moved && Math.hypot(dx, dy) < 3) return;
		p.moved = true;
		const vb = svgEl.viewBox.baseVal;
		userView = { x: p.x - dx * p.k, y: p.y - dy * p.k, w: vb.width, h: vb.height };
	}
	function onCanvasPointerUp() {
		if (pan?.moved) panSuppress = true;
		pan = null;
	}
	function onCanvasClick() {
		if (panSuppress) {
			panSuppress = false;
			return;
		}
		clearAll();
	}
	function onNodePointerDown(e: PointerEvent, id: string) {
		e.stopPropagation();
		try {
			(e.currentTarget as Element).setPointerCapture?.(e.pointerId);
		} catch {
			/* best-effort */
		}
		drag = { id, moved: false, startX: e.clientX, startY: e.clientY };
		const node = simNode(id);
		if (node) {
			node.fx = node.x;
			node.fy = node.y;
		}
		sim?.alphaTarget(0.2).restart();
	}
	function onNodePointerMove(e: PointerEvent, id: string) {
		const d = drag;
		if (!d || d.id !== id) return;
		if (!d.moved && Math.hypot(e.clientX - d.startX, e.clientY - d.startY) > 3) d.moved = true;
		const p = toSvgPoint(e.clientX, e.clientY);
		const node = simNode(id);
		if (p && node) {
			node.fx = p.x;
			node.fy = p.y;
		}
	}
	function onNodePointerUp(e: PointerEvent, id: string) {
		const d = drag;
		if (!d || d.id !== id) return;
		try {
			(e.currentTarget as Element).releasePointerCapture?.(e.pointerId);
		} catch {
			/* best-effort */
		}
		const node = simNode(id);
		if (node) {
			node.fx = null;
			node.fy = null;
		}
		sim?.alphaTarget(0).alpha(0.14).restart();
		if (d.moved) suppressClick = true;
		drag = null;
	}
	function nodeClick(e: MouseEvent, n: KGNode) {
		e.stopPropagation();
		if (suppressClick) {
			suppressClick = false;
			return;
		}
		onNodeClick(n);
	}
	function memNoteClick(e: MouseEvent, id: string) {
		e.stopPropagation();
		if (suppressClick) {
			suppressClick = false;
			return;
		}
		const was = selectedMemoryId === id;
		clearDetail();
		selectedMemoryId = was ? null : id;
	}

	// per-node render props (KnowledgeGraph.tsx's nodes.map body)
	function nodeView(n: KGNode) {
		const cat = CAT[n.kind];
		const color = nodeColor(n);
		const dim = coreExpanded ? n.kind !== 'self' && n.kind !== 'team' : lit ? !lit.has(n.id) : false;
		const inFocus = focusSet?.has(n.id) ?? false;
		const selected = selectedAgentId === n.id || selectedToolId === n.id || selectedTaskId === n.id || selectedHumanId === n.id || selectedBoardId === n.id;
		const showLabel = n.kind === 'self' || n.kind === 'team' || n.kind === 'board' || inFocus || (hoverId ? lit?.has(n.id) ?? false : false);
		const degree = (rel.adjacency.get(n.id)?.size ?? 1) - 1;
		const r = cat.r + (isWorker(n.kind) || n.kind === 'tool' ? Math.min(2.5, degree * 0.3) : 0);
		const sectorTeam = n.kind === 'team' ? n.id : teamForFocus(n.id);
		const inFlankSector = !!(sectorTeam && flankTeams?.has(sectorTeam));
		const isFlank = !!flankTeams?.has(n.id);
		const hidden = !!(dim && !coreExpanded && focusSet && !inFlankSector);
		const opacity = dim ? (coreExpanded ? 0.06 : focusSet ? (isFlank ? 0.6 : inFlankSector ? 0.2 : 0) : 0.15) : TIER_OPACITY[n.kind];
		return { color, dim, selected, showLabel, r, hidden, opacity };
	}
	function boardLabel(p: Pt, r: number) {
		const vx = p.x - CX;
		const vy = p.y - CY;
		const m = Math.hypot(vx, vy) || 1;
		const ux = vx / m;
		const uy = vy / m;
		const side = ux < -0.45 || ux > 0.85;
		return { x: ux * (r + 5) + (side ? Math.sign(ux) * 3 : 0), y: uy * (r + 5) + (side ? 3 : uy >= 0 ? 11 : -5), anchor: side ? (ux > 0 ? 'start' : 'end') : 'middle' };
	}
	function edgeView(l: SimLink, i: number) {
		const sId = typeof l.source === 'object' ? l.source.id : l.source;
		const tId = typeof l.target === 'object' ? l.target.id : l.target;
		const s = pos.get(sId);
		const t = pos.get(tId);
		if (!s || !t) return null;
		const arc = edgeArc(s, t);
		const pathway = coreExpanded && l.kind === 'pillar';
		if (coreExpanded && !pathway) return { arc, stroke: EDGE_COLOR[l.kind] ?? 'var(--bn-text-3)', width: 0.9, opacity: 0.02, ray: false, synapse: null as string | null, i };
		if (pathway) return { arc, stroke: byId.get(tId)?.color ?? 'var(--bn-text)', width: 2.2, opacity: 0.75, ray: true, synapse: null, i };
		const team = byId.get(teamForFocus(sId) ?? teamForFocus(tId) ?? '');
		const tint = team?.color ?? EDGE_COLOR[l.kind] ?? 'var(--bn-text-3)';
		const incident = hoverId !== null && (sId === hoverId || tId === hoverId);
		const onChain = !lit || (lit.has(sId) && lit.has(tId));
		return {
			arc,
			stroke: tint,
			width: incident ? 1.6 : onChain && lit ? 1.2 : 0.9,
			opacity: lit ? (incident ? 0.6 : onChain ? 0.35 : 0.05) : 0.14,
			ray: false,
			synapse: (l.kind === 'pillar' || l.kind === 'sop') && !lit ? (l.kind === 'pillar' ? 'kg-synapse' : 'kg-synapse-sm') : null,
			i
		};
	}
	const legendCounts = $derived(Object.fromEntries(LEGEND_KINDS.map((k) => [k, graph.nodes.filter((n) => n.kind === k).length])));
	const LENS_GROUPS: [string, Lens[]][] = [
		['Entity', ENTITY_LENSES],
		['Function', FUNCTION_LENSES],
		['Action', ACTION_LENSES]
	];
	const COMPACT_LEGEND: { label: string; color: string; kind: KGNodeKind }[] = [
		{ label: 'Engine', color: HUB_COLOR, kind: 'self' },
		{ label: 'Board agent', color: CAT.board.color, kind: 'board' },
		{ label: 'Human', color: CAT.person.color, kind: 'person' },
		{ label: 'AI agent', color: CAT.employee.color, kind: 'employee' },
		{ label: 'Tool', color: CAT.tool.color, kind: 'tool' },
		{ label: 'SOP task', color: CAT.task.color, kind: 'task' }
	];
</script>

<svelte:window onkeydown={onWindowKey} />

{#snippet vaultSearch()}
	{#if coreExpanded}
		<input
			bind:this={memSearchEl}
			bind:value={memQuery}
			onkeydown={(e) => {
				if (e.key === 'Escape' && memQuery) {
					e.stopPropagation();
					memQuery = '';
				}
			}}
			placeholder="find a note…  /"
			aria-label="Search the vault"
			title="Press / to search the vault"
			spellcheck="false"
			class="kg-bs kg-bg85 kg-blur bn-text kg-input kg-fade-state kg-focus-accent w-40 rounded-[5px] border px-2 py-1.5 font-mono text-[10.5px] outline-none"
		/>
	{/if}
{/snippet}

{#snippet compactLegend()}
	<div class="kg-bs kg-bg85 kg-blur flex items-center gap-3 rounded-[5px] border px-2.5 py-1.5" data-part="compact-legend">
		{#each COMPACT_LEGEND as l (l.label)}
			<span class="bn-muted flex items-center gap-1.5 font-mono text-[9.5px]">
				<span style="color: {l.color}" class="inline-flex"><KindIcon kind={l.kind} size={12} /></span>
				{l.label}
			</span>
		{/each}
	</div>
{/snippet}

{#snippet directoryPanel()}
	<GraphDirectory
		groups={directory}
		onPick={pickFromDirectory}
		onHover={hoverFromDirectory}
		{onShowAllPillars}
		collapsed={directoryCollapsed}
		onToggleCollapse={() => (directoryCollapsed = !directoryCollapsed)}
		class="h-full"
	/>
{/snippet}

{#snippet detailBody()}
	{#if detailKind === 'tool' && toolWiki}
		<ToolDetailCard wiki={toolWiki} onClose={() => (selectedToolId = null)} />
	{:else if detailKind === 'agent' && selectedAgent}
		<AgentHarnessCard
			agent={selectedAgent}
			task={selectedAgentTask}
			parentName={selectedAgentParent?.name ?? null}
			parentAgentId={selectedAgentParent?.id ?? null}
			subAgents={selectedAgentSubs}
			lastRun={selectedAgentRun}
			runLabel={selectedAgentRun ? agoLabel(selectedAgentRun.finishedAt) : null}
			headName={agentHeadName}
			onClose={() => (selectedAgentId = null)}
			onTool={selectToolSlug}
			onAgent={(id) => selectAgent(`emp:${id}`)}
			onTask={selectedAgentTaskId ? () => selectTask(selectedAgentTaskId!) : undefined}
		/>
	{:else if detailKind === 'head' && headTeam}
		{@const deptId = headTeam.id.replace('team:', '')}
		<HeadDetailCard
			title={page.execTitles[deptId] ?? 'Lead'}
			deptName={headTeam.label}
			color={headTeam.color ?? 'var(--bn-brain-1)'}
			boardLead={boardLeads[deptId] ?? null}
			sops={headSops}
			onClose={() => (selectedHeadId = null)}
			onTask={(taskId) => selectTask(taskId)}
		/>
	{:else if detailKind === 'board' && boardNode}
		<HeadDetailCard
			title={boardNode.label}
			deptName="Paperclip board"
			roleLabel="board agent"
			showSops={false}
			color={CAT.board.color}
			boardLead={boardLive}
			sops={[]}
			onClose={() => (selectedBoardId = null)}
		>
			{#snippet blurb()}Live seat on the <span class="font-semibold">Paperclip board</span> — a real agent in the operator's company, reporting straight to the core.{/snippet}
		</HeadDetailCard>
	{:else if detailKind === 'task' && selectedTask}
		{@const a = selectedTask.assigneeKind === 'agent' ? agents.find((x) => x.id === selectedTask.assigneeId) : null}
		<SopTaskDetailCard
			task={selectedTask}
			assigneeName={selectedTaskWorkerNode?.label ?? selectedTask.assigneeId}
			assigneeKindLabel={selectedTask.assigneeKind === 'person' ? 'human employee' : 'AI agent'}
			assigneeColor={selectedTask.assigneeKind === 'person' ? 'var(--bn-warn)' : 'var(--bn-accent)'}
			runtime={selectedTask.assigneeKind === 'person' ? 'human · judgment call' : a ? `${a.instance} · ${a.model}` : null}
			tools={toolChips(selectedTaskWorker)}
			onClose={clearDetail}
			onAssignee={selectedTaskWorker ? () => selectWorker(selectedTaskWorker!) : undefined}
			onTool={selectToolSlug}
		/>
	{:else if detailKind === 'human' && selectedHuman}
		<GraphHumanDetailCard
			person={selectedHuman}
			deptName={byId.get(`team:${selectedHuman.departmentId}`)?.label ?? selectedHuman.departmentId}
			color="var(--bn-warn)"
			task={selectedHumanTask}
			tools={toolChips(selectedHumanId)}
			onClose={clearDetail}
			onTask={selectedHumanTaskId ? () => selectTask(selectedHumanTaskId!) : undefined}
			onTool={selectToolSlug}
		/>
	{:else if detailKind === 'memory' && selectedMemory}
		<MemoryNoteCard note={selectedMemory} color={memColor(selectedMemory)} onClose={() => (selectedMemoryId = null)} />
	{:else if detailKind === 'core'}
		<MemoryCoreCard {memory} color="var(--bn-accent)" onClose={() => (coreExpanded = false)} />
	{/if}
{/snippet}

{#snippet detailTopBar()}
	<div class="kg-b flex shrink-0 items-center border-b pr-1">
		<button
			type="button"
			onclick={clearDetail}
			aria-label={`Back to the ${focusedTeam?.label ?? 'directory'}`}
			class="bn-pressable bn-dim kg-h-text flex min-w-0 flex-1 items-center gap-1.5 px-3 py-2 text-left font-mono text-[10px] uppercase tracking-[0.14em]"
		>
			<span class="shrink-0"><ArrowLeft size={12} /></span>
			<span class="truncate">Back · <span style="color: {focusedTeam?.color ?? 'var(--bn-text)'}">{focusedTeam?.label ?? 'directory'}</span></span>
		</button>
		<button
			type="button"
			onclick={() => (detailExpanded = !detailExpanded)}
			aria-label={detailExpanded ? 'Collapse the detail card' : 'Expand the detail card'}
			title={detailExpanded ? 'Collapse' : 'Expand to a wider view'}
			class="bn-pressable bn-dim kg-h-accent flex h-7 w-7 shrink-0 items-center justify-center rounded-[5px]"
		>
			{#if detailExpanded}<Minimize2 size={14} />{:else}<Maximize2 size={14} />{/if}
		</button>
		<button type="button" onclick={clearDetail} aria-label="Close and go back to the directory" title="Close" class="bn-pressable bn-dim kg-h-err flex h-7 w-7 shrink-0 items-center justify-center rounded-[5px]">
			<X size={14} />
		</button>
	</div>
{/snippet}

{#snippet graphInner()}
	<div class="kg-grid pointer-events-none absolute inset-0" aria-hidden="true"></div>
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<svg
		bind:this={svgEl}
		viewBox="0 0 {W} {H}"
		class="h-full w-full"
		role="img"
		aria-label="Operating knowledge graph"
		onpointerdown={onCanvasPointerDown}
		onpointermove={onCanvasPointerMove}
		onpointerup={onCanvasPointerUp}
		onclick={onCanvasClick}
	>
		<!-- orbital rings: the guides ride the same wheel as the nodes -->
		{#if true}
			{@const gc = focusTree ? FOCUS_WHEEL.hub : { x: CX, y: CY }}
			{@const gk = focusTree ? FOCUS_WHEEL.scale : 1}
			<circle cx={gc.x} cy={gc.y} r={((RING_R[2] + RING_R[3]) / 2) * gk} fill="none" stroke="var(--bn-border)" stroke-width="1" opacity="0.28" class="kg-glide" />
			<circle cx={gc.x} cy={gc.y} r={((RING_R[3] + RING_R[4]) / 2) * gk} fill="none" stroke="var(--bn-border)" stroke-width="1" opacity="0.2" class="kg-glide" />
			<g bind:this={rimGuideEl} opacity="0.55" data-part="rim-guides">
				{#if !focusTree}
					<animateTransform attributeName="transform" attributeType="XML" type="rotate" from="0 {CX} {CY}" to="360 {CX} {CY}" dur="150s" repeatCount="indefinite" />
				{/if}
				{#each RING_R.slice(1) as r (r)}
					<circle cx={gc.x} cy={gc.y} r={r * gk} fill="none" stroke="var(--bn-border)" stroke-width="1" stroke-dasharray="2 6" class="kg-glide" />
				{/each}
			</g>
		{/if}

		<!-- the resting web: each edge in its pillar's colour at a whisper; the
		     open core keeps only the pillar spokes as coloured pathways -->
		{#if !focusTree}
			<g class={exitTree ? 'kg-web-in' : undefined} data-part="edges">
				{#each links as l, i (i)}
					{@const v = edgeView(l, i)}
					{#if v}
						<g>
							<path d={v.arc} fill="none" stroke={v.stroke} stroke-width={v.width} stroke-linecap="round" opacity={v.opacity} class={v.ray ? 'kg-ray' : undefined} style="transition: opacity 0.4s" />
							{#if v.synapse}
								<path d={v.arc} fill="none" stroke={v.stroke} stroke-width={l.kind === 'pillar' ? 1.7 : 1} stroke-linecap="round" pathLength="1" class={v.synapse} style="--kg-syn-delay: {(i % 7) * -0.7}s" />
							{/if}
						</g>
					{/if}
				{/each}
			</g>
		{/if}

		<!-- communication pulses between the core and each pillar head -->
		{#if !focusTree && memoryOn}
			<g style="pointer-events: none">
				{#each commTeams as t (t.id)}
					<circle bind:this={commEls[`${t.id}:out`]} r="2" fill={HUB_COLOR} opacity="0" transform="translate(-999,-999)" />
					<circle bind:this={commEls[`${t.id}:in`]} r="2.3" fill={t.color} opacity="0" transform="translate(-999,-999)" />
				{/each}
			</g>
		{/if}

		<!-- a closing tree's skeleton rides the nodes home while it fades -->
		{#if !focusTree && exitTree}
			<g class="kg-tree-exit" style="pointer-events: none">
				{#each exitTree.branches as b, i (i)}
					{@const s = pos.get(b.source)}
					{@const t = pos.get(b.target)}
					{#if s && t}
						<path
							d={branchPath(s, t)}
							fill="none"
							stroke={b.depth === 4 ? 'var(--bn-brain-2)' : b.depth === 3 ? (byId.get(b.target)?.kind === 'person' ? 'var(--bn-warn)' : 'var(--bn-accent)') : b.depth === 2 ? 'var(--bn-accent)' : 'var(--bn-text)'}
							stroke-width={branchWidth(b.depth)}
							stroke-linecap="round"
							stroke-dasharray={b.dashed ? '3 6' : undefined}
						/>
					{/if}
				{/each}
			</g>
		{/if}

		<!-- focused: the department grown as an organic tree -->
		{#if focusTree}
			{#key focusTeamId}
				<g style="pointer-events: none" data-part="focus-tree">
					<defs>
						<radialGradient id="kg-glow">
							<stop offset="0%" stop-color={focusedTeam?.color ?? 'var(--bn-accent)'} stop-opacity="0.18" />
							<stop offset="55%" stop-color={focusedTeam?.color ?? 'var(--bn-accent)'} stop-opacity="0.06" />
							<stop offset="100%" stop-color={focusedTeam?.color ?? 'var(--bn-accent)'} stop-opacity="0" />
						</radialGradient>
					</defs>
					<circle cx={W / 2} cy={H * 0.52} r={W * 0.56} fill="url(#kg-glow)" class="kg-glow" />
					{#each focusTree.extraLinks as l, i (`vine-${i}`)}
						{@const s = pos.get(l.source)}
						{@const t = pos.get(l.target)}
						{#if s && t}<line x1={s.x} y1={s.y} x2={t.x} y2={t.y} stroke="var(--bn-brain-2)" stroke-width="0.8" opacity="0.26" />{/if}
					{/each}
					{#each focusTree.branches as b, i (`br-${i}`)}
						{@const s = pos.get(b.source)}
						{@const t = pos.get(b.target)}
						{#if s && t}
							{#if b.depth === 4}
								<path d={branchPath(s, t)} fill="none" stroke="var(--bn-brain-2)" stroke-width={branchWidth(4)} stroke-linecap="round" class="kg-fade" />
							{:else if b.depth === 3}
								<path d={branchPath(s, t)} fill="none" stroke={byId.get(b.target)?.kind === 'person' ? 'var(--bn-warn)' : 'var(--bn-accent)'} stroke-width={branchWidth(3)} stroke-linecap="round" class="kg-fade" />
							{:else if b.depth === 2}
								<path d={branchPath(s, t)} fill="none" stroke="var(--bn-accent)" stroke-width={branchWidth(2)} stroke-linecap="round" class="kg-dash" />
							{:else}
								<path d={branchPath(s, t)} fill="none" stroke="var(--bn-text)" stroke-width={branchWidth(1)} stroke-linecap="round" pathLength="1" class="kg-grow" />
							{/if}
						{/if}
					{/each}
					{#each focusTree.branches.filter((b) => b.depth === 4) as b, i (`leaf-${i}`)}
						{@const t = pos.get(b.target)}
						{#if t}<circle cx={t.x} cy={t.y} r={CAT.tool.r + 5} fill="none" stroke="var(--bn-brain-2)" stroke-width="1" class="kg-leaf" style="animation-delay: {0.3 + i * 0.06}s" />{/if}
					{/each}
				</g>
			{/key}
		{/if}

		<!-- the flank departments ride the rim already expanded, faint -->
		{#if focusTree && flankTeams}
			<g opacity="0.22" style="pointer-events: none">
				{#each [...flankTeams] as teamId (teamId)}
					{#each allTrees.get(teamId)?.branches ?? [] as b, i (`fl-${teamId}-${i}`)}
						{@const s = pos.get(b.source)}
						{@const t = pos.get(b.target)}
						{#if s && t && b.source !== SELF_ID && b.target !== SELF_ID}
							<path d={branchPath(s, t)} fill="none" stroke={byId.get(teamId)?.color ?? 'var(--bn-text-3)'} stroke-width={branchWidth(b.depth) * 0.8} stroke-linecap="round" />
						{/if}
					{/each}
				{/each}
			</g>
		{/if}

		<g data-part="nodes">
			{#each nodes as n (n.id)}
				{@const p = pos.get(n.id) ?? { x: n.x, y: n.y }}
				{@const v = nodeView(n)}
				{#if n.kind === 'self' && memoryOn}
					<!-- The operator rendered as his memory: the Optimal Engine constellation -->
					<!-- svelte-ignore a11y_click_events_have_key_events -->
					<g
						bind:this={coreGEl}
						data-node={n.id}
						data-lit={lit ? (lit.has(n.id) ? 'true' : 'false') : 'true'}
						data-part="memory-core"
						transform="translate({p.x},{p.y})"
						opacity={v.dim ? 0.15 : 1}
						role="button"
						tabindex="-1"
						aria-label="The memory core"
						style="cursor: grab; transition: opacity 0.25s; outline: none"
						onmouseenter={() => {
							hoverId = n.id;
							coreGEl?.classList.add('kg-stirring');
						}}
						onmouseleave={() => {
							if (hoverId === n.id) hoverId = null;
							coreGEl?.classList.remove('kg-stirring');
						}}
						onpointerdown={(e) => onNodePointerDown(e, n.id)}
						onpointermove={(e) => onNodePointerMove(e, n.id)}
						onpointerup={(e) => onNodePointerUp(e, n.id)}
						onclick={(e) => nodeClick(e, n)}
					>
						<title>Optimal Engine: all of the operator's memory, click to open the graph</title>
						<g class={coreExpanded ? 'kg-core-open' : undefined} style="transform: scale({coreScale}); transition: transform {coreExpanded ? 900 : 450}ms cubic-bezier(0.22, 1, 0.36, 1)">
							<circle r={R_CORE + 10} fill="var(--bn-surface)" fill-opacity="0.72" stroke="var(--bn-border-strong)" stroke-width="1" />
							<circle r={R_CORE - 4} fill={HUB_COLOR} class="kg-core-glow" style="pointer-events: none" />
							<g bind:this={memRotGEl} transform="rotate({memRotDeg()})">
								{#each memEdges as e, i (i)}
									{@const s = memLayout.get(e.source)!}
									{@const t = memLayout.get(e.target)!}
									<line x1={s.x} y1={s.y} x2={t.x} y2={t.y} stroke={HUB_COLOR} stroke-width={e.type === 'wikilink' ? 0.28 : e.type === 'similar' ? 0.24 : 0.26} opacity={e.type === 'wikilink' ? 0.15 : e.type === 'similar' ? 0.1 : 0.2} />
								{/each}
								{#each memLayers as layer, li (li)}
									<g class="kg-mem-layer" style={MEM_LAYERS[li]}>
										<g class="kg-mem-stir" style={MEM_STIRS[li]}>
											{#each layer as m (m.id)}
												{@const mp = memLayout.get(m.id)}
												{#if mp}
													{@const mr = memNodeR(m)}
													<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
													<g
														class={coreExpanded && !memRestIds.has(m.id) ? 'kg-mem-in' : undefined}
														transform="translate({mp.x},{mp.y})"
														style="pointer-events: {coreExpanded ? 'auto' : 'none'}; cursor: pointer"
														data-mem={m.id}
														onmouseenter={() => (memHoverId = m.id)}
														onmouseleave={() => {
															if (memHoverId === m.id) memHoverId = null;
														}}
														onpointerdown={(e) => e.stopPropagation()}
														onpointerup={(e) => e.stopPropagation()}
														onclick={(e) => memNoteClick(e, m.id)}
													>
														<title>{m.label} · memory</title>
														{#if coreExpanded}<circle r={Math.max(6, mr + 4)} fill="transparent" />{/if}
														<polygon
															points={hexPts(mr)}
															fill={HUB_COLOR}
															fill-opacity={m.type === 'folder' ? 1 : m.type === 'page' && m.links === 0 ? 0.75 : 0.86 + ((hashStr(m.id) >> 3) % 15) / 100}
															stroke={HUB_COLOR}
															stroke-width={m.type === 'folder' ? 0.8 : 0}
															stroke-linejoin="round"
														/>
														{#if m.type === 'folder'}
															<g class="kg-mem-upright" transform="rotate({-memRotDeg()})">
																<text y={mr + 4} text-anchor="middle" font-family="var(--bn-font)" font-weight="600" fill={HUB_COLOR} opacity={coreExpanded ? 1 : 0} style="transition: opacity 300ms ease; pointer-events: none; {fixedLabel(10, coreScale)}">
																	{m.label.length > 18 ? `${m.label.slice(0, 16).trimEnd()}…` : m.label}
																</text>
															</g>
														{/if}
													</g>
												{/if}
											{/each}
										</g>
									</g>
								{/each}
							</g>
						</g>
						<g style="transform: scale({coreScale}); transition: transform {coreExpanded ? 900 : 450}ms cubic-bezier(0.22, 1, 0.36, 1); pointer-events: none">
							<g bind:this={synapseRotGEl} transform="rotate({memRotDeg()})">
								{#each Array.from({ length: SYNAPSE_N }, (_, i) => i) as i (i)}
									<circle bind:this={sparkEls[i]} r="0.8" fill={SYNAPSE_COLOR} opacity="0" />
								{/each}
							</g>
						</g>
						{#if memHoverId || selectedMemoryId || memHits.length > 0}
							<g style="transform: scale({coreScale}); transition: transform {coreExpanded ? 900 : 450}ms cubic-bezier(0.22, 1, 0.36, 1); pointer-events: none">
								{#if memHits.length > 0}<circle r={R_CORE + 10} fill="var(--bn-bg)" opacity="0.55" />{/if}
								<g bind:this={memOverlayGEl} transform="rotate({memRotDeg()})">
									{#each memHits as m (`hit-${m.id}`)}
										{@const hp = memLayout.get(m.id)}
										{#if hp}
											{@const mr = memNodeR(m)}
											<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
											<g
												transform="translate({hp.x},{hp.y})"
												style="pointer-events: auto; cursor: pointer"
												onpointerdown={(e) => e.stopPropagation()}
												onpointerup={(e) => e.stopPropagation()}
												onclick={(e) => {
													e.stopPropagation();
													clearDetail();
													selectedMemoryId = m.id;
												}}
											>
												<title>{m.label} · memory</title>
												<circle r={Math.max(4, mr + 3)} fill="transparent" />
												<polygon points={hexPts(mr + 0.4)} fill={memColor(m)} />
												<circle r={mr + 1.6} fill="none" stroke="#ffffff" stroke-width="0.5" opacity="0.9" />
												<g class="kg-mem-upright" transform="rotate({-memRotDeg()})">
													<text y={mr + 4} text-anchor="middle" font-family="var(--bn-font)" font-weight="500" fill="var(--bn-text-2)" style={fixedLabel(9, coreScale)}>{m.label.length > 22 ? `${m.label.slice(0, 20).trimEnd()}…` : m.label}</text>
												</g>
											</g>
										{/if}
									{/each}
									{#if memHoverId}
										{@const hp = memLayout.get(memHoverId)}
										{#if hp}
											{#each [...new Set(memAdj.get(memHoverId) ?? [])].slice(0, 14) as nid (nid)}
												{@const np = memLayout.get(nid)}
												{@const nm = memById.get(nid)}
												{#if np && nm}
													<line x1={hp.x} y1={hp.y} x2={np.x} y2={np.y} stroke="#ffffff" stroke-width="0.4" opacity="0.5" />
													<circle cx={np.x} cy={np.y} r={memNodeR(nm) + 1} fill="none" stroke="#ffffff" stroke-width="0.4" opacity="0.7" />
												{/if}
											{/each}
										{/if}
									{/if}
									{#each [...new Set([selectedMemoryId, memHoverId].filter((x): x is string => !!x))] as id (id)}
										{@const m = memById.get(id)}
										{@const mp = memLayout.get(id)}
										{#if m && mp}
											{@const mr = memNodeR(m)}
											<g transform="translate({mp.x},{mp.y})">
												<circle r={mr + 1.4} fill="none" stroke="#ffffff" stroke-width={selectedMemoryId === id ? 0.9 : 0.55} opacity="0.95" />
												<g class="kg-mem-upright" transform="rotate({-memRotDeg()})">
													<text y={mr + 4} text-anchor="middle" font-family="var(--bn-font)" font-weight="500" fill="var(--bn-text-2)" style={fixedLabel(9.5, coreScale)}>{m.label.length > 24 ? `${m.label.slice(0, 22).trimEnd()}…` : m.label}</text>
												</g>
											</g>
										{/if}
									{/each}
								</g>
							</g>
						{/if}
					</g>
				{:else}
					<!-- svelte-ignore a11y_click_events_have_key_events -->
					<g
						data-node={n.id}
						data-lit={v.dim ? 'false' : 'true'}
						transform="translate({p.x},{p.y})"
						opacity={v.opacity}
						role="button"
						tabindex="-1"
						aria-label={`${n.label} · ${CAT[n.kind].label}`}
						style="cursor: grab; transition: opacity 0.25s; outline: none; {v.hidden ? 'pointer-events: none' : ''}"
						onmouseenter={() => (hoverId = n.id)}
						onmouseleave={() => {
							if (hoverId === n.id) hoverId = null;
						}}
						onpointerdown={(e) => onNodePointerDown(e, n.id)}
						onpointermove={(e) => onNodePointerMove(e, n.id)}
						onpointerup={(e) => onNodePointerUp(e, n.id)}
						onclick={(e) => nodeClick(e, n)}
					>
						<title>{n.label}</title>
						{#if v.selected}<circle r={v.r + 3.5} fill="none" stroke={HUB_COLOR} stroke-width="1" opacity="0.4" />{/if}
						<circle
							data-part="ring"
							r={v.r}
							fill={n.kind === 'self' ? v.color : 'var(--bn-surface)'}
							stroke={v.color}
							stroke-opacity={n.kind === 'board' ? 0.55 : 1}
							stroke-width={n.kind === 'board' ? (v.selected || hoverId === n.id ? 1.4 : 0.6) : v.selected || hoverId === n.id ? 2.5 : 1.5}
						/>
						<g style="color: {n.kind === 'self' ? 'var(--bn-bg)' : v.color}">
							<KindIcon kind={n.kind} size={v.r * 1.24} x={-v.r * 0.62} y={-v.r * 0.62} />
						</g>
						{#if v.showLabel && n.kind === 'board'}
							{@const bl = boardLabel(p, v.r)}
							<text x={bl.x} y={bl.y} text-anchor={bl.anchor as 'start' | 'end' | 'middle'} font-family="var(--bn-font)" font-weight="600" fill="var(--bn-text-2)" style={fixedLabel(9)}>{shortLabel(n)}</text>
						{:else if v.showLabel}
							<text
								x="0"
								y={v.r + 11 + (labelDy.get(n.id) ?? 0)}
								text-anchor="middle"
								font-family="var(--bn-font)"
								font-weight={n.kind === 'self' || n.kind === 'team' ? 600 : 400}
								fill={n.kind === 'team' ? v.color : 'var(--bn-text-2)'}
								style={fixedLabel(n.kind === 'self' || n.kind === 'team' ? 10 : n.kind === 'task' ? 8.5 : 9)}>{shortLabel(n)}</text
							>
						{/if}
					</g>
				{/if}
			{/each}
		</g>
	</svg>
{/snippet}

{#if fullscreen}
	<KnowledgeGraphFullscreen
		{deptList}
		currentTeamId={focusTeamId}
		{currentDept}
		hasDetail={detailOpen}
		coreOpen={coreExpanded}
		onCollapseCore={clearAll}
		hasSearch={coreExpanded}
		{directoryCollapsed}
		onNavDept={navDept}
		onBack={clearDetail}
		onClose={() => (fullscreen = false)}
	>
		{#snippet searchSlot()}{@render vaultSearch()}{/snippet}
		{#snippet legendSlot()}{@render compactLegend()}{/snippet}
		{#snippet directorySlot()}{@render directoryPanel()}{/snippet}
		{#snippet detail()}{@render detailBody()}{/snippet}
		<div class="bn-kg contents">{@render graphInner()}</div>
	</KnowledgeGraphFullscreen>
{:else}
	<div class="bn-kg flex flex-col gap-3 lg:flex-row {fill ? 'h-full' : ''}" data-part="knowledge-graph" data-view="radial">
		{#if detailOpen && detailExpanded}
			<aside class="kg-bs kg-bg95 order-first flex h-[560px] w-full shrink-0 flex-col overflow-hidden rounded-[10px] border lg:w-[420px] {fill ? 'lg:h-full' : 'lg:h-[680px]'}" data-part="detail-wide">
				{@render detailTopBar()}
				<div class="min-h-0 flex-1 overflow-hidden">{@render detailBody()}</div>
			</aside>
		{/if}
		<div class="kg-b kg-surf relative min-w-0 flex-1 overflow-hidden rounded-[10px] border {fill ? 'h-full min-h-[440px]' : 'h-[680px]'}" data-part="graph-canvas">
			{@render graphInner()}
			{#if cardNode}
				<div class="absolute bottom-3 left-3 z-20">
					<GraphNodeCard
						label={cardNode.label}
						kind={cardNode.kind}
						links={(rel.adjacency.get(cardNode.id)?.size ?? 1) - 1}
						sub={byId.get(teamForFocus(cardNode.id) ?? '')?.label}
						onOpen={openCardNode}
						onDismiss={() => {
							cardId = null;
							hoverId = null;
						}}
					/>
				</div>
			{/if}

			<button
				type="button"
				onclick={() => {
					if (!focusTeamId && deptList[0]) navDept(deptList[0].teamId);
					fullscreen = true;
				}}
				title="Open the department wheel"
				class="bn-pressable kg-bs kg-bg80 kg-blur bn-muted kg-h-accent absolute right-3 top-3 z-20 flex items-center gap-1.5 rounded-[5px] border px-2 py-1 font-mono text-[10.5px]"
				data-part="fullscreen"
			>
				<Maximize2 size={14} /> Fullscreen
			</button>

			{#if focusSet && !coreExpanded}
				<div class="pointer-events-none absolute left-1/2 top-3 z-20 -translate-x-1/2" data-part="dept-title">
					<span class="text-[22px] font-bold uppercase leading-none tracking-[0.08em]" style="color: #ffffff; text-shadow: 0 1px 8px rgba(0,0,0,0.75)">{focusedTeam?.label}</span>
				</div>
			{/if}

			{#if focusSet || coreExpanded}
				<div class="absolute left-3 top-3 z-10 flex items-center gap-2">
					<button type="button" onclick={clearAll} aria-label="Back to the home view" title="Back to the home view" class="bn-pressable kg-bs kg-bg85 kg-blur bn-muted kg-h-accent flex items-center gap-1.5 rounded-[5px] border px-2 py-1.5 font-mono text-[10.5px]">
						<ArrowLeft size={14} /> Back
					</button>
					{@render vaultSearch()}
					{#if focusSet}
						<div class="kg-bs kg-bg85 kg-blur flex items-center gap-0.5 rounded-[5px] border px-1 py-1">
							<button type="button" onclick={() => stepDept(-1)} aria-label="Previous department" title="Previous department" class="bn-pressable bn-dim kg-h-text flex h-6 w-6 items-center justify-center rounded-[5px]"><ChevronLeft size={14} /></button>
							<button type="button" onclick={() => stepDept(1)} aria-label="Next department" title="Next department" class="bn-pressable bn-dim kg-h-text flex h-6 w-6 items-center justify-center rounded-[5px]"><ChevronRight size={14} /></button>
							<span class="max-w-[120px] truncate px-1.5 font-mono text-[11px] font-semibold" style="color: {focusedTeam?.color ?? 'var(--bn-text)'}">{focusedTeam?.label}</span>
							<button type="button" onclick={clearAll} aria-label="Close focus" title="Back to all" class="bn-pressable kg-b bn-dim kg-h-err flex h-6 w-6 items-center justify-center rounded-[5px] border-l"><X size={14} /></button>
						</div>
					{/if}
				</div>
			{/if}

			{#if focusSet && !coreExpanded}
				<div class="kg-bs kg-bg90 kg-blur absolute bottom-4 left-1/2 z-20 flex -translate-x-1/2 items-center gap-1 rounded-full border px-1.5 py-1.5" data-part="pillar-stepper">
					<button type="button" onclick={() => stepDept(-1)} aria-label="Turn to the previous pillar" title="Previous pillar (←)" class="bn-pressable bn-muted kg-h-surf kg-h-text flex h-9 w-9 items-center justify-center rounded-full"><ChevronLeft size={20} /></button>
					<span class="min-w-[92px] px-1 text-center font-mono text-[11px] font-semibold leading-none" style="color: {focusedTeam?.color ?? 'var(--bn-text)'}">{focusedTeam?.label ?? 'Pillar'}</span>
					<button type="button" onclick={() => stepDept(1)} aria-label="Turn to the next pillar" title="Next pillar (→)" class="bn-pressable bn-muted kg-h-surf kg-h-text flex h-9 w-9 items-center justify-center rounded-full"><ChevronRight size={20} /></button>
				</div>
			{/if}

			{#if memoryError}
				<div class="kg-warn absolute bottom-2 left-3 font-mono text-[9.5px]" data-part="memory-error">memory core unavailable: {memoryError}</div>
			{/if}

			{#if detailOpen && !detailExpanded}
				<div class="kg-panel kg-bs kg-bg95 kg-blur absolute left-3 top-[46px] z-10 flex h-[calc(100%-58px)] w-[320px] flex-col overflow-hidden rounded-[10px] border" data-part="detail-panel">
					{@render detailTopBar()}
					<div class="min-h-0 flex-1 overflow-hidden">{@render detailBody()}</div>
				</div>
			{/if}
		</div>

		<!-- the always-visible lens + legend + directory, on the right -->
		<aside
			class="kg-b kg-surf flex max-h-[560px] shrink-0 flex-col gap-3.5 rounded-[10px] border p-3 lg:max-h-none {fill ? 'lg:h-full' : 'lg:h-[680px]'} {directoryCollapsed ? 'w-auto lg:w-16' : 'w-full lg:w-72'}"
			data-part="graph-aside"
		>
			<div class={directoryCollapsed ? 'hidden' : undefined}>
				<div class="bn-dim mb-1.5 flex items-baseline justify-between font-mono text-[9px] uppercase tracking-[0.16em]">
					<span>Lens</span>
					{#if lensId}<button type="button" onclick={() => (lensId = null)} class="bn-pressable bn-dim kg-h-err">clear · {lensLit?.size ?? 0} lit</button>{/if}
				</div>
				<div class="mb-3 flex flex-col gap-1">
					{#each LENS_GROUPS as [groupLabel, lenses] (groupLabel)}
						<select
							value={lenses.some((l) => l.id === lensId) ? lensId : ''}
							onchange={(e) => (lensId = (e.currentTarget as HTMLSelectElement).value || null)}
							aria-label={`${groupLabel} lens`}
							class="kg-b kg-bg bn-muted kg-lens-select w-full rounded-[5px] border px-1.5 py-1 font-mono text-[10px] focus:outline-none"
						>
							<option value="">{groupLabel} · all</option>
							{#each lenses as l (l.id)}<option value={l.id}>{l.label}</option>{/each}
						</select>
					{/each}
				</div>
				<div class="bn-dim mb-1.5 font-mono text-[9px] uppercase tracking-[0.16em]">Legend</div>
				<div class="flex flex-col gap-1" data-part="legend">
					{#each LEGEND_KINDS as k (k)}
						<div class="flex items-center gap-2" data-kind={k}>
							<span class="grid h-5 w-5 shrink-0 place-items-center rounded-full border" style="border-color: {CAT[k].color}; color: {CAT[k].color}"><KindIcon kind={k} size={12} /></span>
							<span class="bn-text flex-1 text-[11px] font-semibold">{CAT[k].label}</span>
							<span class="bn-dim font-mono text-[10px]">{legendCounts[k]}</span>
						</div>
					{/each}
				</div>
			</div>
			<div class="flex min-h-0 flex-1 flex-col {directoryCollapsed ? '' : 'kg-b border-t pt-3'}">
				<div class="min-h-0 flex-1">{@render directoryPanel()}</div>
			</div>
		</aside>
	</div>
{/if}

<style>
	.kg-glide {
		transition:
			cx 900ms var(--bn-ease-lens, ease),
			cy 900ms var(--bn-ease-lens, ease),
			r 900ms var(--bn-ease-lens, ease);
	}
	.kg-lens-select:focus {
		border-color: var(--bn-border-strong);
	}
</style>
