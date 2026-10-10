import { getDatabase, DatabaseHelper, generateUUID } from "../database/sqlite";
import { SyncQueue } from "./queue";
import { ConflictResolver, ConflictResolution } from "./conflict";
import { net } from "electron";

/**
 * Sync status for records
 */
export type SyncStatus =
  | "synced"
  | "pending_create"
  | "pending_update"
  | "pending_delete"
  | "conflict";

/**
 * Tables that support sync
 */
export const SYNC_TABLES = [
  "contexts",
  "conversations",
  "messages",
  "projects",
  "project_notes",
  "artifacts",
  "artifact_versions",
  "nodes",
  "node_metrics",
  "team_members",
  "team_member_activities",
  "tasks",
  "focus_items",
  "daily_logs",
  "user_settings",
  "clients",
  "client_contacts",
  "client_interactions",
  "client_deals",
  "calendar_events",
] as const;

export type SyncTable = (typeof SYNC_TABLES)[number];

/**
 * Sync engine configuration
 */
export interface SyncConfig {
  serverUrl: string;
  syncInterval: number; // in milliseconds
  debounceTime: number; // in milliseconds
  maxRetries: number;
  workspaceId?: string; // active workspace for shared (BusinessOSSync) records
}

/**
 * Sync result
 */
export interface SyncResult {
  success: boolean;
  synced: number;
  failed: number;
  conflicts: number;
  errors: string[];
}

/**
 * Main sync engine for bidirectional sync
 */
export class SyncEngine {
  private config: SyncConfig;
  private db: DatabaseHelper;
  private queue: SyncQueue;
  private conflictResolver: ConflictResolver;
  private syncTimer: NodeJS.Timeout | null = null;
  private debounceTimer: NodeJS.Timeout | null = null;
  private isSyncing = false;
  private listeners: ((status: SyncStatus, detail?: any) => void)[] = [];

  constructor(config: Partial<SyncConfig> = {}) {
    this.config = {
      serverUrl: config.serverUrl || "https://app.businessos.dev",
      syncInterval: config.syncInterval || 5 * 60 * 1000, // 5 minutes
      debounceTime: config.debounceTime || 500,
      maxRetries: config.maxRetries || 3,
      workspaceId: config.workspaceId,
    };

    this.db = new DatabaseHelper();
    this.queue = new SyncQueue();
    this.conflictResolver = new ConflictResolver();
  }

  /**
   * Set the active workspace. Shared (BusinessOSSync) records are scoped to this
   * workspace on push (so they're stamped) and pull (so teammates' shared docs
   * come down). Triggers a sync when the workspace changes.
   */
  setWorkspaceId(workspaceId: string | undefined): void {
    if (this.config.workspaceId === workspaceId) return;
    this.config.workspaceId = workspaceId;
    if (workspaceId) this.scheduleSync();
  }

  /**
   * Common headers for cloud sync requests: workspace scope + auth cookie.
   */
  private applyAuthHeaders(request: Electron.ClientRequest): void {
    if (this.config.workspaceId) {
      request.setHeader("X-Workspace-ID", this.config.workspaceId);
    }
  }

  /**
   * Start the sync engine
   */
  start(): void {
    console.log("Starting sync engine...");

    // Initial sync
    this.sync();

    // Set up periodic sync
    this.syncTimer = setInterval(() => {
      this.sync();
    }, this.config.syncInterval);
  }

  /**
   * Stop the sync engine
   */
  stop(): void {
    console.log("Stopping sync engine...");

    if (this.syncTimer) {
      clearInterval(this.syncTimer);
      this.syncTimer = null;
    }

    if (this.debounceTimer) {
      clearTimeout(this.debounceTimer);
      this.debounceTimer = null;
    }
  }

  /**
   * Schedule a sync (debounced)
   */
  scheduleSync(): void {
    if (this.debounceTimer) {
      clearTimeout(this.debounceTimer);
    }

    this.debounceTimer = setTimeout(() => {
      this.sync();
    }, this.config.debounceTime);
  }

  /**
   * Add a listener for sync events
   */
  addListener(
    callback: (status: SyncStatus, detail?: any) => void,
  ): () => void {
    this.listeners.push(callback);
    return () => {
      this.listeners = this.listeners.filter((l) => l !== callback);
    };
  }

  /**
   * Emit sync event
   */
  private emit(status: SyncStatus, detail?: any): void {
    for (const listener of this.listeners) {
      listener(status, detail);
    }
  }

  /**
   * Check if online
   */
  private async isOnline(): Promise<boolean> {
    return new Promise((resolve) => {
      if (!net.isOnline()) {
        resolve(false);
        return;
      }

      // Try to reach the server
      const request = net.request({
        method: "HEAD",
        url: `${this.config.serverUrl}/health`,
      });

      request.on("response", (response) => {
        resolve(response.statusCode === 200);
      });

      request.on("error", () => {
        resolve(false);
      });

      setTimeout(() => {
        resolve(false);
      }, 5000);

      request.end();
    });
  }

