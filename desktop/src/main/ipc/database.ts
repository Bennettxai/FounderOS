import { ipcMain } from "electron";
import {
  DatabaseHelper,
  generateUUID,
  getDatabase,
  initDatabase,
} from "../database/sqlite";
import { SyncEngine, SYNC_TABLES, SyncTable } from "../sync/engine";

// Cloud sync target. Must be a *.businessos.dev host so the session cookie
// (Domain=.businessos.dev) attaches; app.businessos.dev proxies /api/* to the
// Cloud Run backend. Overridable for testing via BUSINESSOS_CLOUD_URL.
const CLOUD_RUN_URL =
  process.env.BUSINESSOS_CLOUD_URL || "https://app.businessos.dev";

let syncEngine: SyncEngine | null = null;
let dbHelper: DatabaseHelper | null = null;

/**
 * Initialize database and sync engine
 */
export function initializeDatabaseSystem(): void {
  console.log("Initializing database system...");

  // Initialize SQLite database
  initDatabase();
  dbHelper = new DatabaseHelper();

  // Initialize sync engine with Cloud Run URL
  syncEngine = new SyncEngine({
    serverUrl: CLOUD_RUN_URL,
    syncInterval: 5 * 60 * 1000, // 5 minutes
    debounceTime: 500,
    maxRetries: 3,
  });

  console.log(`Sync engine configured with server: ${CLOUD_RUN_URL}`);
}

/**
 * Start the sync engine
 */
export function startSync(): void {
  if (syncEngine) {
    syncEngine.start();
    console.log("Sync engine started");
  }
}

/**
 * Stop the sync engine
 */
export function stopSync(): void {
  if (syncEngine) {
    syncEngine.stop();
    console.log("Sync engine stopped");
  }
}

/**
 * Set up all database IPC handlers
 */
