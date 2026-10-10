/**
 * OSA Voice Service
 *
 * Handles text-to-speech for OSA (Operating System Agent) using ElevenLabs.
 * Provides voice responses in 3D Desktop with speaking indicators.
 *
 * Features:
 * - Text-to-speech via ElevenLabs backend
 * - Audio playback with volume control
 * - Speaking state tracking
 * - Voice activity callbacks
 */

import { browser } from "$app/environment";

export type SpeakingCallback = (isSpeaking: boolean) => void;

export class OSAVoiceService {
  private static instance: OSAVoiceService | null = null;

  private isSpeaking = false;
  private currentAudio: HTMLAudioElement | null = null;
  private speakingCallback: SpeakingCallback | null = null;
  private audioQueue: string[] = [];
  private isProcessingQueue = false;

  private constructor() {}

  static getInstance(): OSAVoiceService {
    if (!OSAVoiceService.instance) {
      OSAVoiceService.instance = new OSAVoiceService();
    }
    return OSAVoiceService.instance;
  }

  /**
   * Set callback for speaking state changes
   */
  onSpeakingChange(callback: SpeakingCallback) {
    this.speakingCallback = callback;
  }

  /**
   * Speak text using OSA voice
   */
  async speak(text: string): Promise<boolean> {
    if (!browser) return false;

    if (!text || text.trim().length === 0) {
      return false;
    }

    // Add to queue
    this.audioQueue.push(text);

    // Start processing queue if not already processing
    if (!this.isProcessingQueue) {
      this.processQueue();
    } else {
      // CRITICAL FIX: Ensure queue doesn't stall
      this.ensureQueueProcessing();
    }

    return true;
  }

  /**
   * IMPROVED: Heartbeat mechanism to prevent queue from stalling
   * Multiple checks to ensure queue never gets stuck
   */
  private ensureQueueProcessing() {
    // Check after 1 second
    setTimeout(() => {
      if (
        this.audioQueue.length > 0 &&
        !this.isSpeaking &&
        !this.isProcessingQueue
      ) {
        console.warn("[OSA Voice] ⚠️ Queue stalled (1s), restarting");
        this.processQueue();
      }
    }, 1000);

    // Backup check after 3 seconds (in case first one fails)
    setTimeout(() => {
      if (
        this.audioQueue.length > 0 &&
        !this.isSpeaking &&
        !this.isProcessingQueue
      ) {
        console.warn("[OSA Voice] ⚠️ Queue stalled (3s), forcing restart");
        this.processQueue();
      }
    }, 3000);

    // Final fallback after 5 seconds
    setTimeout(() => {
      if (this.audioQueue.length > 0 && !this.isSpeaking) {
        console.error(
          "[OSA Voice] 🔥 Queue critically stalled (5s), hard reset",
        );
        this.isProcessingQueue = false;
        this.processQueue();
      }
    }, 5000);
  }

  /**
   * Process audio queue (one at a time)
   */
  private async processQueue() {
    if (this.isProcessingQueue) return;
    this.isProcessingQueue = true;

    while (this.audioQueue.length > 0) {
      const text = this.audioQueue.shift()!;
      try {
        await this.speakNow(text);
      } catch (err) {
        console.error("[OSA Voice] Error speaking, continuing queue:", err);
        // Continue processing queue even on error
      }
    }

    this.isProcessingQueue = false;

    // CRITICAL: Check if new items were added during processing
    if (this.audioQueue.length > 0) {
      this.processQueue();
    }
  }

  /**
   * Speak immediately (internal)
   */
  private async speakNow(text: string): Promise<void> {
    try {
      // Call backend TTS endpoint
      const response = await fetch("/api/osa/speak", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify({ text }),
      });

      if (!response.ok) {
        const error = await response.text();
        console.error("[OSA Voice] TTS failed:", response.status, error);
        return;
      }

      // Get audio blob
      const audioBlob = await response.blob();
      const audioUrl = URL.createObjectURL(audioBlob);

      // Play audio
      await this.playAudio(audioUrl);

      // Cleanup
      URL.revokeObjectURL(audioUrl);
    } catch (err) {
      console.error("[OSA Voice] Error:", err);
    }
  }

  /**
   * Play audio and track speaking state
   */
  private async playAudio(audioUrl: string): Promise<void> {
    return new Promise((resolve, reject) => {
      // Stop current audio if playing
      this.stop();

      // Create audio element
      const audio = new Audio(audioUrl);
      audio.volume = 0.8; // 80% volume
      this.currentAudio = audio;

      // Set speaking state
      this.isSpeaking = true;
      this.speakingCallback?.(true);

      // Handle playback events
      audio.onended = () => {
        this.isSpeaking = false;
        this.currentAudio = null;
        this.speakingCallback?.(false);
        resolve();
      };

      audio.onerror = (err) => {
        console.error("[OSA Voice] Playback error:", err);
        this.isSpeaking = false;
        this.currentAudio = null;
        this.speakingCallback?.(false);
        reject(err);
      };

      // Start playback
      audio.play().catch((err) => {
        console.error("[OSA Voice] Play failed:", err);
        this.isSpeaking = false;
        this.currentAudio = null;
        this.speakingCallback?.(false);
        reject(err);
      });
    });
  }

  /**
   * Stop current speech
   */
  stop() {
    if (this.currentAudio) {
      this.currentAudio.pause();
      this.currentAudio.currentTime = 0;
      this.currentAudio = null;
    }

    if (this.isSpeaking) {
      this.isSpeaking = false;
      this.speakingCallback?.(false);
    }

    // Clear queue
    this.audioQueue = [];
    this.isProcessingQueue = false;
  }

  /**
   * Check if currently speaking
   */
  getIsSpeaking(): boolean {
    return this.isSpeaking;
  }

  /**
   * Clear audio queue
   */
  clearQueue() {
    this.audioQueue = [];
  }

  /**
   * Set audio volume (0.0 to 1.0)
   */
  setVolume(volume: number) {
    if (this.currentAudio) {
      this.currentAudio.volume = Math.max(0, Math.min(1, volume));
    }
  }
}

// Export singleton instance
export const osaVoiceService = OSAVoiceService.getInstance();
