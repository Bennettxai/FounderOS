<script lang="ts">
  interface Props {
    value: string;
    onChange: (value: string) => void;
    placeholder?: string;
    rows?: number;
    maxLength?: number;
    id?: string;
    name?: string;
  }

  let {
    value,
    onChange,
    placeholder = 'Enter system prompt...',
    rows = 12,
    maxLength = 4000,
    id = 'system-prompt',
    name = 'systemPrompt'
  }: Props = $props();

  let textareaRef: HTMLTextAreaElement | null = $state(null);
  let showTemplates = $state(false);
  let showVariables = $state(false);
  let showTips = $state(false);

  // Template snippets
  const templates = [
    {
      label: 'Role Definition',
      content: 'You are a {role} that specializes in {domain}. Your primary responsibility is to {responsibility}.'
    },
    {
      label: 'Capabilities',
      content: 'You can:\n1. {capability_1}\n2. {capability_2}\n3. {capability_3}'
    },
    {
      label: 'Limitations',
      content: 'You cannot:\n- {limitation_1}\n- {limitation_2}\n- {limitation_3}'
    },
    {
      label: 'Tone & Style',
      content: 'Use a {tone} tone. Be {style_1} and {style_2}. Keep responses {length}.'
    },
    {
      label: 'Example Interaction',
      content: 'Example:\nUser: {user_input}\nAssistant: {assistant_response}'
    }
  ];

  // Available variables
  const variables = [
    { name: '{{user_name}}', description: 'Current user name' },
    { name: '{{workspace_name}}', description: 'Active workspace name' },
    { name: '{{date}}', description: 'Current date' },
    { name: '{{time}}', description: 'Current time' }
  ];

  // Best practices tips
  const tips = [
    'Be specific about the agent\'s role and expertise',
    'Define clear boundaries and limitations',
    'Provide concrete examples of desired behavior',
    'Use consistent formatting and structure',
    'Keep prompts concise but comprehensive',
    'Test with various input scenarios'
  ];

  function handleInput(e: Event) {
    const target = e.target as HTMLTextAreaElement;
    onChange(target.value);
  }

  function insertAtCursor(text: string) {
    if (!textareaRef) return;

    const start = textareaRef.selectionStart;
    const end = textareaRef.selectionEnd;
    const before = value.substring(0, start);
    const after = value.substring(end);
    const newValue = before + text + after;

    onChange(newValue);

    // Set cursor position after inserted text
    setTimeout(() => {
      if (textareaRef) {
        const newPosition = start + text.length;
        textareaRef.focus();
        textareaRef.setSelectionRange(newPosition, newPosition);
      }
    }, 0);
  }

  function insertTemplate(template: string) {
    insertAtCursor(template);
    showTemplates = false;
  }

  function insertVariable(variable: string) {
    insertAtCursor(variable);
    showVariables = false;
  }

  // Character count with warning
  const charCount = $derived(value.length);
  const isNearLimit = $derived(charCount > maxLength * 0.9);
  const isOverLimit = $derived(charCount > maxLength);
</script>

