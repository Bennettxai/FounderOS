// Window Snap Store - Snap zones, split-screen, and quadrant snapping
// Contains snapWindow logic: left/right halves, four quadrant corners.

import type { WindowStoreShape, SnapZone } from "./desktopTypes";

// Creates the snap methods that operate on the window store
export function createSnapMethods(
  update: (fn: (state: WindowStoreShape) => WindowStoreShape) => void,
) {
  return {
    // Snap window to a zone (split screen / quadrants)
    snapWindow: (
      windowId: string,
      zone: SnapZone,
      workspaceWidth: number,
      workspaceHeight: number,
    ) => {
      update((state) => ({
        ...state,
        windows: state.windows.map((w) => {
          if (w.id !== windowId) return w;

          // If unsnapping, restore previous bounds
          if (!zone) {
            return {
              ...w,
              snapped: null,
              maximized: false,
              x: w.previousBounds?.x ?? w.x,
              y: w.previousBounds?.y ?? w.y,
              width: w.previousBounds?.width ?? w.width,
              height: w.previousBounds?.height ?? w.height,
              previousBounds: undefined,
            };
          }

          // Store current bounds if not already snapped
          const prevBounds = w.snapped
            ? w.previousBounds
            : { x: w.x, y: w.y, width: w.width, height: w.height };

          // Calculate new bounds based on zone
          let newBounds = {
            x: 0,
            y: 0,
            width: workspaceWidth,
            height: workspaceHeight,
          };

          switch (zone) {
            case "left":
              newBounds = {
                x: 0,
                y: 0,
                width: workspaceWidth / 2,
                height: workspaceHeight,
              };
              break;
            case "right":
              newBounds = {
                x: workspaceWidth / 2,
                y: 0,
                width: workspaceWidth / 2,
                height: workspaceHeight,
              };
              break;
            case "top-left":
              newBounds = {
                x: 0,
                y: 0,
                width: workspaceWidth / 2,
                height: workspaceHeight / 2,
              };
              break;
            case "top-right":
              newBounds = {
                x: workspaceWidth / 2,
                y: 0,
                width: workspaceWidth / 2,
                height: workspaceHeight / 2,
              };
              break;
            case "bottom-left":
              newBounds = {
                x: 0,
                y: workspaceHeight / 2,
                width: workspaceWidth / 2,
                height: workspaceHeight / 2,
              };
              break;
            case "bottom-right":
              newBounds = {
                x: workspaceWidth / 2,
                y: workspaceHeight / 2,
                width: workspaceWidth / 2,
                height: workspaceHeight / 2,
              };
              break;
          }

          return {
            ...w,
            snapped: zone,
            maximized: false,
            x: newBounds.x,
            y: newBounds.y,
            width: newBounds.width,
            height: newBounds.height,
            previousBounds: prevBounds,
          };
        }),
      }));
    },
  };
}
