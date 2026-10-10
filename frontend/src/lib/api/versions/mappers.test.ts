import { describe, it, expect } from 'vitest';
import { extractDisplayNumber, mapBackendToVersion, mapBackendVersionsList } from './mappers';
import type { BackendVersionInfo } from './types';

describe('extractDisplayNumber', () => {
  it('extracts patch number from "0.0.X" format', () => {
    expect(extractDisplayNumber('0.0.1')).toBe(1);
    expect(extractDisplayNumber('0.0.5')).toBe(5);
    expect(extractDisplayNumber('0.0.99')).toBe(99);
  });

  it('handles non-standard version strings', () => {
    expect(extractDisplayNumber('1.2.3')).toBe(3);
    expect(extractDisplayNumber('10')).toBe(10);
  });

  it('returns 1 for invalid strings', () => {
    expect(extractDisplayNumber('invalid')).toBe(1);
    expect(extractDisplayNumber('')).toBe(1);
    expect(extractDisplayNumber('0.0.abc')).toBe(1);
  });
});

describe('mapBackendToVersion', () => {
  const baseInfo: BackendVersionInfo = {
    id: 'v-123',
    app_id: 'app-456',
    version_number: '0.0.3',
    change_summary: 'Manual snapshot',
    created_by: 'user-1',
    created_at: '2026-02-08T12:00:00Z',
    snapshot_data: { files: [] },
    snapshot_metadata: {},
  };

  it('maps all fields correctly', () => {
    const result = mapBackendToVersion(baseInfo, true);

    expect(result.id).toBe('v-123');
    expect(result.appId).toBe('app-456');
    expect(result.versionNumber).toBe(3);
    expect(result.backendVersion).toBe('0.0.3');
    expect(result.label).toBe('Manual snapshot');
    expect(result.createdBy).toBe('user-1');
    expect(result.createdAt).toEqual(new Date('2026-02-08T12:00:00Z'));
    expect(result.isCurrent).toBe(true);
    expect(result.configSnapshot).toEqual({ files: [] });
  });

  it('infers trigger from change_summary keywords', () => {
    expect(
      mapBackendToVersion({ ...baseInfo, change_summary: 'restore from v2' }, false).trigger
    ).toBe('restore');

    expect(
      mapBackendToVersion({ ...baseInfo, change_summary: 'auto snapshot' }, false).trigger
    ).toBe('auto_snapshot');

    expect(
      mapBackendToVersion({ ...baseInfo, change_summary: 'AI generation complete' }, false).trigger
    ).toBe('ai_generation');

    expect(
      mapBackendToVersion({ ...baseInfo, change_summary: 'saved changes', created_by: 'user-1' }, false).trigger
    ).toBe('manual_snapshot');
  });

  it('defaults trigger to ai_generation when no keywords match', () => {
    expect(
      mapBackendToVersion({ ...baseInfo, change_summary: null, created_by: null }, false).trigger
    ).toBe('ai_generation');
  });

  it('handles null/undefined optional fields', () => {
    const minimal: BackendVersionInfo = {
      id: 'v-1',
      version_number: '0.0.1',
      created_at: '2026-01-01T00:00:00Z',
    };
    const result = mapBackendToVersion(minimal, false);

    expect(result.appId).toBe('');
    expect(result.label).toBeUndefined();
    expect(result.createdBy).toBeUndefined();
    expect(result.configSnapshot).toEqual({});
  });
});

describe('mapBackendVersionsList', () => {
  const versions: BackendVersionInfo[] = [
    { id: 'v-3', version_number: '0.0.3', created_at: '2026-02-08T12:00:00Z' },
    { id: 'v-2', version_number: '0.0.2', created_at: '2026-02-07T12:00:00Z' },
    { id: 'v-1', version_number: '0.0.1', created_at: '2026-02-06T12:00:00Z' },
  ];

  it('marks only the first item as current', () => {
    const result = mapBackendVersionsList(versions);
    expect(result[0].isCurrent).toBe(true);
    expect(result[1].isCurrent).toBe(false);
    expect(result[2].isCurrent).toBe(false);
  });

  it('preserves order', () => {
    const result = mapBackendVersionsList(versions);
    expect(result.map((v) => v.id)).toEqual(['v-3', 'v-2', 'v-1']);
  });

  it('returns empty array for empty input', () => {
    expect(mapBackendVersionsList([])).toEqual([]);
  });
});
