import { spawn, ChildProcess } from "child_process";
import path from "path";
import net from "net";
import http from "http";
import { app } from "electron";
import { existsSync, mkdirSync, readFileSync } from "fs";
import { loadOrCreateConnectorKey } from "./credential-key";

// Manages the BUNDLED OptimalEngine (Elixir release) that ships inside the app.
// Every BusinessOS has a built-in engine; this spawns it on a free local port
// with a per-user data directory, so it never collides with a developer's own
// engine on :4200 and a downloaded user gets a fresh, self-contained instance.
//
// Users can still point BusinessOS at their OWN external engine via the
// connection settings - this bundled one is the always-present default.

const PREFERRED_PORT = 4200;
const STARTUP_TIMEOUT = 45000;

export class EngineManager {
  private process: ChildProcess | null = null;
  private resourcesPath: string;
  private port = 0;
  private starting = false;

  constructor(resourcesPath: string) {
    this.resourcesPath = resourcesPath;
  }

  getPort(): number {
    return this.port;
  }

  getUrl(): string {
    return this.port ? `http://127.0.0.1:${this.port}` : "";
  }

  isRunning(): boolean {
    return this.process !== null && !this.process.killed;
  }

  isAvailable(): boolean {
    return existsSync(this.getBinaryPath());
  }

  // Path to the bundled engine launcher (per-arch release under resources/engine).
  private getBinaryPath(): string {
    const platform = process.platform;
    const arch = process.arch === "arm64" ? "arm64" : "x64";
    const bin = platform === "win32" ? "optimal.bat" : "optimal";
    return path.join(
      this.resourcesPath,
      "engine",
      `${platform}-${arch}`,
      "bin",
      bin,
    );
  }

  // Find a free TCP port, preferring 4200 but stepping past a busy one (e.g. a
  // developer already running their own engine there).
  private async findPort(start = PREFERRED_PORT): Promise<number> {
    for (let p = start; p < start + 40; p++) {
      const free = await new Promise<boolean>((resolve) => {
        const srv = net.createServer();
        srv.once("error", () => resolve(false));
        srv.once("listening", () => srv.close(() => resolve(true)));
        srv.listen(p, "127.0.0.1");
      });
      if (free) return p;
    }
    return 0;
  }

  private checkHealth(): Promise<boolean> {
    return new Promise((resolve) => {
      const req = http.get(
        `${this.getUrl()}/api/health`,
        { timeout: 1500 },
        (res) => {
          res.resume();
          resolve(res.statusCode === 200);
        },
      );
      req.on("error", () => resolve(false));
      req.on("timeout", () => {
        req.destroy();
        resolve(false);
      });
    });
  }

  private async waitForHealthy(timeoutMs: number): Promise<boolean> {
    const deadline = Date.now() + timeoutMs;
    while (Date.now() < deadline) {
      if (await this.checkHealth()) return true;
      await new Promise((r) => setTimeout(r, 800));
    }
    return false;
  }

