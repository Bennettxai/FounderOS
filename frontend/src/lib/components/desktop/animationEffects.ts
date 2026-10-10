/**
 * Animation effects for AnimatedBackground.svelte
 * Contains all init and draw functions for each animation type.
 * All functions are pure canvas operations passed a context, canvas, and shared state.
 */

export type AnimationEffect =
  // Basic
  | "none"
  | "particles"
  | "gradient"
  | "pulse"
  | "ripples"
  | "dots"
  | "floatingShapes"
  | "smoke"
  // Nature
  | "aurora"
  | "fireflies"
  | "rain"
  | "snow"
  | "nebula"
  | "waves"
  | "bubbles"
  // Tech
  | "starfield"
  | "matrix"
  | "circuit"
  | "confetti"
  | "geometric"
  | "scanlines"
  | "grid"
  | "warp"
  | "hexgrid"
  | "binary";

export type AnimationIntensity = "subtle" | "medium" | "high";

export interface Particle {
  x: number;
  y: number;
  size: number;
  speedX: number;
  speedY: number;
  opacity: number;
  color: string;
}
export interface Star {
  x: number;
  y: number;
  size: number;
  brightness: number;
  twinkleSpeed: number;
  twinklePhase: number;
}
export interface MatrixDrop {
  x: number;
  y: number;
  speed: number;
  chars: string[];
  length: number;
  opacity: number;
}
export interface Bubble {
  x: number;
  y: number;
  size: number;
  speed: number;
  wobble: number;
  wobbleSpeed: number;
  opacity: number;
  color: string;
}
export interface GeoShape {
  x: number;
  y: number;
  size: number;
  rotation: number;
  rotationSpeed: number;
  speedX: number;
  speedY: number;
  sides: number;
  color: string;
  opacity: number;
}
export interface Firefly {
  x: number;
  y: number;
  size: number;
  speedX: number;
  speedY: number;
  glowPhase: number;
  glowSpeed: number;
  color: string;
}
export interface Raindrop {
  x: number;
  y: number;
  length: number;
  speed: number;
  opacity: number;
}
export interface Snowflake {
  x: number;
  y: number;
  size: number;
  speed: number;
  wobble: number;
  wobbleSpeed: number;
  opacity: number;
}
export interface ConfettiPiece {
  x: number;
  y: number;
  size: number;
  speedY: number;
  speedX: number;
  rotation: number;
  rotationSpeed: number;
  color: string;
  opacity: number;
}
export interface Ripple {
  x: number;
  y: number;
  radius: number;
  maxRadius: number;
  speed: number;
  opacity: number;
  color: string;
}
export interface CircuitNode {
  x: number;
  y: number;
  connections: number[];
  pulsePhase: number;
  pulseSpeed: number;
}
export interface Dot {
  x: number;
  y: number;
  baseSize: number;
  phase: number;
  speed: number;
}
export interface FloatingShape {
  x: number;
  y: number;
  size: number;
  rotation: number;
  rotSpeed: number;
  speedX: number;
  speedY: number;
  type: "square" | "triangle" | "circle";
  color: string;
  opacity: number;
}
export interface SmokeParticle {
  x: number;
  y: number;
  size: number;
  opacity: number;
  speedX: number;
  speedY: number;
  life: number;
}
export interface GridLine {
  pos: number;
  speed: number;
  opacity: number;
}
export interface WarpStar {
  x: number;
  y: number;
  z: number;
  prevX: number;
  prevY: number;
}
export interface HexCell {
  x: number;
  y: number;
  phase: number;
  speed: number;
}

export interface AnimationState {
  particles: Particle[];
  stars: Star[];
  matrixDrops: MatrixDrop[];
  bubbles: Bubble[];
  geoShapes: GeoShape[];
  fireflies: Firefly[];
  raindrops: Raindrop[];
  snowflakes: Snowflake[];
  confetti: ConfettiPiece[];
  ripples: Ripple[];
  circuitNodes: CircuitNode[];
  dots: Dot[];
  floatingShapes: FloatingShape[];
  smokeParticles: SmokeParticle[];
  gridLinesH: GridLine[];
  gridLinesV: GridLine[];
  warpStars: WarpStar[];
  hexCells: HexCell[];
  scanlineOffset: number;
  pulsePhase: number;
  nebulaTime: number;
}

export function createAnimationState(): AnimationState {
  return {
    particles: [],
    stars: [],
    matrixDrops: [],
    bubbles: [],
    geoShapes: [],
    fireflies: [],
    raindrops: [],
    snowflakes: [],
    confetti: [],
    ripples: [],
    circuitNodes: [],
    dots: [],
    floatingShapes: [],
    smokeParticles: [],
    gridLinesH: [],
    gridLinesV: [],
    warpStars: [],
    hexCells: [],
    scanlineOffset: 0,
    pulsePhase: 0,
    nebulaTime: 0,
  };
}

export const intensityConfig: Record<
  AnimationIntensity,
  { particleCount: number; opacity: number; speed: number }
> = {
  subtle: { particleCount: 30, opacity: 0.3, speed: 0.5 },
  medium: { particleCount: 60, opacity: 0.5, speed: 1 },
  high: { particleCount: 100, opacity: 0.7, speed: 1.5 },
};

const matrixChars =
  "アイウエオカキクケコサシスセソタチツテトナニヌネノハヒフヘホマミムメモヤユヨラリルレロワヲン0123456789ABCDEF".split(
    "",
  );

// ===== INIT FUNCTIONS =====

export function initParticles(
  state: AnimationState,
  canvas: HTMLCanvasElement,
  count: number,
  colors: string[],
  speed: number,
) {
  state.particles = [];
  for (let i = 0; i < count; i++) {
    state.particles.push({
      x: Math.random() * canvas.width,
      y: Math.random() * canvas.height,
      size: Math.random() * 3 + 1,
      speedX: (Math.random() - 0.5) * 0.5 * speed,
      speedY: (Math.random() - 0.5) * 0.5 * speed,
      opacity: Math.random() * 0.5 + 0.2,
      color: colors[Math.floor(Math.random() * colors.length)],
    });
  }
}

export function initStars(
  state: AnimationState,
  canvas: HTMLCanvasElement,
  count: number,
) {
  state.stars = [];
  for (let i = 0; i < count; i++) {
    state.stars.push({
      x: Math.random() * canvas.width,
      y: Math.random() * canvas.height,
      size: Math.random() * 2 + 0.5,
      brightness: Math.random(),
      twinkleSpeed: Math.random() * 2 + 1,
      twinklePhase: Math.random() * Math.PI * 2,
    });
  }
}

