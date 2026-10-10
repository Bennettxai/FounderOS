/**
 * useVoiceRecorder.svelte.ts
 *
 * Svelte 5 runes-based hook encapsulating all voice recording logic:
 *   - MediaRecorder for audio capture
 *   - AudioContext + AnalyserNode for waveform visualization
 *   - Web Speech API for live transcription during recording
 *   - Whisper (via /transcribe endpoint) for final transcription on stop
 *
 * Consumers: chat/+page.svelte, popup-chat/+page.svelte, SpotlightSearch.svelte
 */

import { apiClient } from "$lib/api";

export interface VoiceRecorderOptions {
  /** Number of waveform bars to render. Default: 40 */
  barCount?: number;
  /** Maximum bar height in pixels. Default: 24 */
  maxBarHeight?: number;
  /** Minimum bar height in pixels. Default: 2 */
  minBarHeight?: number;
  /** Callback invoked with the final transcribed text on a successful stop+transcribe. */
  onTranscription?: (text: string) => void;
  /** Callback invoked when transcription fails (receives the error message). */
  onTranscriptionError?: (message: string) => void;
}

export function useVoiceRecorder(options: VoiceRecorderOptions = {}) {
  const {
    barCount = 40,
    maxBarHeight = 24,
    minBarHeight = 2,
    onTranscription,
    onTranscriptionError,
  } = options;

  // ---------------------------------------------------------------------------
  // Reactive state (Svelte 5 runes)
  // ---------------------------------------------------------------------------
  let isRecording = $state(false);
  let isTranscribing = $state(false);
  let recordingDuration = $state(0);
  let liveTranscript = $state("");
  let waveformBars = $state<number[]>(Array(barCount).fill(minBarHeight));

  // ---------------------------------------------------------------------------
  // Non-reactive internal refs
  // ---------------------------------------------------------------------------
  let mediaRecorder: MediaRecorder | null = null;
  let audioChunks: Blob[] = [];
  let audioContext: AudioContext | null = null;
  let analyser: AnalyserNode | null = null;
  let audioDataArray: Uint8Array<ArrayBuffer> | null = null;
  let animationFrameId: number | null = null;
  let recordingInterval: ReturnType<typeof setInterval> | null = null;
  let speechRecognition: SpeechRecognition | null = null;
  /** Reference to the live media stream so we can stop its tracks. */
  let activeStream: MediaStream | null = null;

  // ---------------------------------------------------------------------------
  // Derived
  // ---------------------------------------------------------------------------
  const recordingTimeDisplay = $derived(formatDuration(recordingDuration));

  // ---------------------------------------------------------------------------
  // Helpers
  // ---------------------------------------------------------------------------
  function formatDuration(seconds: number): string {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins}:${secs.toString().padStart(2, "0")}`;
  }

  function startWaveformAnimation(): void {
    function tick() {
      if (!analyser || !audioDataArray) {
        animationFrameId = requestAnimationFrame(tick);
        return;
      }

      analyser.getByteTimeDomainData(audioDataArray);

      const bars: number[] = [];
      const step = Math.floor(audioDataArray.length / barCount);
      const range = maxBarHeight - minBarHeight;

      for (let i = 0; i < barCount; i++) {
        const value = audioDataArray[i * step] ?? 128;
        const deviation = Math.abs(value - 128); // 0-128
        const height = Math.max(
          minBarHeight,
          Math.min(
            maxBarHeight,
            minBarHeight + (deviation / 128) * (range + 20),
          ),
        );
        bars.push(height);
      }

      waveformBars = bars;
      animationFrameId = requestAnimationFrame(tick);
    }

    animationFrameId = requestAnimationFrame(tick);
  }

  function stopWaveformAnimation(): void {
    if (animationFrameId !== null) {
      cancelAnimationFrame(animationFrameId);
      animationFrameId = null;
    }
  }

  function startSpeechRecognition(): void {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const w = window as any;
    const SpeechRecognitionAPI: (new () => SpeechRecognition) | undefined =
      w.SpeechRecognition ?? w.webkitSpeechRecognition;

    if (!SpeechRecognitionAPI) return;

    const recognition = new SpeechRecognitionAPI();
    recognition.continuous = true;
    recognition.interimResults = true;
    recognition.lang = "en-US";

    recognition.onresult = (event: SpeechRecognitionEvent) => {
      let interim = "";
      let final = "";
      for (let i = event.resultIndex; i < event.results.length; i++) {
        const text = event.results[i][0].transcript;
        if (event.results[i].isFinal) {
          final += text;
        } else {
          interim += text;
        }
      }
      liveTranscript = final || interim;
    };

    recognition.onerror = (event: SpeechRecognitionErrorEvent) => {
      // Non-fatal: live transcript is best-effort
      console.warn("[useVoiceRecorder] Speech recognition error:", event.error);
    };

    recognition.start();
    speechRecognition = recognition;
  }

  function stopSpeechRecognition(): void {
    if (speechRecognition) {
      speechRecognition.stop();
      speechRecognition = null;
    }
  }

  /** Tears down all audio infrastructure without touching isRecording. */
  function teardownAudio(): void {
    stopWaveformAnimation();
    stopSpeechRecognition();

    if (recordingInterval !== null) {
      clearInterval(recordingInterval);
      recordingInterval = null;
    }

    if (audioContext) {
      audioContext.close();
      audioContext = null;
      analyser = null;
      audioDataArray = null;
    }

    if (activeStream) {
      activeStream.getTracks().forEach((track) => track.stop());
      activeStream = null;
    }

    waveformBars = Array(barCount).fill(minBarHeight);
  }

  // ---------------------------------------------------------------------------
  // Transcription
  // ---------------------------------------------------------------------------
  async function transcribeAudio(audioBlob: Blob): Promise<void> {
    isTranscribing = true;
    try {
      const formData = new FormData();
      formData.append("audio", audioBlob, "recording.webm");

      const response = await apiClient.postFormData("/transcribe", formData);

      if (response.ok) {
        const data = await response.json();
        if (data.text) {
          onTranscription?.(data.text as string);
        } else if (liveTranscript) {
          // Server returned empty text — use live transcript
          onTranscription?.(liveTranscript);
        }
      } else {
        // Server error — fall back to live transcript from Web Speech API
        if (liveTranscript) {
          onTranscription?.(liveTranscript);
        } else {
          const error = await response
            .json()
            .catch(() => ({ message: "Unknown error" }));
          const message =
            (error as { message?: string }).message ?? "Unknown error";
          console.error("[useVoiceRecorder] Transcription failed:", message);
          onTranscriptionError?.(message);
        }
      }
    } catch (err) {
      console.error("[useVoiceRecorder] Transcription error:", err);
      // Surface live transcript as fallback when available
      if (liveTranscript) {
        onTranscription?.(liveTranscript);
      } else {
        onTranscriptionError?.(
          "Voice transcription requires whisper.cpp to be installed locally.",
        );
      }
    } finally {
      isTranscribing = false;
      liveTranscript = "";
    }
  }

  // ---------------------------------------------------------------------------
  // Public API
  // ---------------------------------------------------------------------------
  async function startRecording(): Promise<void> {
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      activeStream = stream;

      // Audio analyzer for waveform
      audioContext = new AudioContext();
      analyser = audioContext.createAnalyser();
      analyser.fftSize = 256;
      analyser.smoothingTimeConstant = 0.3;
      const source = audioContext.createMediaStreamSource(stream);
      source.connect(analyser);
      audioDataArray = new Uint8Array(analyser.fftSize);

      // MediaRecorder setup
      mediaRecorder = new MediaRecorder(stream);
      audioChunks = [];
      liveTranscript = "";

      mediaRecorder.ondataavailable = (event) => {
        audioChunks.push(event.data);
      };

      mediaRecorder.onstop = async () => {
        const audioBlob = new Blob(audioChunks, { type: "audio/webm" });
        await transcribeAudio(audioBlob);
      };

      mediaRecorder.start();
      isRecording = true;
      recordingDuration = 0;

      // Duration timer
      recordingInterval = setInterval(() => {
        recordingDuration++;
      }, 1000);

      startWaveformAnimation();
      startSpeechRecognition();
    } catch (error) {
      console.error("[useVoiceRecorder] Failed to start recording:", error);
      alert(
        "Could not access microphone. Please grant microphone permissions.",
      );
    }
  }

  function stopRecording(): void {
    if (mediaRecorder && mediaRecorder.state !== "inactive") {
      mediaRecorder.stop();
    }
    isRecording = false;
    teardownAudio();
    recordingDuration = 0;
  }

  /**
   * Cancels recording without triggering transcription.
   * Clears the onstop handler before stopping the MediaRecorder.
   */
  function cancelRecording(): void {
    if (mediaRecorder && mediaRecorder.state !== "inactive") {
      // Null out handlers so onstop does not fire transcription
      mediaRecorder.ondataavailable = null;
      mediaRecorder.onstop = null;
      mediaRecorder.stop();
    }
    isRecording = false;
    teardownAudio();
    liveTranscript = "";
    recordingDuration = 0;
    audioChunks = [];
  }

  function toggleRecording(): void {
    if (isRecording) {
      stopRecording();
    } else {
      startRecording();
    }
  }

  /** Call in onDestroy to guarantee all resources are released. */
  function cleanup(): void {
    cancelRecording();
  }

  // ---------------------------------------------------------------------------
  // Return surface (getter pattern keeps state readonly outside the hook)
  // ---------------------------------------------------------------------------
  return {
    get isRecording() {
      return isRecording;
    },
    get isTranscribing() {
      return isTranscribing;
    },
    get recordingDuration() {
      return recordingDuration;
    },
    get liveTranscript() {
      return liveTranscript;
    },
    get waveformBars() {
      return waveformBars;
    },
    get recordingTimeDisplay() {
      return recordingTimeDisplay;
    },
    startRecording,
    stopRecording,
    cancelRecording,
    toggleRecording,
    cleanup,
  };
}
