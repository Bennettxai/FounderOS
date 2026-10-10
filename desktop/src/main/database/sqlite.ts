import Database from "better-sqlite3";
import path from "path";
import fs from "fs";
import { app } from "electron";

let db: Database.Database | null = null;

/**
 * Get the database file path
 */
export function getDatabasePath(): string {
  const userDataPath = app.getPath("userData");
  return path.join(userDataPath, "businessos.db");
}

/**
 * Initialize the SQLite database connection
 */
export function initDatabase(): Database.Database {
  if (db) {
    return db;
  }

  const dbPath = getDatabasePath();
  console.log(`Initializing SQLite database at: ${dbPath}`);

  // Ensure the directory exists
  const dbDir = path.dirname(dbPath);
  if (!fs.existsSync(dbDir)) {
    fs.mkdirSync(dbDir, { recursive: true });
  }

  // Create or open the database
  db = new Database(dbPath);

  // Enable foreign keys
  db.pragma("foreign_keys = ON");

  // Enable WAL mode for better performance
  db.pragma("journal_mode = WAL");

  // Run migrations
  runMigrations(db);

  console.log("SQLite database initialized");
  return db;
}

/**
 * Get the database instance
 */
export function getDatabase(): Database.Database {
  if (!db) {
    return initDatabase();
  }
  return db;
}

/**
 * Close the database connection
 */
export function closeDatabase(): void {
  if (db) {
    db.close();
    db = null;
    console.log("SQLite database closed");
  }
}

/**
 * Run database migrations
 */
function runMigrations(database: Database.Database): void {
  // Create migrations table if it doesn't exist
  database.exec(`
    CREATE TABLE IF NOT EXISTS migrations (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      name TEXT UNIQUE NOT NULL,
      applied_at TEXT DEFAULT (datetime('now'))
    )
  `);

  // Get list of applied migrations
  const appliedMigrations = database
    .prepare("SELECT name FROM migrations")
    .all()
    .map((row: any) => row.name);

  // Get migration files
  const migrationsDir = path.join(__dirname, "migrations");

  // In development, the migrations might be in a different location
  let migrationFiles: string[] = [];

  if (fs.existsSync(migrationsDir)) {
    migrationFiles = fs
      .readdirSync(migrationsDir)
      .filter((f) => f.endsWith(".sql"))
      .sort();
  } else {
    // Try to find migrations in the source directory during development
    const devMigrationsDir = path.join(
      app.getAppPath(),
      "src/main/database/migrations",
    );
    if (fs.existsSync(devMigrationsDir)) {
      migrationFiles = fs
        .readdirSync(devMigrationsDir)
        .filter((f) => f.endsWith(".sql"))
        .sort();
    }
  }

  // Apply pending migrations
  for (const file of migrationFiles) {
    if (!appliedMigrations.includes(file)) {
      console.log(`Applying migration: ${file}`);

      const migrationPath = fs.existsSync(migrationsDir)
        ? path.join(migrationsDir, file)
        : path.join(app.getAppPath(), "src/main/database/migrations", file);

      const sql = fs.readFileSync(migrationPath, "utf-8");

      // Run migration in a transaction. Statements run one at a time so an
      // idempotent "ADD COLUMN" that races an already-present column (SQLite has
      // no ADD COLUMN IF NOT EXISTS) is tolerated rather than aborting the whole
      // migration. Any other error still rolls the transaction back.
      const transaction = database.transaction(() => {
        execTolerantOfDuplicateColumn(database, sql);
        database.prepare("INSERT INTO migrations (name) VALUES (?)").run(file);
      });

      transaction();
      console.log(`Migration applied: ${file}`);
    }
  }
}

/**
 * Execute a migration's SQL statement-by-statement, tolerating the single
 * idempotent case SQLite can't express inline: re-adding a column that already
 * exists ("duplicate column name"). Every other error propagates so the
 * surrounding transaction rolls the whole migration back.
 */