export function initMatrix(
  state: AnimationState,
  canvas: HTMLCanvasElement,
  count: number,
) {
  state.matrixDrops = [];
  const columns = Math.floor(canvas.width / 20);
  const dropsPerColumn = Math.max(1, Math.floor(count / columns));
  for (let col = 0; col < columns; col++) {
    for (let d = 0; d < dropsPerColumn; d++) {
      const length = Math.floor(Math.random() * 15) + 5;
      const chars: string[] = [];
      for (let i = 0; i < length; i++) {
        chars.push(matrixChars[Math.floor(Math.random() * matrixChars.length)]);
      }
      state.matrixDrops.push({
        x: col * 20 + 10,
        y: Math.random() * canvas.height - canvas.height,
        speed: Math.random() * 2 + 1,
        chars,
        length,
        opacity: Math.random() * 0.5 + 0.3,
      });
    }
  }
}

export function initBubbles(
  state: AnimationState,
  canvas: HTMLCanvasElement,
  count: number,
  colors: string[],
) {
  state.bubbles = [];
  for (let i = 0; i < count; i++) {
    state.bubbles.push({
      x: Math.random() * canvas.width,
      y: canvas.height + Math.random() * 100,
      size: Math.random() * 20 + 10,
      speed: Math.random() * 1.5 + 0.5,
      wobble: Math.random() * Math.PI * 2,
      wobbleSpeed: Math.random() * 0.02 + 0.01,
      opacity: Math.random() * 0.3 + 0.1,
      color: colors[Math.floor(Math.random() * colors.length)],
    });
  }
}

export function initGeometric(
  state: AnimationState,
  canvas: HTMLCanvasElement,
  count: number,
  colors: string[],
) {
  state.geoShapes = [];
  for (let i = 0; i < count; i++) {
    state.geoShapes.push({
      x: Math.random() * canvas.width,
      y: Math.random() * canvas.height,
      size: Math.random() * 40 + 20,
      rotation: Math.random() * Math.PI * 2,
      rotationSpeed: (Math.random() - 0.5) * 0.02,
      speedX: (Math.random() - 0.5) * 0.5,
      speedY: (Math.random() - 0.5) * 0.5,
      sides: Math.floor(Math.random() * 4) + 3,
      color: colors[Math.floor(Math.random() * colors.length)],
      opacity: Math.random() * 0.3 + 0.1,
    });
  }
}

export function initFireflies(
  state: AnimationState,
  canvas: HTMLCanvasElement,
  count: number,
  colors: string[],
) {
  state.fireflies = [];
  for (let i = 0; i < count; i++) {
    state.fireflies.push({
      x: Math.random() * canvas.width,
      y: Math.random() * canvas.height,
      size: Math.random() * 4 + 2,
      speedX: (Math.random() - 0.5) * 0.5,
      speedY: (Math.random() - 0.5) * 0.5,
      glowPhase: Math.random() * Math.PI * 2,
      glowSpeed: Math.random() * 0.03 + 0.01,
      color: colors[Math.floor(Math.random() * colors.length)] || "#ffff00",
    });
  }
}

export function initRain(
  state: AnimationState,
  canvas: HTMLCanvasElement,
  count: number,
) {
  state.raindrops = [];
  for (let i = 0; i < count; i++) {
    state.raindrops.push({
      x: Math.random() * canvas.width,
      y: Math.random() * canvas.height,
      length: Math.random() * 20 + 10,
      speed: Math.random() * 10 + 5,
      opacity: Math.random() * 0.3 + 0.2,
    });
  }
}

export function initSnow(
  state: AnimationState,
  canvas: HTMLCanvasElement,
  count: number,
) {
  state.snowflakes = [];
  for (let i = 0; i < count; i++) {
    state.snowflakes.push({
      x: Math.random() * canvas.width,
      y: Math.random() * canvas.height,
      size: Math.random() * 4 + 1,
      speed: Math.random() * 1 + 0.5,
      wobble: Math.random() * Math.PI * 2,
      wobbleSpeed: Math.random() * 0.02 + 0.01,
      opacity: Math.random() * 0.6 + 0.4,
    });
  }
}

export function initConfetti(
  state: AnimationState,
  canvas: HTMLCanvasElement,
  count: number,
) {
  state.confetti = [];
  const confettiColors = [
    "#ff6b6b",
    "#4ecdc4",
    "#ffe66d",
    "#a855f7",
    "#3b82f6",
    "#22c55e",
  ];
  for (let i = 0; i < count; i++) {
    state.confetti.push({
      x: Math.random() * canvas.width,
      y: Math.random() * canvas.height - canvas.height,
      size: Math.random() * 8 + 4,
      speedY: Math.random() * 2 + 1,
      speedX: (Math.random() - 0.5) * 2,
      rotation: Math.random() * Math.PI * 2,
      rotationSpeed: (Math.random() - 0.5) * 0.2,
      color: confettiColors[Math.floor(Math.random() * confettiColors.length)],
      opacity: Math.random() * 0.5 + 0.5,
    });
  }
}

export function initRipples(
  state: AnimationState,
  canvas: HTMLCanvasElement,
  count: number,
  colors: string[],
) {
  state.ripples = [];
  for (let i = 0; i < count; i++) {
    state.ripples.push({
      x: Math.random() * canvas.width,
      y: Math.random() * canvas.height,
      radius: Math.random() * 20,
      maxRadius: Math.random() * 100 + 50,
      speed: Math.random() * 1 + 0.5,
      opacity: Math.random() * 0.3 + 0.2,
      color: colors[Math.floor(Math.random() * colors.length)],
    });
  }
}

