import { defineConfig } from "tsdown";

export default defineConfig({
  entry: {
    index: "src/index.ts",
    "adapters/ai-sdk": "src/adapters/ai-sdk.ts",
    "adapters/openai-agents": "src/adapters/openai-agents.ts",
  },
  format: ["esm", "cjs"],
  dts: true,
  clean: true,
  target: "es2022",
  treeshake: true,
  sourcemap: true,
  external: ["ai", "zod", "openai"],
});
