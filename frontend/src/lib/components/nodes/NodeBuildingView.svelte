<script lang="ts">
	import type { NodeTree } from '$lib/api/nodes/types';

	interface Props {
		nodes: NodeTree[];
		activeNodeId?: string | null;
		selectedId?: string | null;
		onSelect?: (node: NodeTree) => void;
		onNavigate?: (node: NodeTree) => void;
		onCreateRoom?: (floorLevel: number) => void;
	}

	let { nodes, activeNodeId = null, selectedId = null, onSelect, onNavigate, onCreateRoom }: Props = $props();

	// Floor organization
	interface FloorData {
		level: number;
		rooms: NodeTree[];
	}

	function flattenWithDepth(nodeList: NodeTree[], depth: number = 0): { node: NodeTree; depth: number }[] {
		let result: { node: NodeTree; depth: number }[] = [];
		for (const node of nodeList) {
			result.push({ node, depth });
			if (node.children && node.children.length > 0) {
				result = result.concat(flattenWithDepth(node.children, depth + 1));
			}
		}
		return result;
	}

	const floors = $derived(() => {
		const flatNodes = flattenWithDepth(nodes);
		const floorMap = new Map<number, NodeTree[]>();

		for (const { node, depth } of flatNodes) {
			if (!floorMap.has(depth)) {
				floorMap.set(depth, []);
			}
			floorMap.get(depth)!.push(node);
		}

		const floorsArray: FloorData[] = [];
		const maxDepth = Math.max(...Array.from(floorMap.keys()), 0);

		// Build floors from top to bottom for display
		for (let i = maxDepth; i >= 0; i--) {
			floorsArray.push({
				level: i,
				rooms: floorMap.get(i) || []
			});
		}

		return floorsArray;
	});

	// Colors
	const typeStyles: Record<string, { window: string; glow: string }> = {
		business: { window: '#3B82F6', glow: 'rgba(59, 130, 246, 0.6)' },
		project: { window: '#10B981', glow: 'rgba(16, 185, 129, 0.6)' },
		learning: { window: '#8B5CF6', glow: 'rgba(139, 92, 246, 0.6)' },
		operational: { window: '#F59E0B', glow: 'rgba(245, 158, 11, 0.6)' }
	};

	const defaultStyle = { window: '#64748B', glow: 'rgba(100, 116, 139, 0.6)' };

	function getStyle(type: string) {
		return typeStyles[type] || defaultStyle;
	}

	function getHealthColor(health: string): string {
		switch (health) {
			case 'healthy': return '#22C55E';
			case 'needs_attention': return '#EAB308';
			case 'critical': return '#EF4444';
			default: return '#6B7280';
		}
	}

	function handleClick(node: NodeTree) {
		onSelect?.(node);
	}

	function handleDblClick(node: NodeTree) {
		onNavigate?.(node);
	}

	function handleAddRoom(level: number) {
		onCreateRoom?.(level);
	}

	const totalRooms = $derived(flattenWithDepth(nodes).length);
	const floorCount = $derived(floors().length || 1);

	// Agent activity messages
	const agentActivities = [
		'Analyzing data...',
		'Checking files...',
		'Processing...',
		'Thinking...',
		'Working...',
		'Organizing...'
	];
	let activityIndex = $state(0);

	// Cycle through activities
	$effect(() => {
		if (activeNodeId) {
			const interval = setInterval(() => {
				activityIndex = (activityIndex + 1) % agentActivities.length;
			}, 2000);
			return () => clearInterval(interval);
		}
	});
</script>