export function initCircuit(
  state: AnimationState,
  canvas: HTMLCanvasElement,
  count: number,
) {
  state.circuitNodes = [];
  const nodeCount = Math.max(count, 20);
  const gridSize = Math.ceil(Math.sqrt(nodeCount));
  const cellWidth = canvas.width / gridSize;
  const cellHeight = canvas.height / gridSize;

  for (let i = 0; i < nodeCount; i++) {
    const gridX = i % gridSize;
    const gridY = Math.floor(i / gridSize);
    state.circuitNodes.push({
      x:
        gridX * cellWidth +
        cellWidth / 2 +
        (Math.random() - 0.5) * cellWidth * 0.3,
      y:
        gridY * cellHeight +
        cellHeight / 2 +
        (Math.random() - 0.5) * cellHeight * 0.3,
      connections: [],
      pulsePhase: Math.random() * Math.PI * 2,
      pulseSpeed: Math.random() * 0.03 + 0.01,
    });
  }

  const maxDist = Math.max(cellWidth, cellHeight) * 1.8;
  state.circuitNodes.forEach((node, index) => {
    const maxConnections = Math.floor(Math.random() * 3) + 2;
    const distances = state.circuitNodes
      .map((other, otherIndex) => ({
        index: otherIndex,
        dist: Math.sqrt((other.x - node.x) ** 2 + (other.y - node.y) ** 2),
      }))
      .filter((d) => d.index !== index && d.dist < maxDist)
      .sort((a, b) => a.dist - b.dist);

    for (let c = 0; c < Math.min(maxConnections, distances.length); c++) {
      if (!node.connections.includes(distances[c].index)) {
        node.connections.push(distances[c].index);
      }
    }
  });
}

export function initDots(
  state: AnimationState,
  canvas: HTMLCanvasElement,
  count: number,
) {
  state.dots = [];
  const cols = Math.ceil(Math.sqrt(count * 1.5));
  const rows = Math.ceil(count / cols);
  const spacingX = canvas.width / cols;
  const spacingY = canvas.height / rows;
  for (let row = 0; row < rows; row++) {
    for (let col = 0; col < cols; col++) {
      state.dots.push({
        x: col * spacingX + spacingX / 2,
        y: row * spacingY + spacingY / 2,
        baseSize: 3,
        phase: Math.random() * Math.PI * 2,
        speed: Math.random() * 0.02 + 0.01,
      });
    }
  }
}

export function initFloatingShapes(
  state: AnimationState,
  canvas: HTMLCanvasElement,
  count: number,
  colors: string[],
) {
  state.floatingShapes = [];
  const shapeTypes: ("square" | "triangle" | "circle")[] = [
    "square",
    "triangle",
    "circle",
  ];
  for (let i = 0; i < count; i++) {
    state.floatingShapes.push({
      x: Math.random() * canvas.width,
      y: Math.random() * canvas.height,
      size: Math.random() * 30 + 15,
      rotation: Math.random() * Math.PI * 2,
      rotSpeed: (Math.random() - 0.5) * 0.02,
      speedX: (Math.random() - 0.5) * 0.5,
      speedY: (Math.random() - 0.5) * 0.5,
      type: shapeTypes[Math.floor(Math.random() * shapeTypes.length)],
      color: colors[Math.floor(Math.random() * colors.length)],
      opacity: Math.random() * 0.3 + 0.1,
    });
  }
}

export function initSmoke(
  state: AnimationState,
  canvas: HTMLCanvasElement,
  count: number,
) {
  state.smokeParticles = [];
  for (let i = 0; i < count; i++) {
    state.smokeParticles.push({
      x: Math.random() * canvas.width,
      y: canvas.height + Math.random() * 50,
      size: Math.random() * 50 + 30,
      opacity: Math.random() * 0.2 + 0.1,
      speedX: (Math.random() - 0.5) * 0.5,
      speedY: -(Math.random() * 0.5 + 0.3),
      life: 1,
    });
  }
}

export function initGrid(
  state: AnimationState,
  canvas: HTMLCanvasElement,
  count: number,
) {
  state.gridLinesH = [];
  state.gridLinesV = [];
  const spacing = Math.max(canvas.width, canvas.height) / count;
  for (let i = 0; i < Math.ceil(canvas.height / spacing) + 1; i++) {
    state.gridLinesH.push({
      pos: i * spacing,
      speed: Math.random() * 0.5 + 0.2,
      opacity: Math.random() * 0.3 + 0.1,
    });
  }
  for (let i = 0; i < Math.ceil(canvas.width / spacing) + 1; i++) {
    state.gridLinesV.push({
      pos: i * spacing,
      speed: Math.random() * 0.5 + 0.2,
      opacity: Math.random() * 0.3 + 0.1,
    });
  }
}

export function initWarp(
  state: AnimationState,
  canvas: HTMLCanvasElement,
  count: number,
) {
  state.warpStars = [];
  for (let i = 0; i < count; i++) {
    state.warpStars.push({
      x: Math.random() * canvas.width - canvas.width / 2,
      y: Math.random() * canvas.height - canvas.height / 2,
      z: Math.random() * 1000,
      prevX: 0,
      prevY: 0,
    });
  }
}

export function initHexgrid(state: AnimationState, canvas: HTMLCanvasElement) {
  state.hexCells = [];
  const hexSize = 30;
  const hexWidth = hexSize * 2;
  const hexHeight = Math.sqrt(3) * hexSize;
  for (let row = 0; row < canvas.height / hexHeight + 1; row++) {
    for (let col = 0; col < canvas.width / (hexWidth * 0.75) + 1; col++) {
      state.hexCells.push({
        x: col * hexWidth * 0.75,
        y: row * hexHeight + ((col % 2) * hexHeight) / 2,
        phase: Math.random() * Math.PI * 2,
        speed: Math.random() * 0.02 + 0.01,
      });
    }
  }
}

// ===== DRAW FUNCTIONS =====

export function drawParticles(
  ctx: CanvasRenderingContext2D,
  state: AnimationState,
  canvas: HTMLCanvasElement,
  intensity: AnimationIntensity,
  speed: number,
) {
  const config = intensityConfig[intensity];
  state.particles.forEach((p) => {
    ctx.beginPath();
    ctx.arc(p.x, p.y, p.size, 0, Math.PI * 2);
    ctx.fillStyle =
      p.color +
      Math.floor(p.opacity * config.opacity * 255)
        .toString(16)
        .padStart(2, "0");
    ctx.fill();
    p.x += p.speedX * speed;
    p.y += p.speedY * speed;
    if (p.x < 0) p.x = canvas.width;
    if (p.x > canvas.width) p.x = 0;
    if (p.y < 0) p.y = canvas.height;
    if (p.y > canvas.height) p.y = 0;
  });
}

