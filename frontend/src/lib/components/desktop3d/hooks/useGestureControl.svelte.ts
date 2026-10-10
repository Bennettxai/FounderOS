/**
 * useGestureControl
 *
 * Svelte 5 runes hook that encapsulates all gesture-control lifecycle:
 * enabling, disabling, toggling, and camera callback wiring.
 *
 * Returns reactive state and action functions so Desktop3D.svelte stays
 * focused on orchestration rather than implementation details.
 */

import { SimpleGestureController } from "$lib/services/simpleGestureController";
import { desktop3dStore } from "$lib/stores/desktop3dStore";
import * as THREE from "three";

interface UseGestureControlOptions {
  /** Getter that returns the current OrbitControls ref (may be null) */
  getOrbitControls: () => unknown;
}

interface UseGestureControlReturn {
  /** Whether gesture control is currently active */
  readonly gestureEnabled: boolean;
  /** True while the controller is initialising */
  readonly gestureLoading: boolean;
  /** Bind this to the hidden <video> element */
  videoElement: HTMLVideoElement | null;
  enableGesture: () => Promise<void>;
  disableGesture: () => void;
  toggleGesture: () => Promise<void>;
}

export function useGestureControl(
  opts: UseGestureControlOptions,
): UseGestureControlReturn {
  let gestureEnabled = $state(false);
  let gestureLoading = $state(false);
  let videoElement = $state<HTMLVideoElement | null>(null);
  let controller = $state<SimpleGestureController | null>(null);

  async function enableGesture(): Promise<void> {
    if (gestureLoading || gestureEnabled) {
      return;
    }

    if (!videoElement) {
      console.error("[useGestureControl] Video element not found");
      alert("Error: Video element not initialized");
      return;
    }

    const orbitControls = opts.getOrbitControls();
    if (!orbitControls) {
      console.error("[useGestureControl] OrbitControls not ready yet");
      alert("3D scene is still loading. Please wait a moment and try again.");
      return;
    }

    gestureLoading = true;

    try {
      const newController = new SimpleGestureController();

      newController.setCallbacks({
        onRotate: (deltaX: number, deltaY: number) => {
          const controls = opts.getOrbitControls() as any;
          if (!controls) return;

          desktop3dStore.setAutoRotate(false);

          const offset = new THREE.Vector3();
          offset.copy(controls.object.position).sub(controls.target);

          const spherical = new THREE.Spherical();
          spherical.setFromVector3(offset);

          spherical.theta -= deltaX * 1.0;
          spherical.phi -= deltaY * 1.0;
          spherical.phi = Math.max(0.1, Math.min(Math.PI - 0.1, spherical.phi));

          offset.setFromSpherical(spherical);
          controls.object.position.copy(controls.target).add(offset);
          controls.update();
        },

        onZoom: (deltaZ: number) => {
          const controls = opts.getOrbitControls() as any;
          if (!controls) return;

          const currentDistance = controls.object.position.length();
          const newDistance = Math.max(
            200,
            Math.min(800, currentDistance + deltaZ),
          );
          controls.object.position.normalize().multiplyScalar(newDistance);
          controls.update();
        },

        onReset: () => {
          const controls = opts.getOrbitControls() as any;
          if (!controls) return;

          controls.object.position.set(0, 40, 400);
          controls.target.set(0, 0, 0);
          controls.update();

          desktop3dStore.setAutoRotate(true);
        },
      });

      await newController.init(videoElement);

      controller = newController;
      gestureEnabled = true;
      gestureLoading = false;
    } catch (error) {
      console.error(
        "[useGestureControl] Failed to enable gesture control:",
        error,
      );

      const msg = error instanceof Error ? error.message : "Unknown error";
      if (
        msg.includes("Permission denied") ||
        msg.includes("NotAllowedError")
      ) {
        alert(
          "Camera permission denied. Please allow camera access and try again.",
        );
      } else if (msg.includes("NotFoundError") || msg.includes("not found")) {
        alert("No camera found. Please connect a webcam and try again.");
      } else {
        alert(`Failed to enable gestures: ${msg}`);
      }

      gestureEnabled = false;
      gestureLoading = false;
    }
  }

  function disableGesture(): void {
    if (controller) {
      controller.destroy();
      controller = null;
    }
    gestureEnabled = false;
  }

  async function toggleGesture(): Promise<void> {
    if (gestureEnabled) {
      disableGesture();
    } else {
      await enableGesture();
    }
  }

  return {
    get gestureEnabled() {
      return gestureEnabled;
    },
    get gestureLoading() {
      return gestureLoading;
    },
    get videoElement() {
      return videoElement;
    },
    set videoElement(v) {
      videoElement = v;
    },
    enableGesture,
    disableGesture,
    toggleGesture,
  };
}