function execTolerantOfDuplicateColumn(
  database: Database.Database,
  sql: string,
): void {
  // Naive ";" split is safe here: our migrations are plain DDL with no
  // semicolons inside literals or triggers. Strip line comments first so a
  // leading migration header does not get executed as part of a statement.
  const statements = sql
    .split("\n")
    .filter((line) => !line.trimStart().startsWith("--"))
    .join("\n")
    .split(";")
    .map((s) => s.trim())
    .filter((s) => s.length > 0);

  for (const statement of statements) {
    try {
      database.exec(statement);
    } catch (error) {
      if (/duplicate column name/i.test(String(error))) {
        console.log(`Skipping already-applied column add: ${error}`);
        continue;
      }
      throw error;
    }
  }
}

/**
 * Generate a UUID (for local record IDs)
 */
export function generateUUID(): string {
  return "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx".replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0;
    const v = c === "x" ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
}

/**
 * Database helper for common operations
 */
export class DatabaseHelper {
  private db: Database.Database;

  constructor() {
    this.db = getDatabase();
  }

  /**
   * Get all records from a table with optional filters
   */
  getAll<T>(table: string, where?: Record<string, any>): T[] {
    let sql = `SELECT * FROM ${table}`;
    const params: any[] = [];

    if (where && Object.keys(where).length > 0) {
      const conditions = Object.keys(where).map((key) => {
        params.push(where[key]);
        return `${key} = ?`;
      });
      sql += ` WHERE ${conditions.join(" AND ")}`;
    }

    return this.db.prepare(sql).all(...params) as T[];
  }

  /**
   * Get a single record by ID
   */
  getById<T>(table: string, id: string): T | undefined {
    return this.db.prepare(`SELECT * FROM ${table} WHERE id = ?`).get(id) as T;
  }

  /**
   * Insert a record
   */
  insert(table: string, data: Record<string, any>): void {
    const keys = Object.keys(data);
    const placeholders = keys.map(() => "?").join(", ");
    const values = keys.map((k) => data[k]);

    this.db
      .prepare(
        `INSERT INTO ${table} (${keys.join(", ")}) VALUES (${placeholders})`,
      )
      .run(...values);
  }

  /**
   * Update a record
   */
  update(table: string, id: string, data: Record<string, any>): void {
    const keys = Object.keys(data).filter((k) => k !== "id");
    const setClause = keys.map((k) => `${k} = ?`).join(", ");
    const values = keys.map((k) => data[k]);

    this.db
      .prepare(
        `UPDATE ${table} SET ${setClause}, updated_at = datetime('now') WHERE id = ?`,
      )
      .run(...values, id);
  }

  /**
   * Delete a record
   */
  delete(table: string, id: string): void {
    this.db.prepare(`DELETE FROM ${table} WHERE id = ?`).run(id);
  }

  /**
   * Mark a record for sync
   */
  markForSync(
    table: string,
    id: string,
    status: "pending_create" | "pending_update" | "pending_delete",
  ): void {
    this.db
      .prepare(
        `UPDATE ${table} SET sync_status = ?, sync_version = sync_version + 1 WHERE id = ?`,
      )
      .run(status, id);
  }

  /**
   * Get records pending sync
   */
  getPendingSync(table: string): any[] {
    return this.db
      .prepare(`SELECT * FROM ${table} WHERE sync_status != 'synced'`)
      .all();
  }

  /**
   * Mark a record as synced
   */
  markSynced(table: string, id: string, serverId?: string): void {
    if (serverId) {
      this.db
        .prepare(
          `UPDATE ${table} SET sync_status = 'synced', server_id = ?, last_synced_at = datetime('now') WHERE id = ?`,
        )
        .run(serverId, id);
    } else {
      this.db
        .prepare(
          `UPDATE ${table} SET sync_status = 'synced', last_synced_at = datetime('now') WHERE id = ?`,
        )
        .run(id);
    }
  }

  /**
   * Run a raw query
   */
  query<T>(sql: string, params?: any[]): T[] {
    return this.db.prepare(sql).all(...(params || [])) as T[];
  }

  /**
   * Run a raw statement
   */
  run(sql: string, params?: any[]): Database.RunResult {
    return this.db.prepare(sql).run(...(params || []));
  }

  /**
   * Begin a transaction
   */
  transaction<T>(fn: () => T): T {
    return this.db.transaction(fn)();
  }
}