export function drawGradient(
  ctx: CanvasRenderingContext2D,
  canvas: HTMLCanvasElement,
  intensity: AnimationIntensity,
  colors: string[],
  speed: number,
  time: number,
) {
  const config = intensityConfig[intensity];
  const angle = (time * 0.0001 * speed) % (Math.PI * 2);
  const x1 = canvas.width / 2 + (Math.cos(angle) * canvas.width) / 2;
  const y1 = canvas.height / 2 + (Math.sin(angle) * canvas.height) / 2;
  const x2 = canvas.width / 2 + (Math.cos(angle + Math.PI) * canvas.width) / 2;
  const y2 =
    canvas.height / 2 + (Math.sin(angle + Math.PI) * canvas.height) / 2;
  const gradient = ctx.createLinearGradient(x1, y1, x2, y2);
  colors.forEach((color, i) => {
    gradient.addColorStop(i / (colors.length - 1), color);
  });
  ctx.globalAlpha = config.opacity * 0.3;
  ctx.fillStyle = gradient;
  ctx.fillRect(0, 0, canvas.width, canvas.height);
  ctx.globalAlpha = 1;
}

export function drawAurora(
  ctx: CanvasRenderingContext2D,
  canvas: HTMLCanvasElement,
  intensity: AnimationIntensity,
  colors: string[],
  speed: number,
  time: number,
) {
  const config = intensityConfig[intensity];
  for (let layer = 0; layer < 3; layer++) {
    ctx.beginPath();
    ctx.moveTo(0, canvas.height);
    const waveHeight = canvas.height * 0.4;
    const baseY = canvas.height * 0.3 + layer * 50;
    for (let x = 0; x <= canvas.width; x += 5) {
      const wave1 =
        Math.sin(x * 0.01 + time * 0.0005 * speed + layer) * waveHeight * 0.3;
      const wave2 =
        Math.sin(x * 0.02 + time * 0.0003 * speed + layer * 2) *
        waveHeight *
        0.2;
      ctx.lineTo(x, baseY + wave1 + wave2);
    }
    ctx.lineTo(canvas.width, canvas.height);
    ctx.closePath();
    const gradient = ctx.createLinearGradient(
      0,
      baseY - waveHeight,
      0,
      canvas.height,
    );
    const color = colors[layer % colors.length];
    gradient.addColorStop(0, color + "00");
    gradient.addColorStop(
      0.5,
      color +
        Math.floor(config.opacity * 100)
          .toString(16)
          .padStart(2, "0"),
    );
    gradient.addColorStop(1, color + "00");
    ctx.fillStyle = gradient;
    ctx.fill();
  }
}

export function drawStarfield(
  ctx: CanvasRenderingContext2D,
  state: AnimationState,
  intensity: AnimationIntensity,
  speed: number,
  time: number,
) {
  const config = intensityConfig[intensity];
  state.stars.forEach((star) => {
    const twinkle = Math.sin(
      time * 0.001 * star.twinkleSpeed * speed + star.twinklePhase,
    );
    const brightness =
      (star.brightness * 0.5 + twinkle * 0.5 + 0.5) * config.opacity;
    ctx.beginPath();
    ctx.arc(star.x, star.y, star.size, 0, Math.PI * 2);
    ctx.fillStyle = `rgba(255, 255, 255, ${brightness})`;
    ctx.fill();
    if (star.size > 1.5 && brightness > 0.5) {
      ctx.beginPath();
      ctx.arc(star.x, star.y, star.size * 2, 0, Math.PI * 2);
      ctx.fillStyle = `rgba(255, 255, 255, ${brightness * 0.2})`;
      ctx.fill();
    }
  });
}

export function drawMatrix(
  ctx: CanvasRenderingContext2D,
  state: AnimationState,
  canvas: HTMLCanvasElement,
  intensity: AnimationIntensity,
  speed: number,
) {
  const config = intensityConfig[intensity];
  ctx.fillStyle = "rgba(0, 0, 0, 0.05)";
  ctx.fillRect(0, 0, canvas.width, canvas.height);
  const fontSize = 14;
  ctx.font = `${fontSize}px 'Courier New', monospace`;
  state.matrixDrops.forEach((drop) => {
    for (let i = 0; i < drop.chars.length; i++) {
      const y = drop.y - i * fontSize;
      if (y < 0 || y > canvas.height + fontSize) continue;
      const fadeRatio = 1 - i / drop.chars.length;
      const alpha = fadeRatio * drop.opacity * config.opacity;
      if (i === 0) {
        ctx.fillStyle = `rgba(180, 255, 180, ${alpha})`;
      } else {
        const green = Math.floor(100 + fadeRatio * 155);
        ctx.fillStyle = `rgba(0, ${green}, 70, ${alpha})`;
      }
      if (Math.random() < 0.02) {
        drop.chars[i] =
          "アイウエオカキクケコサシスセソタチツテトナニヌネノハヒフヘホマミムメモヤユヨラリルレロワヲン0123456789ABCDEF".split(
            "",
          )[Math.floor(Math.random() * 74)];
      }
      ctx.fillText(drop.chars[i], drop.x, y);
    }
    drop.y += drop.speed * speed * 3;
    if (drop.y - drop.length * fontSize > canvas.height) {
      drop.y = -drop.length * fontSize;
      drop.x = Math.floor(Math.random() * (canvas.width / 20)) * 20 + 10;
    }
  });
}

export function drawWaves(
  ctx: CanvasRenderingContext2D,
  canvas: HTMLCanvasElement,
  intensity: AnimationIntensity,
  colors: string[],
  speed: number,
  time: number,
) {
  const config = intensityConfig[intensity];
  for (let layer = 0; layer < 4; layer++) {
    ctx.beginPath();
    const amplitude = 30 + layer * 15;
    const frequency = 0.01 - layer * 0.002;
    const yOffset = canvas.height * 0.5 + layer * 40;
    const phaseOffset = layer * 0.5;
    ctx.moveTo(0, canvas.height);
    for (let x = 0; x <= canvas.width; x += 2) {
      const y =
        yOffset +
        Math.sin(x * frequency + time * 0.001 * speed + phaseOffset) *
          amplitude +
        Math.sin(x * frequency * 2 + time * 0.0015 * speed + phaseOffset) *
          amplitude *
          0.5;
      ctx.lineTo(x, y);
    }
    ctx.lineTo(canvas.width, canvas.height);
    ctx.closePath();
    const gradient = ctx.createLinearGradient(
      0,
      yOffset - amplitude,
      0,
      canvas.height,
    );
    const color = colors[layer % colors.length] || "#3b82f6";
    const opacityHex = Math.floor((config.opacity * 60) / (layer + 1))
      .toString(16)
      .padStart(2, "0");
    gradient.addColorStop(0, color + opacityHex);
    gradient.addColorStop(1, color + "00");
    ctx.fillStyle = gradient;
    ctx.fill();
  }
}

