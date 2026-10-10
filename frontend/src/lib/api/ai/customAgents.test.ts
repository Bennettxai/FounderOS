import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import {
  getCustomAgents,
  getCustomAgent,
  createCustomAgent,
  updateCustomAgent,
  deleteCustomAgent,
  getAgentsByCategory,
  getAgentPresets,
  getAgentPreset,
  createFromPreset,
  testAgent,
  testSandbox
} from './ai';
import type { CustomAgent, AgentPreset, SandboxTestRequest } from './types';

// Mock the base request module
vi.mock('../base', () => ({
  request: vi.fn(),
  getApiBaseUrl: vi.fn(() => 'http://localhost:8080')
}));

// Import mocked functions
import { request, getApiBaseUrl } from '../base';

describe('Custom Agents API Client', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('getCustomAgents', () => {
    it('should fetch all custom agents without inactive agents by default', async () => {
      const mockResponse = {
        agents: [
          {
            id: '1',
            user_id: 'user1',
            name: 'agent1',
            display_name: 'Agent 1',
            system_prompt: 'You are agent 1',
            is_active: true,
            created_at: '2024-01-01',
            updated_at: '2024-01-01'
          }
        ]
      };

      vi.mocked(request).mockResolvedValue(mockResponse);

      const result = await getCustomAgents();

      expect(request).toHaveBeenCalledWith('/ai/custom-agents');
      expect(result).toEqual(mockResponse);
    });

    it('should fetch all custom agents including inactive when specified', async () => {
      const mockResponse = {
        agents: [
          {
            id: '1',
            user_id: 'user1',
            name: 'agent1',
            display_name: 'Agent 1',
            system_prompt: 'You are agent 1',
            is_active: false,
            created_at: '2024-01-01',
            updated_at: '2024-01-01'
          }
        ]
      };

      vi.mocked(request).mockResolvedValue(mockResponse);

      const result = await getCustomAgents(true);

      expect(request).toHaveBeenCalledWith('/ai/custom-agents?include_inactive=true');
      expect(result).toEqual(mockResponse);
    });

    it('should handle empty agent list', async () => {
      const mockResponse = { agents: [] };
      vi.mocked(request).mockResolvedValue(mockResponse);

      const result = await getCustomAgents();

      expect(result.agents).toHaveLength(0);
    });

    it('should handle API errors', async () => {
      const mockError = new Error('Network error');
      vi.mocked(request).mockRejectedValue(mockError);

      await expect(getCustomAgents()).rejects.toThrow('Network error');
    });
  });

  describe('getCustomAgent', () => {
    it('should fetch a specific agent by ID', async () => {
      const mockAgent: CustomAgent = {
        id: '1',
        user_id: 'user1',
        name: 'test-agent',
        display_name: 'Test Agent',
        description: 'A test agent',
        system_prompt: 'You are a test agent',
        is_active: true,
        created_at: '2024-01-01',
        updated_at: '2024-01-01'
      };

      vi.mocked(request).mockResolvedValue({ agent: mockAgent });

      const result = await getCustomAgent('1');

      expect(request).toHaveBeenCalledWith('/ai/custom-agents/1');
      expect(result).toEqual(mockAgent);
    });

    it('should handle non-existent agent', async () => {
      const mockError = new Error('Agent not found');
      vi.mocked(request).mockRejectedValue(mockError);

      await expect(getCustomAgent('999')).rejects.toThrow('Agent not found');
    });
  });

  describe('createCustomAgent', () => {
    it('should create a new custom agent', async () => {
      const newAgentData: Partial<CustomAgent> = {
        name: 'new-agent',
        display_name: 'New Agent',
        description: 'A new agent',
        system_prompt: 'You are a new agent',
        category: 'general'
      };

      const mockCreatedAgent: CustomAgent = {
        id: '1',
        user_id: 'user1',
        ...newAgentData,
        is_active: true,
        created_at: '2024-01-01',
        updated_at: '2024-01-01'
      } as CustomAgent;

      vi.mocked(request).mockResolvedValue({ agent: mockCreatedAgent });

      const result = await createCustomAgent(newAgentData);

      expect(request).toHaveBeenCalledWith('/ai/custom-agents', {
        method: 'POST',
        body: newAgentData
      });
      expect(result).toEqual(mockCreatedAgent);
    });

    it('should create agent with optional fields', async () => {
      const newAgentData: Partial<CustomAgent> = {
        name: 'advanced-agent',
        display_name: 'Advanced Agent',
        system_prompt: 'You are advanced',
        model_preference: 'gpt-4',
        temperature: 0.7,
        max_tokens: 1000,
        capabilities: ['code', 'analysis'],
        tools_enabled: ['calculator'],
        thinking_enabled: true,
        streaming_enabled: true
      };

      const mockCreatedAgent: CustomAgent = {
        id: '2',
        user_id: 'user1',
        ...newAgentData,
        is_active: true,
        created_at: '2024-01-01',
        updated_at: '2024-01-01'
      } as CustomAgent;

      vi.mocked(request).mockResolvedValue({ agent: mockCreatedAgent });

      const result = await createCustomAgent(newAgentData);

      expect(result.model_preference).toBe('gpt-4');
      expect(result.capabilities).toEqual(['code', 'analysis']);
      expect(result.thinking_enabled).toBe(true);
    });

    it('should handle validation errors', async () => {
      const invalidData = { name: '' };
      const mockError = new Error('Validation failed: name is required');
      vi.mocked(request).mockRejectedValue(mockError);

      await expect(createCustomAgent(invalidData)).rejects.toThrow('Validation failed');
    });
  });

  describe('updateCustomAgent', () => {
    it('should update an existing agent', async () => {
      const updateData: Partial<CustomAgent> = {
        display_name: 'Updated Agent',
        description: 'Updated description'
      };

      const mockUpdatedAgent: CustomAgent = {
        id: '1',
        user_id: 'user1',
        name: 'test-agent',
        display_name: 'Updated Agent',
        description: 'Updated description',
        system_prompt: 'Original prompt',
        is_active: true,
        created_at: '2024-01-01',
        updated_at: '2024-01-02'
      };

      vi.mocked(request).mockResolvedValue({ agent: mockUpdatedAgent });

      const result = await updateCustomAgent('1', updateData);

      expect(request).toHaveBeenCalledWith('/ai/custom-agents/1', {
        method: 'PUT',
        body: updateData
      });
      expect(result.display_name).toBe('Updated Agent');
      expect(result.updated_at).toBe('2024-01-02');
    });

    it('should handle partial updates', async () => {
      const updateData = { is_active: false };

      const mockUpdatedAgent: CustomAgent = {
        id: '1',
        user_id: 'user1',
        name: 'test-agent',
        display_name: 'Test Agent',
        system_prompt: 'Test prompt',
        is_active: false,
        created_at: '2024-01-01',
        updated_at: '2024-01-02'
      };

      vi.mocked(request).mockResolvedValue({ agent: mockUpdatedAgent });

      const result = await updateCustomAgent('1', updateData);

      expect(result.is_active).toBe(false);
    });

    it('should handle non-existent agent update', async () => {
      const mockError = new Error('Agent not found');
      vi.mocked(request).mockRejectedValue(mockError);

      await expect(updateCustomAgent('999', { display_name: 'Test' })).rejects.toThrow(
        'Agent not found'
      );
    });
  });

  describe('deleteCustomAgent', () => {
    it('should delete an agent successfully', async () => {
      const mockResponse = { message: 'Agent deleted successfully' };
      vi.mocked(request).mockResolvedValue(mockResponse);

      const result = await deleteCustomAgent('1');

      expect(request).toHaveBeenCalledWith('/ai/custom-agents/1', {
        method: 'DELETE'
      });
      expect(result.message).toBe('Agent deleted successfully');
    });

    it('should handle deletion of non-existent agent', async () => {
      const mockError = new Error('Agent not found');
      vi.mocked(request).mockRejectedValue(mockError);

      await expect(deleteCustomAgent('999')).rejects.toThrow('Agent not found');
    });
  });

  describe('getAgentsByCategory', () => {
    it('should fetch agents by category', async () => {
      const mockResponse = {
        agents: [
          {
            id: '1',
            user_id: 'user1',
            name: 'specialist-agent',
            display_name: 'Specialist Agent',
            system_prompt: 'You are a specialist',
            category: 'specialist',
            is_active: true,
            created_at: '2024-01-01',
            updated_at: '2024-01-01'
          }
        ]
      };

      vi.mocked(request).mockResolvedValue(mockResponse);

      const result = await getAgentsByCategory('specialist');

      expect(request).toHaveBeenCalledWith('/ai/custom-agents/category/specialist');
      expect(result.agents).toHaveLength(1);
      expect(result.agents[0].category).toBe('specialist');
    });

    it('should URL encode category names with special characters', async () => {
      const mockResponse = { agents: [] };
      vi.mocked(request).mockResolvedValue(mockResponse);

      await getAgentsByCategory('custom/special');

      expect(request).toHaveBeenCalledWith('/ai/custom-agents/category/custom%2Fspecial');
    });
  });

  describe('getAgentPresets', () => {
    it('should fetch all agent presets', async () => {
      const mockResponse = {
        presets: [
          {
            id: 'preset1',
            name: 'code-assistant',
            display_name: 'Code Assistant',
            description: 'Helps with coding',
            category: 'specialist',
            system_prompt: 'You are a code assistant',
            created_at: '2024-01-01'
          }
        ]
      };

      vi.mocked(request).mockResolvedValue(mockResponse);

      const result = await getAgentPresets();

      expect(request).toHaveBeenCalledWith('/ai/agents/presets');
      expect(result.presets).toHaveLength(1);
    });

    it('should handle empty presets list', async () => {
      const mockResponse = { presets: [] };
      vi.mocked(request).mockResolvedValue(mockResponse);

      const result = await getAgentPresets();

      expect(result.presets).toHaveLength(0);
    });
  });

  describe('getAgentPreset', () => {
    it('should fetch a specific preset by ID', async () => {
      const mockPreset: AgentPreset = {
        id: 'preset1',
        name: 'code-assistant',
        display_name: 'Code Assistant',
        description: 'Helps with coding',
        category: 'specialist',
        system_prompt: 'You are a code assistant',
        model_preference: 'gpt-4',
        temperature: 0.5,
        capabilities: ['code'],
        is_featured: true,
        copy_count: 10,
        created_at: '2024-01-01'
      };

      vi.mocked(request).mockResolvedValue(mockPreset);

      const result = await getAgentPreset('preset1');

      expect(request).toHaveBeenCalledWith('/ai/agents/presets/preset1');
      expect(result).toEqual(mockPreset);
    });
  });

  describe('createFromPreset', () => {
    it('should create agent from preset with custom name', async () => {
      const mockCreatedAgent: CustomAgent = {
        id: '1',
        user_id: 'user1',
        name: 'my-code-assistant',
        display_name: 'My Code Assistant',
        description: 'Helps with coding',
        system_prompt: 'You are a code assistant',
        category: 'specialist',
        is_active: true,
        created_at: '2024-01-01',
        updated_at: '2024-01-01'
      };

      vi.mocked(request).mockResolvedValue({ agent: mockCreatedAgent });

      const result = await createFromPreset('preset1', 'My Code Assistant');

      expect(request).toHaveBeenCalledWith('/ai/custom-agents/from-preset/preset1', {
        method: 'POST',
        body: { name: 'My Code Assistant' }
      });
      expect(result).toEqual(mockCreatedAgent);
    });

    it('should create agent from preset without custom name', async () => {
      const mockCreatedAgent: CustomAgent = {
        id: '1',
        user_id: 'user1',
        name: 'code-assistant',
        display_name: 'Code Assistant',
        system_prompt: 'You are a code assistant',
        is_active: true,
        created_at: '2024-01-01',
        updated_at: '2024-01-01'
      };

      vi.mocked(request).mockResolvedValue({ agent: mockCreatedAgent });

      const result = await createFromPreset('preset1');

      expect(request).toHaveBeenCalledWith('/ai/custom-agents/from-preset/preset1', {
        method: 'POST',
        body: { name: undefined }
      });
      expect(result).toEqual(mockCreatedAgent);
    });
  });

  describe('testAgent', () => {
    beforeEach(() => {
      // Mock global fetch for streaming responses
      global.fetch = vi.fn();
    });

    it('should test an agent and return a readable stream', async () => {
      const mockStream = new ReadableStream();
      const mockResponse = {
        ok: true,
        body: mockStream
      } as Response;

      vi.mocked(global.fetch).mockResolvedValue(mockResponse);

      const result = await testAgent('agent1', 'Hello, test!');

      expect(global.fetch).toHaveBeenCalledWith('http://localhost:8080/ai/custom-agents/agent1/test', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ test_message: 'Hello, test!' })
      });
      expect(result).toBe(mockStream);
    });

    it('should handle test agent API errors', async () => {
      const mockResponse = {
        ok: false,
        status: 404,
        json: vi.fn().mockResolvedValue({ detail: 'Agent not found' })
      } as unknown as Response;

      vi.mocked(global.fetch).mockResolvedValue(mockResponse);

      await expect(testAgent('999', 'Hello')).rejects.toThrow('Agent not found');
    });

    it('should handle test agent network errors', async () => {
      const mockResponse = {
        ok: false,
        status: 500,
        json: vi.fn().mockRejectedValue(new Error('JSON parse error'))
      } as unknown as Response;

      vi.mocked(global.fetch).mockResolvedValue(mockResponse);

      await expect(testAgent('agent1', 'Hello')).rejects.toThrow('Test agent failed');
    });
  });

  describe('testSandbox', () => {
    beforeEach(() => {
      global.fetch = vi.fn();
    });

    it('should test sandbox configuration with full options', async () => {
      const mockStream = new ReadableStream();
      const mockResponse = {
        ok: true,
        body: mockStream
      } as Response;

      vi.mocked(global.fetch).mockResolvedValue(mockResponse);

      const config: SandboxTestRequest = {
        system_prompt: 'You are a test agent',
        test_message: 'Hello, sandbox!',
        model: 'gpt-4',
        temperature: 0.7
      };

      const result = await testSandbox(config);

      expect(global.fetch).toHaveBeenCalledWith('http://localhost:8080/ai/custom-agents/sandbox', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(config)
      });
      expect(result).toBe(mockStream);
    });

    it('should test sandbox with minimal config', async () => {
      const mockStream = new ReadableStream();
      const mockResponse = {
        ok: true,
        body: mockStream
      } as Response;

      vi.mocked(global.fetch).mockResolvedValue(mockResponse);

      const config: SandboxTestRequest = {
        system_prompt: 'You are a test agent',
        test_message: 'Hello!'
      };

      const result = await testSandbox(config);

      expect(result).toBe(mockStream);
    });

    it('should handle sandbox test errors', async () => {
      const mockResponse = {
        ok: false,
        status: 400,
        json: vi.fn().mockResolvedValue({ detail: 'Invalid configuration' })
      } as unknown as Response;

      vi.mocked(global.fetch).mockResolvedValue(mockResponse);

      const config: SandboxTestRequest = {
        system_prompt: '',
        test_message: 'Hello'
      };

      await expect(testSandbox(config)).rejects.toThrow('Invalid configuration');
    });
  });
});
