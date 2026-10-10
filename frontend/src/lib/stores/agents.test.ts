import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { get } from 'svelte/store';
import { agents, selectedAgent, agentsByCategory, activeAgents } from './agents';
import type { CustomAgent, AgentPreset } from '$lib/api/ai/types';

// Mock the AI API module
vi.mock('$lib/api/ai/ai', () => ({
  getCustomAgents: vi.fn(),
  getCustomAgent: vi.fn(),
  createCustomAgent: vi.fn(),
  updateCustomAgent: vi.fn(),
  deleteCustomAgent: vi.fn(),
  getAgentsByCategory: vi.fn(),
  getAgentPresets: vi.fn(),
  getAgentPreset: vi.fn(),
  createFromPreset: vi.fn(),
  testAgent: vi.fn(),
  testSandbox: vi.fn()
}));

// Import mocked functions
import * as aiApi from '$lib/api/ai/ai';

describe('Agents Store', () => {
  const mockAgent1: CustomAgent = {
    id: '1',
    user_id: 'user1',
    name: 'test-agent-1',
    display_name: 'Test Agent 1',
    description: 'First test agent',
    system_prompt: 'You are test agent 1',
    category: 'general',
    is_active: true,
    usage_count: 5,
    created_at: '2024-01-01',
    updated_at: '2024-01-01'
  };

  const mockAgent2: CustomAgent = {
    id: '2',
    user_id: 'user1',
    name: 'test-agent-2',
    display_name: 'Test Agent 2',
    description: 'Second test agent',
    system_prompt: 'You are test agent 2',
    category: 'specialist',
    is_active: false,
    usage_count: 10,
    created_at: '2024-01-01',
    updated_at: '2024-01-01'
  };

  const mockAgent3: CustomAgent = {
    id: '3',
    user_id: 'user1',
    name: 'test-agent-3',
    display_name: 'Test Agent 3',
    system_prompt: 'You are test agent 3',
    category: 'general',
    is_active: true,
    created_at: '2024-01-01',
    updated_at: '2024-01-01'
  };

  beforeEach(() => {
    vi.clearAllMocks();
    // Reset store to initial state to avoid test pollution
    agents.reset();
    // Reset store state by loading empty agents
    vi.mocked(aiApi.getCustomAgents).mockResolvedValue({ agents: [] });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('loadAgents', () => {
    it('should load agents successfully', async () => {
      const mockResponse = { agents: [mockAgent1, mockAgent2] };
      vi.mocked(aiApi.getCustomAgents).mockResolvedValue(mockResponse);

      await agents.loadAgents();

      const state = get(agents);
      expect(state.agents).toHaveLength(2);
      expect(state.loading).toBe(false);
      expect(state.error).toBe(null);
    });

    it('should set loading state during fetch', async () => {
      vi.mocked(aiApi.getCustomAgents).mockImplementation(
        () => new Promise((resolve) => setTimeout(() => resolve({ agents: [] }), 100))
      );

      const loadPromise = agents.loadAgents();

      // Check loading state immediately
      const loadingState = get(agents);
      expect(loadingState.loading).toBe(true);

      await loadPromise;

      const finalState = get(agents);
      expect(finalState.loading).toBe(false);
    });

    it('should filter agents by category', async () => {
      const mockResponse = { agents: [mockAgent1, mockAgent2, mockAgent3] };
      vi.mocked(aiApi.getCustomAgents).mockResolvedValue(mockResponse);

      await agents.loadAgents({ category: 'general' });

      const state = get(agents);
      expect(state.agents).toHaveLength(2);
      expect(state.agents.every((a) => a.category === 'general')).toBe(true);
    });

    it('should filter agents by search term', async () => {
      const mockResponse = { agents: [mockAgent1, mockAgent2, mockAgent3] };
      vi.mocked(aiApi.getCustomAgents).mockResolvedValue(mockResponse);

      await agents.loadAgents({ search: 'Second' });

      const state = get(agents);
      expect(state.agents).toHaveLength(1);
      expect(state.agents[0].id).toBe('2');
    });

    it('should search across name, display_name, and description', async () => {
      const mockResponse = { agents: [mockAgent1, mockAgent2, mockAgent3] };
      vi.mocked(aiApi.getCustomAgents).mockResolvedValue(mockResponse);

      // Search by name
      await agents.loadAgents({ search: 'test-agent-1' });
      let state = get(agents);
      expect(state.agents).toHaveLength(1);
      expect(state.agents[0].id).toBe('1');

      // Search by display name
      await agents.loadAgents({ search: 'Agent 2' });
      state = get(agents);
      expect(state.agents).toHaveLength(1);
      expect(state.agents[0].id).toBe('2');

      // Search by description
      await agents.loadAgents({ search: 'First test' });
      state = get(agents);
      expect(state.agents).toHaveLength(1);
      expect(state.agents[0].id).toBe('1');
    });

    it('should filter by active status', async () => {
      const mockResponse = { agents: [mockAgent1, mockAgent2, mockAgent3] };
      vi.mocked(aiApi.getCustomAgents).mockResolvedValue(mockResponse);

      await agents.loadAgents({ status: 'active' });

      const state = get(agents);
      expect(state.agents).toHaveLength(2);
      expect(state.agents.every((a) => a.is_active)).toBe(true);
    });

    it('should filter by inactive status', async () => {
      const mockResponse = { agents: [mockAgent1, mockAgent2, mockAgent3] };
      vi.mocked(aiApi.getCustomAgents).mockResolvedValue(mockResponse);

      await agents.loadAgents({ status: 'inactive' });

      const state = get(agents);
      expect(state.agents).toHaveLength(1);
      expect(state.agents[0].is_active).toBe(false);
    });

    it('should include inactive agents when status is null or inactive', async () => {
      const mockResponse = { agents: [mockAgent1, mockAgent2] };
      vi.mocked(aiApi.getCustomAgents).mockResolvedValue(mockResponse);

      await agents.loadAgents({ status: null });

      expect(aiApi.getCustomAgents).toHaveBeenCalledWith(true);
    });

    it('should apply multiple filters together', async () => {
      const mockResponse = { agents: [mockAgent1, mockAgent2, mockAgent3] };
      vi.mocked(aiApi.getCustomAgents).mockResolvedValue(mockResponse);

      await agents.loadAgents({
        category: 'general',
        search: 'Test Agent 1',
        status: 'active'
      });

      const state = get(agents);
      expect(state.agents).toHaveLength(1);
      expect(state.agents[0].id).toBe('1');
    });

    it('should surface an error and leave agents empty when API errors', async () => {
      const mockError = new Error('Failed to load agents');
      vi.mocked(aiApi.getCustomAgents).mockRejectedValue(mockError);

      await agents.loadAgents();

      const state = get(agents);
      expect(state.loading).toBe(false);
      expect(state.error).toBe('Failed to load agents');
      expect(state.agents).toEqual([]);
    });

    // ============ RACE CONDITION TESTS ============

    describe('race condition prevention', () => {
      it('should handle concurrent loadAgents with different filters (category)', async () => {
        const allAgents = [mockAgent1, mockAgent2, mockAgent3];

        // Mock API with delay to simulate network latency
        vi.mocked(aiApi.getCustomAgents).mockImplementation(async () => {
          await new Promise(resolve => setTimeout(resolve, 50));
          return { agents: allAgents };
        });

        // Fire two concurrent calls with different category filters
        const call1 = agents.loadAgents({ category: 'specialist' });
        const call2 = agents.loadAgents({ category: 'general' });

        await Promise.all([call1, call2]);

        const state = get(agents);

        // Latest call (general) should win
        expect(state.filters.category).toBe('general');
        expect(state.agents).toHaveLength(2);
        expect(state.agents.every(a => a.category === 'general')).toBe(true);
        expect(state.loading).toBe(false);
      });

      it('should handle concurrent loadAgents with different search terms', async () => {
        const allAgents = [mockAgent1, mockAgent2, mockAgent3];

        vi.mocked(aiApi.getCustomAgents).mockImplementation(async () => {
          await new Promise(resolve => setTimeout(resolve, 30));
          return { agents: allAgents };
        });

        // Fire two concurrent calls with different search terms
        const call1 = agents.loadAgents({ search: 'Test Agent 1' });
        const call2 = agents.loadAgents({ search: 'Test Agent 2' });

        await Promise.all([call1, call2]);

        const state = get(agents);

        // Latest call's search term should win
        expect(state.filters.search).toBe('Test Agent 2');
        expect(state.agents).toHaveLength(1);
        expect(state.agents[0].id).toBe('2');
        expect(state.loading).toBe(false);
      });

      it('should handle concurrent loadAgents with different status filters', async () => {
        const allAgents = [mockAgent1, mockAgent2, mockAgent3];

        vi.mocked(aiApi.getCustomAgents).mockImplementation(async () => {
          await new Promise(resolve => setTimeout(resolve, 40));
          return { agents: allAgents };
        });

        // Fire two concurrent calls with different status filters
        const call1 = agents.loadAgents({ status: 'active' });
        const call2 = agents.loadAgents({ status: 'inactive' });

        await Promise.all([call1, call2]);

        const state = get(agents);

        // Latest call (inactive) should win
        expect(state.filters.status).toBe('inactive');
        expect(state.agents).toHaveLength(1);
        expect(state.agents[0].is_active).toBe(false);
        expect(state.loading).toBe(false);
      });

      it('should handle rapid filter updates without data corruption', async () => {
        const allAgents = [mockAgent1, mockAgent2, mockAgent3];

        vi.mocked(aiApi.getCustomAgents).mockImplementation(async () => {
          await new Promise(resolve => setTimeout(resolve, 20));
          return { agents: allAgents };
        });

        // Burst test: 10 rapid concurrent calls
        const calls = Array.from({ length: 10 }, (_, i) =>
          agents.loadAgents({ category: i % 2 === 0 ? 'general' : 'specialist' })
        );

        await Promise.all(calls);

        const state = get(agents);

        // Last call's filter should win (specialist since 9 % 2 === 1)
        expect(state.filters.category).toBe('specialist');
        expect(state.agents.every(a => a.category === 'specialist')).toBe(true);
        expect(state.loading).toBe(false);
      });

      it('should handle concurrent load with page mount + filter change', async () => {
        const allAgents = [mockAgent1, mockAgent2, mockAgent3];

        vi.mocked(aiApi.getCustomAgents).mockImplementation(async () => {
          await new Promise(resolve => setTimeout(resolve, 60));
          return { agents: allAgents };
        });

        // Simulate page mount (initial load)
        const initialLoad = agents.loadAgents();

        // Simulate user clicking filter before initial load completes
        await new Promise(resolve => setTimeout(resolve, 10));
        const filterChange = agents.loadAgents({ category: 'specialist' });

        await Promise.all([initialLoad, filterChange]);

        const state = get(agents);

        // Filter change should win
        expect(state.filters.category).toBe('specialist');
        expect(state.agents.every(a => a.category === 'specialist')).toBe(true);
        expect(state.loading).toBe(false);
      });

      it('should only update with latest request result (out-of-order responses)', async () => {
        const allAgents = [mockAgent1, mockAgent2, mockAgent3];

        let call1Delay = 100;
        let call2Delay = 10; // Second call returns faster

        vi.mocked(aiApi.getCustomAgents).mockImplementation(async () => {
          const delay = call1Delay;
          call1Delay = call2Delay;
          call2Delay = 0;

          await new Promise(resolve => setTimeout(resolve, delay));
          return { agents: allAgents };
        });

        // Call 1: Slow response
        const call1 = agents.loadAgents({ category: 'specialist' });

        // Call 2: Fast response (arrives first)
        await new Promise(resolve => setTimeout(resolve, 5));
        const call2 = agents.loadAgents({ category: 'general' });

        await Promise.all([call1, call2]);

        const state = get(agents);

        // Call 2's filter should win even though Call 1's response was slower
        expect(state.filters.category).toBe('general');
        expect(state.agents.every(a => a.category === 'general')).toBe(true);
      });

      it('should handle concurrent error + success requests', async () => {
        const allAgents = [mockAgent1, mockAgent2];

        let firstCall = true;
        vi.mocked(aiApi.getCustomAgents).mockImplementation(async () => {
          await new Promise(resolve => setTimeout(resolve, 30));

          if (firstCall) {
            firstCall = false;
            throw new Error('Network error');
          }

          return { agents: allAgents };
        });

        // Call 1: Will error
        const call1 = agents.loadAgents({ category: 'specialist' });

        // Call 2: Will succeed
        const call2 = agents.loadAgents({ category: 'general' });

        await Promise.all([call1, call2].map(p => p.catch(() => {})));

        const state = get(agents);

        // Second call should have succeeded with filtered results
        expect(state.filters.category).toBe('general');
        expect(state.agents).toHaveLength(1); // Only mockAgent1 has category 'general'
        expect(state.agents[0].id).toBe('1');
        expect(state.error).toBe(null);
        expect(state.loading).toBe(false);
      });

      it('should maintain filter consistency after all concurrent calls complete', async () => {
        const allAgents = [mockAgent1, mockAgent2, mockAgent3];

        vi.mocked(aiApi.getCustomAgents).mockResolvedValue({ agents: allAgents });

        // Make 5 concurrent calls with different filters
        await Promise.all([
          agents.loadAgents({ category: 'general', status: 'active' }),
          agents.loadAgents({ category: 'specialist', search: 'test' }),
          agents.loadAgents({ category: 'general', status: 'inactive' }),
          agents.loadAgents({ search: 'Agent 1' }),
          agents.loadAgents({ status: 'active' })
        ]);

        const state = get(agents);

        // Filters should be consistent with displayed agents
        const filteredAgents = allAgents.filter(a => {
          let matches = true;

          if (state.filters.category) {
            matches = matches && a.category === state.filters.category;
          }

          if (state.filters.search) {
            const search = state.filters.search.toLowerCase();
            matches = matches && (
              a.name.toLowerCase().includes(search) ||
              a.display_name.toLowerCase().includes(search) ||
              (a.description?.toLowerCase().includes(search) ?? false)
            );
          }

          if (state.filters.status === 'active') {
            matches = matches && a.is_active === true;
          } else if (state.filters.status === 'inactive') {
            matches = matches && a.is_active !== true;
          }

          return matches;
        });

        // Displayed agents should match current filters
        expect(state.agents).toEqual(filteredAgents);
      });
    });
  });

  describe('loadAgent', () => {
    it('should load a specific agent', async () => {
      vi.mocked(aiApi.getCustomAgent).mockResolvedValue(mockAgent1);

      const result = await agents.loadAgent('1');

      const state = get(agents);
      expect(state.currentAgent).toEqual(mockAgent1);
      expect(result).toEqual(mockAgent1);
    });

    it('should handle agent not found', async () => {
      const mockError = new Error('Agent not found');
      vi.mocked(aiApi.getCustomAgent).mockRejectedValue(mockError);

      const result = await agents.loadAgent('999');

      const state = get(agents);
      expect(state.currentAgent).toBe(null);
      expect(state.error).toBe('Agent not found');
      expect(result).toBe(null);
    });
  });

  describe('createAgent', () => {
    it('should create a new agent and add to store', async () => {
      const newAgentData: Partial<CustomAgent> = {
        name: 'new-agent',
        display_name: 'New Agent',
        system_prompt: 'You are new'
      };

      const createdAgent: CustomAgent = {
        id: '4',
        user_id: 'user1',
        ...newAgentData,
        is_active: true,
        created_at: '2024-01-02',
        updated_at: '2024-01-02'
      } as CustomAgent;

      vi.mocked(aiApi.createCustomAgent).mockResolvedValue(createdAgent);
      vi.mocked(aiApi.getCustomAgents).mockResolvedValue({ agents: [mockAgent1] });

      await agents.loadAgents();
      const result = await agents.createAgent(newAgentData);

      const state = get(agents);
      expect(state.agents).toHaveLength(2);
      expect(state.agents[0]).toEqual(createdAgent);
      expect(result).toEqual(createdAgent);
    });

    it('should propagate creation errors', async () => {
      const mockError = new Error('Validation failed');
      vi.mocked(aiApi.createCustomAgent).mockRejectedValue(mockError);

      await expect(agents.createAgent({})).rejects.toThrow('Validation failed');
    });
  });

  describe('updateAgent', () => {
    it('should update an existing agent in the store', async () => {
      const updateData = { display_name: 'Updated Agent 1' };
      const updatedAgent = { ...mockAgent1, ...updateData };

      vi.mocked(aiApi.getCustomAgents).mockResolvedValue({ agents: [mockAgent1, mockAgent2] });
      vi.mocked(aiApi.updateCustomAgent).mockResolvedValue(updatedAgent);

      await agents.loadAgents();
      const result = await agents.updateAgent('1', updateData);

      const state = get(agents);
      expect(state.agents[0].display_name).toBe('Updated Agent 1'); // mockAgent1 is at index 0
      expect(result.display_name).toBe('Updated Agent 1');
    });

    it('should update currentAgent if it matches', async () => {
      const updateData = { display_name: 'Updated Agent' };
      const updatedAgent = { ...mockAgent1, ...updateData };

      vi.mocked(aiApi.getCustomAgent).mockResolvedValue(mockAgent1);
      vi.mocked(aiApi.updateCustomAgent).mockResolvedValue(updatedAgent);

      await agents.loadAgent('1');
      await agents.updateAgent('1', updateData);

      const state = get(agents);
      expect(state.currentAgent?.display_name).toBe('Updated Agent');
    });

    it('should not update currentAgent if different agent', async () => {
      const updateData = { display_name: 'Updated Agent 2' };
      const updatedAgent = { ...mockAgent2, ...updateData };

      vi.mocked(aiApi.getCustomAgent).mockResolvedValue(mockAgent1);
      vi.mocked(aiApi.updateCustomAgent).mockResolvedValue(updatedAgent);

      await agents.loadAgent('1');
      await agents.updateAgent('2', updateData);

      const state = get(agents);
      expect(state.currentAgent?.id).toBe('1');
      expect(state.currentAgent?.display_name).toBe('Test Agent 1');
    });

    it('should propagate update errors', async () => {
      const mockError = new Error('Agent not found');
      vi.mocked(aiApi.updateCustomAgent).mockRejectedValue(mockError);

      await expect(agents.updateAgent('999', {})).rejects.toThrow('Agent not found');
    });
  });

  describe('deleteAgent', () => {
    it('should delete an agent from the store', async () => {
      vi.mocked(aiApi.getCustomAgents).mockResolvedValue({ agents: [mockAgent1, mockAgent2] });
      vi.mocked(aiApi.deleteCustomAgent).mockResolvedValue({ message: 'Deleted' });

      await agents.loadAgents();
      await agents.deleteAgent('1');

      const state = get(agents);
      expect(state.agents).toHaveLength(1);
      expect(state.agents[0].id).toBe('2');
    });

    it('should clear currentAgent if deleted', async () => {
      vi.mocked(aiApi.getCustomAgent).mockResolvedValue(mockAgent1);
      vi.mocked(aiApi.deleteCustomAgent).mockResolvedValue({ message: 'Deleted' });

      await agents.loadAgent('1');
      await agents.deleteAgent('1');

      const state = get(agents);
      expect(state.currentAgent).toBe(null);
    });

    it('should not clear currentAgent if different agent deleted', async () => {
      vi.mocked(aiApi.getCustomAgent).mockResolvedValue(mockAgent1);
      vi.mocked(aiApi.deleteCustomAgent).mockResolvedValue({ message: 'Deleted' });

      await agents.loadAgent('1');
      await agents.deleteAgent('2');

      const state = get(agents);
      expect(state.currentAgent?.id).toBe('1');
    });

    it('should propagate deletion errors', async () => {
      const mockError = new Error('Agent not found');
      vi.mocked(aiApi.deleteCustomAgent).mockRejectedValue(mockError);

      await expect(agents.deleteAgent('999')).rejects.toThrow('Agent not found');
    });
  });

  describe('setCurrentAgent', () => {
    it('should set the current agent', () => {
      agents.setCurrentAgent(mockAgent1);

      const state = get(agents);
      expect(state.currentAgent).toEqual(mockAgent1);
    });

    it('should clear the current agent', () => {
      agents.setCurrentAgent(mockAgent1);
      agents.setCurrentAgent(null);

      const state = get(agents);
      expect(state.currentAgent).toBe(null);
    });
  });

  describe('clearCurrent', () => {
    it('should clear the current agent', () => {
      agents.setCurrentAgent(mockAgent1);
      agents.clearCurrent();

      const state = get(agents);
      expect(state.currentAgent).toBe(null);
    });
  });

  describe('setFilters', () => {
    it('should update filters', () => {
      agents.setFilters({ category: 'specialist', search: 'test' });

      const state = get(agents);
      expect(state.filters.category).toBe('specialist');
      expect(state.filters.search).toBe('test');
    });

    it('should merge with existing filters', () => {
      agents.setFilters({ category: 'specialist' });
      agents.setFilters({ search: 'test' });

      const state = get(agents);
      expect(state.filters.category).toBe('specialist');
      expect(state.filters.search).toBe('test');
    });
  });

  describe('clearFilters', () => {
    it('should reset all filters to defaults', () => {
      agents.setFilters({ category: 'specialist', search: 'test', status: 'active' });
      agents.clearFilters();

      const state = get(agents);
      expect(state.filters.category).toBe(null);
      expect(state.filters.search).toBe('');
      expect(state.filters.status).toBe(null);
    });
  });

  describe('clearError', () => {
    it('should clear error state', async () => {
      const mockError = new Error('Test error');
      vi.mocked(aiApi.getCustomAgents).mockRejectedValue(mockError);

      await agents.loadAgents();

      let state = get(agents);
      expect(state.error).toBe('Test error');

      agents.clearError();

      state = get(agents);
      expect(state.error).toBe(null);
    });
  });

  describe('loadPresets', () => {
    it('should load agent presets', async () => {
      const mockPresets: AgentPreset[] = [
        {
          id: 'preset1',
          name: 'code-assistant',
          display_name: 'Code Assistant',
          description: 'Helps with code',
          category: 'specialist',
          system_prompt: 'You assist with code',
          created_at: '2024-01-01'
        }
      ];

      vi.mocked(aiApi.getAgentPresets).mockResolvedValue({ presets: mockPresets });

      await agents.loadPresets();

      const state = get(agents);
      expect(state.presets).toHaveLength(1);
      expect(state.presets[0].id).toBe('preset1');
    });

    it('should surface an error and leave presets empty when preset loading errors', async () => {
      const mockError = new Error('Failed to load presets');
      vi.mocked(aiApi.getAgentPresets).mockRejectedValue(mockError);

      await agents.loadPresets();

      const state = get(agents);
      expect(state.error).toBe('Failed to load presets');
      expect(state.presets).toEqual([]);
    });
  });

  describe('loadPreset', () => {
    it('should load a specific preset', async () => {
      const mockPreset: AgentPreset = {
        id: 'preset1',
        name: 'code-assistant',
        display_name: 'Code Assistant',
        description: 'Helps with code',
        category: 'specialist',
        system_prompt: 'You assist with code',
        created_at: '2024-01-01'
      };

      vi.mocked(aiApi.getAgentPreset).mockResolvedValue(mockPreset);

      const result = await agents.loadPreset('preset1');

      expect(result).toEqual(mockPreset);
    });

    it('should propagate preset loading errors', async () => {
      const mockError = new Error('Preset not found');
      vi.mocked(aiApi.getAgentPreset).mockRejectedValue(mockError);

      await expect(agents.loadPreset('999')).rejects.toThrow('Preset not found');
    });
  });

  describe('createFromPreset', () => {
    it('should create agent from preset', async () => {
      const createdAgent: CustomAgent = {
        id: '4',
        user_id: 'user1',
        name: 'my-code-assistant',
        display_name: 'My Code Assistant',
        system_prompt: 'You assist with code',
        is_active: true,
        created_at: '2024-01-02',
        updated_at: '2024-01-02'
      };

      vi.mocked(aiApi.createFromPreset).mockResolvedValue(createdAgent);
      vi.mocked(aiApi.getCustomAgents).mockResolvedValue({ agents: [] });

      await agents.loadAgents();
      const result = await agents.createFromPreset('preset1', 'My Code Assistant');

      const state = get(agents);
      expect(state.agents).toHaveLength(1);
      expect(state.agents[0].id).toBe('4');
      expect(result).toEqual(createdAgent);
    });
  });

  describe('testAgent', () => {
    it('should return a readable stream for testing', async () => {
      const mockStream = new ReadableStream();
      vi.mocked(aiApi.testAgent).mockResolvedValue(mockStream);

      const result = await agents.testAgent('1', 'Test message');

      expect(result).toBe(mockStream);
      expect(aiApi.testAgent).toHaveBeenCalledWith('1', 'Test message');
    });

    it('should propagate test errors', async () => {
      const mockError = new Error('Test failed');
      vi.mocked(aiApi.testAgent).mockRejectedValue(mockError);

      await expect(agents.testAgent('1', 'Test')).rejects.toThrow('Test failed');
    });
  });

  describe('testSandbox', () => {
    it('should return a readable stream for sandbox testing', async () => {
      const mockStream = new ReadableStream();
      vi.mocked(aiApi.testSandbox).mockResolvedValue(mockStream);

      const config = {
        system_prompt: 'You are a test agent',
        message: 'Hello',
        model: 'gpt-4',
        temperature: 0.7
      };

      const result = await agents.testSandbox(config);

      expect(result).toBe(mockStream);
      expect(aiApi.testSandbox).toHaveBeenCalledWith({
        system_prompt: 'You are a test agent',
        test_message: 'Hello',
        model: 'gpt-4',
        temperature: 0.7
      });
    });

    it('should propagate sandbox test errors', async () => {
      const mockError = new Error('Sandbox test failed');
      vi.mocked(aiApi.testSandbox).mockRejectedValue(mockError);

      await expect(
        agents.testSandbox({ system_prompt: 'test', message: 'hello' })
      ).rejects.toThrow('Sandbox test failed');

      expect(aiApi.testSandbox).toHaveBeenCalledWith({
        system_prompt: 'test',
        test_message: 'hello'
      });
    });
  });

  describe('Derived Stores', () => {
    describe('selectedAgent', () => {
      it('should reflect current agent', () => {
        agents.setCurrentAgent(mockAgent1);

        const selected = get(selectedAgent);
        expect(selected).toEqual(mockAgent1);
      });

      it('should be null when no agent selected', () => {
        agents.setCurrentAgent(null);

        const selected = get(selectedAgent);
        expect(selected).toBe(null);
      });
    });

    describe('agentsByCategory', () => {
      it('should group agents by category', async () => {
        vi.mocked(aiApi.getCustomAgents).mockResolvedValue({
          agents: [mockAgent1, mockAgent2, mockAgent3]
        });

        await agents.loadAgents();

        const byCategory = get(agentsByCategory);
        expect(byCategory['general']).toHaveLength(2);
        expect(byCategory['specialist']).toHaveLength(1);
      });

      it('should handle uncategorized agents', async () => {
        const uncategorizedAgent = { ...mockAgent1, category: undefined };
        vi.mocked(aiApi.getCustomAgents).mockResolvedValue({
          agents: [uncategorizedAgent]
        });

        await agents.loadAgents();

        const byCategory = get(agentsByCategory);
        expect(byCategory['uncategorized']).toHaveLength(1);
      });

      it('should be empty when no agents loaded', async () => {
        vi.mocked(aiApi.getCustomAgents).mockResolvedValue({ agents: [] });

        await agents.loadAgents();

        const byCategory = get(agentsByCategory);
        expect(Object.keys(byCategory)).toHaveLength(0);
      });
    });

    describe('activeAgents', () => {
      it('should filter only active agents', async () => {
        vi.mocked(aiApi.getCustomAgents).mockResolvedValue({
          agents: [mockAgent1, mockAgent2, mockAgent3]
        });

        await agents.loadAgents();

        const active = get(activeAgents);
        expect(active).toHaveLength(2);
        expect(active.every((a) => a.is_active)).toBe(true);
      });

      it('should be empty when no active agents', async () => {
        vi.mocked(aiApi.getCustomAgents).mockResolvedValue({
          agents: [mockAgent2]
        });

        await agents.loadAgents();

        const active = get(activeAgents);
        expect(active).toHaveLength(0);
      });
    });
  });
});
