// Desktop Icon Store — Icon definitions, styles, libraries, size presets, grid management

export type IconStyle =
  | "default"
  | "minimal"
  | "rounded"
  | "square"
  | "macos"
  | "macos-classic"
  | "outlined"
  | "retro"
  | "win95"
  | "glassmorphism"
  | "neon"
  | "flat"
  | "gradient"
  | "paper"
  | "pixel"
  | "frosted"
  | "terminal"
  | "glow"
  | "brutalist"
  | "depth"
  | "neumorphism"
  | "material"
  | "fluent"
  | "aero"
  | "aurora"
  | "crystal"
  | "holographic"
  | "vaporwave"
  | "cyberpunk"
  | "synthwave"
  | "matrix"
  | "glitch"
  | "chrome"
  | "rainbow"
  | "sketch"
  | "comic"
  | "watercolor"
  | "ios"
  | "android"
  | "windows11"
  | "amiga";

export type IconLibrary = "lucide" | "phosphor" | "tabler" | "heroicons";

// Icon Library actually controls line weight/rendering style, not different icon sets
// We only have Lucide icons, but these presets change how they're rendered
export const iconLibraries: {
  id: IconLibrary;
  name: string;
  description: string;
  preview: string;
}[] = [
  {
    id: "lucide",
    name: "Regular",
    description: "Balanced 2px strokes",
    preview: "stroke-[2px]",
  },
  {
    id: "phosphor",
    name: "Bold",
    description: "Thick 3px strokes with shadow",
    preview: "stroke-[3px] + shadow",
  },
  {
    id: "tabler",
    name: "Light",
    description: "Thin 1.2px strokes, subtle",
    preview: "stroke-[1.2px]",
  },
  {
    id: "heroicons",
    name: "Heavy",
    description: "Solid 2.5px strokes",
    preview: "stroke-[2.5px]",
  },
];

export const iconStyles: {
  id: IconStyle;
  name: string;
  description: string;
}[] = [
  // Modern Styles
  {
    id: "default",
    name: "Default",
    description: "Rounded corners with colored backgrounds",
  },
  {
    id: "minimal",
    name: "Minimal",
    description: "Simple icons without backgrounds",
  },
  { id: "rounded", name: "Rounded", description: "Circular icon backgrounds" },
  {
    id: "square",
    name: "Square",
    description: "Square icons with sharp corners",
  },
  { id: "macos", name: "macOS", description: "macOS-style squircle icons" },
  { id: "glassmorphism", name: "Glass", description: "Frosted glass effect" },
  {
    id: "frosted",
    name: "Frosted",
    description: "Clean frosted glass with blur",
  },
  { id: "flat", name: "Flat", description: "Flat design with no shadows" },
  {
    id: "paper",
    name: "Paper",
    description: "Paper card style with soft shadows",
  },
  { id: "depth", name: "Depth", description: "Layered 3D depth shadows" },
  {
    id: "neumorphism",
    name: "Neumorphism",
    description: "Soft 3D with inset/outset shadows",
  },
  {
    id: "material",
    name: "Material",
    description: "Google Material Design elevation",
  },
  {
    id: "fluent",
    name: "Fluent",
    description: "Microsoft Fluent Design acrylic",
  },
  {
    id: "aero",
    name: "Aero",
    description: "Windows Vista/7 glass effect",
  },

  // Classic Styles
  {
    id: "macos-classic",
    name: "Mac Classic",
    description: "Classic Mac OS 9 platinum style",
  },
  { id: "retro", name: "Retro", description: "Classic retro computer style" },
  {
    id: "win95",
    name: "Win95",
    description: "Windows 95 style with 3D borders",
  },
  { id: "pixel", name: "Pixel", description: "8-bit pixel art style" },
  { id: "ios", name: "iOS", description: "iOS app icon rounded square" },
  {
    id: "android",
    name: "Android",
    description: "Material You rounded square",
  },
  {
    id: "windows11",
    name: "Windows 11",
    description: "Modern Windows 11 rounded",
  },
  { id: "amiga", name: "Amiga", description: "Amiga Workbench retro style" },

  // Creative Styles
  {
    id: "outlined",
    name: "Outlined",
    description: "Icons with border outlines",
  },
  { id: "neon", name: "Neon", description: "Glowing neon style" },
  {
    id: "gradient",
    name: "Gradient",
    description: "Gradient background style",
  },
  { id: "glow", name: "Glow", description: "Soft colored glow aura effect" },
  {
    id: "terminal",
    name: "Terminal",
    description: "Green on black hacker aesthetic",
  },
  {
    id: "brutalist",
    name: "Brutalist",
    description: "Bold raw design with thick borders",
  },
  {
    id: "aurora",
    name: "Aurora",
    description: "Animated gradient shimmer effect",
  },
  {
    id: "crystal",
    name: "Crystal",
    description: "Gem-like faceted appearance",
  },
  {
    id: "holographic",
    name: "Holographic",
    description: "Rainbow shifting iridescent",
  },
  {
    id: "vaporwave",
    name: "Vaporwave",
    description: "80s/90s pink and cyan aesthetic",
  },
  {
    id: "cyberpunk",
    name: "Cyberpunk",
    description: "Neon with scan lines",
  },
  {
    id: "synthwave",
    name: "Synthwave",
    description: "Retro futuristic purple/pink",
  },
  { id: "matrix", name: "Matrix", description: "Green code rain style" },
  {
    id: "glitch",
    name: "Glitch",
    description: "Digital glitch distortion effect",
  },
  {
    id: "chrome",
    name: "Chrome",
    description: "Metallic reflective surface",
  },
  {
    id: "rainbow",
    name: "Rainbow",
    description: "Animated rainbow spectrum",
  },
  { id: "sketch", name: "Sketch", description: "Hand-drawn outline style" },
  {
    id: "comic",
    name: "Comic",
    description: "Comic book thick black borders",
  },
  {
    id: "watercolor",
    name: "Watercolor",
    description: "Soft blurred watercolor paint",
  },
];

// Icon size presets for the slider
export const iconSizePresets = [
  { value: 32, label: "Tiny" },
  { value: 48, label: "Small" },
  { value: 64, label: "Medium" },
  { value: 80, label: "Large" },
  { value: 96, label: "Huge" },
  { value: 128, label: "Massive" },
];
