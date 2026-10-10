<script lang="ts">
  import type { ReasoningTemplate, CreateTemplateData, UpdateTemplateData, ReasoningStep } from '$lib/api/thinking/types';

  interface Props {
    template: ReasoningTemplate | null;
    onSave: (data: CreateTemplateData | UpdateTemplateData) => void;
    onCancel: () => void;
  }

  let { template, onSave, onCancel }: Props = $props();

  // Form state
  let name = $state(template?.name || '');
  let description = $state(template?.description || '');
  let steps = $state<ReasoningStep[]>(
    template?.steps || []
  );

  // UI state
  let errors = $state<Record<string, string>>({});
  let isSubmitting = $state(false);

  // Constants
  const stepTypes = [
    { value: 'exploration', label: 'Exploration', description: 'Explore and understand the problem' },
    { value: 'analysis', label: 'Analysis', description: 'Analyze information and patterns' },
    { value: 'conclusion', label: 'Conclusion', description: 'Draw conclusions and synthesize' },
    { value: 'reflection', label: 'Reflection', description: 'Reflect on the process and results' }
  ] as const;

  // Validation
  function validateForm(): boolean {
    const newErrors: Record<string, string> = {};

    // Name required
    if (!name.trim()) {
      newErrors.name = 'Template name is required';
    } else if (name.length > 100) {
      newErrors.name = 'Name must be 100 characters or less';
    }

    // Description optional but limited
    if (description && description.length > 500) {
      newErrors.description = 'Description must be 500 characters or less';
    }

    // At least 1 step required
    if (steps.length === 0) {
      newErrors.steps = 'At least one step is required';
    }

    // Validate each step
    steps.forEach((step, index) => {
      if (!step.type) {
        newErrors[`step-${index}-type`] = 'Step type is required';
      }
      if (!step.prompt.trim()) {
        newErrors[`step-${index}-prompt`] = 'Step prompt is required';
      } else if (step.prompt.length > 2000) {
        newErrors[`step-${index}-prompt`] = 'Step prompt must be 2000 characters or less';
      }
    });

    errors = newErrors;
    return Object.keys(newErrors).length === 0;
  }

  // Step management
  function addStep() {
    steps = [
      ...steps,
      {
        order: steps.length,
        type: 'exploration',
        prompt: ''
      }
    ];
  }

  function removeStep(index: number) {
    steps = steps.filter((_, i) => i !== index).map((step, i) => ({
      ...step,
      order: i
    }));
  }

  function moveStepUp(index: number) {
    if (index === 0) return;
    const newSteps = [...steps];
    [newSteps[index - 1], newSteps[index]] = [newSteps[index], newSteps[index - 1]];
    steps = newSteps.map((step, i) => ({ ...step, order: i }));
  }

  function moveStepDown(index: number) {
    if (index === steps.length - 1) return;
    const newSteps = [...steps];
    [newSteps[index], newSteps[index + 1]] = [newSteps[index + 1], newSteps[index]];
    steps = newSteps.map((step, i) => ({ ...step, order: i }));
  }

  // Calculate estimated token usage
  const estimatedTokens = $derived(() => {
    const nameTokens = Math.ceil(name.length / 4);
    const descTokens = Math.ceil(description.length / 4);
    const stepTokens = steps.reduce((sum, step) => sum + Math.ceil(step.prompt.length / 4), 0);
    return nameTokens + descTokens + stepTokens;
  });

  // Form submission
  function handleSubmit(e: Event) {
    e.preventDefault();

    if (!validateForm()) {
      // Scroll to first error
      const firstError = Object.keys(errors)[0];
      const element = document.querySelector(`[name="${firstError}"]`);
      element?.scrollIntoView({ behavior: 'smooth', block: 'center' });
      return;
    }

    isSubmitting = true;

    const data = {
      name: name.trim(),
      description: description.trim() || undefined,
      steps: steps.map((step, index) => ({
        order: index,
        type: step.type,
        prompt: step.prompt.trim()
      }))
    };

    onSave(data as CreateTemplateData | UpdateTemplateData);
  }

  // Handle cancel
  function handleCancel() {
    onCancel();
  }
</script>

