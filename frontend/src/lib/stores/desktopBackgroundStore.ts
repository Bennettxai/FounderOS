// Desktop Background Store — Background management, wallpaper state, background effects
import { browser } from "$app/environment";

export interface DesktopBackground {
  id: string;
  name: string;
  type: "solid" | "gradient" | "pattern" | "image";
  value: string;
  preview: string;
}

export const desktopBackgrounds: DesktopBackground[] = [
  // Solid colors
  {
    id: "classic-gray",
    name: "Classic Gray",
    type: "solid",
    value: "#E5E5E5",
    preview: "#E5E5E5",
  },
  {
    id: "warm-beige",
    name: "Warm Beige",
    type: "solid",
    value: "#E8E4DE",
    preview: "#E8E4DE",
  },
  {
    id: "soft-blue",
    name: "Soft Blue",
    type: "solid",
    value: "#E3EDF7",
    preview: "#E3EDF7",
  },
  {
    id: "mint-green",
    name: "Mint Green",
    type: "solid",
    value: "#E3F0E8",
    preview: "#E3F0E8",
  },
  {
    id: "lavender",
    name: "Lavender",
    type: "solid",
    value: "#EDE7F3",
    preview: "#EDE7F3",
  },
  {
    id: "warm-sand",
    name: "Warm Sand",
    type: "solid",
    value: "#F5EDE3",
    preview: "#F5EDE3",
  },
  {
    id: "rose-pink",
    name: "Rose Pink",
    type: "solid",
    value: "#F8E8EE",
    preview: "#F8E8EE",
  },
  {
    id: "sky-blue",
    name: "Sky Blue",
    type: "solid",
    value: "#E1F5FE",
    preview: "#E1F5FE",
  },
  {
    id: "peach",
    name: "Peach",
    type: "solid",
    value: "#FFECD2",
    preview: "#FFECD2",
  },
  {
    id: "sage-green",
    name: "Sage Green",
    type: "solid",
    value: "#D4E9D7",
    preview: "#D4E9D7",
  },
  {
    id: "dusty-rose",
    name: "Dusty Rose",
    type: "solid",
    value: "#E8D4D4",
    preview: "#E8D4D4",
  },
  {
    id: "cream",
    name: "Cream",
    type: "solid",
    value: "#FFFEF0",
    preview: "#FFFEF0",
  },
  {
    id: "slate",
    name: "Slate",
    type: "solid",
    value: "#4A5568",
    preview: "#4A5568",
  },
  {
    id: "charcoal",
    name: "Charcoal",
    type: "solid",
    value: "#2D3748",
    preview: "#2D3748",
  },
  {
    id: "dark-mode",
    name: "Dark Mode",
    type: "solid",
    value: "#1E1E1E",
    preview: "#1E1E1E",
  },
  {
    id: "midnight",
    name: "Midnight",
    type: "solid",
    value: "#0D1117",
    preview: "#0D1117",
  },
  {
    id: "navy",
    name: "Navy",
    type: "solid",
    value: "#1A365D",
    preview: "#1A365D",
  },
  {
    id: "deep-purple",
    name: "Deep Purple",
    type: "solid",
    value: "#322659",
    preview: "#322659",
  },

  // Gradients
  {
    id: "sunrise",
    name: "Sunrise",
    type: "gradient",
    value: "linear-gradient(135deg, #FEF3E2 0%, #FAD4D4 50%, #E8D5E7 100%)",
    preview: "linear-gradient(135deg, #FEF3E2, #FAD4D4, #E8D5E7)",
  },
  {
    id: "ocean",
    name: "Ocean",
    type: "gradient",
    value: "linear-gradient(135deg, #E0F4F5 0%, #C6E7F2 50%, #A8D8EA 100%)",
    preview: "linear-gradient(135deg, #E0F4F5, #C6E7F2, #A8D8EA)",
  },
  {
    id: "forest",
    name: "Forest",
    type: "gradient",
    value: "linear-gradient(135deg, #E8F5E9 0%, #C8E6C9 50%, #A5D6A7 100%)",
    preview: "linear-gradient(135deg, #E8F5E9, #C8E6C9, #A5D6A7)",
  },
  {
    id: "sunset",
    name: "Sunset",
    type: "gradient",
    value: "linear-gradient(135deg, #FFE5D9 0%, #FFCAD4 50%, #D8B5FF 100%)",
    preview: "linear-gradient(135deg, #FFE5D9, #FFCAD4, #D8B5FF)",
  },
  {
    id: "cotton-candy",
    name: "Cotton Candy",
    type: "gradient",
    value: "linear-gradient(135deg, #FFECD2 0%, #FCB69F 50%, #FF9A9E 100%)",
    preview: "linear-gradient(135deg, #FFECD2, #FCB69F, #FF9A9E)",
  },
  {
    id: "mint-breeze",
    name: "Mint Breeze",
    type: "gradient",
    value: "linear-gradient(135deg, #D4FC79 0%, #96E6A1 100%)",
    preview: "linear-gradient(135deg, #D4FC79, #96E6A1)",
  },
  {
    id: "lavender-mist",
    name: "Lavender Mist",
    type: "gradient",
    value: "linear-gradient(135deg, #E0C3FC 0%, #8EC5FC 100%)",
    preview: "linear-gradient(135deg, #E0C3FC, #8EC5FC)",
  },
  {
    id: "warm-flame",
    name: "Warm Flame",
    type: "gradient",
    value: "linear-gradient(135deg, #FFE29F 0%, #FFA99F 50%, #FF719A 100%)",
    preview: "linear-gradient(135deg, #FFE29F, #FFA99F, #FF719A)",
  },
  {
    id: "winter-sky",
    name: "Winter Sky",
    type: "gradient",
    value: "linear-gradient(135deg, #A1C4FD 0%, #C2E9FB 100%)",
    preview: "linear-gradient(135deg, #A1C4FD, #C2E9FB)",
  },
  {
    id: "aurora",
    name: "Aurora",
    type: "gradient",
    value:
      "linear-gradient(135deg, #1A1A2E 0%, #16213E 25%, #0F3460 50%, #533483 75%, #E94560 100%)",
    preview: "linear-gradient(135deg, #1A1A2E, #16213E, #0F3460, #533483)",
  },
  {
    id: "cosmic",
    name: "Cosmic",
    type: "gradient",
    value: "linear-gradient(135deg, #0F0C29 0%, #302B63 50%, #24243E 100%)",
    preview: "linear-gradient(135deg, #0F0C29, #302B63, #24243E)",
  },
  {
    id: "night-fade",
    name: "Night Fade",
    type: "gradient",
    value: "linear-gradient(135deg, #232526 0%, #414345 100%)",
    preview: "linear-gradient(135deg, #232526, #414345)",
  },
  {
    id: "deep-space",
    name: "Deep Space",
    type: "gradient",
    value: "linear-gradient(135deg, #000428 0%, #004e92 100%)",
    preview: "linear-gradient(135deg, #000428, #004e92)",
  },
  {
    id: "purple-haze",
    name: "Purple Haze",
    type: "gradient",
    value: "linear-gradient(135deg, #4A00E0 0%, #8E2DE2 100%)",
    preview: "linear-gradient(135deg, #4A00E0, #8E2DE2)",
  },
  {
    id: "midnight-city",
    name: "Midnight City",
    type: "gradient",
    value: "linear-gradient(135deg, #373B44 0%, #4286f4 100%)",
    preview: "linear-gradient(135deg, #373B44, #4286f4)",
  },

  // Patterns (using CSS patterns)
  {
    id: "dots",
    name: "Polka Dots",
    type: "pattern",
    value: `
			radial-gradient(circle, #00000010 1px, transparent 1px),
			#E5E5E5
		`,
    preview: "radial-gradient(circle, #00000040 2px, transparent 2px), #E5E5E5",
  },
  {
    id: "grid",
    name: "Grid",
    type: "pattern",
    value: `
			linear-gradient(#00000008 1px, transparent 1px),
			linear-gradient(90deg, #00000008 1px, transparent 1px),
			#F5F5F5
		`,
    preview:
      "linear-gradient(#00000030 1px, transparent 1px), linear-gradient(90deg, #00000030 1px, transparent 1px), #F5F5F5",
  },
  {
    id: "diagonal-lines",
    name: "Diagonal Lines",
    type: "pattern",
    value: `
			repeating-linear-gradient(
				45deg,
				transparent,
				transparent 10px,
				#00000008 10px,
				#00000008 11px
			),
			#F0F0F0
		`,
    preview:
      "repeating-linear-gradient(45deg, transparent, transparent 4px, #00000030 4px, #00000030 5px), #F0F0F0",
  },
  {
    id: "checkerboard",
    name: "Checkerboard",
    type: "pattern",
    value: `
			linear-gradient(45deg, #00000008 25%, transparent 25%),
			linear-gradient(-45deg, #00000008 25%, transparent 25%),
			linear-gradient(45deg, transparent 75%, #00000008 75%),
			linear-gradient(-45deg, transparent 75%, #00000008 75%),
			#E8E8E8
		`,
    preview:
      "linear-gradient(45deg, #00000025 25%, transparent 25%), linear-gradient(-45deg, #00000025 25%, transparent 25%), linear-gradient(45deg, transparent 75%, #00000025 75%), linear-gradient(-45deg, transparent 75%, #00000025 75%), #E8E8E8",
  },
  {
    id: "zigzag",
    name: "Zigzag",
    type: "pattern",
    value: `
			linear-gradient(135deg, #00000010 25%, transparent 25%) -20px 0,
			linear-gradient(225deg, #00000010 25%, transparent 25%) -20px 0,
			linear-gradient(315deg, #00000010 25%, transparent 25%),
			linear-gradient(45deg, #00000010 25%, transparent 25%),
			#EBEBEB
		`,
    preview:
      "linear-gradient(135deg, #00000030 25%, transparent 25%), linear-gradient(225deg, #00000030 25%, transparent 25%), linear-gradient(315deg, #00000030 25%, transparent 25%), linear-gradient(45deg, #00000030 25%, transparent 25%), #EBEBEB",
  },
  {
    id: "honeycomb",
    name: "Honeycomb",
    type: "pattern",
    value: `
			radial-gradient(circle farthest-side at 0% 50%, #F5F5F5 23.5%, transparent 0) 21px 30px,
			radial-gradient(circle farthest-side at 0% 50%, #EDEDED 24%, transparent 0) 19px 30px,
			linear-gradient(#F5F5F5 14%, transparent 0, transparent 85%, #F5F5F5 0) 0 0,
			linear-gradient(150deg, #F5F5F5 24%, #E5E5E5 0, #E5E5E5 26%, transparent 0, transparent 74%, #E5E5E5 0, #E5E5E5 76%, #F5F5F5 0) 0 0,
			linear-gradient(30deg, #F5F5F5 24%, #E5E5E5 0, #E5E5E5 26%, transparent 0, transparent 74%, #E5E5E5 0, #E5E5E5 76%, #F5F5F5 0) 0 0,
			linear-gradient(90deg, #E5E5E5 2%, #F5F5F5 0, #F5F5F5 98%, #E5E5E5 0%) 0 0,
			#F5F5F5
		`,
    preview: "radial-gradient(circle, #D0D0D0 30%, transparent 30%), #E8E8E8",
  },
  {
    id: "waves",
    name: "Waves",
    type: "pattern",
    value: `
			radial-gradient(ellipse at 50% 0%, transparent 70%, #00000008 70%, #00000008 100%),
			radial-gradient(ellipse at 50% 100%, transparent 70%, #00000008 70%, #00000008 100%),
			#E8E8E8
		`,
    preview:
      "radial-gradient(ellipse at 50% 0%, transparent 50%, #00000025 50%), radial-gradient(ellipse at 50% 100%, transparent 50%, #00000025 50%), #E8E8E8",
  },
  {
    id: "cross-dots",
    name: "Cross Dots",
    type: "pattern",
    value: `
			radial-gradient(#00000015 2px, transparent 2px),
			radial-gradient(#00000015 2px, transparent 2px),
			#F2F2F2
		`,
    preview:
      "radial-gradient(#00000040 2px, transparent 2px), radial-gradient(#00000040 2px, transparent 2px), #F2F2F2",
  },
  {
    id: "dark-grid",
    name: "Dark Grid",
    type: "pattern",
    value: `
			linear-gradient(#ffffff08 1px, transparent 1px),
			linear-gradient(90deg, #ffffff08 1px, transparent 1px),
			#1a1a1a
		`,
    preview:
      "linear-gradient(#ffffff30 1px, transparent 1px), linear-gradient(90deg, #ffffff30 1px, transparent 1px), #1a1a1a",
  },
  {
    id: "dark-dots",
    name: "Dark Dots",
    type: "pattern",
    value: `
			radial-gradient(circle, #ffffff15 1px, transparent 1px),
			#1E1E1E
		`,
    preview: "radial-gradient(circle, #ffffff50 2px, transparent 2px), #1E1E1E",
  },
  {
    id: "stripes",
    name: "Stripes",
    type: "pattern",
    value: `
			repeating-linear-gradient(
				90deg,
				#E8E8E8,
				#E8E8E8 10px,
				#F2F2F2 10px,
				#F2F2F2 20px
			)
		`,
    preview:
      "repeating-linear-gradient(90deg, #D8D8D8, #D8D8D8 5px, #F2F2F2 5px, #F2F2F2 10px)",
  },
  {
    id: "diagonal-stripes",
    name: "Diagonal Stripes",
    type: "pattern",
    value: `
			repeating-linear-gradient(
				-45deg,
				#E5E5E5,
				#E5E5E5 10px,
				#F0F0F0 10px,
				#F0F0F0 20px
			)
		`,
    preview:
      "repeating-linear-gradient(-45deg, #D5D5D5, #D5D5D5 5px, #F0F0F0 5px, #F0F0F0 10px)",
  },
  {
    id: "blueprint",
    name: "Blueprint",
    type: "pattern",
    value: `
			linear-gradient(#3B82F620 1px, transparent 1px),
			linear-gradient(90deg, #3B82F620 1px, transparent 1px),
			#1E3A5F
		`,
    preview:
      "linear-gradient(#3B82F650 1px, transparent 1px), linear-gradient(90deg, #3B82F650 1px, transparent 1px), #1E3A5F",
  },
  {
    id: "carbon-fiber",
    name: "Carbon Fiber",
    type: "pattern",
    value: `
			linear-gradient(27deg, #151515 5px, transparent 5px) 0 5px,
			linear-gradient(207deg, #151515 5px, transparent 5px) 10px 0px,
			linear-gradient(27deg, #222 5px, transparent 5px) 0px 10px,
			linear-gradient(207deg, #222 5px, transparent 5px) 10px 5px,
			linear-gradient(90deg, #1b1b1b 10px, transparent 10px),
			linear-gradient(#1d1d1d 25%, #1a1a1a 25%, #1a1a1a 50%, transparent 50%, transparent 75%, #242424 75%, #242424),
			#131313
		`,
    preview:
      "linear-gradient(27deg, #252525 5px, transparent 5px), linear-gradient(207deg, #252525 5px, transparent 5px), #1a1a1a",
  },
  {
    id: "paper-texture",
    name: "Paper",
    type: "pattern",
    value: `
			radial-gradient(circle at 50% 50%, #00000005 0%, transparent 50%),
			radial-gradient(circle at 20% 80%, #00000008 0%, transparent 40%),
			radial-gradient(circle at 80% 20%, #00000006 0%, transparent 45%),
			#F5F5F0
		`,
    preview: "#F5F5F0",
  },
  {
    id: "diamonds",
    name: "Diamonds",
    type: "pattern",
    value: `
			linear-gradient(45deg, #E0E0E0 25%, transparent 25%),
			linear-gradient(-45deg, #E0E0E0 25%, transparent 25%),
			linear-gradient(45deg, transparent 75%, #E0E0E0 75%),
			linear-gradient(-45deg, transparent 75%, #E0E0E0 75%),
			#F5F5F5
		`,
    preview:
      "linear-gradient(45deg, #D0D0D0 25%, transparent 25%), linear-gradient(-45deg, #D0D0D0 25%, transparent 25%), linear-gradient(45deg, transparent 75%, #D0D0D0 75%), linear-gradient(-45deg, transparent 75%, #D0D0D0 75%), #F5F5F5",
  },
  {
    id: "triangles",
    name: "Triangles",
    type: "pattern",
    value: `
			linear-gradient(60deg, #E8E8E8 25%, transparent 25.5%, transparent 75%, #E8E8E8 75%),
			linear-gradient(-60deg, #E8E8E8 25%, transparent 25.5%, transparent 75%, #E8E8E8 75%),
			#F0F0F0
		`,
    preview:
      "linear-gradient(60deg, #D8D8D8 25%, transparent 25.5%, transparent 75%, #D8D8D8 75%), linear-gradient(-60deg, #D8D8D8 25%, transparent 25.5%, transparent 75%, #D8D8D8 75%), #F0F0F0",
  },
  {
    id: "stars",
    name: "Stars",
    type: "pattern",
    value: `
			radial-gradient(circle at 50% 50%, #FFFFFF 1px, transparent 1px),
			radial-gradient(circle at 30% 70%, #FFFFFF 0.5px, transparent 0.5px),
			radial-gradient(circle at 70% 30%, #FFFFFF 0.5px, transparent 0.5px),
			radial-gradient(circle at 90% 80%, #FFFFFF 1px, transparent 1px),
			radial-gradient(circle at 10% 20%, #FFFFFF 0.5px, transparent 0.5px),
			#0F172A
		`,
    preview: "radial-gradient(circle, #FFFFFF 1px, transparent 1px), #0F172A",
  },
  {
    id: "circuit",
    name: "Circuit",
    type: "pattern",
    value: `
			linear-gradient(#10B98120 1px, transparent 1px),
			linear-gradient(90deg, #10B98120 1px, transparent 1px),
			radial-gradient(circle, #10B98140 2px, transparent 2px),
			#0D1117
		`,
    preview:
      "linear-gradient(#10B98150 1px, transparent 1px), linear-gradient(90deg, #10B98150 1px, transparent 1px), #0D1117",
  },
  {
    id: "brick",
    name: "Brick",
    type: "pattern",
    value: `
			linear-gradient(#C4A98420 1px, transparent 1px),
			linear-gradient(90deg, #C4A98420 1px, transparent 1px),
			#D4B896
		`,
    preview:
      "linear-gradient(#00000020 1px, transparent 1px), linear-gradient(90deg, #00000020 1px, transparent 1px), #D4B896",
  },
  {
    id: "neon-grid",
    name: "Neon Grid",
    type: "pattern",
    value: `
			linear-gradient(#FF00FF15 1px, transparent 1px),
			linear-gradient(90deg, #00FFFF15 1px, transparent 1px),
			#0a0a0a
		`,
    preview:
      "linear-gradient(#FF00FF40 1px, transparent 1px), linear-gradient(90deg, #00FFFF40 1px, transparent 1px), #0a0a0a",
  },
  {
    id: "retro-lines",
    name: "Retro Lines",
    type: "pattern",
    value: `
			repeating-linear-gradient(
				0deg,
				#FF6B6B10,
				#FF6B6B10 2px,
				transparent 2px,
				transparent 4px
			),
			#2D2D2D
		`,
    preview:
      "repeating-linear-gradient(0deg, #FF6B6B30, #FF6B6B30 2px, transparent 2px, transparent 4px), #2D2D2D",
  },
];

