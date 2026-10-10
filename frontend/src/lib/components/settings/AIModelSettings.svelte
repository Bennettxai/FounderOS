<script lang="ts">
  import ModelBrowserSection from './ModelBrowserSection.svelte';
  import ModelSettingsSection from './ModelSettingsSection.svelte';
  import type {
    LLMModel,
    LLMProvider,
    SystemInfo,
    OutputStyle,
    OutputPreference,
  } from '$lib/stores/aiSettings';

  interface Props {
    models: LLMModel[];
    providers: LLMProvider[];
    activeProvider: string;
    defaultModel: string;
    systemInfo: SystemInfo | null;
    outputStyles: OutputStyle[];
    outputPreference: OutputPreference | null;
    selectedDefaultStyleId: string;
    loadingOutputStyles: boolean;
    savingOutputPreference: boolean;
    modelSettings: {
      temperature: number;
      maxTokens: number;
      contextWindow: number;
      topP: number;
      streamResponses: boolean;
      showUsageInChat: boolean;
    };
    isSaving: boolean;
    onSaveSettings: () => void;
    onSaveOutputPreference: () => void;
    onSelectDefaultStyleId: (id: string) => void;
    onUpdateModelSettings: (settings: {
      temperature: number;
      maxTokens: number;
      contextWindow: number;
      topP: number;
      streamResponses: boolean;
      showUsageInChat: boolean;
    }) => void;
    onSetDefaultModel: (modelId: string) => void;
    /** Which parent tab is active — controls which section renders */
    activeTab?: 'models' | 'settings';
  }

  let {
    models,
    providers,
    activeProvider,
    defaultModel,
    systemInfo,
    outputStyles,
    outputPreference,
    selectedDefaultStyleId,
    loadingOutputStyles,
    savingOutputPreference,
    modelSettings,
    isSaving,
    onSaveSettings,
    onSaveOutputPreference,
    onSelectDefaultStyleId,
    onUpdateModelSettings,
    onSetDefaultModel,
    activeTab = 'models',
  }: Props = $props();
</script>

{#if activeTab === 'models'}
  <ModelBrowserSection
    {models}
    {activeProvider}
    {defaultModel}
    {systemInfo}
    {onSetDefaultModel}
    {onSaveSettings}
  />
{/if}

{#if activeTab === 'settings'}
  <ModelSettingsSection
    {models}
    {providers}
    {activeProvider}
    {defaultModel}
    {outputStyles}
    {outputPreference}
    {selectedDefaultStyleId}
    {loadingOutputStyles}
    {savingOutputPreference}
    {modelSettings}
    {isSaving}
    {onSaveSettings}
    {onSaveOutputPreference}
    {onSelectDefaultStyleId}
    {onUpdateModelSettings}
    {onSetDefaultModel}
  />
{/if}