  // Start the bundled engine. NON-FATAL: if it can't start, the app keeps
  // working (cloud mode / external engine); the built-in engine is a bonus.
  async start(): Promise<boolean> {
    if (this.isRunning() || this.starting) return true;
    this.starting = true;
    try {
      const binaryPath = this.getBinaryPath();
      if (!existsSync(binaryPath)) {
        console.warn(`[Engine] bundled engine not found at ${binaryPath}`);
        return false;
      }

      this.port = await this.findPort();
      if (!this.port) {
        console.warn("[Engine] no free port found for the bundled engine");
        return false;
      }

      // Per-user, self-contained data directory (never a developer path).
      const dataDir = path.join(app.getPath("userData"), "optimal-engine");
      const workspaces = path.join(dataDir, "workspaces");
      const cache = path.join(dataDir, "cache");
      for (const d of [dataDir, workspaces, cache]) {
        mkdirSync(d, { recursive: true });
      }
      const connectorKey = loadOrCreateConnectorKey(
        dataDir,
        process.env.CONNECTOR_KEY,
      );

      // Feed the user's saved model connection into the engine. The engine's
      // embeddings/generation use Ollama (config.exs reads OLLAMA_HOST), so a
      // custom Ollama URL is applied here. Anthropic/OpenAI keys are persisted
      // for future engine provider support (the engine has no cloud LLM provider
      // yet), so they are exported as OPTIMAL_* hints without breaking startup.
      const modelEnv: Record<string, string> = {};
      try {
        const cfgPath = path.join(app.getPath("userData"), "model-config.json");
        if (existsSync(cfgPath)) {
          const cfg = JSON.parse(readFileSync(cfgPath, "utf8")) as {
            provider?: string;
            apiKey?: string;
            ollamaUrl?: string;
          };
          if (cfg.ollamaUrl) modelEnv.OLLAMA_HOST = cfg.ollamaUrl;
          if (cfg.provider === "anthropic" && cfg.apiKey)
            modelEnv.ANTHROPIC_API_KEY = cfg.apiKey;
          if (cfg.provider === "openai" && cfg.apiKey)
            modelEnv.OPENAI_API_KEY = cfg.apiKey;
        }
      } catch (e) {
        console.warn("[Engine] could not read model-config.json:", e);
      }

      console.log(`[Engine] starting bundled engine on :${this.port}`);
      this.process = spawn(binaryPath, ["start"], {
        env: {
          ...process.env,
          ...modelEnv,
          OPTIMAL_API_ENABLED: "true",
          OPTIMAL_API_PORT: String(this.port),
          OPTIMAL_API_INTERFACE: "127.0.0.1",
          OPTIMAL_ENGINE_DB: path.join(dataDir, "index.db"),
          OPTIMAL_ENGINE_ROOT: workspaces,
          OPTIMAL_ENGINE_CACHE: cache,
          OPTIMAL_KNOWLEDGE_BACKEND: "rocksdb",
          OPTIMAL_KNOWLEDGE_ROCKSDB_PATH: path.join(
            dataDir,
            "knowledge-rocksdb",
          ),
          CONNECTOR_KEY: connectorKey,
          RELEASE_TMP: dataDir,
        },
        stdio: ["ignore", "pipe", "pipe"],
        detached: false,
      });

      this.process.stdout?.on("data", (d) =>
        console.log(`[Engine] ${d.toString().trim()}`),
      );
      this.process.stderr?.on("data", (d) =>
        console.error(`[Engine:err] ${d.toString().trim()}`),
      );
      this.process.on("exit", (code) => {
        console.log(`[Engine] exited (${code})`);
        this.process = null;
        this.port = 0; // avoid getUrl() handing out a dead port after a crash
      });
      this.process.on("error", (err) => {
        console.error("[Engine] process error:", err);
        this.process = null;
        this.port = 0;
      });

      const healthy = await this.waitForHealthy(STARTUP_TIMEOUT);
      if (!healthy) {
        console.warn("[Engine] did not become healthy in time");
        return false;
      }
      console.log(`[Engine] healthy at ${this.getUrl()}`);
      return true;
    } catch (err) {
      console.error("[Engine] start failed (non-fatal):", err);
      return false;
    } finally {
      this.starting = false;
    }
  }

  async stop(): Promise<void> {
    if (!this.process) return;
    const binaryPath = this.getBinaryPath();
    try {
      // Graceful release stop, then hard kill as a fallback.
      if (existsSync(binaryPath)) {
        spawn(binaryPath, ["stop"], {
          env: { ...process.env, OPTIMAL_API_PORT: String(this.port) },
          stdio: "ignore",
        });
      }
    } catch {
      // ignore
    }
    try {
      this.process.kill();
    } catch {
      // ignore
    }
    this.process = null;
  }
}