export function drawBubbles(
  ctx: CanvasRenderingContext2D,
  state: AnimationState,
  canvas: HTMLCanvasElement,
  intensity: AnimationIntensity,
  speed: number,
) {
  const config = intensityConfig[intensity];
  state.bubbles.forEach((bubble) => {
    bubble.wobble += bubble.wobbleSpeed * speed;
    const wobbleX = Math.sin(bubble.wobble) * 20;
    ctx.beginPath();
    ctx.arc(bubble.x + wobbleX, bubble.y, bubble.size, 0, Math.PI * 2);
    const gradient = ctx.createRadialGradient(
      bubble.x + wobbleX - bubble.size * 0.3,
      bubble.y - bubble.size * 0.3,
      bubble.size * 0.1,
      bubble.x + wobbleX,
      bubble.y,
      bubble.size,
    );
    gradient.addColorStop(
      0,
      `rgba(255, 255, 255, ${bubble.opacity * config.opacity * 0.5})`,
    );
    gradient.addColorStop(
      0.5,
      bubble.color +
        Math.floor(bubble.opacity * config.opacity * 150)
          .toString(16)
          .padStart(2, "0"),
    );
    gradient.addColorStop(1, bubble.color + "00");
    ctx.fillStyle = gradient;
    ctx.fill();
    ctx.beginPath();
    ctx.arc(
      bubble.x + wobbleX - bubble.size * 0.3,
      bubble.y - bubble.size * 0.3,
      bubble.size * 0.15,
      0,
      Math.PI * 2,
    );
    ctx.fillStyle = `rgba(255, 255, 255, ${bubble.opacity * config.opacity * 0.6})`;
    ctx.fill();
    bubble.y -= bubble.speed * speed;
    if (bubble.y + bubble.size < 0) {
      bubble.y = canvas.height + bubble.size;
      bubble.x = Math.random() * canvas.width;
    }
  });
}

export function drawGeometric(
  ctx: CanvasRenderingContext2D,
  state: AnimationState,
  canvas: HTMLCanvasElement,
  intensity: AnimationIntensity,
  speed: number,
) {
  const config = intensityConfig[intensity];
  state.geoShapes.forEach((shape) => {
    ctx.save();
    ctx.translate(shape.x, shape.y);
    ctx.rotate(shape.rotation);
    ctx.beginPath();
    for (let i = 0; i < shape.sides; i++) {
      const angle = (i / shape.sides) * Math.PI * 2 - Math.PI / 2;
      const x = Math.cos(angle) * shape.size;
      const y = Math.sin(angle) * shape.size;
      if (i === 0) ctx.moveTo(x, y);
      else ctx.lineTo(x, y);
    }
    ctx.closePath();
    ctx.strokeStyle =
      shape.color +
      Math.floor(shape.opacity * config.opacity * 255)
        .toString(16)
        .padStart(2, "0");
    ctx.lineWidth = 2;
    ctx.stroke();
    ctx.fillStyle =
      shape.color +
      Math.floor(shape.opacity * config.opacity * 50)
        .toString(16)
        .padStart(2, "0");
    ctx.fill();
    ctx.restore();
    shape.rotation += shape.rotationSpeed * speed;
    shape.x += shape.speedX * speed;
    shape.y += shape.speedY * speed;
    if (shape.x < -shape.size) shape.x = canvas.width + shape.size;
    if (shape.x > canvas.width + shape.size) shape.x = -shape.size;
    if (shape.y < -shape.size) shape.y = canvas.height + shape.size;
    if (shape.y > canvas.height + shape.size) shape.y = -shape.size;
  });
}

export function drawFireflies(
  ctx: CanvasRenderingContext2D,
  state: AnimationState,
  canvas: HTMLCanvasElement,
  intensity: AnimationIntensity,
  speed: number,
) {
  const config = intensityConfig[intensity];
  state.fireflies.forEach((fly) => {
    fly.glowPhase += fly.glowSpeed * speed;
    const glow = (Math.sin(fly.glowPhase) + 1) / 2;
    const alpha = glow * config.opacity;
    const gradient = ctx.createRadialGradient(
      fly.x,
      fly.y,
      0,
      fly.x,
      fly.y,
      fly.size * 3,
    );
    gradient.addColorStop(
      0,
      fly.color +
        Math.floor(alpha * 255)
          .toString(16)
          .padStart(2, "0"),
    );
    gradient.addColorStop(1, fly.color + "00");
    ctx.beginPath();
    ctx.arc(fly.x, fly.y, fly.size * 3, 0, Math.PI * 2);
    ctx.fillStyle = gradient;
    ctx.fill();
    ctx.beginPath();
    ctx.arc(fly.x, fly.y, fly.size, 0, Math.PI * 2);
    ctx.fillStyle = `rgba(255, 255, 200, ${alpha})`;
    ctx.fill();
    fly.x += fly.speedX * speed;
    fly.y += fly.speedY * speed;
    if (Math.random() < 0.02) {
      fly.speedX += (Math.random() - 0.5) * 0.2;
      fly.speedY += (Math.random() - 0.5) * 0.2;
      fly.speedX = Math.max(-1, Math.min(1, fly.speedX));
      fly.speedY = Math.max(-1, Math.min(1, fly.speedY));
    }
    if (fly.x < 0) fly.x = canvas.width;
    if (fly.x > canvas.width) fly.x = 0;
    if (fly.y < 0) fly.y = canvas.height;
    if (fly.y > canvas.height) fly.y = 0;
  });
}

export function drawRain(
  ctx: CanvasRenderingContext2D,
  state: AnimationState,
  canvas: HTMLCanvasElement,
  intensity: AnimationIntensity,
  speed: number,
) {
  const config = intensityConfig[intensity];
  state.raindrops.forEach((drop) => {
    ctx.beginPath();
    ctx.moveTo(drop.x, drop.y);
    ctx.lineTo(drop.x + 1, drop.y + drop.length);
    ctx.strokeStyle = `rgba(150, 180, 220, ${drop.opacity * config.opacity})`;
    ctx.lineWidth = 1;
    ctx.stroke();
    drop.y += drop.speed * speed;
    drop.x += 0.5 * speed;
    if (drop.y > canvas.height) {
      drop.y = -drop.length;
      drop.x = Math.random() * canvas.width;
    }
  });
}

