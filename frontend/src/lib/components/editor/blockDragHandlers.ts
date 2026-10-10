import { editor, type EditorBlock, type EditorState } from "$lib/stores/editor";

export type DragPosition = "above" | "below" | null;

/**
 * Creates drag-and-drop handlers bound to a specific block.
 * Returns handlers and a reactive dragOverPosition reference tracker.
 */
export function createDragHandlers(
  blockId: string,
  getDragOverPosition: () => DragPosition,
  setDragging: (v: boolean) => void,
  setDragOver: (v: boolean) => void,
  setDragOverPosition: (v: DragPosition) => void,
) {
  function handleDragStart(e: DragEvent) {
    setDragging(true);
    e.dataTransfer?.setData("text/plain", blockId);
    e.dataTransfer!.effectAllowed = "move";
    requestAnimationFrame(() => {
      const wrapper = (e.target as HTMLElement).closest(".block-wrapper");
      wrapper?.classList.add("dragging");
    });
  }

  function handleDragEnd(e: DragEvent) {
    setDragging(false);
    const wrapper = (e.target as HTMLElement).closest(".block-wrapper");
    wrapper?.classList.remove("dragging");
    document.querySelectorAll(".block-wrapper").forEach((el) => {
      el.classList.remove("drag-over-above", "drag-over-below");
    });
  }

  function handleDragOver(e: DragEvent) {
    e.preventDefault();
    e.dataTransfer!.dropEffect = "move";

    const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
    const midY = rect.top + rect.height / 2;
    const position: DragPosition = e.clientY < midY ? "above" : "below";

    const wrapper = e.currentTarget as HTMLElement;
    wrapper.classList.remove("drag-over-above", "drag-over-below");
    wrapper.classList.add(`drag-over-${position}`);

    setDragOver(true);
    setDragOverPosition(position);
  }

  function handleDragLeave(e: DragEvent) {
    const wrapper = e.currentTarget as HTMLElement;
    wrapper.classList.remove("drag-over-above", "drag-over-below");
    setDragOver(false);
    setDragOverPosition(null);
  }

  function handleDrop(e: DragEvent) {
    e.preventDefault();
    const sourceBlockId = e.dataTransfer?.getData("text/plain");
    if (!sourceBlockId || sourceBlockId === blockId) return;

    const wrapper = e.currentTarget as HTMLElement;
    wrapper.classList.remove("drag-over-above", "drag-over-below");

    const dragOverPosition = getDragOverPosition();

    editor.update((s: EditorState) => {
      const sourceIdx = s.blocks.findIndex(
        (b: EditorBlock) => b.id === sourceBlockId,
      );
      const targetIdx = s.blocks.findIndex(
        (b: EditorBlock) => b.id === blockId,
      );
      if (sourceIdx === -1 || targetIdx === -1) return s;

      const newBlocks = [...s.blocks];
      const [movedBlock] = newBlocks.splice(sourceIdx, 1);

      let insertIdx = targetIdx;
      if (dragOverPosition === "below") {
        insertIdx = sourceIdx < targetIdx ? targetIdx : targetIdx + 1;
      } else {
        insertIdx = sourceIdx < targetIdx ? targetIdx - 1 : targetIdx;
      }

      newBlocks.splice(insertIdx, 0, movedBlock);
      return { ...s, blocks: newBlocks, isDirty: true };
    });

    setDragOver(false);
    setDragOverPosition(null);
  }

  return {
    handleDragStart,
    handleDragEnd,
    handleDragOver,
    handleDragLeave,
    handleDrop,
  };
}