<div class="w-full h-full overflow-auto" style="background: linear-gradient(180deg, #0F172A 0%, #1E293B 50%, #334155 100%);">
	<div class="min-h-full p-8 flex flex-col items-center justify-center">
		<!-- City skyline background -->
		<div class="fixed inset-0 pointer-events-none overflow-hidden opacity-20">
			<svg class="w-full h-full" preserveAspectRatio="xMidYMax slice" viewBox="0 0 1200 400">
				<rect x="50" y="200" width="60" height="200" fill="#1E293B"/>
				<rect x="130" y="150" width="40" height="250" fill="#1E293B"/>
				<rect x="200" y="180" width="80" height="220" fill="#1E293B"/>
				<rect x="320" y="220" width="50" height="180" fill="#1E293B"/>
				<rect x="900" y="160" width="70" height="240" fill="#1E293B"/>
				<rect x="1000" y="200" width="45" height="200" fill="#1E293B"/>
				<rect x="1080" y="140" width="60" height="260" fill="#1E293B"/>
			</svg>
		</div>

		<!-- Stats HUD -->
		<div class="relative z-10 mb-8 flex items-center gap-6">
			<div class="bg-slate-800/90 backdrop-blur border border-slate-600 rounded-xl px-6 py-3 flex items-center gap-4">
				<div class="flex items-center gap-2">
					<svg class="w-5 h-5 text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5" />
					</svg>
					<span class="text-white font-semibold">BusinessOS Tower</span>
				</div>
				<div class="h-4 w-px bg-slate-600"></div>
				<div class="text-sm text-slate-400">
					<span class="text-white font-medium">{floorCount}</span> Floors
				</div>
				<div class="text-sm text-slate-400">
					<span class="text-white font-medium">{totalRooms}</span> Rooms
				</div>
			</div>
			{#if activeNodeId}
				{@const activeNode = flattenWithDepth(nodes).find(n => n.node.id === activeNodeId)?.node}
				{#if activeNode}
					<div class="bg-blue-500/20 border border-blue-400/50 rounded-xl px-4 py-2 flex items-center gap-3">
						<!-- Animated agent icon -->
						<div class="agent-icon">
							<div class="agent-head"></div>
							<div class="agent-body"></div>
						</div>
						<div class="flex flex-col">
							<span class="text-sm text-blue-300">OSA Agent in: <span class="font-medium text-blue-200">{activeNode.name}</span></span>
							<span class="text-xs text-blue-400/70 animate-pulse">{agentActivities[activityIndex]}</span>
						</div>
					</div>
				{/if}
			{/if}
		</div>

		<!-- Building -->
		<div class="relative z-10" style="perspective: 1000px;">
			<div class="building-container">
				<!-- Roof -->
				<div class="relative mx-auto" style="width: 560px;">
					<div class="relative h-16 bg-gradient-to-b from-slate-600 to-slate-700 rounded-t-lg border-2 border-slate-500 border-b-0">
						<!-- Antenna -->
						<div class="absolute left-1/2 -translate-x-1/2 -top-12">
							<div class="w-1 h-10 bg-slate-500"></div>
							<div class="absolute -top-2 left-1/2 -translate-x-1/2 w-3 h-3 rounded-full bg-red-500 animate-pulse" style="box-shadow: 0 0 12px rgba(239, 68, 68, 0.8);"></div>
						</div>
						<!-- Sign -->
						<div class="absolute inset-x-0 bottom-2 flex justify-center">
							<div class="px-4 py-1 bg-slate-800 rounded border border-slate-600">
								<span class="text-xs font-bold tracking-widest text-amber-400">BUSINESSOS HQ</span>
							</div>
						</div>
					</div>
				</div>

				<!-- Floors -->
				<div class="relative mx-auto border-x-4 border-slate-600" style="width: 560px; background: linear-gradient(90deg, #374151 0%, #475569 50%, #374151 100%);">
					{#each floors() as floor, floorIdx}
						<div class="relative border-b-2 border-slate-600 {floorIdx === 0 ? 'border-t-2' : ''}" style="min-height: 140px;">
							<!-- Floor number indicator -->
							<div class="absolute left-0 top-0 bottom-0 w-14 bg-slate-700 border-r-2 border-slate-600 flex flex-col items-center justify-center">
								<span class="text-2xl font-bold text-slate-400">{floor.level + 1}</span>
								<span class="text-[10px] text-slate-500 uppercase">Floor</span>
							</div>

							<!-- Floor content -->
							<div class="ml-14 mr-12 p-3 flex flex-wrap gap-3 min-h-[136px]">
								{#each floor.rooms as room, roomIdx}
									{@const style = getStyle(room.type)}
									{@const isActive = room.id === activeNodeId}
									{@const isSelected = room.id === selectedId}
									{@const healthColor = getHealthColor(room.health)}

									<!-- Room/Office -->
									<button
										onclick={() => handleClick(room)}
										ondblclick={() => handleDblClick(room)}
										class="room-button relative group transition-all duration-300 focus:outline-none"
										style="
											width: 110px;
											height: 120px;
											{isSelected ? 'transform: scale(1.05);' : ''}
										"
									>
										<!-- Room frame -->
										<div
											class="absolute inset-0 rounded-lg border-2 transition-all duration-300 overflow-hidden"
											style="
												background: linear-gradient(180deg, #1F2937 0%, #111827 100%);
												border-color: {isActive ? style.window : '#4B5563'};
												{isActive ? `box-shadow: 0 0 25px ${style.glow}, inset 0 0 25px ${style.glow};` : ''}
												{isSelected ? 'box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.5);' : ''}
											"
										>
											<!-- Window area -->
											<div
												class="absolute inset-x-2 top-2 bottom-8 rounded transition-all duration-500"
												style="
													background: {isActive
														? `linear-gradient(180deg, ${style.window}50 0%, ${style.window}20 100%)`
														: 'linear-gradient(180deg, #374151 0%, #1F2937 100%)'
													};
													{isActive ? `box-shadow: inset 0 0 20px ${style.glow};` : ''}
												"
											>
												<!-- Window panes -->
												<div class="absolute inset-1 grid grid-cols-2 gap-1 {isActive ? 'opacity-20' : 'opacity-30'}">
													<div class="bg-slate-500 rounded-sm"></div>
													<div class="bg-slate-500 rounded-sm"></div>
												</div>

												<!-- Agent inside room when active -->
												{#if isActive}
													<div class="absolute inset-0 flex items-center justify-center">
														<div class="pixel-agent">
															<!-- Agent at desk -->
															<div class="desk-scene">
																<div class="pixel-desk"></div>
																<div class="pixel-computer">
																	<div class="computer-screen"></div>
																</div>
																<div class="pixel-person">
																	<div class="person-head"></div>
																	<div class="person-body"></div>
																	<div class="person-arm typing"></div>
																</div>
																<div class="pixel-chair"></div>
															</div>
														</div>
													</div>
													<!-- Glow effect -->
													<div class="absolute inset-0 animate-pulse rounded" style="background: {style.glow}; opacity: 0.15;"></div>
												{/if}
											</div>

											<!-- Room number -->
											<div
												class="absolute top-1 left-1 px-1.5 py-0.5 rounded text-[10px] font-bold text-white z-10"
												style="background: {style.window};"
											>
												{floor.level + 1}{String(roomIdx + 1).padStart(2, '0')}
											</div>

											<!-- Health indicator -->
											<div class="absolute top-1.5 right-1.5 z-10">
												<div
													class="w-2.5 h-2.5 rounded-full {room.health === 'critical' ? 'animate-ping' : room.health === 'needs_attention' ? 'animate-pulse' : ''}"
													style="background: {healthColor}; box-shadow: 0 0 8px {healthColor};"
												></div>
											</div>

											<!-- Room name -->
											<div class="absolute bottom-1 inset-x-1 z-10">
												<div class="px-1 py-0.5 bg-slate-900/90 rounded text-center">
													<span class="text-[9px] font-medium text-slate-300 line-clamp-1">{room.name}</span>
												</div>
											</div>

											<!-- Active badge -->
											{#if isActive}
												<div class="absolute -bottom-3 left-1/2 -translate-x-1/2 z-20">
													<div
														class="px-2 py-0.5 rounded-full text-[8px] font-bold text-white flex items-center gap-1 whitespace-nowrap"
														style="background: {style.window}; box-shadow: 0 0 12px {style.glow};"
													>
														<div class="w-1.5 h-1.5 bg-white rounded-full animate-pulse"></div>
														OSA WORKING
													</div>
												</div>
											{/if}

											<!-- Children count -->
											{#if room.children && room.children.length > 0}
												<div class="absolute bottom-7 right-1 w-4 h-4 rounded-full bg-slate-600 border border-slate-500 flex items-center justify-center z-10">
													<span class="text-[8px] font-bold text-white">{room.children.length}</span>
												</div>
											{/if}
										</div>

										<!-- Hover tooltip -->
										<div class="absolute -top-14 left-1/2 -translate-x-1/2 opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none z-30">
											<div class="px-3 py-2 bg-slate-900 border border-slate-600 rounded-lg shadow-xl whitespace-nowrap">
												<p class="text-xs font-medium text-white">{room.name}</p>
												<p class="text-[10px] text-slate-400 capitalize">{room.type} - {room.health.replace('_', ' ')}</p>
												{#if isActive}
													<p class="text-[10px] text-blue-400 mt-1">Agent is working here</p>
												{/if}
											</div>
										</div>
									</button>
								{/each}

								<!-- Add room slot -->
								<button
									onclick={() => handleAddRoom(floor.level)}
									class="btn-pill btn-pill-ghost relative group flex items-center justify-center border-2 border-dashed border-slate-600 hover:border-slate-500 hover:bg-slate-800/50"
									style="width: 110px; height: 120px;"
								>
									<div class="flex flex-col items-center gap-1 text-slate-500 group-hover:text-slate-400 transition-colors">
										<svg class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
										</svg>
										<span class="text-[10px] font-medium">Add Room</span>
									</div>
								</button>
							</div>

							<!-- Elevator shaft -->
							<div class="absolute right-0 top-0 bottom-0 w-12 bg-slate-700 border-l-2 border-slate-600 flex items-center justify-center">
								<div class="w-8 h-14 bg-slate-600 rounded border border-slate-500 flex items-center justify-center relative overflow-hidden">
									<!-- Elevator car animation -->
									<div class="elevator-car absolute w-6 h-8 bg-slate-500 rounded-sm border border-slate-400">
										<div class="absolute inset-x-1 top-1 h-4 bg-slate-400 rounded-sm opacity-50"></div>
									</div>
								</div>
							</div>
						</div>
					{/each}

					<!-- Empty state -->
					{#if floors().length === 0}
						<div class="flex flex-col items-center justify-center py-16 text-slate-400">
							<svg class="w-16 h-16 mb-4 text-slate-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5" />
							</svg>
							<p class="text-lg font-medium mb-1">Empty Building</p>
							<p class="text-sm text-slate-500">Create your first node to add a room</p>
						</div>
					{/if}
				</div>

				<!-- Foundation -->
				<div class="relative mx-auto" style="width: 580px;">
					<div class="h-10 bg-gradient-to-b from-slate-700 to-slate-800 rounded-b-lg border-2 border-slate-600 border-t-0 flex items-center justify-center gap-6">
						<div class="w-12 h-5 bg-slate-600 rounded-sm"></div>
						<div class="w-12 h-5 bg-slate-600 rounded-sm"></div>
						<div class="w-12 h-5 bg-slate-600 rounded-sm"></div>
					</div>
				</div>

				<!-- Ground -->
				<div class="relative mx-auto" style="width: 640px;">
					<div class="h-8 bg-gradient-to-b from-emerald-800 to-emerald-900 rounded-b-xl flex items-center justify-center gap-8">
						<!-- Trees/bushes -->
						<div class="tree"></div>
						<div class="bush"></div>
						<div class="tree"></div>
						<div class="bush"></div>
						<div class="tree"></div>
					</div>
				</div>
			</div>
		</div>

		<!-- Legend -->
		<div class="relative z-10 mt-8 bg-slate-800/80 backdrop-blur border border-slate-600 rounded-xl p-4 max-w-2xl">
			<div class="flex flex-wrap items-center justify-center gap-4 text-xs">
				{#each Object.entries(typeStyles) as [type, style]}
					<div class="flex items-center gap-2">
						<div class="w-4 h-4 rounded" style="background: {style.window};"></div>
						<span class="text-slate-400 capitalize">{type}</span>
					</div>
				{/each}
				<div class="h-4 w-px bg-slate-600"></div>
				<div class="flex items-center gap-2">
					<div class="w-2.5 h-2.5 rounded-full bg-green-500"></div>
					<span class="text-slate-400">Healthy</span>
				</div>
				<div class="flex items-center gap-2">
					<div class="w-2.5 h-2.5 rounded-full bg-yellow-500"></div>
					<span class="text-slate-400">Attention</span>
				</div>
				<div class="flex items-center gap-2">
					<div class="w-2.5 h-2.5 rounded-full bg-red-500"></div>
					<span class="text-slate-400">Critical</span>
				</div>
			</div>
			<p class="text-center text-[10px] text-slate-500 mt-2">Click to select | Double-click to open | Glowing room = OSA Agent is working there</p>
		</div>
	</div>
</div>

<style>
	.building-container {
		transform: rotateX(2deg);
		transform-style: preserve-3d;
	}

	/* Agent icon in HUD */
	.agent-icon {
		width: 20px;
		height: 24px;
		position: relative;
	}

	.agent-head {
		width: 10px;
		height: 10px;
		background: #FCD34D;
		border-radius: 50%;
		position: absolute;
		top: 0;
		left: 5px;
		animation: bob 1s ease-in-out infinite;
	}

	.agent-body {
		width: 14px;
		height: 12px;
		background: #3B82F6;
		border-radius: 4px 4px 2px 2px;
		position: absolute;
		bottom: 0;
		left: 3px;
	}

	@keyframes bob {
		0%, 100% { transform: translateY(0); }
		50% { transform: translateY(-2px); }
	}

	/* Pixel agent in room */
	.pixel-agent {
		width: 100%;
		height: 100%;
		display: flex;
		align-items: flex-end;
		justify-content: center;
		padding-bottom: 4px;
	}

	.desk-scene {
		position: relative;
		width: 50px;
		height: 35px;
	}

	.pixel-desk {
		position: absolute;
		bottom: 0;
		left: 5px;
		width: 40px;
		height: 8px;
		background: #78350F;
		border-radius: 1px;
	}

	.pixel-computer {
		position: absolute;
		bottom: 8px;
		left: 18px;
		width: 14px;
		height: 12px;
		background: #1F2937;
		border-radius: 1px;
	}

	.computer-screen {
		position: absolute;
		inset: 2px;
		background: #60A5FA;
		animation: screenFlicker 0.5s infinite;
	}

	@keyframes screenFlicker {
		0%, 100% { opacity: 1; }
		50% { opacity: 0.8; }
	}

	.pixel-person {
		position: absolute;
		bottom: 8px;
		left: 20px;
	}

	.person-head {
		width: 8px;
		height: 8px;
		background: #FCD34D;
		border-radius: 50%;
		position: absolute;
		bottom: 14px;
		left: -1px;
	}

	.person-body {
		width: 10px;
		height: 10px;
		background: #3B82F6;
		border-radius: 2px;
		position: absolute;
		bottom: 4px;
		left: -2px;
	}

	.person-arm {
		width: 10px;
		height: 3px;
		background: #3B82F6;
		position: absolute;
		bottom: 10px;
		left: -8px;
		border-radius: 1px;
		transform-origin: right center;
	}

	.person-arm.typing {
		animation: typing 0.3s ease-in-out infinite;
	}

	@keyframes typing {
		0%, 100% { transform: rotate(-5deg); }
		50% { transform: rotate(5deg); }
	}

	.pixel-chair {
		position: absolute;
		bottom: 0;
		left: 15px;
		width: 12px;
		height: 10px;
		background: #374151;
		border-radius: 2px 2px 0 0;
	}

	/* Trees and bushes */
	.tree {
		width: 12px;
		height: 16px;
		position: relative;
	}

	.tree::before {
		content: '';
		position: absolute;
		bottom: 4px;
		left: 4px;
		width: 4px;
		height: 6px;
		background: #78350F;
	}

	.tree::after {
		content: '';
		position: absolute;
		top: 0;
		left: 0;
		width: 0;
		height: 0;
		border-left: 6px solid transparent;
		border-right: 6px solid transparent;
		border-bottom: 12px solid #166534;
	}

	.bush {
		width: 16px;
		height: 10px;
		background: #166534;
		border-radius: 50% 50% 40% 40%;
	}

	/* Elevator animation */
	.elevator-car {
		animation: elevatorMove 4s ease-in-out infinite;
	}

	@keyframes elevatorMove {
		0%, 100% { transform: translateY(8px); }
		50% { transform: translateY(-8px); }
	}

	/* Room hover effect */
	.room-button:hover {
		transform: translateY(-2px);
	}
</style>
