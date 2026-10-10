<script lang="ts">
  import type { AgentInfo, CustomAgent } from '$lib/stores/aiSettings';
  import CustomAgentsSection from './CustomAgentsSection.svelte';
  import BuiltinAgentsSection from './BuiltinAgentsSection.svelte';

  interface Props {
    agents: AgentInfo[];
    loadingAgents: boolean;
    customAgents: CustomAgent[];
    loadingCustomAgents: boolean;
    onAgentUpdated: () => void;
  }

  let {
    agents,
    loadingAgents,
    customAgents,
    loadingCustomAgents,
    onAgentUpdated
  }: Props = $props();
</script>

<CustomAgentsSection
  {customAgents}
  {loadingCustomAgents}
  {onAgentUpdated}
/>

<BuiltinAgentsSection
  {agents}
  {loadingAgents}
  {onAgentUpdated}
/>
