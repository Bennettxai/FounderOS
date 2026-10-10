<script lang="ts">
	import { Popover } from 'bits-ui';
	import type { PropertySchema, Context, ContextListItem } from '$lib/api';
	import { contexts } from '$lib/stores/contexts';

	interface Props {
		propertySchema: PropertySchema[];
		properties: Record<string, unknown>;
		onSchemaChange: (schema: PropertySchema[]) => void;
		onPropertiesChange: (properties: Record<string, unknown>) => void;
		allContexts?: ContextListItem[];
	}

	let {
		propertySchema,
		properties,
		onSchemaChange,
		onPropertiesChange,
		allContexts = []
	}: Props = $props();

	let showAddProperty = $state(false);
	let newPropertyName = $state('');
	let newPropertyType = $state<PropertySchema['type']>('text');
	let editingProperty = $state<string | null>(null);
	let showPropertyMenu = $state<string | null>(null);

	const propertyTypes: { value: PropertySchema['type']; label: string; icon: string }[] = [
		{ value: 'text', label: 'Text', icon: 'T' },
		{ value: 'select', label: 'Select', icon: '▼' },
		{ value: 'multi_select', label: 'Multi-select', icon: '☰' },
		{ value: 'date', label: 'Date', icon: '📅' },
		{ value: 'number', label: 'Number', icon: '#' },
		{ value: 'checkbox', label: 'Checkbox', icon: '☑' },
		{ value: 'url', label: 'URL', icon: '🔗' },
		{ value: 'email', label: 'Email', icon: '@' },
		{ value: 'relation', label: 'Relation', icon: '↔' }
	];

	function addProperty() {
		if (!newPropertyName.trim()) return;

		const newProp: PropertySchema = {
			name: newPropertyName.trim(),
			type: newPropertyType
		};

		if (newPropertyType === 'select' || newPropertyType === 'multi_select') {
			newProp.options = [];
		}

		if (newPropertyType === 'relation') {
			newProp.relation_type = 'context';
		}

		onSchemaChange([...propertySchema, newProp]);
		newPropertyName = '';
		newPropertyType = 'text';
		showAddProperty = false;
	}

	function removeProperty(propName: string) {
		onSchemaChange(propertySchema.filter(p => p.name !== propName));
		const newProps = { ...properties };
		delete newProps[propName];
		onPropertiesChange(newProps);
		showPropertyMenu = null;
	}

	function updatePropertyValue(propName: string, value: unknown) {
		onPropertiesChange({ ...properties, [propName]: value });
	}

	function addSelectOption(propName: string, option: string) {
		const schema = propertySchema.find(p => p.name === propName);
		if (!schema) return;
		if (!schema.options) schema.options = [];
		if (!schema.options.includes(option)) {
			schema.options.push(option);
			onSchemaChange([...propertySchema]);
		}
	}

	function getSelectColor(option: string, index: number) {
		const colors = [
			'bg-blue-100 text-blue-700',
			'bg-green-100 text-green-700',
			'bg-purple-100 text-purple-700',
			'bg-amber-100 text-amber-700',
			'bg-pink-100 text-pink-700',
			'bg-cyan-100 text-cyan-700',
			'bg-red-100 text-red-700',
			'bg-indigo-100 text-indigo-700'
		];
		return colors[index % colors.length];
	}
</script>

