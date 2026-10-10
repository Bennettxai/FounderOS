<script lang="ts">
  import { fly } from "svelte/transition";
  import ConversationListPanel from "../conversations/ConversationListPanel.svelte";
  import Pagination from "$lib/components/ui/Pagination.svelte";

  interface Conversation {
    id: string;
    title: string;
    timestamp: string;
    pinned?: boolean;
  }

  interface Props {
    open: boolean;
    conversations: Conversation[];
    archivedConversations: Conversation[];
    activeConversationId: string;
    showArchivedView: boolean;
    searchQuery: string;
    conversationsPage: number;
    conversationsPageSize: number;
    conversationsTotal: number;

    // Callbacks
    onselect: (id: string) => void;
    onnewchat: () => void;
    onrename: (id: string) => void;
    onpin: (id: string) => void;
    ondelete: (id: string) => void;
    onarchive: (id: string) => void;
    onunarchive: (id: string) => void;
    onexport: (id: string) => void;
    onlinkproject: (id: string) => void;
    onviewarchived: () => void;
    onbacktochats: () => void;
    onpagechange: (page: number) => void;
    onsearchchange: (query: string) => void;
  }

  let {
    open,
    conversations,
    archivedConversations,
    activeConversationId,
    showArchivedView,
    searchQuery = $bindable(),
    conversationsPage,
    conversationsPageSize,
    conversationsTotal,
    onselect,
    onnewchat,
    onrename,
    onpin,
    ondelete,
    onarchive,
    onunarchive,
    onexport,
    onlinkproject,
    onviewarchived,
    onbacktochats,
    onpagechange,
    onsearchchange,
  }: Props = $props();
</script>

{#if open}
  <div
    class="chat-sidebar-panel w-72 h-full flex-shrink-0 border-r border-gray-200"
    transition:fly={{ x: -288, duration: 200 }}
  >
    <ConversationListPanel
      {conversations}
      {archivedConversations}
      activeId={activeConversationId}
      showArchived={showArchivedView}
      bind:searchQuery
      onSelect={onselect}
      onNewChat={onnewchat}
      onRename={onrename}
      onPin={onpin}
      onDelete={ondelete}
      onArchive={onarchive}
      onUnarchive={onunarchive}
      onExport={onexport}
      onLinkProject={onlinkproject}
      onViewArchived={onviewarchived}
      onBackToChats={onbacktochats}
    />

    <!-- Pagination for conversations list -->
    {#if !showArchivedView && conversationsTotal > conversationsPageSize}
      <div class="px-4 py-2 border-t border-gray-200 bg-white">
        <Pagination
          page={conversationsPage}
          pageSize={conversationsPageSize}
          total={conversationsTotal}
          onPageChange={onpagechange}
        />
      </div>
    {/if}
  </div>
{/if}

<style>
  /* ── Mobile: ChatSidebar ──────────────────────────────────────────── */

  /* Desktop: a fixed-width inline panel */
  .chat-sidebar-panel {
    background: var(--dbg, #fff);
    overflow-y: auto;
    overflow-x: hidden;
  }

  @media (max-width: 767px) {
    /* On mobile the sidebar becomes an off-canvas overlay drawer.
       The parent is responsible for rendering a backdrop and toggling `open`.
       When open=true, this panel slides in from the left. */
    .chat-sidebar-panel {
      position: fixed;
      top: 0;
      left: 0;
      bottom: 0;
      z-index: 60;
      width: 85vw !important;
      max-width: 320px;
      border-right: 1px solid var(--dbd, #e5e7eb);
      box-shadow: 4px 0 24px rgba(0, 0, 0, 0.18);
    }
  }
</style>