<form onsubmit={handleSubmit} class="bg-white rounded-lg border shadow-sm">
  <!-- Header -->
  <div class="px-6 py-4 border-b">
    <h2 class="text-xl font-semibold text-gray-900">
      {template ? 'Edit Template' : 'Create New Template'}
    </h2>
    <p class="text-sm text-gray-600 mt-1">
      Define a structured reasoning process with multiple steps
    </p>
  </div>

  <!-- Form Content -->
  <div class="px-6 py-6 space-y-6">
    <!-- Name -->
    <div>
      <label for="name" class="block text-sm font-medium text-gray-700 mb-2">
        Template Name <span class="text-red-500">*</span>
      </label>
      <input
        type="text"
        id="name"
        name="name"
        bind:value={name}
        class="w-full px-3 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 {errors.name ? 'border-red-500' : 'border-gray-300'}"
        placeholder="e.g., Problem Analysis Framework"
        maxlength="100"
      />
      {#if errors.name}
        <p class="text-sm text-red-600 mt-1">{errors.name}</p>
      {/if}
      <p class="text-xs text-gray-500 mt-1">{name.length}/100 characters</p>
    </div>

    <!-- Description -->
    <div>
      <label for="description" class="block text-sm font-medium text-gray-700 mb-2">
        Description
      </label>
      <textarea
        id="description"
        name="description"
        bind:value={description}
        rows="3"
        class="w-full px-3 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 {errors.description ? 'border-red-500' : 'border-gray-300'}"
        placeholder="Describe when and how to use this template..."
        maxlength="500"
      ></textarea>
      {#if errors.description}
        <p class="text-sm text-red-600 mt-1">{errors.description}</p>
      {/if}
      <p class="text-xs text-gray-500 mt-1">{description.length}/500 characters</p>
    </div>

    <!-- Steps Section -->
    <div>
      <div class="flex items-center justify-between mb-3">
        <div>
          <h3 class="text-sm font-medium text-gray-900">
            Reasoning Steps <span class="text-red-500">*</span>
          </h3>
          <p class="text-xs text-gray-500 mt-0.5">
            {steps.length} step{steps.length !== 1 ? 's' : ''} defined
          </p>
        </div>
        <button
          type="button"
          onclick={addStep}
          class="btn-pill btn-pill-ghost btn-pill-sm"
        >
          + Add Step
        </button>
      </div>

      {#if errors.steps}
        <p class="text-sm text-red-600 mb-3">{errors.steps}</p>
      {/if}

      <!-- Steps List -->
      <div class="space-y-4">
        {#each steps as step, index (index)}
          <div class="border rounded-lg p-4 bg-gray-50">
            <div class="flex items-start justify-between mb-3">
              <div class="flex items-center space-x-2">
                <span class="flex items-center justify-center w-8 h-8 bg-blue-100 text-blue-700 font-semibold rounded-full text-sm">
                  {index + 1}
                </span>
                <span class="text-sm font-medium text-gray-700">
                  Step {index + 1}
                </span>
              </div>

              <!-- Step Controls -->
              <div class="flex items-center space-x-1">
                <button
                  type="button"
                  onclick={() => moveStepUp(index)}
                  disabled={index === 0}
                  class="p-1 text-gray-600 hover:text-gray-900 hover:bg-gray-200 rounded disabled:opacity-30 disabled:cursor-not-allowed"
                  title="Move up"
                >
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 15l7-7 7 7" />
                  </svg>
                </button>
                <button
                  type="button"
                  onclick={() => moveStepDown(index)}
                  disabled={index === steps.length - 1}
                  class="p-1 text-gray-600 hover:text-gray-900 hover:bg-gray-200 rounded disabled:opacity-30 disabled:cursor-not-allowed"
                  title="Move down"
                >
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
                  </svg>
                </button>
                <button
                  type="button"
                  onclick={() => removeStep(index)}
                  class="p-1 text-red-600 hover:text-red-700 hover:bg-red-50 rounded"
                  title="Delete step"
                >
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                  </svg>
                </button>
              </div>
            </div>

            <!-- Step Type -->
            <div class="mb-3">
              <label for="step-{index}-type" class="block text-xs font-medium text-gray-700 mb-1">
                Type
              </label>
              <select
                id="step-{index}-type"
                name="step-{index}-type"
                bind:value={step.type}
                class="w-full px-3 py-1.5 text-sm border rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 {errors[`step-${index}-type`] ? 'border-red-500' : 'border-gray-300'}"
              >
                {#each stepTypes as type}
                  <option value={type.value}>{type.label}</option>
                {/each}
              </select>
              {#if errors[`step-${index}-type`]}
                <p class="text-xs text-red-600 mt-1">{errors[`step-${index}-type`]}</p>
              {/if}
              <p class="text-xs text-gray-500 mt-1">
                {stepTypes.find(t => t.value === step.type)?.description}
              </p>
            </div>

            <!-- Step Prompt -->
            <div>
              <label for="step-{index}-prompt" class="block text-xs font-medium text-gray-700 mb-1">
                Prompt
              </label>
              <textarea
                id="step-{index}-prompt"
                name="step-{index}-prompt"
                bind:value={step.prompt}
                rows="4"
                class="w-full px-3 py-2 text-sm border rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 {errors[`step-${index}-prompt`] ? 'border-red-500' : 'border-gray-300'}"
                placeholder="Enter the instructions for this step..."
                maxlength="2000"
              ></textarea>
              {#if errors[`step-${index}-prompt`]}
                <p class="text-xs text-red-600 mt-1">{errors[`step-${index}-prompt`]}</p>
              {/if}
              <p class="text-xs text-gray-500 mt-1">{step.prompt.length}/2000 characters</p>
            </div>
          </div>
        {/each}

        {#if steps.length === 0}
          <div class="text-center py-8 border-2 border-dashed border-gray-300 rounded-lg">
            <svg class="w-12 h-12 mx-auto text-gray-400 mb-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
            </svg>
            <p class="text-sm text-gray-600 mb-3">No steps defined yet</p>
            <button
              type="button"
              onclick={addStep}
              class="btn-pill btn-pill-primary btn-pill-sm"
            >
              Add First Step
            </button>
          </div>
        {/if}
      </div>
    </div>

    <!-- Preview Section -->
    {#if steps.length > 0}
      <div class="border-t pt-6">
        <h3 class="text-sm font-medium text-gray-900 mb-3">Preview</h3>
        <div class="bg-gray-50 rounded-lg p-4 space-y-3">
          <div class="flex items-center justify-between text-sm">
            <span class="text-gray-600">Template Structure</span>
            <span class="font-medium text-gray-900">{name || 'Untitled Template'}</span>
          </div>
          <div class="flex items-center justify-between text-sm">
            <span class="text-gray-600">Total Steps</span>
            <span class="font-medium text-gray-900">{steps.length}</span>
          </div>
          <div class="flex items-center justify-between text-sm">
            <span class="text-gray-600">Estimated Tokens</span>
            <span class="font-medium text-gray-900">~{estimatedTokens()}</span>
          </div>
          <div class="pt-2 border-t">
            <p class="text-xs text-gray-600 mb-2">Step Types:</p>
            <div class="flex flex-wrap gap-2">
              {#each Array.from(new Set(steps.map(s => s.type))) as type}
                <span class="px-2 py-1 text-xs font-medium bg-blue-100 text-blue-700 rounded">
                  {stepTypes.find(t => t.value === type)?.label}
                </span>
              {/each}
            </div>
          </div>
        </div>
      </div>
    {/if}
  </div>

  <!-- Actions -->
  <div class="px-6 py-4 border-t bg-gray-50 flex items-center justify-end space-x-3">
    <button
      type="button"
      onclick={handleCancel}
      disabled={isSubmitting}
      class="btn-pill btn-pill-ghost btn-pill-sm"
    >
      Cancel
    </button>
    <button
      type="submit"
      disabled={isSubmitting || steps.length === 0 || !name.trim()}
      class="btn-pill btn-pill-primary btn-pill-sm"
    >
      {#if isSubmitting}
        <span class="flex items-center space-x-2">
          <svg class="animate-spin w-4 h-4" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
          </svg>
          <span>Saving...</span>
        </span>
      {:else}
        {template ? 'Update Template' : 'Create Template'}
      {/if}
    </button>
  </div>
</form>

<style>
  /* Smooth transitions */
  button {
    transition: all 150ms ease-in-out;
  }

  input:focus,
  textarea:focus,
  select:focus {
    outline: none;
  }
</style>
