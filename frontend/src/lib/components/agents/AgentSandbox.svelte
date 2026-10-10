<script lang="ts">
  import { testAgent, testSandbox } from '$lib/api/ai';
  import type { SandboxTestRequest } from '$lib/api/ai/types';
  import { onDestroy } from 'svelte';

  interface Props {
    agentId?: string;
    systemPrompt?: string;
    onTest?: (result: TestResult) => void;
  }

  interface TestResult {
    response: string;
    tokens?: number;
    duration?: number;
    model?: string;
  }

  interface TestHistory {
    id: string;
    timestamp: number;
    message: string;
    response: string;
    tokens?: number;
    duration?: number;
    model?: string;
    error?: boolean;
  }

  let { agentId, systemPrompt, onTest }: Props = $props();

  // Test input state
  let testMessage = $state('');
  let modelOverride = $state('');
  let temperatureOverride = $state<number | undefined>(undefined);
  let showAdvanced = $state(false);

  // Response state
  let streamingResponse = $state('');
  let isLoading = $state(false);
  let error = $state<string | null>(null);
  let currentTokens = $state<number | undefined>(undefined);
  let currentDuration = $state<number | undefined>(undefined);
  let currentModel = $state<string | undefined>(undefined);

  // History state
  let history = $state<TestHistory[]>([]);
  let showHistory = $state(false);
  let historyExpanded = $state<Record<string, boolean>>({});

  // Streaming control
  let abortController: AbortController | null = null;
  let startTime = 0;

  async function handleTest() {
    if (!testMessage.trim()) return;

    isLoading = true;
    error = null;
    streamingResponse = '';
    currentTokens = undefined;
    currentDuration = undefined;
    currentModel = undefined;
    startTime = Date.now();

    abortController = new AbortController();

    try {
      let stream: ReadableStream<Uint8Array> | null = null;

      if (agentId) {
        // Test existing agent
        stream = await testAgent(agentId, testMessage);
      } else if (systemPrompt) {
        // Test sandbox configuration
        const config: SandboxTestRequest = {
          system_prompt: systemPrompt,
          test_message: testMessage,
          model: modelOverride || undefined,
          temperature: temperatureOverride
        };
        stream = await testSandbox(config);
      } else {
        throw new Error('Either agentId or systemPrompt must be provided');
      }

      if (!stream) {
        throw new Error('No stream returned from API');
      }

      // Read the stream
      const reader = stream.getReader();
      const decoder = new TextDecoder();
      let fullResponse = '';
      let sseBuffer = '';

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        const chunk = decoder.decode(value, { stream: true });
        sseBuffer += chunk;

        // Process SSE events
        const lines = sseBuffer.split('\n');
        sseBuffer = lines.pop() || ''; // Keep incomplete line in buffer

        for (const line of lines) {
          if (line.startsWith('data: ')) {
            try {
              const data = JSON.parse(line.slice(6));

              // Handle different event types
              if (data.type === 'content') {
                fullResponse += data.data;
                streamingResponse = fullResponse;
              } else if (data.type === 'metadata') {
                // Handle metadata (tokens, model, etc.)
                if (data.tokens !== undefined) currentTokens = data.tokens;
                if (data.model) currentModel = data.model;
              } else if (data.type === 'done') {
                // Final metadata
                if (data.tokens !== undefined) currentTokens = data.tokens;
                if (data.model) currentModel = data.model;
                currentDuration = data.duration ?? Date.now() - startTime;
              } else if (data.type === 'error') {
                throw new Error(data.message || 'Stream error');
              }
            } catch (e) {
              console.error('Error parsing SSE line:', e, line);
            }
          }
        }
      }

      // Calculate final duration
      const duration = Date.now() - startTime;
      currentDuration = duration;

      // Add to history
      const historyItem: TestHistory = {
        id: Date.now().toString(),
        timestamp: Date.now(),
        message: testMessage,
        response: fullResponse,
        tokens: currentTokens,
        duration: duration,
        model: currentModel,
        error: false
      };

      history = [historyItem, ...history].slice(0, 5); // Keep last 5

      // Call onTest callback
      if (onTest) {
        onTest({
          response: fullResponse,
          tokens: currentTokens,
          duration: duration,
          model: currentModel
        });
      }

    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      error = errorMessage;

      // Add error to history
      const historyItem: TestHistory = {
        id: Date.now().toString(),
        timestamp: Date.now(),
        message: testMessage,
        response: errorMessage,
        error: true
      };

      history = [historyItem, ...history].slice(0, 5);

    } finally {
      isLoading = false;
      abortController = null;
    }
  }

  function handleClear() {
    testMessage = '';
    streamingResponse = '';
    error = null;
    currentTokens = undefined;
    currentDuration = undefined;
    currentModel = undefined;
  }

  function handleStop() {
    if (abortController) {
      abortController.abort();
      abortController = null;
      isLoading = false;
    }
  }

  function toggleHistory(id: string) {
    historyExpanded[id] = !historyExpanded[id];
  }

  function formatDuration(ms?: number): string {
    if (ms === undefined) return 'N/A';
    return ms < 1000 ? `${ms}ms` : `${(ms / 1000).toFixed(2)}s`;
  }

  function formatTokens(tokens?: number): string {
    if (tokens === undefined) return 'N/A';
    return tokens.toLocaleString();
  }

  onDestroy(() => {
    if (abortController) {
      abortController.abort();
    }
  });