export type BackgroundFit = "cover" | "contain" | "fill" | "center";

export const backgroundFitOptions: {
  id: BackgroundFit;
  name: string;
  description: string;
}[] = [
  { id: "cover", name: "Cover", description: "Fill screen, may crop edges" },
  {
    id: "contain",
    name: "Fit",
    description: "Show full image, may have borders",
  },
  { id: "fill", name: "Stretch", description: "Stretch to fill, may distort" },
  { id: "center", name: "Center", description: "Original size, centered" },
];

// Animated background types
export type AnimatedBackgroundEffect =
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
export type AnimatedBackgroundIntensity = "subtle" | "medium" | "high";

export interface AnimatedBackgroundSettings {
  effect: AnimatedBackgroundEffect;
  intensity: AnimatedBackgroundIntensity;
  colors: string[];
  speed: number; // 0.5-2
}

// Boot screen settings
export type BootAnimation =
  | "terminal"
  | "spinner"
  | "progress"
  | "pulse"
  | "glitch";

export interface BootScreenSettings {
  logo: {
    type: "default" | "custom";
    customSvg?: string;
    color?: string;
  };
  animation: BootAnimation;
  messages: {
    enabled: boolean;
    custom: string[];
  };
  colors: {
    background: string;
    text: string;
    accent: string;
  };
  duration: number; // seconds (1-10)
}

