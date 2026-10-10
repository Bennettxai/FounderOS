<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import * as THREE from 'three';
  import { OrbitControls } from 'three/examples/jsm/controls/OrbitControls.js';
  // d3-force-3d exposes the same API as d3-force but produces x / y / z.
  // @ts-expect-error — package ships its own types as any
  import { forceSimulation, forceLink, forceManyBody, forceCenter, forceCollide } from 'd3-force-3d';
  import { loadGraph, type GNode, type GEdge } from './graph';
  import { theme } from '$lib/theme';

  interface Props {
    onSelect?: (node: GNode) => void;
  }
  let { onSelect }: Props = $props();

  // ── Palette ──────────────────────────────────────────────────────────
  const NODE_COLOR: Record<string, number> = {
    org: 0x8b5cf6,
    person: 0x3b82f6,
    operation: 0xf97316,
    project: 0x22c55e,
    transcript: 0x3b82f6,
    note: 0x9ca3af,
    spec: 0x8b5cf6,
    plan: 0x22c55e,
    decision_log: 0xeab308,
    product: 0x22c55e,
    concept: 0xeab308,
    default: 0x6b7280
  };

  const LINK_COLOR = 0x2a2f38;
  const PULSE_COLOR = 0x7ea8ff;
  const MAX_PULSES = 5;
  const PULSE_DURATION_MS = 2800;

  function nodeColor(n: GNode): number {
    return NODE_COLOR[n.sub] ?? NODE_COLOR.default;
  }

  function nodeSize(n: GNode): number {
    if (n.kind === 'node') return 4.5 + Math.min(n.connections * 0.3, 4);
    if (n.kind === 'entity') return 2.2 + Math.min(n.connections * 0.2, 2.6);
    return 2.4;
  }

  // ── Types carried into the sim ───────────────────────────────────────
  type D3Node = GNode & {
    x?: number; y?: number; z?: number;
    vx?: number; vy?: number; vz?: number;
    fx?: number | null; fy?: number | null; fz?: number | null;
    mesh?: THREE.Mesh;
    size: number;
  };
  type D3Edge = {
    source: D3Node | string;
    target: D3Node | string;
    relation: string;
    line?: THREE.Line;
  };

  // ── State ────────────────────────────────────────────────────────────
  let container = $state<HTMLDivElement | null>(null);
  let loading = $state(true);
  let error = $state<string | null>(null);
  let stats = $state({ nodes: 0, edges: 0 });
  let selected = $state<GNode | null>(null);
  let hoveredLabel = $state<{ text: string; x: number; y: number } | null>(null);

  // ── three.js handles ─────────────────────────────────────────────────
  let scene: THREE.Scene;
  let camera: THREE.PerspectiveCamera;
  let renderer: THREE.WebGLRenderer;
  let controls: OrbitControls;
  let raycaster: THREE.Raycaster;
  let mouse = new THREE.Vector2();
  let animationId = 0;
  let ro: ResizeObserver | null = null;

  // Sim data
  let d3nodes: D3Node[] = [];
  let d3edges: D3Edge[] = [];
  let sim: any = null;

  // Pulse particles (small glowing spheres traversing random edges)
  interface Pulse {
    mesh: THREE.Mesh;
    edge: D3Edge;
    startedAt: number;
  }
  let pulses: Pulse[] = [];
  let lastPulseSpawn = 0;

  // Node-drag state — the d3-force SOP is: pin a node while dragging so
  // it follows the pointer, let the sim rearrange neighbors around it,
  // then unpin on release.
  let dragTarget: D3Node | null = null;
  let dragPlane = new THREE.Plane();
  let dragIntersection = new THREE.Vector3();
  let dragOffset = new THREE.Vector3();

  // Theme-reactive edge + pulse colours so the graph reads in both modes.
  let darkMode = $derived($theme === 'dark');
  let linkColor = $derived(darkMode ? 0x3a4250 : 0xbac4d1);
  let linkOpacity = $derived(darkMode ? 0.55 : 0.75);
  let pulseColor = $derived(darkMode ? 0x7ea8ff : 0x1e5cff);

  $effect(() => {
    // Re-read reactive colors so Svelte tracks them.
    const lc = linkColor;
    const lo = linkOpacity;
    const pc = pulseColor;
    // Propagate to live materials (scene may not exist yet on first pass).
    scene?.traverse((obj) => {
      if (obj instanceof THREE.Line) {
        const mat = obj.material as THREE.LineBasicMaterial;
        mat.color.setHex(lc);
        mat.opacity = lo;
      }
    });
    for (const p of pulses) {
      (p.mesh.material as THREE.MeshBasicMaterial).color.setHex(pc);
    }
  });

  onMount(async () => {
    if (!container) return;
    try {
      initThree();
      const data = await loadGraph();
      stats = { nodes: data.nodes.length, edges: data.edges.length };
      buildScene(data.nodes, data.edges);
      loading = false;
      animate();

      ro = new ResizeObserver(() => resize());
      ro.observe(container);
    } catch (e) {
      error = (e as Error).message;
      loading = false;
    }
  });

  onDestroy(() => {
    cancelAnimationFrame(animationId);
    sim?.stop();
    ro?.disconnect();
    renderer?.dispose();
    // Dispose scene materials + geometries
    scene?.traverse((obj) => {
      if (obj instanceof THREE.Mesh || obj instanceof THREE.Line) {
        obj.geometry?.dispose();
        const m = obj.material as THREE.Material | THREE.Material[];
        (Array.isArray(m) ? m : [m]).forEach((mat) => mat?.dispose());
      }
    });
    if (container && renderer) container.removeChild(renderer.domElement);
  });

  // ── three.js init ────────────────────────────────────────────────────
  function initThree() {
    if (!container) return;
    const w = container.clientWidth;
    const h = container.clientHeight;

    scene = new THREE.Scene();
    scene.background = null; // page background shows through — respects theme
    // No fog — it was hiding far nodes. Depth perception comes from the
    // rotation + emissive glow, which is enough at this scale.

    camera = new THREE.PerspectiveCamera(55, w / h, 0.1, 4000);
    camera.position.set(0, 0, 90);

    renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true });
    renderer.setSize(w, h);
    renderer.setPixelRatio(window.devicePixelRatio);
    renderer.setClearColor(0x000000, 0);
    container.appendChild(renderer.domElement);

    controls = new OrbitControls(camera, renderer.domElement);
    controls.enableDamping = true;
    controls.dampingFactor = 0.1;
    controls.rotateSpeed = 0.6;
    controls.zoomSpeed = 1.1;
    controls.enablePan = true;
    controls.panSpeed = 1.0;
    controls.screenSpacePanning = true;
    // Make the three actions obvious via the default mouse buttons:
    //   LEFT: rotate · RIGHT: pan · WHEEL: zoom
    // Shift+LEFT also pans so trackpad users without a right-click still get pan.
    controls.mouseButtons = {
      LEFT: THREE.MOUSE.ROTATE,
      MIDDLE: THREE.MOUSE.DOLLY,
      RIGHT: THREE.MOUSE.PAN
    };
    controls.touches = {
      ONE: THREE.TOUCH.ROTATE,
      TWO: THREE.TOUCH.DOLLY_PAN
    };
    controls.minDistance = 20;
    controls.maxDistance = 600;

    scene.add(new THREE.AmbientLight(0xffffff, 0.7));
    const dir = new THREE.DirectionalLight(0xffffff, 0.95);
    dir.position.set(200, 200, 300);
    scene.add(dir);

    raycaster = new THREE.Raycaster();

    renderer.domElement.addEventListener('pointermove', onPointerMove);
    renderer.domElement.addEventListener('pointerdown', onPointerDown);
    renderer.domElement.addEventListener('pointerup', onPointerUp);
    renderer.domElement.addEventListener('pointerleave', onPointerUp);
  }

  function resize() {
    if (!container || !renderer || !camera) return;
    const w = container.clientWidth;
    const h = container.clientHeight;
    renderer.setSize(w, h);
    camera.aspect = w / h;
    camera.updateProjectionMatrix();
  }

  // ── Build scene ──────────────────────────────────────────────────────
  function buildScene(nodes: GNode[], edges: GEdge[]) {
    d3nodes = nodes.map((n) => ({
      ...n,
      size: nodeSize(n)
    }));

    // Sphere material varies per-node by color; geometry is shared.
    const sphereGeo = new THREE.SphereGeometry(1, 20, 20);

    for (const n of d3nodes) {
      const color = nodeColor(n);
      const mat = new THREE.MeshStandardMaterial({
        color,
        emissive: color,
        emissiveIntensity: 0.35,
        roughness: 0.35,
        metalness: 0.1
      });
      const mesh = new THREE.Mesh(sphereGeo, mat);
      mesh.scale.setScalar(n.size);
      mesh.userData.node = n;
      scene.add(mesh);
      n.mesh = mesh;
    }

    const byId = new Map(d3nodes.map((n) => [n.id, n]));
    d3edges = edges
      .filter((e) => byId.has(e.source) && byId.has(e.target))
      .map((e) => ({
        source: byId.get(e.source)!,
        target: byId.get(e.target)!,
        relation: e.relation
      }));

    // Each edge gets its own material so the theme subscriber can retune
    // colors in-place without rebuilding the scene.
    for (const e of d3edges) {
      const lineMat = new THREE.LineBasicMaterial({
        color: linkColor,
        transparent: true,
        opacity: linkOpacity
      });
      const geo = new THREE.BufferGeometry().setFromPoints([
        new THREE.Vector3(),
        new THREE.Vector3()
      ]);
      const line = new THREE.Line(geo, lineMat);
      scene.add(line);
      e.line = line;
    }

    // 3D force simulation — canonical forces, 3 dimensions.
    sim = forceSimulation(d3nodes, 3)
      .alphaDecay(0.03)
      .alphaMin(0.002)
      .velocityDecay(0.4)
      .force(
        'link',
        forceLink(d3edges)
          .id((d: any) => d.id)
          .distance((e: any) => (e.relation === 'parent' ? 18 : e.relation === 'contains' ? 12 : 22))
          .strength(0.35)
      )
      .force('charge', forceManyBody().strength(-70).distanceMax(140))
      .force('center', forceCenter().strength(0.06))
      .force('collide', forceCollide().radius((d: any) => d.size + 0.6).iterations(2));
  }

  // ── Pulse particles ──────────────────────────────────────────────────
  function spawnPulse(now: number) {
    if (d3edges.length === 0 || pulses.length >= MAX_PULSES) return;
    const edge = d3edges[Math.floor(Math.random() * d3edges.length)];
    const geo = new THREE.SphereGeometry(0.7, 10, 10);
    const mat = new THREE.MeshBasicMaterial({ color: PULSE_COLOR, transparent: true, opacity: 0.95 });
    const mesh = new THREE.Mesh(geo, mat);
    scene.add(mesh);
    pulses.push({ mesh, edge, startedAt: now });
  }

  function updatePulses(now: number) {
    for (let i = pulses.length - 1; i >= 0; i--) {
      const p = pulses[i];
      const src = p.edge.source as D3Node;
      const tgt = p.edge.target as D3Node;
      const t = (now - p.startedAt) / PULSE_DURATION_MS;

      if (t >= 1) {
        scene.remove(p.mesh);
        (p.mesh.material as THREE.Material).dispose();
        p.mesh.geometry.dispose();
        pulses.splice(i, 1);
        continue;
      }

      // Ease in/out so the pulse has a life to it.
      const eased = t < 0.5 ? 2 * t * t : 1 - Math.pow(-2 * t + 2, 2) / 2;
      p.mesh.position.set(
        (src.x ?? 0) + ((tgt.x ?? 0) - (src.x ?? 0)) * eased,
        (src.y ?? 0) + ((tgt.y ?? 0) - (src.y ?? 0)) * eased,
        (src.z ?? 0) + ((tgt.z ?? 0) - (src.z ?? 0)) * eased
      );
      // Fade in / out so it feels like an electron pulse rather than a ball.
      const fade = Math.sin(t * Math.PI);
      (p.mesh.material as THREE.MeshBasicMaterial).opacity = 0.4 + 0.6 * fade;
    }
  }

  // ── Animation loop ───────────────────────────────────────────────────
  function animate() {
    animationId = requestAnimationFrame(animate);
    const now = performance.now();

    // Spawn at most one new pulse every ~900ms so it feels rhythmic, not flood.
    if (now - lastPulseSpawn > 900) {
      spawnPulse(now);
      lastPulseSpawn = now;
    }
    updatePulses(now);

    // Sync meshes to sim positions.
    for (const n of d3nodes) {
      n.mesh?.position.set(n.x ?? 0, n.y ?? 0, n.z ?? 0);
    }
    for (const e of d3edges) {
      if (!e.line) continue;
      const s = e.source as D3Node;
      const t = e.target as D3Node;
      const positions = e.line.geometry.getAttribute('position') as THREE.BufferAttribute;
      positions.setXYZ(0, s.x ?? 0, s.y ?? 0, s.z ?? 0);
      positions.setXYZ(1, t.x ?? 0, t.y ?? 0, t.z ?? 0);
      positions.needsUpdate = true;
    }

    // Pulse the selected node's emissive intensity for a subtle heartbeat.
    for (const n of d3nodes) {
      if (!n.mesh) continue;
      const mat = n.mesh.material as THREE.MeshStandardMaterial;
      if (selected?.id === n.id) {
        mat.emissiveIntensity = 0.55 + 0.25 * Math.sin(now * 0.004);
      } else {
        mat.emissiveIntensity = 0.35;
      }
    }

    controls.update();
    renderer.render(scene, camera);
  }

  // ── Interaction (d3-force drag SOP) ──────────────────────────────────
  // Down:  raycast → if a node was hit, pin it and start dragging;
  //                  else let OrbitControls rotate/pan/dolly.
  // Move:  if dragging, unproject pointer onto a plane through the node
  //        and update fx/fy/fz; the sim keeps running so neighbors
  //        rearrange around the pinned node.
  // Up:    unpin (fx/fy/fz = null), re-enable controls.
  // Click (down→up without drift): treat as selection.
  let downPoint = { x: 0, y: 0 };
  let didDrift = false;

  function updateMouse(ev: PointerEvent) {
    if (!container) return;
    const rect = container.getBoundingClientRect();
    mouse.x = ((ev.clientX - rect.left) / rect.width) * 2 - 1;
    mouse.y = -((ev.clientY - rect.top) / rect.height) * 2 + 1;
  }

  function pickNode(): D3Node | null {
    raycaster.setFromCamera(mouse, camera);
    const meshes = d3nodes.map((n) => n.mesh!).filter(Boolean);
    const hit = raycaster.intersectObjects(meshes, false)[0];
    return hit ? (hit.object.userData.node as D3Node) : null;
  }

  function onPointerMove(ev: PointerEvent) {
    updateMouse(ev);

    // Dragging a node — update its pinned coords by projecting the
    // pointer ray onto a plane passing through the node, perpendicular
    // to the camera direction.
    if (dragTarget) {
      if (Math.abs(ev.clientX - downPoint.x) + Math.abs(ev.clientY - downPoint.y) > 3) {
        didDrift = true;
      }
      raycaster.setFromCamera(mouse, camera);
      if (raycaster.ray.intersectPlane(dragPlane, dragIntersection)) {
        const next = dragIntersection.sub(dragOffset);
        dragTarget.fx = next.x;
        dragTarget.fy = next.y;
        dragTarget.fz = next.z;
      }
      return;
    }

    // Not dragging — just handle hover tooltip + cursor.
    if (!container) return;
    const rect = container.getBoundingClientRect();
    const n = pickNode();
    if (n) {
      hoveredLabel = {
        text: n.label,
        x: ev.clientX - rect.left + 12,
        y: ev.clientY - rect.top + 12
      };
      renderer.domElement.style.cursor = 'grab';
    } else {
      hoveredLabel = null;
      renderer.domElement.style.cursor = 'grab';
    }
  }

  function onPointerDown(ev: PointerEvent) {
    // Only handle left-button drags for nodes. Right-click and middle
    // stay with OrbitControls (pan / dolly).
    if (ev.button !== 0) return;

    updateMouse(ev);
    downPoint = { x: ev.clientX, y: ev.clientY };
    didDrift = false;

    const n = pickNode();
    if (!n) return; // empty space → OrbitControls rotates

    // Start a drag. Pin the node so the sim treats it as fixed.
    dragTarget = n;
    controls.enabled = false; // stop camera from rotating during drag
    renderer.domElement.style.cursor = 'grabbing';

    // Build a plane through the node, perpendicular to camera forward.
    const camForward = new THREE.Vector3();
    camera.getWorldDirection(camForward);
    const nodePos = new THREE.Vector3(n.x ?? 0, n.y ?? 0, n.z ?? 0);
    dragPlane.setFromNormalAndCoplanarPoint(camForward, nodePos);

    // Capture initial offset so the node doesn't jump under the pointer.
    raycaster.setFromCamera(mouse, camera);
    if (raycaster.ray.intersectPlane(dragPlane, dragIntersection)) {
      dragOffset.copy(dragIntersection).sub(nodePos);
    } else {
      dragOffset.set(0, 0, 0);
    }

    n.fx = n.x ?? 0;
    n.fy = n.y ?? 0;
    n.fz = n.z ?? 0;

    // Gentle reheat so neighbours rearrange around the pinned node.
    sim?.alphaTarget(0.2).restart();
  }

  function onPointerUp(ev: PointerEvent) {
    if (dragTarget) {
      // Click without drift = selection. Drift = drag completed.
      if (!didDrift) {
        selected = dragTarget;
        onSelect?.(dragTarget);
      }
      // Release the pin so the sim can reclaim the node.
      dragTarget.fx = null;
      dragTarget.fy = null;
      dragTarget.fz = null;
      dragTarget = null;
      sim?.alphaTarget(0);
      controls.enabled = true;
      renderer.domElement.style.cursor = 'grab';
    }
  }

  function clearSelection() {
    selected = null;
  }

  function resetView() {
    if (!camera) return;
    camera.position.set(0, 0, 220);
    camera.lookAt(0, 0, 0);
    controls?.reset();
  }
