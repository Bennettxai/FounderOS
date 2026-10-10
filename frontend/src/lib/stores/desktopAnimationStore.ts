// Desktop Animation Store — Window animation state, transitions, animation configs

// Window animation settings - expanded options
export type WindowAnimationType =
  | "none"
  | "fade"
  | "scale"
  | "slide"
  | "bounce"
  | "zoom"
  | "flip"
  | "elastic"
  | "glitch"
  | "blur"
  | "pop"
  | "drop";
export type AnimationSpeed = "fast" | "normal" | "slow";

export interface WindowAnimationSettings {
  openAnimation: WindowAnimationType;
  closeAnimation: WindowAnimationType;
  minimizeAnimation: WindowAnimationType | "genie" | "shrink";
  speed: AnimationSpeed;
}

// Window animation descriptions for UI
export const windowAnimationOptions: {
  id: WindowAnimationType;
  name: string;
  description: string;
}[] = [
  { id: "none", name: "None", description: "No animation" },
  { id: "fade", name: "Fade", description: "Simple fade in/out" },
  { id: "scale", name: "Scale", description: "Grow from center" },
  { id: "slide", name: "Slide", description: "Slide from edge" },
  { id: "bounce", name: "Bounce", description: "Bouncy spring effect" },
  { id: "zoom", name: "Zoom", description: "Quick zoom burst" },
  { id: "flip", name: "Flip", description: "3D card flip" },
  { id: "elastic", name: "Elastic", description: "Stretchy rubber band" },
  { id: "glitch", name: "Glitch", description: "Digital glitch effect" },
  { id: "blur", name: "Blur", description: "Focus blur transition" },
  { id: "pop", name: "Pop", description: "Bubble pop effect" },
  { id: "drop", name: "Drop", description: "Drop from above" },
];