export function drawSnow(
  ctx: CanvasRenderingContext2D,
  state: AnimationState,
  canvas: HTMLCanvasElement,
  intensity: AnimationIntensity,
  speed: number,
  time: number,
) {
  const config = intensityConfig[intensity];
  state.snowflakes.forEach((flake) => {
    flake.wobble += flake.wobbleSpeed * speed;
    const wobbleX = Math.sin(flake.wobble) * 20;
    ctx.beginPath();
    ctx.arc(flake.x + wobbleX, flake.y, flake.size, 0, Math.PI * 2);
    ctx.fillStyle = `rgba(255, 255, 255, ${flake.opacity * config.opacity})`;
    ctx.fill();
    flake.y += flake.speed * speed;
    flake.x += Math.sin(time * 0.001) * 0.2 * speed;
    if (flake.y > canvas.height + flake.size) {
      flake.y = -flake.size;
      flake.x = Math.random() * canvas.width;
    }
  });
}

export function drawConfetti(
  ctx: CanvasRenderingContext2D,
  state: AnimationState,
  canvas: HTMLCanvasElement,
  intensity: AnimationIntensity,
  speed: number,
) {
  const config = intensityConfig[intensity];
  state.confetti.forEach((piece) => {
    ctx.save();
    ctx.translate(piece.x, piece.y);
    ctx.rotate(piece.rotation);
    ctx.beginPath();
    ctx.rect(-piece.size / 2, -piece.size / 4, piece.size, piece.size / 2);
    ctx.fillStyle =
      piece.color +
      Math.floor(piece.opacity * config.opacity * 255)
        .toString(16)
        .padStart(2, "0");
    ctx.fill();
    ctx.restore();
    piece.y += piece.speedY * speed;
    piece.x += piece.speedX * speed;
    piece.rotation += piece.rotationSpeed * speed;
    piece.speedX += (Math.random() - 0.5) * 0.1;
    if (piece.y > canvas.height + piece.size) {
      piece.y = -piece.size;
      piece.x = Math.random() * canvas.width;
    }
  });
}

export function drawPulse(
  ctx: CanvasRenderingContext2D,
  state: AnimationState,
  canvas: HTMLCanvasElement,
  intensity: AnimationIntensity,
  colors: string[],
  speed: number,
) {
  const config = intensityConfig[intensity];
  state.pulsePhase += 0.02 * speed;
  const pulse = (Math.sin(state.pulsePhase) + 1) / 2;
  const gradient = ctx.createRadialGradient(
    canvas.width / 2,
    canvas.height / 2,
    0,
    canvas.width / 2,
    canvas.height / 2,
    Math.max(canvas.width, canvas.height) * 0.7,
  );
  const color = colors[0] || "#667eea";
  gradient.addColorStop(
    0,
    color +
      Math.floor(pulse * config.opacity * 100)
        .toString(16)
        .padStart(2, "0"),
  );
  gradient.addColorStop(
    0.5,
    color +
      Math.floor(pulse * config.opacity * 50)
        .toString(16)
        .padStart(2, "0"),
  );
  gradient.addColorStop(1, color + "00");
  ctx.fillStyle = gradient;
  ctx.fillRect(0, 0, canvas.width, canvas.height);
}

export function drawRipples(
  ctx: CanvasRenderingContext2D,
  state: AnimationState,
  canvas: HTMLCanvasElement,
  intensity: AnimationIntensity,
  speed: number,
) {
  const config = intensityConfig[intensity];
  state.ripples.forEach((ripple) => {
    const fadeRatio = 1 - ripple.radius / ripple.maxRadius;
    const alpha = ripple.opacity * fadeRatio * config.opacity;
    ctx.beginPath();
    ctx.arc(ripple.x, ripple.y, ripple.radius, 0, Math.PI * 2);
    ctx.strokeStyle =
      ripple.color +
      Math.floor(alpha * 255)
        .toString(16)
        .padStart(2, "0");
    ctx.lineWidth = 2;
    ctx.stroke();
    ripple.radius += ripple.speed * speed;
    if (ripple.radius >= ripple.maxRadius) {
      ripple.radius = 0;
      ripple.x = Math.random() * canvas.width;
      ripple.y = Math.random() * canvas.height;
      ripple.maxRadius = Math.random() * 100 + 50;
    }
  });
}

export function drawNebula(
  ctx: CanvasRenderingContext2D,
  state: AnimationState,
  canvas: HTMLCanvasElement,
  intensity: AnimationIntensity,
  colors: string[],
  speed: number,
) {
  const config = intensityConfig[intensity];
  state.nebulaTime += 0.005 * speed;
  for (let i = 0; i < 3; i++) {
    const x =
      canvas.width / 2 +
      Math.sin(state.nebulaTime + i * 2) * canvas.width * 0.3;
    const y =
      canvas.height / 2 +
      Math.cos(state.nebulaTime * 0.7 + i * 2) * canvas.height * 0.3;
    const size = Math.max(canvas.width, canvas.height) * (0.3 + i * 0.1);
    const gradient = ctx.createRadialGradient(x, y, 0, x, y, size);
    const color = colors[i % colors.length] || "#667eea";
    gradient.addColorStop(
      0,
      color +
        Math.floor(config.opacity * 60)
          .toString(16)
          .padStart(2, "0"),
    );
    gradient.addColorStop(
      0.5,
      color +
        Math.floor(config.opacity * 30)
          .toString(16)
          .padStart(2, "0"),
    );
    gradient.addColorStop(1, color + "00");
    ctx.fillStyle = gradient;
    ctx.fillRect(0, 0, canvas.width, canvas.height);
  }
  for (let i = 0; i < 50; i++) {
    const starX =
      ((Math.sin(i * 12.3 + state.nebulaTime * 0.1) + 1) / 2) * canvas.width;
    const starY =
      ((Math.cos(i * 7.7 + state.nebulaTime * 0.1) + 1) / 2) * canvas.height;
    const starSize = Math.sin(i * 3.3) * 0.5 + 1;
    ctx.beginPath();
    ctx.arc(starX, starY, starSize, 0, Math.PI * 2);
    ctx.fillStyle = `rgba(255, 255, 255, ${config.opacity * 0.5})`;
    ctx.fill();
  }
}