// Cursor pack settings
export type CursorPackId = "system" | "minimal" | "retro" | "modern" | "custom";

export interface CursorSettings {
  packId: CursorPackId;
  customCursors?: {
    default?: string;
    pointer?: string;
    text?: string;
    grab?: string;
    loading?: string;
  };
}

// List of dark backgrounds that need light text
const darkBackgroundIds = new Set([
  "dark-mode",
  "midnight",
  "slate",
  "charcoal",
  "navy",
  "deep-purple",
  "aurora",
  "cosmic",
  "night-fade",
  "deep-space",
  "purple-haze",
  "midnight-city",
  "dark-grid",
  "dark-dots",
  "blueprint",
  "carbon-fiber",
  "stars",
  "circuit",
  "neon-grid",
  "retro-lines",
]);

// Helper to determine if a background is dark (needs light text)
export function isBackgroundDark(backgroundId: string): boolean {
  return darkBackgroundIds.has(backgroundId);
}

// Helper to get background CSS
export function getBackgroundCSS(
  backgroundId: string,
  customUrl?: string | null,
  isDark?: boolean,
): { background: string; backgroundSize?: string } {
  // Handle custom background
  if (backgroundId === "custom" && customUrl) {
    return {
      background: `url(${customUrl})`,
      backgroundSize: "cover",
    };
  }

  // If dark mode is active and background is light, use a dark equivalent
  if (isDark && !isBackgroundDark(backgroundId)) {
    const bg = desktopBackgrounds.find((b) => b.id === backgroundId);
    if (bg && (bg.type === "solid" || bg.type === "gradient")) {
      return { background: "#1c1c1e" };
    }
  }

  const bg = desktopBackgrounds.find((b) => b.id === backgroundId);
  if (!bg) return { background: "#E5E5E5" };

  if (bg.type === "pattern") {
    const patternSizes: Record<string, string> = {
      dots: "20px 20px",
      grid: "40px 40px",
      "diagonal-lines": "20px 20px",
      checkerboard: "40px 40px",
      zigzag: "40px 20px",
      honeycomb: "42px 60px",
      waves: "100px 50px",
      "cross-dots": "20px 20px, 20px 20px",
      "dark-grid": "40px 40px",
      "dark-dots": "20px 20px",
      stripes: "20px 20px",
      "diagonal-stripes": "20px 20px",
      blueprint: "40px 40px",
      "carbon-fiber": "20px 20px",
      "paper-texture": "200px 200px",
      diamonds: "40px 40px",
      triangles: "40px 40px",
      stars: "50px 50px",
      circuit: "40px 40px",
      brick: "40px 20px",
      "neon-grid": "40px 40px",
      "retro-lines": "4px 4px",
    };
    return {
      background: bg.value,
      backgroundSize: patternSizes[bg.id] || "20px 20px",
    };
  }

  return { background: bg.value };
}
