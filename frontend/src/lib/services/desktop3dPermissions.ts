/**
 * Desktop 3D Permissions Service
 *
 * Manages camera and microphone permissions exclusively for 3D Desktop mode.
 *
 * Features:
 * - Request camera access for hand tracking and gesture recognition
 * - Request microphone access for voice commands and clap detection
 * - Automatic cleanup when leaving 3D Desktop
 * - Privacy-first: All processing happens locally, no video/audio sent to servers
 *
 * Usage:
 * ```typescript
 * import { desktop3dPermissions } from '$lib/services/desktop3dPermissions';
 *
 * // On 3D Desktop mount
 * await desktop3dPermissions.requestAll();
 *
 * // On 3D Desktop unmount
 * desktop3dPermissions.cleanup();
 * ```
 */

import { writable, get } from "svelte/store";
import { browser } from "$app/environment";

// Permission states
export type PermissionState = "prompt" | "granted" | "denied";

// Stores for permission states
export const cameraPermission = writable<PermissionState>("prompt");
export const microphonePermission = writable<PermissionState>("prompt");
export const cameraStream = writable<MediaStream | null>(null);
export const microphoneStream = writable<MediaStream | null>(null);

/**
 * Desktop 3D Permissions Manager
 * Singleton service for managing camera and microphone access
 */
export class Desktop3DPermissions {
  private static instance: Desktop3DPermissions | null = null;
  private isInitialized = false;

  private constructor() {
    // Private constructor for singleton pattern
  }

  /**
   * Get singleton instance
   */
  static getInstance(): Desktop3DPermissions {
    if (!this.instance) {
      this.instance = new Desktop3DPermissions();
    }
    return this.instance;
  }

  /**
   * Initialize the service
   * Should be called when entering 3D Desktop mode
   */
  initialize(): void {
    if (this.isInitialized) {
      return;
    }

    this.isInitialized = true;
  }

  /**
   * Request camera access
   *
   * @returns Promise<boolean> - true if granted, false if denied
   */
  async requestCamera(): Promise<boolean> {
    if (!browser) {
      return false;
    }

    // Check if we already have a stream
    const existingStream = get(cameraStream);
    if (existingStream && existingStream.active) {
      return true;
    }

    try {
      // Request stream to get browser permission
      const stream = await navigator.mediaDevices.getUserMedia({
        video: {
          width: { ideal: 1280 },
          height: { ideal: 720 },
          frameRate: { ideal: 30, max: 60 },
        },
      });

      // IMMEDIATELY stop all tracks (turn off camera)
      stream.getTracks().forEach((track) => {
        track.stop();
      });

      // Store permission status but NOT the stream
      cameraPermission.set("granted");
      cameraStream.set(null); // Don't store stream - will be requested when actually needed

      return true;
    } catch (err) {
      const error = err as Error;
      console.error(
        "[Desktop3D Permissions] ❌ Camera access denied:",
        error.message,
      );

      cameraPermission.set("denied");
      cameraStream.set(null);

      return false;
    }
  }

  /**
   * Request microphone access
   *
   * @returns Promise<boolean> - true if granted, false if denied
   */
  async requestMicrophone(): Promise<boolean> {
    if (!browser) {
      return false;
    }

    // Check if we already have a stream
    const existingStream = get(microphoneStream);
    if (existingStream && existingStream.active) {
      return true;
    }

    try {
      // Request stream to get browser permission
      const stream = await navigator.mediaDevices.getUserMedia({
        audio: {
          echoCancellation: true,
          noiseSuppression: true,
          autoGainControl: true,
        },
      });

      // IMMEDIATELY stop all tracks (turn off microphone)
      stream.getTracks().forEach((track) => {
        track.stop();
      });

      // Store permission status but NOT the stream
      microphonePermission.set("granted");
      microphoneStream.set(null); // Don't store stream - will be requested when actually needed

      return true;
    } catch (err) {
      const error = err as Error;
      console.error(
        "[Desktop3D Permissions] ❌ Microphone access denied:",
        error.message,
      );

      microphonePermission.set("denied");
      microphoneStream.set(null);

      return false;
    }
  }