  /**
   * Perform a full sync
   */
  async sync(): Promise<SyncResult> {
    if (this.isSyncing) {
      console.log("Sync already in progress, skipping...");
      return { success: true, synced: 0, failed: 0, conflicts: 0, errors: [] };
    }

    this.isSyncing = true;
    this.emit("pending_update", { phase: "starting" });

    const result: SyncResult = {
      success: true,
      synced: 0,
      failed: 0,
      conflicts: 0,
      errors: [],
    };

    try {
      // Check if online
      const online = await this.isOnline();
      if (!online) {
        console.log("Offline, skipping sync");
        this.emit("synced", { offline: true });
        return result;
      }

      // Push local changes to server
      const pushResult = await this.pushChanges();
      result.synced += pushResult.synced;
      result.failed += pushResult.failed;
      result.errors.push(...pushResult.errors);

      // Pull changes from server
      const pullResult = await this.pullChanges();
      result.synced += pullResult.synced;
      result.conflicts += pullResult.conflicts;
      result.errors.push(...pullResult.errors);

      // Update sync metadata
      this.updateSyncMetadata();

      result.success = result.failed === 0 && result.errors.length === 0;
      this.emit("synced", result);
    } catch (error) {
      console.error("Sync error:", error);
      result.success = false;
      result.errors.push(String(error));
      this.emit("conflict", { error: String(error) });
    } finally {
      this.isSyncing = false;
    }

    return result;
  }

  /**
   * Push local changes to the server
   */
  private async pushChanges(): Promise<{
    synced: number;
    failed: number;
    errors: string[];
  }> {
    const result = { synced: 0, failed: 0, errors: [] as string[] };

    for (const table of SYNC_TABLES) {
      const pending = this.db.getPendingSync(table);

      for (const record of pending) {
        try {
          const success = await this.pushRecord(table, record);
          if (success) {
            result.synced++;
          } else {
            result.failed++;
          }
        } catch (error) {
          result.failed++;
          result.errors.push(`Failed to sync ${table}/${record.id}: ${error}`);
        }
      }
    }

    return result;
  }

  /**
   * Push a single record to the server
   */
  private async pushRecord(table: string, record: any): Promise<boolean> {
    const { sync_status, sync_version, last_synced_at, ...data } = record;

    let endpoint: string;
    let method: string;

    switch (sync_status) {
      case "pending_create":
        endpoint = `/api/${table}`;
        method = "POST";
        break;
      case "pending_update":
        endpoint = `/api/${table}/${record.server_id || record.id}`;
        method = "PUT";
        break;
      case "pending_delete":
        endpoint = `/api/${table}/${record.server_id || record.id}`;
        method = "DELETE";
        break;
      default:
        return true; // Already synced
    }

    return new Promise((resolve) => {
      const request = net.request({
        method,
        url: `${this.config.serverUrl}${endpoint}`,
        useSessionCookies: true, // send the better-auth session cookie
      });

      request.setHeader("Content-Type", "application/json");
      this.applyAuthHeaders(request);

      let responseData = "";

      request.on("response", (response) => {
        response.on("data", (chunk) => {
          responseData += chunk.toString();
        });

        response.on("end", () => {
          if (
            response.statusCode &&
            response.statusCode >= 200 &&
            response.statusCode < 300
          ) {
            // Mark as synced
            let serverId = record.server_id;
            if (sync_status === "pending_create" && responseData) {
              try {
                const serverRecord = JSON.parse(responseData);
                serverId = serverRecord.id;
              } catch {
                // Ignore parse errors
              }
            }

            if (sync_status === "pending_delete") {
              // Actually delete the local record
              this.db.delete(table, record.id);
            } else {
              this.db.markSynced(table, record.id, serverId);
            }

            resolve(true);
          } else if (response.statusCode === 409) {
            // Conflict - need to resolve
            this.handleConflict(table, record, JSON.parse(responseData));
            resolve(false);
          } else {
            console.error(
              `Failed to push ${table}/${record.id}: ${response.statusCode}`,
            );
            resolve(false);
          }
        });
      });

      request.on("error", (error) => {
        console.error(`Network error pushing ${table}/${record.id}:`, error);
        resolve(false);
      });

      if (method !== "DELETE") {
        request.write(JSON.stringify(data));
      }

      request.end();
    });
  }

