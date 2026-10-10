<script lang="ts">
  /**
   * ChatVoice.svelte — Pure display component for the voice recording overlay.
   * State is owned by the parent (+page.svelte). This component renders:
   *   - Recording waveform overlay (when isRecording)
   *   - Transcribing spinner (when isTranscribing)
   *   - Microphone trigger button (idle state)
   */

  interface Props {
    isRecording: boolean;
    isTranscribing: boolean;
    waveformBars: number[];
    liveTranscript: string;
    recordingTimeDisplay: string;
    onToggleRecording: () => void;
    onCancelRecording: () => void;
    onStopRecording: () => void;
  }

  let {
    isRecording,
    isTranscribing,
    waveformBars,
    liveTranscript,
    recordingTimeDisplay,
    onToggleRecording,
    onCancelRecording,
    onStopRecording,
  }: Props = $props();
</script>

{#if isRecording}
  <!-- Recording waveform overlay -->
  <div class="mb-3">
    <!-- Live transcript preview -->
    {#if liveTranscript}
      <div class="text-gray-600 text-sm mb-3 min-h-[24px] animate-pulse">
        {liveTranscript}
      </div>
    {:else}
      <div class="text-gray-400 text-sm mb-3 min-h-[24px]">Listening...</div>
    {/if}

    <!-- Waveform bar -->
    <div class="flex items-center gap-3 bg-gray-800 rounded-full px-4 py-2">
      <!-- Cancel button -->
      <button
        onclick={(e) => {
          e.stopPropagation();
          onCancelRecording();
        }}
        class="p-1.5 text-gray-400 hover:text-white transition-colors"
        aria-label="Cancel recording"
      >
        <svg
          class="w-5 h-5"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M6 18L18 6M6 6l12 12"
          />
        </svg>
      </button>

      <!-- Waveform visualization -->
      <div class="flex-1 flex items-center justify-center gap-[1px] h-8">
        {#each waveformBars as height}
          <div
            class="w-[2px] bg-white rounded-full"
            style="height: {height}px"
          ></div>
        {/each}
      </div>

      <!-- Duration -->
      <span class="text-white font-mono text-sm min-w-[40px] text-right"
        >{recordingTimeDisplay}</span
      >

      <!-- Confirm / stop button -->
      <button
        onclick={(e) => {
          e.stopPropagation();
          onStopRecording();
        }}
        class="p-1.5 bg-white text-gray-800 rounded-full hover:bg-gray-200 transition-colors"
        aria-label="Stop and transcribe"
      >
        <svg
          class="w-5 h-5"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M5 13l4 4L19 7"
          />
        </svg>
      </button>
    </div>
  </div>
{:else if isTranscribing}
  <!-- Transcribing spinner -->
  <div class="flex items-center gap-3 mb-3 py-4 px-2">
    <svg
      class="w-5 h-5 animate-spin text-blue-500"
      fill="none"
      viewBox="0 0 24 24"
    >
      <circle
        class="opacity-25"
        cx="12"
        cy="12"
        r="10"
        stroke="currentColor"
        stroke-width="4"
      ></circle>
      <path
        class="opacity-75"
        fill="currentColor"
        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
      ></path>
    </svg>
    <span class="text-blue-500 font-medium">Transcribing audio...</span>
  </div>
{/if}
