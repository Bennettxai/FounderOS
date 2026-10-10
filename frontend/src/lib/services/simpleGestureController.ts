/**
 * Simple Gesture Controller
 *
 * Clean, minimal gesture system based on the successful particle system pattern.
 * NO complex state machines, NO abstraction layers - just direct MediaPipe → Camera control.
 *
 * Architecture:
 * MediaPipe Hands → Detect Pose → Apply Camera Action (2 steps, instant)
 *
 * Supported Gestures:
 * - ✊ Fist: Rotate camera (track hand movement)
 * - 🤏 Pinch: Zoom camera (track hand up/down)
 * - ✋ Open Palm: Reset camera to default position
 */

import { Hands, type Results, type NormalizedLandmark } from '@mediapipe/hands';
import { Camera } from '@mediapipe/camera_utils';

type GesturePose = 'fist' | 'pinch' | 'open' | 'none';

interface GestureCallbacks {
	onRotate: (deltaX: number, deltaY: number) => void;
	onZoom: (deltaZ: number) => void;
	onReset: () => void;
}

export class SimpleGestureController {
	private hands: Hands | null = null;
	private camera: Camera | null = null;
	private videoElement: HTMLVideoElement | null = null;

	// Callbacks for direct camera control
	private onRotate?: (deltaX: number, deltaY: number) => void;
	private onZoom?: (deltaZ: number) => void;
	private onReset?: (

) => void;

	// Previous hand position for delta calculation
	private prevHandPos: { x: number; y: number; z: number } | null = null;

	// Current gesture (what hand is doing RIGHT NOW)
	private currentGesture: GesturePose = 'none';

	// Momentum/inertia for smooth rotation
	private velocity = { x: 0, y: 0 };
	private momentumInterval: number | null = null;

	// FPS tracking (disabled for performance)
	private lastFrameTime = 0;
	private fps = 0;
	private frameCount = 0;
	private enableDebugLogs = false; // TURN OFF for production
	private frameSkipCounter = 0; // Frame skipping for performance

	/**
	 * Initialize MediaPipe Hands and start camera
	 */
	async init(videoElement: HTMLVideoElement): Promise<void> {
		this.videoElement = videoElement;

		try {
			// 1. Setup MediaPipe Hands (CDN loaded, hard version lock for stability)
			this.hands = new Hands({
				locateFile: (file) => {
					// HARD VERSION LOCK: v0.4.1646424915 (same as working particle system)
					return `https://cdn.jsdelivr.net/npm/@mediapipe/hands@0.4.1646424915/${file}`;
				}
			});

			// 2. Configure for performance and stability
			this.hands.setOptions({
				maxNumHands: 1, // ONE hand only (simpler, faster)
				modelComplexity: 0, // LITE model (fastest)
				minDetectionConfidence: 0.7, // Higher = more stable
				minTrackingConfidence: 0.7, // Higher = less flickering
				selfieMode: true // Mirror mode for natural interaction
			});

			// 3. Register results callback
			this.hands.onResults((results) => this.handleResults(results));

			// 4. Initialize MediaPipe models BEFORE starting camera
			await this.hands.initialize();

			// 5. Start camera
			this.camera = new Camera(videoElement, {
				onFrame: async () => {
					if (this.hands) {
						await this.hands.send({ image: videoElement });
					}
				},
				width: 320, // Low res for performance
				height: 240,
				facingMode: 'user'
			});

			await this.camera.start();
		} catch (error) {
			console.error('[SimpleGesture] Initialization failed:', error);
			throw error;
		}
	}

	/**
	 * Set camera control callbacks (called from Desktop3DScene)
	 */
	setCallbacks(callbacks: GestureCallbacks): void {
		this.onRotate = callbacks.onRotate;
		this.onZoom = callbacks.onZoom;
		this.onReset = callbacks.onReset;
	}

