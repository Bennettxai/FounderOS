<script lang="ts">
  import type { CustomAgent } from '$lib/api/ai/types';

  interface Props {
    agent?: CustomAgent;
    onSave: (agent: Partial<CustomAgent>) => void;
    onCancel: () => void;
  }

  let { agent, onSave, onCancel }: Props = $props();

  // ── Step ──
  let step = $state(1);

  // ── Step 1: Identity ──
  let displayName = $state(agent?.display_name ?? '');
  let handle     = $state(agent?.name ?? '');
  let handleEdited = $state(!!agent?.name);
  let description = $state(agent?.description ?? '');
  let category   = $state(agent?.category ?? 'general');
  let avatarColor = $state('#111111');

  // ── Step 2: Behavior ──
  let modelPref    = $state(agent?.model_preference ?? 'claude-sonnet-4-6');
  let temperature  = $state(agent?.temperature ?? 0.7);
  let systemPrompt = $state(agent?.system_prompt ?? '');
  let welcomeMsg   = $state(agent?.welcome_message ?? '');
  let suggestedPrompts = $state<string[]>(agent?.suggested_prompts ?? []);
  let newPrompt    = $state('');
  let isActive     = $state(agent?.is_active ?? true);

  // ── UI state ──
  let errors      = $state<Record<string, string>>({});
  let submitted   = $state(false);
  let isSubmitting = $state(false);

  // ── Constants ──
  const avatarColors = [
    '#111111','#1d4ed8','#7c3aed','#db2777','#dc2626',
    '#d97706','#059669','#0891b2','#64748b','#9333ea'
  ];

  const categoryOptions = [
    { id: 'general',   label: 'General',   icon: 'general' },
    { id: 'coding',    label: 'Coding',    icon: 'code' },
    { id: 'writing',   label: 'Writing',   icon: 'pencil' },
    { id: 'analysis',  label: 'Analysis',  icon: 'chart' },
    { id: 'research',  label: 'Research',  icon: 'search' },
    { id: 'support',   label: 'Support',   icon: 'headset' },
    { id: 'sales',     label: 'Sales',     icon: 'briefcase' },
    { id: 'marketing', label: 'Marketing', icon: 'megaphone' },
  ];

  const modelOptions = [
    {
      id: 'claude-haiku-4-5-20251001',
      name: 'Haiku 4.5',
      speed: 'Fast',
      desc: 'Quick responses, high-volume tasks',
      tags: ['Fastest', 'Efficient'],
    },
    {
      id: 'claude-sonnet-4-6',
      name: 'Sonnet 4.6',
      speed: 'Balanced',
      desc: 'Best intelligence-to-speed ratio',
      tags: ['Recommended', 'Versatile'],
      recommended: true,
    },
    {
      id: 'claude-opus-4-6',
      name: 'Opus 4.6',
      speed: 'Powerful',
      desc: 'Maximum capability for complex tasks',
      tags: ['Most Capable'],
    },
  ];

  const promptTemplates = [
    { label: 'Customer Support', prompt: 'You are a friendly, empathetic customer support specialist. Help users resolve issues quickly and professionally. Acknowledge frustration, provide clear solutions, and follow up to ensure satisfaction.' },
    { label: 'Code Reviewer',    prompt: 'You are an expert code reviewer with deep software engineering knowledge. Review code for correctness, performance, security, and readability. Provide specific, constructive feedback with actionable improvements.' },
    { label: 'Sales Qualifier',  prompt: 'You are a consultative sales specialist. Qualify leads, understand customer needs, and guide prospects through the sales process. Ask thoughtful discovery questions and focus on solving real business problems.' },
    { label: 'Content Writer',   prompt: 'You are a skilled content writer and strategist. Create engaging, well-researched content tailored to the target audience. Focus on clarity, compelling storytelling, and driving the desired reader action.' },
    { label: 'Data Analyst',     prompt: 'You are a data analysis expert. Interpret data, identify patterns, and generate actionable insights. Explain complex findings in clear business-friendly language with concrete next steps.' },
  ];

  const presets = [
    { label: 'Sales Agent',      category: 'sales',    model: 'claude-sonnet-4-6',          color: '#059669', desc: 'Qualify leads and close deals' },
    { label: 'Code Reviewer',    category: 'coding',   model: 'claude-opus-4-6',             color: '#7c3aed', desc: 'Review PRs and find bugs' },
    { label: 'Support Bot',      category: 'support',  model: 'claude-haiku-4-5-20251001',   color: '#1d4ed8', desc: 'Answer questions, resolve tickets' },
    { label: 'Content Writer',   category: 'writing',  model: 'claude-sonnet-4-6',           color: '#db2777', desc: 'Draft copy and long-form content' },
  ];

  // ── Helpers ──
  function getInitials(n: string): string {
    if (!n.trim()) return '?';
    const w = n.trim().split(/\s+/).filter(Boolean);
    if (w.length === 1) return w[0].slice(0, 2).toUpperCase();
    return (w[0][0] + w[1][0]).toUpperCase();
  }

  function deriveHandle(n: string): string {
    return n.toLowerCase().replace(/\s+/g, '_').replace(/[^a-z0-9_]/g, '').slice(0, 40);
  }

  function onNameInput(e: Event) {
    displayName = (e.target as HTMLInputElement).value;
    if (!handleEdited) handle = deriveHandle(displayName);
    if (submitted && !displayName.trim()) errors.displayName = 'Required';
    else { const { displayName: _, ...rest } = errors; errors = rest; }
  }

  function onHandleInput(e: Event) {
    handleEdited = true;
    handle = (e.target as HTMLInputElement).value.toLowerCase().replace(/[^a-z0-9_]/g, '');
  }

  function applyPreset(p: typeof presets[0]) {
    displayName = p.label;
    handle = deriveHandle(p.label);
    category = p.category;
    modelPref = p.model;
    avatarColor = p.color;
    const tmpl = promptTemplates.find(t => t.label.toLowerCase().includes(p.category) || p.label.toLowerCase().includes(t.label.split(' ')[0].toLowerCase()));
    if (tmpl) systemPrompt = tmpl.prompt;
  }

  function applyTemplate(prompt: string) { systemPrompt = prompt; }

  function addPrompt() {
    const t = newPrompt.trim();
    if (t && !suggestedPrompts.includes(t)) {
      suggestedPrompts = [...suggestedPrompts, t];
      newPrompt = '';
    }
  }

  function removePrompt(i: number) {
    suggestedPrompts = suggestedPrompts.filter((_, idx) => idx !== i);
  }

  function onPromptKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') { e.preventDefault(); addPrompt(); }
  }

  function nextStep() {
    submitted = true;
    if (!displayName.trim()) { errors.displayName = 'Required'; return; }
    if (!handle.trim())      { errors.handle = 'Required'; return; }
    errors = {};
    step = 2;
    submitted = false;
  }

  function handleSubmit() {
    submitted = true;
    if (!displayName.trim()) { errors.displayName = 'Required'; step = 1; return; }
    if (!handle.trim())      { errors.handle = 'Required'; step = 1; return; }
    isSubmitting = true;
    onSave({
      display_name:     displayName.trim(),
      name:             handle.trim(),
      description:      description.trim() || undefined,
      category,
      model_preference: modelPref,
      temperature,
      system_prompt:    systemPrompt.trim(),
      welcome_message:  welcomeMsg.trim() || undefined,
      suggested_prompts: suggestedPrompts.length ? suggestedPrompts : undefined,
      is_active:        isActive,
    });
  }

  const tempLabel = $derived(
    temperature < 0.3  ? 'Conservative' :
    temperature < 0.6  ? 'Balanced' :
    temperature < 0.85 ? 'Creative' : 'Experimental'
  );
