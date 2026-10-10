<script lang="ts">
	/**
	 * Column Type Icon
	 * Renders appropriate icon for each database column type
	 */
	import {
		Type,
		Hash,
		List,
		Calendar,
		CheckSquare,
		Link,
		Mail,
		Phone,
		User,
		File,
		GitBranch,
		Calculator,
		Clock,
		UserCheck,
		AlignLeft
	} from 'lucide-svelte';
	import type { ColumnType } from '../../entities/block';

	interface Props {
		type: ColumnType;
		class?: string;
	}

	let { type, class: className = '' }: Props = $props();

	const iconMap: Record<ColumnType, typeof Type> = {
		'title': Type,
		'text': AlignLeft,
		'number': Hash,
		'select': List,
		'multi-select': List,
		'date': Calendar,
		'checkbox': CheckSquare,
		'url': Link,
		'email': Mail,
		'phone': Phone,
		'person': User,
		'file': File,
		'relation': GitBranch,
		'rollup': Calculator,
		'formula': Calculator,
		'created-time': Clock,
		'created-by': UserCheck,
		'updated-time': Clock,
		'updated-by': UserCheck
	};

	const Icon = $derived(iconMap[type] ?? Type);
</script>

<svelte:component this={Icon} class="bos-column-icon {className}" />

<style>
	:global(.bos-column-icon) {
		width: 14px;
		height: 14px;
		color: var(--dt4);
		flex-shrink: 0;
	}
</style>
