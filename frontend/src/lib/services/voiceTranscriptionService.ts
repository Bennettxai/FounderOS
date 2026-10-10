/**
 * Voice Transcription Service
 *
 * Real-time speech-to-text using the Web Speech API (built into modern browsers).
 * Zero dependencies, zero API keys.
 * Features:
 * - Continuous recognition with interim results
 * - Auto-restart on silence (browser stops after pauses — we restart if still active)
 * - Callback API: (text, isFinal) => void
 */

// Web Speech API types (not in default TS lib)
interface SpeechRecognitionResult {
  readonly isFinal: boolean;
  readonly length: number;
  [index: number]: { transcript: string; confidence: number };
}

interface SpeechRecognitionResultList {
  readonly length: number;
  [index: number]: SpeechRecognitionResult;
}

interface SpeechRecognitionEvent extends Event {
  readonly resultIndex: number;
  readonly results: SpeechRecognitionResultList;
}

interface SpeechRecognitionErrorEvent extends Event {
  readonly error: string;
  readonly message: string;
}

interface SpeechRecognitionInstance extends EventTarget {
  continuous: boolean;
  interimResults: boolean;
  lang: string;
  onresult: ((event: SpeechRecognitionEvent) => void) | null;
  onerror: ((event: SpeechRecognitionErrorEvent) => void) | null;
  onend: (() => void) | null;
  start(): void;
  stop(): void;
  abort(): void;
}

interface SpeechRecognitionConstructor {
  new (): SpeechRecognitionInstance;
}

declare global {
  interface Window {
    SpeechRecognition?: SpeechRecognitionConstructor;
    webkitSpeechRecognition?: SpeechRecognitionConstructor;
  }
}

export type TranscriptCallback = (text: string, isFinal: boolean) => void;

class VoiceTranscriptionService {
  private recognition: SpeechRecognitionInstance | null = null;
  private callback: TranscriptCallback | null = null;
  private onStopCallback: (() => void) | null = null;
  private isActive = false;
  private shouldRestart = false;
  private restartAttempts = 0;
  private restartTimer: ReturnType<typeof setTimeout> | null = null;
  private static MAX_RESTARTS = 10;
  private static RESTART_DELAY = 300; // ms between restarts

  async start(
    onTranscript: TranscriptCallback,
    onStop?: () => void,
  ): Promise<boolean> {
    if (this.isActive) return false;

    const SpeechRecognition =
      window.SpeechRecognition || window.webkitSpeechRecognition;
    if (!SpeechRecognition) {
      console.error("[Voice] Web Speech API not supported in this browser");
      return false;
    }

    try {
      this.callback = onTranscript;
      this.onStopCallback = onStop ?? null;
      this.restartAttempts = 0;

      const recognition = new SpeechRecognition();
      recognition.continuous = true;
      recognition.interimResults = true;
      recognition.lang = "en-US";

      recognition.onresult = (event: SpeechRecognitionEvent) => {
        // Successful result → reset restart counter
        this.restartAttempts = 0;
        for (let i = event.resultIndex; i < event.results.length; i++) {
          const result = event.results[i];
          const text = result[0].transcript.trim();
          if (text) {
            this.callback?.(text, result.isFinal);
          }
        }
      };

      recognition.onerror = (event: SpeechRecognitionErrorEvent) => {
        // 'no-speech' is normal — user just paused. 'aborted' means we called stop().
        if (event.error === "no-speech" || event.error === "aborted") return;
        console.warn("[Voice] Error:", event.error);
        // Fatal errors — stop completely, don't retry
        if (
          event.error === "network" ||
          event.error === "not-allowed" ||
          event.error === "service-not-allowed"
        ) {
          console.error(`[Voice] Fatal error (${event.error}), stopping.`);
          this.shouldRestart = false;
          this.stop();
        }
      };

      // Browser may stop recognition after silence. Restart if we're still active.
      recognition.onend = () => {
        if (this.shouldRestart && this.isActive) {
          this.restartAttempts++;
          if (this.restartAttempts > VoiceTranscriptionService.MAX_RESTARTS) {
            console.warn("[Voice] Too many restarts, stopping.");
            this.cleanupAndNotify();
            return;
          }
          // Delay restart to avoid rapid-fire loops
          this.restartTimer = setTimeout(() => {
            if (!this.shouldRestart || !this.isActive) return;
            try {
              recognition.start();
            } catch {
              // Already started or disposed — ignore
            }
          }, VoiceTranscriptionService.RESTART_DELAY);
        } else if (this.isActive) {
          // Recognition ended but we didn't ask — notify caller
          this.cleanupAndNotify();
        }
      };

      recognition.start();
      this.recognition = recognition;
      this.isActive = true;
      this.shouldRestart = true;

      return true;
    } catch (err) {
      console.error("[Voice] Start failed:", err);
      this.stop();
      return false;
    }
  }

  /** Called by the user/component — deliberate stop. Does NOT fire onStopCallback. */
  stop() {
    this.shouldRestart = false;
    this.isActive = false;

    if (this.restartTimer) {
      clearTimeout(this.restartTimer);
      this.restartTimer = null;
    }

    if (this.recognition) {
      try {
        this.recognition.stop();
      } catch {
        // Already stopped
      }
      this.recognition = null;
    }

    this.callback = null;
    this.onStopCallback = null;
  }

  /** Called internally when recognition dies unexpectedly — fires onStopCallback. */
  private cleanupAndNotify() {
    this.shouldRestart = false;
    this.isActive = false;

    if (this.restartTimer) {
      clearTimeout(this.restartTimer);
      this.restartTimer = null;
    }

    this.recognition = null;
    this.callback = null;

    const cb = this.onStopCallback;
    this.onStopCallback = null;
    cb?.();
  }

  isListening(): boolean {
    return this.isActive;
  }
}

export const voiceTranscription = new VoiceTranscriptionService();