export function setupDatabaseHandlers(): void {
  if (!dbHelper) {
    dbHelper = new DatabaseHelper();
  }

  // ============================================
  // GENERIC CRUD OPERATIONS
  // ============================================

  // Get all records from a table
  ipcMain.handle(
    "db:getAll",
    async (_, table: string, where?: Record<string, any>) => {
      try {
        return { success: true, data: dbHelper!.getAll(table, where) };
      } catch (error) {
        console.error(`db:getAll error for ${table}:`, error);
        return { success: false, error: String(error) };
      }
    },
  );

  // Get a single record by ID
  ipcMain.handle("db:getById", async (_, table: string, id: string) => {
    try {
      const record = dbHelper!.getById(table, id);
      return { success: true, data: record || null };
    } catch (error) {
      console.error(`db:getById error for ${table}/${id}:`, error);
      return { success: false, error: String(error) };
    }
  });

  // Create a new record
  ipcMain.handle(
    "db:create",
    async (_, table: string, data: Record<string, any>) => {
      try {
        const id = data.id || generateUUID();
        const record = {
          id,
          ...data,
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
          sync_status: "pending_create",
          sync_version: 1,
        };

        dbHelper!.insert(table, record);

        // Schedule sync if table is syncable
        if (SYNC_TABLES.includes(table as SyncTable)) {
          syncEngine?.scheduleSync();
        }

        return { success: true, data: record };
      } catch (error) {
        console.error(`db:create error for ${table}:`, error);
        return { success: false, error: String(error) };
      }
    },
  );

  // Update a record
  ipcMain.handle(
    "db:update",
    async (_, table: string, id: string, data: Record<string, any>) => {
      try {
        const updates = {
          ...data,
          updated_at: new Date().toISOString(),
        };

        // Mark for sync if not already pending create
        const existing = dbHelper!.getById<any>(table, id);
        if (existing && existing.sync_status === "synced") {
          dbHelper!.markForSync(table, id, "pending_update");
        }

        dbHelper!.update(table, id, updates);

        // Schedule sync if table is syncable
        if (SYNC_TABLES.includes(table as SyncTable)) {
          syncEngine?.scheduleSync();
        }

        return { success: true, data: { id, ...updates } };
      } catch (error) {
        console.error(`db:update error for ${table}/${id}:`, error);
        return { success: false, error: String(error) };
      }
    },
  );

  // Delete a record
  ipcMain.handle("db:delete", async (_, table: string, id: string) => {
    try {
      // Mark for sync delete (actual delete happens after sync)
      const existing = dbHelper!.getById<any>(table, id);
      if (existing && existing.sync_status !== "pending_create") {
        // If already synced to server, mark for delete
        dbHelper!.markForSync(table, id, "pending_delete");
        syncEngine?.scheduleSync();
      } else {
        // If never synced, just delete locally
        dbHelper!.delete(table, id);
      }

      return { success: true };
    } catch (error) {
      console.error(`db:delete error for ${table}/${id}:`, error);
      return { success: false, error: String(error) };
    }
  });

  // Query with custom SQL (for complex queries)
  ipcMain.handle("db:query", async (_, sql: string, params?: any[]) => {
    try {
      const results = dbHelper!.query(sql, params);
      return { success: true, data: results };
    } catch (error) {
      console.error("db:query error:", error);
      return { success: false, error: String(error) };
    }
  });

  // ============================================
  // SYNC OPERATIONS
  // ============================================

  // Get sync status
  ipcMain.handle("sync:getStatus", async () => {
    if (!syncEngine) {
      return { status: "offline", lastSync: null, pendingChanges: 0 };
    }
    return syncEngine.getStatus();
  });

  // Set the active workspace for shared-record sync (BusinessOSSync).
  ipcMain.handle("sync:setWorkspace", async (_, workspaceId: string | null) => {
    syncEngine?.setWorkspaceId(workspaceId || undefined);
    return { success: true };
  });

  // Trigger manual sync
  ipcMain.handle("sync:trigger", async () => {
    if (!syncEngine) {
      return { success: false, error: "Sync engine not initialized" };
    }
    try {
      const result = await syncEngine.sync();
      return { success: result.success, data: result };
    } catch (error) {
      console.error("sync:trigger error:", error);
      return { success: false, error: String(error) };
    }
  });

  // Get pending sync items
  ipcMain.handle("sync:getPending", async () => {
    if (!dbHelper) {
      return { success: false, error: "Database not initialized" };
    }

    const pending: { table: string; count: number }[] = [];
    for (const table of SYNC_TABLES) {
      const items = dbHelper.getPendingSync(table);
      if (items.length > 0) {
        pending.push({ table, count: items.length });
      }
    }

    return { success: true, data: pending };
  });

  // ============================================
  // DOMAIN-SPECIFIC HELPERS
  // ============================================

  // Contexts
  ipcMain.handle(
    "db:contexts:getWithChildren",
    async (_, parentId?: string) => {
      try {
        const sql = parentId
          ? "SELECT * FROM contexts WHERE parent_id = ? AND is_archived = 0 ORDER BY name"
          : "SELECT * FROM contexts WHERE parent_id IS NULL AND is_archived = 0 ORDER BY name";
        const results = dbHelper!.query(sql, parentId ? [parentId] : []);
        return { success: true, data: results };
      } catch (error) {
        return { success: false, error: String(error) };
      }
    },
  );

  // Conversations with messages
  ipcMain.handle(
    "db:conversations:getWithMessages",
    async (_, conversationId: string) => {
      try {
        const conversation = dbHelper!.getById("conversations", conversationId);
        if (!conversation) {
          return { success: false, error: "Conversation not found" };
        }

        const messages = dbHelper!.query(
          "SELECT * FROM messages WHERE conversation_id = ? ORDER BY created_at ASC",
          [conversationId],
        );

        return { success: true, data: { ...conversation, messages } };
      } catch (error) {
        return { success: false, error: String(error) };
      }
    },
  );

  // Tasks by status
  ipcMain.handle("db:tasks:getByStatus", async (_, status?: string) => {
    try {
      const sql = status
        ? "SELECT * FROM tasks WHERE status = ? ORDER BY priority, due_date"
        : "SELECT * FROM tasks ORDER BY status, priority, due_date";
      const results = dbHelper!.query(sql, status ? [status] : []);
      return { success: true, data: results };
    } catch (error) {
      return { success: false, error: String(error) };
    }
  });

  // Projects with tasks
  ipcMain.handle("db:projects:getWithTasks", async (_, projectId: string) => {
    try {
      const project = dbHelper!.getById("projects", projectId);
      if (!project) {
        return { success: false, error: "Project not found" };
      }

      const tasks = dbHelper!.query(
        "SELECT * FROM tasks WHERE project_id = ? ORDER BY priority, due_date",
        [projectId],
      );

      return { success: true, data: { ...project, tasks } };
    } catch (error) {
      return { success: false, error: String(error) };
    }
  });

  // Calendar events by date range
  ipcMain.handle(
    "db:calendar:getByRange",
    async (_, startDate: string, endDate: string) => {
      try {
        const events = dbHelper!.query(
          `SELECT * FROM calendar_events
         WHERE start_time >= ? AND start_time <= ?
         ORDER BY start_time ASC`,
          [startDate, endDate],
        );
        return { success: true, data: events };
      } catch (error) {
        return { success: false, error: String(error) };
      }
    },
  );

  // Daily log for today
  ipcMain.handle("db:dailyLog:getToday", async (_, userId: string) => {
    try {
      const today = new Date().toISOString().split("T")[0];
      const logs = dbHelper!.query(
        "SELECT * FROM daily_logs WHERE user_id = ? AND date = ?",
        [userId, today],
      );
      return { success: true, data: logs[0] || null };
    } catch (error) {
      return { success: false, error: String(error) };
    }
  });

  // Clients with deals
  ipcMain.handle("db:clients:getWithDeals", async (_, clientId: string) => {
    try {
      const client = dbHelper!.getById("clients", clientId);
      if (!client) {
        return { success: false, error: "Client not found" };
      }

      const contacts = dbHelper!.query(
        "SELECT * FROM client_contacts WHERE client_id = ?",
        [clientId],
      );

      const deals = dbHelper!.query(
        "SELECT * FROM client_deals WHERE client_id = ? ORDER BY created_at DESC",
        [clientId],
      );

      const interactions = dbHelper!.query(
        "SELECT * FROM client_interactions WHERE client_id = ? ORDER BY occurred_at DESC LIMIT 10",
        [clientId],
      );

      return {
        success: true,
        data: { ...client, contacts, deals, interactions },
      };
    } catch (error) {
      return { success: false, error: String(error) };
    }
  });

  // User settings
  ipcMain.handle("db:settings:get", async (_, userId: string) => {
    try {
      const settings = dbHelper!.query(
        "SELECT * FROM user_settings WHERE user_id = ?",
        [userId],
      );
      return { success: true, data: settings[0] || null };
    } catch (error) {
      return { success: false, error: String(error) };
    }
  });

  ipcMain.handle(
    "db:settings:upsert",
    async (_, userId: string, settings: Record<string, any>) => {
      try {
        const existing = dbHelper!.query(
          "SELECT * FROM user_settings WHERE user_id = ?",
          [userId],
        );

        if (existing.length > 0) {
          dbHelper!.update("user_settings", (existing[0] as any).id, settings);
        } else {
          const id = generateUUID();
          dbHelper!.insert("user_settings", {
            id,
            user_id: userId,
            ...settings,
            sync_status: "pending_create",
            sync_version: 1,
          });
        }

        syncEngine?.scheduleSync();
        return { success: true };
      } catch (error) {
        return { success: false, error: String(error) };
      }
    },
  );

  console.log("Database IPC handlers registered");
}
