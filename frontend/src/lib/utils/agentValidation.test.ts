/**
 * Unit tests for agent validation utility
 */

import { describe, it, expect } from 'vitest';
import { validateAgentForm, VALIDATION_LIMITS, ALLOWED_CATEGORIES } from './agentValidation';
import type { CustomAgent } from '$lib/api/ai/types';

describe('validateAgentForm', () => {
  describe('Name validation', () => {
    it('should fail when name is empty', () => {
      const result = validateAgentForm({ name: '' } as Partial<CustomAgent>);
      expect(result.valid).toBe(false);
      expect(result.errors.some(e => e.field === 'name')).toBe(true);
    });

    it('should fail when name is too short', () => {
      const result = validateAgentForm({ name: 'a' } as Partial<CustomAgent>);
      expect(result.valid).toBe(false);
      expect(result.errors.some(e => e.field === 'name' && e.message.includes('at least'))).toBe(true);
    });

    it('should fail when name is too long', () => {
      const result = validateAgentForm({ name: 'a'.repeat(51) } as Partial<CustomAgent>);
      expect(result.valid).toBe(false);
      expect(result.errors.some(e => e.field === 'name' && e.message.includes('cannot exceed'))).toBe(true);
    });

    it('should fail when name contains uppercase letters', () => {
      const result = validateAgentForm({ name: 'MyAgent' } as Partial<CustomAgent>);
      expect(result.valid).toBe(false);
      expect(result.errors.some(e => e.field === 'name' && e.message.includes('lowercase'))).toBe(true);
    });

    it('should pass with valid name', () => {
      const result = validateAgentForm({
        name: 'my-agent-123',
        display_name: 'My Agent',
        system_prompt: 'This is a valid system prompt for testing purposes.'
      } as Partial<CustomAgent>);
      expect(result.errors.some(e => e.field === 'name')).toBe(false);
    });
  });

  describe('Display name validation', () => {
    it('should fail when display_name is empty', () => {
      const result = validateAgentForm({ display_name: '' } as Partial<CustomAgent>);
      expect(result.valid).toBe(false);
      expect(result.errors.some(e => e.field === 'display_name')).toBe(true);
    });

    it('should fail when display_name is too short', () => {
      const result = validateAgentForm({ display_name: 'A' } as Partial<CustomAgent>);
      expect(result.valid).toBe(false);
      expect(result.errors.some(e => e.field === 'display_name' && e.message.includes('at least'))).toBe(true);
    });

    it('should fail when display_name is too long', () => {
      const result = validateAgentForm({ display_name: 'A'.repeat(101) } as Partial<CustomAgent>);
      expect(result.valid).toBe(false);
      expect(result.errors.some(e => e.field === 'display_name' && e.message.includes('cannot exceed'))).toBe(true);
    });

    it('should pass with valid display_name', () => {
      const result = validateAgentForm({
        name: 'test-agent',
        display_name: 'Test Agent',
        system_prompt: 'This is a valid system prompt for testing purposes.'
      } as Partial<CustomAgent>);
      expect(result.errors.some(e => e.field === 'display_name')).toBe(false);
    });
  });

  describe('System prompt validation', () => {
    it('should fail when system_prompt is empty', () => {
      const result = validateAgentForm({ system_prompt: '' } as Partial<CustomAgent>);
      expect(result.valid).toBe(false);
      expect(result.errors.some(e => e.field === 'system_prompt')).toBe(true);
    });

    it('should fail when system_prompt is too short', () => {
      const result = validateAgentForm({ system_prompt: 'short' } as Partial<CustomAgent>);
      expect(result.valid).toBe(false);
      expect(result.errors.some(e => e.field === 'system_prompt' && e.message.includes('at least'))).toBe(true);
    });

    it('should fail when system_prompt is too long', () => {
      const result = validateAgentForm({ system_prompt: 'A'.repeat(5001) } as Partial<CustomAgent>);
      expect(result.valid).toBe(false);
      expect(result.errors.some(e => e.field === 'system_prompt' && e.message.includes('cannot exceed'))).toBe(true);
    });

    it('should pass with valid system_prompt', () => {
      const result = validateAgentForm({
        name: 'test-agent',
        display_name: 'Test Agent',
        system_prompt: 'This is a valid system prompt that meets the minimum length requirement.'
      } as Partial<CustomAgent>);
      expect(result.errors.some(e => e.field === 'system_prompt')).toBe(false);
    });
  });

  describe('Temperature validation', () => {
    it('should fail when temperature is below minimum', () => {
      const result = validateAgentForm({ temperature: -0.1 } as Partial<CustomAgent>);
      expect(result.valid).toBe(false);
      expect(result.errors.some(e => e.field === 'temperature')).toBe(true);
    });

    it('should fail when temperature is above maximum', () => {
      const result = validateAgentForm({ temperature: 2.1 } as Partial<CustomAgent>);
      expect(result.valid).toBe(false);
      expect(result.errors.some(e => e.field === 'temperature')).toBe(true);
    });

    it('should pass with valid temperature', () => {
      const result = validateAgentForm({
        name: 'test-agent',
        display_name: 'Test Agent',
        system_prompt: 'Valid system prompt for testing.',
        temperature: 1.0
      } as Partial<CustomAgent>);
      expect(result.errors.some(e => e.field === 'temperature')).toBe(false);
    });
  });

  describe('Suggested prompts validation', () => {
    it('should fail when too many prompts', () => {
      const result = validateAgentForm({
        suggested_prompts: Array(11).fill('prompt')
      } as Partial<CustomAgent>);
      expect(result.valid).toBe(false);
      expect(result.errors.some(e => e.field === 'suggested_prompts')).toBe(true);
    });

    it('should fail when prompt is empty', () => {
      const result = validateAgentForm({
        suggested_prompts: ['valid prompt', '   ', 'another valid']
      } as Partial<CustomAgent>);
      expect(result.valid).toBe(false);
      expect(result.errors.some(e => e.field.startsWith('suggested_prompt_'))).toBe(true);
    });

    it('should fail when prompt is too long', () => {
      const result = validateAgentForm({
        suggested_prompts: ['a'.repeat(501)]
      } as Partial<CustomAgent>);
      expect(result.valid).toBe(false);
      expect(result.errors.some(e => e.field.startsWith('suggested_prompt_'))).toBe(true);
    });

    it('should pass with valid prompts', () => {
      const result = validateAgentForm({
        name: 'test-agent',
        display_name: 'Test Agent',
        system_prompt: 'Valid system prompt for testing.',
        suggested_prompts: ['Help me with X', 'Explain Y', 'Show me Z']
      } as Partial<CustomAgent>);
      expect(result.errors.some(e => e.field.startsWith('suggested_prompt_'))).toBe(false);
    });
  });

  describe('Category validation', () => {
    it('should fail with invalid category', () => {
      const result = validateAgentForm({
        category: 'invalid-category'
      } as Partial<CustomAgent>);
      expect(result.valid).toBe(false);
      expect(result.errors.some(e => e.field === 'category')).toBe(true);
    });

    it('should pass with valid category', () => {
      ALLOWED_CATEGORIES.forEach(category => {
        const result = validateAgentForm({
          name: 'test-agent',
          display_name: 'Test Agent',
          system_prompt: 'Valid system prompt for testing.',
          category
        } as Partial<CustomAgent>);
        expect(result.errors.some(e => e.field === 'category')).toBe(false);
      });
    });
  });

  describe('Complete agent validation', () => {
    it('should pass with all valid fields', () => {
      const validAgent: Partial<CustomAgent> = {
        name: 'test-agent',
        display_name: 'Test Agent',
        description: 'This is a test agent',
        system_prompt: 'You are a helpful assistant that provides accurate information.',
        category: 'general',
        temperature: 0.7,
        max_tokens: 2000,
        welcome_message: 'Hello! How can I help you today?',
        suggested_prompts: ['Help me with task', 'Explain concept'],
        avatar: 'https://example.com/avatar.png'
      };

      const result = validateAgentForm(validAgent);
      expect(result.valid).toBe(true);
      expect(result.errors).toHaveLength(0);
    });

    it('should collect multiple errors', () => {
      const invalidAgent: Partial<CustomAgent> = {
        name: '',
        display_name: 'A',
        system_prompt: 'short',
        temperature: 3.0,
        category: 'invalid'
      };

      const result = validateAgentForm(invalidAgent);
      expect(result.valid).toBe(false);
      expect(result.errors.length).toBeGreaterThan(3);
    });
  });
});
