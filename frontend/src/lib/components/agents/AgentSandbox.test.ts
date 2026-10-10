import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, fireEvent, screen, waitFor } from '@testing-library/svelte';
import AgentSandbox from './AgentSandbox.svelte';
import type { SandboxTestRequest } from '$lib/api/ai/types';

// Mock the AI API module
vi.mock('$lib/api/ai', () => ({
  testAgent: vi.fn(),
  testSandbox: vi.fn()
}));

// Import mocked functions
import { testAgent, testSandbox } from '$lib/api/ai';

// Helper to create a mock ReadableStream with SSE data
function createMockSSEStream(events: Array<{ type: string; data?: string; [key: string]: any }>) {
  const encoder = new TextEncoder();

  return new ReadableStream({
    start(controller) {
      for (const event of events) {
        const sseData = `data: ${JSON.stringify(event)}\n\n`;
        controller.enqueue(encoder.encode(sseData));
      }
      controller.close();
    }
  });
}

describe('AgentSandbox Component', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('Rendering', () => {
    it('should render sandbox with all basic elements', () => {
      render(AgentSandbox, {
        props: { agentId: 'agent1' }
      });

      expect(screen.getByText('Agent Sandbox')).toBeTruthy();
      expect(screen.getByText('Test your agent configuration in real-time')).toBeTruthy();
      expect(screen.getByLabelText('Test Message')).toBeTruthy();
      expect(screen.getByText('Test')).toBeTruthy();
      expect(screen.getByText('Clear')).toBeTruthy();
    });

    it('should render with systemPrompt prop', () => {
      render(AgentSandbox, {
        props: { systemPrompt: 'You are a test agent' }
      });

      expect(screen.getByLabelText('Test Message')).toBeTruthy();
    });

    it('should show advanced options when expanded', async () => {
      render(AgentSandbox, {
        props: { agentId: 'agent1' }
      });

      const advancedButton = screen.getByText('Advanced Options');
      await fireEvent.click(advancedButton);

      expect(screen.getByLabelText('Model Override (optional)')).toBeTruthy();
      // Temperature slider exists (check by ID since value changes with interaction)
      expect(screen.getByLabelText(/^Temperature:/)).toBeTruthy();
    });

    it('should not show advanced options by default', () => {
      render(AgentSandbox, {
        props: { agentId: 'agent1' }
      });

      expect(screen.queryByLabelText('Model Override (optional)')).toBeFalsy();
    });

    it('should show empty response placeholder initially', () => {
      render(AgentSandbox, {
        props: { agentId: 'agent1' }
      });

      expect(
        screen.getByText('No response yet. Send a test message to see the agent\'s response.')
      ).toBeTruthy();
    });
  });

  describe('Test Input', () => {
    it('should update test message on input', async () => {
      render(AgentSandbox, {
        props: { agentId: 'agent1' }
      });

      const textarea = screen.getByLabelText('Test Message') as HTMLTextAreaElement;
      await fireEvent.input(textarea, { target: { value: 'Hello, agent!' } });

      expect(textarea.value).toBe('Hello, agent!');
    });

    it('should disable test button when message is empty', () => {
      render(AgentSandbox, {
        props: { agentId: 'agent1' }
      });

      const testButton = screen.getByText('Test') as HTMLButtonElement;
      expect(testButton.disabled).toBe(true);
    });

    it('should enable test button when message is entered', async () => {
      render(AgentSandbox, {
        props: { agentId: 'agent1' }
      });

      const textarea = screen.getByLabelText('Test Message');
      await fireEvent.input(textarea, { target: { value: 'Test message' } });

      const testButton = screen.getByText('Test') as HTMLButtonElement;
      expect(testButton.disabled).toBe(false);
    });

    it('should clear message when Clear button is clicked', async () => {
      render(AgentSandbox, {
        props: { agentId: 'agent1' }
      });

      const textarea = screen.getByLabelText('Test Message') as HTMLTextAreaElement;
      await fireEvent.input(textarea, { target: { value: 'Test message' } });

      const clearButton = screen.getByText('Clear');
      await fireEvent.click(clearButton);

      expect(textarea.value).toBe('');
    });
  });

  describe('Advanced Options', () => {
    it('should update model override', async () => {
      render(AgentSandbox, {
        props: { agentId: 'agent1' }
      });

      const advancedButton = screen.getByText('Advanced Options');
      await fireEvent.click(advancedButton);

      const modelInput = screen.getByLabelText('Model Override (optional)') as HTMLInputElement;
      await fireEvent.input(modelInput, { target: { value: 'gpt-4o' } });

      expect(modelInput.value).toBe('gpt-4o');
    });

    it('should update temperature slider', async () => {
      render(AgentSandbox, {
        props: { agentId: 'agent1' }
      });

      const advancedButton = screen.getByText('Advanced Options');
      await fireEvent.click(advancedButton);

      const tempSlider = screen.getByLabelText(/Temperature:/) as HTMLInputElement;
      await fireEvent.input(tempSlider, { target: { value: '0.5' } });

      expect(screen.getByText('Temperature: 0.5')).toBeTruthy();
    });

    it('should toggle advanced options visibility', async () => {
      render(AgentSandbox, {
        props: { agentId: 'agent1' }
      });

      const advancedButton = screen.getByText('Advanced Options');

      await fireEvent.click(advancedButton);
      expect(screen.getByLabelText('Model Override (optional)')).toBeTruthy();

      await fireEvent.click(advancedButton);
      expect(screen.queryByLabelText('Model Override (optional)')).toBeFalsy();
    });
  });

  describe('Testing with Agent ID', () => {
    it('should call testAgent API with correct parameters', async () => {
      const mockStream = createMockSSEStream([
        { type: 'content', data: 'Hello' },
        { type: 'content', data: ' world!' },
        { type: 'done', tokens: 10, model: 'gpt-4' }
      ]);

      vi.mocked(testAgent).mockResolvedValue(mockStream);

      render(AgentSandbox, {
        props: { agentId: 'agent1' }
      });

      const textarea = screen.getByLabelText('Test Message');
      await fireEvent.input(textarea, { target: { value: 'Test message' } });

      const testButton = screen.getByText('Test');
      await fireEvent.click(testButton);

      await waitFor(() => {
        expect(testAgent).toHaveBeenCalledWith('agent1', 'Test message');
      });
    });

    it('should display streaming response content', async () => {
      const mockStream = createMockSSEStream([
        { type: 'content', data: 'Hello' },
        { type: 'content', data: ' from' },
        { type: 'content', data: ' agent!' },
        { type: 'done' }
      ]);

      vi.mocked(testAgent).mockResolvedValue(mockStream);

      render(AgentSandbox, {
        props: { agentId: 'agent1' }
      });

      const textarea = screen.getByLabelText('Test Message');
      await fireEvent.input(textarea, { target: { value: 'Test' } });

      const testButton = screen.getByText('Test');
      await fireEvent.click(testButton);

      await waitFor(() => {
        expect(screen.getByText(/Hello from agent!/)).toBeTruthy();
      }, { timeout: 3000 });
    });

    it('should display metadata when available', async () => {
      const mockStream = createMockSSEStream([
        { type: 'content', data: 'Response' },
        { type: 'done', tokens: 42, model: 'gpt-4', duration: 1500 }
      ]);

      vi.mocked(testAgent).mockResolvedValue(mockStream);

      render(AgentSandbox, {
        props: { agentId: 'agent1' }
      });

      const textarea = screen.getByLabelText('Test Message');
      await fireEvent.input(textarea, { target: { value: 'Test' } });

      const testButton = screen.getByText('Test');
      await fireEvent.click(testButton);

      await waitFor(() => {
        expect(screen.getByText(/gpt-4/)).toBeTruthy();
        expect(screen.getByText(/42/)).toBeTruthy(); // tokens
      }, { timeout: 3000 });
    });
  });

  describe('Testing with Sandbox Configuration', () => {
    it('should call testSandbox API with correct parameters', async () => {
      const mockStream = createMockSSEStream([
        { type: 'content', data: 'Sandbox response' },
        { type: 'done' }
      ]);

      vi.mocked(testSandbox).mockResolvedValue(mockStream);

      render(AgentSandbox, {
        props: { systemPrompt: 'You are a test agent' }
      });

      const textarea = screen.getByLabelText('Test Message');
      await fireEvent.input(textarea, { target: { value: 'Test message' } });

      const testButton = screen.getByText('Test');
      await fireEvent.click(testButton);

      await waitFor(() => {
        expect(testSandbox).toHaveBeenCalledWith({
          system_prompt: 'You are a test agent',
          test_message: 'Test message',
          model: undefined,
          temperature: undefined
        });
      });
    });

    it('should include model override in sandbox config', async () => {
      const mockStream = createMockSSEStream([{ type: 'content', data: 'OK' }, { type: 'done' }]);
      vi.mocked(testSandbox).mockResolvedValue(mockStream);

      render(AgentSandbox, {
        props: { systemPrompt: 'You are a test agent' }
      });

      // Open advanced options
      const advancedButton = screen.getByText('Advanced Options');
      await fireEvent.click(advancedButton);

      // Set model override
      const modelInput = screen.getByLabelText('Model Override (optional)');
      await fireEvent.input(modelInput, { target: { value: 'gpt-4o' } });

      // Set message and test
      const textarea = screen.getByLabelText('Test Message');
      await fireEvent.input(textarea, { target: { value: 'Test' } });

      const testButton = screen.getByText('Test');
      await fireEvent.click(testButton);

      await waitFor(() => {
        expect(testSandbox).toHaveBeenCalledWith(
          expect.objectContaining({
            model: 'gpt-4o'
          })
        );
      });
    });

    it('should include temperature in sandbox config', async () => {
      const mockStream = createMockSSEStream([{ type: 'content', data: 'OK' }, { type: 'done' }]);
      vi.mocked(testSandbox).mockResolvedValue(mockStream);

      render(AgentSandbox, {
        props: { systemPrompt: 'You are a test agent' }
      });

      const advancedButton = screen.getByText('Advanced Options');
      await fireEvent.click(advancedButton);

      const tempSlider = screen.getByLabelText(/Temperature:/);
      await fireEvent.input(tempSlider, { target: { value: '0.8' } });

      const textarea = screen.getByLabelText('Test Message');
      await fireEvent.input(textarea, { target: { value: 'Test' } });

      const testButton = screen.getByText('Test');
      await fireEvent.click(testButton);

      await waitFor(() => {
        expect(testSandbox).toHaveBeenCalledWith(
          expect.objectContaining({
            temperature: 0.8
          })
        );
      });
    });
  });

  describe('Loading States', () => {
    it('should show loading indicator during test', async () => {
      const mockStream = createMockSSEStream([
        { type: 'content', data: 'Response' },
        { type: 'done' }
      ]);

      vi.mocked(testAgent).mockImplementation(
        () => new Promise((resolve) => setTimeout(() => resolve(mockStream), 100))
      );

      render(AgentSandbox, {
        props: { agentId: 'agent1' }
      });

      const textarea = screen.getByLabelText('Test Message');
      await fireEvent.input(textarea, { target: { value: 'Test' } });

      const testButton = screen.getByText('Test');
      await fireEvent.click(testButton);

      expect(screen.getByText('Waiting for response...')).toBeTruthy();
    });

    it('should show Stop button during loading', async () => {
      const mockStream = createMockSSEStream([{ type: 'content', data: 'Test' }]);

      vi.mocked(testAgent).mockResolvedValue(mockStream);

      render(AgentSandbox, {
        props: { agentId: 'agent1' }
      });

      const textarea = screen.getByLabelText('Test Message');
      await fireEvent.input(textarea, { target: { value: 'Test' } });

      const testButton = screen.getByText('Test');
      await fireEvent.click(testButton);

      await waitFor(() => {
        expect(screen.queryByText('Stop')).toBeTruthy();
      });
    });

    it('should disable textarea during loading', async () => {
      const mockStream = createMockSSEStream([{ type: 'content', data: 'Test' }]);
      vi.mocked(testAgent).mockResolvedValue(mockStream);

      render(AgentSandbox, {
        props: { agentId: 'agent1' }
      });

      const textarea = screen.getByLabelText('Test Message') as HTMLTextAreaElement;
      await fireEvent.input(textarea, { target: { value: 'Test' } });

      const testButton = screen.getByText('Test');
      await fireEvent.click(testButton);

      await waitFor(() => {
        expect(textarea.disabled).toBe(true);
      });
    });
  });

  describe('Error Handling', () => {
    it('should display error message on API failure', async () => {
      const mockError = new Error('Test failed: Network error');
      vi.mocked(testAgent).mockRejectedValue(mockError);

      render(AgentSandbox, {
        props: { agentId: 'agent1' }
      });

      const textarea = screen.getByLabelText('Test Message');
      await fireEvent.input(textarea, { target: { value: 'Test' } });

      const testButton = screen.getByText('Test');
      await fireEvent.click(testButton);

      await waitFor(() => {
        expect(screen.getByText('Error')).toBeTruthy();
        expect(screen.getByText(/Network error/)).toBeTruthy();
      });
    });

    it('should handle missing agentId and systemPrompt', async () => {
      render(AgentSandbox, {
        props: {}
      });

      const textarea = screen.getByLabelText('Test Message');
      await fireEvent.input(textarea, { target: { value: 'Test' } });

      const testButton = screen.getByText('Test');
      await fireEvent.click(testButton);

      await waitFor(() => {
        expect(screen.getByText(/Either agentId or systemPrompt must be provided/)).toBeTruthy();
      });
    });

    it('should handle null stream from API', async () => {
      vi.mocked(testAgent).mockResolvedValue(null);

      render(AgentSandbox, {
        props: { agentId: 'agent1' }
      });

      const textarea = screen.getByLabelText('Test Message');
      await fireEvent.input(textarea, { target: { value: 'Test' } });

      const testButton = screen.getByText('Test');
      await fireEvent.click(testButton);

      await waitFor(() => {
        expect(screen.getByText(/No stream returned from API/)).toBeTruthy();
      });
    });

    it('should add error to history', async () => {
      const mockError = new Error('Test error');
      vi.mocked(testAgent).mockRejectedValue(mockError);

      render(AgentSandbox, {
        props: { agentId: 'agent1' }
      });

      const textarea = screen.getByLabelText('Test Message');
      await fireEvent.input(textarea, { target: { value: 'Test message' } });

      const testButton = screen.getByText('Test');
      await fireEvent.click(testButton);

      await waitFor(() => {
        const historyButton = screen.getByText(/Test History/);
        expect(historyButton).toBeTruthy();
      });
    });
  });

  describe('History', () => {
    it('should add successful test to history', async () => {
      const mockStream = createMockSSEStream([
        { type: 'content', data: 'Response' },
        { type: 'done', tokens: 5 }
      ]);

      vi.mocked(testAgent).mockResolvedValue(mockStream);

      render(AgentSandbox, {
        props: { agentId: 'agent1' }
      });

      const textarea = screen.getByLabelText('Test Message');
      await fireEvent.input(textarea, { target: { value: 'Test message' } });

      const testButton = screen.getByText('Test');
      await fireEvent.click(testButton);

      await waitFor(() => {
        expect(screen.getByText(/Test History/)).toBeTruthy();
      }, { timeout: 3000 });
    });

    it('should toggle history visibility', async () => {
      const mockStream = createMockSSEStream([
        { type: 'content', data: 'Response' },
        { type: 'done' }
      ]);

      vi.mocked(testAgent).mockResolvedValue(mockStream);

      render(AgentSandbox, {
        props: { agentId: 'agent1' }
      });

      const textarea = screen.getByLabelText('Test Message');
      await fireEvent.input(textarea, { target: { value: 'Test' } });

      const testButton = screen.getByText('Test');
      await fireEvent.click(testButton);

      await waitFor(() => {
        const historyButton = screen.getByText(/Test History/);
        expect(historyButton).toBeTruthy();
      }, { timeout: 3000 });

      const historyButton = screen.getByText(/Test History/);
      await fireEvent.click(historyButton);

      await waitFor(() => {
        // Look for the test message text in the history item (more specific than just "Test" button)
        const historyItems = screen.getAllByText('Test');
        // Should find at least 2: the button and the history item message
        expect(historyItems.length).toBeGreaterThan(1);
      });
    });

    it('should limit history to 5 items', async () => {
      const mockStream = createMockSSEStream([
        { type: 'content', data: 'Response' },
        { type: 'done' }
      ]);

      vi.mocked(testAgent).mockResolvedValue(mockStream);

      render(AgentSandbox, {
        props: { agentId: 'agent1' }
      });

      // Run 6 tests
      for (let i = 1; i <= 6; i++) {
        const textarea = screen.getByLabelText('Test Message');
        await fireEvent.input(textarea, { target: { value: `Test ${i}` } });

        const testButton = screen.getByText('Test');
        await fireEvent.click(testButton);

        await waitFor(() => {
          expect(screen.getByText(/Test History/)).toBeTruthy();
        }, { timeout: 3000 });
      }

      // History should show "(5)" not "(6)"
      await waitFor(() => {
        expect(screen.getByText(/Test History \(5\)/)).toBeTruthy();
      });
    });
  });

  describe('Callbacks', () => {
    it('should call onTest callback with result', async () => {
      const onTest = vi.fn();
      const mockStream = createMockSSEStream([
        { type: 'content', data: 'Response text' },
        { type: 'done', tokens: 10, model: 'gpt-4' }
      ]);

      vi.mocked(testAgent).mockResolvedValue(mockStream);

      render(AgentSandbox, {
        props: { agentId: 'agent1', onTest }
      });

      const textarea = screen.getByLabelText('Test Message');
      await fireEvent.input(textarea, { target: { value: 'Test' } });

      const testButton = screen.getByText('Test');
      await fireEvent.click(testButton);

      await waitFor(() => {
        expect(onTest).toHaveBeenCalledWith(
          expect.objectContaining({
            response: 'Response text',
            tokens: 10,
            model: 'gpt-4'
          })
        );
      }, { timeout: 3000 });
    });
  });

  describe('Utility Functions', () => {
    it('should format duration correctly', async () => {
      const mockStream = createMockSSEStream([
        { type: 'content', data: 'Test' },
        { type: 'done' }
      ]);

      vi.mocked(testAgent).mockResolvedValue(mockStream);

      render(AgentSandbox, {
        props: { agentId: 'agent1' }
      });

      const textarea = screen.getByLabelText('Test Message');
      await fireEvent.input(textarea, { target: { value: 'Test' } });

      const testButton = screen.getByText('Test');
      await fireEvent.click(testButton);

      // Should eventually show duration in ms or seconds
      await waitFor(() => {
        const text = document.body.textContent || '';
        expect(text.match(/\d+ms|\d+\.\d+s/)).toBeTruthy();
      }, { timeout: 3000 });
    });

    it('should format token count with locale', async () => {
      const mockStream = createMockSSEStream([
        { type: 'content', data: 'Test' },
        { type: 'done', tokens: 1234 }
      ]);

      vi.mocked(testAgent).mockResolvedValue(mockStream);

      render(AgentSandbox, {
        props: { agentId: 'agent1' }
      });

      const textarea = screen.getByLabelText('Test Message');
      await fireEvent.input(textarea, { target: { value: 'Test' } });

      const testButton = screen.getByText('Test');
      await fireEvent.click(testButton);

      await waitFor(() => {
        // Should format with locale separator: "1,234" (US) or "1.234" (EU)
        // Just check that the number is present with some separator
        expect(screen.getByText(/1[.,]234/)).toBeTruthy();
      }, { timeout: 3000 });
    });
  });
});