	/**
	 * Handle MediaPipe results (called every frame)
	 */
	private handleResults(results: Results): void {
		this.frameCount++;

		// FRAME SKIPPING: Only process every 2nd frame for performance
		// This cuts MediaPipe processing in half (30 FPS → 15 FPS effective)
		this.frameSkipCounter++;
		if (this.frameSkipCounter % 2 !== 0) {
			return; // Skip this frame
		}

		// Calculate FPS
		const now = performance.now();
		const delta = now - this.lastFrameTime;
		this.fps = delta > 0 ? 1000 / delta : 0;
		this.lastFrameTime = now;

		// No hand detected
		if (!results.multiHandLandmarks?.[0]) {
			this.currentGesture = 'none';
			this.prevHandPos = null;
			return;
		}

		const landmarks = results.multiHandLandmarks[0];

		// STEP 1: Detect what gesture hand is making RIGHT NOW
		const previousGesture = this.currentGesture;
		this.currentGesture = this.detectGesture(landmarks);

		// STEP 2: Apply camera action DIRECTLY based on gesture
		this.applyGestureAction(landmarks);
	}

	/**
	 * Detect hand gesture using simple distance checks
	 * NO state machine, just "what is the hand doing RIGHT NOW?"
	 */
	private detectGesture(landmarks: NormalizedLandmark[]): GesturePose {
		// Key landmarks (MediaPipe hand model has 21 points)
		const thumb = landmarks[4]; // Thumb tip
		const index = landmarks[8]; // Index finger tip
		const middle = landmarks[12]; // Middle finger tip
		const ring = landmarks[16]; // Ring finger tip
		const pinky = landmarks[20]; // Pinky tip
		const palm = landmarks[0]; // Wrist (palm base)

		// FIST: All fingertips close to palm
		const allFingersClosed =
			this.distance(index, palm) < 0.15 &&
			this.distance(middle, palm) < 0.15 &&
			this.distance(ring, palm) < 0.15 &&
			this.distance(pinky, palm) < 0.15;

		if (allFingersClosed) return 'fist';

		// PINCH: Thumb + Index touching, other 3 fingers open
		const thumbIndexClose = this.distance(thumb, index) < 0.08;
		const othersOpen =
			this.distance(middle, palm) > 0.18 &&
			this.distance(ring, palm) > 0.18 &&
			this.distance(pinky, palm) > 0.18;

		if (thumbIndexClose && othersOpen) return 'pinch';

		// OPEN PALM: All fingers spread wide
		const allFingersOpen =
			this.distance(index, palm) > 0.30 &&
			this.distance(middle, palm) > 0.30 &&
			this.distance(ring, palm) > 0.30 &&
			this.distance(pinky, palm) > 0.30;

		if (allFingersOpen) return 'open';

		return 'none';
	}

