import type { Config } from "tailwindcss";

const config: Config = {
  content: ["./src/**/*.{ts,tsx,html}"],
  darkMode: "class",
  theme: {
    extend: {
      colors: {
        bg: {
          DEFAULT: "#0a0a0f",
          secondary: "#111118",
          elevated: "#1a1a24",
          border: "#2a2a38",
        },
        accent: {
          DEFAULT: "#7ea8ff",
          hover: "#a0bfff",
          muted: "rgba(126,168,255,0.15)",
        },
        text: {
          primary: "#e8e8f0",
          secondary: "#9090a8",
          muted: "#606078",
        },
        success: "#4ade80",
        warning: "#facc15",
        error: "#f87171",
      },
      fontFamily: {
        sans: [
          "system-ui",
          "-apple-system",
          "BlinkMacSystemFont",
          "Segoe UI",
          "sans-serif",
        ],
        mono: ["JetBrains Mono", "Fira Code", "Menlo", "monospace"],
      },
      borderRadius: {
        sm: "4px",
        DEFAULT: "6px",
        md: "8px",
        lg: "12px",
      },
    },
  },
  plugins: [],
};

export default config;
