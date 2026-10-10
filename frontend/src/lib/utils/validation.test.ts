import { describe, it, expect } from 'vitest';
import { isValidUUID } from './validation';

describe('isValidUUID', () => {
  it('accepts valid UUIDs', () => {
    expect(isValidUUID('550e8400-e29b-41d4-a716-446655440000')).toBe(true);
    expect(isValidUUID('6ba7b810-9dad-11d1-80b4-00c04fd430c8')).toBe(true);
    expect(isValidUUID('A550E840-E29B-41D4-A716-446655440000')).toBe(true);
  });

  it('rejects invalid formats', () => {
    expect(isValidUUID('')).toBe(false);
    expect(isValidUUID('not-a-uuid')).toBe(false);
    expect(isValidUUID('550e8400-e29b-41d4-a716')).toBe(false);
    expect(isValidUUID('550e8400e29b41d4a716446655440000')).toBe(false);
    expect(isValidUUID('550e8400-e29b-41d4-a716-44665544000g')).toBe(false);
  });

  it('rejects UUIDs with extra characters', () => {
    expect(isValidUUID(' 550e8400-e29b-41d4-a716-446655440000')).toBe(false);
    expect(isValidUUID('550e8400-e29b-41d4-a716-446655440000 ')).toBe(false);
    expect(isValidUUID('{550e8400-e29b-41d4-a716-446655440000}')).toBe(false);
  });
});