  /**
   * Request both camera and microphone access
   *
   * @returns Promise with results for both permissions
   */
  async requestAll(): Promise<{ camera: boolean; microphone: boolean }> {
    const [camera, microphone] = await Promise.all([
      this.requestCamera(),
      this.requestMicrophone(),
    ]);

    return { camera, microphone };
  }

  /**
   * Cleanup all media streams
   * IMPORTANT: Must be called when leaving 3D Desktop mode
   */
  cleanup(): void {
    // Stop camera stream
    const camera = get(cameraStream);
    if (camera) {
      camera.getTracks().forEach((track) => {
        track.stop();
      });
      cameraStream.set(null);
    }

    // Stop microphone stream
    const microphone = get(microphoneStream);
    if (microphone) {
      microphone.getTracks().forEach((track) => {
        track.stop();
      });
      microphoneStream.set(null);
    }

    this.isInitialized = false;
  }

  /**
   * Check if camera is currently active
   */
  hasCamera(): boolean {
    const permission = get(cameraPermission);
    const stream = get(cameraStream);
    return permission === "granted" && stream !== null && stream.active;
  }

  /**
   * Check if microphone is currently active
   */
  hasMicrophone(): boolean {
    const permission = get(microphonePermission);
    const stream = get(microphoneStream);
    return permission === "granted" && stream !== null && stream.active;
  }

  /**
   * Get current camera stream (may be null if not acquired yet)
   */
  getCameraStream(): MediaStream | null {
    return get(cameraStream);
  }

  /**
   * Get current microphone stream (may be null if not acquired yet)
   */
  getMicrophoneStream(): MediaStream | null {
    return get(microphoneStream);
  }

  /**
   * Actually acquire camera stream (turns camera ON)
   * Call this when user enables gesture control
   * This will request permission if not already granted
   */
  async acquireCameraStream(): Promise<MediaStream | null> {
    if (!browser) return null;

    // Check if we already have an active stream
    const existing = get(cameraStream);
    if (existing && existing.active) {
      return existing;
    }

    try {
      // Request stream (will prompt for permission if not granted)
      const stream = await navigator.mediaDevices.getUserMedia({
        video: {
          width: { ideal: 1280 },
          height: { ideal: 720 },
          frameRate: { ideal: 30, max: 60 },
        },
      });

      // Store permission and stream
      cameraPermission.set("granted");
      cameraStream.set(stream);

      return stream;
    } catch (err) {
      console.error(
        "[Desktop3D Permissions] Failed to acquire camera stream:",
        err,
      );
      cameraPermission.set("denied");
      return null;
    }
  }

  /**
   * Actually acquire microphone stream (turns mic ON)
   * Call this when user enables voice commands
   * This will request permission if not already granted
   */
  async acquireMicrophoneStream(): Promise<MediaStream | null> {
    if (!browser) return null;

    // Check if we already have an active stream
    const existing = get(microphoneStream);
    if (existing && existing.active) {
      return existing;
    }

    try {
      // Request stream (will prompt for permission if not granted)
      const stream = await navigator.mediaDevices.getUserMedia({
        audio: {
          echoCancellation: true,
          noiseSuppression: true,
          autoGainControl: true,
        },
      });

      // Store permission and stream
      microphonePermission.set("granted");
      microphoneStream.set(stream);

      return stream;
    } catch (err) {
      console.error(
        "[Desktop3D Permissions] Failed to acquire microphone stream:",
        err,
      );
      microphonePermission.set("denied");
      return null;
    }
  }

  /**
   * Check if permissions API is supported
   */
  isSupported(): boolean {
    if (!browser) return false;
    return !!(navigator.mediaDevices && navigator.mediaDevices.getUserMedia);
  }

  /**
   * Get detailed permission status
   */
  getStatus() {
    return {
      supported: this.isSupported(),
      initialized: this.isInitialized,
      camera: {
        permission: get(cameraPermission),
        active: this.hasCamera(),
        stream: get(cameraStream),
      },
      microphone: {
        permission: get(microphonePermission),
        active: this.hasMicrophone(),
        stream: get(microphoneStream),
      },
    };
  }
}

// Export singleton instance
export const desktop3dPermissions = Desktop3DPermissions.getInstance();

// Default export
export default desktop3dPermissions;