	/**
	 * Apply camera action based on current gesture
	 * DIRECT control - no queuing, no delays, instant response
	 */
	private applyGestureAction(landmarks: NormalizedLandmark[]): void {
		const wrist = landmarks[0]; // Track wrist position for movement
		const thumb = landmarks[4]; // Thumb tip
		const index = landmarks[8]; // Index finger tip

		if (this.currentGesture === 'fist') {
			// FIST: Rotate (X/Y movement) + Zoom (Z depth) + MOMENTUM
			if (this.prevHandPos) {
				// ROTATE: X/Y hand movement
				const deltaX = (wrist.x - this.prevHandPos.x) * 8.0;
				const deltaY = (wrist.y - this.prevHandPos.y) * 8.0;

				// ZOOM: Z depth (hand moving toward/away from camera)
				const deltaZ = (wrist.z - this.prevHandPos.z) * -300;

				// Update velocity for momentum (store raw deltas)
				this.velocity.x = deltaX;
				this.velocity.y = deltaY;

				// Apply rotation
				if (this.onRotate) {
					this.onRotate(deltaX, deltaY);
				}

				// Apply zoom (fist moving toward/away)
				if (Math.abs(deltaZ) > 1 && this.onZoom) {
					this.onZoom(deltaZ);
				}
			}
			this.prevHandPos = { x: wrist.x, y: wrist.y, z: wrist.z };
		} else if (this.currentGesture === 'pinch') {
			// PINCH: ZOOM based on hand Z-depth (distance from camera)
			// Hand CLOSER to camera = zoom OUT (reverse intuitive)
			// Hand FURTHER from camera = zoom IN
			// Stop momentum when pinching
			this.stopMomentum();

			// Use midpoint of thumb and index for Z tracking
			const midZ = (thumb.z + index.z) / 2;

			if (this.prevHandPos) {
				// Z change controls zoom (REVERSED)
				// Hand moves CLOSER (Z decreases) = positive deltaZ = zoom OUT
				// Hand moves FURTHER (Z increases) = negative deltaZ = zoom IN
				const zChange = midZ - this.prevHandPos.z;
				const deltaZ = zChange * 1500; // INCREASED sensitivity for smoother response

				if (this.onZoom) {
					this.onZoom(deltaZ);
				}
			}
			// Store Z position for next frame
			this.prevHandPos = { x: 0, y: 0, z: midZ };
		} else if (this.currentGesture === 'open') {
			// RESET: Call reset once when palm opens
			this.stopMomentum();
			if (this.prevHandPos && this.onReset) {
				this.onReset();
			}
			this.prevHandPos = null;
		} else {
			// NONE: Start momentum if we just released fist
			if (this.prevHandPos && (Math.abs(this.velocity.x) > 0.1 || Math.abs(this.velocity.y) > 0.1)) {
				this.startMomentum();
			}
			this.prevHandPos = null;
		}
	}

	/**
	 * Start momentum/inertia animation
	 */
	private startMomentum(): void {
		// Clear existing momentum
		this.stopMomentum();

		// Apply momentum loop (60 FPS)
		this.momentumInterval = window.setInterval(() => {
			// Apply current velocity
			if (this.onRotate) {
				this.onRotate(this.velocity.x, this.velocity.y);
			}

			// Apply friction (reduce velocity by 5% each frame)
			this.velocity.x *= 0.95;
			this.velocity.y *= 0.95;

			// Stop when velocity is negligible
			if (Math.abs(this.velocity.x) < 0.01 && Math.abs(this.velocity.y) < 0.01) {
				this.stopMomentum();
			}
		}, 16); // ~60 FPS
	}

	/**
	 * Stop momentum animation
	 */
	private stopMomentum(): void {
		if (this.momentumInterval !== null) {
			clearInterval(this.momentumInterval);
			this.momentumInterval = null;
			this.velocity.x = 0;
			this.velocity.y = 0;
		}
	}

	/**
	 * Calculate 3D distance between two hand landmarks
	 */
	private distance(a: NormalizedLandmark, b: NormalizedLandmark): number {
		const dx = a.x - b.x;
		const dy = a.y - b.y;
		const dz = a.z - b.z;
		return Math.sqrt(dx * dx + dy * dy + dz * dz);
	}

	/**
	 * Get current FPS
	 */
	getFPS(): number {
		return this.fps;
	}

	/**
	 * Get current gesture state
	 */
	getCurrentGesture(): GesturePose {
		return this.currentGesture;
	}

	/**
	 * Cleanup resources
	 */
	destroy(): void {
		// Stop momentum
		this.stopMomentum();

		// Stop camera
		if (this.camera) {
			this.camera.stop();
			this.camera = null;
		}

		// Stop video stream tracks
		if (this.videoElement?.srcObject) {
			const stream = this.videoElement.srcObject as MediaStream;
			stream.getTracks().forEach((track) => track.stop());
			this.videoElement.srcObject = null;
		}

		// Close MediaPipe
		if (this.hands) {
			this.hands.close();
			this.hands = null;
		}

		this.videoElement = null;
		this.prevHandPos = null;
		this.currentGesture = 'none';
	}
}