</script>

<div class="pg">
  <div class="pg__canvas" bind:this={container}></div>

  {#if loading}
    <div class="pg__overlay">Loading graph…</div>
  {:else if error}
    <div class="pg__overlay pg__overlay--error">{error}</div>
  {/if}

  {#if hoveredLabel}
    <div class="pg__tooltip" style="left: {hoveredLabel.x}px; top: {hoveredLabel.y}px">
      {hoveredLabel.text}
    </div>
  {/if}

  <div class="pg__legend">
    <div class="pg__stats">
      <strong>{stats.nodes}</strong> nodes · <strong>{stats.edges}</strong> edges
    </div>
    <div class="pg__kinds">
      <span class="pg__kind"><span class="dot" style="background: #3b82f6"></span> person</span>
      <span class="pg__kind"><span class="dot" style="background: #8b5cf6"></span> org</span>
      <span class="pg__kind"><span class="dot" style="background: #22c55e"></span> product / project</span>
      <span class="pg__kind"><span class="dot" style="background: #eab308"></span> concept</span>
      <span class="pg__kind"><span class="dot" style="background: #f97316"></span> operation</span>
    </div>
    <div class="pg__hint">
      drag empty space to rotate · right-click drag to pan · scroll to zoom ·
      drag a node to reposition · click a node to open
    </div>
  </div>

  <div class="pg__controls">
    <button onclick={resetView} title="Reset view">⟳</button>
  </div>

  {#if selected}
    <aside class="pg__panel">
      <header>
        <span class="pg__pill">{selected.kind}</span>
        <span class="pg__pill pg__pill--sub">{selected.sub}</span>
        <button class="pg__close" onclick={clearSelection} aria-label="Close">×</button>
      </header>
      <h3>{selected.label}</h3>
      {#if selected.kind === 'node'}
        <p class="muted">Workspace node · {selected.connections} signal{selected.connections === 1 ? '' : 's'}</p>
      {:else if selected.kind === 'signal'}
        <p class="muted">Signal in <code>{selected.node_slug}</code></p>
      {:else if selected.kind === 'entity'}
        <p class="muted">Entity · {selected.connections} context{selected.connections === 1 ? '' : 's'}</p>
      {/if}
    </aside>
  {/if}
</div>

<style>
  .pg {
    position: relative;
    width: 100%;
    height: 100%;
    overflow: hidden;
  }
  .pg__canvas {
    position: absolute;
    inset: 0;
    cursor: grab;
  }
  .pg__canvas :global(canvas) {
    display: block;
  }
  .pg__overlay {
    position: absolute;
    inset: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--muted, #888);
    background: rgba(11, 13, 16, 0.7);
    pointer-events: none;
  }
  .pg__overlay--error { color: #f88; }

  .pg__tooltip {
    position: absolute;
    padding: 3px 8px;
    background: rgba(14, 17, 22, 0.9);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 4px;
    color: #ddd;
    font-size: 0.75rem;
    pointer-events: none;
    white-space: nowrap;
    z-index: 5;
  }

  .pg__legend {
    position: absolute;
    top: 1rem;
    left: 1rem;
    padding: 0.75rem 1rem;
    background: rgba(14, 17, 22, 0.86);
    backdrop-filter: blur(8px);
    border: 1px solid rgba(255, 255, 255, 0.06);
    border-radius: 10px;
    color: #bbb;
    font-size: 0.78rem;
    max-width: 320px;
    pointer-events: none;
  }
  .pg__stats strong { color: #eee; font-weight: 600; }
  .pg__kinds { display: flex; flex-wrap: wrap; gap: 0.5rem 0.8rem; margin-top: 0.5rem; }
  .pg__kind { display: inline-flex; align-items: center; gap: 5px; font-size: 0.72rem; }
  .pg__hint { margin-top: 0.5rem; color: #555; font-size: 0.7rem; }
  .dot { display: inline-block; width: 8px; height: 8px; border-radius: 50%; }

  .pg__controls { position: absolute; bottom: 1rem; right: 1rem; }
  .pg__controls button {
    width: 36px; height: 36px;
    background: rgba(14, 17, 22, 0.92);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 8px;
    color: #ccc;
    cursor: pointer; font-size: 1rem;
  }
  .pg__controls button:hover { color: #fff; }

  .pg__panel {
    position: absolute;
    top: 1rem;
    right: 1rem;
    width: 320px;
    padding: 1rem 1.25rem;
    background: rgba(14, 17, 22, 0.95);
    backdrop-filter: blur(8px);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 10px;
    color: #ddd;
  }
  .pg__panel header { display: flex; align-items: center; gap: 0.4rem; margin-bottom: 0.5rem; }
  .pg__panel h3 { margin: 0.35rem 0 0.5rem; font-size: 1rem; color: #eee; }
  .pg__pill {
    padding: 2px 8px; background: #143; color: #6f8; border-radius: 999px;
    font-size: 0.68rem; text-transform: uppercase; letter-spacing: 0.5px; font-weight: 600;
  }
  .pg__pill--sub { background: #223; color: #8af; }
  .pg__close {
    margin-left: auto; background: none; border: none; color: #888;
    font-size: 1.3rem; cursor: pointer; padding: 0 0.3rem; line-height: 1;
  }
  .pg__close:hover { color: #ddd; }
  .muted { color: #888; font-size: 0.85rem; margin: 0.25rem 0 0; }
  code {
    font-family: 'SF Mono', Menlo, monospace;
    background: #0e1116; padding: 0 4px; border-radius: 3px; color: #cdf;
  }
</style>