{#if propertySchema.length > 0 || showAddProperty}
	<div class="mb-6 space-y-2 border-b border-gray-100 pb-4">
		{#each propertySchema as prop, propIndex}
			<div class="flex items-center gap-3 group">
				<!-- Property Name -->
				<div class="relative">
					<button
						onclick={() => showPropertyMenu = showPropertyMenu === prop.name ? null : prop.name}
						class="text-xs text-gray-500 font-medium w-28 text-left hover:text-gray-700 flex items-center gap-1"
					>
						{prop.name}
						<svg class="w-3 h-3 opacity-0 group-hover:opacity-100 transition-opacity" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
						</svg>
					</button>
					{#if showPropertyMenu === prop.name}
						<div class="absolute top-full left-0 mt-1 bg-white rounded-lg shadow-xl border border-gray-200 py-1 z-50 min-w-32">
							<button
								onclick={() => removeProperty(prop.name)}
								class="w-full px-3 py-1.5 text-left text-sm text-red-600 hover:bg-red-50"
							>
								Delete property
							</button>
						</div>
					{/if}
				</div>

				<!-- Property Value -->
				<div class="flex-1">
					{#if prop.type === 'text'}
						<input
							type="text"
							value={properties[prop.name] as string || ''}
							oninput={(e) => updatePropertyValue(prop.name, (e.target as HTMLInputElement).value)}
							class="w-full px-2 py-1 text-sm border-0 rounded hover:bg-gray-50 focus:bg-gray-50 focus:ring-1 focus:ring-gray-300"
							placeholder="Empty"
						/>
					{:else if prop.type === 'number'}
						<input
							type="number"
							value={properties[prop.name] as number || ''}
							oninput={(e) => updatePropertyValue(prop.name, Number((e.target as HTMLInputElement).value))}
							class="w-full px-2 py-1 text-sm border-0 rounded hover:bg-gray-50 focus:bg-gray-50 focus:ring-1 focus:ring-gray-300"
							placeholder="Empty"
						/>
					{:else if prop.type === 'date'}
						<input
							type="date"
							value={properties[prop.name] as string || ''}
							oninput={(e) => updatePropertyValue(prop.name, (e.target as HTMLInputElement).value)}
							class="w-full px-2 py-1 text-sm border-0 rounded hover:bg-gray-50 focus:bg-gray-50 focus:ring-1 focus:ring-gray-300"
						/>
					{:else if prop.type === 'checkbox'}
						<button
							onclick={() => updatePropertyValue(prop.name, !properties[prop.name])}
							class="w-5 h-5 rounded border-2 flex items-center justify-center transition-colors {properties[prop.name] ? 'bg-gray-900 border-gray-900' : 'border-gray-300 hover:border-gray-400'}"
						>
							{#if properties[prop.name]}
								<svg class="w-3 h-3 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M5 13l4 4L19 7" />
								</svg>
							{/if}
						</button>
					{:else if prop.type === 'url'}
						<input
							type="url"
							value={properties[prop.name] as string || ''}
							oninput={(e) => updatePropertyValue(prop.name, (e.target as HTMLInputElement).value)}
							class="w-full px-2 py-1 text-sm border-0 rounded hover:bg-gray-50 focus:bg-gray-50 focus:ring-1 focus:ring-gray-300 text-blue-600"
							placeholder="https://..."
						/>
					{:else if prop.type === 'email'}
						<input
							type="email"
							value={properties[prop.name] as string || ''}
							oninput={(e) => updatePropertyValue(prop.name, (e.target as HTMLInputElement).value)}
							class="w-full px-2 py-1 text-sm border-0 rounded hover:bg-gray-50 focus:bg-gray-50 focus:ring-1 focus:ring-gray-300"
							placeholder="email@example.com"
						/>
					{:else if prop.type === 'select'}
						<Popover.Root>
							<Popover.Trigger class="px-2 py-1 text-sm rounded hover:bg-gray-50 text-left">
								{#if properties[prop.name]}
									<span class="px-2 py-0.5 rounded-full text-xs {getSelectColor(properties[prop.name] as string, prop.options?.indexOf(properties[prop.name] as string) || 0)}">
										{properties[prop.name]}
									</span>
								{:else}
									<span class="text-gray-400">Select...</span>
								{/if}
							</Popover.Trigger>
							<Popover.Content class="z-50 bg-white rounded-lg shadow-xl border border-gray-200 p-2 min-w-40">
								{#each (prop.options || []) as option, i}
									<button
										onclick={() => updatePropertyValue(prop.name, option)}
										class="w-full px-2 py-1 text-left rounded hover:bg-gray-50 text-sm"
									>
										<span class="px-2 py-0.5 rounded-full text-xs {getSelectColor(option, i)}">{option}</span>
									</button>
								{/each}
								<div class="border-t border-gray-100 mt-1 pt-1">
									<input
										type="text"
										placeholder="Add option..."
										class="w-full px-2 py-1 text-sm border-0 focus:ring-0"
										onkeydown={(e) => {
											if (e.key === 'Enter') {
												const input = e.target as HTMLInputElement;
												if (input.value.trim()) {
													addSelectOption(prop.name, input.value.trim());
													input.value = '';
												}
											}
										}}
									/>
								</div>
							</Popover.Content>
						</Popover.Root>
					{:else if prop.type === 'multi_select'}
						<div class="flex flex-wrap gap-1 items-center">
							{#each ((properties[prop.name] as string[]) || []) as selected, i}
								<span class="px-2 py-0.5 rounded-full text-xs {getSelectColor(selected, i)} flex items-center gap-1">
									{selected}
									<button
										onclick={() => {
											const current = (properties[prop.name] as string[]) || [];
											updatePropertyValue(prop.name, current.filter(s => s !== selected));
										}}
										class="hover:text-red-600"
									>
										<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
										</svg>
									</button>
								</span>
							{/each}
							<Popover.Root>
								<Popover.Trigger class="text-gray-400 hover:text-gray-600 text-sm">+ Add</Popover.Trigger>
								<Popover.Content class="z-50 bg-white rounded-lg shadow-xl border border-gray-200 p-2 min-w-40">
									{#each (prop.options || []).filter(o => !((properties[prop.name] as string[]) || []).includes(o)) as option, i}
										<button
											onclick={() => {
												const current = (properties[prop.name] as string[]) || [];
												updatePropertyValue(prop.name, [...current, option]);
											}}
											class="w-full px-2 py-1 text-left rounded hover:bg-gray-50 text-sm"
										>
											<span class="px-2 py-0.5 rounded-full text-xs {getSelectColor(option, i)}">{option}</span>
										</button>
									{/each}
									<div class="border-t border-gray-100 mt-1 pt-1">
										<input
											type="text"
											placeholder="Add option..."
											class="w-full px-2 py-1 text-sm border-0 focus:ring-0"
											onkeydown={(e) => {
												if (e.key === 'Enter') {
													const input = e.target as HTMLInputElement;
													if (input.value.trim()) {
														addSelectOption(prop.name, input.value.trim());
														const current = (properties[prop.name] as string[]) || [];
														updatePropertyValue(prop.name, [...current, input.value.trim()]);
														input.value = '';
													}
												}
											}}
										/>
									</div>
								</Popover.Content>
							</Popover.Root>
						</div>
					{:else if prop.type === 'relation'}
						<div class="flex flex-wrap gap-1 items-center">
							{#each ((properties[prop.name] as string[]) || []) as relatedId}
								{@const relatedDoc = allContexts.find(c => c.id === relatedId)}
								{#if relatedDoc}
									<span class="px-2 py-0.5 rounded-full text-xs bg-gray-100 text-gray-700 flex items-center gap-1">
										{relatedDoc.icon || '📄'} {relatedDoc.name}
										<button
											onclick={() => {
												const current = (properties[prop.name] as string[]) || [];
												updatePropertyValue(prop.name, current.filter(s => s !== relatedId));
											}}
											class="hover:text-red-600"
										>
											<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
											</svg>
										</button>
									</span>
								{/if}
							{/each}
							<Popover.Root>
								<Popover.Trigger class="text-gray-400 hover:text-gray-600 text-sm">+ Link</Popover.Trigger>
								<Popover.Content class="z-50 bg-white rounded-lg shadow-xl border border-gray-200 p-2 min-w-48 max-h-64 overflow-y-auto">
									{#each allContexts.filter(c => !((properties[prop.name] as string[]) || []).includes(c.id)) as ctx}
										<button
											onclick={() => {
												const current = (properties[prop.name] as string[]) || [];
												updatePropertyValue(prop.name, [...current, ctx.id]);
											}}
											class="w-full px-2 py-1.5 text-left rounded hover:bg-gray-50 text-sm flex items-center gap-2"
										>
											<span>{ctx.icon || '📄'}</span>
											<span class="truncate">{ctx.name}</span>
										</button>
									{/each}
									{#if allContexts.length === 0}
										<p class="text-xs text-gray-400 px-2 py-1">No documents available</p>
									{/if}
								</Popover.Content>
							</Popover.Root>
						</div>
					{/if}
				</div>
			</div>
		{/each}

		<!-- Add Property -->
		{#if showAddProperty}
			<div class="flex items-center gap-2 mt-2 p-2 bg-gray-50 rounded-lg">
				<input
					type="text"
					bind:value={newPropertyName}
					placeholder="Property name"
					class="flex-1 px-2 py-1 text-sm border border-gray-200 rounded focus:ring-1 focus:ring-gray-300"
					onkeydown={(e) => e.key === 'Enter' && addProperty()}
				/>
				<select
					bind:value={newPropertyType}
					class="px-2 py-1 text-sm border border-gray-200 rounded focus:ring-1 focus:ring-gray-300"
				>
					{#each propertyTypes as pt}
						<option value={pt.value}>{pt.icon} {pt.label}</option>
					{/each}
				</select>
				<button
					onclick={addProperty}
					class="btn-pill btn-pill-primary btn-pill-sm"
				>
					Add
				</button>
				<button
					onclick={() => showAddProperty = false}
					class="px-3 py-1 text-sm text-gray-500 hover:text-gray-700"
				>
					Cancel
				</button>
			</div>
		{/if}
	</div>
{/if}

<!-- Add Property Button (always visible when no properties) -->
{#if propertySchema.length === 0 && !showAddProperty}
	<button
		onclick={() => showAddProperty = true}
		class="mb-4 text-sm text-gray-400 hover:text-gray-600 flex items-center gap-1 opacity-0 hover:opacity-100 focus:opacity-100 transition-opacity"
	>
		<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
		</svg>
		Add property
	</button>
{:else if !showAddProperty}
	<button
		onclick={() => showAddProperty = true}
		class="text-sm text-gray-400 hover:text-gray-600 flex items-center gap-1 mb-2"
	>
		<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
		</svg>
		Add property
	</button>
{/if}