<div class="space-y-3">
  <!-- Header with controls -->
  <div class="flex items-center justify-between">
    <label for={id} class="block text-sm font-medium text-gray-700 dark:text-gray-300">System Prompt</label>
    <div class="flex gap-2">
      <!-- Templates dropdown -->
      <div class="relative">
        <button
          type="button"
          onclick={() => showTemplates = !showTemplates}
          class="text-xs px-3 py-1.5 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700 bg-white dark:bg-gray-800 rounded-md transition-colors"
          title="Insert template"
        >
          Templates
        </button>
        {#if showTemplates}
          <div class="absolute right-0 mt-1 w-56 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-md shadow-lg z-10">
            <div class="py-1">
              {#each templates as template}
                <button
                  type="button"
                  onclick={() => insertTemplate(template.content)}
                  class="w-full text-left px-3 py-2 text-xs text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
                >
                  {template.label}
                </button>
              {/each}
            </div>
          </div>
        {/if}
      </div>

      <!-- Variables dropdown -->
      <div class="relative">
        <button
          type="button"
          onclick={() => showVariables = !showVariables}
          class="text-xs px-3 py-1.5 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700 bg-white dark:bg-gray-800 rounded-md transition-colors"
          title="Insert variable"
        >
          Variables
        </button>
        {#if showVariables}
          <div class="absolute right-0 mt-1 w-64 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-md shadow-lg z-10">
            <div class="py-1">
              {#each variables as variable}
                <button
                  type="button"
                  onclick={() => insertVariable(variable.name)}
                  class="w-full text-left px-3 py-2 text-xs text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
                >
                  <div class="font-mono font-medium">{variable.name}</div>
                  <div class="text-gray-500 text-[10px]">{variable.description}</div>
                </button>
              {/each}
            </div>
            <div class="border-t border-gray-200 dark:border-gray-600 px-3 py-2 bg-gray-50 dark:bg-gray-700">
              <p class="text-[10px] text-gray-600 dark:text-gray-400">
                Variables are replaced at runtime with actual values
              </p>
            </div>
          </div>
        {/if}
      </div>

      <!-- Tips toggle -->
      <button
        type="button"
        onclick={() => showTips = !showTips}
        class="text-xs px-3 py-1.5 border border-gray-300 rounded-md hover:bg-gray-50 transition-colors"
        class:bg-blue-50={showTips}
        class:border-blue-300={showTips}
        title="Toggle best practices"
      >
        Tips
      </button>
    </div>
  </div>

  <div class="flex gap-3">
    <!-- Main editor -->
    <div class="flex-1 space-y-2">
      <div class="relative">
        <textarea
          bind:this={textareaRef}
          {id}
          {name}
          {value}
          oninput={handleInput}
          {placeholder}
          {rows}
          maxlength={maxLength}
          class="w-full border border-gray-300 dark:border-gray-600 rounded-md px-3 py-2 font-mono text-sm
                 bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100
                 focus:ring-2 focus:ring-blue-500 focus:border-transparent
                 resize-y transition-shadow"
          class:border-red-300={isOverLimit}
          class:border-yellow-300={isNearLimit && !isOverLimit}
        ></textarea>

        <!-- Character counter overlay (bottom-right corner) -->
        <div class="absolute bottom-2 right-2 text-xs font-mono bg-white/80 dark:bg-gray-800/80 px-2 py-1 rounded border border-gray-200 dark:border-gray-600"
             class:text-red-600={isOverLimit}
             class:text-yellow-600={isNearLimit && !isOverLimit}
             class:text-gray-500={!isNearLimit}>
          {charCount} / {maxLength}
        </div>
      </div>

      <!-- Stats row -->
      <div class="flex items-center justify-between text-xs text-gray-500">
        <div class="flex gap-4">
          <span>{charCount} characters</span>
          <span>~{Math.ceil(charCount / 4)} tokens (approx)</span>
          <span>{value.split('\n').length} lines</span>
        </div>
        {#if isOverLimit}
          <span class="text-red-600 font-medium">Exceeds recommended length</span>
        {:else if isNearLimit}
          <span class="text-yellow-600 font-medium">Approaching limit</span>
        {/if}
      </div>
    </div>

    <!-- Best practices sidebar -->
    {#if showTips}
      <div class="w-64 bg-blue-50 border border-blue-200 rounded-md p-3 space-y-2">
        <h4 class="text-xs font-semibold text-blue-900">Prompt Engineering Tips</h4>
        <ul class="space-y-2">
          {#each tips as tip}
            <li class="text-xs text-blue-800 flex gap-2">
              <span class="text-blue-500">•</span>
              <span>{tip}</span>
            </li>
          {/each}
        </ul>
      </div>
    {/if}
  </div>

  <!-- Variable syntax help -->
  <div class="bg-gray-50 dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-md p-2">
    <p class="text-xs text-gray-600 dark:text-gray-400">
      <strong class="font-medium">Variables:</strong>
      Use <code class="bg-white px-1 rounded font-mono">{'{{variable_name}}'}</code> to insert dynamic values.
      Click <strong>Variables</strong> button to see available options.
    </p>
  </div>
</div>