</script>

<div class="ab">
  <!-- ═══ LEFT PANEL ═══ -->
  <div class="ab-left">

    <!-- Step progress -->
    <div class="ab-steps">
      <button class="ab-step" class:ab-step--done={step > 1} class:ab-step--active={step === 1} onclick={() => step === 2 && (step = 1)}>
        <span class="ab-step__num">{step > 1 ? '✓' : '1'}</span>
        <span class="ab-step__label">Identity</span>
      </button>
      <span class="ab-step__line" class:ab-step__line--done={step > 1}></span>
      <button class="ab-step" class:ab-step--active={step === 2} onclick={() => step > 1 && (step = 2)} disabled={step < 2}>
        <span class="ab-step__num">2</span>
        <span class="ab-step__label">Behavior</span>
      </button>
    </div>

    <!-- Scrollable form body -->
    <div class="ab-body">

      <!-- ── STEP 1: Identity ── -->
      {#if step === 1}

        <!-- Quick-start presets (improvement #15) -->
        <section class="ab-section">
          <p class="ab-section__label">Start from a template</p>
          <div class="ab-presets">
            {#each presets as p}
              <button class="ab-preset" onclick={() => applyPreset(p)}>
                <span class="ab-preset__dot" style="background:{p.color}"></span>
                <span class="ab-preset__name">{p.label}</span>
                <span class="ab-preset__desc">{p.desc}</span>
              </button>
            {/each}
          </div>
        </section>

        <div class="ab-divider"><span>or build from scratch</span></div>

        <!-- Avatar color picker (improvement #3) -->
        <section class="ab-section">
          <p class="ab-section__label">Avatar</p>
          <div class="ab-avatar-row">
            <div class="ab-avatar-preview" style="background:{avatarColor}">
              <span class="ab-avatar-initials">{getInitials(displayName)}</span>
            </div>
            <div class="ab-color-swatches">
              {#each avatarColors as color}
                <button
                  class="ab-swatch"
                  class:ab-swatch--active={avatarColor === color}
                  style="background:{color}"
                  onclick={() => avatarColor = color}
                  aria-label="Color {color}"
                ></button>
              {/each}
            </div>
          </div>
        </section>

        <!-- Display name + handle (improvement #5) -->
        <section class="ab-section">
          <div class="ab-row">
            <div class="ab-field ab-field--grow">
              <label class="ab-label" for="ab-name">
                Display Name <span class="ab-req">*</span>
              </label>
              <input
                id="ab-name"
                type="text"
                class="ab-input"
                class:ab-input--error={errors.displayName}
                value={displayName}
                oninput={onNameInput}
                placeholder="e.g. Sales Assistant"
                maxlength="100"
                autocomplete="off"
              />
              <div class="ab-input-meta">
                {#if errors.displayName}<span class="ab-error-text">{errors.displayName}</span>{/if}
                <span class="ab-char-count" class:ab-char-count--warn={displayName.length > 80}>{displayName.length}/100</span>
              </div>
            </div>

            <div class="ab-field">
              <label class="ab-label" for="ab-handle">Handle <span class="ab-req">*</span></label>
              <div class="ab-handle-wrap" class:ab-handle-wrap--error={errors.handle}>
                <span class="ab-handle-at">@</span>
                <input
                  id="ab-handle"
                  type="text"
                  class="ab-input ab-input--handle"
                  value={handle}
                  oninput={onHandleInput}
                  placeholder="sales_assistant"
                  maxlength="40"
                  autocomplete="off"
                  spellcheck="false"
                />
              </div>
              {#if errors.handle}<p class="ab-error-text">{errors.handle}</p>{/if}
            </div>
          </div>

          <div class="ab-field">
            <label class="ab-label" for="ab-desc">Description</label>
            <input
              id="ab-desc"
              type="text"
              class="ab-input"
              bind:value={description}
              placeholder="What does this agent do?"
              maxlength="300"
              autocomplete="off"
            />
            <div class="ab-input-meta">
              <span></span>
              <span class="ab-char-count" class:ab-char-count--warn={description.length > 240}>{description.length}/300</span>
            </div>
          </div>
        </section>

        <!-- Category tiles (improvement #4) -->
        <section class="ab-section">
          <p class="ab-section__label">Category</p>
          <div class="ab-categories">
            {#each categoryOptions as cat}
              <button
                class="ab-cat"
                class:ab-cat--active={category === cat.id}
                onclick={() => category = cat.id}
              >
                <span class="ab-cat__icon" aria-hidden="true">
                  {#if cat.icon === 'general'}
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"/><path d="M12 1v4M12 19v4M4.22 4.22l2.83 2.83M16.95 16.95l2.83 2.83M1 12h4M19 12h4M4.22 19.78l2.83-2.83M16.95 7.05l2.83-2.83"/></svg>
                  {:else if cat.icon === 'code'}
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/></svg>
                  {:else if cat.icon === 'pencil'}
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
                  {:else if cat.icon === 'chart'}
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="20" x2="18" y2="10"/><line x1="12" y1="20" x2="12" y2="4"/><line x1="6" y1="20" x2="6" y2="14"/></svg>
                  {:else if cat.icon === 'search'}
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>
                  {:else if cat.icon === 'headset'}
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 18v-6a9 9 0 0 1 18 0v6"/><path d="M21 19a2 2 0 0 1-2 2h-1a2 2 0 0 1-2-2v-3a2 2 0 0 1 2-2h3z"/><path d="M3 19a2 2 0 0 0 2 2h1a2 2 0 0 0 2-2v-3a2 2 0 0 0-2-2H3z"/></svg>
                  {:else if cat.icon === 'briefcase'}
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="7" width="20" height="14" rx="2"/><path d="M16 7V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v2"/><line x1="12" y1="12" x2="12" y2="17"/><line x1="9.5" y1="14.5" x2="14.5" y2="14.5"/></svg>
                  {:else if cat.icon === 'megaphone'}
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 11l19-9-9 19-2-8-8-2z"/></svg>
                  {/if}
                </span>
                <span class="ab-cat__label">{cat.label}</span>
              </button>
            {/each}
          </div>
        </section>

      <!-- ── STEP 2: Behavior ── -->
      {:else}

        <!-- Model cards (improvement #6) -->
        <section class="ab-section">
          <p class="ab-section__label">Model</p>
          <div class="ab-models">
            {#each modelOptions as m}
              <button
                class="ab-model"
                class:ab-model--active={modelPref === m.id}
                onclick={() => modelPref = m.id}
              >
                {#if m.recommended}
                  <span class="ab-model__rec">Recommended</span>
                {/if}
                <span class="ab-model__speed ab-model__speed--{m.speed.toLowerCase()}">{m.speed}</span>
                <p class="ab-model__name">{m.name}</p>
                <p class="ab-model__desc">{m.desc}</p>
                <div class="ab-model__tags">
                  {#each m.tags as tag}
                    <span class="ab-model__tag">{tag}</span>
                  {/each}
                </div>
              </button>
            {/each}
          </div>
        </section>

        <!-- Temperature slider (improvement #8) -->
        <section class="ab-section">
          <div class="ab-slider-header">
            <p class="ab-section__label" style="margin:0">Temperature</p>
            <span class="ab-slider-val">{tempLabel} · {temperature.toFixed(1)}</span>
          </div>
          <div class="ab-slider-wrap">
            <span class="ab-slider-edge">Conservative</span>
            <input
              type="range"
              class="ab-slider"
              min="0" max="1" step="0.05"
              bind:value={temperature}
            />
            <span class="ab-slider-edge">Experimental</span>
          </div>
        </section>

        <!-- System prompt + templates (improvements #7, #9) -->
        <section class="ab-section">
          <p class="ab-section__label">System Prompt</p>
          <div class="ab-template-chips">
            {#each promptTemplates as t}
              <button class="ab-chip" class:ab-chip--active={systemPrompt === t.prompt} onclick={() => applyTemplate(t.prompt)}>{t.label}</button>
            {/each}
          </div>
          <textarea
            class="ab-textarea"
            bind:value={systemPrompt}
            placeholder="You are a helpful assistant that..."
            rows="6"
          ></textarea>
          <div class="ab-input-meta">
            <span class="ab-hint">Defines your agent's personality and purpose</span>
            <span class="ab-char-count" class:ab-char-count--warn={systemPrompt.length > 8000}>{systemPrompt.length}/10000</span>
          </div>
        </section>

        <!-- Welcome message + bubble preview (improvement #11) -->
        <section class="ab-section">
          <p class="ab-section__label">Welcome Message <span class="ab-optional">(optional)</span></p>
          <input
            type="text"
            class="ab-input"
            bind:value={welcomeMsg}
            placeholder="Hi! I'm here to help. What can I do for you?"
            maxlength="300"
          />
          {#if welcomeMsg.trim()}
            <div class="ab-bubble-preview">
              <div class="ab-bubble-avatar" style="background:{avatarColor}">
                <span>{getInitials(displayName)}</span>
              </div>
              <div class="ab-bubble">
                <p class="ab-bubble__name">{displayName || 'Agent'}</p>
                <p class="ab-bubble__text">{welcomeMsg}</p>
              </div>
            </div>
          {/if}
        </section>

        <!-- Suggested prompts chip input (improvement #10) -->
        <section class="ab-section">
          <p class="ab-section__label">Suggested Prompts <span class="ab-optional">(optional)</span></p>
          <div class="ab-chip-input-row">
            <input
              type="text"
              class="ab-input"
              bind:value={newPrompt}
              onkeydown={onPromptKeydown}
              placeholder="Type a prompt and press Enter…"
              maxlength="200"
            />
            <button class="ab-chip-add" onclick={addPrompt} disabled={!newPrompt.trim()}>Add</button>
          </div>
          {#if suggestedPrompts.length}
            <div class="ab-chips-list">
              {#each suggestedPrompts as p, i}
                <span class="ab-chip-tag">
                  {p}
                  <button class="ab-chip-tag__remove" onclick={() => removePrompt(i)} aria-label="Remove">
                    <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round"><path d="M18 6 6 18M6 6l12 12"/></svg>
                  </button>
                </span>
              {/each}
            </div>
          {/if}
        </section>

        <!-- Active toggle -->
        <section class="ab-section">
          <div class="ab-toggle-row">
            <div>
              <p class="ab-toggle-label">Active</p>
              <p class="ab-toggle-hint">Inactive agents won't appear in chat</p>
            </div>
            <button
              class="ab-toggle"
              class:ab-toggle--on={isActive}
              onclick={() => isActive = !isActive}
              role="switch"
              aria-checked={isActive}
            >
              <span class="ab-toggle__thumb"></span>
            </button>
          </div>
        </section>

      {/if}
    </div>

    <!-- ── Sticky footer ── -->
    <div class="ab-footer">
      <button class="ab-btn-ghost" onclick={onCancel} disabled={isSubmitting}>Cancel</button>
      <div class="ab-footer-right">
        {#if step === 2}
          <button class="ab-btn-back" onclick={() => step = 1} disabled={isSubmitting}>
            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"><path d="M15 18l-6-6 6-6"/></svg>
            Back
          </button>
        {/if}
        {#if step === 1}
          <button class="ab-btn-next" onclick={nextStep}>
            Continue
            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"><path d="M9 18l6-6-6-6"/></svg>
          </button>
        {:else}
          <button class="ab-btn-create" onclick={handleSubmit} disabled={isSubmitting}>
            {isSubmitting ? 'Creating…' : 'Create Agent'}
          </button>
        {/if}
      </div>
    </div>
  </div>

  <!-- ═══ RIGHT PANEL — Live Preview ═══ -->
  <div class="ab-right">
    <p class="ab-right__label">Live Preview</p>

    <!-- Agent card preview (improvement #12, #1) -->
    <div class="ab-preview-card">
      <div class="ab-preview-card__top">
        <div class="ab-preview-avatar" style="background:{avatarColor}">
          <span class="ab-preview-initials">{getInitials(displayName)}</span>
          <span class="ab-preview-dot" class:ab-preview-dot--on={isActive}></span>
        </div>
        <div class="ab-preview-info">
          <p class="ab-preview-name">{displayName || 'Unnamed Agent'}</p>
          <span class="ab-preview-handle">@{handle || 'handle'}</span>
        </div>
      </div>
      <p class="ab-preview-desc">{description || 'No description yet.'}</p>
      {#if category}
        <div class="ab-preview-footer">
          <span class="ab-preview-badge">{category}</span>
          <span class="ab-preview-model">
            {modelPref.replace('claude-','').replace('-4-6',' 4.6').replace('-4-5-20251001',' 4.5')}
          </span>
        </div>
      {/if}
    </div>

    <!-- Welcome message bubble preview -->
    {#if welcomeMsg.trim()}
      <div class="ab-right__section">
        <p class="ab-right__sublabel">Welcome message</p>
        <div class="ab-preview-bubble">
          <div class="ab-preview-bubble__avatar" style="background:{avatarColor}">
            <span>{getInitials(displayName)}</span>
          </div>
          <div class="ab-preview-bubble__body">
            <p class="ab-preview-bubble__name">{displayName || 'Agent'}</p>
            <p class="ab-preview-bubble__text">{welcomeMsg}</p>
          </div>
        </div>
      </div>
    {/if}

    <!-- Suggested prompts preview -->
    {#if suggestedPrompts.length}
      <div class="ab-right__section">
        <p class="ab-right__sublabel">Suggested prompts</p>
        <div class="ab-preview-prompts">
          {#each suggestedPrompts.slice(0, 3) as p}
            <div class="ab-preview-prompt">{p}</div>
          {/each}
        </div>
      </div>
    {/if}

    <!-- Temp/model info -->
    <div class="ab-right__section">
      <p class="ab-right__sublabel">Configuration</p>
      <div class="ab-preview-meta">
        <div class="ab-preview-meta__row">
          <span class="ab-preview-meta__key">Model</span>
          <span class="ab-preview-meta__val">{modelPref.replace('claude-','').replace(/-4-6$/,' 4.6').replace(/-4-5-20251001$/,' 4.5')}</span>
        </div>
        <div class="ab-preview-meta__row">
          <span class="ab-preview-meta__key">Temperature</span>
          <span class="ab-preview-meta__val">{tempLabel} ({temperature.toFixed(1)})</span>
        </div>
        <div class="ab-preview-meta__row">
          <span class="ab-preview-meta__key">Status</span>
          <span class="ab-preview-meta__val" class:ab-preview-meta__val--active={isActive}>{isActive ? 'Active' : 'Inactive'}</span>
        </div>
      </div>
    </div>
  </div>
</div>

<!-- Building overlay (improvement #13) -->
{#if isSubmitting}
  <div class="ab-building" aria-live="polite">
    <div class="ab-building__card">
      <div class="ab-building__avatar" style="background:{avatarColor}">
        <span class="ab-building__initials">{getInitials(displayName)}</span>
        <div class="ab-building__ring"></div>
      </div>
      <p class="ab-building__title">Building your agent…</p>
      <p class="ab-building__sub">Setting up {displayName || 'your agent'}</p>
      <div class="ab-building__dots">
        <span></span><span></span><span></span>
      </div>
    </div>
  </div>
{/if}

<style>
  /* ═══ AgentBuilder — BOS Tokens ═══ */

  .ab {
    display: flex;
    height: 100%;
    overflow: hidden;
  }

  /* ── Left panel ── */
  .ab-left {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    border-right: 1px solid var(--dbd, #e5e5e5);
    background: var(--dbg, #fff);
    overflow: hidden;
  }

  /* Step progress */
  .ab-steps {
    display: flex;
    align-items: center;
    gap: 0;
    padding: 1.125rem 1.5rem;
    border-bottom: 1px solid var(--dbd2, #f0f0f0);
    background: var(--dbg, #fff);
    flex-shrink: 0;
  }
  .ab-step {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    background: none;
    border: none;
    cursor: pointer;
    padding: 0;
    color: var(--dt3, #888);
  }
  .ab-step--active { color: var(--dt, #111); }
  .ab-step--done   { color: var(--bos-status-success, #22c55e); cursor: pointer; }
  .ab-step__num {
    width: 22px;
    height: 22px;
    border-radius: 50%;
    border: 1.5px solid currentColor;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 0.6875rem;
    font-weight: 700;
    flex-shrink: 0;
  }
  .ab-step--active .ab-step__num {
    background: var(--dt, #111);
    border-color: var(--dt, #111);
    color: var(--dbg, #fff);
  }
  .ab-step--done .ab-step__num {
    background: var(--bos-status-success, #22c55e);
    border-color: var(--bos-status-success, #22c55e);
    color: #fff;
    font-size: 0.625rem;
  }
  .ab-step__label { font-size: 0.8125rem; font-weight: 600; }
  .ab-step__line {
    flex: 1;
    height: 1.5px;
    background: var(--dbd, #e5e5e5);
    margin: 0 0.75rem;
  }
  .ab-step__line--done { background: var(--bos-status-success, #22c55e); }

  /* Scrollable body */
  .ab-body {
    flex: 1;
    overflow-y: auto;
    padding: 1.5rem;
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
  }

  /* Sections */
  .ab-section { display: flex; flex-direction: column; gap: 0.625rem; }
  .ab-section__label {
    font-size: 0.6875rem;
    font-weight: 700;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    color: var(--dt3, #888);
  }
  .ab-optional { font-weight: 400; text-transform: none; letter-spacing: 0; font-size: 0.6875rem; }

  /* Divider */
  .ab-divider {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    color: var(--dt4, #bbb);
    font-size: 0.75rem;
  }
  .ab-divider::before,
  .ab-divider::after {
    content: '';
    flex: 1;
    height: 1px;
    background: var(--dbd, #e5e5e5);
  }

  /* Preset cards */
  .ab-presets {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 0.5rem;
  }
  .ab-preset {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.625rem 0.75rem;
    border: 1px solid var(--dbd, #e5e5e5);
    border-radius: 8px;
    background: var(--dbg, #fff);
    cursor: pointer;
    text-align: left;
    transition: border-color 0.12s, background 0.12s, box-shadow 0.12s;
  }
  .ab-preset:hover {
    border-color: var(--dt2, #555);
    background: var(--dbg2, #fafafa);
    box-shadow: 0 2px 8px rgba(0,0,0,0.06);
  }
  .ab-preset__dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    flex-shrink: 0;
  }
  .ab-preset__name { font-size: 0.8125rem; font-weight: 600; color: var(--dt, #111); white-space: nowrap; }
  .ab-preset__desc { font-size: 0.71875rem; color: var(--dt3, #888); margin-left: auto; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }

  /* Avatar row */
  .ab-avatar-row {
    display: flex;
    align-items: center;
    gap: 1rem;
  }
  .ab-avatar-preview {
    width: 3rem;
    height: 3rem;
    border-radius: 12px;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    transition: background 0.18s;
  }
  .ab-avatar-initials {
    font-size: 1rem;
    font-weight: 700;
    color: rgba(255,255,255,0.92);
    letter-spacing: 0.02em;
    user-select: none;
  }
  .ab-color-swatches {
    display: flex;
    flex-wrap: wrap;
    gap: 0.375rem;
  }
  .ab-swatch {
    width: 22px;
    height: 22px;
    border-radius: 50%;
    border: 2px solid transparent;
    cursor: pointer;
    transition: transform 0.12s, border-color 0.12s;
    padding: 0;
  }
  .ab-swatch:hover { transform: scale(1.15); }
  .ab-swatch--active { border-color: var(--dbg, #fff); box-shadow: 0 0 0 2px var(--dt2, #555); transform: scale(1.1); }

  /* Fields */
  .ab-row { display: flex; gap: 0.75rem; }
  .ab-field { display: flex; flex-direction: column; gap: 0.3rem; }
  .ab-field--grow { flex: 1; min-width: 0; }
  .ab-label { font-size: 0.75rem; font-weight: 600; color: var(--dt2, #444); }
  .ab-req { color: var(--bos-status-error, #ef4444); }

  .ab-input {
    padding: 0.5625rem 0.75rem;
    font-size: 0.875rem;
    font-family: inherit;
    border: 1px solid var(--dbd, #e0e0e0);
    border-radius: 8px;
    background: var(--dbg, #fff);
    color: var(--dt, #111);
    outline: none;
    width: 100%;
    box-sizing: border-box;
    transition: border-color 0.15s, box-shadow 0.15s;
  }
  .ab-input::placeholder { color: var(--dt4, #bbb); }
  .ab-input:focus { border-color: var(--dt2, #555); box-shadow: 0 0 0 3px rgba(0,0,0,0.06); }
  .ab-input--error { border-color: var(--bos-status-error, #ef4444); }
  .ab-input--handle { padding-left: 1.25rem; border: none; outline: none; background: transparent; box-shadow: none; width: 100%; font-family: var(--bos-font-code-family, monospace); font-size: 0.8125rem; }
  .ab-input--handle:focus { box-shadow: none; }

  .ab-handle-wrap {
    display: flex;
    align-items: center;
    border: 1px solid var(--dbd, #e0e0e0);
    border-radius: 8px;
    background: var(--dbg, #fff);
    padding: 0 0.75rem;
    transition: border-color 0.15s, box-shadow 0.15s;
    min-width: 140px;
  }
  .ab-handle-wrap:focus-within { border-color: var(--dt2, #555); box-shadow: 0 0 0 3px rgba(0,0,0,0.06); }
  .ab-handle-wrap--error { border-color: var(--bos-status-error, #ef4444); }
  .ab-handle-at { font-size: 0.875rem; color: var(--dt3, #888); user-select: none; flex-shrink: 0; }

  .ab-input-meta {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 0.5rem;
    min-height: 1rem;
  }
  .ab-char-count { font-size: 0.6875rem; color: var(--dt4, #bbb); margin-left: auto; white-space: nowrap; }
  .ab-char-count--warn { color: var(--bos-status-warning, #f59e0b); }
  .ab-error-text { font-size: 0.71875rem; color: var(--bos-status-error, #ef4444); }
  .ab-hint { font-size: 0.71875rem; color: var(--dt4, #bbb); }

  /* Category tiles */
  .ab-categories {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 0.375rem;
  }
  .ab-cat {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0.3rem;
    padding: 0.625rem 0.25rem;
    border: 1px solid var(--dbd, #e5e5e5);
    border-radius: 8px;
    background: var(--dbg, #fff);
    cursor: pointer;
    transition: all 0.12s;
    color: var(--dt3, #888);
  }
  .ab-cat:hover { border-color: var(--dt2, #555); color: var(--dt, #111); background: var(--dbg2, #fafafa); }
  .ab-cat--active { background: var(--dt, #111); border-color: var(--dt, #111); color: var(--dbg, #fff); }
  .ab-cat__icon { display: flex; align-items: center; justify-content: center; }
  .ab-cat__label { font-size: 0.6875rem; font-weight: 600; white-space: nowrap; }

  /* Model cards */
  .ab-models { display: grid; grid-template-columns: repeat(3, 1fr); gap: 0.5rem; }
  .ab-model {
    position: relative;
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
    padding: 0.875rem 0.75rem;
    border: 1.5px solid var(--dbd, #e5e5e5);
    border-radius: 10px;
    background: var(--dbg, #fff);
    cursor: pointer;
    text-align: left;
    transition: all 0.15s;
  }
  .ab-model:hover { border-color: var(--dt2, #555); background: var(--dbg2, #fafafa); }
  .ab-model--active { border-color: var(--dt, #111); background: var(--dt, #111); color: var(--dbg, #fff); }
  .ab-model__rec {
    position: absolute;
    top: -1px;
    right: -1px;
    font-size: 0.5625rem;
    font-weight: 700;
    letter-spacing: 0.05em;
    text-transform: uppercase;
    background: var(--bos-accent-blue, #3b82f6);
    color: #fff;
    padding: 0.125rem 0.4rem;
    border-radius: 0 9px 0 6px;
  }
  .ab-model__speed {
    font-size: 0.625rem;
    font-weight: 700;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    padding: 0.125rem 0.375rem;
    border-radius: 4px;
    width: fit-content;
    margin-bottom: 0.125rem;
  }
  .ab-model__speed--fast      { background: rgba(34,197,94,0.12);  color: #16a34a; }
  .ab-model__speed--balanced  { background: rgba(59,130,246,0.12); color: #2563eb; }
  .ab-model__speed--powerful  { background: rgba(124,58,237,0.12); color: #7c3aed; }
  .ab-model--active .ab-model__speed--fast     { background: rgba(255,255,255,0.15); color: rgba(255,255,255,0.85); }
  .ab-model--active .ab-model__speed--balanced { background: rgba(255,255,255,0.15); color: rgba(255,255,255,0.85); }
  .ab-model--active .ab-model__speed--powerful { background: rgba(255,255,255,0.15); color: rgba(255,255,255,0.85); }
  .ab-model__name { font-size: 0.875rem; font-weight: 700; }
  .ab-model__desc { font-size: 0.6875rem; color: var(--dt3, #888); line-height: 1.4; }
  .ab-model--active .ab-model__desc { color: rgba(255,255,255,0.65); }
  .ab-model__tags { display: flex; flex-wrap: wrap; gap: 0.25rem; margin-top: 0.25rem; }
  .ab-model__tag {
    font-size: 0.5625rem;
    font-weight: 600;
    padding: 0.1rem 0.3rem;
    border-radius: 3px;
    background: var(--dbg2, #f5f5f5);
    color: var(--dt3, #888);
    border: 1px solid var(--dbd, #e5e5e5);
  }
  .ab-model--active .ab-model__tag { background: rgba(255,255,255,0.12); color: rgba(255,255,255,0.7); border-color: transparent; }

  /* Temperature slider */
  .ab-slider-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }
  .ab-slider-val {
    font-size: 0.75rem;
    font-weight: 600;
    color: var(--dt2, #555);
    font-family: var(--bos-font-code-family, monospace);
  }
  .ab-slider-wrap {
    display: flex;
    align-items: center;
    gap: 0.625rem;
  }
  .ab-slider-edge { font-size: 0.6875rem; color: var(--dt4, #bbb); white-space: nowrap; }
  .ab-slider {
    flex: 1;
    height: 4px;
    appearance: none;
    -webkit-appearance: none;
    background: var(--dbd, #e5e5e5);
    border-radius: 9999px;
    outline: none;
    cursor: pointer;
  }
  .ab-slider::-webkit-slider-thumb {
    appearance: none;
    width: 16px;
    height: 16px;
    border-radius: 50%;
    background: var(--dt, #111);
    border: 2px solid var(--dbg, #fff);
    box-shadow: 0 1px 4px rgba(0,0,0,0.2);
    cursor: pointer;
  }
  .ab-slider::-moz-range-thumb {
    width: 16px;
    height: 16px;
    border-radius: 50%;
    background: var(--dt, #111);
    border: 2px solid var(--dbg, #fff);
    box-shadow: 0 1px 4px rgba(0,0,0,0.2);
    cursor: pointer;
  }

  /* Template chips */
  .ab-template-chips { display: flex; flex-wrap: wrap; gap: 0.3rem; }
  .ab-chip {
    font-size: 0.71875rem;
    font-weight: 500;
    padding: 0.25rem 0.625rem;
    border-radius: 9999px;
    border: 1px solid var(--dbd, #e5e5e5);
    background: var(--dbg, #fff);
    color: var(--dt2, #555);
    cursor: pointer;
    transition: all 0.12s;
    white-space: nowrap;
  }
  .ab-chip:hover { border-color: var(--dt2, #555); color: var(--dt, #111); }
  .ab-chip--active { background: var(--dt, #111); border-color: var(--dt, #111); color: var(--dbg, #fff); }

  .ab-textarea {
    padding: 0.625rem 0.75rem;
    font-size: 0.875rem;
    font-family: inherit;
    line-height: 1.55;
    border: 1px solid var(--dbd, #e0e0e0);
    border-radius: 8px;
    background: var(--dbg, #fff);
    color: var(--dt, #111);
    outline: none;
    width: 100%;
    box-sizing: border-box;
    resize: vertical;
    min-height: 120px;
    transition: border-color 0.15s, box-shadow 0.15s;
  }
  .ab-textarea::placeholder { color: var(--dt4, #bbb); }
  .ab-textarea:focus { border-color: var(--dt2, #555); box-shadow: 0 0 0 3px rgba(0,0,0,0.06); }

  /* Welcome message bubble preview */
  .ab-bubble-preview {
    display: flex;
    gap: 0.625rem;
    padding: 0.75rem;
    background: var(--dbg2, #f9f9f9);
    border: 1px solid var(--dbd2, #f0f0f0);
    border-radius: 10px;
  }
  .ab-bubble-avatar {
    width: 2rem;
    height: 2rem;
    border-radius: 8px;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    font-size: 0.6875rem;
    font-weight: 700;
    color: rgba(255,255,255,0.9);
  }
  .ab-bubble { flex: 1; min-width: 0; }
  .ab-bubble__name { font-size: 0.71875rem; font-weight: 700; color: var(--dt, #111); margin: 0 0 0.25rem; }
  .ab-bubble__text { font-size: 0.8125rem; color: var(--dt2, #555); margin: 0; line-height: 1.5; }

  /* Suggested prompts chip input */
  .ab-chip-input-row { display: flex; gap: 0.5rem; }
  .ab-chip-add {
    padding: 0 0.875rem;
    font-size: 0.8125rem;
    font-weight: 600;
    border: 1px solid var(--dbd, #e0e0e0);
    border-radius: 8px;
    background: var(--dbg, #fff);
    color: var(--dt, #111);
    cursor: pointer;
    white-space: nowrap;
    transition: all 0.12s;
  }
  .ab-chip-add:hover:not(:disabled) { background: var(--dt, #111); color: var(--dbg, #fff); border-color: var(--dt, #111); }
  .ab-chip-add:disabled { opacity: 0.4; cursor: not-allowed; }

  .ab-chips-list { display: flex; flex-wrap: wrap; gap: 0.3rem; }
  .ab-chip-tag {
    display: inline-flex;
    align-items: center;
    gap: 0.3rem;
    font-size: 0.75rem;
    padding: 0.25rem 0.5rem 0.25rem 0.625rem;
    background: var(--dbg2, #f5f5f5);
    border: 1px solid var(--dbd, #e5e5e5);
    border-radius: 9999px;
    color: var(--dt2, #555);
    max-width: 280px;
  }
  .ab-chip-tag__remove {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 14px;
    height: 14px;
    border-radius: 50%;
    border: none;
    background: transparent;
    color: var(--dt3, #888);
    cursor: pointer;
    flex-shrink: 0;
    padding: 0;
  }
  .ab-chip-tag__remove:hover { background: var(--dbd, #e0e0e0); color: var(--dt, #111); }

  /* Active toggle */
  .ab-toggle-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0.75rem 0.875rem;
    background: var(--dbg2, #f9f9f9);
    border: 1px solid var(--dbd2, #f0f0f0);
    border-radius: 8px;
  }
  .ab-toggle-label { font-size: 0.8125rem; font-weight: 600; color: var(--dt, #111); margin: 0; }
  .ab-toggle-hint  { font-size: 0.71875rem; color: var(--dt3, #888); margin: 0.1rem 0 0; }
  .ab-toggle {
    width: 36px; height: 20px;
    border-radius: 9999px;
    border: none;
    background: var(--dbd, #ddd);
    cursor: pointer;
    position: relative;
    transition: background 0.18s;
    flex-shrink: 0;
  }
  .ab-toggle--on { background: var(--dt, #111); }
  .ab-toggle__thumb {
    position: absolute;
    top: 2px; left: 2px;
    width: 16px; height: 16px;
    border-radius: 50%;
    background: #fff;
    box-shadow: 0 1px 3px rgba(0,0,0,0.18);
    transition: transform 0.18s;
  }
  .ab-toggle--on .ab-toggle__thumb { transform: translateX(16px); }

  /* Sticky footer */
  .ab-footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0.875rem 1.5rem;
    border-top: 1px solid var(--dbd2, #f0f0f0);
    background: var(--dbg, #fff);
    flex-shrink: 0;
  }
  .ab-footer-right { display: flex; gap: 0.5rem; align-items: center; }

  .ab-btn-ghost {
    padding: 0.5rem 1rem;
    font-size: 0.8125rem;
    font-weight: 500;
    border: 1px solid var(--dbd, #e0e0e0);
    border-radius: 7px;
    background: transparent;
    color: var(--dt2, #555);
    cursor: pointer;
    transition: all 0.12s;
  }
  .ab-btn-ghost:hover { background: var(--dbg2, #f5f5f5); }
  .ab-btn-ghost:disabled { opacity: 0.4; cursor: not-allowed; }

  .ab-btn-back {
    display: flex;
    align-items: center;
    gap: 0.3rem;
    padding: 0.5rem 0.875rem;
    font-size: 0.8125rem;
    font-weight: 500;
    border: 1px solid var(--dbd, #e0e0e0);
    border-radius: 7px;
    background: transparent;
    color: var(--dt2, #555);
    cursor: pointer;
    transition: all 0.12s;
  }
  .ab-btn-back:hover { background: var(--dbg2, #f5f5f5); }
  .ab-btn-back:disabled { opacity: 0.4; cursor: not-allowed; }

  .ab-btn-next {
    display: flex;
    align-items: center;
    gap: 0.3rem;
    padding: 0.5rem 1.125rem;
    font-size: 0.875rem;
    font-weight: 600;
    border: none;
    border-radius: 8px;
    background: var(--dt, #111);
    color: var(--dbg, #fff);
    cursor: pointer;
    transition: background 0.12s;
  }
  .ab-btn-next:hover { background: var(--dt2, #333); }

  .ab-btn-create {
    padding: 0.5rem 1.25rem;
    font-size: 0.875rem;
    font-weight: 600;
    border: none;
    border-radius: 8px;
    background: var(--dt, #111);
    color: var(--dbg, #fff);
    cursor: pointer;
    transition: background 0.12s;
  }
  .ab-btn-create:hover:not(:disabled) { background: var(--dt2, #333); }
  .ab-btn-create:disabled { opacity: 0.45; cursor: not-allowed; }

  /* ── Right preview panel ── */
  .ab-right {
    width: 280px;
    flex-shrink: 0;
    background: var(--dbg2, #f7f7f7);
    display: flex;
    flex-direction: column;
    gap: 0;
    overflow-y: auto;
    padding: 1.25rem 1.125rem;
  }
  .ab-right__label {
    font-size: 0.6875rem;
    font-weight: 700;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    color: var(--dt3, #888);
    margin: 0 0 0.875rem;
  }
  .ab-right__section { margin-top: 1.125rem; }
  .ab-right__sublabel {
    font-size: 0.6875rem;
    font-weight: 700;
    letter-spacing: 0.05em;
    text-transform: uppercase;
    color: var(--dt4, #bbb);
    margin: 0 0 0.5rem;
  }

  /* Live agent card preview */
  .ab-preview-card {
    background: var(--dbg, #fff);
    border: 1px solid var(--dbd, #e0e0e0);
    border-radius: 12px;
    padding: 1rem;
    display: flex;
    flex-direction: column;
    gap: 0.625rem;
    box-shadow: 0 2px 8px rgba(0,0,0,0.05);
  }
  .ab-preview-card__top { display: flex; align-items: center; gap: 0.625rem; }
  .ab-preview-avatar {
    width: 2.25rem;
    height: 2.25rem;
    border-radius: 8px;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    position: relative;
    transition: background 0.18s;
  }
  .ab-preview-initials { font-size: 0.75rem; font-weight: 700; color: rgba(255,255,255,0.92); user-select: none; }
  .ab-preview-dot {
    position: absolute;
    bottom: -1px;
    right: -1px;
    width: 7px;
    height: 7px;
    border-radius: 50%;
    border: 2px solid var(--dbg, #fff);
    background: var(--dt4, #ccc);
  }
  .ab-preview-dot--on { background: var(--bos-status-success, #22c55e); }
  .ab-preview-info { flex: 1; min-width: 0; }
  .ab-preview-name { font-size: 0.875rem; font-weight: 600; color: var(--dt, #111); margin: 0; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .ab-preview-handle { font-size: 0.625rem; color: var(--dt3, #888); font-family: var(--bos-font-code-family, monospace); }
  .ab-preview-desc { font-size: 0.75rem; color: var(--dt2, #555); margin: 0; line-height: 1.5; display: -webkit-box; -webkit-line-clamp: 2; line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; }
  .ab-preview-footer { display: flex; align-items: center; justify-content: space-between; padding-top: 0.5rem; border-top: 1px solid var(--dbd2, #f0f0f0); }
  .ab-preview-badge { font-size: 0.625rem; font-weight: 600; padding: 0.125rem 0.4rem; background: var(--dbg2, #f5f5f5); border-radius: 4px; color: var(--dt2, #555); text-transform: capitalize; }
  .ab-preview-model { font-size: 0.625rem; color: var(--dt3, #888); font-family: var(--bos-font-code-family, monospace); }

  /* Welcome bubble preview in right panel */
  .ab-preview-bubble {
    display: flex;
    gap: 0.5rem;
    padding: 0.625rem;
    background: var(--dbg, #fff);
    border: 1px solid var(--dbd, #e0e0e0);
    border-radius: 10px;
  }
  .ab-preview-bubble__avatar {
    width: 1.75rem;
    height: 1.75rem;
    border-radius: 7px;
    flex-shrink: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 0.5625rem;
    font-weight: 700;
    color: rgba(255,255,255,0.9);
  }
  .ab-preview-bubble__body { flex: 1; min-width: 0; }
  .ab-preview-bubble__name { font-size: 0.6875rem; font-weight: 700; color: var(--dt, #111); margin: 0 0 0.2rem; }
  .ab-preview-bubble__text { font-size: 0.75rem; color: var(--dt2, #555); margin: 0; line-height: 1.45; }

  /* Suggested prompts preview */
  .ab-preview-prompts { display: flex; flex-direction: column; gap: 0.3rem; }
  .ab-preview-prompt {
    font-size: 0.71875rem;
    color: var(--dt2, #555);
    padding: 0.375rem 0.625rem;
    background: var(--dbg, #fff);
    border: 1px solid var(--dbd, #e0e0e0);
    border-radius: 7px;
    line-height: 1.4;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }

  /* Meta table */
  .ab-preview-meta { display: flex; flex-direction: column; gap: 0.375rem; }
  .ab-preview-meta__row { display: flex; justify-content: space-between; align-items: center; }
  .ab-preview-meta__key { font-size: 0.6875rem; color: var(--dt3, #888); }
  .ab-preview-meta__val { font-size: 0.6875rem; font-weight: 600; color: var(--dt2, #555); font-family: var(--bos-font-code-family, monospace); }
  .ab-preview-meta__val--active { color: var(--bos-status-success, #22c55e); }

  /* Building overlay (improvement #13) */
  .ab-building {
    position: fixed;
    inset: 0;
    z-index: 100;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(0,0,0,0.5);
    backdrop-filter: blur(4px);
  }
  .ab-building__card {
    background: var(--dbg, #fff);
    border-radius: 16px;
    padding: 2.5rem;
    text-align: center;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0.75rem;
    box-shadow: 0 24px 64px rgba(0,0,0,0.2);
  }
  .ab-building__avatar {
    width: 4rem;
    height: 4rem;
    border-radius: 16px;
    display: flex;
    align-items: center;
    justify-content: center;
    position: relative;
  }
  .ab-building__initials { font-size: 1.25rem; font-weight: 700; color: rgba(255,255,255,0.92); }
  .ab-building__ring {
    position: absolute;
    inset: -5px;
    border-radius: 20px;
    border: 2.5px solid transparent;
    border-top-color: var(--bos-accent-blue, #3b82f6);
    animation: ab-spin 0.9s linear infinite;
  }
  @keyframes ab-spin { to { transform: rotate(360deg); } }
  .ab-building__title { font-size: 1rem; font-weight: 700; color: var(--dt, #111); margin: 0; }
  .ab-building__sub   { font-size: 0.8125rem; color: var(--dt3, #888); margin: 0; }
  .ab-building__dots {
    display: flex;
    gap: 0.375rem;
    margin-top: 0.25rem;
  }
  .ab-building__dots span {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--dt3, #ccc);
    animation: ab-dot 1.2s ease-in-out infinite;
  }
  .ab-building__dots span:nth-child(2) { animation-delay: 0.2s; }
  .ab-building__dots span:nth-child(3) { animation-delay: 0.4s; }
  @keyframes ab-dot {
    0%, 80%, 100% { transform: scale(0.7); opacity: 0.4; }
    40%           { transform: scale(1);   opacity: 1; }
  }
</style>
