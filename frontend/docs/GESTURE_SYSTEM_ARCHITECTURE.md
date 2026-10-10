# 🖐️ Gesture Control System Architecture

**Version**: 2.0 (Production)
**Last Updated**: January 2026
**Status**: ✅ Production Ready

---

## 📋 Table of Contents

1. [Overview](#overview)
2. [System Architecture](#system-architecture)
3. [Core Components](#core-components)
4. [Gesture Detection Flow](#gesture-detection-flow)
5. [Supported Gestures](#supported-gestures)
6. [Configuration](#configuration)
7. [Performance Optimization](#performance-optimization)
8. [Known Limitations](#known-limitations)
9. [Future Improvements](#future-improvements)
10. [Troubleshooting](#troubleshooting)

---

## Overview

The Gesture Control System enables hands-free 3D navigation of the BusinessOS Desktop using real-time hand tracking. Users can rotate, zoom, and interact with the 3D workspace using natural hand gestures.

### Key Features

- **Real-time hand tracking** at 30 FPS using MediaPipe Hands
- **Gesture locking system** prevents flickering between gestures
- **Smart gesture discrimination** (fist vs. pinch detection)
- **Hand loss buffer** maintains gestures through dropped frames
- **Debug visualization** with live landmark overlay
- **Performance optimized** for smooth operation

---

## System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Desktop3D.svelte                         │
│                   (Main Integration)                        │
└────────────┬────────────────────────────────────────────────┘
             │
             ├──► GestureDebugView.svelte (Debug UI)
             │    │
             │    ├──► HandTrackingService (Singleton)
             │    │    │
             │    │    └──► MediaPipe Hands Library
             │    │         - Camera stream (320x240 @ 30fps)
             │    │         - Hand landmark detection (21 points)
             │    │         - Real-time pose estimation
             │    │
             │    └──► GestureDetector (State Machine)
             │         - Gesture recognition
             │         - State locking system
             │         - Movement tracking
             │
             └──► Desktop3DScene.svelte (3D Rendering)
                  - Camera rotation via OrbitControls
                  - Zoom control
                  - THREE.js integration
```

### Data Flow

```
Camera Stream
    ↓
MediaPipe Hands (landmark detection)
    ↓
HandTrackingService (preprocessing)
    ↓
GestureDetector (state machine)
    ↓
GestureDebugView (emit gesture events)
    ↓
Desktop3D (action mapping)
    ↓
Desktop3DScene (apply to 3D view)
```

---

## Core Components

### 1. **Type Definitions** (`gestures.ts`)

**Purpose**: Master type definitions for the entire gesture system

**Key Types**:
- `GestureType`: Union of all gesture types (fist, pinch, open_palm, etc.)
- `GestureAction`: Actions triggered by gestures (drag, zoom_in, zoom_out, etc.)
- `GestureState`: Complete gesture state with position, velocity, metadata
- `HandTrackingResult`: Raw results from MediaPipe
- `GestureDetectorConfig`: Configuration with thresholds

**Configuration**:
```typescript
export const DEFAULT_GESTURE_CONFIG = {
    maxHands: 1,                      // Track only one hand
    modelComplexity: 0,               // LITE model (best performance)
    minDetectionConfidence: 0.7,      // Stable hand detection
    minTrackingConfidence: 0.7,       // Stable tracking
    pinchThreshold: 0.08,             // Thumb+index distance for pinch
    fistThreshold: 0.15,              // Fingers-to-palm distance for fist
    smoothingFactor: 0.75,            // Position smoothing (0=none, 1=max)
    updateIntervalMs: 16,             // 60 FPS gesture detection
    debug: false                      // Console logging
};
```

---

### 2. **Hand Tracking Service** (`handTrackingService.ts`)

**Purpose**: Singleton wrapper around MediaPipe Hands library

**Responsibilities**:
- Initialize MediaPipe with optimal settings
- Manage camera stream lifecycle
- Process hand landmarks at 30 FPS
- Emit `HandTrackingResult` events
- Calculate FPS and hand count

**Key Methods**:
```typescript
// Get singleton instance
const handTracking = HandTrackingService.getInstance(config);

// Initialize MediaPipe
await handTracking.initialize(videoElement, canvasElement);

// Register callback for results
handTracking.onResults((result: HandTrackingResult) => {
    // Handle landmarks
});

// Control tracking
await handTracking.start();
handTracking.stop();
handTracking.destroy();
```

**Performance Features**:
- Frame throttling (TARGET_FPS = 30)
- Singleton pattern prevents multiple MediaPipe instances
- Automatic cleanup on destroy
- Error handling with callbacks

**Camera Settings**:
- Resolution: 320x240 (performance optimized)
- Mode: Selfie (mirrored)
- Frame rate: 30 FPS target

---

### 3. **Gesture Detector** (`gestureDetector.ts`)

**Purpose**: State machine for gesture recognition and locking

**State Machine**:
```
idle → fist → idle
idle → pinch → idle
```

**Gesture Locking System**:

The locking system prevents rapid flickering between gestures:

1. **Lock Entry**: When gesture detected, enter locked state
2. **Lock Maintenance**: Ignore other gestures while locked
3. **Lock Exit**: Explicit exit condition must be met
   - Fist exits when hand opens fully (fingers > 0.30 from wrist)
   - Pinch exits when fingers separate (distance > 0.14)

**Hand Loss Buffer**:

Prevents reset on temporary tracking loss:
- Allows up to 8 dropped frames (~250ms at 30 FPS)
- Gesture lock persists through brief hand loss
- Resets only after sustained absence

**Movement Tracking**:

- Position smoothing via exponential moving average
- Movement threshold to filter micro-jitter
- Delta calculation for rotation/zoom
- Velocity tracking for gesture metadata

**Key Methods**:
```typescript
// Detect gestures from tracking result
const gestures = gestureDetector.detect(result);

// Map gesture to action
const action = gestureDetector.mapGestureToAction(gesture);
```

**Gesture Detection Logic**:

**FIST Detection**:
```typescript
private isFist(landmarks): boolean {
    // Measure distance of 4 fingertips from palm center
    const avgDistance = (index + middle + ring + pinky) / 4;
    return avgDistance < 0.15; // All fingers curled
}
```

**PINCH Detection** (Fixed - no more false positives!):
```typescript
private isPinch(landmarks): boolean {
    // Check 1: Thumb and index close together
    const thumbIndexDistance = distance(thumb, index);
    if (thumbIndexDistance >= 0.08) return false;

    // Check 2: Other 3 fingers EXTENDED (not curled)
    const otherFingersDistance = (middle + ring + pinky) / 3;
    return otherFingersDistance > 0.18; // Must be extended!
}
```

This prevents false pinch detection when making a fist (thumb wraps around index).

---

### 4. **Gesture Debug View** (`GestureDebugView.svelte`)

**Purpose**: Real-time gesture visualization and debugging

**Features**:
- Live camera feed with hand landmark overlay
- FPS counter and hand detection count
- Current gesture and action display
- Gesture log (state changes only, no spam)
- Start/stop controls
- Landmark visibility toggle

**Landmark Visualization**:
- 21 hand landmarks color-coded:
  - Red: Fingertips (4, 8, 12, 16, 20)
  - Yellow: Wrist (0)
  - Green: Joints
- Connections (bones) between landmarks
- Landmark indices for debugging

**Log Spam Prevention**:
```typescript
// Only log on state change
let lastLoggedGesture = '';

function addToGestureLog(gesture, action) {
    const logKey = `${gesture}:${action}`;
    if (logKey === lastLoggedGesture) return; // Same state - skip
    lastLoggedGesture = logKey;
    // Add to log
}
```

---

### 5. **Desktop3D Integration** (`Desktop3D.svelte`)

**Purpose**: Map gestures to 3D desktop actions

**Gesture Actions**:

| Gesture | Action | Behavior |
|---------|--------|----------|
| **Fist** | `drag` | Rotate camera with hand movement |
| **Pinch + Move Toward** | `zoom_in` | Decrease camera distance (zoom IN) |
| **Pinch + Move Away** | `zoom_out` | Increase camera distance (zoom OUT) |
| **Open Palm** | `none` | Exit gesture mode |

**Sensitivity Settings**:
```typescript
// Rotation (fist drag)
const rotX = gesture.deltaPosition.x * 25.0; // High sensitivity
const rotY = gesture.deltaPosition.y * 25.0;

// Zoom (pinch)
const zoomSpeed = gesture.deltaPosition.z * -200; // Inverted Z-axis
```

**Auto-Rotate Control**:
- Disabled when user drags with fist
- Re-enabled when gesture stops
- Prevents conflict between manual and auto rotation

---

## Gesture Detection Flow

### Frame Processing Pipeline

```
1. Camera captures frame (30 FPS)
   ↓
2. MediaPipe processes frame
   ↓
3. HandTrackingService receives landmarks
   ↓
4. GestureDetector.detect() called
   ↓
5. Check for hand loss (buffer system)
   ↓
6. If hand detected:
   a. Smooth position
   b. Check if locked in gesture
   c. If locked: verify exit condition
   d. If not locked: detect new gesture
   e. Calculate movement delta
   f. Return gesture state
   ↓
7. GestureDebugView emits gesture event
   ↓
8. Desktop3D maps gesture to action
   ↓
9. Desktop3DScene applies transformation
```

### Gesture State Transitions

```
IDLE STATE
    ↓ (detect fist)
FIST LOCKED
    ↓ (hand moves)
EMIT: fist → drag (with delta)
    ↓ (hand opens fully)
IDLE STATE

IDLE STATE
    ↓ (detect pinch)
PINCH LOCKED
    ↓ (hand moves toward/away)
EMIT: pinch → zoom_in/zoom_out (with delta)
    ↓ (fingers separate)
IDLE STATE
```

---

## Supported Gestures

### 1. Fist (Drag/Rotate)

**Hand Pose**: All 4 fingers curled close to palm

**Detection**: Average distance of fingertips from palm < 0.15

**Action**: Rotate 3D view (like mouse drag)

**Sensitivity**: 25x multiplier on hand movement

**Exit Condition**: Open hand fully (fingers > 0.30 from wrist)

**Use Case**: Natural way to explore 3D workspace from different angles

---

### 2. Pinch (Zoom)

**Hand Pose**:
- Thumb and index finger touching
- Middle, ring, pinky fingers EXTENDED

**Detection**:
- Thumb-index distance < 0.08 AND
- Other 3 fingers distance from palm > 0.18

**Action**: Zoom in/out based on Z-axis movement
- Move hand toward camera → Zoom IN (modules bigger)
- Move hand away from camera → Zoom OUT (modules smaller)

**Sensitivity**: 200x multiplier on Z-axis delta

**Exit Condition**: Fingers separate (distance > 0.14)

**Use Case**: Focus on specific modules or get overview

---

### 3. Open Palm (Release)

**Hand Pose**: All fingers extended far from wrist

**Detection**: Average distance of fingertips from wrist > 0.30

**Action**: Exit current gesture mode

**Use Case**: Reset to idle state, re-enable auto-rotate

---

## Configuration

### Tuning Gesture Detection

**Adjust Thresholds** (`gestures.ts`):

```typescript
// Stricter fist (smaller value = tighter fist required)
fistThreshold: 0.12  // Very strict
fistThreshold: 0.15  // Balanced (current)
fistThreshold: 0.18  // Looser

// Stricter pinch (smaller value = closer fingers required)
pinchThreshold: 0.05  // Very strict
pinchThreshold: 0.08  // Balanced (current)
pinchThreshold: 0.10  // Looser

// Adjust smoothing (higher = smoother but more lag)
smoothingFactor: 0.5   // More responsive, less smooth
smoothingFactor: 0.75  // Balanced (current)
smoothingFactor: 0.9   // Very smooth, more lag
```

**Adjust Sensitivity** (`Desktop3D.svelte`):

```typescript
// Rotation sensitivity (higher = more responsive)
const rotX = gesture.deltaPosition.x * 15.0;  // Less sensitive
const rotX = gesture.deltaPosition.x * 25.0;  // Current
const rotX = gesture.deltaPosition.x * 35.0;  // More sensitive

// Zoom sensitivity (higher = faster zoom)
const zoomSpeed = gesture.deltaPosition.z * -100;  // Slower
const zoomSpeed = gesture.deltaPosition.z * -200;  // Current
const zoomSpeed = gesture.deltaPosition.z * -300;  // Faster
```

**Adjust Movement Threshold** (`gestureDetector.ts`):

```typescript
// Fist drag threshold (lower = more sensitive to small movements)
if (movementMagnitude > 0.005) // Very sensitive, may jitter
if (movementMagnitude > 0.008) // Current
if (movementMagnitude > 0.015) // Less sensitive, smoother

// Pinch zoom threshold
if (movementMagnitude > 0.01)  // Current
if (movementMagnitude > 0.02)  // Less sensitive
```

---

## Performance Optimization

### Current Optimizations

1. **MediaPipe LITE Model** (modelComplexity: 0)
   - Fastest hand detection
   - Lower accuracy acceptable for gestures

2. **Low Resolution Camera** (320x240)
   - Reduces processing overhead
   - Sufficient for landmark detection

3. **Frame Throttling** (30 FPS target)
   - Limits CPU usage
   - Smooth gesture detection

4. **Singleton Pattern** (HandTrackingService)
   - Prevents multiple MediaPipe instances
   - Avoids resource leaks

5. **Movement Thresholds**
   - Filters micro-jitter
   - Reduces unnecessary updates

6. **Gesture Cooldowns** (16ms)
   - Limits event emission rate
   - Prevents gesture spam

7. **Position Smoothing**
   - Exponential moving average
   - Reduces jitter without lag

### Performance Metrics

**Target**: 30 FPS hand tracking, 60 FPS gesture detection

**Current**: ~8-10 FPS (needs investigation)

**Bottlenecks** (suspected):
- MediaPipe processing time
- Canvas drawing overhead
- High sensitivity multipliers causing rapid updates

---

## Known Limitations

### 1. Low FPS (~8-10 FPS)

**Impact**: Sluggish gesture response, dropped frames

**Possible Causes**:
- MediaPipe WASM performance
- Canvas redraw overhead
- Browser WebGL context issues

**Mitigation**:
- Hand loss buffer handles dropped frames
- Gesture locking prevents flickering

**Future Fix**: Profile and optimize render loop

---

### 2. Lighting Sensitivity

**Impact**: Poor hand detection in low light

**Cause**: MediaPipe relies on visual landmarks

**Mitigation**: Increase camera brightness if possible

**Best Practice**: Use in well-lit environment

---

### 3. Single Hand Only

**Current**: System tracks only 1 hand (maxHands: 1)

**Reason**: Performance optimization, simpler logic

**Future**: Could support two-hand gestures (pinch-zoom, rotate)

---

### 4. No Gesture Customization UI

**Current**: Thresholds hardcoded in config

**Future**: Add calibration UI for user-specific hands

---

### 5. Occlusion Handling

**Issue**: MediaPipe loses tracking if hand goes off-screen or is occluded

**Mitigation**: 8-frame hand loss buffer maintains gesture

**Future**: Add hand-out-of-bounds indicator

---

## Future Improvements

### Priority 1: Performance

**Goal**: Achieve 25-30 FPS consistent tracking

**Actions**:
1. Profile MediaPipe processing time
2. Optimize canvas redraw (use OffscreenCanvas?)
3. Investigate WASM performance issues
4. Consider WebGL-based landmark rendering
5. Add FPS performance mode (disable debug overlay)

---

### Priority 2: Gesture Calibration

**Goal**: Adapt to user's specific hand size and pose

**Features**:
- Calibration wizard on first use
- User makes fist → measure threshold
- User makes pinch → measure threshold
- Store calibration in localStorage
- Re-calibration option in settings

**UI Flow**:
```
1. "Let's calibrate your gestures"
2. "Make a tight fist" → capture fingertip distances
3. "Open your hand fully" → capture open palm threshold
4. "Make a pinch" → capture pinch threshold
5. "Done! You're ready to use gestures"
```

---

### Priority 3: Additional Gestures

**Wave** (rotate view):
- Detect hand moving side-to-side
- Use for continuous rotation

**Two-Hand Pinch-Zoom**:
- Both hands pinching
- Move hands apart → expand modules
- Move hands together → contract modules

**Thumbs Up** (context menu):
- Quick way to show command palette

**Point** (select object):
- Point at window → highlight
- Make fist → grab window

---

### Priority 4: Audio Gestures

**Status**: AudioGestureDetector implemented but not integrated

**Feature**: Double clap to toggle auto-rotate

**Integration**:
```typescript
// In Desktop3D.svelte
audioDetector.onDoubleClap(() => {
    desktop3dStore.toggleAutoRotate();
});
```

---

### Priority 5: Gesture Feedback

**Visual Feedback**:
- Show hand cursor on 3D view (following wrist position)
- Gesture name overlay ("DRAGGING", "ZOOMING")
- Haptic feedback if supported

**Audio Feedback**:
- Subtle click sound on gesture lock
- Whoosh sound on view rotation

---

### Priority 6: Gesture Recording

**Use Case**: Test gesture consistency, debug issues

**Features**:
- Record gesture session (landmarks + timestamps)
- Replay recording
- Export as JSON
- Import for testing

---

### Priority 7: Multi-Context Gestures

**Goal**: Different gestures in different view modes

**Contexts**:
- **Orb View**: Rotate orb, zoom orb
- **Focus View**: Resize window, move window
- **Grid View**: Scroll grid, adjust spacing

**Implementation**:
```typescript
interface ContextGestureMap {
    context: 'orb' | 'focus' | 'grid';
    gesture: GestureType;
    action: GestureAction;
}
```

---

## Troubleshooting

### Problem: Fist not detected

**Possible Causes**:
1. Fingers not close enough to palm
2. Fist threshold too strict (< 0.15)
3. Hand partially occluded

**Solutions**:
- Close fingers tighter
- Increase `fistThreshold` to 0.18
- Ensure full hand visible in camera
- Check lighting conditions

---

### Problem: False pinch detection on fist

**Fixed!** ✅

**Previous Cause**: System only checked thumb-index distance

**Current Solution**: Also checks that other 3 fingers are extended

**If still occurring**:
- Verify fix applied (check `isPinch()` method)
- Increase extended finger threshold from 0.18 to 0.20

---

### Problem: Gesture log spamming

**Fixed!** ✅

**Previous Cause**: Log added entry every frame

**Current Solution**: Only logs on state change

**If still occurring**:
- Verify `lastLoggedGesture` tracker implemented
- Check console for other log sources

---

### Problem: Camera rotation sluggish

**Fixed!** ✅

**Previous Cause**: Low sensitivity multiplier (8.0)

**Current Solution**: Increased to 25.0

**If still sluggish**:
- Increase multiplier to 30-35
- Lower movement threshold to 0.005
- Check FPS (if < 10 FPS, performance issue)

---

### Problem: Low FPS (< 10 FPS)

**Status**: Known issue under investigation

**Temporary Workarounds**:
1. Close other browser tabs
2. Use Chrome (better WebGL performance)
3. Reduce canvas size (not recommended - affects UX)
4. Disable debug overlay (showLandmarks = false)

**Long-term Fix**: Performance profiling required

---

### Problem: Gestures flicker on/off rapidly

**Fixed!** ✅ (via hand loss buffer)

**Previous Cause**: Immediate reset on dropped frame

**Current Solution**: 8-frame buffer before reset

**If still occurring**:
- Increase `MAX_FRAMES_WITHOUT_HAND` to 12
- Check lighting (poor lighting causes unstable tracking)
- Verify MediaPipe confidence settings (0.7)

---

### Problem: Zoom not working

**Check**:
1. Making proper pinch (other fingers extended)?
2. Moving hand toward/away camera (not left/right)?
3. Movement exceeds threshold (> 0.01)?

**Debug**:
```typescript
// In gestureDetector.ts handlePinchMovement()
console.log('Pinch delta:', delta);
console.log('Z movement:', Math.abs(delta.z));
```

---

## File Structure

```
frontend/
├── src/lib/
│   ├── types/
│   │   └── gestures.ts                 # Type definitions + config
│   ├── services/
│   │   ├── gestureDetector.ts          # Gesture state machine ✅
│   │   ├── handTrackingService.ts      # MediaPipe wrapper ✅
│   │   └── audioGestureDetector.ts     # Clap detection (not integrated)
│   └── components/desktop3d/
│       ├── GestureDebugView.svelte     # Debug UI ✅
│       ├── Desktop3D.svelte            # Main integration ✅
│       └── Desktop3DScene.svelte       # 3D rendering ✅
└── docs/
    └── GESTURE_SYSTEM_ARCHITECTURE.md  # This file
```

**Legend**:
- ✅ Production ready
- ⏳ Implemented but not integrated
- ❌ Removed (legacy)

---

## Quick Reference

### Enable Gestures

```typescript
// In Desktop3D.svelte
gestureControlEnabled = true;
showGestureDebug = true;
```

### Adjust Sensitivity

```typescript
// Rotation
const rotX = gesture.deltaPosition.x * 25.0; // Increase for more sensitivity

// Zoom
const zoomSpeed = gesture.deltaPosition.z * -200; // Increase for faster zoom
```

### Debug Gestures

```typescript
// Enable debug logging
DEFAULT_GESTURE_CONFIG.debug = true;

// Check gesture state
console.log('[Gesture]', gesture.type, gesture.metadata?.action);
```

---

## Change Log

### v2.0 (January 2026) - Current

**Major Changes**:
- ✅ Fixed false pinch detection on fists (added extended finger check)
- ✅ Fixed gesture log spam (state change tracking)
- ✅ Increased rotation sensitivity (8.0 → 25.0)
- ✅ Lowered movement thresholds (0.015 → 0.008)
- ✅ Loosened fist threshold (0.12 → 0.15)
- ✅ Removed legacy files (gestureDetector_old.ts, gestureDetector_v2.ts, handGestureService.ts)
- ✅ Added comprehensive documentation

**Performance**:
- Current: ~8-10 FPS (needs improvement)
- Target: 25-30 FPS

**Known Issues**:
- Low FPS under investigation
- No calibration UI yet

---

## Credits

**MediaPipe**: Google's ML solution for hand tracking
**THREE.js**: 3D rendering engine
**Threlte**: Svelte wrapper for THREE.js

---

**For questions or issues, check the troubleshooting section or contact the development team.**