export function drawCircuit(
  ctx: CanvasRenderingContext2D,
  state: AnimationState,
  intensity: AnimationIntensity,
  colors: string[],
  speed: number,
  time: number,
) {
  const config = intensityConfig[intensity];
  const color = colors[0] || "#3b82f6";
  state.circuitNodes.forEach((node) => {
    node.connections.forEach((targetIndex) => {
      const target = state.circuitNodes[targetIndex];
      const pulse = (Math.sin(time * 0.002 + node.pulsePhase) + 1) / 2;
      ctx.beginPath();
      ctx.moveTo(node.x, node.y);
      ctx.lineTo(target.x, target.y);
      ctx.strokeStyle =
        color +
        Math.floor(pulse * config.opacity * 100)
          .toString(16)
          .padStart(2, "0");
      ctx.lineWidth = 1;
      ctx.stroke();
    });
  });
  state.circuitNodes.forEach((node) => {
    const pulse = (Math.sin(time * 0.002 + node.pulsePhase) + 1) / 2;
    ctx.beginPath();
    ctx.arc(node.x, node.y, 4, 0, Math.PI * 2);
    ctx.fillStyle =
      color +
      Math.floor(pulse * config.opacity * 200)
        .toString(16)
        .padStart(2, "0");
    ctx.fill();
    const gradient = ctx.createRadialGradient(
      node.x,
      node.y,
      0,
      node.x,
      node.y,
      15,
    );
    gradient.addColorStop(
      0,
      color +
        Math.floor(pulse * config.opacity * 80)
          .toString(16)
          .padStart(2, "0"),
    );
    gradient.addColorStop(1, color + "00");
    ctx.beginPath();
    ctx.arc(node.x, node.y, 15, 0, Math.PI * 2);
    ctx.fillStyle = gradient;
    ctx.fill();
    node.pulsePhase += node.pulseSpeed * speed;
  });
}

export function drawDots(
  ctx: CanvasRenderingContext2D,
  state: AnimationState,
  intensity: AnimationIntensity,
  colors: string[],
  speed: number,
) {
  const config = intensityConfig[intensity];
  const color = colors[0] || "#667eea";
  state.dots.forEach((dot) => {
    dot.phase += dot.speed * speed;
    const pulse = (Math.sin(dot.phase) + 1) / 2;
    const size = dot.baseSize + pulse * 2;
    const alpha = 0.3 + pulse * 0.4;
    ctx.beginPath();
    ctx.arc(dot.x, dot.y, size, 0, Math.PI * 2);
    ctx.fillStyle =
      color +
      Math.floor(alpha * config.opacity * 255)
        .toString(16)
        .padStart(2, "0");
    ctx.fill();
  });
}

export function drawFloatingShapes(
  ctx: CanvasRenderingContext2D,
  state: AnimationState,
  canvas: HTMLCanvasElement,
  intensity: AnimationIntensity,
  speed: number,
) {
  const config = intensityConfig[intensity];
  state.floatingShapes.forEach((shape) => {
    ctx.save();
    ctx.translate(shape.x, shape.y);
    ctx.rotate(shape.rotation);
    ctx.beginPath();
    if (shape.type === "square") {
      ctx.rect(-shape.size / 2, -shape.size / 2, shape.size, shape.size);
    } else if (shape.type === "triangle") {
      ctx.moveTo(0, -shape.size / 2);
      ctx.lineTo(shape.size / 2, shape.size / 2);
      ctx.lineTo(-shape.size / 2, shape.size / 2);
      ctx.closePath();
    } else {
      ctx.arc(0, 0, shape.size / 2, 0, Math.PI * 2);
    }
    ctx.fillStyle =
      shape.color +
      Math.floor(shape.opacity * config.opacity * 100)
        .toString(16)
        .padStart(2, "0");
    ctx.fill();
    ctx.strokeStyle =
      shape.color +
      Math.floor(shape.opacity * config.opacity * 200)
        .toString(16)
        .padStart(2, "0");
    ctx.lineWidth = 1;
    ctx.stroke();
    ctx.restore();
    shape.x += shape.speedX * speed;
    shape.y += shape.speedY * speed;
    shape.rotation += shape.rotSpeed * speed;
    if (shape.x < -shape.size) shape.x = canvas.width + shape.size;
    if (shape.x > canvas.width + shape.size) shape.x = -shape.size;
    if (shape.y < -shape.size) shape.y = canvas.height + shape.size;
    if (shape.y > canvas.height + shape.size) shape.y = -shape.size;
  });
}

export function drawSmoke(
  ctx: CanvasRenderingContext2D,
  state: AnimationState,
  canvas: HTMLCanvasElement,
  intensity: AnimationIntensity,
  colors: string[],
  speed: number,
) {
  const config = intensityConfig[intensity];
  const color = colors[0] || "#888888";
  state.smokeParticles.forEach((particle) => {
    const gradient = ctx.createRadialGradient(
      particle.x,
      particle.y,
      0,
      particle.x,
      particle.y,
      particle.size,
    );
    const alpha = particle.opacity * particle.life * config.opacity;
    gradient.addColorStop(
      0,
      color +
        Math.floor(alpha * 150)
          .toString(16)
          .padStart(2, "0"),
    );
    gradient.addColorStop(
      0.5,
      color +
        Math.floor(alpha * 80)
          .toString(16)
          .padStart(2, "0"),
    );
    gradient.addColorStop(1, color + "00");
    ctx.beginPath();
    ctx.arc(particle.x, particle.y, particle.size, 0, Math.PI * 2);
    ctx.fillStyle = gradient;
    ctx.fill();
    particle.x += particle.speedX * speed;
    particle.y += particle.speedY * speed;
    particle.size += 0.3 * speed;
    particle.life -= 0.005 * speed;
    if (particle.life <= 0) {
      particle.x = Math.random() * canvas.width;
      particle.y = canvas.height + 20;
      particle.size = Math.random() * 50 + 30;
      particle.opacity = Math.random() * 0.2 + 0.1;
      particle.life = 1;
    }
  });
}

export function drawScanlines(
  ctx: CanvasRenderingContext2D,
  state: AnimationState,
  canvas: HTMLCanvasElement,
  intensity: AnimationIntensity,
  colors: string[],
  speed: number,
  time: number,
) {
  const config = intensityConfig[intensity];
  const color = colors[0] || "#00ff00";
  state.scanlineOffset = (state.scanlineOffset + speed * 0.5) % 4;
  ctx.strokeStyle =
    color +
    Math.floor(config.opacity * 30)
      .toString(16)
      .padStart(2, "0");
  ctx.lineWidth = 1;
  for (let y = state.scanlineOffset; y < canvas.height; y += 4) {
    ctx.beginPath();
    ctx.moveTo(0, y);
    ctx.lineTo(canvas.width, y);
    ctx.stroke();
  }
  const brightY = (time * 0.1 * speed) % canvas.height;
  const brightGradient = ctx.createLinearGradient(
    0,
    brightY - 20,
    0,
    brightY + 20,
  );
  brightGradient.addColorStop(0, color + "00");
  brightGradient.addColorStop(
    0.5,
    color +
      Math.floor(config.opacity * 100)
        .toString(16)
        .padStart(2, "0"),
  );
  brightGradient.addColorStop(1, color + "00");
  ctx.fillStyle = brightGradient;
  ctx.fillRect(0, brightY - 20, canvas.width, 40);
}

