import { SyncTable } from './engine';

/**
 * Conflict resolution strategies
 */
export type ConflictResolution = 'local' | 'server' | 'merge' | 'conflict';

/**
 * Conflict resolution strategy per table
 */
export type ResolutionStrategy = 'last_write_wins' | 'server_wins' | 'manual' | 'merge';

/**
 * Configuration for conflict resolution per table
 */
const RESOLUTION_STRATEGIES: Record<SyncTable, ResolutionStrategy> = {
  // Last write wins (based on updated_at timestamp)
  tasks: 'last_write_wins',
  focus_items: 'last_write_wins',
  daily_logs: 'last_write_wins',
  user_settings: 'last_write_wins',
  team_members: 'last_write_wins',
  team_member_activities: 'last_write_wins',
  node_metrics: 'last_write_wins',
  client_contacts: 'last_write_wins',
  client_interactions: 'last_write_wins',
  client_deals: 'last_write_wins',
  calendar_events: 'last_write_wins',

  // Server wins (for data integrity - conversation history should not diverge)
  conversations: 'server_wins',
  messages: 'server_wins',

  // Merge non-conflicting fields
  projects: 'merge',
  nodes: 'merge',
  clients: 'merge',

  // Manual review required
  contexts: 'manual',
  project_notes: 'manual',
  artifacts: 'manual',
  artifact_versions: 'server_wins',
};

/**
 * Conflict resolver for handling sync conflicts
 */
export class ConflictResolver {
  /**
   * Resolve a conflict between local and server records
   */
  resolve(
    table: SyncTable,
    localRecord: any,
    serverRecord: any
  ): ConflictResolution {
    const strategy = RESOLUTION_STRATEGIES[table];

    switch (strategy) {
      case 'last_write_wins':
        return this.resolveLastWriteWins(localRecord, serverRecord);
      case 'server_wins':
        return 'server';
      case 'merge':
        return this.resolveMerge(localRecord, serverRecord);
      case 'manual':
      default:
        return 'conflict';
    }
  }

  /**
   * Last write wins resolution
   */
  private resolveLastWriteWins(localRecord: any, serverRecord: any): ConflictResolution {
    const localUpdated = new Date(localRecord.updated_at || localRecord.created_at);
    const serverUpdated = new Date(serverRecord.updated_at || serverRecord.created_at);

    if (localUpdated > serverUpdated) {
      return 'local';
    } else {
      return 'server';
    }
  }

  /**
   * Merge resolution - attempt to merge non-conflicting fields
   */
  private resolveMerge(localRecord: any, serverRecord: any): ConflictResolution {
    // Check if there are actually conflicting changes
    const conflicts = this.findConflicts(localRecord, serverRecord);

    if (conflicts.length === 0) {
      return 'merge';
    }

    // If there are conflicts, check if they're the same value
    const realConflicts = conflicts.filter(
      (field) => localRecord[field] !== serverRecord[field]
    );

    if (realConflicts.length === 0) {
      return 'merge';
    }

    // There are real conflicts - need manual resolution
    return 'conflict';
  }

  /**
   * Find conflicting fields between local and server records
   */
  private findConflicts(localRecord: any, serverRecord: any): string[] {
    const localChanged = this.getChangedFields(localRecord);
    const serverChanged = this.getChangedFields(serverRecord);

    // Find fields that changed in both
    return localChanged.filter((field) => serverChanged.includes(field));
  }

  /**
   * Get fields that have changed from original
   * This is a simplified version - in production, you'd track original values
   */
  private getChangedFields(record: any): string[] {
    // For now, assume all non-meta fields are potentially changed
    const metaFields = [
      'id',
      'server_id',
      'created_at',
      'updated_at',
      'sync_status',
      'sync_version',
      'last_synced_at',
    ];

    return Object.keys(record).filter((key) => !metaFields.includes(key));
  }

  /**
   * Merge two records, preferring server for metadata and newer values for content
   */
  mergeRecords(localRecord: any, serverRecord: any): any {
    const merged: any = {};

    // Meta fields from server
    const metaFields = ['id', 'created_at'];
    for (const field of metaFields) {
      merged[field] = serverRecord[field];
    }

    // Content fields - prefer newer
    const allFields = new Set([
      ...Object.keys(localRecord),
      ...Object.keys(serverRecord),
    ]);

    const excludeFields = new Set([
      ...metaFields,
      'server_id',
      'sync_status',
      'sync_version',
      'last_synced_at',
    ]);

    for (const field of allFields) {
      if (excludeFields.has(field)) continue;

      if (localRecord[field] !== undefined && serverRecord[field] !== undefined) {
        // Both have the field - use newer based on updated_at
        const localTime = new Date(localRecord.updated_at || 0);
        const serverTime = new Date(serverRecord.updated_at || 0);

        merged[field] = localTime > serverTime ? localRecord[field] : serverRecord[field];
      } else if (localRecord[field] !== undefined) {
        merged[field] = localRecord[field];
      } else {
        merged[field] = serverRecord[field];
      }
    }

    // Updated timestamp is now
    merged.updated_at = new Date().toISOString();

    return merged;
  }
}

/**
 * Conflict record for manual resolution
 */
export interface ConflictRecord {
  table: string;
  localId: string;
  serverId: string;
  localRecord: any;
  serverRecord: any;
  conflictingFields: string[];
  createdAt: string;
}

/**
 * Manage conflicts that need manual resolution
 */
export class ConflictManager {
  /**
   * Get all unresolved conflicts
   */
  getConflicts(): ConflictRecord[] {
    // This would query the database for records with sync_status = 'conflict'
    // For now, return empty array
    return [];
  }

  /**
   * Resolve a conflict by choosing a version
   */
  resolveConflict(
    table: string,
    localId: string,
    resolution: 'keep_local' | 'keep_server' | 'keep_both'
  ): void {
    // Implementation would update the record based on resolution
    console.log(`Resolving conflict for ${table}/${localId}: ${resolution}`);
  }
}
