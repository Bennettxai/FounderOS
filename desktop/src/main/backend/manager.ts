import { spawn, ChildProcess } from "child_process";
import path from "path";
import { app } from "electron";
import http from "http";
import { healthUrl } from "./health-url";

const DEFAULT_PORT = 8001;
const HEALTH_CHECK_INTERVAL = 5000;
const STARTUP_TIMEOUT = 30000;

export class BackendManager {
  private process: ChildProcess | null = null;
  private resourcesPath: string;
  private port: number = DEFAULT_PORT;
  private healthCheckTimer: NodeJS.Timeout | null = null;
  private isStarting = false;
  private restartCount = 0;
  private maxRestarts = 3;

  constructor(resourcesPath: string) {
    this.resourcesPath = resourcesPath;
    this.port = Number(process.env.BUSINESSOS_BACKEND_PORT) || DEFAULT_PORT;
  }

  /**
   * Get the backend server URL
   */
  getUrl(): string {
    return `http://localhost:${this.port}`;
  }

  /**
   * Get the backend server port
   */
  getPort(): number {
    return this.port;
  }

  /**
   * Check if the backend is running
   */
  isRunning(): boolean {
    return this.process !== null && !this.process.killed;
  }

  /**
   * Get the path to the backend binary
   */
  private getBinaryPath(): string {
    const platform = process.platform;
    const arch = process.arch;

    let binaryName = "businessos-server";
    if (platform === "win32") {
      binaryName += ".exe";
    }

    // Map Node.js arch to Go arch naming
    const archMap: Record<string, string> = {
      x64: "x64",
      arm64: "arm64",
    };

    const goArch = archMap[arch] || arch;
    const platformDir = `${platform}-${goArch}`;

    return path.join(this.resourcesPath, "bin", platformDir, binaryName);
  }

  /**
   * Check backend health
   */
  private async checkHealth(): Promise<boolean> {
    return new Promise((resolve) => {
      const req = http.get(
        healthUrl(this.port),
        { timeout: 2000 },
        (res) => {
          resolve(res.statusCode === 200);
        },
      );

      req.on("error", () => {
        resolve(false);
      });

      req.on("timeout", () => {
        req.destroy();
        resolve(false);
      });
    });
  }

  /**
   * Wait for backend to be healthy
   */
  private async waitForHealthy(timeout: number): Promise<boolean> {
    const startTime = Date.now();

    while (Date.now() - startTime < timeout) {
      const isHealthy = await this.checkHealth();
      if (isHealthy) {
        return true;
      }
      await new Promise((resolve) => setTimeout(resolve, 500));
    }

    return false;
  }

  /**
   * Start the backend server
   */
  async start(): Promise<void> {
    if (this.isStarting) {
      console.log("Backend is already starting...");
      return;
    }

    if (this.isRunning()) {
      console.log("Backend is already running");
      return;
    }

    this.isStarting = true;

    try {
      if (!app.isPackaged) {
        this.port = Number(process.env.BUSINESSOS_BACKEND_PORT) || DEFAULT_PORT;
        if (await this.checkHealth()) {
          console.log(
            `Development mode: Using external backend at ${this.getUrl()}`,
          );
          return;
        }

        throw new Error(
          `Development backend is not healthy at ${this.getUrl()}. Run scripts/dev-local.sh or make dev-local-start first.`,
        );
      }

      const binaryPath = this.getBinaryPath();
      console.log(`Starting backend from: ${binaryPath}`);

      // Check if binary exists
      const fs = await import("fs");
      if (!fs.existsSync(binaryPath)) {
        throw new Error(`Backend binary not found at: ${binaryPath}`);
      }

      // Get user data path for SQLite database
      const userDataPath = app.getPath("userData");
      const dbPath = path.join(userDataPath, "businessos.db");

      // Spawn the backend process
      this.process = spawn(binaryPath, [], {
        env: {
          ...process.env,
          PORT: String(this.port),
          // The Go backend reads SERVER_PORT, not PORT — without this it
          // falls back to :8001 and collides with any local docker stack.
          SERVER_PORT: String(this.port),
          DATABASE_MODE: "sqlite",
          DATABASE_PATH: dbPath,
          ELECTRON_MODE: "true",
        },
        stdio: ["ignore", "pipe", "pipe"],
        detached: false,
      });

      // Handle stdout
      this.process.stdout?.on("data", (data) => {
        console.log(`[Backend] ${data.toString().trim()}`);
      });

      // Handle stderr
      this.process.stderr?.on("data", (data) => {
        console.error(`[Backend Error] ${data.toString().trim()}`);
      });

      // Handle process exit
      this.process.on("exit", (code, signal) => {
        console.log(`Backend exited with code ${code}, signal ${signal}`);
        this.process = null;

        // Attempt restart if unexpected exit
        if (code !== 0 && this.restartCount < this.maxRestarts) {
          this.restartCount++;
          console.log(
            `Attempting restart ${this.restartCount}/${this.maxRestarts}...`,
          );
          setTimeout(() => this.start(), 1000);
        }
      });

      // Handle errors
      this.process.on("error", (error) => {
        console.error("Backend process error:", error);
        this.process = null;
      });

      // Wait for backend to be healthy
      console.log("Waiting for backend to be healthy...");
      const isHealthy = await this.waitForHealthy(STARTUP_TIMEOUT);

      if (!isHealthy) {
        throw new Error("Backend failed to start within timeout");
      }

      console.log("Backend is healthy and ready");
      this.restartCount = 0; // Reset restart count on successful start

      // Start health check monitoring
      this.startHealthMonitoring();
    } finally {
      this.isStarting = false;
    }
  }

  /**
   * Stop the backend server
   */
  async stop(): Promise<void> {
    this.stopHealthMonitoring();

    if (!this.process) {
      return;
    }

    console.log("Stopping backend...");

    return new Promise((resolve) => {
      if (!this.process) {
        resolve();
        return;
      }

      // Set up timeout for forceful kill
      const killTimeout = setTimeout(() => {
        if (this.process && !this.process.killed) {
          console.log("Force killing backend...");
          this.process.kill("SIGKILL");
        }
      }, 5000);

      this.process.once("exit", () => {
        clearTimeout(killTimeout);
        this.process = null;
        console.log("Backend stopped");
        resolve();
      });

      // Try graceful shutdown first
      this.process.kill("SIGTERM");
    });
  }

  /**
   * Restart the backend server
   */
  async restart(): Promise<void> {
    await this.stop();
    await this.start();
  }

  /**
   * Start health monitoring
   */
  private startHealthMonitoring(): void {
    this.healthCheckTimer = setInterval(async () => {
      const isHealthy = await this.checkHealth();
      if (!isHealthy && this.process) {
        console.warn("Backend health check failed");
        // Could trigger restart or notify user
      }
    }, HEALTH_CHECK_INTERVAL);
  }

  /**
   * Stop health monitoring
   */
  private stopHealthMonitoring(): void {
    if (this.healthCheckTimer) {
      clearInterval(this.healthCheckTimer);
      this.healthCheckTimer = null;
    }
  }
}