  /**
   * Pull changes from the server
   */
  private async pullChanges(): Promise<{
    synced: number;
    conflicts: number;
    errors: string[];
  }> {
    const result = { synced: 0, conflicts: 0, errors: [] as string[] };

    // Get last sync time
    const metadata = this.db.query<any>(
      "SELECT * FROM sync_metadata WHERE id = 1",
    )[0];
    const lastSync = metadata?.last_sync_at || "1970-01-01T00:00:00Z";

    for (const table of SYNC_TABLES) {
      try {
        const changes = await this.fetchServerChanges(table, lastSync);
        for (const serverRecord of changes) {
          const localRecord = this.db.query<any>(
            `SELECT * FROM ${table} WHERE server_id = ?`,
            [serverRecord.id],
          )[0];

          if (localRecord) {
            // Check for conflicts
            if (localRecord.sync_status !== "synced") {
              const resolution = this.conflictResolver.resolve(
                table as SyncTable,
                localRecord,
                serverRecord,
              );
              await this.applyResolution(
                table,
                localRecord,
                serverRecord,
                resolution,
              );
              if (resolution === "conflict") {
                result.conflicts++;
              } else {
                result.synced++;
              }
            } else {
              // Update local with server data
              this.updateLocalRecord(table, localRecord.id, serverRecord);
              result.synced++;
            }
          } else {
            // Insert new record
            this.insertServerRecord(table, serverRecord);
            result.synced++;
          }
        }
      } catch (error) {
        result.errors.push(`Failed to pull ${table}: ${error}`);
      }
    }

    return result;
  }

  /**
   * Fetch changes from server since last sync
   */
  private async fetchServerChanges(
    table: string,
    since: string,
  ): Promise<any[]> {
    return new Promise((resolve, reject) => {
      const request = net.request({
        method: "GET",
        url: `${this.config.serverUrl}/api/${table}/sync?since=${encodeURIComponent(since)}`,
        useSessionCookies: true, // send the better-auth session cookie
      });
      this.applyAuthHeaders(request);

      let data = "";

      request.on("response", (response) => {
        response.on("data", (chunk) => {
          data += chunk.toString();
        });

        response.on("end", () => {
          if (response.statusCode === 200) {
            try {
              resolve(JSON.parse(data));
            } catch {
              resolve([]);
            }
          } else {
            resolve([]);
          }
        });
      });

      request.on("error", reject);
      request.end();
    });
  }

  /**
   * Handle a conflict
   */
  private handleConflict(
    table: string,
    localRecord: any,
    serverRecord: any,
  ): void {
    this.db.run(`UPDATE ${table} SET sync_status = 'conflict' WHERE id = ?`, [
      localRecord.id,
    ]);
  }

  /**
   * Apply conflict resolution
   */
  private async applyResolution(
    table: string,
    localRecord: any,
    serverRecord: any,
    resolution: ConflictResolution,
  ): Promise<void> {
    switch (resolution) {
      case "local":
        // Keep local, push to server later
        break;
      case "server":
        // Accept server version
        this.updateLocalRecord(table, localRecord.id, serverRecord);
        break;
      case "merge":
        // Merge non-conflicting fields (simplified)
        const merged = { ...serverRecord, ...localRecord };
        this.updateLocalRecord(table, localRecord.id, merged);
        break;
      case "conflict":
        // Mark as conflict for manual resolution
        this.db.markForSync(table, localRecord.id, "conflict" as any);
        break;
    }
  }

  /**
   * Update a local record with server data
   */
  private updateLocalRecord(
    table: string,
    localId: string,
    serverRecord: any,
  ): void {
    const { id, created_at, ...data } = serverRecord;
    data.server_id = id;
    data.sync_status = "synced";
    data.last_synced_at = new Date().toISOString();

    this.db.update(table, localId, data);
  }

  /**
   * Insert a new record from server
   */
  private insertServerRecord(table: string, serverRecord: any): void {
    const { id, ...data } = serverRecord;
    const localId = generateUUID();

    this.db.insert(table, {
      id: localId,
      server_id: id,
      ...data,
      sync_status: "synced",
      last_synced_at: new Date().toISOString(),
    });
  }

  /**
   * Update sync metadata
   */
  private updateSyncMetadata(): void {
    this.db.run(
      `UPDATE sync_metadata SET last_sync_at = datetime('now'), updated_at = datetime('now') WHERE id = 1`,
    );
  }

  /**
   * Get current sync status
   */
  getStatus(): {
    status: string;
    lastSync: string | null;
    pendingChanges: number;
  } {
    const metadata = this.db.query<any>(
      "SELECT * FROM sync_metadata WHERE id = 1",
    )[0];

    let pendingCount = 0;
    for (const table of SYNC_TABLES) {
      const pending = this.db.query<any>(
        `SELECT COUNT(*) as count FROM ${table} WHERE sync_status != 'synced'`,
      )[0];
      pendingCount += pending?.count || 0;
    }

    return {
      status: this.isSyncing
        ? "syncing"
        : pendingCount > 0
          ? "pending"
          : "synced",
      lastSync: metadata?.last_sync_at || null,
      pendingChanges: pendingCount,
    };
  }
}