export function drawGrid(
  ctx: CanvasRenderingContext2D,
  state: AnimationState,
  intensity: AnimationIntensity,
  colors: string[],
  speed: number,
  time: number,
) {
  const config = intensityConfig[intensity];
  const color = colors[0] || "#3b82f6";
  state.gridLinesH.forEach((line) => {
    const pulse = (Math.sin(time * 0.001 + line.pos * 0.01) + 1) / 2;
    ctx.beginPath();
    ctx.moveTo(0, line.pos);
    ctx.lineTo(10000, line.pos);
    ctx.strokeStyle =
      color +
      Math.floor((line.opacity + pulse * 0.2) * config.opacity * 150)
        .toString(16)
        .padStart(2, "0");
    ctx.lineWidth = 1;
    ctx.stroke();
  });
  state.gridLinesV.forEach((line) => {
    const pulse = (Math.sin(time * 0.001 + line.pos * 0.01) + 1) / 2;
    ctx.beginPath();
    ctx.moveTo(line.pos, 0);
    ctx.lineTo(line.pos, 10000);
    ctx.strokeStyle =
      color +
      Math.floor((line.opacity + pulse * 0.2) * config.opacity * 150)
        .toString(16)
        .padStart(2, "0");
    ctx.lineWidth = 1;
    ctx.stroke();
  });
  state.gridLinesH.forEach((hLine) => {
    state.gridLinesV.forEach((vLine) => {
      const pulse =
        (Math.sin(time * 0.002 + hLine.pos * 0.01 + vLine.pos * 0.01) + 1) / 2;
      if (pulse > 0.7) {
        ctx.beginPath();
        ctx.arc(vLine.pos, hLine.pos, 3, 0, Math.PI * 2);
        ctx.fillStyle =
          color +
          Math.floor(pulse * config.opacity * 255)
            .toString(16)
            .padStart(2, "0");
        ctx.fill();
      }
    });
  });
}

export function drawWarp(
  ctx: CanvasRenderingContext2D,
  state: AnimationState,
  canvas: HTMLCanvasElement,
  intensity: AnimationIntensity,
  speed: number,
) {
  const config = intensityConfig[intensity];
  const centerX = canvas.width / 2;
  const centerY = canvas.height / 2;
  state.warpStars.forEach((star) => {
    star.prevX = (star.x / star.z) * 200 + centerX;
    star.prevY = (star.y / star.z) * 200 + centerY;
    star.z -= speed * 10;
    const sx = (star.x / star.z) * 200 + centerX;
    const sy = (star.y / star.z) * 200 + centerY;
    if (
      star.z <= 0 ||
      sx < 0 ||
      sx > canvas.width ||
      sy < 0 ||
      sy > canvas.height
    ) {
      star.x = Math.random() * canvas.width - centerX;
      star.y = Math.random() * canvas.height - centerY;
      star.z = 1000;
      star.prevX = sx;
      star.prevY = sy;
      return;
    }
    const brightness = (1 - star.z / 1000) * config.opacity;
    ctx.beginPath();
    ctx.moveTo(star.prevX, star.prevY);
    ctx.lineTo(sx, sy);
    ctx.strokeStyle = `rgba(255, 255, 255, ${brightness})`;
    ctx.lineWidth = (1 - star.z / 1000) * 3;
    ctx.stroke();
    ctx.beginPath();
    ctx.arc(sx, sy, (1 - star.z / 1000) * 2, 0, Math.PI * 2);
    ctx.fillStyle = `rgba(255, 255, 255, ${brightness})`;
    ctx.fill();
  });
}

export function drawHexgrid(
  ctx: CanvasRenderingContext2D,
  state: AnimationState,
  intensity: AnimationIntensity,
  colors: string[],
  speed: number,
) {
  const config = intensityConfig[intensity];
  const color = colors[0] || "#667eea";
  const hexSize = 30;
  state.hexCells.forEach((hex) => {
    hex.phase += hex.speed * speed;
    const pulse = (Math.sin(hex.phase) + 1) / 2;
    ctx.beginPath();
    for (let i = 0; i < 6; i++) {
      const angle = (i / 6) * Math.PI * 2 - Math.PI / 6;
      const x = hex.x + Math.cos(angle) * hexSize;
      const y = hex.y + Math.sin(angle) * hexSize;
      if (i === 0) ctx.moveTo(x, y);
      else ctx.lineTo(x, y);
    }
    ctx.closePath();
    ctx.strokeStyle =
      color +
      Math.floor((0.2 + pulse * 0.3) * config.opacity * 255)
        .toString(16)
        .padStart(2, "0");
    ctx.lineWidth = 1;
    ctx.stroke();
    if (pulse > 0.7) {
      ctx.fillStyle =
        color +
        Math.floor((pulse - 0.7) * config.opacity * 100)
          .toString(16)
          .padStart(2, "0");
      ctx.fill();
    }
  });
}

export function drawBinary(
  ctx: CanvasRenderingContext2D,
  state: AnimationState,
  canvas: HTMLCanvasElement,
  intensity: AnimationIntensity,
  colors: string[],
  speed: number,
) {
  const config = intensityConfig[intensity];
  const color = colors[0] || "#00ff00";
  const fontSize = 12;
  ctx.font = `${fontSize}px 'Courier New', monospace`;
  state.matrixDrops.forEach((drop) => {
    for (let i = 0; i < drop.chars.length; i++) {
      const y = drop.y - i * fontSize;
      if (y < 0 || y > canvas.height + fontSize) continue;
      const fadeRatio = 1 - i / drop.chars.length;
      const alpha = fadeRatio * drop.opacity * config.opacity;
      const char = Math.random() > 0.5 ? "1" : "0";
      if (i === 0) {
        ctx.fillStyle = `rgba(200, 255, 200, ${alpha})`;
      } else {
        ctx.fillStyle =
          color +
          Math.floor(alpha * 200)
            .toString(16)
            .padStart(2, "0");
      }
      ctx.fillText(char, drop.x, y);
    }
    drop.y += drop.speed * speed * 2;
    if (drop.y - drop.length * fontSize > canvas.height) {
      drop.y = -drop.length * fontSize;
      drop.x = Math.floor(Math.random() * (canvas.width / 15)) * 15 + 7;
    }
  });
}
