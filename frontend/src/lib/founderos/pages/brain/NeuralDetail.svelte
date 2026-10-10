<!-- The radial view's detail cards wired for the neural view (FounderOS v1
     components/NeuralDetail.tsx): resolves a clicked neuron (emp: / person: /
     task: / tool:) to its card with the chain derived from the raw rows. -->
<script lang="ts">
	import AgentHarnessCard from './detail/AgentHarnessCard.svelte';
	import GraphHumanDetailCard from './detail/GraphHumanDetailCard.svelte';
	import SopTaskDetailCard from './detail/SopTaskDetailCard.svelte';
	import ToolDetailCard from './detail/ToolDetailCard.svelte';
	import { buildToolWiki, prettifySlug, toolSlugOf } from './kg';
	import type { Agent, AgentRun, Department, KnowledgeGraph, Person, SopTask } from './types';

	let {
		nodeId,
		graph,
		agents,
		departments,
		people,
		tasks,
		runsByAgent,
		onSelect,
		onClose
	}: {
		nodeId: string;
		graph: KnowledgeGraph;
		agents: Agent[];
		departments: Department[];
		people: Person[];
		tasks: SopTask[];
		runsByAgent: Record<string, AgentRun>;
		onSelect: (nodeId: string) => void;
		onClose: () => void;
	} = $props();

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
	const deptName = (id: string) => departments.find((d) => d.id === id)?.name ?? id;
	const chips = (slugs: string[]) => slugs.map((slug) => ({ slug, name: prettifySlug(slug), mcp: buildToolWiki(slug).mcp }));
	function selectTool(slug: string) {
		const id = graph.nodes.find((n) => n.kind === 'tool' && toolSlugOf(n.id) === slug)?.id;
		if (id) onSelect(id);
	}

	const agent = $derived(nodeId.startsWith('emp:') ? agents.find((a) => `emp:${a.id}` === nodeId) ?? null : null);
	const person = $derived(nodeId.startsWith('person:') ? people.find((p) => `person:${p.id}` === nodeId) ?? null : null);
	const task = $derived(nodeId.startsWith('task:') ? tasks.find((t) => `task:${t.id}` === nodeId) ?? null : null);
</script>

{#if agent}
	{@const t = tasks.find((x) => x.assigneeKind === 'agent' && x.assigneeId === agent.id) ?? null}
	{@const parent = agent.parentId ? agents.find((a) => a.id === agent.parentId) ?? null : null}
	{@const run = runsByAgent[agent.id] ?? null}
	<AgentHarnessCard
		{agent}
		task={t}
		parentName={parent?.name ?? null}
		parentAgentId={parent?.id ?? null}
		subAgents={agents.filter((a) => a.parentId === agent.id).map((a) => ({ id: a.id, name: a.name }))}
		lastRun={run ? { ok: run.ok, summary: run.summary } : null}
		runLabel={run ? agoLabel(run.finishedAt) : null}
		headName={null}
		{onClose}
		onTool={selectTool}
		onAgent={(id) => onSelect(`emp:${id}`)}
		onTask={t ? () => onSelect(`task:${t.id}`) : undefined}
	/>
{:else if person}
	{@const t = tasks.find((x) => x.assigneeKind === 'person' && x.assigneeId === person.id) ?? null}
	<GraphHumanDetailCard {person} deptName={deptName(person.departmentId)} color="var(--bn-warn)" task={t} tools={chips(person.tools)} {onClose} onTask={t ? () => onSelect(`task:${t.id}`) : undefined} onTool={selectTool} />
{:else if task}
	{@const a = task.assigneeKind === 'agent' ? agents.find((x) => x.id === task.assigneeId) ?? null : null}
	{@const p = task.assigneeKind === 'person' ? people.find((x) => x.id === task.assigneeId) ?? null : null}
	{@const worker = a ? `emp:${a.id}` : p ? `person:${p.id}` : null}
	<SopTaskDetailCard
		{task}
		assigneeName={a?.name ?? p?.name ?? task.assigneeId}
		assigneeKindLabel={a ? 'AI agent' : 'human'}
		assigneeColor={a ? 'var(--bn-accent)' : 'var(--bn-warn)'}
		runtime={a ? `${a.instance} · ${a.model}` : 'human · judgment call'}
		tools={chips(a?.tools ?? p?.tools ?? [])}
		{onClose}
		onAssignee={worker ? () => onSelect(worker) : undefined}
		onTool={selectTool}
	/>
{:else if nodeId.startsWith('tool:')}
	{@const slug = toolSlugOf(nodeId)}
	<ToolDetailCard wiki={buildToolWiki(slug, [...agents.filter((a) => a.tools.includes(slug)).map((a) => a.name), ...people.filter((p) => p.tools.includes(slug)).map((p) => p.name)])} {onClose} />
{/if}