</script>

<div class="border border-gray-200 dark:border-gray-700 rounded-lg bg-white dark:bg-gray-900 shadow-sm overflow-hidden">
  <div class="p-4 border-b border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800">
    <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100">Agent Sandbox</h3>
    <p class="text-sm text-gray-600 dark:text-gray-400 mt-1">Test your agent configuration in real-time</p>
  </div>

  <div class="grid grid-cols-1 lg:grid-cols-2 gap-4 p-4">
    <!-- Left: Test Input -->
    <div class="space-y-4">
      <div>
        <label for="test-message" class="block text-sm font-medium text-gray-700 mb-2">
          Test Message
        </label>
        <textarea
          id="test-message"
          bind:value={testMessage}
          disabled={isLoading}
          class="w-full border border-gray-300 dark:border-gray-600 rounded-lg px-3 py-2 bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent disabled:bg-gray-100 dark:disabled:bg-gray-700 disabled:cursor-not-allowed"
          rows="6"
          placeholder="Enter a test message for the agent..."
        ></textarea>
      </div>

      <!-- Advanced Options -->
      <div>
        <button
          type="button"
          onclick={() => showAdvanced = !showAdvanced}
          class="text-sm text-gray-600 hover:text-gray-900 flex items-center gap-1"
        >
          <svg class="w-4 h-4 transition-transform {showAdvanced ? 'rotate-90' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
          </svg>
          Advanced Options
        </button>

        {#if showAdvanced}
          <div class="mt-3 space-y-3 pl-5">
            <div>
              <label for="model-override" class="block text-xs font-medium text-gray-600 mb-1">
                Model Override (optional)
              </label>
              <input
                id="model-override"
                type="text"
                bind:value={modelOverride}
                placeholder="e.g., gpt-4o, claude-sonnet-4"
                class="w-full border border-gray-300 rounded px-2 py-1.5 text-sm"
              />
            </div>

            <div>
              <label for="temperature-override" class="block text-xs font-medium text-gray-600 mb-1">
                Temperature: {temperatureOverride ?? 'default'}
              </label>
              <input
                id="temperature-override"
                type="range"
                min="0"
                max="2"
                step="0.1"
                bind:value={temperatureOverride}
                class="w-full"
              />
              <div class="flex justify-between text-xs text-gray-500 mt-1">
                <span>Focused (0)</span>
                <span>Balanced (1)</span>
                <span>Creative (2)</span>
              </div>
            </div>
          </div>
        {/if}
      </div>

      <!-- Action Buttons -->
      <div class="flex gap-2">
        {#if isLoading}
          <button
            type="button"
            onclick={handleStop}
            class="btn-pill btn-pill-danger"
          >
            Stop
          </button>
        {:else}
          <button
            type="button"
            onclick={handleTest}
            disabled={!testMessage.trim()}
            class="btn-pill btn-pill-primary disabled:bg-gray-300"
          >
            Test
          </button>
        {/if}

        <button
          type="button"
          onclick={handleClear}
          class="btn-pill btn-pill-ghost"
        >
          Clear
        </button>
      </div>
    </div>

    <!-- Right: Response Display -->
    <div class="space-y-3">
      <div class="flex items-center justify-between">
        <span class="block text-sm font-medium text-gray-700">
          Response
        </span>

        {#if isLoading || streamingResponse}
          <div class="flex items-center gap-3 text-xs text-gray-600">
            {#if currentModel}
              <span class="flex items-center gap-1">
                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
                </svg>
                {currentModel}
              </span>
            {/if}
            {#if currentTokens !== undefined}
              <span class="flex items-center gap-1">
                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 20l4-16m2 16l4-16M6 9h14M4 15h14" />
                </svg>
                {formatTokens(currentTokens)}
              </span>
            {/if}
            {#if currentDuration !== undefined}
              <span class="flex items-center gap-1">
                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                {formatDuration(currentDuration)}
              </span>
            {/if}
          </div>
        {/if}
      </div>

      <div class="border border-gray-300 rounded-lg bg-gray-50 p-4 min-h-[300px] max-h-[400px] overflow-y-auto">
        {#if error}
          <div class="bg-red-50 border border-red-200 rounded-lg p-3 text-red-800 text-sm">
            <div class="flex items-start gap-2">
              <svg class="w-5 h-5 flex-shrink-0 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
                <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd" />
              </svg>
              <div>
                <p class="font-medium">Error</p>
                <p class="mt-1">{error}</p>
              </div>
            </div>
          </div>
        {:else if isLoading}
          <div class="flex items-start gap-3">
            <div class="flex-shrink-0">
              <div class="w-5 h-5 border-2 border-blue-600 border-t-transparent rounded-full animate-spin"></div>
            </div>
            <div class="flex-1 text-gray-900 text-sm whitespace-pre-wrap break-words">
              {streamingResponse || 'Waiting for response...'}
              <span class="inline-block w-2 h-4 bg-blue-600 animate-pulse ml-1"></span>
            </div>
          </div>
        {:else if streamingResponse}
          <div class="text-gray-900 text-sm whitespace-pre-wrap break-words">
            {streamingResponse}
          </div>
        {:else}
          <div class="text-gray-400 text-sm italic text-center mt-8">
            No response yet. Send a test message to see the agent's response.
          </div>
        {/if}
      </div>
    </div>
  </div>

  <!-- History Section -->
  {#if history.length > 0}
    <div class="border-t">
      <button
        type="button"
        onclick={() => showHistory = !showHistory}
        class="w-full px-4 py-3 flex items-center justify-between bg-gray-50 hover:bg-gray-100 transition-colors"
      >
        <span class="text-sm font-medium text-gray-700">
          Test History ({history.length})
        </span>
        <svg class="w-4 h-4 text-gray-500 transition-transform {showHistory ? 'rotate-180' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
        </svg>
      </button>

      {#if showHistory}
        <div class="p-4 space-y-2 max-h-64 overflow-y-auto bg-gray-50">
          {#each history as item (item.id)}
            <div class="border border-gray-200 rounded-lg bg-white overflow-hidden">
              <button
                type="button"
                onclick={() => toggleHistory(item.id)}
                class="w-full px-3 py-2 flex items-center justify-between hover:bg-gray-50 transition-colors"
              >
                <div class="flex items-center gap-2 flex-1 min-w-0">
                  {#if item.error}
                    <svg class="w-4 h-4 text-red-500 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                      <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd" />
                    </svg>
                  {:else}
                    <svg class="w-4 h-4 text-green-500 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                      <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" />
                    </svg>
                  {/if}
                  <span class="text-sm text-gray-700 truncate">{item.message}</span>
                  <span class="text-xs text-gray-500 flex-shrink-0">
                    {new Date(item.timestamp).toLocaleTimeString()}
                  </span>
                </div>
                <svg class="w-4 h-4 text-gray-400 flex-shrink-0 transition-transform {historyExpanded[item.id] ? 'rotate-180' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
                </svg>
              </button>

              {#if historyExpanded[item.id]}
                <div class="px-3 pb-3 pt-1 border-t border-gray-100">
                  <div class="space-y-2 text-xs">
                    <div>
                      <p class="text-gray-500 font-medium mb-1">Message:</p>
                      <p class="text-gray-700 whitespace-pre-wrap">{item.message}</p>
                    </div>
                    <div>
                      <p class="text-gray-500 font-medium mb-1">Response:</p>
                      <p class="text-gray-700 whitespace-pre-wrap {item.error ? 'text-red-600' : ''}">{item.response}</p>
                    </div>
                    {#if !item.error}
                      <div class="flex gap-3 text-gray-600 pt-1">
                        {#if item.model}
                          <span>Model: {item.model}</span>
                        {/if}
                        {#if item.tokens !== undefined}
                          <span>Tokens: {formatTokens(item.tokens)}</span>
                        {/if}
                        {#if item.duration !== undefined}
                          <span>Duration: {formatDuration(item.duration)}</span>
                        {/if}
                      </div>
                    {/if}
                  </div>
                </div>
              {/if}
            </div>
          {/each}
        </div>
      {/if}
    </div>
  {/if}
</div>
