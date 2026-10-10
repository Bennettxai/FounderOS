import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import {
  getMCPConnectors,
  createMCPConnector,
  updateMCPConnector,
  deleteMCPConnector,
  testMCPConnector,
  discoverMCPConnectorTools
} from './integrations';
import type { MCPConnector, CreateMCPConnectorData, TestMCPConnectorResponse } from './integrations';

vi.mock('../base', () => ({
  request: vi.fn(),
  getApiBaseUrl: vi.fn(() => 'http://localhost:8080')
}));

import { request } from '../base';

const mockServer: MCPConnector = {
  id: 'srv-1',
  name: 'test-server',
  description: 'A test MCP server',
  server_url: 'https://mcp.example.com/sse',
  auth_type: 'none',
  has_auth: false,
  enabled: true,
  status: 'connected',
  transport: 'sse',
  tools: [{ name: 'greet', description: 'Says hello', input_schema: {} }],
  tool_count: 1,
  last_connected_at: '2026-03-08T00:00:00Z',
  created_at: '2026-03-08T00:00:00Z',
  updated_at: '2026-03-08T00:00:00Z'
};

describe('MCP Connectors API', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('getMCPConnectors', () => {
    it('should fetch connector list', async () => {
      const mockResponse = { servers: [mockServer] };
      vi.mocked(request).mockResolvedValue(mockResponse);

      const result = await getMCPConnectors();

      expect(request).toHaveBeenCalledWith('/integrations/mcp/connectors');
      expect(result.servers).toHaveLength(1);
      expect(result.servers[0].name).toBe('test-server');
    });

    it('should handle empty list', async () => {
      vi.mocked(request).mockResolvedValue({ servers: [] });

      const result = await getMCPConnectors();

      expect(result.servers).toHaveLength(0);
    });
  });

  describe('createMCPConnector', () => {
    it('should send correct payload', async () => {
      vi.mocked(request).mockResolvedValue(mockServer);

      const data: CreateMCPConnectorData = {
        name: 'new-server',
        server_url: 'https://mcp.example.com/sse',
        auth_type: 'none'
      };
      const result = await createMCPConnector(data);

      expect(request).toHaveBeenCalledWith('/integrations/mcp/connectors', {
        method: 'POST',
        body: data
      });
      expect(result.id).toBe('srv-1');
    });

    it('should send auth_token when provided', async () => {
      vi.mocked(request).mockResolvedValue(mockServer);

      const data: CreateMCPConnectorData = {
        name: 'authed-server',
        server_url: 'https://mcp.example.com/sse',
        auth_type: 'bearer',
        auth_token: 'secret-token'
      };
      await createMCPConnector(data);

      expect(request).toHaveBeenCalledWith('/integrations/mcp/connectors', {
        method: 'POST',
        body: data
      });
    });
  });

  describe('updateMCPConnector', () => {
    it('should send PUT with id', async () => {
      vi.mocked(request).mockResolvedValue(mockServer);

      const updates = { description: 'Updated desc' };
      await updateMCPConnector('srv-1', updates);

      expect(request).toHaveBeenCalledWith('/integrations/mcp/connectors/srv-1', {
        method: 'PUT',
        body: updates
      });
    });
  });

  describe('deleteMCPConnector', () => {
    it('should send DELETE with id', async () => {
      vi.mocked(request).mockResolvedValue({});

      await deleteMCPConnector('srv-1');

      expect(request).toHaveBeenCalledWith('/integrations/mcp/connectors/srv-1', {
        method: 'DELETE'
      });
    });
  });

  describe('testMCPConnector', () => {
    it('should return success with tools count', async () => {
      const testResponse: TestMCPConnectorResponse = {
        success: true,
        message: 'Connected',
        tools_count: 3,
        tools: [
          { name: 'tool1', description: 'desc1', input_schema: {} },
          { name: 'tool2', description: 'desc2', input_schema: {} },
          { name: 'tool3', description: 'desc3', input_schema: {} }
        ]
      };
      vi.mocked(request).mockResolvedValue(testResponse);

      const result = await testMCPConnector('srv-1');

      expect(request).toHaveBeenCalledWith('/integrations/mcp/connectors/srv-1/test', {
        method: 'POST'
      });
      expect(result.success).toBe(true);
      expect(result.tools_count).toBe(3);
    });

    it('should return failure message', async () => {
      const testResponse: TestMCPConnectorResponse = {
        success: false,
        message: 'Connection refused'
      };
      vi.mocked(request).mockResolvedValue(testResponse);

      const result = await testMCPConnector('srv-1');

      expect(result.success).toBe(false);
      expect(result.message).toBe('Connection refused');
    });
  });

  describe('discoverMCPConnectorTools', () => {
    it('should call discover endpoint', async () => {
      const discoverResponse: TestMCPConnectorResponse = {
        success: true,
        message: 'Discovered',
        tools_count: 2,
        tools: [
          { name: 'tool-a', description: 'A', input_schema: {} },
          { name: 'tool-b', description: 'B', input_schema: {} }
        ]
      };
      vi.mocked(request).mockResolvedValue(discoverResponse);

      const result = await discoverMCPConnectorTools('srv-1');

      expect(request).toHaveBeenCalledWith('/integrations/mcp/connectors/srv-1/discover', {
        method: 'POST'
      });
      expect(result.tools).toHaveLength(2);
    });
  });
});
