import { getDatabase, generateUUID } from '../database/sqlite';

/**
 * Sync operation types
 */
export type SyncOperation = 'create' | 'update' | 'delete';

/**
 * Sync queue item
 */
export interface QueueItem {
  id: number;
  table_name: string;
  record_id: string;
  operation: SyncOperation;
  payload: any;
  attempts: number;
  last_error: string | null;
  created_at: string;
  processed_at: string | null;
}

/**
 * Manages the sync queue for offline operations
 */
export class SyncQueue {
  private db = getDatabase();

  /**
   * Add an operation to the sync queue
   */
  enqueue(tableName: string, recordId: string, operation: SyncOperation, payload?: any): void {
    // Check if there's already a pending operation for this record
    const existing = this.db
      .prepare(
        `SELECT * FROM sync_queue
         WHERE table_name = ? AND record_id = ? AND processed_at IS NULL`
      )
      .get(tableName, recordId) as QueueItem | undefined;

    if (existing) {
      // Merge operations
      const mergedOperation = this.mergeOperations(existing.operation as SyncOperation, operation);

      if (mergedOperation === null) {
        // Operations cancel out (e.g., create + delete)
        this.db
          .prepare('DELETE FROM sync_queue WHERE id = ?')
          .run(existing.id);
      } else {
        // Update existing queue item
        this.db
          .prepare(
            `UPDATE sync_queue
             SET operation = ?, payload = ?, attempts = 0, last_error = NULL
             WHERE id = ?`
          )
          .run(mergedOperation, JSON.stringify(payload), existing.id);
      }
    } else {
      // Add new queue item
      this.db
        .prepare(
          `INSERT INTO sync_queue (table_name, record_id, operation, payload)
           VALUES (?, ?, ?, ?)`
        )
        .run(tableName, recordId, operation, JSON.stringify(payload));
    }
  }

  /**
   * Merge two operations on the same record
   */
  private mergeOperations(
    existing: SyncOperation,
    incoming: SyncOperation
  ): SyncOperation | null {
    // Create + Delete = null (cancel out)
    if (existing === 'create' && incoming === 'delete') {
      return null;
    }

    // Create + Update = Create (with new payload)
    if (existing === 'create' && incoming === 'update') {
      return 'create';
    }

    // Update + Delete = Delete
    if (existing === 'update' && incoming === 'delete') {
      return 'delete';
    }

    // Update + Update = Update (with new payload)
    if (existing === 'update' && incoming === 'update') {
      return 'update';
    }

    // For all other cases, use the incoming operation
    return incoming;
  }

  /**
   * Get all pending items in the queue
   */
  getPending(): QueueItem[] {
    return this.db
      .prepare(
        `SELECT * FROM sync_queue
         WHERE processed_at IS NULL
         ORDER BY created_at ASC`
      )
      .all() as QueueItem[];
  }

  /**
   * Get pending items for a specific table
   */
  getPendingForTable(tableName: string): QueueItem[] {
    return this.db
      .prepare(
        `SELECT * FROM sync_queue
         WHERE table_name = ? AND processed_at IS NULL
         ORDER BY created_at ASC`
      )
      .all(tableName) as QueueItem[];
  }

  /**
   * Mark an item as processed
   */
  markProcessed(id: number): void {
    this.db
      .prepare(
        `UPDATE sync_queue
         SET processed_at = datetime('now')
         WHERE id = ?`
      )
      .run(id);
  }

  /**
   * Mark an item as failed
   */
  markFailed(id: number, error: string): void {
    this.db
      .prepare(
        `UPDATE sync_queue
         SET attempts = attempts + 1, last_error = ?
         WHERE id = ?`
      )
      .run(error, id);
  }

  /**
   * Get items that have failed too many times
   */
  getFailedItems(maxAttempts: number = 3): QueueItem[] {
    return this.db
      .prepare(
        `SELECT * FROM sync_queue
         WHERE processed_at IS NULL AND attempts >= ?
         ORDER BY created_at ASC`
      )
      .all(maxAttempts) as QueueItem[];
  }

  /**
   * Remove failed items from the queue
   */
  clearFailed(maxAttempts: number = 3): number {
    const result = this.db
      .prepare(
        `DELETE FROM sync_queue
         WHERE processed_at IS NULL AND attempts >= ?`
      )
      .run(maxAttempts);

    return result.changes;
  }

  /**
   * Clear all processed items
   */
  clearProcessed(): number {
    const result = this.db
      .prepare('DELETE FROM sync_queue WHERE processed_at IS NOT NULL')
      .run();

    return result.changes;
  }

  /**
   * Get queue statistics
   */
  getStats(): {
    pending: number;
    failed: number;
    processed: number;
    byTable: Record<string, number>;
  } {
    const pending = this.db
      .prepare(
        'SELECT COUNT(*) as count FROM sync_queue WHERE processed_at IS NULL AND attempts < 3'
      )
      .get() as { count: number };

    const failed = this.db
      .prepare(
        'SELECT COUNT(*) as count FROM sync_queue WHERE processed_at IS NULL AND attempts >= 3'
      )
      .get() as { count: number };

    const processed = this.db
      .prepare('SELECT COUNT(*) as count FROM sync_queue WHERE processed_at IS NOT NULL')
      .get() as { count: number };

    const byTable = this.db
      .prepare(
        `SELECT table_name, COUNT(*) as count
         FROM sync_queue
         WHERE processed_at IS NULL
         GROUP BY table_name`
      )
      .all() as { table_name: string; count: number }[];

    return {
      pending: pending.count,
      failed: failed.count,
      processed: processed.count,
      byTable: byTable.reduce(
        (acc, row) => {
          acc[row.table_name] = row.count;
          return acc;
        },
        {} as Record<string, number>
      ),
    };
  }
}
